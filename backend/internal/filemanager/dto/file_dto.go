package dto

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/clario360/platform/internal/filemanager/model"
)

// UploadRequest holds parsed upload parameters from a multipart request.
type UploadRequest struct {
	Suite           string   `json:"suite"`
	EntityType      string   `json:"entity_type,omitempty"`
	EntityID        string   `json:"entity_id,omitempty"`
	Tags            []string `json:"tags,omitempty"`
	Encrypt         bool     `json:"encrypt"`
	LifecyclePolicy string   `json:"lifecycle_policy"`
	ExpiresAt       string   `json:"expires_at,omitempty"`
	DedupCheck      bool     `json:"dedup_check"`
}

// Validate checks upload request fields.
func (u *UploadRequest) Validate() error {
	if u.Suite == "" {
		return fmt.Errorf("suite is required")
	}
	if !model.ValidSuites[u.Suite] {
		return fmt.Errorf("invalid suite: %s", u.Suite)
	}
	switch u.LifecyclePolicy {
	case "", model.LifecycleStandard, model.LifecycleTemporary, model.LifecycleArchive, model.LifecycleAuditRetention:
		// valid
	default:
		return fmt.Errorf("invalid lifecycle_policy: %s", u.LifecyclePolicy)
	}
	if u.LifecyclePolicy == "" {
		u.LifecyclePolicy = model.LifecycleStandard
	}
	return nil
}

// FileResponse is the API response for a file record.
type FileResponse struct {
	ID                  string     `json:"id"`
	TenantID            string     `json:"tenant_id"`
	Name                string     `json:"name"`
	OriginalName        string     `json:"original_name"`
	SanitizedName       string     `json:"sanitized_name"`
	ContentType         string     `json:"content_type"`
	DetectedContentType string     `json:"detected_content_type,omitempty"`
	Size                int64      `json:"size"`
	SizeBytes           int64      `json:"size_bytes"`
	Status              string     `json:"status"`
	ChecksumSHA256      string     `json:"checksum_sha256"`
	Encrypted           bool       `json:"encrypted"`
	VirusScanStatus     string     `json:"virus_scan_status"`
	UploadedBy          string     `json:"uploaded_by"`
	Suite               string     `json:"suite"`
	EntityType          *string    `json:"entity_type,omitempty"`
	EntityID            *string    `json:"entity_id,omitempty"`
	Tags                []string   `json:"tags"`
	VersionNumber       int        `json:"version_number"`
	IsPublic            bool       `json:"is_public"`
	LifecyclePolicy     string     `json:"lifecycle_policy"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

// FileResponseFromModel converts a FileRecord to an API response.
func FileResponseFromModel(f *model.FileRecord) *FileResponse {
	return &FileResponse{
		ID:                  f.ID,
		TenantID:            f.TenantID,
		Name:                fileDisplayName(f),
		OriginalName:        f.OriginalName,
		SanitizedName:       f.SanitizedName,
		ContentType:         f.ContentType,
		DetectedContentType: f.DetectedContentType,
		Size:                f.SizeBytes,
		SizeBytes:           f.SizeBytes,
		Status:              fileStatus(f),
		ChecksumSHA256:      f.ChecksumSHA256,
		Encrypted:           f.Encrypted,
		VirusScanStatus:     f.VirusScanStatus,
		UploadedBy:          f.UploadedBy,
		Suite:               f.Suite,
		EntityType:          f.EntityType,
		EntityID:            f.EntityID,
		Tags:                f.Tags,
		VersionNumber:       f.VersionNumber,
		IsPublic:            f.IsPublic,
		LifecyclePolicy:     f.LifecyclePolicy,
		ExpiresAt:           f.ExpiresAt,
		CreatedAt:           f.CreatedAt,
		UpdatedAt:           f.UpdatedAt,
	}
}

func fileDisplayName(f *model.FileRecord) string {
	if f.SanitizedName != "" {
		return f.SanitizedName
	}
	return f.OriginalName
}

func fileStatus(f *model.FileRecord) string {
	if f.DeletedAt != nil {
		return "deleted"
	}
	switch f.VirusScanStatus {
	case model.ScanStatusPending:
		return "pending"
	case model.ScanStatusScanning:
		return "processing"
	case model.ScanStatusInfected:
		return "quarantined"
	case model.ScanStatusClean, model.ScanStatusSkipped:
		return "available"
	case model.ScanStatusError:
		// Scan failed; treat as available so the file is not permanently inaccessible.
		// The virus_scan_status field itself surfaces the "error" state to the client.
		return "available"
	default:
		return "processing"
	}
}

// ListFilesParams holds query parameters for listing files.
type ListFilesParams struct {
	TenantID   string
	Suite      string
	EntityType string
	EntityID   string
	UploadedBy string
	Tag        string
	Page       int
	PerPage    int
}

// ParseListParams extracts list parameters from the HTTP request.
func ParseListParams(r *http.Request) ListFilesParams {
	p := ListFilesParams{
		Page:    1,
		PerPage: 20,
	}
	q := r.URL.Query()

	p.Suite = q.Get("suite")
	p.EntityType = q.Get("entity_type")
	p.EntityID = q.Get("entity_id")
	p.UploadedBy = q.Get("uploaded_by")
	p.Tag = q.Get("tag")

	if v, err := strconv.Atoi(q.Get("page")); err == nil && v > 0 {
		p.Page = v
	}
	if v, err := strconv.Atoi(q.Get("per_page")); err == nil && v > 0 && v <= 100 {
		p.PerPage = v
	}
	return p
}

// ListResponse is a paginated response.
type ListResponse struct {
	Data interface{}    `json:"data"`
	Meta PaginationMeta `json:"meta"`
}

type PaginationMeta struct {
	Page       int `json:"page"`
	PerPage    int `json:"per_page"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func NewListResponse(data interface{}, total, page, perPage int) ListResponse {
	totalPages := total / perPage
	if total%perPage > 0 {
		totalPages++
	}
	if totalPages < 1 {
		totalPages = 1
	}
	return ListResponse{
		Data: data,
		Meta: PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	}
}

// PresignedConfirmRequest is the body for confirming a presigned upload.
type PresignedConfirmRequest struct {
	FileID string `json:"file_id"`
}

// ParseUploadForm parses multipart form values into UploadRequest.
func ParseUploadForm(r *http.Request) *UploadRequest {
	req := &UploadRequest{
		Suite:           r.FormValue("suite"),
		EntityType:      r.FormValue("entity_type"),
		EntityID:        r.FormValue("entity_id"),
		LifecyclePolicy: r.FormValue("lifecycle_policy"),
		ExpiresAt:       r.FormValue("expires_at"),
		DedupCheck:      r.FormValue("dedup_check") == "true",
		Encrypt:         r.FormValue("encrypt") == "true",
	}
	if tags := r.FormValue("tags"); tags != "" {
		req.Tags = strings.Split(tags, ",")
		for i := range req.Tags {
			req.Tags[i] = strings.TrimSpace(req.Tags[i])
		}
	}
	return req
}
