package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/visus/aggregator"
	"github.com/clario360/platform/internal/visus/model"
	"github.com/clario360/platform/internal/visus/repository"
)

func (s *WidgetService) resolveKPICard(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	kpiID, err := configUUID(widget.Config, "kpi_id")
	if err != nil {
		return nil, err
	}
	definition, err := s.kpis.Get(ctx, tenantID, kpiID)
	if err != nil {
		return nil, err
	}
	latest, _ := s.kpiSnapshots.LatestByKPI(ctx, tenantID, kpiID)
	history, _ := s.kpiSnapshots.ListByKPI(ctx, tenantID, kpiID, model.KPIQuery{Limit: 7})
	trend := make([]map[string]any, 0, len(history))
	for i := len(history) - 1; i >= 0; i-- {
		trend = append(trend, map[string]any{
			"at":    history[i].CreatedAt,
			"value": history[i].Value,
		})
	}
	if latest == nil {
		return map[string]any{
			"value":  0,
			"status": model.KPIStatusUnknown,
			"trend":  trend,
			"unit":   definition.Unit,
		}, nil
	}
	return map[string]any{
		"value":         latest.Value,
		"status":        latest.Status,
		"trend":         trend,
		"target":        definition.TargetValue,
		"unit":          definition.Unit,
		"delta":         latest.Delta,
		"delta_percent": latest.DeltaPercent,
	}, nil
}

func (s *WidgetService) resolveGauge(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	kpiID, err := configUUID(widget.Config, "kpi_id")
	if err != nil {
		return nil, err
	}
	definition, err := s.kpis.Get(ctx, tenantID, kpiID)
	if err != nil {
		return nil, err
	}
	latest, err := s.kpiSnapshots.LatestByKPI(ctx, tenantID, kpiID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"value": latest.Value,
		"min":   0,
		"max":   100,
		"thresholds": map[string]any{
			"warning":  definition.WarningThreshold,
			"critical": definition.CriticalThreshold,
		},
		"status": latest.Status,
	}, nil
}

func (s *WidgetService) resolveSparkline(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	kpiID, err := configUUID(widget.Config, "kpi_id")
	if err != nil {
		return nil, err
	}
	points := int(numberFromConfig(widget.Config, "points", 30))
	history, err := s.kpiSnapshots.ListByKPI(ctx, tenantID, kpiID, model.KPIQuery{Limit: points})
	if err != nil {
		return nil, err
	}
	values := make([]float64, 0, len(history))
	minValue := 0.0
	maxValue := 0.0
	for i := len(history) - 1; i >= 0; i-- {
		value := history[i].Value
		values = append(values, value)
		if len(values) == 1 || value < minValue {
			minValue = value
		}
		if len(values) == 1 || value > maxValue {
			maxValue = value
		}
	}
	current := 0.0
	if len(values) > 0 {
		current = values[len(values)-1]
	}
	direction := "flat"
	if len(values) >= 2 {
		switch {
		case values[len(values)-1] > values[len(values)-2]:
			direction = "up"
		case values[len(values)-1] < values[len(values)-2]:
			direction = "down"
		}
	}
	return map[string]any{
		"values":          values,
		"min":             minValue,
		"max":             maxValue,
		"current":         current,
		"trend_direction": direction,
	}, nil
}

func (s *WidgetService) resolveTrendIndicator(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	kpiID, err := configUUID(widget.Config, "kpi_id")
	if err != nil {
		return nil, err
	}
	history, err := s.kpiSnapshots.ListByKPI(ctx, tenantID, kpiID, model.KPIQuery{Limit: 3})
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return map[string]any{"value": 0, "direction": "flat", "change_percent": 0, "periods": 0}, nil
	}
	current := history[0].Value
	change := 0.0
	direction := "flat"
	if len(history) > 1 && history[1].Value != 0 {
		change = ((current - history[1].Value) / history[1].Value) * 100
		switch {
		case current > history[1].Value:
			direction = "up"
		case current < history[1].Value:
			direction = "down"
		}
	}
	periods := make([]model.KPISnapshot, len(history))
	copy(periods, history)
	return map[string]any{
		"value":          current,
		"direction":      direction,
		"change_percent": change,
		"periods":        periods,
	}, nil
}

func (s *WidgetService) resolveAlertFeed(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	maxAlerts := int(numberFromConfig(widget.Config, "max_alerts", 10))
	severityFilter := stringSliceFromConfig(widget.Config, "severity_filter")
	sourceSuites := stringSliceFromConfig(widget.Config, "alert_sources")
	alerts, _, err := s.alerts.List(ctx, tenantID, repository.AlertListFilters{
		Severity:     severityFilter,
		SourceSuites: sourceSuites,
		Status:       []string{"new", "viewed", "acknowledged", "escalated"},
	}, 1, maxAlerts)
	if err != nil {
		return nil, err
	}
	return map[string]any{"alerts": alerts}, nil
}

