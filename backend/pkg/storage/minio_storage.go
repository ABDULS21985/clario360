package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var storageTracer = otel.Tracer("pkg/storage")

// MinIOStorage implements Storage using MinIO (S3-compatible).
type MinIOStorage struct {
	client       *minio.Client
	bucketPrefix string
	region       string
}

// NewMinIOStorage creates a MinIO storage client.
func NewMinIOStorage(cfg Config) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio: creating client: %w", err)
	}
	return &MinIOStorage{
		client:       client,
		bucketPrefix: cfg.BucketPrefix,
		region:       cfg.Region,
	}, nil
}

// Client returns the underlying minio.Client for health checks.
func (s *MinIOStorage) Client() *minio.Client {
	return s.client
}

func (s *MinIOStorage) Upload(ctx context.Context, params UploadParams) (*UploadResult, error) {
	ctx, span := storageTracer.Start(ctx, "minio.upload", trace.WithAttributes(
		attribute.String("bucket", params.Bucket),
		attribute.String("key", params.Key),
		attribute.Int64("size", params.Size),
	))
	defer span.End()

	opts := minio.PutObjectOptions{
		ContentType:  params.ContentType,
		UserMetadata: params.Metadata,
	}

	info, err := s.client.PutObject(ctx, params.Bucket, params.Key, params.Body, params.Size, opts)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("minio: upload %s/%s: %w", params.Bucket, params.Key, err)
	}

	return &UploadResult{
		Bucket:    info.Bucket,
		Key:       info.Key,
		VersionID: info.VersionID,
		ETag:      info.ETag,
		Size:      info.Size,
	}, nil
}

func (s *MinIOStorage) Download(ctx context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error) {
	ctx, span := storageTracer.Start(ctx, "minio.download", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	))
	defer span.End()

	obj, err := s.client.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
	if err != nil {
		span.RecordError(err)
		return nil, nil, fmt.Errorf("minio: download %s/%s: %w", bucket, key, err)
	}

	stat, err := obj.Stat()
	if err != nil {
		obj.Close()
		span.RecordError(err)
		return nil, nil, fmt.Errorf("minio: stat %s/%s: %w", bucket, key, err)
	}

	info := &ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		VersionID:    stat.VersionID,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
	}

	return obj, info, nil
}

func (s *MinIOStorage) Delete(ctx context.Context, bucket, key string) error {
	ctx, span := storageTracer.Start(ctx, "minio.delete", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	))
	defer span.End()

	err := s.client.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: delete %s/%s: %w", bucket, key, err)
	}
	return nil
}

func (s *MinIOStorage) GeneratePresignedUploadURL(ctx context.Context, params PresignedUploadParams) (*PresignedURL, error) {
	ctx, span := storageTracer.Start(ctx, "minio.presigned_upload", trace.WithAttributes(
		attribute.String("bucket", params.Bucket),
		attribute.String("key", params.Key),
	))
	defer span.End()

	expiry := params.Expiry
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	reqParams := make(url.Values)
	if params.ContentType != "" {
		reqParams.Set("Content-Type", params.ContentType)
	}

	presignedURL, err := s.client.PresignedPutObject(ctx, params.Bucket, params.Key, expiry)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("minio: presigned upload %s/%s: %w", params.Bucket, params.Key, err)
	}

	headers := make(map[string]string)
	if params.ContentType != "" {
		headers["Content-Type"] = params.ContentType
	}

	return &PresignedURL{
		URL:       presignedURL.String(),
		Method:    "PUT",
		Headers:   headers,
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

func (s *MinIOStorage) GeneratePresignedDownloadURL(ctx context.Context, bucket, key string, expiry time.Duration) (*PresignedURL, error) {
	ctx, span := storageTracer.Start(ctx, "minio.presigned_download", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	))
	defer span.End()

	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	presignedURL, err := s.client.PresignedGetObject(ctx, bucket, key, expiry, nil)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("minio: presigned download %s/%s: %w", bucket, key, err)
	}

	return &PresignedURL{
		URL:       presignedURL.String(),
		Method:    "GET",
		ExpiresAt: time.Now().Add(expiry),
	}, nil
}

func (s *MinIOStorage) Exists(ctx context.Context, bucket, key string) (bool, error) {
	_, span := storageTracer.Start(ctx, "minio.exists", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	))
	defer span.End()

	_, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}
		span.RecordError(err)
		return false, fmt.Errorf("minio: exists %s/%s: %w", bucket, key, err)
	}
	return true, nil
}

func (s *MinIOStorage) GetObjectInfo(ctx context.Context, bucket, key string) (*ObjectInfo, error) {
	ctx, span := storageTracer.Start(ctx, "minio.get_info", trace.WithAttributes(
		attribute.String("bucket", bucket),
		attribute.String("key", key),
	))
	defer span.End()

	stat, err := s.client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("minio: stat %s/%s: %w", bucket, key, err)
	}

	return &ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         stat.Size,
		ContentType:  stat.ContentType,
		ETag:         stat.ETag,
		VersionID:    stat.VersionID,
		LastModified: stat.LastModified,
		Metadata:     stat.UserMetadata,
	}, nil
}

func (s *MinIOStorage) CopyObject(ctx context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	ctx, span := storageTracer.Start(ctx, "minio.copy", trace.WithAttributes(
		attribute.String("src_bucket", srcBucket),
		attribute.String("src_key", srcKey),
		attribute.String("dst_bucket", dstBucket),
		attribute.String("dst_key", dstKey),
	))
	defer span.End()

	src := minio.CopySrcOptions{
		Bucket: srcBucket,
		Object: srcKey,
	}
	dst := minio.CopyDestOptions{
		Bucket: dstBucket,
		Object: dstKey,
	}

	_, err := s.client.CopyObject(ctx, dst, src)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: copy %s/%s → %s/%s: %w", srcBucket, srcKey, dstBucket, dstKey, err)
	}
	return nil
}

func (s *MinIOStorage) EnsureBucket(ctx context.Context, bucket string) error {
	ctx, span := storageTracer.Start(ctx, "minio.ensure_bucket", trace.WithAttributes(
		attribute.String("bucket", bucket),
	))
	defer span.End()

	exists, err := s.client.BucketExists(ctx, bucket)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: check bucket %s: %w", bucket, err)
	}
	if exists {
		return nil
	}

	if err := s.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: s.region}); err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: create bucket %s: %w", bucket, err)
	}

	// Enable versioning
	err = s.client.EnableVersioning(ctx, bucket)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("minio: enable versioning %s: %w", bucket, err)
	}

	return nil
}
