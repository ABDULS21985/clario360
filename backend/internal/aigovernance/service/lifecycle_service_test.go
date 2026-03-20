package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmodel "github.com/clario360/platform/internal/aigovernance/model"
)

func TestPromoteDevToStaging(t *testing.T) {
	artifact := map[string]any{"weights": map[string]float64{"risk": 0.5}}
	hash, err := aigovernance.HashJSON(artifact)
	if err != nil {
		t.Fatalf("HashJSON() error = %v", err)
	}
	version := &aigovmodel.ModelVersion{
		ID:             uuid.New(),
		TenantID:       uuid.New(),
		ModelID:        uuid.New(),
		ModelSlug:      "cyber-risk-scorer",
		ModelRiskTier:  aigovmodel.RiskTierHigh,
		Status:         aigovmodel.VersionStatusDevelopment,
		Description:    "valid artifact",
		ArtifactConfig: aigovernance.MustJSON(artifact),
		ArtifactHash:   hash,
	}

	service := &LifecycleService{}
	next, err := service.nextStatus(context.Background(), version, nil, false)
	if err != nil {
		t.Fatalf("nextStatus() error = %v", err)
	}
	if next != aigovmodel.VersionStatusStaging {
		t.Fatalf("nextStatus() = %s, want staging", next)
	}
}

func TestPromoteInvalidTransition(t *testing.T) {
	service := &LifecycleService{}
	version := &aigovmodel.ModelVersion{Status: aigovmodel.VersionStatusProduction}
	if _, err := service.nextStatus(context.Background(), version, nil, false); err == nil {
		t.Fatal("expected invalid transition error")
	}
}

func TestValidateArtifactHashMismatch(t *testing.T) {
	version := &aigovmodel.ModelVersion{
		Status:         aigovmodel.VersionStatusDevelopment,
		Description:    "broken artifact",
		ArtifactConfig: aigovernance.MustJSON(map[string]any{"threshold": 3}),
		ArtifactHash:   "bad-hash",
	}
	if err := validateArtifact(version); err == nil {
		t.Fatal("expected artifact hash validation error")
	}
}

func TestValidateFailureTransition(t *testing.T) {
	tests := []struct {
		name    string
		status  aigovmodel.VersionStatus
		wantErr bool
	}{
		{name: "development", status: aigovmodel.VersionStatusDevelopment},
		{name: "staging", status: aigovmodel.VersionStatusStaging},
		{name: "shadow", status: aigovmodel.VersionStatusShadow},
		{name: "production", status: aigovmodel.VersionStatusProduction, wantErr: true},
		{name: "retired", status: aigovmodel.VersionStatusRetired, wantErr: true},
		{name: "failed", status: aigovmodel.VersionStatusFailed, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFailureTransition(&aigovmodel.ModelVersion{
				VersionNumber: 7,
				Status:        tt.status,
			})
			if tt.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