func (s *WidgetService) resolveStatusGrid(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	items, _ := widget.Config["items"].([]interface{})
	resolved := make([]map[string]any, 0, len(items))
	for _, raw := range items {
		mapped, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		label := stringValue(mapped["label"])
		if rawID := stringValue(mapped["kpi_id"]); rawID != "" {
			kpiID, err := uuid.Parse(rawID)
			if err != nil {
				return nil, err
			}
			definition, err := s.kpis.Get(ctx, tenantID, kpiID)
			if err != nil {
				return nil, err
			}
			latest, _ := s.kpiSnapshots.LatestByKPI(ctx, tenantID, kpiID)
			status := model.KPIStatusUnknown
			value := 0.0
			if latest != nil {
				status = latest.Status
				value = latest.Value
			}
			resolved = append(resolved, map[string]any{"label": label, "status": status, "value": value, "unit": definition.Unit})
			continue
		}
		suite := stringValue(mapped["suite"])
		source := stringValue(mapped["data_source"])
		valuePath := stringValue(mapped["value_path"])
		payload, err := s.fetchWidgetSource(ctx, tenantID, suite, source)
		if err != nil {
			return nil, err
		}
		value := mustValue(payload, valuePath)
		resolved = append(resolved, map[string]any{"label": label, "status": statusFromValue(value), "value": value})
	}
	return map[string]any{"items": resolved}, nil
}

func (s *WidgetService) resolveSeriesWidget(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	suite := stringFromConfig(widget.Config, "suite")
	source := stringFromConfig(widget.Config, "data_source")
	xAxis := stringFromConfig(widget.Config, "x_axis")
	yAxis := stringSliceFromConfig(widget.Config, "y_axis")
	dataPath := stringFromConfig(widget.Config, "data_path")
	if dataPath == "" {
		dataPath = "$.data"
	}
	payload, err := s.fetchWidgetSource(ctx, tenantID, suite, source)
	if err != nil {
		return nil, err
	}
	rows, err := extractArray(payload, dataPath)
	if err != nil {
		return nil, err
	}
	if widget.Type == model.WidgetTypeBarChart {
		categories := make([]interface{}, 0, len(rows))
		series := make([]map[string]any, 0, len(yAxis))
		for _, key := range yAxis {
			series = append(series, map[string]any{"name": key, "data": make([]float64, 0, len(rows))})
		}
		for _, row := range rows {
			categories = append(categories, row[xAxis])
			for idx, key := range yAxis {
				series[idx]["data"] = append(series[idx]["data"].([]float64), mustValue(row, key))
			}
		}
		return map[string]any{"categories": categories, "series": series}, nil
	}
	series := make([]map[string]any, 0, len(yAxis))
	for _, key := range yAxis {
		points := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			points = append(points, map[string]any{"x": row[xAxis], "y": mustValue(row, key)})
		}
		series = append(series, map[string]any{"name": key, "data": points})
	}
	return map[string]any{"series": series, "x_label": xAxis, "y_label": strings.Join(yAxis, ", ")}, nil
}

func (s *WidgetService) resolvePieChart(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	suite := stringFromConfig(widget.Config, "suite")
	source := stringFromConfig(widget.Config, "data_source")
	dataPath := stringFromConfig(widget.Config, "data_path")
	if dataPath == "" {
		dataPath = "$.data"
	}
	labelPath := stringFromConfig(widget.Config, "label_path")
	valuePath := stringFromConfig(widget.Config, "value_path")
	payload, err := s.fetchWidgetSource(ctx, tenantID, suite, source)
	if err != nil {
		return nil, err
	}
	rows, err := extractArray(payload, dataPath)
	if err != nil {
		return nil, err
	}
	colors := []string{"#0f766e", "#dc2626", "#2563eb", "#f59e0b", "#9333ea", "#14b8a6"}
	slices := make([]map[string]any, 0, len(rows))
	for idx, row := range rows {
		slices = append(slices, map[string]any{
			"label": row[labelPath],
			"value": mustValue(row, valuePath),
			"color": colors[idx%len(colors)],
		})
	}
	return map[string]any{"slices": slices}, nil
}

func (s *WidgetService) resolveTable(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	suite := stringFromConfig(widget.Config, "suite")
	source := stringFromConfig(widget.Config, "data_source")
	dataPath := stringFromConfig(widget.Config, "data_path")
	if dataPath == "" {
		dataPath = "$.data"
	}
	payload, err := s.fetchWidgetSource(ctx, tenantID, suite, source)
	if err != nil {
		return nil, err
	}
	rows, err := extractArray(payload, dataPath)
	if err != nil {
		return nil, err
	}
	maxRows := int(numberFromConfig(widget.Config, "max_rows", float64(len(rows))))
	if maxRows < len(rows) {
		rows = rows[:maxRows]
	}
	columns := columnsFromConfig(widget.Config, rows)
	return map[string]any{"columns": columns, "rows": rows, "total_count": len(rows)}, nil
}

