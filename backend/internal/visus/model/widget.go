package model

import (
	"time"

	"github.com/google/uuid"
)

type WidgetType string

const (
	WidgetTypeKPICard        WidgetType = "kpi_card"
	WidgetTypeLineChart      WidgetType = "line_chart"
	WidgetTypeBarChart       WidgetType = "bar_chart"
	WidgetTypeAreaChart      WidgetType = "area_chart"
	WidgetTypePieChart       WidgetType = "pie_chart"
	WidgetTypeGauge          WidgetType = "gauge"
	WidgetTypeTable          WidgetType = "table"
	WidgetTypeAlertFeed      WidgetType = "alert_feed"
	WidgetTypeText           WidgetType = "text"
	WidgetTypeSparkline      WidgetType = "sparkline"
	WidgetTypeHeatmap        WidgetType = "heatmap"
	WidgetTypeStatusGrid     WidgetType = "status_grid"
	WidgetTypeTrendIndicator WidgetType = "trend_indicator"
)

type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type Widget struct {
	ID                     uuid.UUID      `json:"id"`
	TenantID               uuid.UUID      `json:"tenant_id"`
	DashboardID            uuid.UUID      `json:"dashboard_id"`
	Title                  string         `json:"title"`
	Subtitle               *string        `json:"subtitle,omitempty"`
	Type                   WidgetType     `json:"type"`
	Config                 map[string]any `json:"config"`
	Position               WidgetPosition `json:"position"`
	RefreshIntervalSeconds int            `json:"refresh_interval_seconds"`
	CreatedAt              time.Time      `json:"created_at"`
	UpdatedAt              time.Time      `json:"updated_at"`
}
