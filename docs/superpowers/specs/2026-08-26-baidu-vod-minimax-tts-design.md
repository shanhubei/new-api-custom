# 百度 VOD MiniMax TTS 渠道设计

**日期：** 2026-08-26  
**状态：** 已确认  
**范围：** 新增同步渠道类型「百度 VOD MiniMax」；一期仅 TTS（`/v2/tts`）  
**上游鉴权文档：** [生成认证字符串（bce-auth-v1）](https://cloud.baidu.com/doc/Reference/s/njwvz1yfu)  
**使用说明：** [docs/channel/baidu-vod-minimax.md](../../channel/baidu-vod-minimax.md)

## 背景

项目已有：

1. **官方 MiniMax**（`ChannelTypeMiniMax = 35`，`relay/channel/minimax`）  
   - TTS：`POST {api.minimax.chat}/v1/t2a_v2`  
   - 鉴权：`Authorization: Bearer <apikey>`  
   - 客户端：`POST /v1/audio/speech`

2. **百度 VOD Vidu**（`ChannelTypeBaiduVodVidu = 59`）  
   - Base：`http://vod.bj.baidubce.com/v3/aigc/vd`  
   - 鉴权：`Bearer <apikey>`  
   - 协议：Vidu 透传（视频 / 异步生图）

百度 VOD 另提供 MiniMax 风格 TTS：

```http
POST https://vod.bj.baidubce.com/v2/tts
Host: vod.bj.baidubce.com
Authorization: bce-auth-v1/{accessKeyId}/{timestamp}/{expirationPeriodInSeconds}/host/{signature}
Content-Type: application/json
```

请求/响应体与官方 MiniMax TTS 同构（`model` / `text` / `voice_setting` / `audio_setting` / `data.audio` / `extra_info.usage_characters` / `base_resp`），鉴权改为 BCE AK/SK 签名。

需要新增独立渠道，使客户端仍走统一的 `POST /v1/audio/speech`。

## 目标

1. 新增渠道类型「Baidu VOD MiniMax」，后台可选。
2. 客户端入口复用 `POST /v1/audio/speech`（与官方 MiniMax 用法一致）。
3. 上游固定 `POST {base}/v2/tts`；默认 Base `https://vod.bj.baidubce.com`。
4. 鉴权：`bce-auth-v1`，渠道 Key 格式 `AK|SK`。
5. 请求体按 MiniMax 兼容字段组包；响应处理对齐现有 MiniMax TTS（url 重定向 / hex 解码 / 字符用量计费）。
6. 与官方 MiniMax、百度 VOD Vidu 解耦，不改其行为。

## 非目标

- 一期不做视频 / Chat / 生图。
- 不使用 Bearer apikey 鉴权。
- 不新建独立客户端路径（如 `/v1/baidu-vod/tts`）。
- 不修改官方 MiniMax(35)、百度 VOD Vidu(59)。
- 不按上游 `credits` 差额结算（沿用现有 TTS 字符/用量计费路径）。

## 决策摘要

| 项 | 决定 |
|----|------|
| 实现方式 | 新渠道类型 + 同步 Adaptor；TTS 转换逻辑对齐/复用官方 MiniMax，改 URL 与鉴权 |
| 渠道 ID | `ChannelTypeBaiduVodMinimax = 60`（插在 `ChannelTypeDummy` 之前） |
| APIType | 新增 `APITypeBaiduVodMinimax`（或等价独立 type，勿复用 `APITypeMiniMax`） |
| 展示名 | `Baidu VOD MiniMax` |
| 默认 Base | `https://vod.bj.baidubce.com` |
| 上游 Path | `/v2/tts` |
| 鉴权 | `Authorization: bce-auth-v1/{AK}/{timestamp}/{expiration}/host/{signature}`；样例 signedHeaders=`host` |
| Key 格式 | `AK\|SK` |
| 客户端入口 | `POST /v1/audio/speech` |
| 防混策略 | 运营侧：勿与官方 MiniMax 在同一分组挂同名模型；可用模型映射加前缀 |
| 一期模型 | `speech-2.8-hd` / `speech-2.8-turbo` / `speech-2.6-hd` / `speech-2.6-turbo` / `speech-02-hd` / `speech-02-turbo` / `speech-01-hd` / `speech-01-turbo` |

## 方案对比（已选）

| 方案 | 说明 | 结论 |
|------|------|------|
| 1 扩展官方 MiniMax(35) 按 Base 切签名 | 官方与百度耦合，Key 形态不一 | 否 |
| 2 新渠道 + 独立 Adaptor（复制 MiniMax TTS 后改差异） | 与 Vidu VOD 新增渠道模式一致 | **采用** |
| 3 独立客户端路径 | 调用方要改 URL | 否（已选复用 `/v1/audio/speech`） |

## 架构

```
客户端 POST /v1/audio/speech
  → TokenAuth + Distribute（按 body.model 选渠道 type=60）
  → Relay（同步 AudioSpeech）
       → baidu_vod_minimax.Adaptor
            → ConvertAudioRequest：OpenAI AudioRequest → MiniMax 风格 JSON
            → SetupRequestHeader：BCE 签名（解析 AK|SK）
            → GET URL：{ChannelBaseUrl}/v2/tts
            → DoRequest → 上游
            → DoResponse：对齐 MiniMax handleTTSResponse
                 - base_resp.status_code != 0 → 错误
                 - data.audio 为 http(s) → 302/透传 URL 或写回
                 - 否则按 hex 解码写音频字节
                 - usage：extra_info.usage_characters
```

### 客户端字段映射

| `/v1/audio/speech` | 上游 `/v2/tts` |
|--------------------|----------------|
| `model` | `model` |
| `input` | `text` |
| `voice` | `voice_setting.voice_id` |
| `speed` | `voice_setting.speed` |
| `response_format` | `audio_setting.format`；非 hex 时 `output_format=url`（与现 MiniMax 行为对齐） |
| `metadata` | 透传到上游 struct（`emotion` / `vol` / `pitch` / `timbre_weights` / `pronunciation_dict` / `voice_modify` / `language_boost` / `subtitle_enable` / `aigc_watermark` 等） |

### 上游请求示例

```json
{
  "model": "speech-2.8-hd",
  "text": "你好，欢迎使用百度智能云语音合成服务。",
  "output_format": "url",
  "voice_setting": {
    "voice_id": "Boyan_new_hd",
    "speed": 1.0,
    "vol": 1.0,
    "pitch": 0,
    "emotion": "calm"
  },
  "audio_setting": {
    "sample_rate": 32000,
    "bitrate": 128000,
    "format": "mp3",
    "channel": 1
  }
}
```

### 上游响应示例

```json
{
  "data": {
    "audio": "https://bce-multimedia.cdn.bcebos.com/tmp/minimax/.../xxx.mp3?...",
    "status": 1
  },
  "trace_id": "trace-abc123",
  "extra_info": {
    "usage_characters": 20,
    "audio_length": 3200,
    "audio_sample_rate": 32000,
    "audio_size": 51200,
    "bitrate": 128,
    "audio_format": "mp3",
    "audio_channel": 1,
    "word_count": 18
  },
  "base_resp": {
    "status_code": 0,
    "status_msg": "OK"
  }
}
```

## BCE 签名

按百度文档生成：

`bce-auth-v1/{accessKeyId}/{timestamp}/{expirationPeriodInSeconds}/{signedHeaders}/{signature}`

一期约束（与用户样例对齐）：

1. Key：`strings.Split(apiKey, "|")` → AK、SK；格式非法则 400。
2. `timestamp`：UTC `yyyy-mm-ddThh:mm:ssZ`。
3. `expirationPeriodInSeconds`：默认 `1800`（或 `3600`，与样例一致可选常量，实现时固定一处并单测）。
4. `signedHeaders`：显式 `host`（小写）。
5. CanonicalHeaders 至少包含 `host:{hostname}`。
6. 请求 Header 必须设置：
   - `Host: vod.bj.baidubce.com`（或 Base 解析出的 host）
   - `Authorization: <认证串>`
   - `Content-Type: application/json`
7. 签名实现放在渠道包内 `sign.go`（或可复用的 `pkg/bceauth`）；**创建请求即时签名**，无缓存 token。
8. 不在 Header 使用 `Bearer`。

真网若要求额外签 `x-bce-date` / `content-type`，再扩展 signedHeaders；一期以样例 `host` 为准。

## Adaptor 行为

1. `Init`：无状态或仅缓存 channel meta。
2. `GetRequestURL`：`{base}/v2/tts`；base 空则用渠道默认 URL。
3. `SetupRequestHeader`：BCE 签名；禁止写 `Bearer`。
4. `ConvertAudioRequest`：仅支持 `RelayModeAudioSpeech`；映射逻辑对齐 `minimax.ConvertAudioRequest`。
5. `DoResponse`：对齐 `minimax.handleTTSResponse`（含 `base_resp`、`data.audio`、`usage_characters`）。
6. 其它 RelayMode（chat/image/video）：返回 not implemented（一期）。
7. `GetModelList`：一期 speech 模型列表；`GetChannelName`：`baidu_vod_minimax`。

## 与官方 MiniMax 防混

路径相同不会代码冲突；分流靠 `model` + 渠道配置：

1. **推荐**：同一分组内，官方与百度 VOD **不要**配置同名 speech 模型。
2. 若必须同名对外：用渠道模型映射，例如客户端 `baidu-speech-2.8-hd` → 上游 `speech-2.8-hd`。
3. 文档/运营说明写清两渠道差异（Base、鉴权、Key 形态）。

## 挂载点清单

### 后端

1. `constant/channel.go`：`ChannelTypeBaiduVodMinimax = 60`、默认 Base、`ChannelTypeNames`
2. `constant/api_type.go`：新 `APITypeBaiduVodMinimax`
3. `common/api_type.go`：`ChannelType2APIType` case
4. `relay/relay_adaptor.go`：`GetAdaptor` case → 新 Adaptor
5. `relay/channel/baidu_vod_minimax/`：`adaptor.go` / `sign.go` / `tts.go` / `constants.go` / 测试
6. `controller/model.go`（若需）：模型列表 OwnedBy
7. `model/pricing_default.go`（按需）：展示名映射
8. `controller/channel-test.go`：若 TTS 测不通则列入 unsupported 或不强制

### 前端

1. `web/default/src/features/channels/constants.ts`
2. `web/default/src/features/channels/lib/channel-utils.ts`
3. `web/classic/src/constants/channel.constants.js`
4. i18n：展示名可保留英文专有名词 `Baidu VOD MiniMax`

## 配置说明（运营）

- 渠道类型：Baidu VOD MiniMax (60)
- Base URL：`https://vod.bj.baidubce.com`（勿拼 `/v2/tts`）
- Key：`{AccessKeyId}|{SecretAccessKey}`
- 模型：按开通情况配置（至少验证 `speech-2.8-hd`）
- 与官方 MiniMax 分组/模型隔离，避免同名抢路由

## 测试与验收

1. **单测**
   - Key `AK|SK` 解析；非法 Key 报错
   - `GetRequestURL` = `{base}/v2/tts`
   - Authorization 前缀 `bce-auth-v1/`，含 `/host/`，**不含** `Bearer`
   - Canonical 签名对固定输入可复现（fixture）
   - ConvertAudioRequest：`input`→`text`，`voice`→`voice_setting.voice_id`
   - 响应：`status_code!=0` 失败；`audio` URL / hex 分支；`usage_characters` 计入 usage
2. **不要求** CI 打真网
3. **人工**：配 AK/SK + `speech-2.8-hd`；`POST /v1/audio/speech` 拿到音频 URL 或二进制；管理后台可选新渠道类型

## 风险与注意

1. **签名头域**：样例仅 `host`；若线上 403 `SignatureDoesNotMatch`，优先核对 Host、时间 skew、URI 编码与是否需追加 `x-bce-date`。
2. **bitrate 单位**：文档枚举 `32000/64000/128000/256000`；实现透传客户端/metadata，不做错误缩放；样例 JSON 若写 `128` 需以文档枚举为准。
3. **模型名重叠**：`speech-02-hd` 等与官方 MiniMax 可能重名，靠分组/映射隔离。
4. **真实 AK/SK 不得入库**。
5. 二期若加视频，另开 spec；勿在本渠道混入百度统一 AIGC（`H23` / `/v2/aigc/...`）除非另有 curl 确认。

## 验收标准（Definition of Done）

- [ ] 后台可选渠道「Baidu VOD MiniMax」
- [ ] Key=`AK|SK`，请求带 `bce-auth-v1` 且 signedHeaders 含 `host`
- [ ] `POST /v1/audio/speech` + 已配置模型可出音频
- [ ] 官方 MiniMax、百度 VOD Vidu 回归不受影响
- [ ] 单测覆盖 URL / 签名形态 / 字段映射 / 错误响应
