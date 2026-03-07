package dto

import "github.com/google/uuid"

type RunContradictionScanRequest struct {
	TriggeredBy *uuid.UUID `json:"triggered_by,omitempty"`
}

type UpdateContradictionStatusRequest struct {
	Status string `json:"status"`
}

type ResolveContradictionRequest struct {
	ResolutionAction string `json:"resolution_action"`
	ResolutionNotes  string `json:"resolution_notes"`
}

type ListContradictionsParams struct {
	Page      int
	PerPage   int
	Type      string
	Severity  string
	Status    string
	Search    string
	Sort      string
	Order     string
}

type ListContradictionScansParams struct {
	Page    int
	PerPage int
	Status  string
}
