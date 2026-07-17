package controller

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type newuserLoginRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	OwnerUserId int    `json:"owner_user_id"`
}

type newuserRegisterRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	DisplayName      string `json:"display_name"`
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	VerificationCode string `json:"verification_code"`
	OwnerUserId      int    `json:"owner_user_id"`
	RegisterCode     string `json:"register_code"`
	RegisterType     string `json:"register_type"`
}

type newuserAdminCreateRequest struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	QuotaLimit  int    `json:"quota_limit"`
}

type newuserAdminUpdateRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Status      int    `json:"status"`
	QuotaLimit  int    `json:"quota_limit"`
	Password    string `json:"password"`
}

type newuserAdminSettingsRequest struct {
	Enabled         bool   `json:"enabled"`
	RegisterEnabled bool   `json:"register_enabled"`
	RegisterCode    string `json:"register_code"`
}

func newuserOwnerInfo(ownerUserId int) gin.H {
	brief := model.GetNewuserOwnerBrief(ownerUserId)
	return gin.H{
		"owner_user_id":      ownerUserId,
		"owner_username":     brief.Username,
		"owner_display_name": brief.DisplayName,
		"owner_quota":        brief.Quota,
		"owner_used_quota":   brief.UsedQuota,
	}
}

// verifyNewuserRegisterContact validates optional email/phone verification codes.
// When both email and SMS verification are enabled, pass only one contact channel.
func verifyNewuserRegisterContact(email, phone, verificationCode string) error {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	needEmail := email != "" && common.EmailVerificationEnabled
	needSms := phone != "" && common.SmsVerificationEnabled
	if needEmail && needSms {
		return errors.New("provide either email or phone with verification_code, not both")
	}
	if needEmail {
		email = model.NormalizeEmail(email)
		if verificationCode == "" || !common.VerifyCodeWithKey(email, verificationCode, common.EmailVerificationPurpose) {
			return errors.New("invalid verification code")
		}
		return nil
	}
	if needSms {
		normalized, err := common.NormalizePhone(phone)
		if err != nil {
			return errors.New("invalid phone number")
		}
		if verificationCode == "" || !common.VerifyCodeWithKey(normalized, verificationCode, common.SmsRegisterPurpose) {
			return errors.New("invalid verification code")
		}
	}
	return nil
}

func newuserAdminOwnerId(c *gin.Context) int {
	if id := c.GetInt("id"); id > 0 {
		return id
	}
	if raw, ok := c.Get("newuser"); ok {
		return raw.(*model.Newuser).OwnerUserId
	}
	return 0
}

func newuserAccountMeta(nu *model.Newuser) gin.H {
	registerType := model.NewuserRegisterTypeMember
	if nu.IsOrgOwner {
		registerType = model.NewuserRegisterTypeOrg
	}
	return gin.H{
		"is_org_owner":       nu.IsOrgOwner,
		"can_manage_users":   nu.CanManageNewusers(),
		"register_type":      registerType,
	}
}

func newuserAuthResponse(nu *model.Newuser, jwtToken, apiKey string) gin.H {
	nu.Clean()
	data := gin.H{
		"id":           nu.Id,
		"username":     nu.Username,
		"display_name": nu.DisplayName,
		"token":        jwtToken,
		"api_key":      apiKey,
	}
	for k, v := range newuserOwnerInfo(nu.OwnerUserId) {
		data[k] = v
	}
	for k, v := range newuserAccountMeta(nu) {
		data[k] = v
	}
	return data
}

func NewuserLogin(c *gin.Context) {
	var req newuserLoginRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	var nu *model.Newuser
	var err error
	if req.OwnerUserId > 0 {
		nu, err = model.AuthenticateNewuser(req.OwnerUserId, req.Username, req.Password)
	} else {
		nu, err = model.AuthenticateNewuserByUsername(req.Username, req.Password)
	}
	if err != nil {
		msg := "username or password error"
		if errors.Is(err, model.ErrNewuserDisabled) {
			msg = "user disabled"
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	if model.IsNewuserQuotaExceeded(nu.TokenId, nu.QuotaLimit) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "quota limit exceeded"})
		return
	}
	token, err := model.GetTokenByIds(nu.TokenId, nu.OwnerUserId)
	if err != nil || token.Status != common.TokenStatusEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "api key unavailable"})
		return
	}
	jwtToken, err := service.IssueNewuserJWT(nu.Id, nu.OwnerUserId, nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	_ = nu.TouchLastLogin()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    newuserAuthResponse(nu, jwtToken, "sk-"+token.Key),
	})
}

