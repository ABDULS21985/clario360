package dto

import "github.com/google/uuid"

type PromoteRequest struct {
	ApprovedBy *uuid.UUID `json:"approved_by"`
	Override   bool       `json:"override"`
}

type RetireVersionRequest struct {
	Reason string `json:"reason"`
}

type FailVersionRequest struct {
	Reason string `json:"reason"`
}

type RollbackRequest struct {
	Reason string `json:"reason"`
}
