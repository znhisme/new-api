package service

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestResolveOrProvisionUserDoesNotMergeByEmail(t *testing.T) {
	setupEcosystemServiceTestDB(t)
	restore := configureEcosystemServiceSettings(t)
	defer restore()

	require.NoError(t, model.DB.Create(&model.User{
		Username: "existing",
		Email:    "same@example.com",
		AffCode:  "aff-existing",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	user, created, err := ResolveOrProvisionUser("logto-sub-1", "same@example.com", "", "")
	require.NoError(t, err)
	require.True(t, created)
	require.NotEqual(t, "existing", user.Username)
	require.Equal(t, "logto-sub-1", user.OidcId)
	require.Equal(t, "same@example.com", user.Email)
}

func TestResolveOrProvisionUserExistingAndDisabled(t *testing.T) {
	setupEcosystemServiceTestDB(t)
	restore := configureEcosystemServiceSettings(t)
	defer restore()

	require.NoError(t, model.DB.Create(&model.User{
		Username: "bound",
		OidcId:   "logto-bound",
		AffCode:  "aff-bound",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}).Error)

	user, created, err := ResolveOrProvisionUser("logto-bound", "bound@example.com", "", "")
	require.NoError(t, err)
	require.False(t, created)
	require.Equal(t, "bound", user.Username)

	require.NoError(t, model.DB.Create(&model.User{
		Username: "disabled",
		OidcId:   "logto-disabled",
		AffCode:  "aff-disabled",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusDisabled,
		Group:    "default",
	}).Error)

	_, _, err = ResolveOrProvisionUser("logto-disabled", "disabled@example.com", "", "")
	require.ErrorIs(t, err, ErrEcosystemUserDisabled)
}

func TestUpsertEcosystemTokenIsIdempotent(t *testing.T) {
	setupEcosystemServiceTestDB(t)
	restore := configureEcosystemServiceSettings(t)
	defer restore()

	user := &model.User{
		Username: "token-user",
		AffCode:  "aff-token",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, model.DB.Create(user).Error)

	req := EcosystemTokenUpsertRequest{
		AppID:      "canvas",
		Capability: "image",
		Group:      "default",
	}
	token, key, created, err := UpsertEcosystemToken(user, req)
	require.NoError(t, err)
	require.True(t, created)
	require.Equal(t, "ecosystem:canvas:image", token.Name)
	require.Equal(t, "default", token.Group)
	require.True(t, strings.HasPrefix(key, "sk-"))

	secondToken, secondKey, secondCreated, err := UpsertEcosystemToken(user, req)
	require.NoError(t, err)
	require.False(t, secondCreated)
	require.Equal(t, token.Id, secondToken.Id)
	require.Equal(t, key, secondKey)
}

func TestUpsertEcosystemTokenValidatesAppCapabilityAndGroup(t *testing.T) {
	setupEcosystemServiceTestDB(t)
	restore := configureEcosystemServiceSettings(t)
	defer restore()

	user := &model.User{
		Username: "validate-user",
		AffCode:  "aff-validate",
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, model.DB.Create(user).Error)

	_, _, _, err := UpsertEcosystemToken(user, EcosystemTokenUpsertRequest{AppID: "unknown", Capability: "image", Group: "default"})
	require.ErrorIs(t, err, ErrEcosystemAppNotAllowed)

	_, _, _, err = UpsertEcosystemToken(user, EcosystemTokenUpsertRequest{AppID: "canvas", Capability: "video", Group: "default"})
	require.ErrorIs(t, err, ErrEcosystemCapabilityNotAllowed)

	_, _, _, err = UpsertEcosystemToken(user, EcosystemTokenUpsertRequest{AppID: "canvas", Capability: "image", Group: "not-a-real-group"})
	require.ErrorIs(t, err, ErrEcosystemGroupNotAllowed)
}

func setupEcosystemServiceTestDB(t *testing.T) {
	t.Helper()
	oldDB := model.DB
	oldLogDB := model.LOG_DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Log{}))
	t.Cleanup(func() {
		model.DB = oldDB
		model.LOG_DB = oldLogDB
	})
}

func configureEcosystemServiceSettings(t *testing.T) func() {
	t.Helper()
	settings := system_setting.GetLogtoEcosystemSettings()
	originalSettings := *settings
	originalRegisterEnabled := common.RegisterEnabled
	originalMaxUserTokens := operation_setting.GetTokenSetting().MaxUserTokens
	*settings = system_setting.LogtoEcosystemSettings{
		Enabled:                    true,
		Issuer:                     "http://logto.example/oidc",
		Audience:                   "https://newapi.example",
		JWKSURI:                    "http://logto.example/oidc/jwks",
		AllowedApps:                []string{"canvas"},
		AllowedCapabilities:        []string{"default", "image"},
		DefaultTokenExpiredTime:    -1,
		DefaultTokenUnlimitedQuota: true,
		BaseURL:                    "https://newapi.example",
	}
	common.RegisterEnabled = true
	operation_setting.GetTokenSetting().MaxUserTokens = 1000
	return func() {
		*settings = originalSettings
		common.RegisterEnabled = originalRegisterEnabled
		operation_setting.GetTokenSetting().MaxUserTokens = originalMaxUserTokens
	}
}
