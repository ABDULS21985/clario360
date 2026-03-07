package strategies

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/clario360/platform/internal/data/darkdata"
	"github.com/clario360/platform/internal/data/model"
)

type OrphanedFilesStrategy struct {
	db        *pgxpool.Pool
	endpoint  string
	accessKey string
	secretKey string
	bucket    string
	secure    bool
}

func NewOrphanedFilesStrategy(db *pgxpool.Pool, endpoint, accessKey, secretKey, bucket string) *OrphanedFilesStrategy {
	secure := strings.HasPrefix(endpoint, "https://")
	endpoint = strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
	return &OrphanedFilesStrategy{
		db:        db,
		endpoint:  endpoint,
		accessKey: accessKey,
		secretKey: secretKey,
		bucket:    bucket,
		secure:    secure,
	}
}

func (s *OrphanedFilesStrategy) Name() string {
	return "orphaned_files"
}

func (s *OrphanedFilesStrategy) Scan(ctx context.Context, tenantID uuid.UUID) ([]darkdata.RawDarkDataAsset, error) {
	if s.endpoint == "" || s.bucket == "" || s.accessKey == "" || s.secretKey == "" {
		return nil, nil
	}
	hasFilesTable, err := s.hasFilesTable(ctx)
	if err != nil {
		return nil, err
	}
	registered := map[string]struct{}{}
	if hasFilesTable {
		rows, err := s.db.Query(ctx, `
			SELECT storage_key
			FROM files
			WHERE tenant_id = $1 AND bucket = $2 AND deleted_at IS NULL`,
			tenantID, s.bucket,
		)
		if err != nil {
			return nil, fmt.Errorf("query registered files: %w", err)
		}
		for rows.Next() {
			var key string
			if err := rows.Scan(&key); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan registered file: %w", err)
			}
			registered[key] = struct{}{}
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("iterate registered files: %w", err)
		}
		rows.Close()
	}

	client, err := minio.New(s.endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
		Secure: s.secure,
	})
	if err != nil {
		return nil, fmt.Errorf("create dark-data minio client: %w", err)
	}
	prefix := tenantID.String() + "/"
	results := make([]darkdata.RawDarkDataAsset, 0)
	for object := range client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true}) {
		if object.Err != nil {
			return nil, fmt.Errorf("list orphaned files: %w", object.Err)
		}
		if _, ok := registered[object.Key]; ok {
			continue
		}
		filePath := object.Key
		lastModified := object.LastModified.UTC()
		size := object.Size
		results = append(results, darkdata.RawDarkDataAsset{
			Name:               object.Key,
			AssetType:          model.DarkDataAssetFile,
			FilePath:           &filePath,
			Reason:             model.DarkDataReasonOrphanedFile,
			EstimatedSizeBytes: &size,
			LastModifiedAt:     &lastModified,
			Metadata: map[string]any{
				"etag":         object.ETag,
				"content_type": object.ContentType,
			},
		})
	}
	return results, nil
}

func (s *OrphanedFilesStrategy) hasFilesTable(ctx context.Context) (bool, error) {
	row := s.db.QueryRow(ctx, `SELECT to_regclass('public.files') IS NOT NULL`)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("check files table: %w", err)
	}
	return exists, nil
}

