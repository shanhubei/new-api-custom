# Aliyun SMS Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add Aliyun SMS verification for password registration (email OR phone), phone bind, and SMS code login, with Default + Classic admin/auth UI.

**Architecture:** Store `users.phone` (CN mobile, unique when non-empty). Reuse existing in-memory verification codes with new purposes (`sr`/`sl`/`sb`). Send via Aliyun Dysmsapi using OptionMap settings (mirrors SMTP). Controllers gate on `SmsVerificationEnabled` / `SmsLoginEnabled`. Frontends read `/api/status` flags.

**Tech Stack:** Go 1.25 + Gin + GORM; Aliyun Dysmsapi Go SDK; React 19 (Default) + React 18 Semi (Classic); existing verification + session login (`setupLogin`).

**Spec:** `docs/superpowers/specs/2026-07-14-aliyun-sms-auth-design.md`

---

## File map

| File | Responsibility |
|------|----------------|
| `common/constants.go` | `SmsVerificationEnabled`, `SmsLoginEnabled`, Aliyun SMS config vars |
| `common/verification.go` | SMS purpose constants |
| `common/phone.go` | `NormalizePhone`, `ValidateCNMobile`, mask helper |
| `common/phone_test.go` | Table tests for normalize/validate |
| `common/aliyun_sms.go` | `SendAliyunSms(phone, code)`, config completeness check |
| `model/user.go` | `Phone` field + availability/bind helpers |
| `model/user_phone_test.go` | Availability / uniqueness helpers (DB fixture if pattern exists) |
| `model/option.go` | Init + update handlers for new options |
| `middleware/sms-verification-rate-limit.go` | IP rate limit (copy email limiter) |
| `controller/misc.go` | Status flags + send SMS register/login |
| `controller/user.go` | Register branch, `LoginSms`, `PhoneBind`, send bind SMS |
| `controller/user_register_sms_test.go` | Register E/S quadrant unit tests (extract pure helper if needed) |
| `router/api-router.go` | New routes |
| `i18n/` | New message keys (en/zh) |
| `go.mod` / `go.sum` | Aliyun Dysmsapi deps |
| Default: `basic-auth-section.tsx`, ops SMS section, status types, sign-up/sign-in, profile bind, i18n | |
| Classic: `SystemSetting.jsx`, `RegisterForm`, `LoginForm`, personal bind modal | |

---

### Task 1: Phone helpers + verification purposes

**Files:**
- Create: `common/phone.go`
- Create: `common/phone_test.go`
- Modify: `common/verification.go`
- Modify: `common/constants.go` (add bool + string vars only; wiring in Task 3)

- [ ] **Step 1: Write failing tests for phone normalize**

```go
package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePhone(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"13800138000", "13800138000", false},
		{" 13800138000 ", "13800138000", false},
		{"+8613800138000", "13800138000", false},
		{"8613800138000", "13800138000", false},
		{"1380013800", "", true},
		{"23800138000", "", true},
		{"", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := NormalizePhone(tt.in)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./common -run TestNormalizePhone -count=1`

Expected: FAIL (`NormalizePhone` undefined)

- [ ] **Step 3: Implement phone helpers + purposes**

`common/phone.go`:

```go
package common

import (
	"errors"
	"regexp"
	"strings"
)

var cnMobileRegexp = regexp.MustCompile(`^1[3-9]\d{9}$`)

func NormalizePhone(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "-", "")
	if strings.HasPrefix(s, "+86") {
		s = s[3:]
	} else if strings.HasPrefix(s, "86") && len(s) == 13 {
		s = s[2:]
	}
	if !cnMobileRegexp.MatchString(s) {
		return "", errors.New("invalid phone number")
	}
	return s, nil
}

func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return "***"
	}
	return phone[:3] + "****" + phone[7:]
}
```

In `common/verification.go` add:

```go
	SmsRegisterPurpose = "sr"
	SmsLoginPurpose    = "sl"
	SmsBindPurpose     = "sb"
```

In `common/constants.go` add (defaults):

```go
var SmsVerificationEnabled = false
var SmsLoginEnabled = false

var AliyunSmsAccessKeyId = ""
var AliyunSmsAccessKeySecret = ""
var AliyunSmsSignName = ""
var AliyunSmsTemplateCode = ""
var AliyunSmsTemplateParamCodeKey = "code"
var AliyunSmsEndpoint = ""
```

