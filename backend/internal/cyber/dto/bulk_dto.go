package dto

import (
	"github.com/google/uuid"
)

// BulkCreateResult is returned after a successful bulk asset import.
type BulkCreateResult struct {
	Count int         `json:"count"`
	IDs   []uuid.UUID `json:"ids"`
}

// BulkValidationError is returned when one or more rows fail validation.
type BulkValidationError struct {
	Code    string                      `json:"code"`
	Message string                      `json:"message"`
	Rows    map[int]map[string][]string `json:"rows"`
}

// BulkTagRequest is the body for PUT /api/v1/cyber/assets/bulk/tags.
type BulkTagRequest struct {
	AssetIDs []string `json:"asset_ids" validate:"required,min=1,max=1000,dive,uuid"`
	Add      []string `json:"add,omitempty" validate:"omitempty,max=20,dive,min=1,max=50,alphanumdash"`
	Remove   []string `json:"remove,omitempty" validate:"omitempty,max=20,dive,min=1,max=50"`
}

// BulkDeleteRequest is the body for DELETE /api/v1/cyber/assets/bulk.
type BulkDeleteRequest struct {
	AssetIDs []string `json:"asset_ids" validate:"required,min=1,max=1000,dive,uuid"`
}
