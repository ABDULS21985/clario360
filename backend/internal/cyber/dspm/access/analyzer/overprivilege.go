package analyzer

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// MappingProvider loads access mappings for analysis.
type MappingProvider interface {
	ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.AccessMapping, error)
}

// OverprivilegeAnalyzer detects identities with permissions they never exercise.
// An identity is overprivileged if it HAS a permission but HASN'T used it in the
// configured lookback window (90 days by default).
type OverprivilegeAnalyzer struct {
	repo   MappingProvider
	logger zerolog.Logger
}

// NewOverprivilegeAnalyzer creates a new overprivilege detector.
func NewOverprivilegeAnalyzer(repo MappingProvider, logger zerolog.Logger) *OverprivilegeAnalyzer {
	return &OverprivilegeAnalyzer{
		repo:   repo,
		logger: logger.With().Str("analyzer", "overprivilege").Logger(),
	}
}

// Analyze detects all overprivileged access mappings for a tenant.
// Severity is determined by permission type and data classification:
//   - write/admin/delete/full_control on confidential/restricted → high
//   - read-only on internal → low
//   - everything else → medium
//
// Confidence is based on whether the identity is otherwise active:
//   - Identity active (other permissions used) but THIS one unused → 0.9
//   - Identity completely inactive (all permissions unused) → 0.5
//
// MITRE: T1078.004 (Valid Accounts: Cloud Accounts)
func (a *OverprivilegeAnalyzer) Analyze(ctx context.Context, tenantID uuid.UUID) ([]model.OverprivilegeResult, error) {
	mappings, err := a.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	// Build identity activity map: has this identity used ANY permission?
	identityActive := make(map[string]bool)
	for _, m := range mappings {
		key := m.IdentityType + "|" + m.IdentityID
		if m.UsageCount90d > 0 {
			identityActive[key] = true
		}
	}

	var results []model.OverprivilegeResult
	for _, m := range mappings {
		if m.UsageCount90d > 0 {
			continue // Permission is actively used → not overprivileged.
		}

		identityKey := m.IdentityType + "|" + m.IdentityID
		isActive := identityActive[identityKey]

		severity := overprivilegeSeverity(m.PermissionType, m.DataClassification)
		confidence := 0.5
		if isActive {
			confidence = 0.9 // Strong signal: identity is active but skips this permission.
		}

		daysUnused := 0
		if m.LastUsedAt != nil {
			daysUnused = int(math.Ceil(time.Since(*m.LastUsedAt).Hours() / 24))
		} else {
			daysUnused = int(math.Ceil(time.Since(m.DiscoveredAt).Hours() / 24))
		}

		recommendation := fmt.Sprintf(
			"Revoke %s on %s — unused for %d days",
			m.PermissionType, m.DataAssetName, daysUnused,
		)

		results = append(results, model.OverprivilegeResult{
			MappingID:          m.ID,
			IdentityType:       m.IdentityType,
			IdentityID:         m.IdentityID,
			IdentityName:       m.IdentityName,
			DataAssetID:        m.DataAssetID,
			DataAssetName:      m.DataAssetName,
			DataClassification: m.DataClassification,
			PermissionType:     m.PermissionType,
			PermissionSource:   m.PermissionSource,
			UsageCount90d:      m.UsageCount90d,
			LastUsedAt:         m.LastUsedAt,
			Severity:           severity,
			Confidence:         confidence,
			Recommendation:     recommendation,
			DaysUnused:         daysUnused,
		})
	}

	return results, nil
}

func overprivilegeSeverity(permType, classification string) string {
	isHighPerm := permType == "write" || permType == "admin" || permType == "delete" || permType == "full_control" || permType == "alter"
	isSensitive := classification == "confidential" || classification == "restricted"

	if isHighPerm && isSensitive {
		return "high"
	}
	if permType == "read" && classification == "internal" {
		return "low"
	}
	return "medium"
}
