package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LocalStorage implements Storage using the local filesystem.
// Intended for testing and air-gapped deployments.
type LocalStorage struct {
	basePath string
}

// NewLocalStorage creates a local filesystem storage.
func NewLocalStorage(basePath string) (*LocalStorage, error) {
	abs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("local storage: resolving path: %w", err)
	}
	if err := os.MkdirAll(abs, 0o750); err != nil {
		return nil, fmt.Errorf("local storage: creating base dir: %w", err)
	}
	return &LocalStorage{basePath: abs}, nil
}

func (s *LocalStorage) Upload(ctx context.Context, params UploadParams) (*UploadResult, error) {
	path, err := s.safePath(params.Bucket, params.Key)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, fmt.Errorf("local storage: creating dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("local storage: creating file: %w", err)
	}
	defer f.Close()

	written, err := io.Copy(f, params.Body)
	if err != nil {
		os.Remove(path)
		return nil, fmt.Errorf("local storage: writing file: %w", err)
	}

	return &UploadResult{
		Bucket: params.Bucket,
		Key:    params.Key,
		Size:   written,
	}, nil
}

func (s *LocalStorage) Download(_ context.Context, bucket, key string) (io.ReadCloser, *ObjectInfo, error) {
	path, err := s.safePath(bucket, key)
	if err != nil {
		return nil, nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("local storage: file not found: %s/%s", bucket, key)
		}
		return nil, nil, fmt.Errorf("local storage: opening file: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, fmt.Errorf("local storage: stat file: %w", err)
	}

	info := &ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         stat.Size(),
		LastModified: stat.ModTime(),
	}

	return f, info, nil
}

func (s *LocalStorage) Delete(_ context.Context, bucket, key string) error {
	path, err := s.safePath(bucket, key)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("local storage: deleting file: %w", err)
	}
	return nil
}

func (s *LocalStorage) GeneratePresignedUploadURL(_ context.Context, _ PresignedUploadParams) (*PresignedURL, error) {
	return nil, fmt.Errorf("local storage: presigned upload URLs not supported")
}

func (s *LocalStorage) GeneratePresignedDownloadURL(_ context.Context, _, _ string, _ time.Duration) (*PresignedURL, error) {
	return nil, fmt.Errorf("local storage: presigned download URLs not supported")
}

func (s *LocalStorage) Exists(_ context.Context, bucket, key string) (bool, error) {
	path, err := s.safePath(bucket, key)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("local storage: stat: %w", err)
	}
	return true, nil
}

func (s *LocalStorage) GetObjectInfo(_ context.Context, bucket, key string) (*ObjectInfo, error) {
	path, err := s.safePath(bucket, key)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("local storage: stat: %w", err)
	}
	return &ObjectInfo{
		Bucket:       bucket,
		Key:          key,
		Size:         stat.Size(),
		LastModified: stat.ModTime(),
	}, nil
}

func (s *LocalStorage) CopyObject(_ context.Context, srcBucket, srcKey, dstBucket, dstKey string) error {
	srcPath, err := s.safePath(srcBucket, srcKey)
	if err != nil {
		return err
	}
	dstPath, err := s.safePath(dstBucket, dstKey)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o750); err != nil {
		return fmt.Errorf("local storage: creating dest dir: %w", err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("local storage: opening source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("local storage: creating dest: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("local storage: copying: %w", err)
	}
	return nil
}

func (s *LocalStorage) EnsureBucket(_ context.Context, bucket string) error {
	path := filepath.Join(s.basePath, bucket)
	resolved, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("local storage: resolving bucket path: %w", err)
	}
	if !strings.HasPrefix(resolved, s.basePath) {
		return fmt.Errorf("local storage: bucket path traversal detected")
	}
	return os.MkdirAll(resolved, 0o750)
}

// safePath builds and validates a filesystem path, preventing path traversal.
func (s *LocalStorage) safePath(bucket, key string) (string, error) {
	// Reject null bytes
	if strings.ContainsRune(bucket, 0) || strings.ContainsRune(key, 0) {
		return "", fmt.Errorf("local storage: null byte in path")
	}

	combined := filepath.Join(s.basePath, bucket, key)
	resolved, err := filepath.Abs(combined)
	if err != nil {
		return "", fmt.Errorf("local storage: resolving path: %w", err)
	}

	if !strings.HasPrefix(resolved, s.basePath) {
		return "", fmt.Errorf("local storage: path traversal detected")
	}

	return resolved, nil
}
