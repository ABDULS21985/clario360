package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalStorage_UploadDownloadDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	bucket := "test-bucket"
	key := "tenant/suite/2026/03/test-file.pdf"
	content := []byte("This is the file content for upload/download/delete test.")

	// Ensure bucket
	if err := store.EnsureBucket(ctx, bucket); err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}

	// Upload
	result, err := store.Upload(ctx, UploadParams{
		Bucket:      bucket,
		Key:         key,
		Body:        bytes.NewReader(content),
		Size:        int64(len(content)),
		ContentType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}
	if result.Bucket != bucket {
		t.Fatalf("expected bucket %q, got %q", bucket, result.Bucket)
	}
	if result.Key != key {
		t.Fatalf("expected key %q, got %q", key, result.Key)
	}
	if result.Size != int64(len(content)) {
		t.Fatalf("expected size %d, got %d", len(content), result.Size)
	}

	// Download
	reader, info, err := store.Download(ctx, bucket, key)
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading downloaded content: %v", err)
	}
	if !bytes.Equal(downloaded, content) {
		t.Fatalf("downloaded content mismatch:\n  got:  %q\n  want: %q", downloaded, content)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected info size %d, got %d", len(content), info.Size)
	}
	if info.Bucket != bucket || info.Key != key {
		t.Fatalf("unexpected info: bucket=%q key=%q", info.Bucket, info.Key)
	}

	// Delete
	if err := store.Delete(ctx, bucket, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify file is gone
	exists, err := store.Exists(ctx, bucket, key)
	if err != nil {
		t.Fatalf("Exists after delete: %v", err)
	}
	if exists {
		t.Fatal("file should not exist after deletion")
	}
}

func TestLocalStorage_PathTraversal_Rejected(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()

	// Keys that, when combined with bucket "bucket", resolve outside basePath.
	// filepath.Join(basePath, "bucket", key) must resolve above basePath to be rejected.
	traversalKeys := []string{
		"../../etc/passwd",
		"../../../etc/shadow",
	}

	for _, key := range traversalKeys {
		t.Run(key, func(t *testing.T) {
			// Upload with path traversal key should fail
			_, err := store.Upload(ctx, UploadParams{
				Bucket: "bucket",
				Key:    key,
				Body:   bytes.NewReader([]byte("malicious")),
				Size:   9,
			})
			if err == nil {
				t.Fatalf("expected error for path traversal key %q, got nil", key)
			}

			// Download with path traversal key should fail
			_, _, err = store.Download(ctx, "bucket", key)
			if err == nil {
				t.Fatalf("expected error for download with path traversal key %q", key)
			}

			// Exists with path traversal key should fail
			_, err = store.Exists(ctx, "bucket", key)
			if err == nil {
				t.Fatalf("expected error for exists with path traversal key %q", key)
			}

			// Delete with path traversal key should fail
			err = store.Delete(ctx, "bucket", key)
			if err == nil {
				t.Fatalf("expected error for delete with path traversal key %q", key)
			}
		})
	}
}

func TestLocalStorage_Exists(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	bucket := "test-bucket"
	key := "some/key.txt"

	if err := store.EnsureBucket(ctx, bucket); err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}

	// File should not exist yet
	exists, err := store.Exists(ctx, bucket, key)
	if err != nil {
		t.Fatalf("Exists before upload: %v", err)
	}
	if exists {
		t.Fatal("file should not exist before upload")
	}

	// Upload file
	_, err = store.Upload(ctx, UploadParams{
		Bucket: bucket,
		Key:    key,
		Body:   bytes.NewReader([]byte("content")),
		Size:   7,
	})
	if err != nil {
		t.Fatalf("Upload: %v", err)
	}

	// File should now exist
	exists, err = store.Exists(ctx, bucket, key)
	if err != nil {
		t.Fatalf("Exists after upload: %v", err)
	}
	if !exists {
		t.Fatal("file should exist after upload")
	}
}

func TestLocalStorage_CopyObject(t *testing.T) {
	dir := t.TempDir()
	store, err := NewLocalStorage(dir)
	if err != nil {
		t.Fatalf("NewLocalStorage: %v", err)
	}

	ctx := context.Background()
	srcBucket := "src-bucket"
	srcKey := "original.txt"
	dstBucket := "dst-bucket"
	dstKey := "copy.txt"
	content := []byte("content to be copied")

	// Ensure both buckets
	if err := store.EnsureBucket(ctx, srcBucket); err != nil {
		t.Fatalf("EnsureBucket src: %v", err)
	}
	if err := store.EnsureBucket(ctx, dstBucket); err != nil {
		t.Fatalf("EnsureBucket dst: %v", err)
	}

	// Upload source file
	_, err = store.Upload(ctx, UploadParams{
		Bucket: srcBucket,
		Key:    srcKey,
		Body:   bytes.NewReader(content),
		Size:   int64(len(content)),
	})
	if err != nil {
		t.Fatalf("Upload source: %v", err)
	}

	// Copy object
	if err := store.CopyObject(ctx, srcBucket, srcKey, dstBucket, dstKey); err != nil {
		t.Fatalf("CopyObject: %v", err)
	}

	// Verify destination exists
	exists, err := store.Exists(ctx, dstBucket, dstKey)
	if err != nil {
		t.Fatalf("Exists dest: %v", err)
	}
	if !exists {
		t.Fatal("destination file should exist after copy")
	}

	// Verify destination content matches
	reader, info, err := store.Download(ctx, dstBucket, dstKey)
	if err != nil {
		t.Fatalf("Download dest: %v", err)
	}
	defer reader.Close()

	downloaded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("reading dest content: %v", err)
	}
	if !bytes.Equal(downloaded, content) {
		t.Fatalf("copied content mismatch:\n  got:  %q\n  want: %q", downloaded, content)
	}
	if info.Size != int64(len(content)) {
		t.Fatalf("expected dest size %d, got %d", len(content), info.Size)
	}

	// Verify source still exists (copy, not move)
	exists, err = store.Exists(ctx, srcBucket, srcKey)
	if err != nil {
		t.Fatalf("Exists source after copy: %v", err)
	}
	if !exists {
		t.Fatal("source file should still exist after copy")
	}
}