func NewuserRegister(c *gin.Context) {
	var req newuserRegisterRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}

	registerType := model.NormalizeNewuserRegisterType(req.RegisterType, req.OwnerUserId, req.RegisterCode)
	var nu *model.Newuser
	var token *model.Token
	var err error

	switch registerType {
	case model.NewuserRegisterTypeOrg:
		nu, token, err = model.CreateNewuserOrgWithMainUser(req.Username, req.Password, req.DisplayName, 0)
	default:
		ownerId := req.OwnerUserId
		if ownerId <= 0 {
			if strings.TrimSpace(req.RegisterCode) == "" {
				ownerId, err = middleware.ResolveNewuserOwnerId(c, req.OwnerUserId)
				if err != nil {
					c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
					return
				}
			} else {
				ownerId, err = model.FindOwnerUserIdByRegisterCode(req.RegisterCode)
				if err != nil {
					c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid register code"})
					return
				}
			}
		}
		if !model.IsNewuserOwnerRegisterEnabled(ownerId) {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "registration disabled for this organization"})
			return
		}
		expectedCode := model.GetNewuserOwnerRegisterCode(ownerId)
		if expectedCode == "" || req.RegisterCode != expectedCode {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid register code"})
			return
		}
		if err = verifyNewuserRegisterContact(req.Email, req.Phone, req.VerificationCode); err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
			return
		}
		nu, token, err = model.CreateNewuserAccount(ownerId, req.Username, req.Password, req.DisplayName, 0, req.Email, req.Phone)
	}

	if err != nil {
		msg := err.Error()
		if errors.Is(err, model.ErrNewuserDuplicate) {
			msg = "username already exists"
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}

	jwtToken, err := service.IssueNewuserJWT(nu.Id, nu.OwnerUserId, nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    newuserAuthResponse(nu, jwtToken, "sk-"+token.Key),
	})
}

// NewuserRegisterInfo resolves a unique invite code for the public web register page.
func NewuserRegisterInfo(c *gin.Context) {
	code := strings.TrimSpace(c.Query("code"))
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "register code is required"})
		return
	}
	ownerId, err := model.FindOwnerUserIdByRegisterCode(code)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid register code"})
		return
	}
	if !model.IsNewuserOwnerRegisterEnabled(ownerId) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "registration disabled for this organization"})
		return
	}
	brief := model.GetNewuserOwnerBrief(ownerId)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"owner_user_id":      ownerId,
			"owner_username":     brief.Username,
			"owner_display_name": brief.DisplayName,
			"register_code":      code,
		},
	})
}

func NewuserGetSelf(c *gin.Context) {
	nu := c.MustGet("newuser").(*model.Newuser)
	nu.Clean()
	used, remain, unlimited, _ := model.GetNewuserUsage(nu.TokenId)
	data := gin.H{
		"id":           nu.Id,
		"username":     nu.Username,
		"display_name": nu.DisplayName,
		"token_id":     nu.TokenId,
		"quota_limit":  nu.QuotaLimit,
		"used_quota":   used,
		"remain_quota": remain,
		"unlimited":    unlimited,
		"status":       nu.Status,
		"last_login":   nu.LastLoginTime,
	}
	for k, v := range newuserOwnerInfo(nu.OwnerUserId) {
		data[k] = v
	}
	for k, v := range newuserAccountMeta(nu) {
		data[k] = v
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

type newuserChangePasswordRequest struct {
	OriginalPassword string `json:"original_password"`
	Password         string `json:"password"`
}

// NewuserChangePassword lets a logged-in team user change their own password (JWT).
// If the account is is_org_owner, also syncs the password to the mirrored users row.
func NewuserChangePassword(c *gin.Context) {
	nu := c.MustGet("newuser").(*model.Newuser)
	var req newuserChangePasswordRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	if req.OriginalPassword == "" || req.Password == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "original_password and password are required"})
		return
	}
	if len(req.Password) < model.NewuserPasswordMinLength || len(req.Password) > model.NewuserPasswordMaxLength {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "password length invalid"})
		return
	}
	if req.OriginalPassword == req.Password {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "new password must be different from current password"})
		return
	}
	fresh, err := model.GetNewuserById(nu.Id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user not found"})
		return
	}
	if fresh.Password == "" || !common.ValidatePasswordAndHash(req.OriginalPassword, fresh.Password) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "original password error"})
		return
	}
	hashed, err := common.Password2Hash(req.Password)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := fresh.UpdatePassword(hashed); err != nil {
		common.ApiError(c, err)
		return
	}
	if fresh.IsOrgOwner {
		if syncErr := model.SyncMainUserPasswordFromOrgOwner(fresh.OwnerUserId, hashed); syncErr != nil {
			common.SysError(fmt.Sprintf("sync main user password from org-owner newuser %d: %v", fresh.Id, syncErr))
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func NewuserGetToken(c *gin.Context) {
	nu := c.MustGet("newuser").(*model.Newuser)
	if model.IsNewuserQuotaExceeded(nu.TokenId, nu.QuotaLimit) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "quota limit exceeded"})
		return
	}
	token, err := model.GetTokenByIds(nu.TokenId, nu.OwnerUserId)
	if err != nil || token.Status != common.TokenStatusEnabled {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "api key unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"api_key":    "sk-" + token.Key,
			"token_id":   token.Id,
			"token_name": token.Name,
		},
	})
}

