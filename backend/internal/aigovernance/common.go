package aigovernance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/google/uuid"

	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
	"github.com/clario360/platform/internal/config"
)

type ModelOutput struct {
	Output     any            `json:"output"`
	Confidence float64        `json:"confidence"`
	Metadata   map[string]any `json:"metadata"`
}

type PredictParams struct {
	TenantID        uuid.UUID
	ModelSlug       string
	UseCase         string
	EntityType      string
	EntityID        *uuid.UUID
	Input           any
	InputSummary    map[string]any
	ModelFunc       func(ctx context.Context, input any) (*ModelOutput, error)
	ShadowModelFunc func(ctx context.Context, input any) (*ModelOutput, error)
}

type PredictionResult struct {
	Output      any                     `json:"output"`
	Confidence  float64                 `json:"confidence"`
	Explanation *aigovmodel.Explanation `json:"explanation"`
	ModelID     uuid.UUID               `json:"model_id"`
	VersionID   uuid.UUID               `json:"version_id"`
	PredictionLogID uuid.UUID           `json:"prediction_log_id"`
	LatencyMS   int                     `json:"latency_ms"`
}

func HashJSON(value any) (string, error) {
	normalized, err := normalizeJSON(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(normalized)
	return hex.EncodeToString(sum[:]), nil
}

func MustJSON(value any) json.RawMessage {
	payload, _ := json.Marshal(value)
	return payload
}

func BuildPlatformCoreDSN(cfg config.DatabaseConfig) string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:   "platform_core",
	}
	q := u.Query()
	q.Set("sslmode", cfg.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func SummarizeInput(value any) map[string]any {
	switch typed := value.(type) {
	case nil:
		return map[string]any{}
	case map[string]any:
		return sanitizeMap(typed, 0)
	case string:
		return map[string]any{"text": truncate(typed, 240)}
	default:
		payload, err := json.Marshal(typed)
		if err != nil {
			return map[string]any{"type": reflect.TypeOf(value).String()}
		}
		var decoded any
		if err := json.Unmarshal(payload, &decoded); err != nil {
			return map[string]any{"json": truncate(string(payload), 240)}
		}
		if data, ok := decoded.(map[string]any); ok {
			return sanitizeMap(data, 0)
		}
		if data, ok := decoded.([]any); ok {
			return map[string]any{"items": len(data), "sample": sanitizeSlice(data, 0)}
		}
		return map[string]any{"value": decoded}
	}
}

func normalizeJSON(value any) ([]byte, error) {
	payload, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var decoded any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return nil, err
	}
	normalized := canonicalize(decoded)
	return json.Marshal(normalized)
}

func canonicalize(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		out := make(map[string]any, len(typed))
		for _, key := range keys {
			out[key] = canonicalize(typed[key])
		}
		return out
	case []any:
		out := make([]any, len(typed))
		for idx := range typed {
			out[idx] = canonicalize(typed[idx])
		}
		return out
	default:
		return typed
	}
}

func sanitizeMap(value map[string]any, depth int) map[string]any {
	if depth >= 2 {
		return map[string]any{"keys": len(value)}
	}
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make(map[string]any, min(len(keys), 12))
	for idx, key := range keys {
		if idx >= 12 {
			out["_truncated"] = true
			break
		}
		lower := strings.ToLower(key)
		switch {
		case strings.Contains(lower, "password"),
			strings.Contains(lower, "token"),
			strings.Contains(lower, "secret"),
			strings.Contains(lower, "key"),
			strings.Contains(lower, "content"),
			strings.Contains(lower, "text"):
			out[key] = "[redacted]"
		default:
			out[key] = sanitizeValue(value[key], depth+1)
		}
	}
	return out
}

func sanitizeSlice(value []any, depth int) []any {
	out := make([]any, 0, min(len(value), 5))
	for idx, item := range value {
		if idx >= 5 {
			out = append(out, "[truncated]")
			break
		}
		out = append(out, sanitizeValue(item, depth+1))
	}
	return out
}

func sanitizeValue(value any, depth int) any {
	switch typed := value.(type) {
	case string:
		return truncate(typed, 120)
	case map[string]any:
		return sanitizeMap(typed, depth)
	case []any:
		return sanitizeSlice(typed, depth)
	default:
		return typed
	}
}

func truncate(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max] + "..."
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
