# 百度 VOD Vidu 对口型（lip-sync）设计

**日期：** 2026-09-16  
**状态：** 已确认  
**范围：** 百度 VOD Vidu 渠道增加对口型异步任务  
**上游文档：** [Vidu 对口型 / lip-sync](https://platform.vidu.cn/docs/lip-sync)  
**使用说明：** [docs/channel/baidu-vod-vidu.md](../../channel/baidu-vod-vidu.md)

## 背景

百度 VOD Vidu（`ChannelTypeBaiduVodVidu = 59`）已支持视频 Task 与 `POST /v1/async/images` 生图。对口型为另一异步能力：

- 上游：`POST {base}/ent/v2/lip-sync`
- 默认 Base：`http://vod.bj.baidubce.com/v3/aigc/vd`（与现渠道一致；文档正文偶见 `/v2/aigc/vd`）
- 鉴权：`Authorization: Bearer <apikey>`
- 创建返回 `task_id`；查询可用统一 creations 接口

官网请求体无 `model` 字段，但 NewAPI 选渠道与预扣依赖模型名。

## 目标

1. `POST /v1/async/lip-sync` 创建；`GET /v1/async/lip-sync/:task_id` 查询。
2. 请求体对齐官网字段；另加网关 `model`（不发给上游）。
3. 仅百度 VOD Vidu 渠道处理该入口。
4. 创建/查询响应形态与现有视频 Task / async images 对齐。
5. 预扣按模型单价；成功后按上游 `credits` 差额结算（**1 元 = 10 积分**）。

## 非目标

- 官方 Vidu 渠道（type=52）对口型。
- 同步等到成片再返回。
- 默认注入 `moderation`。
- ak/sk 签名。
- 新建独立渠道类型。
- 改视频 / async images 既有行为。

## 决策摘要

| 项 | 决定 |
|----|------|
| 入口 | `POST /v1/async/lip-sync` + `GET /v1/async/lip-sync/:task_id` |
| 实现 | 扩展 `baidu_vod_vidu` TaskAdaptor |
| model | 客户端传（建议 `vidu-lip-sync`）；上游 body 去掉 |
| 请求字段 | 完整对齐官网 |
| 校验 | 跳过 `ValidateBasicTaskRequest`（强制 prompt）；自定义校验 |
| 预扣 | `ModelPrice × QuotaPerUnit × 分组倍率` |
| 结算 | `(credits / 10) × QuotaPerUnit × 分组倍率`，多退少补 |
| 失败 | 退预扣 |
| credits 缺失/≤0 | 保持预扣 |

## 方案对比（已选）

| 方案 | 说明 | 结论 |
|------|------|------|
| 1 扩展 baidu_vod_vidu + 新路径 | 复用 RelayTask / 轮询 | **采用** |
| 2 挂 video/generations + metadata | 与视频混、prompt 校验冲突 | 否 |
| 3 新渠道类型 | 过重 | 否 |

## 架构

```
POST /v1/async/lip-sync
  → TokenAuth + Distribute（body.model 选 type=59）
  → RelayTaskSubmit
       → action = lip-sync；自定义校验（无强制 prompt）
       → 组官网 payload（去掉 model）
       → POST {base}/ent/v2/lip-sync  Bearer
       → 按模型单价预扣 → 入库
       → 返回与视频创建相同的 JSON（含 task_id）

GET /v1/async/lip-sync/:task_id
  → RelayTaskFetch（RelayModeVideoFetchByID）
  → {code:"success", data: TaskDto}；成功时 result_url 为成片

轮询
  → FetchTask creations
  → SUCCESS：AdjustBillingOnComplete 按 credits 差额结算
  → FAILURE：退预扣
```

## 客户端协议

### 创建

`POST /v1/async/lip-sync`  
`Authorization: Bearer <NewAPI sk>`

```json
{
  "model": "vidu-lip-sync",
  "video_url": "https://example.com/a.mp4",
  "text": "你好，欢迎使用vod平台",
  "voice_id": "wumei_yujie",
  "audio_url": "",
  "speed": 1.0,
  "ref_photo_url": "",
  "volume": 0,
  "callback_url": ""
}
```

| 字段 | 必填 | 发给上游 | 说明 |
|------|------|----------|------|
| model | 是（网关） | 否 | 选渠道/预扣；建议 `vidu-lip-sync` |
| video_url | 是 | 是 | 原视频 URL |
| audio_url | 与 text 至少其一 | 是 | 与 text 同时有值时以上游规则：以 audio 为准 |
| text | 与 audio_url 至少其一 | 是 | 文案驱动 |
| speed | 否 | 是 | 默认上游 1.0；仅文字驱动 |
| voice_id | 否 | 是 | 音色；仅文字驱动 |
| ref_photo_url | 否 | 是 | 多人脸时指定目标 |
| volume | 否 | 是 | 0–10；仅文字驱动 |
| callback_url | 否 | 是 | 透传 |

创建响应（与 `POST /v1/video/generations` 相同）：

```json
{
  "id": "task_xxxx",
  "task_id": "task_xxxx",
  "object": "video",
  "model": "vidu-lip-sync",
  "status": "queued",
  "progress": 0,
  "created_at": 1710000000
}
```

### 查询

`GET /v1/async/lip-sync/:task_id` → 与 `GET /v1/video/generations/:task_id` 相同：`{code,data}` + TaskDto；成片 URL 在 `data.result_url`。

## 扣费

系统额度：`金额（元）× QuotaPerUnit`（默认 `QuotaPerUnit = 500000`）。约定 **1 元人民币 = 10 Vidu 积分**（1 积分 = 0.1 元）。

1. **预扣（提交成功）**  
   `quota_pre = ModelPrice(vidu-lip-sync) × QuotaPerUnit × GroupRatio`

2. **成功结算**（`AdjustBillingOnComplete`）  
   从任务数据读取 `credits`：  
   `quota_actual = (credits / 10) × QuotaPerUnit × GroupRatio`（`QuotaFromFloatChecked`）  
   再 `RecalculateTaskQuota`：相对预扣多退少补。  
   现有 `settleTaskBillingOnComplete` **优先**调用 `AdjustBillingOnComplete`，返回正数即结算，不因 `PerCallBilling` 跳过。

3. **失败**  
   `RefundTaskQuota` 退预扣。

4. **credits ≤ 0 或解析失败**  
   返回 0，保持预扣。

**credits 来源：** 优先轮询写入的 `task.Data`。若 creations 响应无 `credits` 而创建响应有，创建阶段须把 `credits` 持久化（例如写入 PrivateData / 合并进后续 Data），避免被覆盖后丢结算依据。

运营：渠道配置模型 `vidu-lip-sync` 及预扣用单价（可设保守估计值）。

## Adaptor 行为

1. `constant.TaskActionLipSync = "lip-sync"`。
2. 路径含 `/v1/async/lip-sync`：不调用依赖 prompt 的 `ValidateBasicTaskRequest`；自定义校验 + `storeTaskRequest`（`Prompt` 可用 `text` 或占位 `[lip-sync]`）。
3. `BuildRequestBody`：专用 payload，去掉 `model`。
4. `BuildRequestURL`：`/ent/v2/lip-sync`。
5. Header / FetchTask / ParseTaskResult / DoResponse：沿用现有（Bearer、creations、OpenAI video 形创建响应）。
6. 实现 `AdjustBillingOnComplete`：仅 lip-sync action 按 credits 结算；其它 action 返回 0。
7. `GetModelList` 增加 `vidu-lip-sync`。
8. 视频与 async images 分支不变。

## 挂载点

1. `constant/task.go`：新 action  
2. `router/video-router.go`：POST/GET `/async/lip-sync`  
3. `middleware/distributor.go`：同 async/images 分支  
4. `relay/channel/task/baidu_vod_vidu/adaptor.go`（+ 测试）  
5. 不改 `task/vidu`

## 测试与验收

1. 单测：URL、上游无 model、缺 video_url / 双空驱动 400、credits→额度公式、既有测试仍过。  
2. 不要求 CI 打真网。  
3. 人工：配 `vidu-lip-sync` 单价 → 创建拿 task_id → 查询 SUCCESS 且 result_url → 日志额度与 (credits/10)×QuotaPerUnit×分组 一致。

## 风险

1. 创建响应 `object` 仍为 `"video"`（复用 DoResponse）。  
2. creations 若无 credits，必须保留创建侧 credits，否则只能保持预扣。  
3. 预扣单价若远小于真实 credits，用户余额不足时补扣可能失败——需运营把预扣单价设够保守，或接受现有 Recalculate 失败处理。  
4. 真实 apikey 不得入库。
