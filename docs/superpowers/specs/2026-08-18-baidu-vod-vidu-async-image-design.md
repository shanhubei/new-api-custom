# 百度 VOD Vidu 异步生图设计

**日期：** 2026-08-18  
**状态：** 已确认  
**范围：** 百度 VOD Vidu 渠道增加 `reference2image`；新建异步图片 Task 入口  
**上游文档：** [Vidu 图片生成 / reference2image](https://platform.vidu.cn/docs/reference-to-image)  
**使用说明：** [docs/channel/baidu-vod-vidu.md](../../channel/baidu-vod-vidu.md)

## 背景

百度 VOD Vidu 渠道（`ChannelTypeBaiduVodVidu = 59`）已支持视频 Task：`text2video` / `img2video` / `start-end2video` / `reference2video`。

直连已验证可走百度 VOD 生图：

```
POST {base}/ent/v2/reference2image
Authorization: Bearer <apikey>
{"model":"viduq2","prompt":"生成一个打游戏的画面"}
```

Base 与视频相同：`http://vod.bj.baidubce.com/v3/aigc/vd`。

NewAPI 的 `POST /v1/images/generations` 是同步 Image Adaptor，没有 Task 轮询。官方 Vidu 渠道也未接生图。因此新增异步图片入口，复用现有 Task 提交/轮询/扣费。

## 目标

1. 客户端 `POST /v1/async/images` 创建生图任务；`GET /v1/async/images/:task_id` 查询。
2. 请求体与官网 `reference2image` 字段对齐（只换网关 URL）。
3. 仅百度 VOD Vidu 渠道处理该入口（模型如 `viduq2` / `viduq1` 配在该渠道上）。
4. 创建响应与现有视频创建一致（含 `task_id`）；查询响应与 `GET /v1/video/generations/:task_id` 相同。
5. 扣费复用现有 Task 按次预扣；不改视频路径行为。

## 非目标

- 不使用 `/v1/images/generations`。
- 网关内同步等到出图再返回 OpenAI `{data:[{url}]}`。
- 官方 Vidu 渠道（type=52）生图。
- 按上游 `credits` 差额结算。
- 默认注入 `"moderation": "disabled"`。
- ak/sk 签名。
- 新建独立渠道类型。

## 决策摘要

| 项 | 决定 |
|----|------|
| 入口 | `POST /v1/async/images` + `GET /v1/async/images/:task_id` |
| 实现 | 扩展现有 `baidu_vod_vidu` TaskAdaptor，按路径/action 分叉 |
| 请求体 | 官网顶层字段原样，不包在 `metadata` 里 |
| 上游 | `{base}/ent/v2/reference2image`；查询仍 `/ent/v2/tasks/{id}/creations` |
| 创建响应 | 与视频创建相同（`id`/`task_id`/`object`/`status`…，`object` 仍为 `"video"`） |
| 查询响应 | 与 `GET /v1/video/generations/:task_id` 相同：`{code,data}` + TaskDto |
| 模型路由 | 同一 `viduq2`：异步图片路径走生图，`/v1/video/generations` 仍走视频 |
| 扣费 | 模型单价预扣；创建失败/轮询失败退款 |

## 方案对比（已选）

| 方案 | 说明 | 结论 |
|------|------|------|
| 1 扩展 baidu_vod_vidu + 新路径 | 复用 RelayTask | **采用** |
| 2 `/v1/images/generations` 同步轮询 | 与现有图片链路混用 | 否 |
| 3 中间件改写成 video/generations | 易被当成文生视频 | 否 |

## 架构

```
POST /v1/async/images
  → TokenAuth + Distribute（按 body.model 选渠道 type=59）
  → RelayTaskSubmit
       → baidu_vod_vidu.Validate：action = reference2image
       → 从原始 JSON 组官网 payload（不含 duration/bgm 等视频字段）
       → POST {base}/ent/v2/reference2image  Bearer
       → 预扣模型单价 → 入库 Task
       → 返回与视频创建相同的 JSON（含 task_id）

GET /v1/async/images/:task_id
  → RelayTaskFetch（RelayModeVideoFetchByID）
  → {code:"success", data: TaskDto}，成功时 result_url 为图片 URL

后台轮询（现有 15s）
  → FetchTask GET {base}/ent/v2/tasks/{upstream_id}/creations
  → ParseTaskResult 与视频相同；FAILURE 退款
```

## 客户端协议

### 创建

`POST /v1/async/images`  
`Authorization: Bearer <NewAPI sk>`  
`Content-Type: application/json`

请求体（对齐官网，可选字段可省略）：

```json
{
  "model": "viduq2",
  "prompt": "生成一个打游戏的画面",
  "images": ["https://example.com/ref.png"],
  "seed": 0,
  "aspect_ratio": "16:9",
  "resolution": "1080p",
  "payload": "",
  "callback_url": ""
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| model | 是 | `viduq2`（文生图/参考生图）、`viduq1`（须有图） |
| prompt | 是 | ≤2000 字符 |
| images | 否 | URL 或 data URI；viduq2：0–7；viduq1：1–7 |
| seed | 否 | 0 或不传则上游随机 |
| aspect_ratio | 否 | 默认上游 16:9；viduq2 可 `auto` |
| resolution | 否 | 默认上游 1080p；viduq2 可 2K/4K |
| payload | 否 | 透传 |
| callback_url | 否 | 透传到上游 |
| moderation / audio / off_peak | 否 | 百度 VOD 扩展，不默认写入 |

创建响应（与现 `POST /v1/video/generations` 相同）：

```json
{
  "id": "task_xxxx",
  "task_id": "task_xxxx",
  "object": "video",
  "model": "viduq2",
  "status": "queued",
  "progress": 0,
  "created_at": 1710000000
}
```

### 查询

`GET /v1/async/images/:task_id`

成功示例（结构同视频 generations 查询；`result_url` 为图片）：

```json
{
  "code": "success",
  "data": {
    "task_id": "task_xxxx",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://.../xxx.png",
    "action": "reference2image",
    "quota": 50000
  }
}
```

`status`：`SUBMITTED` / `IN_PROGRESS` / `SUCCESS` / `FAILURE`。

## Adaptor 行为

1. 新增 `constant.TaskActionReference2Image = "reference2image"`。
2. `ValidateRequestAndSetAction`：请求路径为 `/v1/async/images` 时固定该 action，跳过视频的按图数量分支。
3. 生图组包：`common.UnmarshalBodyReusable` 解到专用 struct（指针 + omitempty），**不要**复用视频 `convertToRequestPayload`（会默认 `duration=5`、`movement_amplitude=auto`）。
4. `viduq1` 且 `images` 为空 → 400。
5. `BuildRequestURL`：该 action → `{base}/ent/v2/reference2image`。
6. Header / `FetchTask` / `ParseTaskResult` / `ConvertToOpenAIVideo` / `DoResponse` 保持现有实现。
7. 视频路径（`/v1/video/generations`）行为不变。

## 挂载点

1. `constant/task.go`：新 action。
2. `router/video-router.go`（或等价 router）：注册 POST/GET `/v1/async/images`。
3. `middleware/distributor.go`：该路径按 `model` 选渠道；GET 设 `RelayModeVideoFetchByID`，并走 `getTaskOriginModelName`。
4. `relay/constant/relay_mode.go`：`Path2RelayMode` 如需识别该 path（若 distributor 已 set 则可只改 distributor）。
5. `relay/channel/task/baidu_vod_vidu/adaptor.go`：action + 生图 payload + URL。
6. 测试：`adaptor_test.go`。

不改官方 `task/vidu`。前端渠道列表无需为生图新增 type。

## 扣费

与现有视频 Task 相同：

1. `ModelPriceHelperPerCall` × 分组倍率；生图 **不** 乘 duration。
2. 提交成功预扣；上游创建失败立刻退。
3. 轮询 `FAILURE` 走 `RefundTaskQuota`。
4. 成功不按 `credits` 补差。

运营：在百度 VOD Vidu 渠道配置 `viduq2`（及需要的 `viduq1`）及单价。同一模型名用于视频和生图时共用该单价。

## 测试与验收

1. 单测：生图 URL、纯 prompt 上游 JSON 无 `duration`/`movement_amplitude`、`aspect_ratio`/`resolution`/`seed` 出现在 body、现有 img2video 测试仍过、viduq1 无图 400。
2. 不要求 CI 打真网。
3. 人工：渠道配 `viduq2`；`POST /v1/async/images` 拿到 `task_id`；轮询 `GET /v1/async/images/{id}` 直到 `SUCCESS` 且 `result_url` 为图片。

## 风险

1. 创建响应 `object` 仍为 `"video"`（复用 DoResponse）。接受为与视频创建对齐的代价。
2. `TaskSubmitReq` 吃掉未知字段；必须从原始 body 组生图 payload。
3. `viduq2` 同时用于视频与生图，靠 **请求路径** 区分，不能靠模型名。
4. 真实 apikey 不得入库。
