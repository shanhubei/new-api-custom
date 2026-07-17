package model

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	NewuserUsernameMaxLength = 64
	NewuserPasswordMinLength = 8
	NewuserPasswordMaxLength = 64

	NewuserRegisterTypeOrg    = "org"
	NewuserRegisterTypeMember = "member"
)

var (
	ErrNewuserNotFound      = errors.New("newuser not found")
	ErrNewuserDisabled      = errors.New("newuser disabled")
	ErrNewuserWrongPassword = errors.New("newuser wrong password")
	ErrNewuserDuplicate     = errors.New("newuser already exists")
)

// Newuser 团队用户，映射到 OwnerUserId（原有 new-api 用户，即“组织/团队”）下的一个 Token。
// Username 全站唯一，登录时仅需 username + password 即可定位所属组织。
type Newuser struct {
	Id            int            `json:"id"`
	OwnerUserId   int            `json:"owner_user_id" gorm:"index"`
	Username      string         `json:"username" gorm:"size:64;uniqueIndex"`
	Password      string         `json:"-"`
	DisplayName   string         `json:"display_name" gorm:"size:64"`
	Email         string         `json:"email" gorm:"size:64;index"`
	Phone         string         `json:"phone" gorm:"size:20;index"`
	Status        int            `json:"status" gorm:"type:int;default:1"`
	TokenId       int            `json:"token_id" gorm:"index"`
	QuotaLimit    int            `json:"quota_limit" gorm:"type:int;default:0"`
	IsOrgOwner    bool           `json:"is_org_owner" gorm:"default:false"`
	CreatedTime   int64          `json:"created_time" gorm:"bigint"`
	LastLoginTime int64          `json:"last_login_time" gorm:"bigint;default:0"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (nu *Newuser) Clean() {
	nu.Password = ""
}

func (nu *Newuser) CanManageNewusers() bool {
	return nu.IsOrgOwner
}

func IsNewuserOrgSelfRegisterEnabled() bool {
	if !common.NewuserEnabled {
		return false
	}
	return common.NewuserOrgSelfRegisterEnabled
}

func NormalizeNewuserRegisterType(registerType string, ownerUserId int, registerCode string) string {
	registerType = strings.ToLower(strings.TrimSpace(registerType))
	switch registerType {
	case NewuserRegisterTypeOrg, NewuserRegisterTypeMember:
		return registerType
	}
	if ownerUserId > 0 || strings.TrimSpace(registerCode) != "" {
		return NewuserRegisterTypeMember
	}
	return NewuserRegisterTypeOrg
}

func newuserOwnerOptionKey(ownerUserId int, suffix string) string {
	return fmt.Sprintf("newuser.owner.%d.%s", ownerUserId, suffix)
}

func IsNewuserOwnerEnabled(ownerUserId int) bool {
	if !common.NewuserEnabled {
		return false
	}
	if ownerUserId <= 0 {
		return false
	}
	key := newuserOwnerOptionKey(ownerUserId, "enabled")
	val, ok := common.OptionMap[key]
	if !ok || val == "" {
		return true
	}
	return val == "true"
}

func IsNewuserOwnerRegisterEnabled(ownerUserId int) bool {
	if !IsNewuserOwnerEnabled(ownerUserId) {
		return false
	}
	key := newuserOwnerOptionKey(ownerUserId, "register_enabled")
	val, ok := common.OptionMap[key]
	if !ok || val == "" {
		return false
	}
	return val == "true"
}

func GetNewuserOwnerRegisterCode(ownerUserId int) string {
	return common.OptionMap[newuserOwnerOptionKey(ownerUserId, "register_code")]
}

// FindOwnerUserIdByRegisterCode resolves the organization from a globally unique invite code.
func FindOwnerUserIdByRegisterCode(code string) (int, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return 0, errors.New("invalid register code")
	}
	const prefix = "newuser.owner."
	const suffix = ".register_code"
	found := 0
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	for key, val := range common.OptionMap {
		if val != code || !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
			continue
		}
		mid := strings.TrimSuffix(strings.TrimPrefix(key, prefix), suffix)
		id, err := strconv.Atoi(mid)
		if err != nil || id <= 0 {
			continue
		}
		if found > 0 && found != id {
			return 0, errors.New("register code conflict")
		}
		found = id
	}
	if found == 0 {
		return 0, errors.New("invalid register code")
	}
	return found, nil
}

// IsNewuserRegisterCodeTaken reports whether another organization already uses this invite code.
func IsNewuserRegisterCodeTaken(code string, excludeOwnerId int) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	ownerId, err := FindOwnerUserIdByRegisterCode(code)
	if err != nil {
		return false
	}
	return ownerId != excludeOwnerId
}

func GenerateUniqueNewuserRegisterCode() string {
	for i := 0; i < 16; i++ {
		code := common.GetRandomString(10)
		if !IsNewuserRegisterCodeTaken(code, 0) {
			return code
		}
	}
	return common.GetRandomString(16)
}

func SetNewuserOwnerSettings(ownerUserId int, enabled, registerEnabled bool, registerCode string) error {
	registerCode = strings.TrimSpace(registerCode)
	if registerCode != "" && IsNewuserRegisterCodeTaken(registerCode, ownerUserId) {
		return errors.New("register code already used by another organization")
	}
	values := map[string]string{
		newuserOwnerOptionKey(ownerUserId, "enabled"):          boolToOption(enabled),
		newuserOwnerOptionKey(ownerUserId, "register_enabled"): boolToOption(registerEnabled),
		newuserOwnerOptionKey(ownerUserId, "register_code"):    registerCode,
	}
	return UpdateOptionsBulk(values)
}

func NormalizeNewuserContact(email, phone string) (string, string, error) {
	email = strings.TrimSpace(email)
	phone = strings.TrimSpace(phone)
	if email != "" {
		email = NormalizeEmail(email)
		if !strings.Contains(email, "@") {
			return "", "", errors.New("invalid email")
		}
	}
	if phone != "" {
		normalized, err := common.NormalizePhone(phone)
		if err != nil {
			return "", "", errors.New("invalid phone number")
		}
		phone = normalized
	}
	return email, phone, nil
}

func IsNewuserEmailTaken(email string, excludeId int) bool {
	email = NormalizeEmail(email)
	if email == "" {
		return false
	}
	var count int64
	q := DB.Model(&Newuser{}).Where("email = ?", email)
	if excludeId > 0 {
		q = q.Where("id <> ?", excludeId)
	}
	_ = q.Count(&count).Error
	return count > 0
}

func IsNewuserPhoneTaken(phone string, excludeId int) bool {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return false
	}
	var count int64
	q := DB.Model(&Newuser{}).Where("phone = ?", phone)
	if excludeId > 0 {
		q = q.Where("id <> ?", excludeId)
	}
	_ = q.Count(&count).Error
	return count > 0
}

var (
	ErrNewuserEmailNotFound = errors.New("newuser email not found")
	ErrNewuserPhoneNotFound = errors.New("newuser phone not found")
	ErrNewuserContactAmbiguous = errors.New("newuser contact matches multiple accounts")
)

// GetEnabledNewuserByEmail returns the single enabled team user with this email.
func GetEnabledNewuserByEmail(email string) (*Newuser, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return nil, ErrNewuserEmailNotFound
	}
	var users []*Newuser
	err := DB.Where("email = ? AND status = ?", email, common.UserStatusEnabled).Find(&users).Error
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrNewuserEmailNotFound
	}
	if len(users) > 1 {
		return nil, ErrNewuserContactAmbiguous
	}
	return users[0], nil
}

// GetEnabledNewuserByPhone returns the single enabled team user with this phone.
func GetEnabledNewuserByPhone(phone string) (*Newuser, error) {
	phone = strings.TrimSpace(phone)
	if phone == "" {
		return nil, ErrNewuserPhoneNotFound
	}
	var users []*Newuser
	err := DB.Where("phone = ? AND status = ?", phone, common.UserStatusEnabled).Find(&users).Error
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, ErrNewuserPhoneNotFound
	}
	if len(users) > 1 {
		return nil, ErrNewuserContactAmbiguous
	}
	return users[0], nil
}

func ResetNewuserPasswordByEmail(email, plainPassword string) error {
	nu, err := GetEnabledNewuserByEmail(email)
	if err != nil {
		return err
	}
	hashed, err := common.Password2Hash(plainPassword)
	if err != nil {
		return err
	}
	return nu.UpdatePassword(hashed)
}

func ResetNewuserPasswordByPhone(phone, plainPassword string) error {
	nu, err := GetEnabledNewuserByPhone(phone)
	if err != nil {
		return err
	}
	hashed, err := common.Password2Hash(plainPassword)
	if err != nil {
		return err
	}
	return nu.UpdatePassword(hashed)
}

func boolToOption(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func ValidateNewuserCredentials(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return errors.New("username or password is empty")
	}
	if len(username) > NewuserUsernameMaxLength {
		return errors.New("username too long")
	}
	if len(password) < NewuserPasswordMinLength || len(password) > NewuserPasswordMaxLength {
		return errors.New("password length invalid")
	}
	return nil
}

func GetNewuserById(id int) (*Newuser, error) {
	if id <= 0 {
		return nil, ErrNewuserNotFound
	}
	nu := &Newuser{}
	err := DB.First(nu, "id = ?", id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNewuserNotFound
		}
		return nil, err
	}
	return nu, nil
}

func GetNewuserByUsername(username string) (*Newuser, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return nil, ErrNewuserNotFound
	}
	nu := &Newuser{}
	err := DB.Where("username = ?", username).First(nu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNewuserNotFound
		}
		return nil, err
	}
	return nu, nil
}

func GetNewuserByOwnerAndUsername(ownerUserId int, username string) (*Newuser, error) {
	username = strings.TrimSpace(username)
	if ownerUserId <= 0 || username == "" {
		return nil, ErrNewuserNotFound
	}
	nu := &Newuser{}
	err := DB.Where("owner_user_id = ? AND username = ?", ownerUserId, username).First(nu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNewuserNotFound
		}
		return nil, err
	}
	return nu, nil
}

func ListNewusersByOwner(ownerUserId int) ([]*Newuser, error) {
	if ownerUserId <= 0 {
		return nil, errors.New("invalid owner user id")
	}
	var users []*Newuser
	err := DB.Where("owner_user_id = ?", ownerUserId).Order("id desc").Find(&users).Error
	return users, err
}

func AuthenticateNewuserByUsername(username, password string) (*Newuser, error) {
	if err := ValidateNewuserCredentials(username, password); err != nil {
		return nil, err
	}
	nu, err := GetNewuserByUsername(username)
	if err != nil {
		if errors.Is(err, ErrNewuserNotFound) {
			return nil, ErrNewuserWrongPassword
		}
		return nil, err
	}
	if !IsNewuserOwnerEnabled(nu.OwnerUserId) {
		return nil, errors.New("newuser module disabled for this organization")
	}
	if nu.Status != common.UserStatusEnabled {
		return nil, ErrNewuserDisabled
	}
	if !common.ValidatePasswordAndHash(password, nu.Password) {
		return nil, ErrNewuserWrongPassword
	}
	return nu, nil
}

func AuthenticateNewuser(ownerUserId int, username, password string) (*Newuser, error) {
	if err := ValidateNewuserCredentials(username, password); err != nil {
		return nil, err
	}
	if !IsNewuserOwnerEnabled(ownerUserId) {
		return nil, errors.New("newuser module disabled for this organization")
	}
	nu, err := GetNewuserByOwnerAndUsername(ownerUserId, username)
	if err != nil {
		if errors.Is(err, ErrNewuserNotFound) {
			return nil, ErrNewuserWrongPassword
		}
		return nil, err
	}
	if nu.Status != common.UserStatusEnabled {
		return nil, ErrNewuserDisabled
	}
	if !common.ValidatePasswordAndHash(password, nu.Password) {
		return nil, ErrNewuserWrongPassword
	}
	return nu, nil
}

func (nu *Newuser) Insert() error {
	return DB.Create(nu).Error
}

func (nu *Newuser) Update() error {
	return DB.Model(nu).Select(
		"display_name", "email", "phone", "status", "token_id", "quota_limit", "last_login_time",
	).Updates(nu).Error
}

func (nu *Newuser) UpdatePassword(hashedPassword string) error {
	return DB.Model(nu).Update("password", hashedPassword).Error
}

// SyncOrgOwnerNewuserPassword updates the org-owner newuser row that mirrors a main User account.
// hashedPassword must already be bcrypt-hashed (same as users.password).
func SyncOrgOwnerNewuserPassword(ownerUserId int, hashedPassword string) error {
	if ownerUserId <= 0 || hashedPassword == "" {
		return nil
	}
	return DB.Model(&Newuser{}).
		Where("owner_user_id = ? AND is_org_owner = ?", ownerUserId, true).
		Update("password", hashedPassword).Error
}

// SyncMainUserPasswordFromOrgOwner updates the main users.password when org-owner newuser changes password.
func SyncMainUserPasswordFromOrgOwner(ownerUserId int, hashedPassword string) error {
	if ownerUserId <= 0 || hashedPassword == "" {
		return nil
	}
	if err := DB.Model(&User{}).Where("id = ?", ownerUserId).Update("password", hashedPassword).Error; err != nil {
		return err
	}
	return InvalidateUserCache(ownerUserId)
}

// GetOrgOwnerNewuserByOwnerUserId returns the is_org_owner newuser for a main account, if any.
func GetOrgOwnerNewuserByOwnerUserId(ownerUserId int) (*Newuser, error) {
	if ownerUserId <= 0 {
		return nil, ErrNewuserNotFound
	}
	nu := &Newuser{}
	err := DB.Where("owner_user_id = ? AND is_org_owner = ?", ownerUserId, true).First(nu).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNewuserNotFound
		}
		return nil, err
	}
	return nu, nil
}

func (nu *Newuser) TouchLastLogin() error {
	nu.LastLoginTime = common.GetTimestamp()
	return DB.Model(nu).Update("last_login_time", nu.LastLoginTime).Error
}

func CreateNewuserToken(ownerUserId int, username string, quotaLimit int) (*Token, error) {
	key, err := common.GenerateKey()
	if err != nil {
		return nil, err
	}
	token := &Token{
		UserId:         ownerUserId,
		Name:           fmt.Sprintf("newuser:%s", username),
		Key:            key,
		CreatedTime:    common.GetTimestamp(),
		AccessedTime:   common.GetTimestamp(),
		ExpiredTime:    -1,
		Status:         common.TokenStatusEnabled,
		UnlimitedQuota: quotaLimit <= 0,
		RemainQuota:    quotaLimit,
	}
	if err := token.Insert(); err != nil {
		return nil, err
	}
	return token, nil
}

func ApplyNewuserQuotaLimit(tokenId int, quotaLimit int) error {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return err
	}
	token.UnlimitedQuota = quotaLimit <= 0
	token.RemainQuota = quotaLimit
	return token.Update()
}

func CreateNewuserAccount(ownerUserId int, username, password, displayName string, quotaLimit int, email, phone string) (*Newuser, *Token, error) {
	if err := ValidateNewuserCredentials(username, password); err != nil {
		return nil, nil, err
	}
	if !IsNewuserOwnerEnabled(ownerUserId) {
		return nil, nil, errors.New("newuser module disabled for this organization")
	}
	owner, err := GetUserById(ownerUserId, false)
	if err != nil {
		return nil, nil, errors.New("organization owner not found")
	}
	if owner.Status != common.UserStatusEnabled {
		return nil, nil, errors.New("organization owner is disabled")
	}
	username = strings.TrimSpace(username)
	if _, err := GetNewuserByUsername(username); err == nil {
		return nil, nil, ErrNewuserDuplicate
	} else if !errors.Is(err, ErrNewuserNotFound) {
		return nil, nil, err
	}
	email, phone, err = NormalizeNewuserContact(email, phone)
	if err != nil {
		return nil, nil, err
	}
	if email != "" && IsNewuserEmailTaken(email, 0) {
		return nil, nil, errors.New("email already exists")
	}
	if phone != "" && IsNewuserPhoneTaken(phone, 0) {
		return nil, nil, errors.New("phone already exists")
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return nil, nil, err
	}
	token, err := CreateNewuserToken(ownerUserId, username, quotaLimit)
	if err != nil {
		return nil, nil, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}
	nu := &Newuser{
		OwnerUserId: ownerUserId,
		Username:    username,
		Password:    hashedPassword,
		DisplayName: displayName,
		Email:       email,
		Phone:       phone,
		Status:      common.UserStatusEnabled,
		TokenId:     token.Id,
		QuotaLimit:  quotaLimit,
		IsOrgOwner:  false,
		CreatedTime: common.GetTimestamp(),
	}
	if err := nu.Insert(); err != nil {
		_ = token.Delete()
		return nil, nil, err
	}
	return nu, token, nil
}

func CreateNewuserOrgWithMainUser(username, password, displayName string, quotaLimit int) (*Newuser, *Token, error) {
	if err := ValidateNewuserCredentials(username, password); err != nil {
		return nil, nil, err
	}
	if !IsNewuserOrgSelfRegisterEnabled() {
		return nil, nil, errors.New("organization self-registration is disabled")
	}
	if !common.RegisterEnabled || !common.PasswordRegisterEnabled {
		return nil, nil, errors.New("user registration is disabled")
	}
	username = strings.TrimSpace(username)
	if len(username) > UserNameMaxLength {
		return nil, nil, fmt.Errorf("username must be at most %d characters for organization registration", UserNameMaxLength)
	}
	exist, err := CheckUserExistOrDeleted(username, "")
	if err != nil {
		return nil, nil, err
	}
	if exist {
		return nil, nil, errors.New("username already exists in main user table")
	}
	if _, err := GetNewuserByUsername(username); err == nil {
		return nil, nil, ErrNewuserDuplicate
	} else if !errors.Is(err, ErrNewuserNotFound) {
		return nil, nil, err
	}
	hashedPassword, err := common.Password2Hash(password)
	if err != nil {
		return nil, nil, err
	}
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		displayName = username
	}
	mainUser := &User{
		Username:    username,
		Password:    password,
		DisplayName: displayName,
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
	}
	if err := mainUser.Insert(0); err != nil {
		return nil, nil, err
	}
	ownerUserId := mainUser.Id
	if ownerUserId == 0 {
		var loaded User
		if err := DB.Where("username = ?", username).First(&loaded).Error; err != nil {
			return nil, nil, err
		}
		ownerUserId = loaded.Id
	}
	if err := SetNewuserOwnerSettings(ownerUserId, true, false, ""); err != nil {
		return nil, nil, err
	}
	token, err := CreateNewuserToken(ownerUserId, username, quotaLimit)
	if err != nil {
		return nil, nil, err
	}
	nu := &Newuser{
		OwnerUserId: ownerUserId,
		Username:    username,
		Password:    hashedPassword,
		DisplayName: displayName,
		Status:      common.UserStatusEnabled,
		TokenId:     token.Id,
		QuotaLimit:  quotaLimit,
		IsOrgOwner:  true,
		CreatedTime: common.GetTimestamp(),
	}
	if err := nu.Insert(); err != nil {
		_ = token.Delete()
		return nil, nil, err
	}
	return nu, token, nil
}

func DeleteNewuserById(id, ownerUserId int) error {
	nu, err := GetNewuserById(id)
	if err != nil {
		return err
	}
	if nu.OwnerUserId != ownerUserId {
		return errors.New("unauthorized")
	}
	if nu.TokenId > 0 {
		token, tokenErr := GetTokenByIds(nu.TokenId, ownerUserId)
		if tokenErr == nil && token != nil {
			token.Status = common.TokenStatusDisabled
			_ = token.Update()
		}
	}
	nu.Status = common.UserStatusDisabled
	return nu.Update()
}

func GetNewuserUsage(tokenId int) (usedQuota int, remainQuota int, unlimited bool, err error) {
	token, err := GetTokenById(tokenId)
	if err != nil {
		return 0, 0, false, err
	}
	return token.UsedQuota, token.RemainQuota, token.UnlimitedQuota, nil
}

func IsNewuserQuotaExceeded(tokenId int, quotaLimit int) bool {
	if quotaLimit <= 0 {
		return false
	}
	used, _, _, err := GetNewuserUsage(tokenId)
	if err != nil {
		return true
	}
	return used >= quotaLimit
}

func GetNewuserOwnerProfile(ownerUserId int) (username, displayName string) {
	brief := GetNewuserOwnerBrief(ownerUserId)
	return brief.Username, brief.DisplayName
}

type NewuserOwnerBrief struct {
	Username    string
	DisplayName string
	Quota       int
	UsedQuota   int
}

func GetNewuserOwnerBrief(ownerUserId int) NewuserOwnerBrief {
	brief := NewuserOwnerBrief{}
	if ownerUserId <= 0 {
		return brief
	}
	owner, err := GetUserById(ownerUserId, false)
	if err != nil || owner == nil {
		return brief
	}
	brief.Username = owner.Username
	brief.DisplayName = strings.TrimSpace(owner.DisplayName)
	if brief.DisplayName == "" {
		brief.DisplayName = owner.Username
	}
	brief.Quota = owner.Quota
	brief.UsedQuota = owner.UsedQuota
	return brief
}

func migrateNewuserUsernameIndex() error {
	if DB == nil || !DB.Migrator().HasTable(&Newuser{}) {
		return nil
	}
	if DB.Migrator().HasIndex(&Newuser{}, "idx_newuser_owner_username") {
		return DB.Migrator().DropIndex(&Newuser{}, "idx_newuser_owner_username")
	}
	return nil
}