- [ ] **Step 4: Run tests**

Run: `go test ./common -run TestNormalizePhone -count=1`

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add common/phone.go common/phone_test.go common/verification.go common/constants.go
git commit -m "feat(sms): add CN phone normalize and SMS verification purposes"
```

---

### Task 2: User.Phone model helpers

**Files:**
- Modify: `model/user.go` (struct + helpers near email helpers)
- Create: `model/user_phone_test.go` (unit-test Normalize usage + error sentinels if no DB; if project tests use sqlite in-memory elsewhere, follow that pattern)

- [ ] **Step 1: Add field and helpers**

On `User` struct add after `Email`:

```go
Phone string `json:"phone" gorm:"index" validate:"max=20"`
```

Add (mirror email helpers; use `NormalizePhone` from common):

```go
var (
	ErrPhoneAlreadyTaken = errors.New("phone already taken")
	ErrPhoneNotFound     = errors.New("phone not found")
	ErrPhoneAmbiguous    = errors.New("phone ambiguous")
)

func CountUsersByPhone(phone string) (int64, error) {
	phone, err := common.NormalizePhone(phone)
	if err != nil {
		return 0, err
	}
	var count int64
	err = DB.Model(&User{}).Where("phone = ?", phone).Count(&count).Error
	return count, err
}

func IsPhoneAlreadyTaken(phone string) bool {
	count, err := CountUsersByPhone(phone)
	return err == nil && count > 0
}

func EnsurePhoneAvailable(phone string, excludeUserID int) error {
	phone, err := common.NormalizePhone(phone)
	if err != nil {
		return err
	}
	q := DB.Model(&User{}).Where("phone = ?", phone)
	if excludeUserID > 0 {
		q = q.Where("id <> ?", excludeUserID)
	}
	var count int64
	if err := q.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrPhoneAlreadyTaken
	}
	return nil
}

func GetUniqueUserByPhone(phone string) (*User, error) {
	phone, err := common.NormalizePhone(phone)
	if err != nil {
		return nil, ErrPhoneNotFound
	}
	var users []User
	if err := DB.Where("phone = ?", phone).Find(&users).Error; err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrPhoneNotFound
	}
	if len(users) > 1 {
		return nil, ErrPhoneAmbiguous
	}
	return &users[0], nil
}

func BindPhoneToUser(user *User, phone string) error {
	phone, err := common.NormalizePhone(phone)
	if err != nil {
		return err
	}
	if err := EnsurePhoneAvailable(phone, user.Id); err != nil {
		return err
	}
	user.Phone = phone
	return DB.Model(user).Update("phone", phone).Error
}
```

Ensure `AutoMigrate` already migrates `User` (it does via existing startup) — no manual SQL.

- [ ] **Step 2: Add a small unit test for Normalize failure path on EnsurePhoneAvailable**

```go
func TestEnsurePhoneAvailableRejectsInvalid(t *testing.T) {
	err := EnsurePhoneAvailable("bad", 0)
	require.Error(t, err)
}
```

(If `DB` is nil in unit tests without fixture, skip DB-dependent tests with `t.Skip` or only test invalid phone which returns before DB.)

- [ ] **Step 3: Commit**

```bash
git add model/user.go model/user_phone_test.go
git commit -m "feat(sms): add users.phone field and availability helpers"
```

---

### Task 3: OptionMap wiring + status flags

**Files:**
- Modify: `model/option.go` (`InitOptionMap`, `updateOptionMap` bool/`string` cases)
- Modify: `controller/misc.go` (`GetStatus` data map)

- [ ] **Step 1: InitOptionMap entries**

Near email verification options:

```go
common.OptionMap["SmsVerificationEnabled"] = strconv.FormatBool(common.SmsVerificationEnabled)
common.OptionMap["SmsLoginEnabled"] = strconv.FormatBool(common.SmsLoginEnabled)
common.OptionMap["AliyunSmsAccessKeyId"] = ""
common.OptionMap["AliyunSmsAccessKeySecret"] = ""
common.OptionMap["AliyunSmsSignName"] = ""
common.OptionMap["AliyunSmsTemplateCode"] = ""
common.OptionMap["AliyunSmsTemplateParamCodeKey"] = common.AliyunSmsTemplateParamCodeKey
common.OptionMap["AliyunSmsEndpoint"] = ""
```

In bool switch (`HasSuffix Enabled`), add:

```go
case "SmsVerificationEnabled":
	common.SmsVerificationEnabled = boolValue
