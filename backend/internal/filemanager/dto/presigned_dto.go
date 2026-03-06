package dto

import (
	"fmt"

	"github.com/clario360/platform/internal/filemanager/model"
)

// PresignedUploadRequest describes a request for a presigned upload URL.
type PresignedUploadRequest struct {
	Filename        string `json:"filename"`
	ContentType     string `json:"content_type"`
	Suite           string `json:"suite"`
	EntityType      string `json:"entity_type,omitempty"`
	EntityID        string `json:"entity_id,omitempty"`
	SizeBytes       int64  `json:"size_bytes"`
	Encrypt         bool   `json:"encrypt"`
	LifecyclePolicy string `json:"lifecycle_policy,omitempty"`
}

// Validate checks the presigned upload request.
func (p *PresignedUploadRequest) Validate() error {
	if p.Filename == "" {
		return fmt.Errorf("filename is required")
	}
	if p.ContentType == "" {
		return fmt.Errorf("content_type is required")
	}
	if p.Suite == "" {
		return fmt.Errorf("suite is required")
	}
	if !model.ValidSuites[p.Suite] {
		return fmt.Errorf("invalid suite: %s", p.Suite)
	}
	if p.SizeBytes <= 0 {
		return fmt.Errorf("size_bytes must be positive")
	}
	return nil
}

// PresignedUploadResponse is returned after generating a presigned upload URL.
type PresignedUploadResponse struct {
	FileID    string            `json:"file_id"`
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt string            `json:"expires_at"`
}

// PresignedDownloadResponse is returned for presigned download URLs.
type PresignedDownloadResponse struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	ExpiresAt string `json:"expires_at"`
}

// QuarantineResolveRequest is the body for resolving a quarantined file.
type QuarantineResolveRequest struct {
	Action string `json:"action"` // "deleted", "restored", "false_positive"
}

// Validate checks the quarantine resolve request.
func (q *QuarantineResolveRequest) Validate() error {
	switch q.Action {
	case "deleted", "restored", "false_positive":
		return nil
	default:
		return fmt.Errorf("invalid resolution action: %s (must be deleted, restored, or false_positive)", q.Action)
	}
}
