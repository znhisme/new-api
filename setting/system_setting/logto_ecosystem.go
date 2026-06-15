package system_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type LogtoEcosystemSettings struct {
	Enabled                    bool     `json:"enabled"`
	Issuer                     string   `json:"issuer"`
	Audience                   string   `json:"audience"`
	JWKSURI                    string   `json:"jwks_uri"`
	RequiredScopes             []string `json:"required_scopes"`
	AllowedApps                []string `json:"allowed_apps"`
	AllowedCapabilities        []string `json:"allowed_capabilities"`
	DefaultTokenExpiredTime    int64    `json:"default_token_expired_time"`
	DefaultTokenUnlimitedQuota bool     `json:"default_token_unlimited_quota"`
	BaseURL                    string   `json:"base_url"`
}

var logtoEcosystemSettings = LogtoEcosystemSettings{
	Enabled:                    common.GetEnvOrDefaultBool("LOGTO_ECOSYSTEM_ENABLED", false),
	Issuer:                     common.GetEnvOrDefaultString("LOGTO_ISSUER", ""),
	Audience:                   common.GetEnvOrDefaultString("LOGTO_AUDIENCE", ""),
	JWKSURI:                    firstNonEmpty(common.GetEnvOrDefaultString("LOGTO_JWKS_URI_INTERNAL", ""), common.GetEnvOrDefaultString("LOGTO_JWKS_URI", "")),
	RequiredScopes:             splitLogtoEcosystemList(common.GetEnvOrDefaultString("LOGTO_REQUIRED_SCOPES", "")),
	AllowedApps:                splitLogtoEcosystemList(common.GetEnvOrDefaultString("ECOSYSTEM_ALLOWED_APPS", "")),
	AllowedCapabilities:        defaultLogtoCapabilities(common.GetEnvOrDefaultString("ECOSYSTEM_ALLOWED_CAPABILITIES", "")),
	DefaultTokenExpiredTime:    int64(common.GetEnvOrDefault("ECOSYSTEM_DEFAULT_TOKEN_EXPIRED_TIME", -1)),
	DefaultTokenUnlimitedQuota: common.GetEnvOrDefaultBool("ECOSYSTEM_DEFAULT_TOKEN_UNLIMITED_QUOTA", true),
	BaseURL:                    common.GetEnvOrDefaultString("ECOSYSTEM_BASE_URL", ""),
}

func init() {
	config.GlobalConfig.Register("logto_ecosystem", &logtoEcosystemSettings)
}

func GetLogtoEcosystemSettings() *LogtoEcosystemSettings {
	return &logtoEcosystemSettings
}

func splitLogtoEcosystemList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' '
	})
	result := make([]string, 0, len(parts))
	seen := make(map[string]bool)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || seen[part] {
			continue
		}
		seen[part] = true
		result = append(result, part)
	}
	return result
}

func defaultLogtoCapabilities(value string) []string {
	capabilities := splitLogtoEcosystemList(value)
	if len(capabilities) > 0 {
		return capabilities
	}
	return []string{"default", "text", "image", "audio", "video"}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
