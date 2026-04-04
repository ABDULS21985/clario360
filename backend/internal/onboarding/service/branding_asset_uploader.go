package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	filedto "github.com/clario360/platform/internal/filemanager/dto"
	filemodel "github.com/clario360/platform/internal/filemanager/model"
	filesvc "github.com/clario360/platform/internal/filemanager/service"
	iammodel "github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/pkg/storage"
)

const (
	maxBrandingLogoBytes = 2 << 20
)

type BrandingAssetUploader interface {
	UploadLogo(ctx context.Context, req BrandingLogoUploadRequest) (uuid.UUID, error)
}

type BrandingLogoUploadRequest struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	File        io.Reader
	Filename    string
	ContentType string
	IPAddress   string
	UserAgent   string
	Size        int64
}

type fileBrandingAssetUploader struct {
	fileSvc *filesvc.FileService
}

func NewBrandingAssetUploader(fileSvc *filesvc.FileService) BrandingAssetUploader {
	if fileSvc == nil {
		return nil
	}
	return &fileBrandingAssetUploader{fileSvc: fileSvc}
}

func (u *fileBrandingAssetUploader) UploadLogo(ctx context.Context, req BrandingLogoUploadRequest) (uuid.UUID, error) {
	if req.File == nil {
		return uuid.Nil, fmt.Errorf("logo file is required: %w", iammodel.ErrValidation)
	}
	if req.Size > maxBrandingLogoBytes {
		return uuid.Nil, fmt.Errorf("logo file exceeds the 2MB limit: %w", iammodel.ErrValidation)
	}

	content, err := io.ReadAll(io.LimitReader(req.File, maxBrandingLogoBytes+1))
	if err != nil {
		return uuid.Nil, fmt.Errorf("read logo file: %w", err)
	}
	if len(content) == 0 {
		return uuid.Nil, fmt.Errorf("logo file is empty: %w", iammodel.ErrValidation)
	}
	if len(content) > maxBrandingLogoBytes {
		return uuid.Nil, fmt.Errorf("logo file exceeds the 2MB limit: %w", iammodel.ErrValidation)
	}

	validation, err := storage.ValidateContent(bytes.NewReader(content), req.ContentType, filemodel.SuiteVisus)
	if err != nil {
		return uuid.Nil, fmt.Errorf("validate logo content: %w", err)
	}
	if validation.Blocked {
		return uuid.Nil, fmt.Errorf("logo file type is blocked: %w", iammodel.ErrValidation)
	}

	detectedType := normalizeBrandingContentType(validation.DetectedType)
	if !allowedBrandingContentType(detectedType) {
		return uuid.Nil, fmt.Errorf("logo must be a PNG or SVG image: %w", iammodel.ErrValidation)
	}

	record, err := u.fileSvc.Upload(
		ctx,
		&filedto.UploadRequest{
			Suite:           filemodel.SuiteVisus,
			EntityType:      "tenant_branding",
			EntityID:        req.TenantID.String(),
			Tags:            []string{"onboarding", "branding", "logo"},
			Encrypt:         false,
			LifecyclePolicy: filemodel.LifecycleStandard,
			DedupCheck:      false,
		},
		bytes.NewReader(content),
		int64(len(content)),
		req.Filename,
		detectedType,
		req.TenantID.String(),
		req.UserID.String(),
		req.IPAddress,
		req.UserAgent,
	)
	if err != nil {
		var fileErr *filesvc.ServiceError
		if errors.As(err, &fileErr) && fileErr.Code < 500 {
			return uuid.Nil, fmt.Errorf("%s: %w", fileErr.Message, iammodel.ErrValidation)
		}
		return uuid.Nil, err
	}

	fileID, err := uuid.Parse(record.ID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parse uploaded logo ID: %w", err)
	}
	return fileID, nil
}

func allowedBrandingContentType(contentType string) bool {
	switch normalizeBrandingContentType(contentType) {
	case "image/png", "image/svg+xml":
		return true
	default:
		return false
	}
}

func normalizeBrandingContentType(contentType string) string {
	contentType = strings.TrimSpace(strings.ToLower(contentType))
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = strings.TrimSpace(contentType[:idx])
	}
	return contentType
}
