package kpi

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/model"
)

type KPIFetcher struct {
	client *aggregator.SuiteClient
}

func NewFetcher(client *aggregator.SuiteClient) *KPIFetcher {
	return &KPIFetcher{client: client}
}

func (f *KPIFetcher) Fetch(ctx context.Context, tenantID uuid.UUID, kpi *model.KPIDefinition) (float64, time.Duration, error) {
	if kpi == nil {
		return 0, 0, fmt.Errorf("kpi is required")
	}
	endpoint, err := applyQueryParams(kpi.QueryEndpoint, kpi.QueryParams)
	if err != nil {
		return 0, 0, err
	}
	var payload map[string]any
	meta := f.client.Fetch(ctx, string(kpi.Suite), endpoint, tenantID, &payload)
	if meta.Status == "unavailable" {
		return 0, meta.Latency, meta.Error
	}
	value, err := aggregator.ExtractValue(payload, kpi.ValuePath)
	if err != nil {
		return 0, meta.Latency, err
	}
	return value, meta.Latency, nil
}

func applyQueryParams(endpoint string, params map[string]any) (string, error) {
	if len(params) == 0 {
		return endpoint, nil
	}
	base := endpoint
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	u, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	values := u.Query()
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		values.Set(key, stringifyParam(params[key]))
	}
	u.RawQuery = values.Encode()
	return u.String(), nil
}

func stringifyParam(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []string:
		return strings.Join(typed, ",")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, stringifyParam(item))
		}
		return strings.Join(parts, ",")
	default:
		return fmt.Sprint(value)
	}
}
