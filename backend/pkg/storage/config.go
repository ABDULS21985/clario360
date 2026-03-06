package storage

import "time"

// Config holds configuration for the storage backend.
type Config struct {
	// Backend selects the storage implementation: "minio" or "local".
	Backend string

	// MinIO settings
	Endpoint  string
	AccessKey string
	SecretKey string
	UseSSL    bool
	Region    string

	// Bucket settings
	BucketPrefix    string
	QuarantineBucket string

	// Local filesystem settings (for testing / air-gap)
	LocalBasePath string

	// Presigned URL settings
	PresignedURLExpiry time.Duration

	// Encryption
	EncryptionMasterKey []byte // 32 bytes for AES-256
	EncryptionKeyID     string
}