case "SmsLoginEnabled":
	common.SmsLoginEnabled = boolValue
```

In string updates (follow how `SMTPAccount` is assigned — typically default branch copies into OptionMap then specific cases set package vars). Add explicit cases matching SMTP:

```go
case "AliyunSmsAccessKeyId":
	common.AliyunSmsAccessKeyId = value
case "AliyunSmsAccessKeySecret":
	common.AliyunSmsAccessKeySecret = value
case "AliyunSmsSignName":
	common.AliyunSmsSignName = value
case "AliyunSmsTemplateCode":
	common.AliyunSmsTemplateCode = value
case "AliyunSmsTemplateParamCodeKey":
	if value == "" {
		value = "code"
	}
	common.AliyunSmsTemplateParamCodeKey = value
case "AliyunSmsEndpoint":
	common.AliyunSmsEndpoint = value
```

Note: `AliyunSmsAccessKeySecret` ends with `Secret` so `GetOptions` already omits it from admin list responses — frontend must treat empty input as “keep unchanged” (same as SMTPToken).

- [ ] **Step 2: Status flags in `controller/misc.go`**

Next to `"email_verification"`:

```go
"sms_verification": common.SmsVerificationEnabled,
"sms_login":        common.SmsLoginEnabled,
```

- [ ] **Step 3: Commit**

```bash
git add model/option.go controller/misc.go
git commit -m "feat(sms): wire SMS options and status flags"
```

---

### Task 4: Aliyun SMS sender

**Files:**
- Create: `common/aliyun_sms.go`
- Create: `common/aliyun_sms_test.go` (config-complete check only; no live network)
- Modify: `go.mod` / `go.sum` via `go get`

- [ ] **Step 1: Add dependency**

Run:

```bash
go get github.com/alibabacloud-go/dysmsapi-20170525/v4
go get github.com/alibabacloud-go/darabonba-openapi/v2
go get github.com/alibabacloud-go/tea-utils/v2
go get github.com/alibabacloud-go/tea
```

(If module major path differs at fetch time, use the latest v3/v4 dysmsapi module that `go get` resolves; keep OpenAPI config pattern.)

- [ ] **Step 2: Implement sender**

```go
package common

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

func AliyunSmsConfigured() bool {
	return strings.TrimSpace(AliyunSmsAccessKeyId) != "" &&
		strings.TrimSpace(AliyunSmsAccessKeySecret) != "" &&
		strings.TrimSpace(AliyunSmsSignName) != "" &&
		strings.TrimSpace(AliyunSmsTemplateCode) != ""
}

