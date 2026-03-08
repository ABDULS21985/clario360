package servicenow

import intmodel "github.com/clario360/platform/internal/integration/model"

func SanitizeConfig(cfg intmodel.ServiceNowConfig) intmodel.ServiceNowConfig {
	if cfg.Password != "" {
		cfg.Password = "****"
	}
	if cfg.OAuthToken != "" {
		cfg.OAuthToken = "****"
	}
	if cfg.WebhookSecret != "" {
		cfg.WebhookSecret = "****"
	}
	return cfg
}
