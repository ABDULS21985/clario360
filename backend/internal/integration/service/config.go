package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/events"
	intmodel "github.com/clario360/platform/internal/integration/model"
)

func NormalizeAndValidateConfig(typ intmodel.IntegrationType, config map[string]any) (map[string]any, error) {
	if config == nil {
		config = map[string]any{}
	}

	switch typ {
	case intmodel.IntegrationTypeWebhook:
		if strings.TrimSpace(stringValue(config["url"])) == "" {
			return nil, fmt.Errorf("webhook url is required")
		}
		if strings.TrimSpace(stringValue(config["method"])) == "" {
			config["method"] = "POST"
		}
		if strings.TrimSpace(stringValue(config["content_type"])) == "" {
			config["content_type"] = "application/json"
		}
		if _, ok := config["headers"]; !ok {
			config["headers"] = map[string]string{}
		}
	case intmodel.IntegrationTypeSlack:
		if strings.TrimSpace(stringValue(config["bot_token"])) == "" {
			return nil, fmt.Errorf("slack bot_token is required")
		}
		if strings.TrimSpace(stringValue(config["channel_id"])) == "" && strings.TrimSpace(stringValue(config["incoming_webhook_url"])) == "" {
			return nil, fmt.Errorf("slack channel_id or incoming_webhook_url is required")
		}
	case intmodel.IntegrationTypeTeams:
		required := []string{"bot_app_id", "bot_password", "service_url", "conversation_id"}
		for _, field := range required {
			if strings.TrimSpace(stringValue(config[field])) == "" {
				return nil, fmt.Errorf("teams %s is required", field)
			}
		}
	case intmodel.IntegrationTypeJira:
		required := []string{"base_url", "project_key"}
		for _, field := range required {
			if strings.TrimSpace(stringValue(config[field])) == "" {
				return nil, fmt.Errorf("jira %s is required", field)
			}
		}
		if strings.TrimSpace(stringValue(config["auth_token"])) == "" && strings.TrimSpace(stringValue(config["refresh_token"])) == "" {
			return nil, fmt.Errorf("jira auth_token or refresh_token is required")
		}
	case intmodel.IntegrationTypeServiceNow:
		if strings.TrimSpace(stringValue(config["instance_url"])) == "" {
			return nil, fmt.Errorf("servicenow instance_url is required")
		}
		authType := strings.TrimSpace(stringValue(config["auth_type"]))
		if authType == "" {
			authType = "basic"
			config["auth_type"] = authType
		}
		switch authType {
		case "basic":
			if strings.TrimSpace(stringValue(config["username"])) == "" || strings.TrimSpace(stringValue(config["password"])) == "" {
				return nil, fmt.Errorf("servicenow username and password are required for basic auth")
			}
		case "oauth":
			if strings.TrimSpace(stringValue(config["oauth_token"])) == "" {
				return nil, fmt.Errorf("servicenow oauth_token is required for oauth auth")
			}
		default:
			return nil, fmt.Errorf("servicenow auth_type must be basic or oauth")
		}
	default:
		return nil, fmt.Errorf("unsupported integration type %q", typ)
	}

	return config, nil
}

func SanitizeConfig(config map[string]any) map[string]any {
	if config == nil {
		return map[string]any{}
	}
	result := make(map[string]any, len(config))
	for key, value := range config {
		lower := strings.ToLower(key)
		switch typed := value.(type) {
		case map[string]any:
			result[key] = SanitizeConfig(typed)
		case map[string]string:
			nested := make(map[string]any, len(typed))
			for nestedKey, nestedValue := range typed {
				nested[nestedKey] = nestedValue
			}
			result[key] = SanitizeConfig(nested)
		case []any:
			copied := make([]any, len(typed))
			for idx, item := range typed {
				if nested, ok := item.(map[string]any); ok {
					copied[idx] = SanitizeConfig(nested)
				} else {
					copied[idx] = item
				}
			}
			result[key] = copied
		case string:
			if isSecretLikeKey(lower) {
				result[key] = maskSecret(typed)
			} else {
				result[key] = typed
			}
		default:
			result[key] = value
		}
	}
	return result
}

func DecodeInto[T any](config map[string]any, target *T) error {
	payload, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}
	return nil
}

func MatchesEventFilters(event *events.Event, filters []intmodel.EventFilter) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if matchesFilter(event, filter) {
			return true
		}
	}
	return false
}

func matchesFilter(event *events.Event, filter intmodel.EventFilter) bool {
	if len(filter.EventTypes) > 0 && !contains(filter.EventTypes, event.Type) && !contains(filter.EventTypes, trimEventType(event.Type)) {
		return false
	}

	suite := extractSuite(event.Type)
	if len(filter.Suites) > 0 && !contains(filter.Suites, suite) {
		return false
	}

	var data map[string]any
	if len(event.Data) > 0 {
		_ = json.Unmarshal(event.Data, &data)
	}

	if len(filter.Severities) > 0 {
		severity := extractString(data, "severity")
		if severity != "" && !contains(filter.Severities, severity) {
			return false
		}
	}

	if filter.MinConfidence > 0 {
		confidence := extractConfidence(data)
		if confidence > 0 && confidence < filter.MinConfidence {
			return false
		}
	}

	return true
}

func extractSuite(eventType string) string {
	trimmed := trimEventType(eventType)
	parts := strings.Split(trimmed, ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func trimEventType(eventType string) string {
	return strings.TrimPrefix(eventType, "com.clario360.")
}

func extractConfidence(data map[string]any) float64 {
	for _, key := range []string{"confidence", "confidence_score", "confidenceScore"} {
		if value, ok := data[key]; ok {
			switch typed := value.(type) {
			case float64:
				return typed
			case int:
				return float64(typed)
			}
		}
	}
	return 0
}

func extractString(data map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := data[key]; ok {
			if str, ok := value.(string); ok {
				return str
			}
		}
	}
	return ""
}

func contains(items []string, needle string) bool {
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), strings.TrimSpace(needle)) {
			return true
		}
	}
	return false
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func isSecretLikeKey(key string) bool {
	for _, fragment := range []string{"token", "secret", "password", "key", "authorization"} {
		if strings.Contains(key, fragment) {
			return true
		}
	}
	return false
}

func maskSecret(secret string) string {
	if secret == "" {
		return ""
	}
	if len(secret) <= 8 {
		return strings.Repeat("*", len(secret))
	}
	return secret[:4] + strings.Repeat("*", len(secret)-8) + secret[len(secret)-4:]
}
