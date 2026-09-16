# 百度 VOD Vidu 渠道设计

**日期：** 2026-08-11  
**状态：** 已确认  
**范围：** 新增 Task 渠道类型「百度 VOD Vidu」（Vidu 协议透传）  
**使用说明：** [docs/channel/baidu-vod-vidu.md](../../channel/baidu-vod-vidu.md)

## 背景

项目已有官方 Vidu 任务渠道（`constant.ChannelTypeVidu = 52`，`relay/channel/task/vidu`）：

- 路径：`{base}/ent/v2/{text2video|img2video|start-end2video|reference2video}`
- 查询：`{base}/ent/v2/tasks/{id}/creations`
- 鉴权：`Authorization: Token <key>`
- 默认 Base：`https://api.vidu.cn`

百度 VOD 提供 Vidu「透传」接口：在原厂路径前增加前缀（文档正文写 `/v2/aigc/vd`，文档 curl 与已验证直连脚本使用 `/v3/aigc/vd`），协议体与原厂对齐，鉴权为 Bearer apikey。

直连已验证可生成视频：

- Base：`http://vod.bj.baidubce.com/v3/aigc/vd`
- 创建：`POST {base}/ent/v2/img2video`
- 查询：`GET {base}/ent/v2/tasks/{task_id}/creations`
- Header：`Authorization: Bearer <bce-v3/ALTAK-...>`

需要在 NewAPI 中增加对应渠道，使客户端仍走统一的 `POST /v1/video/generations`。

## 目标

1. 新增独立渠道类型，后台可选「百度 VOD Vidu」。
2. 能力与现有 Vidu 对齐：文生 / 图生 / 首尾帧 / 参考图。
3. 默认 Base 指向已验证的百度 VOD 前缀；创建与轮询均使用 Bearer apikey。
4. 不默认注入免审核字段；客户端可通过 `metadata` 传入 `moderation` 等扩展参数。

## 非目标

- 百度 ak/sk 签名鉴权（仅 apikey + Bearer）。
- 默认自动写入 `"moderation": "disabled"`。
- 接入百度原生非 Vidu 接口（例如 `/v3/aigc/text_to_video` + P40）；该接口与本渠道无关。
- 修改现有官方 Vidu 渠道行为。
- 新增独立计费逻辑（沿用现有 task 预扣/结算）。

## 决策摘要

| 项 | 决定 |
|----|------|
| 实现方式 | **方案 2**：新渠道类型 + 整包复制 `task/vidu` 后改差异点 |
| 渠道 ID | `ChannelTypeBaiduVodVidu = 59`（插在 `ChannelTypeDummy` 之前） |
| 展示名 | `Baidu VOD Vidu` |
| 默认 Base | `http://vod.bj.baidubce.com/v3/aigc/vd`（可配置；若需文档中的 v2 前缀可改 Base） |
| 鉴权 | 仅 `Authorization: Bearer <apikey>`（创建 + FetchTask） |
| 能力范围 | 与 Vidu 完全对齐（文生/图生/首尾帧/参考图） |
| 免审核 | 不默认注入；由客户端 / `metadata` 传入 |
| 扩展字段 | 在复制的 `requestPayload` 上增加可选指针字段：`moderation` / `audio` / `off_peak`（`omitempty`），保证 metadata 可进上游 |
| 模型列表 | 在 Vidu 列表基础上增加 `viduq3-pro`；实际以渠道配置为准 |

## 方案对比（已选）

| 方案 | 说明 | 结论 |
|------|------|------|
| 1 新 type + 薄封装复用 | 抽公共逻辑，仅改 Auth/Base | 否（用户选 2） |
| 2 新 type + 整包复制 Vidu | 拷贝 adaptor，改差异点 | **采用** |
| 3 扩展现有 Vidu | 按 Base/配置切换 Token/Bearer | 否（官方与百度耦合） |

## 架构

```
客户端 POST /v1/video/generations
  → 选渠道 type=59 (Baidu VOD Vidu)
  → GetTaskAdaptor → baidu_vod_vidu.TaskAdaptor
  → BuildRequestURL: {ChannelBaseUrl}/ent/v2/{action}
  → BuildRequestHeader: Bearer + ApiKey
  → 上游创建 → 返回 task_id → 入库

轮询
  → FetchTask: GET {base}/ent/v2/tasks/{id}/creations
  → Header: Bearer
  → ParseTaskResult（状态映射与 Vidu 相同）
```

