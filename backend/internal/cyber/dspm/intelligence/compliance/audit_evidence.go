package compliance

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// AuditEvidenceGenerator produces audit evidence packages for regulatory
// compliance, including asset inventories, gap analyses, and score trends.
type AuditEvidenceGenerator struct {
	assets AssetLister
	logger zerolog.Logger
}

// NewAuditEvidenceGenerator creates a new audit evidence generator.
func NewAuditEvidenceGenerator(assets AssetLister, logger zerolog.Logger) *AuditEvidenceGenerator {
	return &AuditEvidenceGenerator{
		assets: assets,
		logger: logger.With().Str("component", "audit_evidence_generator").Logger(),
	}
}

// Generate produces a comprehensive audit evidence report for a specific
// compliance framework, including the compliance summary, asset inventory,
// gap analysis, exception log, and historical score trend.
func (a *AuditEvidenceGenerator) Generate(
	ctx context.Context,
	tenantID uuid.UUID,
	posture *model.CompliancePosture,
	gaps []model.ComplianceGap,
) (*model.AuditReport, error) {
	a.logger.Info().
		Str("tenant_id", tenantID.String()).
		Str("framework", string(posture.Framework)).
		Msg("generating audit evidence report")

	assets, err := a.assets.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("listing assets for audit report: %w", err)
	}

	report := &model.AuditReport{
		Framework:         posture.Framework,
		TenantID:          tenantID,
		GeneratedAt:       time.Now().UTC(),
		ComplianceSummary: *posture,
	}

	// Build asset inventory.
	for _, asset := range assets {
		if !IsInScope(asset, scopeForFramework(posture.Framework)) {
			continue
		}

		encrypted := false
		if asset.EncryptedAtRest != nil && *asset.EncryptedAtRest {
			encrypted = true
		}

		accessControl := "unknown"
		if asset.AccessControlType != nil {
			accessControl = *asset.AccessControlType
		}

		entry := model.AuditAssetEntry{
			AssetID:        asset.AssetID,
			AssetName:      asset.AssetName,
			Classification: asset.DataClassification,
			PostureScore:   asset.PostureScore,
			RiskScore:      asset.RiskScore,
			Encrypted:      encrypted,
			AccessControl:  accessControl,
		}
		report.AssetInventory = append(report.AssetInventory, entry)
	}

	// Filter gaps to this framework.
	for _, gap := range gaps {
		if gap.Framework == posture.Framework {
			report.GapAnalysis = append(report.GapAnalysis, gap)
		}
	}

	// Build exception log from assets with risk exceptions in metadata.
	for _, asset := range assets {
		if asset.Metadata == nil {
			continue
		}
		if _, ok := asset.Metadata["risk_exception"]; ok {
			exception := model.AuditException{
				ExceptionID:   uuid.New(),
				AssetName:     asset.AssetName,
				Justification: extractString(asset.Metadata, "exception_justification", "Risk accepted by management"),
				ApprovedBy:    extractString(asset.Metadata, "exception_approved_by", "Unknown"),
				ExpiresAt:     extractExpiryDate(asset.Metadata),
			}
			report.ExceptionLog = append(report.ExceptionLog, exception)
		}
	}

	// Build score trend from available historical data.
	report.ScoreTrend = buildScoreTrend(posture)

	a.logger.Info().
		Str("framework", string(posture.Framework)).
		Int("assets_in_scope", len(report.AssetInventory)).
		Int("gaps", len(report.GapAnalysis)).
		Int("exceptions", len(report.ExceptionLog)).
		Msg("audit evidence report generated")

	return report, nil
}

// scopeForFramework returns the default asset scope for a framework.
func scopeForFramework(framework model.ComplianceFramework) string {
	switch framework {
	case model.FrameworkGDPR:
		return "pii"
	case model.FrameworkHIPAA:
		return "healthcare"
	case model.FrameworkPCIDSS:
		return "payment"
	case model.FrameworkSaudiPDPL:
		return "pii"
	default:
		return "all"
	}
}

// extractString gets a string from metadata with a default value.
func extractString(md map[string]interface{}, key, defaultVal string) string {
	if val, ok := md[key]; ok {
		if str, isStr := val.(string); isStr && str != "" {
			return str
		}
	}
	return defaultVal
}

// extractExpiryDate extracts an exception expiry date from metadata.
// Defaults to one year from now if not found.
func extractExpiryDate(md map[string]interface{}) time.Time {
	if val, ok := md["exception_expires_at"]; ok {
		if str, isStr := val.(string); isStr {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
			if t, err := time.Parse("2006-01-02", str); err == nil {
				return t
			}
		}
	}
	return time.Now().UTC().AddDate(1, 0, 0)
}

// buildScoreTrend creates a score trend from the posture's historical score fields.
func buildScoreTrend(posture *model.CompliancePosture) []model.AuditScorePoint {
	now := time.Now().UTC()
	var trend []model.AuditScorePoint

	// Add historical points if available.
	if posture.Score90dAgo != nil {
		trend = append(trend, model.AuditScorePoint{
			Date:  now.AddDate(0, 0, -90).Format("2006-01-02"),
			Score: *posture.Score90dAgo,
		})
	}
	if posture.Score30dAgo != nil {
		trend = append(trend, model.AuditScorePoint{
			Date:  now.AddDate(0, 0, -30).Format("2006-01-02"),
			Score: *posture.Score30dAgo,
		})
	}
	if posture.Score7dAgo != nil {
		trend = append(trend, model.AuditScorePoint{
			Date:  now.AddDate(0, 0, -7).Format("2006-01-02"),
			Score: *posture.Score7dAgo,
		})
	}

	// Add current score.
	trend = append(trend, model.AuditScorePoint{
		Date:  now.Format("2006-01-02"),
		Score: posture.OverallScore,
	})

	return trend
}
