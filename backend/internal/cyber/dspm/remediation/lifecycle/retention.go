package lifecycle

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
)

// AssetLister abstracts fetching all active DSPM data assets for a tenant.
type AssetLister interface {
	ListAllActive(ctx context.Context, tenantID uuid.UUID) ([]*cybermodel.DSPMDataAsset, error)
}

// RetentionViolation records a single data asset that exceeds its retention window.
type RetentionViolation struct {
	AssetID        uuid.UUID `json:"asset_id"`
	AssetName      string    `json:"asset_name"`
	Classification string    `json:"classification"`
	DaysOverdue    int       `json:"days_overdue"`
	Severity       string    `json:"severity"`
}

// RetentionEnforcer evaluates data assets against retention policies and
// identifies assets that have exceeded their maximum retention period.
type RetentionEnforcer struct {
	assetLister AssetLister
	logger      zerolog.Logger
}

// NewRetentionEnforcer constructs a RetentionEnforcer with the required dependencies.
func NewRetentionEnforcer(assetLister AssetLister, logger zerolog.Logger) *RetentionEnforcer {
	return &RetentionEnforcer{
		assetLister: assetLister,
		logger:      logger.With().Str("component", "retention_enforcer").Logger(),
	}
}

// Evaluate scans all active data assets for the given tenant and returns
// violations for any asset whose age exceeds maxDays and whose classification
// falls within classificationScope. If classificationScope is empty, all
// classifications are in scope.
func (re *RetentionEnforcer) Evaluate(ctx context.Context, tenantID uuid.UUID, maxDays int, classificationScope []string) ([]RetentionViolation, error) {
	if maxDays <= 0 {
		return nil, fmt.Errorf("retention enforcer: maxDays must be positive, got %d", maxDays)
	}

	assets, err := re.assetLister.ListAllActive(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("retention enforcer: list assets: %w", err)
	}

	re.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("asset_count", len(assets)).
		Int("max_days", maxDays).
		Int("scope_count", len(classificationScope)).
		Msg("evaluating retention policy")

	now := time.Now().UTC()
	scopeSet := buildScopeSet(classificationScope)
	var violations []RetentionViolation

	for _, asset := range assets {

		if len(scopeSet) > 0 {
			if _, ok := scopeSet[strings.ToLower(asset.DataClassification)]; !ok {
				continue
			}
		}

		ageDays := int(now.Sub(asset.CreatedAt).Hours() / 24)
		daysOverdue := ageDays - maxDays

		if daysOverdue <= 0 {
			continue
		}

		violation := RetentionViolation{
			AssetID:        asset.ID,
			AssetName:      asset.AssetName,
			Classification: asset.DataClassification,
			DaysOverdue:    daysOverdue,
			Severity:       overdueSeverity(daysOverdue),
		}

		violations = append(violations, violation)
	}

	re.logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("violations_found", len(violations)).
		Msg("retention evaluation complete")

	return violations, nil
}

// overdueSeverity maps the number of days overdue to a severity level.
//
//	>= 180 days → critical
//	>= 90 days  → high
//	>= 30 days  → medium
//	< 30 days   → low
func overdueSeverity(daysOverdue int) string {
	switch {
	case daysOverdue >= 180:
		return "critical"
	case daysOverdue >= 90:
		return "high"
	case daysOverdue >= 30:
		return "medium"
	default:
		return "low"
	}
}

// buildScopeSet creates a lookup set from a classification scope slice,
// normalising values to lowercase for case-insensitive matching.
func buildScopeSet(scope []string) map[string]struct{} {
	if len(scope) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(scope))
	for _, s := range scope {
		set[strings.ToLower(s)] = struct{}{}
	}
	return set
}
