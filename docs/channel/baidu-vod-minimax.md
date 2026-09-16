# 百度 VOD MiniMax 渠道使用说明

渠道类型：**Baidu VOD MiniMax**（type `60`）  
一期能力：**仅 TTS**（语音合成）。

| 项 | 值 |
|----|-----|
| **推荐客户端入口** | `POST /v1/baidu-vod/tts`（本渠道专用；成功时**原样透传**上游 JSON） |
| 兼容入口 | `POST /v1/audio/speech`（与官方 MiniMax 同路径；URL 时为 **302**，不改其它渠道行为） |
| 上游 | `POST {base}/v2/tts`，默认 Base `https://vod.bj.baidubce.com`（**不要**在 Base 里拼 `/v2/tts`） |
| 渠道 Key | `{AccessKeyId}\|{SecretAccessKey}`（AK/SK）；鉴权 `bce-auth-v1`，**不是** Bearer |

设计稿：[docs/superpowers/specs/2026-08-26-baidu-vod-minimax-tts-design.md](../superpowers/specs/2026-08-26-baidu-vod-minimax-tts-design.md)  
上游鉴权文档：[生成认证字符串（bce-auth-v1）](https://cloud.baidu.com/doc/Reference/s/njwvz1yfu)

---

## 1. 后台配置

1. 渠道 → 新建 → 类型选 **Baidu VOD MiniMax**。
2. **密钥**填 `AK|SK`（例如 `ALTAKxxxx|xxxxxxxx`）。格式错误会在请求时报 400。
3. **Base URL** 用默认 `https://vod.bj.baidubce.com` 即可。
4. 模型按开通情况配置，一期支持列表：
   - `speech-2.8-hd` / `speech-2.8-turbo`
   - `speech-2.6-hd` / `speech-2.6-turbo`
   - `speech-02-hd` / `speech-02-turbo`
   - `speech-01-hd` / `speech-01-turbo`
5. 为模型配置单价（沿用现有 TTS 按字符/用量计费路径；**不**按上游 credits 差额结算）。

### 与官方 MiniMax 防混

1. **推荐**：客户端走 `/v1/baidu-vod/tts`，与官方 `/v1/audio/speech` 路径隔离。
2. 若仍走 `/v1/audio/speech`：同一分组内勿与官方 MiniMax 挂同名 speech 模型；或用映射如 `baidu-speech-2.8-hd` → `speech-2.8-hd`。

本渠道与官方 MiniMax（35）、百度 VOD Vidu（59）相互独立。

---

## 2. TTS 调用（推荐）

`POST /v1/baidu-vod/tts`  
`Authorization: Bearer <NewAPI sk>`  
`Content-Type: application/json`

### 请求示例

```json
{
  "model": "speech-2.8-hd",
  "input": "你好，欢迎使用百度智能云语音合成服务。",
  "voice": "Boyan_new_hd",
  "speed": 1.0,
  "response_format": "mp3",
  "metadata": {
    "voice_setting": {
      "emotion": "calm",
      "vol": 1.0,
      "pitch": 0
    },
    "audio_setting": {
      "sample_rate": 32000,
      "bitrate": 128000,
      "channel": 1
    }
  }
}
```

### 字段映射

| 客户端字段 | 上游 `/v2/tts` |
|------------|----------------|
| `model` | `model` |
| `input` | `text` |
| `voice` | `voice_setting.voice_id`（并补百度侧可能校验的顶层 `voiceId`） |
| `speed` | `voice_setting.speed` |
| `response_format` | `audio_setting.format`；非 `hex` 时上游 `output_format=url` |
| `metadata` | 透传：`emotion` / `vol` / `pitch` / `timbre_weights` / `pronunciation_dict` / `voice_modify` / `language_boost` / `subtitle_enable` / `aigc_watermark` 等 |

`voice_setting` 与 `timbre_weights` 按上游文档 **二选一**。  
`bitrate` 按上游枚举透传（常见 `32000` / `64000` / `128000` / `256000`）。

### 网关响应（`/v1/baidu-vod/tts`）

校验通过后**原样透传上游 JSON**，不裁剪字段。上游完整成功体示例：

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

音频地址：`data.audio`。若上游偶发简化体 `{"url":"..."}`，也会原样返回该体。

`/v1/audio/speech` 走本渠道时仍按兼容逻辑（URL→302，hex→音频二进制），不改其它渠道。

---

## 3. curl 示例

```bash
curl -sS -X POST "https://YOUR_HOST/v1/baidu-vod/tts" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "speech-2.8-hd",
    "input": "你好，欢迎使用百度智能云语音合成服务。",
    "voice": "Boyan_new_hd",
    "response_format": "mp3"
  }'
```

---

## 4. 注意

- 一期**不做**视频 / Chat / 生图。
- 签名若线上 403 `SignatureDoesNotMatch`：核对 Host、机器时间、URI，以及是否需追加 `x-bce-date`（一期默认 signedHeaders 为 `host`）。
- 真实 AK/SK 不得写入仓库或公开文档。