func NewuserGetUsage(c *gin.Context) {
	nu := c.MustGet("newuser").(*model.Newuser)
	used, remain, unlimited, err := model.GetNewuserUsage(nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logs, err := model.GetLogByTokenId(nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{
		"quota_limit":  nu.QuotaLimit,
		"used_quota":   used,
		"remain_quota": remain,
		"unlimited":    unlimited,
		"recent_logs":  logs,
	}
	for k, v := range newuserOwnerInfo(nu.OwnerUserId) {
		data[k] = v
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

func newuserAdminItem(nu *model.Newuser) gin.H {
	nu.Clean()
	used, remain, unlimited, _ := model.GetNewuserUsage(nu.TokenId)
	return gin.H{
		"id":              nu.Id,
		"owner_user_id":   nu.OwnerUserId,
		"username":        nu.Username,
		"display_name":    nu.DisplayName,
		"status":          nu.Status,
		"token_id":        nu.TokenId,
		"quota_limit":     nu.QuotaLimit,
		"created_time":    nu.CreatedTime,
		"last_login_time": nu.LastLoginTime,
		"used_quota":      used,
		"remain_quota":    remain,
		"unlimited":       unlimited,
		"is_org_owner":    nu.IsOrgOwner,
		"can_manage_users": nu.CanManageNewusers(),
	}
}

func NewuserAdminList(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	users, err := model.ListNewusersByOwner(ownerId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(users))
	var totalUsed int
	activeCount := 0
	for _, u := range users {
		item := newuserAdminItem(u)
		if used, ok := item["used_quota"].(int); ok {
			totalUsed += used
		}
		if u.Status == common.UserStatusEnabled {
			activeCount++
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"items": items,
			"summary": gin.H{
				"total_users":   len(users),
				"active_users":  activeCount,
				"total_used":    totalUsed,
			},
		},
	})
}

func NewuserAdminGetUsage(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	nu, err := model.GetNewuserById(id)
	if err != nil || nu.OwnerUserId != ownerId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user not found"})
		return
	}
	used, remain, unlimited, err := model.GetNewuserUsage(nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	logs, err := model.GetLogByTokenId(nu.TokenId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	nu.Clean()
	data := gin.H{
		"user":         nu,
		"quota_limit":  nu.QuotaLimit,
		"used_quota":   used,
		"remain_quota": remain,
		"unlimited":    unlimited,
		"recent_logs":  logs,
	}
	for k, v := range newuserOwnerInfo(nu.OwnerUserId) {
		data[k] = v
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    data,
	})
}

func NewuserAdminCreate(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	if ownerId <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "unauthorized"})
		return
	}
	var req newuserAdminCreateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	nu, token, err := model.CreateNewuserAccount(ownerId, req.Username, req.Password, req.DisplayName, req.QuotaLimit, req.Email, req.Phone)
	if err != nil {
		msg := err.Error()
		if errors.Is(err, model.ErrNewuserDuplicate) {
			msg = "username already exists"
		}
		c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
		return
	}
	nu.Clean()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"user":    nu,
			"token_id": token.Id,
		},
	})
}

