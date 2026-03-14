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

// AuditProvider fetches recent access audit data for anomaly detection.
type AuditProvider interface {
	CountAccessLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) (int, error)
	NewRestrictedAssetsLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) ([]string, error)
	NewSourceIPsLast24h(ctx context.Context, tenantID uuid.UUID, identityID string) ([]string, error)
}

// IdentityProfileProvider provides identity profile data for baseline comparison.
type IdentityProfileProvider interface {
	GetByIdentityID(ctx context.Context, tenantID uuid.UUID, identityID string) (*model.IdentityProfile, error)
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.IdentityProfile, error)
}

// AccessAnomalyDetector integrates with DSPM access audit data to detect anomalous
// data access patterns. It compares recent behavior against each identity's baseline
// (avg_daily_access_count, frequent_assets from dspm_identity_profiles).
type AccessAnomalyDetector struct {
	auditProvider   AuditProvider
	profileProvider IdentityProfileProvider
	logger          zerolog.Logger
}

// NewAccessAnomalyDetector creates a new access anomaly detector.
func NewAccessAnomalyDetector(
	auditProvider AuditProvider,
	profileProvider IdentityProfileProvider,
	logger zerolog.Logger,
) *AccessAnomalyDetector {
	return &AccessAnomalyDetector{
		auditProvider:   auditProvider,
		profileProvider: profileProvider,
		logger:          logger.With().Str("analyzer", "access_anomaly").Logger(),
	}
}

// Detect scans all active identities for access anomalies in the last 24 hours.
// It generates findings for:
//   - volume anomaly: access count > 3× avg_daily_access_count
//   - new sensitive access: accessed a restricted asset not in frequent_assets
//   - new location: accessed from a new source IP
func (d *AccessAnomalyDetector) Detect(ctx context.Context, tenantID uuid.UUID) ([]model.AccessAnomaly, error) {
	profiles, err := d.profileProvider.ListActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var anomalies []model.AccessAnomaly

	for _, profile := range profiles {
		// Skip profiles with no baseline.
		if profile.AvgDailyAccessCount == 0 {
			continue
		}

		// Volume anomaly: access_count > 3 × avg.
		count, err := d.auditProvider.CountAccessLast24h(ctx, tenantID, profile.IdentityID)
		if err != nil {
			d.logger.Warn().Err(err).Str("identity", profile.IdentityID).Msg("failed to count access")
			continue
		}
		if count > 0 && float64(count) > 3*profile.AvgDailyAccessCount {
			deviation := float64(count) / profile.AvgDailyAccessCount
			confidence := math.Min(0.5+deviation*0.1, 0.95)
			anomalies = append(anomalies, model.AccessAnomaly{
				IdentityType: profile.IdentityType,
				IdentityID:   profile.IdentityID,
				IdentityName: profile.IdentityName,
				AnomalyType:  "volume_anomaly",
				Description:  fmt.Sprintf("Access count (%d) is %.1fx the daily average (%.1f)", count, deviation, profile.AvgDailyAccessCount),
				Severity:     volumeAnomalySeverity(deviation),
				Confidence:   confidence,
				DetectedAt:   now,
			})
		}

		// New sensitive access: accessed restricted asset not in frequent_assets.
		newRestricted, err := d.auditProvider.NewRestrictedAssetsLast24h(ctx, tenantID, profile.IdentityID)
		if err != nil {
			d.logger.Warn().Err(err).Str("identity", profile.IdentityID).Msg("failed to check new restricted access")
			continue
		}
		for _, assetName := range newRestricted {
			if !isFrequentAsset(profile.AccessPatternSummary, assetName) {
				anomalies = append(anomalies, model.AccessAnomaly{
					IdentityType: profile.IdentityType,
					IdentityID:   profile.IdentityID,
					IdentityName: profile.IdentityName,
					AnomalyType:  "new_sensitive_access",
					Description:  fmt.Sprintf("First-time access to restricted asset: %s", assetName),
					Severity:     "high",
					Confidence:   0.8,
					DetectedAt:   now,
				})
			}
		}

		// New source IP.
		newIPs, err := d.auditProvider.NewSourceIPsLast24h(ctx, tenantID, profile.IdentityID)
		if err != nil {
			d.logger.Warn().Err(err).Str("identity", profile.IdentityID).Msg("failed to check new source IPs")
			continue
		}
		for _, ip := range newIPs {
			anomalies = append(anomalies, model.AccessAnomaly{
				IdentityType: profile.IdentityType,
				IdentityID:   profile.IdentityID,
				IdentityName: profile.IdentityName,
				AnomalyType:  "new_location",
				Description:  fmt.Sprintf("Access from previously unseen source IP: %s", ip),
				Severity:     "medium",
				Confidence:   0.7,
				DetectedAt:   now,
			})
		}
	}

	return anomalies, nil
}

func volumeAnomalySeverity(deviation float64) string {
	switch {
	case deviation >= 10:
		return "critical"
	case deviation >= 5:
		return "high"
	case deviation >= 3:
		return "medium"
	default:
		return "low"
	}
}

func isFrequentAsset(patternSummary map[string]interface{}, assetName string) bool {
	if patternSummary == nil {
		return false
	}
	frequentRaw, ok := patternSummary["frequent_assets"]
	if !ok {
		return false
	}
	frequent, ok := frequentRaw.([]interface{})
	if !ok {
		return false
	}
	for _, item := range frequent {
		if m, ok := item.(map[string]interface{}); ok {
			if name, ok := m["name"].(string); ok && name == assetName {
				return true
			}
		}
	}
	return false
}