func (s *WidgetService) resolveHeatmap(ctx context.Context, tenantID uuid.UUID, widget *model.Widget) (interface{}, error) {
	suite := stringFromConfig(widget.Config, "suite")
	source := stringFromConfig(widget.Config, "data_source")
	dataPath := stringFromConfig(widget.Config, "data_path")
	if dataPath == "" {
		dataPath = "$.data"
	}
	payload, err := s.fetchWidgetSource(ctx, tenantID, suite, source)
	if err != nil {
		return nil, err
	}
	rows, err := extractArray(payload, dataPath)
	if err != nil {
		return nil, err
	}
	xAxis := stringFromConfig(widget.Config, "x_axis")
	yAxis := stringFromConfig(widget.Config, "y_axis")
	valueKey := stringFromConfig(widget.Config, "value_key")
	cells := make([]map[string]any, 0, len(rows))
	xSet := map[string]struct{}{}
	ySet := map[string]struct{}{}
	for _, row := range rows {
		x := stringValue(row[xAxis])
		y := stringValue(row[yAxis])
		xSet[x] = struct{}{}
		ySet[y] = struct{}{}
		cells = append(cells, map[string]any{"x": x, "y": y, "value": mustValue(row, valueKey)})
	}
	return map[string]any{"cells": cells, "x_labels": sortedKeys(xSet), "y_labels": sortedKeys(ySet)}, nil
}

func (s *WidgetService) fetchWidgetSource(ctx context.Context, tenantID uuid.UUID, suite, source string) (map[string]any, error) {
	var payload map[string]any
	meta := s.suiteClient.Fetch(ctx, suite, source, tenantID, &payload)
	if meta.Status == "unavailable" {
		return nil, meta.Error
	}
	return payload, nil
}

func extractArray(payload map[string]any, path string) ([]map[string]any, error) {
	value, err := aggregator.Extract(payload, path)
	if err != nil {
		return nil, err
	}
	items, ok := value.([]any)
	if !ok {
		return nil, fmt.Errorf("%w: data at %s is not an array", ErrValidation, path)
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if mapped, ok := item.(map[string]any); ok {
			out = append(out, mapped)
		}
	}
	return out, nil
}

func configUUID(config map[string]any, key string) (uuid.UUID, error) {
	raw := stringFromConfig(config, key)
	if raw == "" {
		return uuid.Nil, fmt.Errorf("%w: %s is required", ErrValidation, key)
	}
	return uuid.Parse(raw)
}

func stringFromConfig(config map[string]any, key string) string {
	return stringValue(config[key])
}

func stringSliceFromConfig(config map[string]any, key string) []string {
	raw, ok := config[key]
	if !ok || raw == nil {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			value := stringValue(item)
			if value != "" {
				out = append(out, value)
			}
		}
		return out
	default:
		return nil
	}
}

func numberFromConfig(config map[string]any, key string, fallback float64) float64 {
	value, ok := config[key]
	if !ok {
		return fallback
	}
	if number, err := aggregator.ExtractValue(map[string]any{key: value}, "$."+key); err == nil {
		return number
	}
	return fallback
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return fmt.Sprint(value)
	}
}

func mustValue(payload map[string]any, path string) float64 {
	value, err := aggregator.ExtractValue(payload, path)
	if err != nil {
		return 0
	}
	return value
}

func columnsFromConfig(config map[string]any, rows []map[string]any) []map[string]any {
	if raw, ok := config["columns"].([]any); ok {
		out := make([]map[string]any, 0, len(raw))
		for _, item := range raw {
			if mapped, ok := item.(map[string]any); ok {
				out = append(out, mapped)
			}
		}
		return out
	}
	if len(rows) == 0 {
		return nil
	}
	keys := make([]string, 0, len(rows[0]))
	for key := range rows[0] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]map[string]any, 0, len(keys))
	for _, key := range keys {
		out = append(out, map[string]any{"key": key, "label": key})
	}
	return out
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func statusFromValue(value float64) string {
	switch {
	case value <= 0:
		return "normal"
	case value < 50:
		return "warning"
	default:
		return "critical"
	}
}

func (s *WidgetService) syncMetrics(ctx context.Context, tenantID uuid.UUID) {
	if s.metrics == nil || s.metrics.WidgetsTotal == nil {
		return
	}
	counts, err := s.widgets.CountByType(ctx, tenantID)
	if err != nil {
		return
	}
	for _, typ := range []string{
		string(model.WidgetTypeKPICard),
		string(model.WidgetTypeLineChart),
		string(model.WidgetTypeBarChart),
		string(model.WidgetTypeAreaChart),
		string(model.WidgetTypePieChart),
		string(model.WidgetTypeGauge),
		string(model.WidgetTypeTable),
		string(model.WidgetTypeAlertFeed),
		string(model.WidgetTypeText),
		string(model.WidgetTypeSparkline),
		string(model.WidgetTypeHeatmap),
		string(model.WidgetTypeStatusGrid),
		string(model.WidgetTypeTrendIndicator),
	} {
		s.metrics.WidgetsTotal.WithLabelValues(tenantID.String(), typ).Set(float64(counts[typ]))
	}
}
