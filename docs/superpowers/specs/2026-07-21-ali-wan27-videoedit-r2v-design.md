# 阿里万相 wan2.7-videoedit / wan2.7-r2v 设计

**日期：** 2026-07-21  
**状态：** 已确认（按对话决策直接落地）  
**范围：** `relay/channel/task/ali` + 异步任务结算路径

## 背景

现有阿里视频 TaskAdaptor 支持文生/图生（含 `wan2.7-i2v` 的 `input.media`），但：

1. 未支持 `wan2.7-videoedit`（视频编辑）与 `wan2.7-r2v`（参考生视频）。
2. 计费只按请求里的输出 `duration`（默认 5）做 `OtherRatios["seconds"]` 预扣。
3. 阿里对 videoedit / r2v 的账单是 **输入视频计费时长 + 输出视频时长**；现有轮询完成后 **不会** 按 `usage` 差额结算。
4. 更关键：有模型单价（`UsePrice`）的任务会把 `PerCallBilling=true`，`settleTaskBillingOnComplete` 直接跳过 `AdjustBillingOnComplete`。

## 目标

1. 支持模型：`wan2.7-videoedit`、`wan2.7-r2v`（及同前缀变体如带日期后缀）。
2. 请求形态：继续走 `POST /v1/video/generations`，媒体走 `metadata.input.media`（不新增友好顶层字段、不接 multipart 上传）。
3. 预扣：服务端保守估计（输入上限 + 输出时长），不依赖客户端声明输入秒数。
4. 结算：复用现有 15s 异步轮询；任务成功后读阿里 `usage.duration` 做差额补扣/退还。
5. 文档写清客户端调用参数与计费语义。

## 非目标

- 新开独立定时任务（已有 `asyncTaskPollHandler`）。
- 客户端友好字段（顶层 `video` / 自动映射）——二期。
- OpenAI multipart `/v1/videos` 文件上传转 URL。
- 改聊天侧 `relay/channel/ali`（与视频 task 无关）。
- 自动改运营后台模型价格（需人工配置单价）。

## 决策摘要

| 项 | 决定 |
|----|------|
| 范围 | videoedit + r2v 一起做 |
| 请求 | `metadata.input.media` 透传阿里结构 |
| 预扣 | 保守上限：videoedit ≈ `2 × max(输出,10)` 或 `2×duration`；r2v = `输出 + 5` |
| 结算 | 实现 `AdjustBillingOnComplete`，以 `usage.duration` 为准 |
| 轮询 | 不新增 cron；修 PerCallBilling 跳过逻辑，使 adaptor 结算可执行 |
| Endpoint | 仍为 `.../video-generation/video-synthesis` |

## 方案对比（已选）

| 方案 | 说明 | 结论 |
|------|------|------|
| A 仅透传 + 文档 | 渠道加模型名，靠 metadata | 协议可通，计费必错 |
| B 协议适配 + 保守预扣 + usage 结算 | 本设计 | **采用** |
| C 新独立计费服务 / 新定时任务 | 重复造轮 | 否 |

## 架构

```
POST /v1/video/generations
  → Distribute(选 Ali 渠道)
  → RelayTaskSubmit
       → convertToAliRequest
            → normalizeWan27VideoEditInput / normalizeWan27R2VInput
       → EstimateBilling（含输入视频预估秒数）
       → 预扣
       → 上游创建任务
  → 入库 Task + BillingContext

每 15s 轮询
  → FetchTask → ParseTaskResult
  → 成功：AdjustBillingOnComplete(读 usage.duration) → RecalculateTaskQuota
```

### 协议分支

| 模型前缀 | media 要求 | duration 默认 | 预扣 seconds |
|----------|------------|---------------|--------------|
| `wan2.7-i2v` | 已有：first_frame 等 | 5 | 输出时长（不变） |
| `wan2.7-videoedit` | 恰好 1 个 `video`；可选 ≤4 `reference_image` | **0**（跟原片，不强制 5） | `2 * max(duration, 10)` 若 duration≤0；否则 `2 * duration` |
| `wan2.7-r2v` | ≥1 个 `reference_image` 或 `reference_video`；可选 `first_frame`、`reference_voice` | 5 | `duration + 5`（输入账单上限约 5s） |
| 其它 | 现有逻辑 | 5 | 现有 |

### 结算

