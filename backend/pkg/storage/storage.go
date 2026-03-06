package storage

import (
	"context"
	"io"
	"time"
)

// UploadParams describes a file to upload.
type UploadParams struct {
	Bucket      string
	Key         string
	Body        io.Reader
	Size        int64
	ContentType string
	Metadata    map[string]string
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	Bucket    string
	Key       string
	VersionID string
	ETag      string
	Size      int64
}

// ObjectInfo describes a stored object.
type ObjectInfo struct {
	Bucket       string
	Key          string
	Size         int64
	ContentType  string
	ETag         string
	VersionID    string
	LastModified time.Time
	Metadata     map[string]string
}

// PresignedUploadParams describes constraints for a presigned upload URL.
type PresignedUploadParams struct {
	Bucket         string
	Key            string
	ContentType    string
	MaxSizeBytes   int64
	Expiry         time.Duration
	Metadata       map[string]string
}

// PresignedURL holds a generated presigned URL.
type PresignedURL struct {
	URL       string            `json:"url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// Storage is the interface for object storage operations.
type Storage interface {
	Upload(ctx context.Context, params UploadParams) (*UploadResult, error)
	Download(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error)
	Delete(ctx context.Context, bucket, key string) error
	GeneratePresignedUploadURL(ctx context.Context, params PresignedUploadParams) (*PresignedURL, error)
	GeneratePresignedDownloadURL(ctx context.Context, bucket, key string, expiry time.Duration) (*PresignedURL, error)
	Exists(ctx context.Context, bucket, key string) (bool, error)
	GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error)
	CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error
	EnsureBucket(ctx context.Context, bucket string) error
}

// NewStorage creates a Storage implementation based on Config.Backend.
func NewStorage(cfg Config) (Storage, error) {
	switch cfg.Backend {
	case "local":
		return NewLocalStorage(cfg.LocalBasePath)
	default:
		return NewMinIOStorage(cfg)
	}
}