func NewuserAdminUpdate(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	nu, err := model.GetNewuserById(id)
	if err != nil || nu.OwnerUserId != ownerId {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "user not found"})
		return
	}
	var req newuserAdminUpdateRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	if strings.TrimSpace(req.DisplayName) != "" {
		nu.DisplayName = strings.TrimSpace(req.DisplayName)
	}
	email, phone, contactErr := model.NormalizeNewuserContact(req.Email, req.Phone)
	if contactErr != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": contactErr.Error()})
		return
	}
	if email != "" && model.IsNewuserEmailTaken(email, nu.Id) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "email already exists"})
		return
	}
	if phone != "" && model.IsNewuserPhoneTaken(phone, nu.Id) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "phone already exists"})
		return
	}
	nu.Email = email
	nu.Phone = phone
	if req.Status == common.UserStatusEnabled || req.Status == common.UserStatusDisabled {
		nu.Status = req.Status
		if nu.TokenId > 0 {
			token, tokenErr := model.GetTokenByIds(nu.TokenId, ownerId)
			if tokenErr == nil {
				if req.Status == common.UserStatusEnabled {
					token.Status = common.TokenStatusEnabled
				} else {
					token.Status = common.TokenStatusDisabled
				}
				_ = token.Update()
			}
		}
	}
	if req.QuotaLimit >= 0 {
		nu.QuotaLimit = req.QuotaLimit
		_ = model.ApplyNewuserQuotaLimit(nu.TokenId, req.QuotaLimit)
	}
	if req.Password != "" {
		if len(req.Password) < model.NewuserPasswordMinLength || len(req.Password) > model.NewuserPasswordMaxLength {
			c.JSON(http.StatusOK, gin.H{"success": false, "message": "password length invalid"})
			return
		}
		hashed, hashErr := common.Password2Hash(req.Password)
		if hashErr != nil {
			common.ApiError(c, hashErr)
			return
		}
		_ = nu.UpdatePassword(hashed)
	}
	if err := nu.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	nu.Clean()
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": nu})
}

func NewuserAdminDelete(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid id"})
		return
	}
	if err := model.DeleteNewuserById(id, ownerId); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func NewuserAdminGetSettings(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":          model.IsNewuserOwnerEnabled(ownerId),
			"register_enabled": model.IsNewuserOwnerRegisterEnabled(ownerId),
			"register_code":    model.GetNewuserOwnerRegisterCode(ownerId),
			"owner_user_id":    ownerId,
		},
	})
}

func NewuserAdminUpdateSettings(c *gin.Context) {
	ownerId := newuserAdminOwnerId(c)
	var req newuserAdminSettingsRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	registerCode := strings.TrimSpace(req.RegisterCode)
	if req.RegisterEnabled && registerCode == "" {
		registerCode = model.GenerateUniqueNewuserRegisterCode()
	}
	if req.RegisterEnabled && registerCode == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "register_code is required when register is enabled"})
		return
	}
	if err := model.SetNewuserOwnerSettings(ownerId, req.Enabled, req.RegisterEnabled, registerCode); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"enabled":          req.Enabled,
			"register_enabled": req.RegisterEnabled,
			"register_code":    registerCode,
			"owner_user_id":    ownerId,
		},
	})
}

func newuserSmsPasswordResetEnabled() bool {
	return common.SmsLoginEnabled || common.SmsVerificationEnabled
}

