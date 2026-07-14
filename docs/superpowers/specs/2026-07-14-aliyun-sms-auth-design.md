# 阿里云短信认证设计（注册 / 绑定 / 登录）

**日期：** 2026-07-14  
**状态：** 已确认（待实现计划）  
**范围：** 后端 + Default 主题 + Classic 主题

## 背景

现有密码注册仅支持邮箱验证码（`EmailVerificationEnabled` + SMTP）。业务需要增加阿里云短信能力，并在注册时支持与邮箱**二选一**验证；同时支持个人中心绑定手机、登录页「验证码登录」Tab。

## 目标

1. 系统设置可开关短信注册验证、短信登录，并可配置阿里云短信参数（对标 SMTP）。
2. 注册：邮箱验证与短信验证可同时开启；开启两者时用户二选一完成验证。
3. 个人中心：绑定 / 换绑手机（对标邮箱绑定）。
4. 登录页：密码登录与「手机号 + 验证码」Tab 并列。
5. Default（`web/default`）与 Classic（`web/classic`）前端均落地。

## 非目标

- 国际手机号 / 多国家区号（本期仅中国大陆 11 位）。
- 短信找回密码。
- newuser / 第三方用户表的短信体系。
- 腾讯云或其他短信通道（本期仅阿里云 Dysmsapi）。
- 为 2FA 单独再加一层短信（现有 2FA 逻辑不变；短信登录成功后仍走既有 `setupLogin`，若账号启用 2FA 则按现有密码登录的 2FA 流程处理）。

## 决策摘要

| 项 | 决定 |
|----|------|
| 短信通道 | 阿里云短信（Dysmsapi）直连，无 Provider 抽象 |
| 开关 | `SmsVerificationEnabled`、`SmsLoginEnabled` 独立 |
| 配置存放 | OptionMap / 系统设置 UI（对标 SMTP） |
| 注册验证 | 仅邮 / 仅短 / 都开二选一 / 都关不强制 |
| 短信登录 UI | 与密码登录并列 Tab |
| 前端 | Default + Classic |
| 模板 | 本期单个 `AliyunSmsTemplateCode`（变量含验证码）；不拆注册/登录/绑定三套模板 |

## 架构

```
注册 / 登录 / 绑定 UI
    │
    ▼
controller（发码、注册、登录、绑定）
    │
    ├── common.verification（key = phone + purpose，复用现有验证码存储）
    ├── common/aliyun_sms.go（调用阿里云发送）
    └── model.User.Phone（规范化唯一存储）
```

## 数据模型

在 `users` 表增加：

- `phone`：`string`，`gorm:"index"`，存规范化后的大陆手机号（`1` + 10 位数字，无 `+86` 前缀）。
- 空字符串表示未绑定。
- 业务层保证「非空 phone 全局唯一」（对标邮箱：`EnsurePhoneAvailable` / `IsPhoneAlreadyTaken`）。
- SQLite / MySQL / PostgreSQL 通过 GORM `AutoMigrate` 增加列；不做数据库特有 JSON 类型。

`User` 请求体复用 `VerificationCode`（`gorm:"-:all"`），注册/绑定短信时传该字段。

## 系统配置

### Auth 开关（basic-auth 区）

| Key | 类型 | 含义 |
|-----|------|------|
| `SmsVerificationEnabled` | bool | 注册允许/要求短信验证（与邮箱规则见下） |
| `SmsLoginEnabled` | bool | 登录页启用验证码登录 Tab；并允许发送登录短信 |

### 阿里云参数（Operations 新增 SMS 段，对标 SMTP）

| Key | 说明 |
|-----|------|
| `AliyunSmsAccessKeyId` | AccessKey ID |
| `AliyunSmsAccessKeySecret` | AccessKey Secret（UI 不回显明文，空表示不更新） |
| `AliyunSmsSignName` | 短信签名 |
| `AliyunSmsTemplateCode` | 模板 CODE；模板内须含验证码变量（默认变量名 `code`） |
| `AliyunSmsTemplateParamCodeKey` | 可选，默认 `code`；对应模板 JSON 键名 |
| `AliyunSmsEndpoint` | 可选，默认阿里云国内 Dysmsapi 默认 endpoint |

未配齐必填项时：发短信接口返回明确错误（「短信服务未配置」），不伪装成功。

### `/api/status` 对外字段

- `sms_verification`: `SmsVerificationEnabled`
- `sms_login`: `SmsLoginEnabled`

前端仅用这两项控制 UI；不把 AccessKey 暴露给匿名接口。

## 注册验证规则（明确判定）

记 `E = EmailVerificationEnabled`，`S = SmsVerificationEnabled`。

| E | S | 行为 |
|---|---|------|
| 0 | 0 | 不要求邮箱/手机验证码（与现网一致） |
| 1 | 0 | 必须 `email` + `verification_code`；验邮箱 purpose |
| 0 | 1 | 必须 `phone` + `verification_code`；验短信注册 purpose |
| 1 | 1 | **二选一**：若请求带了非空 `phone`，则走短信路径（可不上送邮箱）；否则必须走邮箱路径。禁止「两边都不完整」：短信路径缺 phone/code，或邮箱路径缺 email/code，均失败。若同时带了 phone 与 email，**优先短信路径**，邮箱仅作普通资料写入与否：本期注册成功时，短信路径写入 `phone`，`email` 仅在同时通过邮箱校验时写入；为简化：**二选一模式下同时提交两者时只校验并写入短信侧（phone），忽略 email 校验与写入**。 |

