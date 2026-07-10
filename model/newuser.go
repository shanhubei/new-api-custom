package model

import (
	"errors"
	"fmt"
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

// Newuser 第三方 UI 用户，映射到 OwnerUserId（原有 new-api 用户，即“组织”）下的一个 Token。
// Username 全站唯一，登录时仅需 username + password 即可定位所属组织。
type Newuser struct {
	Id            int            `json:"id"`
	OwnerUserId   int            `json:"owner_user_id" gorm:"index"`
	Username      string         `json:"username" gorm:"size:64;uniqueIndex"`
	Password      string         `json:"-"`
	DisplayName   string         `json:"display_name" gorm:"size:64"`
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

func NormalizeNewuserRegisterType(registerType string, ownerUserId int) string {
	registerType = strings.ToLower(strings.TrimSpace(registerType))
	switch registerType {
	case NewuserRegisterTypeOrg, NewuserRegisterTypeMember:
		return registerType
	}
	if ownerUserId > 0 {
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

func SetNewuserOwnerSettings(ownerUserId int, enabled, registerEnabled bool, registerCode string) error {
	values := map[string]string{
		newuserOwnerOptionKey(ownerUserId, "enabled"):           boolToOption(enabled),
		newuserOwnerOptionKey(ownerUserId, "register_enabled"):  boolToOption(registerEnabled),
		newuserOwnerOptionKey(ownerUserId, "register_code"):     strings.TrimSpace(registerCode),
	}
	return UpdateOptionsBulk(values)
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
		"display_name", "status", "token_id", "quota_limit", "last_login_time",
	).Updates(nu).Error
}

func (nu *Newuser) UpdatePassword(hashedPassword string) error {
	return DB.Model(nu).Update("password", hashedPassword).Error
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

func CreateNewuserAccount(ownerUserId int, username, password, displayName string, quotaLimit int) (*Newuser, *Token, error) {
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
