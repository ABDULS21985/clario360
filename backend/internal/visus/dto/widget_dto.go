package dto

import "github.com/google/uuid"

type CreateWidgetRequest struct {
	Title                  string         `json:"title"`
	Subtitle               *string        `json:"subtitle"`
	Type                   string         `json:"type"`
	Config                 map[string]any `json:"config"`
	Position               WidgetPosition `json:"position"`
	RefreshIntervalSeconds int            `json:"refresh_interval_seconds"`
}

type UpdateWidgetRequest struct {
	Title                  string         `json:"title"`
	Subtitle               *string        `json:"subtitle"`
	Config                 map[string]any `json:"config"`
	Position               WidgetPosition `json:"position"`
	RefreshIntervalSeconds int            `json:"refresh_interval_seconds"`
}

type WidgetPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

type WidgetLayoutPosition struct {
	WidgetID uuid.UUID `json:"widget_id"`
	X        int       `json:"x"`
	Y        int       `json:"y"`
	W        int       `json:"w"`
	H        int       `json:"h"`
}

type UpdateWidgetLayoutRequest struct {
	Positions []WidgetLayoutPosition `json:"positions"`
}
