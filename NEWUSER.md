# 团队用户模块（newuser）

> **仓库说明**  
> 本文档属于二次开发仓库 [shanhubei/new-api-custom](https://github.com/shanhubei/new-api-custom)。  
> 基于上游 [QuantumNous/new-api](https://github.com/QuantumNous/new-api)（AGPL-3.0）增量扩展。

**newuser** 在不改动原有 `/api/user/login`、relay 计费与 Token 鉴权的前提下，为每个组织主账号挂载独立的「团队用户」子账号（桌面端 / 外部 UI 使用）。

---

## 目录

- [快速约定（先读）](#快速约定先读)
- [两套账号体系](#两套账号体系)
- [设计目标与架构](#设计目标与架构)
- [环境变量与部署](#环境变量与部署)
- [组织主账号（users）](#组织主账号users)
- [团队用户（newusers）](#团队用户newusers)
- [管理后台 UI](#管理后台-ui)
- [上线步骤](#上线步骤)
- [API 参考](#api-参考)
- [主账号短信认证与防轰炸](#主账号短信认证与防轰炸)
- [计费与限额](#计费与限额)
- [数据库](#数据库)
- [升级维护](#升级维护)
- [常见问题](#常见问题)
- [集成示例](#集成示例)

---

## 快速约定（先读）

| 角色 | 登录哪里 | 鉴权 | 用途 |
|------|----------|------|------|
| **组织管理员** | Web `/sign-in` → `POST /api/user/login` | Cookie Session | 管理后台、充值、管团队用户 |
| **团队用户** | **仅桌面客户端** → `POST /api/newuser/login` | JWT + `sk-` | 调 AI、查自己用量 |

- 自带 Web **只提供团队注册页** `/newuser/register?code=`，**不提供**团队 Web 登录。
- 团队用户费用走**组织主账号钱包**；每人绑定组织下的一个 `sk-` Token。
- 组织邀请码 `register_code` **全局唯一**；凭码即可归属团队（可不传 `owner_user_id`）。
- 团队找回密码：`/api/newuser/reset_password`（邮箱）/ `/api/newuser/reset_password/sms`（短信），与主账号 `/api/user/reset*` 分离。
- 主账号改密（`PUT /api/user/self`）会同步 `is_org_owner` 的 newuser；团队用户改密用 `PUT /api/newuser/self/password`。

```
管理员：Web 登录 → /newusers 建用户 / 开自助注册 / 分享注册链接
团队成员：打开 /newuser/register?code=xxx 注册 → 桌面端 /api/newuser/login → sk- 调 /v1/*
```

---

## 两套账号体系

| 类型 | 用户表 | 登录接口 | 鉴权 | 典型用途 |
|------|--------|----------|------|----------|
| **组织主账号** | `users` | `POST /api/user/login` | Cookie Session | 管理后台、充值、创建团队用户 |
| **团队用户** | `newusers` | `POST /api/newuser/login` | JWT Bearer | 桌面端 / 外部 UI |

关系：**一个主账号 = 一个组织**，其下可有多个团队用户。

| 对比项 | 主账号 | 团队用户 |
|--------|--------|----------|
| 响应 token | 无（Cookie） | JWT + `sk-` api_key |
| 管理团队用户 | 可以 | 不可以（除非 `is_org_owner`） |
| 调 `/v1/*` | 需自行创建 API Key | 登录直接返回 `sk-` |

---

## 设计目标与架构

| 目标 | 说明 |
|------|------|
| 不破坏原有登录 | 管理后台仍走 `/api/user/login`（Cookie） |
| 增量扩展 | 团队能力走 `/api/newuser/*`，JWT 与 Session 隔离 |
| 组织共用钱包 | 每人映射组织下的一个 **sk- Token**，扣主账号 Quota |
| 计费不变 | AI 仍走 `/v1/*`，不改 relay 核心 |
| 用量与限额 | 单用户统计 / 限额在 newuser 模块内 |

```
┌──────────────────┐   POST /api/user/login      ┌──────────────────┐
│  组织主账号       │ ─────────────────────────► │   new-api        │
│  (管理后台 Web)   │ ◄── Cookie Session ──────── │   users 表       │
└────────┬─────────┘                             └────────┬─────────┘
         │ 创建团队用户 / 开自助注册                          │ 钱包 Quota
         ▼                                                  │
┌──────────────────┐   POST /api/newuser/login     ┌────────▼─────────┐
│  桌面客户端       │ ───────────────────────────► │  newuser 模块    │
│  (团队用户)       │ ◄── JWT + sk- api_key ────── │  newusers 表     │
└────────┬─────────┘                               └──────────────────┘
         │ Authorization: Bearer sk-xxx
         ▼
┌──────────────────┐
│  /v1/chat/...    │  ← relay（原有，未改动）
└──────────────────┘
```

```
组织主账号 (User.id = owner_user_id)
  ├── 钱包 Quota（统一扣费）
  └── Token 列表
        └── newuser:alice
        └── newuser:bob

newusers
  ├── owner_user_id → 组织主账号
  ├── username / password（全站唯一）
  ├── email / phone（选填）
  └── token_id → 上述 Token
```

---

## 环境变量与部署

```env
# 启用团队用户模块（默认 false）
NEWUSER_ENABLED=true

# 【可选】自助注册未带 owner_user_id、也未带 register_code 时的默认组织
# 推荐：成员注册只传全局唯一 register_code，一般不必设此项
# NEWUSER_DEFAULT_OWNER_ID=2

# JWT 密钥（可选，默认 SESSION_SECRET）
# NEWUSER_JWT_SECRET=your-random-secret

# JWT 有效期（小时，默认 168 = 7 天）
# NEWUSER_JWT_EXPIRE_HOURS=168

# 允许 register_type=org 同时注册 users + newusers（桌面端组织自助注册）
# NEWUSER_ORG_SELF_REGISTER_ENABLED=true

# 建议固定，避免重启后 Session/JWT 失效
# SESSION_SECRET=your-fixed-random-string
```

```yaml
services:
  new-api:
    environment:
      - NEWUSER_ENABLED=true
      # - NEWUSER_ORG_SELF_REGISTER_ENABLED=true
```

```bash
docker compose build && docker compose up -d
```

---

## 组织主账号（users）

接口为系统原有 API，**未被 newuser 修改**。用于管理后台、充值、管理团队用户。

### 注册 `POST /api/user/register`

需开启系统注册。若开启邮箱/短信验证，按主站规则带验证码。

```json
{
  "username": "myorg",
  "password": "12345678",
  "password2": "12345678",
  "email": "admin@example.com",
  "verification_code": "123456",
  "aff_code": ""
}
```

成功后该用户的 `id` 即为团队用户的 `owner_user_id`。

### 登录 `POST /api/user/login`

```json
{ "username": "myorg", "password": "12345678" }
```

响应通过 `Set-Cookie` 写 **Session**（非 JWT）。若开启 2FA，需再调 `POST /api/user/login/2fa`。

| role | 含义 |
|------|------|
| `1` | 普通用户（可作组织主账号） |
| `10` | 管理员 |
| `100` | Root |

相关：`GET /api/user/logout`、`GET /api/user/self`（`quota` / `used_quota` 即组织钱包；团队侧的 `owner_quota` / `owner_used_quota` 来自此）。

### 桌面端：组织管理员怎么调管理 API

主账号登录**不会**在 JSON 里返回 token。推荐：

1. 浏览器登录后台 → `GET /api/user/token` 拿到 Access Token，记下用户 `id`
2. 桌面端请求同时带：

```http
Authorization: Bearer <ACCESS_TOKEN>
New-Api-User: <USER_ID>
```

也可用 Cookie Jar / 内嵌 WebView；纯 `fetch` 默认不跨请求保 Cookie。

团队成员（终端用户）**不要**走主账号登录，走 `POST /api/newuser/login`。

---

## 团队用户（newusers）

### 多租户

- 每个主账号 = 一个组织；只能管理自己名下的团队用户。
- **用户名全站唯一**（跨组织也不能重名）。
- 登录只需 `username` + `password`，不必传 `owner_user_id`。

### 统一注册 `POST /api/newuser/register`

一次注册返回 JWT + `sk-`，适合桌面端。

| register_type | 关键参数 | 行为 |
|---------------|----------|------|
| `"org"` | 不传邀请码 | 同时建 `users` + `newusers`（`is_org_owner=true`）；需 `NEWUSER_ORG_SELF_REGISTER_ENABLED=true` |
| `"member"` | **全局唯一** `register_code` | 只建 `newusers`，挂到邀请码对应组织；需组织开启自助注册 |

未传 `register_type`：有 `owner_user_id` 或 `register_code` → `member`；否则 → `org`。

#### 请求字段

| 字段 | 必填 | 说明 |
|------|------|------|
| `username` | 是 | 全站唯一 |
| `password` | 是 | 8–64 位 |
| `display_name` | 否 | 默认等于 username |
| `register_type` | 否 | `"org"` / `"member"` |
| `register_code` | member 时是 | 组织邀请码，**全局唯一** |
| `owner_user_id` | 否 | 有 `register_code` 时可不传 |
| `email` | 否 | 选填 |
| `phone` | 否 | 选填 |
| `verification_code` | 条件 | 见下表 |

#### 成员注册示例

```json
{
  "register_type": "member",
  "register_code": "TEAM-UNIQUE-CODE",
  "username": "alice",
  "password": "12345678",
  "display_name": "Alice",
  "email": "alice@example.com",
  "verification_code": "AB12CD"
}
```

短信路径：传 `phone` + `verification_code`，不要同时强验证邮箱（见规则）。

#### 邮箱 / 短信验证（桌面端）

团队注册**复用**主站发码接口，再在注册体里带 `verification_code`。

| 步骤 | 邮箱 | 短信 |
|------|------|------|
| 1. 发码 | `GET /api/verification?email=` | `GET /api/verification/sms?phone=` |
| 2. 注册 | JSON：`email` + `verification_code` | JSON：`phone` + `verification_code` |

**Turnstile：** 后台**未开启**时可省略 `turnstile`；**开启**时必须 `&turnstile=<有效 token>`，空值会报错。桌面端通常关闭 Turnstile，依赖短信防轰炸。

| 场景 | 是否要 `verification_code` |
|------|----------------------------|
| 不填 email、不填 phone | 不需要 |
| 填了 email，且开启邮箱验证 | 需要（先 `/api/verification`） |
| 填了 phone，且开启短信验证 | 需要（先 `/api/verification/sms`） |
| 邮箱与短信验证都开，且同时传 email + phone | **不允许**，二选一 |
| 对应验证开关关闭 | 可只存联系方式，不校验码 |

```javascript
const qs = new URLSearchParams({ phone });
// if (turnstileEnabled) qs.set('turnstile', turnstileToken);
await fetch(`${base}/api/verification/sms?${qs}`);

await fetch(`${base}/api/newuser/register`, {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    register_type: 'member',
    register_code: inviteCode,
    username,
    password,
    phone,
    verification_code: code,
  }),
});
// data.token → /api/newuser/* ；data.api_key → /v1/*
```

#### 组织自助注册（可选）

`register_type: "org"` 需 `NEWUSER_ORG_SELF_REGISTER_ENABLED=true`，会同时创建主账号 + 团队表 `is_org_owner` 记录 + sk-。

登录/注册成功额外字段：`is_org_owner`、`can_manage_users`、`register_type`。  
`is_org_owner=true` 可用 JWT 调 `/api/newuser/manage/*`（能力同 `/api/newuser/admin/*`，无需 Cookie）。

---

## 管理后台 UI

路径：侧边栏 **个人 → 团队用户**（`/newusers`，Default 主题）。

| 功能 | 说明 |
|------|------|
| 组织设置 | 启用团队用户、自助注册、**全局唯一**邀请码、复制分享链接 |
| 用户列表 | 用户名、显示名、邮箱、手机、状态、用量、限额 |
| 新建/编辑 | 可选填邮箱/手机；改密码 / 限额 / 状态 |
| 用量 | 本用户用量 + **组织钱包** + 最近日志 |

### 自助注册与邀请码

- `register_code` 全局唯一，冲突无法保存。
- 分享链接：`/newuser/register?code=<唯一码>`（Web **仅注册**）。
- 桌面端/外部 UI：`POST /api/newuser/register` + `register_code`。
- 校验邀请码：`GET /api/newuser/register/info?code=`。

---

## 上线步骤

1. `NEWUSER_ENABLED=true`，重建重启。
2. 注册/登录组织主账号（Web 或 `/api/user/*`）。
3. 开启本组织能力：

```http
PUT /api/newuser/admin/settings
Content-Type: application/json

{
  "enabled": true,
  "register_enabled": true,
  "register_code": "TEAM-UNIQUE-CODE"
}
```

4. 创建团队用户：后台新建 / `POST /api/newuser/admin/users` / 自助注册。
5. 桌面端：`POST /api/newuser/login` → JWT 调 `/api/newuser/*`，`sk-` 调 `/v1/*`。

管理端创建示例：

```json
{
  "username": "soubao123",
  "password": "12345678",
  "display_name": "搜宝用户",
  "email": "user@example.com",
  "phone": "13800138000",
  "quota_limit": 0
}
```

---

## API 参考

### 公共（无需登录）

#### `POST /api/newuser/login`

```json
{ "username": "soubao123", "password": "12345678" }
```

`owner_user_id` 可选（传入则额外校验归属）。

成功 `data` 主要字段：

| 字段 | 说明 |
|------|------|
| `token` | JWT，用于 `/api/newuser/*` |
| `api_key` | `sk-`，用于 `/v1/*` |
| `owner_*` | 组织 ID / 名 / 钱包余量与已用 |
| `is_org_owner` / `can_manage_users` / `register_type` | 角色元数据 |

#### `POST /api/newuser/register`

见上文 [统一注册](#统一注册-post-apinewuserregister)。字段与发码规则以该节为准，此处不重复。

#### `GET /api/newuser/register/info?code=`

校验邀请码，返回所属组织摘要（Web 注册页用）。

#### 团队找回密码（桌面端）

查的是 **`newusers` 表**，**不要**用 `/api/user/reset*`（那是主账号）。

前提：账号注册/创建时已填写对应 `email` 或 `phone`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/newuser/reset_password?email=` | 发邮箱验证码（邮件正文是短码，不是 Web 链接） |
| GET | `/api/newuser/reset_password/sms?phone=` | 发短信验证码 |
| POST | `/api/newuser/reset` | `{ "email", "verification_code" }` → 返回新密码 |
| POST | `/api/newuser/reset/sms` | `{ "phone", "verification_code" }` → 返回新密码 |

- 发码接口默认**防枚举**：无论是否存在均返回 `success: true`。
- 桌面端发码传 `require_exists=1`：未绑定邮箱/手机时返回 `success: false` 与明确提示，存在才真正发码。
- `turnstile`：后台未开启 Turnstile 时可省略。
- 短信需开启 `SmsVerificationEnabled` 或 `SmsLoginEnabled`，并配置阿里云。
- 验证码 purpose 与主账号隔离（`nr` / `nsp`），互不影响。

**邮箱示例：**

```http
GET /api/newuser/reset_password?email=alice@example.com

POST /api/newuser/reset
Content-Type: application/json

{ "email": "alice@example.com", "verification_code": "AB12CD" }
```

成功：`{ "success": true, "data": "<新密码>" }`，桌面端展示给用户后用新密码登录。

**短信示例：**

```http
GET /api/newuser/reset_password/sms?phone=13800138000

POST /api/newuser/reset/sms
Content-Type: application/json

{ "phone": "13800138000", "verification_code": "123456" }
```

成功同样返回 `data` 为新密码字符串。

#### 登录后修改自己的密码

| 角色 | 接口 | 说明 |
|------|------|------|
| **组织主账号**（Web） | `PUT /api/user/self` | 传 `original_password` + `password`；若存在 `is_org_owner` 的 newuser，**两边密码一起改** |
| **团队用户**（桌面 JWT） | `PUT /api/newuser/self/password` | 只改 `newusers`；若本人是 `is_org_owner`，同时同步到 `users` |

团队用户改密示例：

```http
PUT /api/newuser/self/password
Authorization: Bearer <JWT>
Content-Type: application/json

{
  "original_password": "旧密码",
  "password": "新密码至少8位"
}
```

主账号（Web 个人设置改密）走现有 `PUT /api/user/self`，服务端会自动同步 org-owner 的 newuser 密码。

---

### 团队用户（JWT）

请求头：`Authorization: Bearer <login 返回的 token>`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/newuser/self` | 个人信息与用量摘要 |
| PUT | `/api/newuser/self/password` | 修改自己的密码（`original_password` + `password`） |
| GET | `/api/newuser/token` | 重新获取 sk- |
| GET | `/api/newuser/usage` | 详细用量 + 组织钱包 |

`usage` 要点：

| 字段 | 含义 |
|------|------|
| `used_quota` / `remain_quota` | **本团队用户** Token 用量 |
| `unlimited` | Token 是否不限额（`true` 时费用走组织钱包） |
| `quota_limit` | 单用户上限（0 = 不限） |
| `owner_quota` | **组织钱包剩余**（能否继续调用看此值） |
| `owner_used_quota` | 组织历史总消耗 |

> `unlimited: true` 且 `remain_quota: 0` **不代表**组织没钱，请看 `owner_quota`。

---

### 管理接口

| 前缀 | 鉴权 | 适用 |
|------|------|------|
| `/api/newuser/admin/*` | 主账号 Cookie / Access Token | Web 管理后台 |
| `/api/newuser/manage/*` | 团队 JWT 且 `is_org_owner` | 桌面端组织主 |

路径能力相同：

| 方法 | 路径后缀 | 说明 |
|------|----------|------|
| GET/POST | `/users` | 列表 / 创建 |
| GET | `/users/:id/usage` | 单用户用量 |
| PUT/DELETE | `/users/:id` | 更新 / 禁用 |
| GET/PUT | `/settings` | 组织配置 |

---

## 主账号短信认证与防轰炸

本节为 **`users` 主账号** 能力（挂在 `/api/*`，**不是** `/api/newuser/*`）。团队成员注册发码可复用其中的 `/api/verification`、`/api/verification/sms`。

### 开关

| 配置 | 说明 |
|------|------|
| `SmsVerificationEnabled` | 注册可用短信验证 |
| `SmsLoginEnabled` | 登录短信 Tab；忘记密码短信路径依赖「登录或注册短信」任一开启 |
| 阿里云短信 | AccessKey、签名、模板（须为**数字验证码**模板） |
| `/api/status` | 返回 `sms_verification`、`sms_login` |

### 防轰炸

| 层级 | 默认 | 配置 |
|------|------|------|
| IP 短时 | 30 秒最多 2 次 | `SmsIPMaxRequests` / `SmsIPWindowSeconds` |
| 同号冷却 | 60 秒 | `SmsPhoneCooldownSeconds` |
| 同号日限额 | 10 | `SmsPhoneDailyLimit` |
| 同 IP 日限额 | 40 | `SmsIPDailyLimit` |
| Turnstile | 可选 | 开启后匿名发码需 token |

登录/忘记密码发码**防枚举**：无论号码是否存在均 `success: true`。生产建议 Turnstile + Redis。

### 相关接口摘要

| 方法 | 路径 | 用途 |
|------|------|------|
| GET | `/api/verification?email=` | 邮箱注册发码 |
| GET | `/api/verification/sms?phone=` | 短信注册发码（Turnstile 可选） |
| GET | `/api/verification/sms_login?phone=` | 短信登录发码 |
| GET | `/api/reset_password/sms?phone=` | 忘记密码短信发码 |
| POST | `/api/user/login/sms` | 短信登录 |
| POST | `/api/user/reset/sms` | 短信重置密码，返回新密码 |
| POST | `/api/user/register` | 主账号注册，可带 `phone` + `verification_code` |

邮件：`GET /api/reset_password?email=`、`POST /api/user/reset`。

---

## 计费与限额

- 每个团队用户 ↔ 组织下的一个 Token。
- `quota_limit = 0` → Token `unlimited_quota`，费用从**主账号钱包**扣。
- `quota_limit > 0` → 对该 Token 硬限制，登录/取 key 时校验。

| 接口 | 本用户用量 | 组织钱包 |
|------|------------|----------|
| `/api/newuser/usage`、`/self` | `used_quota` 等 | `owner_quota` / `owner_used_quota` |
| `/api/user/self` | — | `quota` / `used_quota` |

---

## 数据库

表 `newusers`（AutoMigrate）：

| 字段 | 说明 |
|------|------|
| `owner_user_id` | 组织主账号 ID |
| `username` | 全站唯一 |
| `password` | bcrypt |
| `display_name` | 显示名 |
| `email` / `phone` | 选填联系方式 |
| `status` | 1 启用 / 2 禁用 |
| `token_id` | 关联 Token |
| `quota_limit` | 单用户限额（0 = 不限） |
| `is_org_owner` | 是否组织主在团队表中的标记 |

组织配置（`options`）：

- `newuser.owner.{id}.enabled`
- `newuser.owner.{id}.register_enabled`
- `newuser.owner.{id}.register_code`（**全局唯一**）

---

## 升级维护

| 文件 | 作用 |
|------|------|
| `model/newuser.go` | 模型与业务 |
| `service/newuser_auth.go` | JWT |
| `middleware/newuser_auth.go` | 鉴权 |
| `controller/newuser.go` | API |
| `router/newuser-router.go` | 路由 |
| `web/default/src/features/newusers/` | 管理 UI |
| `web/default/src/features/newuser-register/` | Web 注册页 |

注册点：`model/main.go`、`router/api-router.go`、`common/init.go`、`.env.example`。

1. 合并上游时保留上述文件与 AutoMigrate / `SetNewuserRouter`。
2. `NEWUSER_ENABLED=false` 时 `/api/newuser/*` 返回 503，不影响主账号。
3. 仅新增 `newusers` 表，不覆盖原有 `users` / `tokens`。

---

## 常见问题

**模块 503？** → `NEWUSER_ENABLED=true` 并重建重启。

**登录还要 owner_user_id？** → 旧行为；现只需用户名密码。仍报错请重建镜像。

**remain_quota 为 0 没余额？** → 若 `unlimited: true`，看 `owner_quota`。

**登录成功但 AI 401？** → `/v1/*` 用 `api_key`（sk-），不要用 JWT。

**团队账号能在 Web `/sign-in` 登录吗？** → 不能。Web 登录是组织管理员；团队用户用桌面端 `/api/newuser/login`。

**团队用户怎么找回密码？** → 用 `/api/newuser/reset_password`（邮箱）或 `/api/newuser/reset_password/sms`（短信），再 `POST /api/newuser/reset` 或 `/reset/sms`。账号须已绑定邮箱/手机。不要用 `/api/user/reset*`。

**邀请码冲突？** → 每个组织的 `register_code` 必须全局唯一。

**Turnstile 必须传吗？** → 后台关闭时可省略；开启时必须有效 token。

**JWT/Cookie 重启失效？** → 固定 `SESSION_SECRET`（或 `NEWUSER_JWT_SECRET`）。

**开发 CORS？** → 后端 `/api/*` 已开 CORS；或 Vite 代理 `/api`。改代码后需重建容器。

---

## 集成示例

### 主账号（管理端）

```javascript
await fetch('/api/user/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({ username: 'myorg', password: '12345678' }),
});

const self = await fetch('/api/user/self', { credentials: 'include' });
const { data: wallet } = await self.json();
console.log('组织剩余配额:', wallet.quota);
```

### 团队用户（桌面端）

```javascript
const loginRes = await fetch('/api/newuser/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'soubao123', password: '12345678' }),
});
const { data } = await loginRes.json();
const { token, api_key } = data;

await fetch('/api/newuser/usage', {
  headers: { Authorization: `Bearer ${token}` },
});

await fetch('/v1/chat/completions', {
  method: 'POST',
  headers: {
    Authorization: `Bearer ${api_key}`,
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    model: 'gpt-4o-mini',
    messages: [{ role: 'user', content: '你好' }],
  }),
});
```

---

*文档与 newuser 增量模块同步。部署示例：agent.tpsns.com / 万职欣 AI 数字员工。*
