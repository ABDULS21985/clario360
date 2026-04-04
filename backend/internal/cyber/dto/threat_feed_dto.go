package dto

import (
	"encoding/json"

	"github.com/clario360/platform/internal/cyber/model"
)

type ThreatFeedConfigRequest struct {
	Name              string                   `json:"name" validate:"required,min=1,max=255"`
	Type              model.ThreatFeedType     `json:"type" validate:"required"`
	URL               string                   `json:"url,omitempty" validate:"omitempty,max=2048"`
	AuthType          model.ThreatFeedAuthType `json:"auth_type" validate:"required"`
	AuthConfig        json.RawMessage          `json:"auth_config,omitempty"`
	SyncInterval      model.ThreatFeedInterval `json:"sync_interval" validate:"required"`
	DefaultSeverity   model.Severity           `json:"default_severity" validate:"required"`
	DefaultConfidence float64                  `json:"default_confidence" validate:"gte=0,lte=1"`
	DefaultTags       []string                 `json:"default_tags,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	IndicatorTypes    []string                 `json:"indicator_types,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
	Enabled           bool                     `json:"enabled"`
}

type ThreatFeedListResponse struct {
	Data []*model.ThreatFeedConfig `json:"data"`
	Meta PaginationMeta            `json:"meta"`
}

type ThreatFeedHistoryResponse struct {
	Data []*model.ThreatFeedSyncHistory `json:"data"`
}

type ThreatFeedSyncResponse struct {
	Data map[string]interface{} `json:"data"`
}