func SendAliyunSms(phone, code string) error {
	if !AliyunSmsConfigured() {
		return errors.New("sms service not configured")
	}
	phone, err := NormalizePhone(phone)
	if err != nil {
		return err
	}
	codeKey := AliyunSmsTemplateParamCodeKey
	if codeKey == "" {
		codeKey = "code"
	}
	paramBytes, _ := json.Marshal(map[string]string{codeKey: code})
	// Use Dysmsapi client SendSms with:
	// PhoneNumbers=phone, SignName, TemplateCode, TemplateParam=string(paramBytes)
	// Endpoint from AliyunSmsEndpoint if non-empty
	// On API error: return fmt.Errorf("aliyun sms: %s", message); SysLog request id + MaskPhone(phone)
	_ = paramBytes
	return fmt.Errorf("implement SendAliyunSms with Dysmsapi client")
}
```

Replace the stub return with real SDK call (follow Aliyun Go SDK `SendSms` sample). Do not log full AccessKeySecret.

- [ ] **Step 3: Test configured check**

```go
func TestAliyunSmsConfigured(t *testing.T) {
	oldID, oldSecret, oldSign, oldTpl := AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode
	t.Cleanup(func() {
		AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = oldID, oldSecret, oldSign, oldTpl
	})
	AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = "", "", "", ""
	assert.False(t, AliyunSmsConfigured())
	AliyunSmsAccessKeyId, AliyunSmsAccessKeySecret, AliyunSmsSignName, AliyunSmsTemplateCode = "id", "sec", "sign", "tpl"
	assert.True(t, AliyunSmsConfigured())
}
```

- [ ] **Step 4: Commit**

```bash
git add common/aliyun_sms.go common/aliyun_sms_test.go go.mod go.sum
git commit -m "feat(sms): add Aliyun Dysmsapi send helper"
```

---

### Task 5: Rate limit middleware + send-code controllers

**Files:**
- Create: `middleware/sms-verification-rate-limit.go` (clone `middleware/email-verification-rate-limit.go`, rename mark to `SV`, same 2/30s)
- Modify: `controller/misc.go` — `SendSmsVerification`, `SendSmsLoginVerification`
- Modify: `controller/user.go` — `SendSmsBindVerification`, `PhoneBind`
- Modify: `router/api-router.go`
- Modify: `i18n` message catalogs as needed

- [ ] **Step 1: Routes**

In `router/api-router.go` next to email verification:

```go
apiRouter.GET("/verification/sms", middleware.SmsVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendSmsVerification)
apiRouter.GET("/verification/sms_login", middleware.SmsVerificationRateLimit(), middleware.TurnstileCheck(), controller.SendSmsLoginVerification)
```

Under `selfRoute` (UserAuth):

```go
selfRoute.GET("/sms_bind", middleware.SmsVerificationRateLimit(), controller.SendSmsBindVerification)
selfRoute.POST("/phone/bind", middleware.CriticalRateLimit(), controller.PhoneBind)
```

And:

```go
userRoute.POST("/login/sms", middleware.CriticalRateLimit(), anonymousRequestBodyLimit, middleware.TurnstileCheck(), controller.LoginSms)
```

- [ ] **Step 2: SendSmsVerification (register)**

```go
func SendSmsVerification(c *gin.Context) {
	if !common.SmsVerificationEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sms verification is not enabled"})
		return
	}
	phone, err := common.NormalizePhone(c.Query("phone"))
	if err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	if model.IsPhoneAlreadyTaken(phone) {
		// add i18n MsgUserPhoneAlreadyTaken if missing
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "phone already taken"})
		return
	}
	if !common.AliyunSmsConfigured() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sms service not configured"})
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(phone, code, common.SmsRegisterPurpose)
	if err := common.SendAliyunSms(phone, code); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}
```

- [ ] **Step 3: SendSmsLoginVerification (anti-enumeration)**

Always `success: true` if `SmsLoginEnabled`; only send when `GetUniqueUserByPhone` returns enabled user.

- [ ] **Step 4: Bind send + PhoneBind**

Bind send uses `SmsBindPurpose`, requires session user id, `EnsurePhoneAvailable(phone, id)`.

`PhoneBind` body `{ "phone", "code" }` verify `SmsBindPurpose` then `BindPhoneToUser`.

- [ ] **Step 5: Commit**

```bash
git add middleware/sms-verification-rate-limit.go controller/misc.go controller/user.go router/api-router.go i18n
git commit -m "feat(sms): add send-code and phone bind APIs"
```

---

### Task 6: Register verification branching + LoginSms

**Files:**
- Modify: `controller/user.go` (`Register`, add `LoginSms`, extend `loginMethodFromContext`)
- Create: `controller/register_verification.go` helper for pure branch decision (keeps Register readable)
- Create: `controller/register_verification_test.go`

- [ ] **Step 1: Pure helper + tests**

```go
package controller

type registerContactMode int

const (
	registerContactNone registerContactMode = iota
	registerContactEmail
	registerContactPhone
)

