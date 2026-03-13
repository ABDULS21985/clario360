package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type FeatureType string

const (
	FeatureTypeNumeric     FeatureType = "numeric"
	FeatureTypeCategorical FeatureType = "categorical"
	FeatureTypeBoolean     FeatureType = "boolean"
	FeatureTypeEmbedding   FeatureType = "embedding"
	FeatureTypeTimeSeries  FeatureType = "time_series"
)

type Feature struct {
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Type        FeatureType `json:"type"`
	Value       float64     `json:"value,omitempty"`
	RawValue    any         `json:"raw_value,omitempty"`
}

type FeatureVector struct {
	EntityType string               `json:"entity_type"`
	EntityID   string               `json:"entity_id,omitempty"`
	Timestamp  time.Time            `json:"timestamp"`
	Values     map[string]float64   `json:"values"`
	Categories map[string]string    `json:"categories,omitempty"`
	Arrays     map[string][]float64 `json:"arrays,omitempty"`
	Labels     map[string]float64   `json:"labels,omitempty"`
	Metadata   map[string]any       `json:"metadata,omitempty"`
}

func (v FeatureVector) Clone() FeatureVector {
	out := FeatureVector{
		EntityType: v.EntityType,
		EntityID:   v.EntityID,
		Timestamp:  v.Timestamp,
		Values:     map[string]float64{},
		Categories: map[string]string{},
		Arrays:     map[string][]float64{},
		Labels:     map[string]float64{},
		Metadata:   map[string]any{},
	}
	for key, value := range v.Values {
		out.Values[key] = value
	}
	for key, value := range v.Categories {
		out.Categories[key] = value
	}
	for key, value := range v.Arrays {
		out.Arrays[key] = append([]float64(nil), value...)
	}
	for key, value := range v.Labels {
		out.Labels[key] = value
	}
	for key, value := range v.Metadata {
		out.Metadata[key] = value
	}
	return out
}

func (v FeatureVector) Value(name string) float64 {
	if v.Values == nil {
		return 0
	}
	return v.Values[name]
}

type FeatureSnapshot struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	TenantID   uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	FeatureSet string          `json:"feature_set" db:"feature_set"`
	EntityType string          `json:"entity_type" db:"entity_type"`
	EntityID   *string         `json:"entity_id,omitempty" db:"entity_id"`
	VectorJSON json.RawMessage `json:"vector_json" db:"vector_json"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}