### 路径约定

渠道 **Base URL 已包含** VOD 前缀（默认 `/v3/aigc/vd`）。Adaptor 只拼接原厂后缀，与现有 Vidu 的 `BuildRequestURL` / `FetchTask` 路径拼接方式一致：

| Action | 上游路径 |
|--------|----------|
| 文生 | `{base}/ent/v2/text2video` |
| 图生（1 图） | `{base}/ent/v2/img2video` |
| 首尾帧（2 图） | `{base}/ent/v2/start-end2video` |
| 参考图（>2 图） | `{base}/ent/v2/reference2video` |
| 查询生成物 | `{base}/ent/v2/tasks/{id}/creations` |

说明：文档正文前缀为 `/v2/aigc/vd`，curl 与直连脚本为 `/v3/aigc/vd`。以可配置 Base 解决；默认用已验证的 v3。

### 相对官方 Vidu 的代码差异（复制后必改）

| 位置 | Vidu | 百度 VOD Vidu |
|------|------|----------------|
| package / 目录 | `relay/channel/task/vidu` | `relay/channel/task/baidu_vod_vidu` |
| `BuildRequestHeader` / `FetchTask` | `Token ` + key | `Bearer ` + key |
| `GetChannelName` | `vidu` | `baidu_vod_vidu`（或同等清晰名） |
| `ValidateRequestAndSetAction` 中 type 判断 | `ChannelTypeVidu` | `ChannelTypeBaiduVodVidu` |
| `GetModelList` | viduq2/viduq1/... | 同上 + `viduq3-pro` |
| `requestPayload` | 无 moderation/audio/off_peak | 增加可选指针字段 |

其余：请求/响应结构、`DoResponse`、`ParseTaskResult`、`ConvertToOpenAIVideo`、参考图模型名规范化（`viduq2*` → `viduq2`）均保持与复制源一致。

## 挂载点清单

### 后端

1. `constant/channel.go`：常量、默认 Base、`ChannelTypeNames`
2. `relay/channel/task/baidu_vod_vidu/adaptor.go`：复制并修改
3. `relay/relay_adaptor.go`：`GetTaskAdaptor` case
4. `controller/channel-test.go`：与 Vidu 一并列入 unsupported 测试渠道（异步视频）
5. 如有定价展示映射（`model/pricing_default.go`）：按需增加名称映射

### 前端

1. `web/default/src/features/channels/constants.ts`：`CHANNEL_TYPES` + 展示顺序
2. `web/default/src/features/channels/lib/channel-utils.ts`：图标/标签映射
3. `web/classic/src/constants/channel.constants.js`：渠道选项
4. i18n：渠道展示名若走翻译键则补各 locale（专有名词可保留英文）

## 错误与状态

与 Vidu 相同：

- 创建响应 `state=failed` → 本地 TaskError
- 轮询：`created`/`queueing` → Submitted；`processing` → InProgress；`success` → Success（`creations[0].url`）；`failed` → Failure（`err_code`）

## 测试与验收

1. **单测（推荐）：** `BuildRequestURL` 拼接、`BuildRequestHeader` 为 Bearer、`metadata` 中 `moderation`/`audio`/`off_peak` 出现在上游 body。
2. **不**要求 CI 打真网。
3. **人工验收：** 渠道 Base/Key 配百度 VOD；`POST /v1/video/generations`（模型如 `viduq3-pro` + 图）→ 轮询拿到视频 URL；需要免审核时在 `metadata` 带 `"moderation": "disabled"`。

## 配置说明（运营）

- 渠道类型：Baidu VOD Vidu (59)
- Base URL：`http://vod.bj.baidubce.com/v3/aigc/vd`（勿再手动拼一层 `/ent/v2`）
- Key：百度 VOD apikey（形如 `bce-v3/ALTAK-.../...`），Header 自动加 `Bearer `
- 模型：按开通情况配置（至少验证 `viduq3-pro`）

## 风险与注意

1. **v2 vs v3 前缀：** 以可配置 Base 兜底；默认 v3。
2. **整包复制维护成本：** 官方 Vidu 后续修复不会自动同步；接受为方案 2 的代价。
3. **勿与百度原生 P40 等接口混淆：** 本渠道只服务 `/v3/aigc/vd` + Vidu `/ent/v2/...` 透传。
4. **密钥安全：** 文档/脚本中的真实 key 不得写入仓库。
