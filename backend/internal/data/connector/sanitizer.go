package connector

import (
	"encoding/json"
	"strings"

	"github.com/clario360/platform/internal/data/model"
)

// SecretKeys is the set of connection-config field names whose values are
// considered sensitive. Exported so the service layer can use it for
// credential-preservation during updates.
var SecretKeys = map[string]struct{}{
	"password":      {},
	"secret_key":    {},
	"api_key":       {},
	"token":         {},
	"auth_config":   {},
	"client_secret": {},
	"access_key":    {},
	"refresh_token": {},
	"keytab":        {},
	"ca_cert_path":  {},
}

func SanitizeConnectionConfig(_ model.DataSourceType, config json.RawMessage) json.RawMessage {
	if len(config) == 0 {
		return json.RawMessage(`{}`)
	}

	var payload any
	if err := json.Unmarshal(config, &payload); err != nil {
		return json.RawMessage(`{}`)
	}
	sanitized := sanitizeValue(payload)
	bytes, err := json.Marshal(sanitized)
	if err != nil {
		return json.RawMessage(`{}`)
	}
	return bytes
}

func sanitizeValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, inner := range typed {
			if _, blocked := SecretKeys[strings.ToLower(key)]; blocked {
				continue
			}
			out[key] = sanitizeValue(inner)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, inner := range typed {
			out = append(out, sanitizeValue(inner))
		}
		return out
	default:
		return value
	}
}
