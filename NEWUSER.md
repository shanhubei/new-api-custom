# 第三方 UI 增量用户模块（newuser）

> **仓库说明**  
> 本文档属于二次开发仓库 [shanhubei/new-api-custom](https://github.com/shanhubei/new-api-custom)。  
> 基于上游 [QuantumNous/new-api](https://github.com/QuantumNous/new-api)（AGPL-3.0）增量扩展。

本文档说明 **newuser** 模块的设计、部署与 API 用法，并包含 **组织主账号**（原有 new-api 用户）的注册与登录说明。

该模块为**增量扩展**：不修改原有 `/api/user/login`、relay 计费与 Token 鉴权逻辑，便于后续合并上游 new-api 升级。

---

## 目录

- [两套账号体系](#两套账号体系)
- [设计目标](#设计目标)
- [架构示意](#架构示意)
- [环境变量与部署](#环境变量与部署)
- [主账号（组织）注册与登录](#主账号组织注册与登录)
- [第三方用户模块](#第三方用户模块)
- [管理后台 UI](#管理后台-ui)
- [上线步骤](#上线步骤)
- [API 参考](#api-参考)
- [计费与限额](#计费与限额)
- [数据库](#数据库)
- [升级维护](#升级维护)
- [常见问题](#常见问题)
- [集成示例](#集成示例)

---

## 两套账号体系

| 类型 | 用户表 | 登录接口 | 鉴权方式 | 典型用途 |
|------|--------|----------|----------|----------|
| **组织主账号** | `users` | `POST /api/user/login` | Cookie Session | 管理后台、充值、创建第三方用户 |
| **第三方 UI 用户** | `newusers` | `POST /api/newuser/login` | JWT（Bearer） | 独立前端、终端用户 |

关系：**一个主账号 = 一个组织**，其下可创建多个第三方用户；第三方用户的 AI 费用从该主账号钱包扣除。

```
主账号注册/登录 (/api/user/*)
    ↓
管理后台 / API 创建第三方用户 (/api/newuser/admin/*)
    ↓
第三方 UI 登录 (/api/newuser/login) → 拿 sk- 调 /v1/*
```

---

## 设计目标

| 目标 | 说明 |
|------|------|
| 不破坏原有登录 | 管理后台、Classic/Default 前端仍走 `/api/user/login`（Cookie Session） |
| 增量扩展 | 第三方 UI 走 `/api/newuser/*`，独立 JWT，与原有 Session 隔离 |
| 组织共用钱包 | 原有 new-api **User** 视为「组织」；第三方用户每人映射该组织下的一个 **sk- Token** |
| 计费不变 | AI 调用仍直连 `/v1/*`，费用从组织主账号钱包扣除 |
| 用量与限额 | 单用户统计、限额在 newuser 模块内管理，不改动 relay 核心 |

---

## 架构示意

```
┌──────────────────┐   POST /api/user/login      ┌──────────────────┐
│  组织主账号       │ ─────────────────────────► │   new-api        │
│  (管理后台)       │ ◄── Cookie Session ──────── │   users 表       │
└────────┬─────────┘                             └────────┬─────────┘
         │ 创建第三方用户                                      │ 钱包 Quota
         ▼                                                  │
┌──────────────────┐   POST /api/newuser/login     ┌────────▼─────────┐
│   第三方 UI       │ ───────────────────────────► │  newuser 模块    │
│  (独立前端)       │ ◄── JWT + sk- api_key ────── │  newusers 表     │
└────────┬─────────┘                               └──────────────────┘
         │ Authorization: Bearer sk-xxx
         ▼
┌──────────────────┐
│  /v1/chat/...    │  ← relay（原有，未改动）
└──────────────────┘
```

**数据关系：**

```
组织主账号 (User.id = owner_user_id)
  ├── 钱包 Quota（统一扣费）
  └── Token 列表
        └── newuser:alice   ← 第三方用户 alice 专用
        └── newuser:bob     ← 第三方用户 bob 专用

newusers 表
  ├── owner_user_id → 指向组织主账号
  ├── username / password（第三方独立登录，用户名全站唯一）
  └── token_id → 指向上述 Token
```

---

## 环境变量与部署

在 `docker-compose.yml` 或 `.env` 中配置：

```env
# 启用第三方 UI 模块（默认 false）
NEWUSER_ENABLED=true

# 【可选】仅用于第三方自助注册时的默认组织 ID
# NEWUSER_DEFAULT_OWNER_ID=2

# JWT 密钥（可选，默认使用 SESSION_SECRET）
# NEWUSER_JWT_SECRET=your-random-secret

# JWT 有效期（小时，默认 168 = 7 天）
# NEWUSER_JWT_EXPIRE_HOURS=168

# 允许 register_type=org 同时注册主用户表 + 第三方表（Electron 组织自助注册）
# NEWUSER_ORG_SELF_REGISTER_ENABLED=true

# 建议固定，避免重启后 Session/JWT 失效
# SESSION_SECRET=your-fixed-random-string
```

### docker-compose 示例

```yaml
services:
  new-api:
    environment:
      - NEWUSER_ENABLED=true
      # Electron 组织自助注册（register_type=org）需开启：
      # - NEWUSER_ORG_SELF_REGISTER_ENABLED=true
      # 多租户场景不必设置 NEWUSER_DEFAULT_OWNER_ID
```

修改后重建并启动：

```bash
docker compose build
docker compose up -d
```

### `NEWUSER_DEFAULT_OWNER_ID` 说明

- **第三方登录不需要**，用户名全站唯一，只传 `username` + `password` 即可。
- **仅用于** `POST /api/newuser/register` 自助注册：请求未带 `owner_user_id` 时使用。

---

## 主账号（组织）注册与登录

组织主账号即 new-api 原生 **User**，用于登录管理后台、充值、管理第三方用户。接口为系统原有 API，**未被 newuser 模块修改**。

### POST `/api/user/register` — 注册主账号

需在系统设置中开启注册（`RegisterEnabled` / 密码注册开关）。若开启邮箱验证，还需传验证码。

**请求体：**

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

| 字段 | 说明 |
|------|------|
| `username` | 登录名，最长 20 字符 |
| `password` | 密码，8–20 字符 |
| `password2` | 确认密码（前端校验用，后端 Register 主要读 username/password） |
| `email` | 邮箱（开启邮箱验证时必填） |
| `verification_code` | 邮箱验证码（开启验证时必填） |
| `aff_code` | 邀请码（可选） |

**成功响应：**

```json
{
  "success": true,
  "message": ""
}
```

注册成功后，该用户的 `id` 即为后续第三方用户关联的 **`owner_user_id`**。

> 首次部署若未初始化，系统可能自动创建 root 用户（见安装日志）。生产环境请尽快修改默认密码。

---

### POST `/api/user/login` — 登录主账号

**请求体：**

```json
{
  "username": "myorg",
  "password": "12345678"
}
```

**成功响应（无 2FA）：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 2,
    "username": "myorg",
    "display_name": "myorg",
    "role": 1,
    "status": 1,
    "group": "default"
  }
}
```

**鉴权方式：** 响应会通过 `Set-Cookie` 写入 **Session Cookie**（非 JWT）。后续请求浏览器自动携带 Cookie；Apifox/脚本需手动保存 Cookie。

**若启用 2FA：**

```json
{
  "success": true,
  "message": "需要两步验证",
  "data": { "require_2fa": true }
}
```

需再调用 `POST /api/user/login/2fa` 完成验证。

| role 值 | 含义 |
|---------|------|
| `1` | 普通用户（可作组织主账号） |
| `10` | 管理员 |
| `100` | 超级管理员 Root |

---

### GET `/api/user/logout` — 退出登录

清除 Session Cookie，无需请求体。

---

### GET `/api/user/self` — 获取当前主账号信息

需主账号 Session（Cookie）。

**响应 `data` 主要字段：**

| 字段 | 说明 |
|------|------|
| `id` | 用户 ID（即 `owner_user_id`） |
| `username` / `display_name` | 登录名 / 显示名 |
| `quota` | 钱包**剩余**配额 |
| `used_quota` | 累计已用配额 |
| `role` / `status` / `group` | 角色、状态、分组 |
| `permissions` | 权限信息 |

第三方 UI 中的 `owner_quota` / `owner_used_quota` 即来自此主账号的 `quota` / `used_quota`。

---

### 主账号 vs 第三方用户 — 登录对比

| 项目 | 主账号 `/api/user/login` | 第三方 `/api/newuser/login` |
|------|--------------------------|------------------------------|
| 用户表 | `users` | `newusers` |
| 鉴权 | Cookie Session | JWT Bearer |
| 响应 token | 无（Cookie） | 有 JWT + sk- api_key |
| 管理第三方用户 | 可以（需 Session） | 不可以 |
| 调 `/v1/*` | 需自行创建 API Key | 登录直接返回 sk- |

---

### Electron / 桌面端怎么处理？

这是**设计差异**，不是 bug：

| 场景 | 推荐登录方式 | 桌面端怎么带鉴权 |
|------|--------------|------------------|
| **终端用户**（只用 AI、查自己的用量） | `POST /api/newuser/login` | 保存返回的 `token`（JWT）+ `api_key`（sk-），本地存储即可 |
| **组织管理员**（管第三方用户、看组织余额） | `POST /api/user/login` | 需 Cookie **或** Access Token（见下方） |

#### 方案 A：终端用户 — 直接用第三方登录（推荐）

桌面端**不要**走主账号登录，走 newuser 接口，和 Web 第三方 UI 一样：

```javascript
// Electron 渲染进程 / preload
const res = await fetch('https://your-api.com/api/newuser/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'soubao123', password: '12345678' }),
});
const { data } = await res.json();
// 持久化到 electron-store / safeStorage
// data.token     → 调 /api/newuser/*
// data.api_key   → 调 /v1/*
// data.owner_quota → 组织余量
```

#### 方案 B：组织管理员 — Access Token（适合 Electron 调管理 API）

主账号 `/api/user/login` **不会**在 JSON 里返回 token，但 new-api 支持用 **Access Token** 代替 Cookie 调管理接口（如 `/api/newuser/admin/*`、`/api/user/self`）。

**第一步：在浏览器登录一次，生成 Token**

1. 浏览器打开管理后台，登录主账号
2. 调用（浏览器 Cookie 自动带上）：

```http
GET /api/user/token
Cookie: session=...
```

响应：

```json
{
  "success": true,
  "data": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```

3. 同时记下登录返回的 `data.id`（用户 ID）

**第二步：Electron 里持久化，后续请求带头：**

```javascript
const ACCESS_TOKEN = '...'; // 从 /api/user/token 拿到
const USER_ID = 2;          // 主账号 id

await fetch('https://your-api.com/api/user/self', {
  headers: {
    Authorization: `Bearer ${ACCESS_TOKEN}`,
    'New-Api-User': String(USER_ID),
  },
});

await fetch('https://your-api.com/api/newuser/admin/users', {
  headers: {
    Authorization: `Bearer ${ACCESS_TOKEN}`,
    'New-Api-User': String(USER_ID),
  },
});
```

> **注意：** 两个头必须同时带：`Authorization` + `New-Api-User`（且 User ID 必须与 Token 所属用户一致）。

#### 方案 C：组织管理员 — Cookie Jar（Electron 自己登录）

若希望桌面端也能 `POST /api/user/login`，需让 HTTP 客户端**自动保存并回传 Cookie**：

```javascript
// 使用 axios + tough-cookie，或 electron net/session 模块
// 关键：登录和后续请求必须共用同一个 cookie jar

await axios.post('/api/user/login', { username, password }, {
  withCredentials: true,
  jar: cookieJar, // 同一 jar
});

await axios.get('/api/user/self', {
  withCredentials: true,
  jar: cookieJar,
});
```

纯 `fetch` 默认**不会**跨请求保存 Cookie，Electron 里容易登录成功但后续 401。

#### 方案 D：内置 WebView 登录

Electron 打开内嵌 BrowserWindow 加载管理后台登录页，登录成功后从 `session.cookies` 读取 Cookie，主进程后续请求复用该 Cookie。适合「管理员偶尔操作」的场景。

#### 该怎么选？

```
桌面端给谁用？
├── 普通用户（聊天、调 AI）     → 方案 A：/api/newuser/login
└── 组织管理员（管账号、充值）   → 方案 B（Access Token）或 方案 C（Cookie Jar）
```

**AI 调用**无论哪种身份，最终都用 **sk- api_key** 调 `/v1/*`，不要用主账号 Session 调 AI。

---

## 第三方用户模块

### 多租户说明

- 每个 new-api 主账号 = 一个独立组织。
- 主账号 A 登录后，只能管理 A 名下的第三方用户（`/api/newuser/admin/*`）。
- **第三方用户名全站唯一**（不同组织也不能重名）。
- 登录时**不需要**传 `owner_user_id`，系统按用户名自动定位组织。

### 统一注册 `POST /api/newuser/register`（带 `register_type`）

适合 **Electron 桌面端**：一次注册、返回 JWT，无需 Cookie。

| register_type | owner_user_id | 行为 |
|---------------|-----------------|------|
| `"org"` | 不传 | 同时在 **`users` 主表** 和 **`newusers` 第三方表** 各建一条；标记 `is_org_owner=true`，可管理本组织第三方用户 |
| `"member"` | 必传（或 `NEWUSER_DEFAULT_OWNER_ID`） | 仅在 **`newusers` 表** 注册，挂在指定组织下；需组织开启自助注册 + 邀请码 |

**推断规则（未传 `register_type` 时）：**

- 有 `owner_user_id` → 按 `member` 处理
- 无 `owner_user_id` → 按 `org` 处理

#### 注册组织主账号（Electron 推荐）

需开启 `NEWUSER_ORG_SELF_REGISTER_ENABLED=true`，且系统允许主用户注册。

```json
POST /api/newuser/register

{
  "register_type": "org",
  "username": "myorg",
  "password": "12345678",
  "display_name": "我的组织"
}
```

成功后同时创建：

1. `users` 表主账号（钱包、充值主体）
2. `newusers` 表记录（`is_org_owner: true`，`can_manage_users: true`）
3. 对应 sk- Token

#### 注册组织成员

```json
{
  "register_type": "member",
  "owner_user_id": 2,
  "register_code": "invite-code",
  "username": "alice",
  "password": "12345678"
}
```

#### 登录返回新增字段

| 字段 | 说明 |
|------|------|
| `is_org_owner` | 是否组织主账号（第三方表标记） |
| `can_manage_users` | 是否可新建/管理本组织第三方用户 |
| `register_type` | `"org"` 或 `"member"` |

#### 组织主账号用 JWT 管理第三方用户（Electron）

`is_org_owner=true` 的用户登录后，用 JWT 调 **`/api/newuser/manage/*`**（与 `/api/newuser/admin/*` 能力相同，但用 Bearer JWT，无需 Cookie）：

```http
GET  /api/newuser/manage/users
POST /api/newuser/manage/users
PUT  /api/newuser/manage/settings
...
Authorization: Bearer <login 返回的 token>
```

---

## 管理后台 UI

登录主账号后，侧边栏 **个人 → 第三方用户**（路径 `/newusers`，Default 主题）：

| 功能 | 说明 |
|------|------|
| 组织设置 | 启用/关闭本组织第三方用户、自助注册、邀请码 |
| 统计卡片 | 用户总数、活跃数、总已用配额 |
| 用户列表 | 用户名、状态、已用/限额、上次登录 |
| 新建/编辑 | 创建用户、修改密码/限额/状态 |
| 查看用量 | 本用户用量 + **组织钱包余量** + 最近日志 |
| 禁用 | 禁用用户并停用对应 Token |

---

## 上线步骤

### 1. 启用模块并重启

`NEWUSER_ENABLED=true`，`docker compose build && docker compose up -d`。

### 2. 注册/登录组织主账号

- 新用户：`POST /api/user/register` → `POST /api/user/login`
- 或使用已有账号登录管理后台

### 3. 开启本组织 newuser 功能

用主账号 Session 调用：

```http
PUT /api/newuser/admin/settings
Content-Type: application/json
Cookie: session=...

{
  "enabled": true,
  "register_enabled": false,
  "register_code": ""
}
```

### 4. 创建第三方用户

**方式 A：管理后台 UI** — 个人 → 第三方用户 → 新建

**方式 B：API**

```http
POST /api/newuser/admin/users
Content-Type: application/json
Cookie: session=...

{
  "username": "soubao123",
  "password": "12345678",
  "display_name": "搜宝用户",
  "quota_limit": 0
}
```

**方式 C：自助注册** — 开启 `register_enabled` 后，第三方 UI 调 `POST /api/newuser/register`

### 5. 第三方 UI 集成

1. `POST /api/newuser/login` → 获取 JWT + sk-
2. 管理接口带 `Authorization: Bearer <JWT>`
3. AI 调用带 `Authorization: Bearer sk-xxx` 调 `/v1/*`

---

## API 参考

### 第三方公共接口（无需登录）

#### POST `/api/newuser/login`

**请求体：**

```json
{
  "username": "soubao123",
  "password": "12345678"
}
```

`owner_user_id` 可选；传入时会额外校验用户是否属于该组织。

**成功响应：**

```json
{
  "success": true,
  "message": "",
  "data": {
    "id": 1,
    "username": "soubao123",
    "display_name": "搜宝用户",
    "owner_user_id": 2,
    "owner_username": "myorg",
    "owner_display_name": "万职欣 AI",
    "owner_quota": 9850000,
    "owner_used_quota": 150000,
    "is_org_owner": false,
    "can_manage_users": false,
    "register_type": "member",
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "api_key": "sk-xxxxxxxx"
  }
}
```

| 字段 | 说明 |
|------|------|
| `token` | newuser JWT，用于 `/api/newuser/*` |
| `api_key` | sk- Token，用于 `/v1/*` |
| `owner_user_id` | 组织主账号 ID |
| `owner_username` | 组织主账号登录名 |
| `owner_display_name` | 组织主账号显示名 |
| `owner_quota` | 组织主账号钱包剩余配额 |
| `owner_used_quota` | 组织主账号累计已用配额 |
| `is_org_owner` | 是否组织主账号（第三方表标记） |
| `can_manage_users` | 是否可管理本组织第三方用户 |
| `register_type` | `"org"` 或 `"member"` |

---

#### POST `/api/newuser/register`

统一注册接口，通过 `register_type` 区分组织主账号与成员。

**`register_type: "org"`** — 同时注册主用户表 + 第三方表（需 `NEWUSER_ORG_SELF_REGISTER_ENABLED=true`）：

```json
{
  "register_type": "org",
  "username": "myorg",
  "password": "12345678",
  "display_name": "我的组织"
}
```

**`register_type: "member"`** — 仅在第三方表注册（需组织开启 `register_enabled` + 邀请码）：

```json
{
  "register_type": "member",
  "owner_user_id": 2,
  "register_code": "your-invite-code",
  "username": "bob",
  "password": "12345678",
  "display_name": "Bob"
}
```

未传 `register_type` 时：有 `owner_user_id` → `member`；无 → `org`。

未传 `owner_user_id` 的 `member` 注册可使用 `NEWUSER_DEFAULT_OWNER_ID`。

---

### 第三方用户接口（JWT 鉴权）

请求头：`Authorization: Bearer <login 返回的 token>`

#### GET `/api/newuser/self`

当前用户信息与用量摘要（含 `owner_quota` / `owner_used_quota`）。

#### GET `/api/newuser/token`

重新获取 sk- api_key。

#### GET `/api/newuser/usage`

详细用量与组织钱包信息。

**响应示例：**

```json
{
  "success": true,
  "data": {
    "quota_limit": 0,
    "used_quota": 1200,
    "remain_quota": 0,
    "unlimited": true,
    "owner_user_id": 2,
    "owner_username": "myorg",
    "owner_display_name": "万职欣 AI",
    "owner_quota": 9850000,
    "owner_used_quota": 150000,
    "recent_logs": []
  }
}
```

**字段说明：**

| 字段 | 含义 |
|------|------|
| `used_quota` / `remain_quota` | **本第三方用户** Token 用量 |
| `unlimited` | Token 是否不限额（`true` 时费用走组织钱包） |
| `quota_limit` | newuser 模块设置的单用户上限（0=不限） |
| `owner_quota` | **组织主账号钱包剩余**（实际能否继续调用看此值） |
| `owner_used_quota` | 组织主账号历史总消耗 |
| `recent_logs` | 本 Token 最近消费日志 |

> 当 `unlimited: true` 且 `remain_quota: 0` 时，表示 Token 本身不设上限，**不代表组织余额为 0**，请查看 `owner_quota`。

---

### 组织主账号 JWT 管理接口（Electron 友好）

`is_org_owner=true` 的第三方用户登录后，用 **JWT** 调用（无需 Cookie Session）。路径与 admin 相同，前缀为 `/api/newuser/manage/*`：

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/newuser/manage/users` | 列出本组织第三方用户 |
| POST | `/api/newuser/manage/users` | 创建第三方用户 |
| GET | `/api/newuser/manage/users/:id/usage` | 单用户用量详情 |
| PUT | `/api/newuser/manage/users/:id` | 更新用户 |
| DELETE | `/api/newuser/manage/users/:id` | 禁用用户 |
| GET/PUT | `/api/newuser/manage/settings` | 组织 newuser 配置 |

请求头：`Authorization: Bearer <login 返回的 token>`

---

### 组织管理员接口（主账号 Session）

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/newuser/admin/users` | 列出本组织第三方用户（含用量统计） |
| POST | `/api/newuser/admin/users` | 创建用户并分配 Token |
| GET | `/api/newuser/admin/users/:id/usage` | 单用户用量详情 |
| PUT | `/api/newuser/admin/users/:id` | 更新用户 |
| DELETE | `/api/newuser/admin/users/:id` | 禁用用户 |
| GET/PUT | `/api/newuser/admin/settings` | 组织 newuser 配置 |

---

## 计费与限额

### 组织统一扣费

- 每个第三方用户对应主账号下的一个 Token。
- `quota_limit = 0` 时 Token 为 `unlimited_quota = true`，费用从**主账号钱包**扣除。

### 单用户限额（可选）

设置 `quota_limit > 0` 时，对该 Token 做硬限制，并在登录/取 key 时校验。

### 用量统计

| 接口 | 本用户用量 | 组织钱包 |
|------|------------|----------|
| `/api/newuser/usage` | `used_quota` 等 | `owner_quota` / `owner_used_quota` |
| `/api/newuser/self` | 同上 | 同上 |
| 主账号 `/api/user/self` | — | `quota` / `used_quota` |

---

## 数据库

表 `newusers`（启动时 AutoMigrate）：

| 字段 | 说明 |
|------|------|
| `owner_user_id` | 组织主账号 User ID |
| `username` | 第三方用户名（**全站唯一**） |
| `password` | bcrypt 哈希 |
| `display_name` | 显示名 |
| `status` | 1 启用 / 2 禁用 |
| `token_id` | 关联 Token ID |
| `quota_limit` | 单用户限额（0 = 不限） |

组织配置键（`options` 表）：

- `newuser.owner.{ownerUserId}.enabled`
- `newuser.owner.{ownerUserId}.register_enabled`
- `newuser.owner.{ownerUserId}.register_code`

---

## 新增代码文件（升级合并参考）

| 文件 | 作用 |
|------|------|
| `model/newuser.go` | 表结构、CRUD、组织信息 |
| `service/newuser_auth.go` | JWT |
| `middleware/newuser_auth.go` | 鉴权 |
| `controller/newuser.go` | API |
| `router/newuser-router.go` | 路由 |
| `web/default/src/features/newusers/` | 管理后台 UI |

注册点：`model/main.go`、`router/api-router.go`、`common/init.go`、`.env.example`。

---

## 升级维护

1. 保留上述 newuser 相关文件。
2. `model/main.go` 保留 `&Newuser{}` AutoMigrate。
3. `router/api-router.go` 保留 `SetNewuserRouter(...)`。
4. `NEWUSER_ENABLED=false` 时 `/api/newuser/*` 返回 503，不影响主账号功能。

**老数据：** 不会覆盖原有 `users`、`tokens` 等表，仅新增 `newusers` 表。

---

## 常见问题

**Q：模块返回 503？**  
A：设置 `NEWUSER_ENABLED=true` 并重建重启。

**Q：第三方登录报 owner_user_id is required？**  
A：旧版本行为。升级后只需 `username` + `password`。仍报错请重建镜像。

**Q：usage 里 remain_quota 为 0 是不是没余额了？**  
A：不一定。`unlimited: true` 时看 **`owner_quota`** 才是组织钱包余量。

**Q：第三方登录成功但 AI 401？**  
A：用 `api_key`（sk-）调 `/v1/*`，不是 JWT。

**Q：主账号和第三方有什么区别？**  
A：主账号管后台和钱包；第三方是给外部 UI 用的子账号，共用主账号钱包。

**Q：Electron / Vite 开发时 CORS 报错？**  
A：`localhost:5173` 访问 `127.0.0.1:3000` 属于跨域。后端 `/api/*` 已启用 CORS；改代码后需 `docker compose build && docker compose up -d`。开发时也可在 Vite 配置 `server.proxy` 把 `/api` 代理到后端，避免浏览器跨域。

**Q：JWT / Cookie 重启失效？**  
A：固定 `SESSION_SECRET`（或 `NEWUSER_JWT_SECRET`）。

---

## 集成示例

### 主账号：注册并登录（Apifox / fetch）

```javascript
// 1. 注册组织主账号（若系统开放注册）
await fetch('/api/user/register', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({
    username: 'myorg',
    password: '12345678',
  }),
});

// 2. 登录主账号（Cookie 自动保存）
const loginRes = await fetch('/api/user/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  credentials: 'include',
  body: JSON.stringify({ username: 'myorg', password: '12345678' }),
});
const { data: org } = await loginRes.json();
console.log('组织 ID:', org.id, '余额相关见 /api/user/self');

// 3. 查看主账号钱包
const self = await fetch('/api/user/self', { credentials: 'include' });
const { data: wallet } = await self.json();
console.log('剩余配额:', wallet.quota);
```

### 第三方 UI：登录、查用量、调 AI

```javascript
// 1. 第三方用户登录（无需 owner_user_id）
const loginRes = await fetch('/api/newuser/login', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ username: 'soubao123', password: '12345678' }),
});
const { data } = await loginRes.json();
const { token, api_key, owner_quota, owner_display_name } = data;

// 2. 查用量（含组织钱包余量）
const usageRes = await fetch('/api/newuser/usage', {
  headers: { Authorization: `Bearer ${token}` },
});
const { data: usage } = await usageRes.json();
console.log('本用户已用:', usage.used_quota);
console.log('组织剩余:', usage.owner_quota, owner_display_name);

// 3. 调 AI
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

*文档版本：与 newuser 增量模块同步。部署示例：agent.tpsns.com / 万职欣 AI数字员工。*
