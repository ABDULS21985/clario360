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

// ConfigStrategy handles configuration-change remediations.
type ConfigStrategy struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewConfigStrategy creates a ConfigStrategy.
func NewConfigStrategy(db *pgxpool.Pool, logger zerolog.Logger) *ConfigStrategy {
	return &ConfigStrategy{db: db, logger: logger}
}

func (s *ConfigStrategy) Type() model.RemediationType { return model.RemediationTypeConfigChange }

func (s *ConfigStrategy) DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	start := time.Now()
	result := &model.DryRunResult{
		Success:          true,
		SimulatedChanges: make([]model.SimulatedChange, 0),
		Warnings:         make([]string, 0),
		Blockers:         make([]string, 0),
		AffectedServices: make([]string, 0),
	}

	if len(action.Plan.TargetConfig) == 0 {
		result.Blockers = append(result.Blockers, "no target configuration specified in plan.target_config")
		result.Success = false
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	for _, assetID := range action.AffectedAssetIDs {
		var assetName string
		_ = s.db.QueryRow(ctx, "SELECT name FROM assets WHERE id=$1", assetID).Scan(&assetName)

		for key, newVal := range action.Plan.TargetConfig {
			result.SimulatedChanges = append(result.SimulatedChanges, model.SimulatedChange{
				AssetID:     assetID.String(),
				AssetName:   assetName,
				ChangeType:  "config_change",
				Description: fmt.Sprintf("Set %s on '%s'", key, assetName),
				AfterValue:  fmt.Sprintf("%v", newVal),
			})
		}
	}

	result.EstimatedImpact = model.ImpactEstimate{
		Downtime:        action.Plan.EstimatedDowntime,
		RiskLevel:       action.Plan.RiskLevel,
		RecommendWindow: "business hours ok",
	}
	if action.Plan.RequiresReboot {
		result.EstimatedImpact.RecommendWindow = "maintenance window"
		result.Warnings = append(result.Warnings, "Configuration change requires reboot")
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *ConfigStrategy) Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
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
			Action:     fmt.Sprintf("apply config to %s", assetName),
		}

		// Build the JSONB patch from target_config
		configJSON, err := json.Marshal(action.Plan.TargetConfig)
		if err != nil {
			sr.Status = "failure"
			sr.Error = fmt.Sprintf("marshal config for '%s': %v", assetName, err)
			result.StepResults = append(result.StepResults, sr)
			result.Success = false
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}

		_, err = s.db.Exec(ctx, `
			UPDATE assets
			SET metadata = metadata || $1::jsonb, updated_at = now()
			WHERE id=$2 AND tenant_id=$3`,
			configJSON, assetID, action.TenantID,
		)
		if err != nil {
			sr.Status = "failure"
			sr.Error = fmt.Sprintf("apply config to '%s': %v", assetName, err)
			result.StepResults = append(result.StepResults, sr)
			result.Success = false
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}

		sr.Status = "success"
		sr.Output = fmt.Sprintf("Config applied to '%s'", assetName)
		sr.DurationMs = time.Since(stepStart).Milliseconds()
		result.StepResults = append(result.StepResults, sr)
		result.StepsExecuted++
		result.ChangesApplied = append(result.ChangesApplied, model.AppliedChange{
			AssetID:     assetID.String(),
			ChangeType:  "config_applied",
			Description: fmt.Sprintf("Configuration change applied to '%s'", assetName),
		})
	}

	result.Success = true
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *ConfigStrategy) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	start := time.Now()
	result := &model.VerificationResult{
		Checks: make([]model.VerificationCheck, 0),
	}

	allPassed := true
	for _, assetID := range action.AffectedAssetIDs {
		var assetName string
		var metaJSON []byte
		_ = s.db.QueryRow(ctx, "SELECT name, metadata FROM assets WHERE id=$1", assetID).Scan(&assetName, &metaJSON)

		var current map[string]interface{}
		_ = json.Unmarshal(metaJSON, &current)

		for key, expected := range action.Plan.TargetConfig {
			actual, exists := current[key]
			passed := exists && fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
			if !passed {
				allPassed = false
			}
			result.Checks = append(result.Checks, model.VerificationCheck{
				Name:     fmt.Sprintf("Config check: %s on '%s'", key, assetName),
				Expected: fmt.Sprintf("%v", expected),
				Actual:   fmt.Sprintf("%v", actual),
				Passed:   passed,
			})
		}
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "One or more configuration keys do not match the desired state"
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *ConfigStrategy) Rollback(ctx context.Context, action *model.RemediationAction) error {
	if action.PreExecutionState == nil {
		return fmt.Errorf("no pre-execution state captured for config rollback")
	}

	type preState struct {
		Assets []struct {
			AssetID string                 `json:"asset_id"`
			Config  map[string]interface{} `json:"config"`
		} `json:"assets"`
	}

	var state preState
	if err := json.Unmarshal(action.PreExecutionState, &state); err != nil {
		return fmt.Errorf("unmarshal pre-execution state: %w", err)
	}

	for _, assetState := range state.Assets {
		if len(assetState.Config) == 0 {
			continue
		}
		restoreJSON, err := json.Marshal(assetState.Config)
		if err != nil {
			return fmt.Errorf("marshal restore config: %w", err)
		}
		_, err = s.db.Exec(ctx, `
			UPDATE assets SET metadata = metadata || $1::jsonb, updated_at = now()
			WHERE id=$2 AND tenant_id=$3`,
			restoreJSON, assetState.AssetID, action.TenantID,
		)
		if err != nil {
			return fmt.Errorf("restore config for asset %s: %w", assetState.AssetID, err)
		}
	}
	return nil
}

func (s *ConfigStrategy) CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
	type assetState struct {
		AssetID string                 `json:"asset_id"`
		Name    string                 `json:"name"`
		Config  map[string]interface{} `json:"config"`
	}

	states := make([]assetState, 0, len(action.AffectedAssetIDs))
	for _, assetID := range action.AffectedAssetIDs {
		st := assetState{AssetID: assetID.String()}
		var metaJSON []byte
		_ = s.db.QueryRow(ctx, "SELECT name, metadata FROM assets WHERE id=$1", assetID).Scan(&st.Name, &metaJSON)
		_ = json.Unmarshal(metaJSON, &st.Config)

		// Capture only the keys that will be changed
		if len(action.Plan.TargetConfig) > 0 {
			relevantConfig := make(map[string]interface{})
			for key := range action.Plan.TargetConfig {
				if v, ok := st.Config[key]; ok {
					relevantConfig[key] = v
				}
			}
			st.Config = relevantConfig
		}
		states = append(states, st)
	}
	return json.Marshal(map[string]interface{}{"assets": states, "captured_at": time.Now().UTC()})
}