// decideRegisterContact implements E/S table from the spec.
// preferPhone when both email and phone non-empty under E&&S.
func decideRegisterContact(emailEnabled, smsEnabled bool, email, phone string) (registerContactMode, error) {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	switch {
	case !emailEnabled && !smsEnabled:
		return registerContactNone, nil
	case emailEnabled && !smsEnabled:
		if email == "" {
			return 0, errors.New("email required")
		}
		return registerContactEmail, nil
	case !emailEnabled && smsEnabled:
		if phone == "" {
			return 0, errors.New("phone required")
		}
		return registerContactPhone, nil
	default: // both
		if phone != "" {
			return registerContactPhone, nil
		}
		if email != "" {
			return registerContactEmail, nil
		}
		return 0, errors.New("email or phone required")
	}
}
```

Table-test all four E/S combinations + both-provided prefers phone.

- [ ] **Step 2: Wire Register**

Replace the hard-coded `if common.EmailVerificationEnabled { ... }` block with:

1. `mode, err := decideRegisterContact(...)` using `user.Email` and `user.Phone`
2. Phone mode: normalize phone; verify code with `SmsRegisterPurpose`; `EnsurePhoneAvailable`; set `cleanUser.Phone`
3. Email mode: existing email verify path
4. Existence check: username always; email only in email mode; phone only in phone mode

Decode still uses `json` decoder into `model.User` (already has `Phone` + `VerificationCode`). Prefer migrate this decode to `common.DecodeJson` only if touching the same lines; do not drive-by rewrite entire Register.

- [ ] **Step 3: LoginSms**

Mirror `Login` after credential check:

```go
func LoginSms(c *gin.Context) {
	if !common.SmsLoginEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sms login is not enabled"})
		return
	}
	var req struct {
		Phone            string `json:"phone"`
		VerificationCode string `json:"verification_code"`
	}
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	phone, err := common.NormalizePhone(req.Phone)
	if err != nil || req.VerificationCode == "" {
		common.ApiErrorI18n(c, i18n.MsgInvalidParams)
		return
	}
	fail := func() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "phone or verification code error"})
	}
	if !common.VerifyCodeWithKey(phone, req.VerificationCode, common.SmsLoginPurpose) {
		fail()
		return
	}
	user, err := model.GetUniqueUserByPhone(phone)
	if err != nil || user.Status != common.UserStatusEnabled {
		fail()
		return
	}
	// same 2FA pending session branch as Login, then setupLogin(user, c)
}
```

Add `case "/api/user/login/sms": return "sms"` in `loginMethodFromContext`.

- [ ] **Step 4: Ensure GetSelf / setupLogin JSON include `phone` if they already project email** (do not put phone into localStorage-sensitive dumps if Clean already strips secrets — phone is OK like email).

- [ ] **Step 5: Run tests**

```bash
go test ./controller -run 'TestDecideRegisterContact|TestNormalizePhone' -count=1
go test ./common -count=1
```

- [ ] **Step 6: Commit**

```bash
git add controller/user.go controller/register_verification.go controller/register_verification_test.go
git commit -m "feat(sms): register email/phone either-or and SMS login"
```

---

### Task 7: Default — system settings UI

**Files:**
- Modify: `web/default/src/features/system-settings/types.ts`
- Modify: `web/default/src/features/system-settings/auth/basic-auth-section.tsx`
- Modify: `web/default/src/features/system-settings/auth/section-registry.tsx`
- Modify: `web/default/src/features/system-settings/auth/index.tsx`
- Create: `web/default/src/features/system-settings/integrations/aliyun-sms-settings-section.tsx` (clone email-settings-section shape)
- Modify: `web/default/src/features/system-settings/operations/section-registry.tsx` (+ index defaults)

- [ ] **Step 1: Types + basic-auth switches** for `SmsVerificationEnabled`, `SmsLoginEnabled`.

- [ ] **Step 2: Aliyun SMS settings section** fields: AccessKeyId, AccessKeySecret (password input, submit only if non-empty), SignName, TemplateCode, TemplateParamCodeKey, Endpoint. Save via `useUpdateOption` like SMTP.

- [ ] **Step 3: Register section in operations registry** title key `'Aliyun SMS'`.

- [ ] **Step 4: i18n** — add English keys used in `t('...')` to locales (run `bun run i18n:sync` from `web/default` per project skill if available).

- [ ] **Step 5: `bun run typecheck` in `web/default`.

- [ ] **Step 6: Commit**

```bash
git add web/default/src/features/system-settings web/default/src/i18n
git commit -m "feat(sms): Default admin SMS switches and Aliyun settings"
```

---

### Task 8: Default — register / login / profile

**Files:**
- Modify: status/types that parse `/api/status` (find `email_verification` usage)
- Modify: `web/default/src/features/auth/sign-up/components/sign-up-form.tsx`
- Modify: `web/default/src/features/auth/api.ts` (+ hooks)
- Modify: `web/default/src/features/auth/sign-in/components/user-auth-form.tsx` (or tabs wrapper)
- Create: profile `phone-bind-dialog.tsx` (clone email-bind-dialog)
- Modify: `account-bindings-tab.tsx`

- [ ] **Step 1: API helpers**

```ts
export async function sendSmsVerification(phone: string, turnstileToken?: string) { ... `/api/verification/sms?phone=` }
export async function sendSmsLoginCode(phone: string, turnstileToken?: string) { ... `/api/verification/sms_login?phone=` }
export async function loginWithSms(phone: string, verification_code: string) { ... POST `/api/user/login/sms` }
export async function sendSmsBind(phone: string) { ... GET `/api/user/sms_bind?phone=` }
export async function bindPhone(phone: string, code: string) { ... POST `/api/user/phone/bind` }
```

- [ ] **Step 2: Sign-up UI**

When `status.sms_verification` and/or `status.email_verification`: show toggle Email | Phone. Phone path: phone input + code + send. Submit payload includes `phone` + `verification_code` without requiring email when on phone path.

- [ ] **Step 3: Sign-in UI**

If `status.sms_login`: Tabs “Password” | “SMS code”. SMS tab calls send + `loginWithSms`. Handle `require_2fa` same as password login.

- [ ] **Step 4: Profile bind phone dialog + show masked phone on bindings tab.**

- [ ] **Step 5: typecheck + i18n.**

- [ ] **Step 6: Commit**

```bash
git add web/default/src/features/auth web/default/src/features/profile web/default/src/i18n
git commit -m "feat(sms): Default sign-up, SMS login, and phone bind UI"
```

---

### Task 9: Classic theme parity

**Files:**
- Modify: `web/classic/src/components/settings/SystemSetting.jsx` (switches + Aliyun fields in operations/SMTP area)
- Modify: `web/classic/src/components/auth/RegisterForm.jsx`
- Modify: `web/classic/src/components/auth/LoginForm.jsx`
- Create: `web/classic/src/components/settings/personal/modals/PhoneBindModal.jsx`
- Modify: `web/classic/src/components/settings/PersonalSetting.jsx`

- [ ] **Step 1: SystemSetting** — load/save same option keys as Default.

- [ ] **Step 2: RegisterForm** — either-or UI driven by status `email_verification` / `sms_verification`.

- [ ] **Step 3: LoginForm** — tab for SMS login when `sms_login`.

- [ ] **Step 4: PhoneBindModal** — mirror EmailBindModal.

- [ ] **Step 5: Manual smoke checklist** documented in commit body (Classic often lacks typecheck).

- [ ] **Step 6: Commit**

```bash
git add web/classic
git commit -m "feat(sms): Classic theme SMS register, login, and bind"
```

---

### Task 10: End-to-end verification checklist

- [ ] **Step 1: Backend compile**

```bash
go test ./common ./controller ./model -count=1
go build -o nul .
```

Expected: PASS / build OK

- [ ] **Step 2: Manual API smoke (with real Aliyun creds in admin UI)**

1. Enable `SmsVerificationEnabled`, configure Aliyun SMS, send `/api/verification/sms?phone=...`
2. Register with phone + code
3. Enable `SmsLoginEnabled`, send login code, `POST /api/user/login/sms`
4. Bind phone from authenticated session

- [ ] **Step 3: Mark design status implemented** (optional one-line update in spec header)

- [ ] **Step 4: Final commit if checklist notes/docs changed**

---

## Spec coverage self-check

| Spec item | Task |
|-----------|------|
| `users.phone` + unique | Task 2 |
| Options + Aliyun config | Tasks 3–4, 7, 9 |
| Status flags | Task 3, 8–9 |
| Register E/S either-or | Task 6, 8–9 |
| Purposes sr/sl/sb | Task 1, 5 |
| Send APIs + rate limit | Task 5 |
| Login SMS + 2FA | Task 6, 8–9 |
| Phone bind | Task 5, 8–9 |
| Default UI | Tasks 7–8 |
| Classic UI | Task 9 |
| No SMS password reset / international / newuser | Intentionally omitted |

## Placeholder / consistency notes

- Dysmsapi module major (`v3`/`v4`) resolved at `go get` time in Task 4.
- Register prefers phone when both submitted under dual-enable (matches spec).
- Secret option omitted from GetOptions via existing `Secret` suffix.