// NewuserSendPasswordResetEmail sends a short code (desktop-friendly, not a web link).
// By default anti-enumeration (always success). Pass require_exists=1 to fail when unbound.
func NewuserSendPasswordResetEmail(c *gin.Context) {
	email := model.NormalizeEmail(c.Query("email"))
	if email == "" || !strings.Contains(email, "@") {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid email"})
		return
	}
	requireExists := newuserRequireExists(c)
	nu, err := model.GetEnabledNewuserByEmail(email)
	if err != nil || nu == nil {
		if requireExists {
			msg := "该邮箱未绑定团队账号"
			if err != nil && !errors.Is(err, model.ErrNewuserEmailNotFound) {
				msg = "该邮箱无法用于找回密码，请联系管理员"
			}
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		if err != nil && !errors.Is(err, model.ErrNewuserEmailNotFound) {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("skip newuser password reset email for %s: %s", email, err.Error()))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
		return
	}
	code := common.GenerateVerificationCode(6)
	common.RegisterVerificationCodeWithKey(email, code, common.NewuserPasswordResetPurpose)
	subject := fmt.Sprintf("%s团队账号密码重置", common.SystemName)
	content := fmt.Sprintf("<p>您好，你正在进行%s团队账号密码重置。</p>"+
		"<p>您的验证码为: <strong>%s</strong></p>"+
		"<p>验证码 %d 分钟内有效，如果不是本人操作，请忽略。</p>",
		common.SystemName, code, common.VerificationValidMinutes)
	if sendErr := common.SendEmail(subject, email, content); sendErr != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("failed to send newuser password reset email to %s: %s", email, sendErr.Error()))
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

func newuserRequireExists(c *gin.Context) bool {
	v := strings.TrimSpace(c.Query("require_exists"))
	return v == "1" || strings.EqualFold(v, "true")
}

// NewuserSendPasswordResetSms sends SMS reset code for team users.
// By default anti-enumeration. Pass require_exists=1 to fail when unbound.
func NewuserSendPasswordResetSms(c *gin.Context) {
	if !newuserSmsPasswordResetEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sms password reset disabled"})
		return
	}
	phone, err := common.NormalizePhone(c.Query("phone"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid phone number"})
		return
	}
	requireExists := newuserRequireExists(c)
	nu, lookupErr := model.GetEnabledNewuserByPhone(phone)
	if lookupErr != nil || nu == nil {
		if requireExists {
			msg := "该手机号未绑定团队账号"
			if lookupErr != nil && !errors.Is(lookupErr, model.ErrNewuserPhoneNotFound) {
				msg = "该手机号无法用于找回密码，请联系管理员"
			}
			c.JSON(http.StatusOK, gin.H{"success": false, "message": msg})
			return
		}
		if lookupErr != nil && !errors.Is(lookupErr, model.ErrNewuserPhoneNotFound) {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("skip newuser sms password reset for %s: %s", common.MaskPhone(phone), lookupErr.Error()))
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
		return
	}
	if nu.Status == common.UserStatusEnabled && common.AliyunSmsConfigured() {
		if err := common.AllowSmsSend(c.ClientIP(), phone); err != nil {
			logger.LogWarn(c.Request.Context(), fmt.Sprintf("newuser sms password reset blocked by abuse guard for %s: %s", common.MaskPhone(phone), err.Error()))
		} else {
			code := common.GenerateNumericVerificationCode(6)
			common.RegisterVerificationCodeWithKey(phone, code, common.NewuserSmsPasswordResetPurpose)
			if err := common.SendAliyunSms(phone, code); err != nil {
				logger.LogError(c.Request.Context(), fmt.Sprintf("failed to send newuser sms password reset to %s: %s", common.MaskPhone(phone), err.Error()))
			} else {
				common.MarkSmsSent(c.ClientIP(), phone)
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": ""})
}

type newuserResetPasswordRequest struct {
	Email            string `json:"email"`
	Phone            string `json:"phone"`
	VerificationCode string `json:"verification_code"`
}

// NewuserResetPassword resets team user password by email + verification code.
// Returns a newly generated password (desktop-friendly).
func NewuserResetPassword(c *gin.Context) {
	var req newuserResetPasswordRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	email := model.NormalizeEmail(req.Email)
	if email == "" || req.VerificationCode == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	if !common.VerifyCodeWithKey(email, req.VerificationCode, common.NewuserPasswordResetPurpose) {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "email or verification code error"})
		return
	}
	password := common.GenerateVerificationCode(12)
	if err := model.ResetNewuserPasswordByEmail(email, password); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "email or verification code error"})
		return
	}
	common.DeleteKey(email, common.NewuserPasswordResetPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    password,
	})
}

// NewuserResetPasswordBySms resets team user password by phone + SMS code.
func NewuserResetPasswordBySms(c *gin.Context) {
	if !newuserSmsPasswordResetEnabled() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "sms password reset disabled"})
		return
	}
	var req newuserResetPasswordRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	phone, err := common.NormalizePhone(req.Phone)
	if err != nil || req.VerificationCode == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "invalid parameters"})
		return
	}
	fail := func() {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "phone or verification code error"})
	}
	if !common.VerifyCodeWithKey(phone, req.VerificationCode, common.NewuserSmsPasswordResetPurpose) {
		fail()
		return
	}
	password := common.GenerateVerificationCode(12)
	if err := model.ResetNewuserPasswordByPhone(phone, password); err != nil {
		fail()
		return
	}
	common.DeleteKey(phone, common.NewuserSmsPasswordResetPurpose)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    password,
	})
}