存在性检查：走邮箱路径时检查 email 占用；走短信路径时检查 phone 占用；username 始终检查。

## 验证码 purpose

在 `common/verification.go` 增加（或与现有并列的字符串常量）：

- 注册短信：如 `SmsRegisterPurpose = "sr"`
- 登录短信：如 `SmsLoginPurpose = "sl"`
- 绑定短信：如 `SmsBindPurpose = "sb"`

生成长度、有效期复用现有 `GenerateVerificationCode` / `VerificationValidMinutes`。  
发码限流：对标邮箱，按 IP + 可选 phone 维度（新建 `SmsVerificationRateLimit`，参数可与邮件同量级：30s 内最多 2 次）。

## API

### 发送注册短信

- `GET /api/verification/sms?phone=`（路由风格对齐现有 `SendEmailVerification` 的 query 方式；若项目已有统一 POST，则与邮件发码同一风格）
- 前置：`SmsVerificationEnabled`；手机号格式合法；未被占用；阿里云已配置。
- 成功：写入验证码并真正调用阿里云发送。
- 失败：返回具体原因（格式、占用、未开启、发送失败）。

### 发送登录短信

- `GET /api/verification/sms_login?phone=`（名称可微调但须单独 purpose）
- 前置：`SmsLoginEnabled`；手机号已绑定到**恰好一个**启用状态用户；否则统一返回成功或模糊失败以避免枚举（推荐：**一律返回 success，仅在命中唯一启用用户时真正发信**，对标密码重置邮件的防枚举策略）。

### 发送绑定短信

- 需登录 Session；`GET /api/verification/sms_bind?phone=`（或挂在 `/api/user/...`）
- 前置：短信服务已配置；目标 phone 未被他人占用。
- 绑定提交接口对标 `BindEmail`。

### 注册

- 扩展现有 `POST /api/user/register`：可接收 `phone`。
- 按「注册验证规则」校验。
- 成功写入 `phone`（若走短信路径）。

### 短信登录

- `POST /api/user/login/sms` body：`{ "phone", "verification_code" }`
- 前置：`SmsLoginEnabled` + `PasswordLoginEnabled` 的关系：**短信登录只依赖 `SmsLoginEnabled`**，不强制密码登录开关（密码 Tab 仍由 `PasswordLoginEnabled` 控制）。
- 校验验证码 → 按 phone 找唯一启用用户 → 若启用了站点级/用户 2FA，走与密码登录相同的 2FA 后续（返回需 2FA 的既有响应形状）；否则 `setupLogin`。
- 错误文案统一「手机号或验证码错误」，避免枚举。

## 阿里云发送实现

新增 `common/aliyun_sms.go`（或 `service/aliyun_sms.go`）：

- 使用阿里云官方 Go SDK（Dysmsapi）或稳定 HTTP OpenAPI 签名调用；优先官方 SDK 以降低签名错误。
- 请求参数：`PhoneNumbers`、`SignName`、`TemplateCode`、`TemplateParam`（JSON，`{ "<codeKey>": "<6位码>" }`）。
- 将上游错误映射为可读 message；日志记录 RequestId，不落完整手机号明文可考虑脱敏（中间四位 `*`）。

## 前端

### Default

1. **system-settings / auth / basic-auth**：增加两个短信开关。  
2. **system-settings / operations**：新增「阿里云短信」配置段（对标 SMTP Email）。  
3. **sign-up**：当 `email_verification` 与/或 `sms_verification` 开启时，提供 Email / Phone 切换（或 Tabs）；二选一提交。  
4. **sign-in**：`sms_login` 为 true 时增加「验证码登录」Tab：手机号 + 验证码 + 发送按钮。  
5. **profile**：绑定手机对话框（对标 email-bind）。  
6. i18n：所有文案走 `t()`，补齐 locales。

### Classic

同等能力：

- `SystemSetting.jsx` 开关与短信配置  
- `RegisterForm` / `LoginForm`  
- `PersonalSetting` + 绑定弹窗  

## 安全与计费注意

- 发码接口必须限流（IP + phone）。
- 登录发码防用户枚举。
- AccessKeySecret 不出现在 `/api/status`；管理端读取时可掩码。
- 手机号唯一，防止一号多户。

## 测试要点

- 注册规则四象限（E/S 组合）表驱动。
- 手机规范化与唯一性。
- 短信登录：未绑定 / 验证码错 / 成功；`SmsLoginEnabled=false` 拒绝。
- 发送：配置缺失时失败；限流触发。
- 不引入随机压力测或纯覆盖率测试。

## 实现顺序

1. 后端模型、Option、阿里云发送、验证码 API、注册/登录/绑定。  
2. Default 前端。  
3. Classic 前端。  
4. 必要时补简短运维说明（非必须）。

## 与并行需求的关系

同会话中另有「`/api/newuser/login` 主账号懒迁移」设计；**本 spec 不包含该项**，单独排期。
