# 百度 VOD Vidu 渠道使用说明

渠道类型：**Baidu VOD Vidu**（type `59`）  
鉴权：网关使用 NewAPI Token（`Authorization: Bearer sk-...`）；上游使用渠道 Key（Bearer apikey）。  
默认上游 Base：`http://vod.bj.baidubce.com/v3/aigc/vd`（可在渠道里改）。

设计稿（实现细节）：

- [渠道](../superpowers/specs/2026-08-11-baidu-vod-vidu-channel-design.md)
- [异步生图](../superpowers/specs/2026-08-18-baidu-vod-vidu-async-image-design.md)
- [对口型](../superpowers/specs/2026-09-16-baidu-vod-vidu-lip-sync-design.md)

上游官方：[Vidu 对口型](https://platform.vidu.cn/docs/lip-sync) · [参考生图](https://platform.vidu.cn/docs/reference-to-image)

---

## 1. 后台配置

1. 渠道 → 新建 → 类型选 **Baidu VOD Vidu**。
2. 填入百度 VOD Vidu apikey；Base URL 一般用默认值。
3. 模型列表按需配置，例如：
   - 视频：`viduq3-pro` / `viduq2` / `viduq1` / `vidu2.0` / `vidu1.5`
   - 异步生图：同上（常用 `viduq2`、`viduq1`）
   - 对口型：建议单独配 **`vidu-lip-sync`**（网关选渠道与预扣用，**不会**发给上游）
4. 为各模型设置单价（按次预扣）。对口型预扣建议设得偏保守，成功后会按上游 `credits` 多退少补。

---

## 2. 视频生成

`POST /v1/video/generations`  
`GET /v1/video/generations/:task_id`

与官方 Vidu 任务用法相同：无图 → 文生视频；1 图 → 图生；2 图 → 首尾帧；多于 2 图 → 参考图。可选扩展可通过 `metadata` 传 `moderation` / `audio` / `off_peak` 等（不默认注入免审核）。

查询时同样保持原有 TaskDto 结构；官方 creations 响应全文在 **`data.data`**（字段同下方异步生图 / 对口型示例）。

---

## 3. 异步生图

### 创建

`POST /v1/async/images`  
`Authorization: Bearer <sk>`  
`Content-Type: application/json`

```json
{
  "model": "viduq2",
  "prompt": "生成一个打游戏的画面",
  "images": ["https://example.com/ref.png"],
  "seed": 0,
  "aspect_ratio": "16:9",
  "resolution": "1080p",
  "callback_url": ""
}
```

| 字段 | 必填 | 说明 |
|------|------|------|
| model | 是 | 如 `viduq2`；`viduq1` 必须带图 |
| prompt | 是 | 提示词 |
| images | 否 | URL 或 data URI；viduq2：0–7；viduq1：1–7 |
| seed / aspect_ratio / resolution / payload / callback_url | 否 | 对齐官网；可省略 |

创建响应示例：

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

网关仍返回既有任务结构（`code` + TaskDto），**不改字段名**。官方 creations 完整响应放在 TaskDto 的 **`data`** 字段里（轮询写入的上游原文）。

成功示例：

```json
{
  "code": "success",
  "data": {
    "task_id": "task_xxxx",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://.../xxx.png",
    "action": "reference2image",
    "quota": 50000,
    "data": {
      "state": "success",
      "err_code": "",
      "credits": 4,
      "payload": "",
      "creations": [
        {
          "id": "your_creations_id",
          "url": "https://.../xxx.png",
          "cover_url": "https://.../cover.png"
        }
      ]
    }
  }
}
```

| 位置 | 说明 |
|------|------|
| `data.status` / `data.result_url` / `data.progress` … | 网关任务字段（原有结构） |
| `data.data` | **官方查询响应全文**（`state` / `err_code` / `credits` / `payload` / `creations` 等带齐） |

`status`：`SUBMITTED` / `IN_PROGRESS` / `SUCCESS` / `FAILURE`。成图快捷字段：`data.result_url`（一般等于 `data.data.creations[0].url`）。

**扣费：** 按模型单价预扣；失败退预扣；成功不按 `credits` 补差。

---

## 4. 对口型（lip-sync）

### 创建

`POST /v1/async/lip-sync`  
`Authorization: Bearer <sk>`  
`Content-Type: application/json`

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
| model | 是 | 否 | 选渠道/预扣；建议 `vidu-lip-sync` |
| video_url | 是 | 是 | 原视频 URL |
| audio_url | 与 text 至少其一 | 是 | 与 text 同时有值时以上游规则：以 audio 为准 |
| text | 与 audio_url 至少其一 | 是 | 文案驱动 |
| speed | 否 | 是 | 默认上游约 1.0；仅文字驱动 |
| voice_id | 否 | 是 | 音色；仅文字驱动 |
| ref_photo_url | 否 | 是 | 多人脸时指定目标 |
| volume | 否 | 是 | 0–10；仅文字驱动 |
| callback_url | 否 | 是 | 透传 |

创建响应形态与视频创建相同（含 `task_id`；`object` 仍为 `"video"`）。

### 查询

`GET /v1/async/lip-sync/:task_id`

与 `GET /v1/video/generations/:task_id` 相同：外层任务结构不变；官方 creations 全文在 **`data.data`**。

成功示例：

```json
{
  "code": "success",
  "data": {
    "task_id": "task_xxxx",
    "status": "SUCCESS",
    "progress": "100%",
    "result_url": "https://.../lip-sync.mp4",
    "action": "lip-sync",
    "quota": 6000000,
    "data": {
      "state": "success",
      "err_code": "",
      "credits": 12,
      "payload": "",
      "creations": [
        {
          "id": "your_creations_id",
          "url": "https://.../lip-sync.mp4",
          "cover_url": "https://.../cover.png"
        }
      ]
    }
  }
}
```

| 位置 | 说明 |
|------|------|
| `data.result_url` | 成片快捷字段（原有） |
| `data.data` | 官方查询响应全文（须带齐 `state` / `err_code` / `credits` / `payload` / `creations[].id|url|cover_url`） |
| `data.data.credits` | 结算依据（**1 元 = 10 积分**，即 1 积分 = 0.1 元） |

成片 URL 有效期以上游为准（官网通常约 24 小时）。

### 扣费

约定：**1 元人民币 = 10 上游积分**（1 积分 = 0.1 元；额度 = 金额元 × QuotaPerUnit，默认 `QuotaPerUnit = 500000`）。

1. **预扣（提交成功）**  
   `ModelPrice(vidu-lip-sync) × QuotaPerUnit × 分组倍率`
2. **成功结算**  
   `(credits / 10) × QuotaPerUnit × 分组倍率`，相对预扣多退少补  
3. **失败**  
   退预扣  
4. **credits 缺失或 ≤ 0**  
   保持预扣不变  

---

## 5. curl 示例（对口型）

```bash
# 创建
curl -sS -X POST "https://YOUR_HOST/v1/async/lip-sync" \
  -H "Authorization: Bearer sk-xxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "vidu-lip-sync",
    "video_url": "https://example.com/a.mp4",
    "text": "你好，欢迎使用vod平台",
    "voice_id": "wumei_yujie"
  }'

# 查询（替换 task_xxxx）
curl -sS "https://YOUR_HOST/v1/async/lip-sync/task_xxxx" \
  -H "Authorization: Bearer sk-xxx"
```

---

## 6. 注意

- 本渠道与官方 **Vidu**（type 52）相互独立；对口型 / `/v1/async/*` 仅百度 VOD Vidu 处理。
- 创建响应里的 `object` 复用视频任务形态，可能为 `"video"`（含生图、对口型）。
- 查询不改动原有任务字段；官方返回通过既有字段 **`data`（即 `data.data`）** 原样带回，勿拆散到其它键。
- 真实 apikey 勿写入仓库或公开文档。
