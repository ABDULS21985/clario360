package dto

import "github.com/google/uuid"

type StartShadowRequest struct {
	VersionID uuid.UUID `json:"version_id"`
}

type StopShadowRequest struct {
	VersionID uuid.UUID `json:"version_id"`
	Reason    string    `json:"reason"`
}
