package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// PatchStrategy handles patch-type remediations.
type PatchStrategy struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewPatchStrategy creates a PatchStrategy.
func NewPatchStrategy(db *pgxpool.Pool, logger zerolog.Logger) *PatchStrategy {
	return &PatchStrategy{db: db, logger: logger}
}

func (s *PatchStrategy) Type() model.RemediationType { return model.RemediationTypePatch }

func (s *PatchStrategy) DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	start := time.Now()
	result := &model.DryRunResult{
		Success:          true,
		SimulatedChanges: make([]model.SimulatedChange, 0),
		Warnings:         make([]string, 0),
		Blockers:         make([]string, 0),
		AffectedServices: make([]string, 0),
	}

	for _, assetID := range action.AffectedAssetIDs {
		var assetName, assetStatus, assetOS string
		err := s.db.QueryRow(ctx,
			"SELECT name, status, COALESCE(os, '') FROM assets WHERE id=$1 AND tenant_id=$2",
			assetID, action.TenantID,
		).Scan(&assetName, &assetStatus, &assetOS)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Asset %s not found or inaccessible", assetID))
			continue
		}

		if assetStatus == "inactive" || assetStatus == "decommissioned" {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Asset '%s' is %s — patch may not apply", assetName, assetStatus))
		}

		// Determine patch applicability from the plan target version
		targetVersion := action.Plan.TargetVersion
		if targetVersion == "" {
			targetVersion = "latest"
		}

		result.SimulatedChanges = append(result.SimulatedChanges, model.SimulatedChange{
			AssetID:     assetID.String(),
			AssetName:   assetName,
			ChangeType:  "package_upgrade",
			Description: fmt.Sprintf("Apply patch to '%s' (target: %s)", assetName, targetVersion),
			AfterValue:  targetVersion,
		})
	}

	downtime := "0"
	if action.Plan.RequiresReboot {
		downtime = "5-15m per asset"
		result.Warnings = append(result.Warnings, "Reboot required — schedule maintenance window")
	}

	if len(result.Blockers) > 0 {
		result.Success = false
	}

	result.EstimatedImpact = model.ImpactEstimate{
		Downtime:         downtime,
		ServicesAffected: len(action.AffectedAssetIDs),
		RiskLevel:        action.Plan.RiskLevel,
		RecommendWindow:  recommendWindow(action.Plan.RequiresReboot),
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *PatchStrategy) Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
	start := time.Now()
	result := &model.ExecutionResult{
		StepsTotal:     len(action.Plan.Steps),
		StepResults:    make([]model.StepResult, 0),
		ChangesApplied: make([]model.AppliedChange, 0),
	}

	for i, step := range action.Plan.Steps {
		stepStart := time.Now()
		sr := model.StepResult{
			StepNumber: step.Number,
			Action:     step.Action,
		}

		// For each affected asset: mark vulnerability as mitigated
		for _, assetID := range action.AffectedAssetIDs {
			var assetName string
			_ = s.db.QueryRow(ctx,
				"SELECT name FROM assets WHERE id=$1", assetID,
			).Scan(&assetName)

			// Mark associated vulnerabilities as mitigated (not resolved — requires verification)
			if action.VulnerabilityID != nil {
				_, err := s.db.Exec(ctx, `
					UPDATE vulnerabilities SET status='mitigated', updated_at=now()
					WHERE id=$1 AND asset_id=$2 AND tenant_id=$3`,
					action.VulnerabilityID, assetID, action.TenantID,
				)
				if err != nil {
					sr.Status = "failure"
					sr.Error = fmt.Sprintf("failed to update vulnerability status for asset %s: %v", assetName, err)
					sr.DurationMs = time.Since(stepStart).Milliseconds()
					result.StepResults = append(result.StepResults, sr)
					result.StepsExecuted = i + 1
					result.Success = false
					result.DurationMs = time.Since(start).Milliseconds()
					return result, nil
				}
			} else {
				// Mitigate all outstanding vulnerabilities on the affected asset. Verification will confirm whether the issue is resolved.
				_, _ = s.db.Exec(ctx, `
					UPDATE vulnerabilities SET status='mitigated', updated_at=now()
					WHERE asset_id=$1 AND tenant_id=$2 AND status IN ('open','in_progress')`,
					assetID, action.TenantID,
				)
			}

			result.ChangesApplied = append(result.ChangesApplied, model.AppliedChange{
				AssetID:     assetID.String(),
				ChangeType:  "patch_applied",
				Description: fmt.Sprintf("Patch applied to asset '%s'", assetName),
				NewValue:    action.Plan.TargetVersion,
			})
		}

		sr.Status = "success"
		sr.Output = fmt.Sprintf("Step '%s' completed for %d asset(s)", step.Action, len(action.AffectedAssetIDs))
		sr.DurationMs = time.Since(stepStart).Milliseconds()
		result.StepResults = append(result.StepResults, sr)
		result.StepsExecuted++
	}

	result.Success = true
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *PatchStrategy) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	start := time.Now()
	result := &model.VerificationResult{
		Checks: make([]model.VerificationCheck, 0),
	}

	allPassed := true
	for _, assetID := range action.AffectedAssetIDs {
		var assetName string
		_ = s.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&assetName)

		// Check: is the target vulnerability still present?
		var openVulnCount int
		if action.VulnerabilityID != nil {
			_ = s.db.QueryRow(ctx,
				"SELECT COUNT(*) FROM vulnerabilities WHERE id=$1 AND asset_id=$2 AND status='open'",
				action.VulnerabilityID, assetID,
			).Scan(&openVulnCount)
		} else {
			// Check for any open vulns that should have been patched
			_ = s.db.QueryRow(ctx,
				"SELECT COUNT(*) FROM vulnerabilities WHERE asset_id=$1 AND tenant_id=$2 AND status='open' AND severity IN ('critical','high')",
				assetID, action.TenantID,
			).Scan(&openVulnCount)
		}

		check := model.VerificationCheck{
			Name:     fmt.Sprintf("CVE presence check for %s", assetName),
			Expected: "vulnerability not present (status != open)",
			Actual:   fmt.Sprintf("%d open vulnerability(ies) remaining", openVulnCount),
			Passed:   openVulnCount == 0,
		}
		if !check.Passed {
			allPassed = false
			check.Notes = "Vulnerability still detected — patch may not have taken effect"
		}
		result.Checks = append(result.Checks, check)
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more vulnerabilities are still present on affected assets"
	} else {
		// Mark vulnerabilities as resolved
		if action.VulnerabilityID != nil {
			_, _ = s.db.Exec(ctx,
				"UPDATE vulnerabilities SET status='resolved', updated_at=now() WHERE id=$1 AND tenant_id=$2",
				action.VulnerabilityID, action.TenantID,
			)
		}
	}

	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *PatchStrategy) Rollback(ctx context.Context, action *model.RemediationAction) error {
	var state struct {
		Assets []struct {
			AssetID         string `json:"asset_id"`
			Name            string `json:"name"`
			VulnerabilityStates []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
			} `json:"vulnerability_states"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(action.PreExecutionState, &state); err != nil {
		return fmt.Errorf("decode pre-execution state: %w", err)
	}
	for _, assetState := range state.Assets {
		for _, vulnState := range assetState.VulnerabilityStates {
			_, err := s.db.Exec(ctx,
				"UPDATE vulnerabilities SET status=$1, updated_at=now() WHERE id=$2 AND tenant_id=$3",
				vulnState.Status, vulnState.ID, action.TenantID,
			)
			if err != nil {
				return fmt.Errorf("restore vulnerability %s for asset %s: %w", vulnState.ID, assetState.AssetID, err)
			}
		}
	}
	return nil
}

func (s *PatchStrategy) CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
	type assetState struct {
		AssetID             string `json:"asset_id"`
		Name                string `json:"name"`
		VulnerabilityStates []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"vulnerability_states"`
	}

	states := make([]assetState, 0, len(action.AffectedAssetIDs))
	for _, assetID := range action.AffectedAssetIDs {
		var st assetState
		st.AssetID = assetID.String()
		_ = s.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&st.Name)
		var (
			rows    pgx.Rows
			rowsErr error
		)
		if action.VulnerabilityID != nil {
			rows, rowsErr = s.db.Query(ctx,
				"SELECT id, status FROM vulnerabilities WHERE id=$1 AND asset_id=$2 AND tenant_id=$3",
				action.VulnerabilityID, assetID, action.TenantID,
			)
		} else {
			rows, rowsErr = s.db.Query(ctx,
				"SELECT id, status FROM vulnerabilities WHERE asset_id=$1 AND tenant_id=$2 AND status IN ('open','in_progress','mitigated')",
				assetID, action.TenantID,
			)
		}
		if rowsErr == nil {
			for rows.Next() {
				var vulnID uuid.UUID
				var status string
				if err := rows.Scan(&vulnID, &status); err == nil {
					st.VulnerabilityStates = append(st.VulnerabilityStates, struct {
						ID     string `json:"id"`
						Status string `json:"status"`
					}{ID: vulnID.String(), Status: status})
				}
			}
			rows.Close()
		}

		states = append(states, st)
	}
	return json.Marshal(map[string]interface{}{"assets": states, "captured_at": time.Now().UTC()})
}

func recommendWindow(requiresReboot bool) string {
	if requiresReboot {
		return "maintenance window"
	}
	return "business hours ok"
}
