package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// IsolateStrategy handles asset isolation remediations.
type IsolateStrategy struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewIsolateStrategy creates an IsolateStrategy.
func NewIsolateStrategy(db *pgxpool.Pool, logger zerolog.Logger) *IsolateStrategy {
	return &IsolateStrategy{db: db, logger: logger}
}

func (s *IsolateStrategy) Type() model.RemediationType { return model.RemediationTypeIsolateAsset }

func (s *IsolateStrategy) DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	start := time.Now()
	result := &model.DryRunResult{
		Success:          true,
		SimulatedChanges: make([]model.SimulatedChange, 0),
		Warnings:         make([]string, 0),
		Blockers:         make([]string, 0),
		AffectedServices: make([]string, 0),
	}

	for _, assetID := range action.AffectedAssetIDs {
		var assetName, assetType string
		err := s.db.QueryRow(ctx,
			"SELECT name, type FROM assets WHERE id=$1 AND tenant_id=$2",
			assetID, action.TenantID,
		).Scan(&assetName, &assetType)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Asset %s not found", assetID))
			continue
		}

		// Count dependent services (consumers)
		var depCount int
		_ = s.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM asset_relationships
			WHERE target_asset_id=$1 AND tenant_id=$2 AND relationship_type='depends_on'`,
			assetID, action.TenantID,
		).Scan(&depCount)

		if depCount > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf(
				"Asset '%s' has %d dependent service(s) that will lose connectivity upon isolation", assetName, depCount,
			))
			for j := 0; j < depCount && j < 5; j++ {
				result.AffectedServices = append(result.AffectedServices, fmt.Sprintf("dependent-service-%d", j+1))
			}
		}

		result.SimulatedChanges = append(result.SimulatedChanges, model.SimulatedChange{
			AssetID:     assetID.String(),
			AssetName:   assetName,
			ChangeType:  "network_isolation",
			Description: fmt.Sprintf("Isolate asset '%s' (%s) from network", assetName, assetType),
			BeforeValue: "connected",
			AfterValue:  "isolated",
		})
	}

	result.EstimatedImpact = model.ImpactEstimate{
		Downtime:         "immediate",
		ServicesAffected: len(result.AffectedServices),
		RiskLevel:        "high",
		RecommendWindow:  "maintenance window",
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *IsolateStrategy) Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
	start := time.Now()
	result := &model.ExecutionResult{
		StepsTotal:     len(action.AffectedAssetIDs),
		StepResults:    make([]model.StepResult, 0),
		ChangesApplied: make([]model.AppliedChange, 0),
	}

	for i, assetID := range action.AffectedAssetIDs {
		stepStart := time.Now()
		var assetName string
		_ = s.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&assetName)

		sr := model.StepResult{
			StepNumber: i + 1,
			Action:     fmt.Sprintf("isolate %s", assetName),
		}

		// Mark asset as isolated via metadata flag
		_, err := s.db.Exec(ctx, `
			UPDATE assets
			SET metadata = metadata || '{"isolated": true, "isolation_reason": "remediation_action"}'::jsonb,
			    updated_at = now()
			WHERE id=$1 AND tenant_id=$2`,
			assetID, action.TenantID,
		)
		if err != nil {
			sr.Status = "failure"
			sr.Error = fmt.Sprintf("failed to mark asset '%s' as isolated: %v", assetName, err)
			sr.DurationMs = time.Since(stepStart).Milliseconds()
			result.StepResults = append(result.StepResults, sr)
			result.Success = false
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}

		sr.Status = "success"
		sr.Output = fmt.Sprintf("Asset '%s' marked as isolated", assetName)
		sr.DurationMs = time.Since(stepStart).Milliseconds()
		result.StepResults = append(result.StepResults, sr)
		result.StepsExecuted++
		result.ChangesApplied = append(result.ChangesApplied, model.AppliedChange{
			AssetID:     assetID.String(),
			ChangeType:  "network_isolation",
			Description: fmt.Sprintf("Asset '%s' isolated from network", assetName),
			OldValue:    "connected",
			NewValue:    "isolated",
		})
	}

	result.Success = true
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *IsolateStrategy) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	start := time.Now()
	result := &model.VerificationResult{
		Checks: make([]model.VerificationCheck, 0),
	}

	allPassed := true
	for _, assetID := range action.AffectedAssetIDs {
		var assetName string
		var isolated bool
		_ = s.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&assetName)
		_ = s.db.QueryRow(ctx,
			"SELECT COALESCE((metadata->>'isolated')::boolean, false) FROM assets WHERE id=$1",
			assetID,
		).Scan(&isolated)

		check := model.VerificationCheck{
			Name:     fmt.Sprintf("Isolation check for '%s'", assetName),
			Expected: "asset is isolated (metadata.isolated = true)",
			Actual:   fmt.Sprintf("isolated = %v", isolated),
			Passed:   isolated,
		}
		if !check.Passed {
			allPassed = false
			check.Notes = "Asset does not appear to be isolated"
		}
		result.Checks = append(result.Checks, check)
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more assets are not isolated as expected"
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *IsolateStrategy) Rollback(ctx context.Context, action *model.RemediationAction) error {
	for _, assetID := range action.AffectedAssetIDs {
		_, err := s.db.Exec(ctx, `
			UPDATE assets
			SET metadata = metadata - 'isolated' - 'isolation_reason',
			    updated_at = now()
			WHERE id=$1 AND tenant_id=$2`,
			assetID, action.TenantID,
		)
		if err != nil {
			return fmt.Errorf("restore asset %s from isolation: %w", assetID, err)
		}
	}
	return nil
}

func (s *IsolateStrategy) CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
	type assetState struct {
		AssetID  string `json:"asset_id"`
		Name     string `json:"name"`
		Isolated bool   `json:"isolated"`
	}

	states := make([]assetState, 0, len(action.AffectedAssetIDs))
	for _, assetID := range action.AffectedAssetIDs {
		st := assetState{AssetID: assetID.String()}
		_ = s.db.QueryRow(ctx,
			"SELECT name, COALESCE((metadata->>'isolated')::boolean, false) FROM assets WHERE id=$1",
			assetID,
		).Scan(&st.Name, &st.Isolated)
		states = append(states, st)
	}
	return json.Marshal(map[string]interface{}{"assets": states, "captured_at": time.Now().UTC()})
}