1. `settleTaskBillingOnComplete`：**先**调用 `AdjustBillingOnComplete`；仅当 adaptor 返回 0 时，才对 `PerCallBilling` 跳过 token 重算。
2. 阿里 adaptor：若模型为 videoedit/r2v，从 `task.Data` 解析 `usage.duration`（float，向上取整为计费秒）；用 `BillingContext` 的 `ModelPrice / GroupRatio / OtherRatios` 重算额度（用实际秒数替换原 `seconds` 项）。
3. 解析不到 usage 或 duration≤0：返回 0，保持预扣（偏保守预扣时略多收、不补差）。

### 结构体扩展

- `AliVideoMedia`：增加 `reference_voice`（r2v）。
- `AliVideoParameters`：增加 `ratio`、`audio_setting`。
- `AliUsage`：`duration` / `input_video_duration` / `output_video_duration` 改为 float 友好解析。

## 客户端调用参数（对外文档）

接口：`POST /v1/video/generations`  
鉴权：`Authorization: Bearer <token>`  
`Content-Type: application/json`

### 公共字段

| 字段 | 类型 | 说明 |
|------|------|------|
| `model` | string | `wan2.7-videoedit` 或 `wan2.7-r2v` |
| `prompt` | string | 编辑/生成提示词 |
| `duration` | int | 输出相关时长；videoedit 传 `0` 或不传表示跟原片；r2v 建议 2–10 |
| `size` | string | 分辨率档，如 `720P` / `1080P`（非 `宽*高`） |
| `metadata` | object | 透传上游 `input` / `parameters` |

### videoedit 示例

```json
{
  "model": "wan2.7-videoedit",
  "prompt": "将整个画面转换为黏土风格",
  "size": "720P",
  "duration": 0,
  "metadata": {
    "input": {
      "prompt": "将整个画面转换为黏土风格",
      "negative_prompt": "低质量",
      "media": [
        {
          "type": "video",
          "url": "https://example.com/source.mp4"
        },
        {
          "type": "reference_image",
          "url": "https://example.com/ref.png"
        }
      ]
    },
    "parameters": {
      "resolution": "720P",
      "duration": 0,
      "prompt_extend": true,
      "watermark": false,
      "audio_setting": "origin",
      "ratio": "16:9"
    }
  }
}
```

约束：`media` 必须含且仅含 1 个 `type=video`；`reference_image` 最多 4；源视频约 2–10s。

### r2v 示例

```json
{
  "model": "wan2.7-r2v",
  "prompt": "character1 和 character2 在雨中对话",
  "size": "1080P",
  "duration": 5,
  "metadata": {
    "input": {
      "prompt": "character1 和 character2 在雨中对话",
      "media": [
        {
          "type": "reference_image",
          "url": "https://example.com/girl.jpg",
          "reference_voice": "https://example.com/girl.mp3"
        },
        {
          "type": "reference_video",
          "url": "https://example.com/boy.mp4",
          "reference_voice": "https://example.com/boy.mp3"
        }
      ]
    },
    "parameters": {
      "resolution": "1080P",
      "duration": 5,
      "ratio": "16:9",
      "prompt_extend": true,
      "watermark": false
    }
  }
}
```

约束：参考图+参考视频总数 ≤5；参考视频 0–3；可选 1 个 `first_frame`。

### 查询

`GET /v1/video/generations/{task_id}`  
完成后结果 URL 在任务详情 / OpenAI Video `metadata.url`（与现有阿里行为一致）。

### 运营配置

1. 渠道类型：阿里（17）；Base URL 与 Key 同地域。  
2. 渠道模型列表加入 `wan2.7-videoedit`、`wan2.7-r2v`。  
3. 模型价格按「每秒」配置（与现有视频秒计费一致）；账单秒数 = 阿里 `usage.duration`。  
4. 保持 `UPDATE_TASK` / 异步轮询开启。

## 测试计划

- 单测：videoedit / r2v 的 media 校验、duration=0 不强制改成 5、EstimateBilling 秒数、AdjustBillingOnComplete 用 usage 重算。
- 单测：`settleTaskBillingOnComplete` 在 PerCallBilling 下仍执行 adaptor 正数结算。
- 手工：真实渠道各跑一条成功任务，核对预扣与结算日志。

## 风险

- 保守预扣可能暂时多扣，成功后按 usage 退差；若余额刚好够「偏低预估」但不够真实账单，补扣可能失败——预扣已按上限偏高，降低该风险。
- 上游 `usage.duration` 缺失时保持预扣，可能略高于真实账单。
