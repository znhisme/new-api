package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"gorm.io/gorm"
)

var ErrEcosystemAppNotAllowed = errors.New("app_not_allowed")
var ErrEcosystemCapabilityNotAllowed = errors.New("capability_not_allowed")
var ErrEcosystemGroupNotAllowed = errors.New("group_not_allowed")
var ErrEcosystemUserDisabled = errors.New("user_disabled")
var ErrEcosystemInvalidToken = errors.New("invalid_ecosystem_token")

type EcosystemUserInfo struct {
	LogtoSub     string `json:"logto_sub"`
	NewAPIUserID int    `json:"newapi_user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Group        string `json:"group"`
	Provisioned  bool   `json:"provisioned"`
}

type EcosystemTokenUpsertRequest struct {
	AppID      string `json:"app_id"`
	Capability string `json:"capability"`
	Group      string `json:"group"`
}

type EcosystemTokenUpsertResponse struct {
	TokenID    int    `json:"token_id"`
	TokenName  string `json:"token_name"`
	APIKey     string `json:"api_key,omitempty"`
	BaseURL    string `json:"base_url"`
	Group      string `json:"group"`
	Capability string `json:"capability"`
	Created    bool   `json:"created"`
}

func ResolveOrProvisionUser(logtoSub string, email string, profileName string, profileUsername string) (*model.User, bool, error) {
	logtoSub = strings.TrimSpace(logtoSub)
	if logtoSub == "" {
		return nil, false, errors.New("logto sub 为空！")
	}

	user := &model.User{}
	if err := model.DB.Where("oidc_id = ?", logtoSub).First(user).Error; err == nil {
		if user.Status != common.UserStatusEnabled {
			return nil, false, ErrEcosystemUserDisabled
		}
		return user, false, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, fmt.Errorf("resolve user by oidc id failed: %w", err)
	}

	if !common.RegisterEnabled {
		return nil, false, errors.New("管理员关闭了新用户注册")
	}

	username := buildEcosystemUsername(logtoSub)
	displayName := strings.TrimSpace(profileName)
	if displayName == "" {
		displayName = truncateEcosystemUsername(firstNonEmpty(strings.TrimSpace(email), username))
	}
	displayName = truncateEcosystemUsername(displayName)

	newUser := &model.User{
		Username:    username,
		DisplayName: displayName,
		Email:       strings.TrimSpace(email),
		Role:        common.RoleCommonUser,
		Status:      common.UserStatusEnabled,
		Group:       "default",
		OidcId:      logtoSub,
	}

	if err := newUser.Insert(0); err != nil {
		return nil, false, err
	}
	if err := model.DB.Where("oidc_id = ?", logtoSub).First(newUser).Error; err != nil {
		return nil, true, err
	}
	return newUser, true, nil
}

func GetEcosystemUserGroups(user *model.User) map[string]map[string]interface{} {
	if user == nil {
		return map[string]map[string]interface{}{}
	}
	usableGroups := make(map[string]map[string]interface{})
	userUsableGroups := GetUserUsableGroups(user.Group)
	for groupName := range ratio_setting.GetGroupRatioCopy() {
		if desc, ok := userUsableGroups[groupName]; ok {
			usableGroups[groupName] = map[string]interface{}{
				"ratio": GetUserGroupRatio(user.Group, groupName),
				"desc":  desc,
			}
		}
	}
	if _, ok := userUsableGroups["auto"]; ok {
		usableGroups["auto"] = map[string]interface{}{
			"ratio": "自动",
			"desc":  setting.GetUsableGroupDescription("auto"),
		}
	}
	return usableGroups
}

func GetEcosystemUserModels(user *model.User) []string {
	if user == nil {
		return []string{}
	}
	groups := GetUserUsableGroups(user.Group)
	models := make([]string, 0)
	seen := make(map[string]bool)
	for group := range groups {
		for _, enabledModel := range model.GetGroupEnabledModels(group) {
			if seen[enabledModel] {
				continue
			}
			seen[enabledModel] = true
			models = append(models, enabledModel)
		}
	}
	return models
}

func UpsertEcosystemToken(user *model.User, req EcosystemTokenUpsertRequest) (*model.Token, string, bool, error) {
	if user == nil {
		return nil, "", false, errors.New("user 为空！")
	}
	if user.Status != common.UserStatusEnabled {
		return nil, "", false, ErrEcosystemUserDisabled
	}
	req.AppID = strings.TrimSpace(req.AppID)
	req.Capability = strings.TrimSpace(req.Capability)
	if req.Capability == "" {
		req.Capability = "default"
	}
	req.Group = strings.TrimSpace(req.Group)

	if err := validateEcosystemAppAndCapability(req.AppID, req.Capability); err != nil {
		return nil, "", false, err
	}

	resolvedGroup := req.Group
	if resolvedGroup == "" {
		resolvedGroup = user.Group
	}
	if !GroupInUserUsableGroups(user.Group, resolvedGroup) {
		return nil, "", false, ErrEcosystemGroupNotAllowed
	}

	name := ecosystemTokenName(req.AppID, req.Capability)
	existing, err := findExistingEcosystemToken(user.Id, name)
	if err != nil {
		return nil, "", false, err
	}

	if existing != nil {
		if existing.Status != common.TokenStatusEnabled {
			return nil, "", false, ErrEcosystemInvalidToken
		}
		existing.Group = resolvedGroup
		existing.ExpiredTime = system_setting.GetLogtoEcosystemSettings().DefaultTokenExpiredTime
		existing.UnlimitedQuota = system_setting.GetLogtoEcosystemSettings().DefaultTokenUnlimitedQuota
		if err := existing.Update(); err != nil {
			return nil, "", false, err
		}
		apiKey := formatEcosystemAPIKey(existing.GetFullKey())
		return existing, apiKey, false, nil
	}

	count, err := model.CountUserTokens(user.Id)
	if err != nil {
		return nil, "", false, err
	}
	maxTokens := operation_setting.GetMaxUserTokens()
	if maxTokens > 0 && int(count) >= maxTokens {
		return nil, "", false, errors.New("已达到最大令牌数量限制")
	}

	key, err := common.GenerateKey()
	if err != nil {
		return nil, "", false, err
	}
	token := &model.Token{
		UserId:             user.Id,
		Name:               name,
		Key:                key,
		Status:             common.TokenStatusEnabled,
		CreatedTime:        common.GetTimestamp(),
		AccessedTime:       common.GetTimestamp(),
		ExpiredTime:        system_setting.GetLogtoEcosystemSettings().DefaultTokenExpiredTime,
		RemainQuota:        0,
		UnlimitedQuota:     system_setting.GetLogtoEcosystemSettings().DefaultTokenUnlimitedQuota,
		ModelLimitsEnabled: false,
		Group:              resolvedGroup,
	}
	if err := token.Insert(); err != nil {
		return nil, "", false, err
	}
	apiKey := formatEcosystemAPIKey(token.GetFullKey())
	return token, apiKey, true, nil
}

func validateEcosystemAppAndCapability(appID string, capability string) error {
	settings := system_setting.GetLogtoEcosystemSettings()
	if !stringInSlice(strings.TrimSpace(appID), settings.AllowedApps) {
		return ErrEcosystemAppNotAllowed
	}
	if !stringInSlice(strings.TrimSpace(capability), settings.AllowedCapabilities) {
		return ErrEcosystemCapabilityNotAllowed
	}
	return nil
}

func findExistingEcosystemToken(userID int, name string) (*model.Token, error) {
	var token model.Token
	err := model.DB.Where("user_id = ? AND name = ?", userID, name).First(&token).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func ecosystemTokenName(appID string, capability string) string {
	return fmt.Sprintf("ecosystem:%s:%s", strings.TrimSpace(appID), strings.TrimSpace(capability))
}

func buildEcosystemUsername(logtoSub string) string {
	sum := sha256.Sum256([]byte(logtoSub))
	base := "eco_" + hex.EncodeToString(sum[:])[:12]
	if exists, err := model.CheckUserExistOrDeleted(base, ""); err == nil && !exists {
		return base
	}
	for i := 0; i < 10; i++ {
		username := "eco_" + hex.EncodeToString(sum[:])[:8] + "_" + common.GetRandomString(4)
		if exists, err := model.CheckUserExistOrDeleted(username, ""); err == nil && !exists {
			return username
		}
	}
	return "eco_" + common.GetRandomString(16)
}

func truncateEcosystemUsername(username string) string {
	username = strings.TrimSpace(username)
	if len(username) <= model.UserNameMaxLength {
		return username
	}
	return username[:model.UserNameMaxLength]
}

func stringInSlice(value string, list []string) bool {
	value = strings.TrimSpace(value)
	for _, item := range list {
		if strings.TrimSpace(item) == value {
			return true
		}
	}
	return false
}

func GetLogtoEcosystemBaseURL() string {
	baseURL := strings.TrimSpace(system_setting.GetLogtoEcosystemSettings().BaseURL)
	if baseURL == "" {
		baseURL = strings.TrimSpace(system_setting.ServerAddress)
	}
	if baseURL == "" {
		return "/v1"
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL
	}
	return baseURL + "/v1"
}

func formatEcosystemAPIKey(key string) string {
	key = strings.TrimSpace(key)
	if key == "" || strings.HasPrefix(key, "sk-") {
		return key
	}
	return "sk-" + key
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
