package dto

import "github.com/clario360/platform/internal/data/model"

type SchemaResponse struct {
	SourceID string                 `json:"source_id"`
	Schema   *model.DiscoveredSchema `json:"schema"`
}
