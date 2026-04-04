package playbook

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// Validator performs pre-execution validation of playbooks against remediation targets.
// It produces a DryRunResult without making any actual changes.
type Validator struct {
	registry *Registry
	logger   zerolog.Logger
}

// NewValidator creates a Validator backed by the given playbook registry.
func NewValidator(registry *Registry, logger zerolog.Logger) *Validator {
	return &Validator{
		registry: registry,
		logger:   logger.With().Str("component", "playbook_validator").Logger(),
	}
}

// assetRelatedFindingTypes returns the set of finding types that require a target asset.
func assetRelatedFindingTypes() map[model.FindingType]bool {
	return map[model.FindingType]bool{
		model.FindingEncryptionMissing:    true,
		model.FindingExposureRisk:         true,
		model.FindingShadowCopy:           true,
		model.FindingPIIUnprotected:       true,
		model.FindingClassificationDrift:  true,
		model.FindingRetentionExpired:     true,
		model.FindingPostureGap:           true,
	}
}

// identityRelatedFindingTypes returns the set of finding types that benefit from an identity target.
func identityRelatedFindingTypes() map[model.FindingType]bool {
	return map[model.FindingType]bool{
		model.FindingOverprivilegedAccess:  true,
		model.FindingStaleAccess:           true,
		model.FindingBlastRadiusExcessive:  true,
	}
}

// DryRun validates a playbook against a remediation target and estimates the impact
// without executing any steps. It checks that the playbook exists, validates the target
// parameters, and computes an estimated risk reduction.
func (v *Validator) DryRun(ctx context.Context, playbookID string, assetID *uuid.UUID, identityID string) (*model.DryRunResult, error) {
	v.logger.Info().
		Str("playbook_id", playbookID).
		Str("identity_id", identityID).
		Msg("performing dry run validation")

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	result := &model.DryRunResult{
		Valid:  true,
		Issues: make([]string, 0),
	}

	// Check playbook existence.
	pb, ok := v.registry.Get(playbookID)
	if !ok {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("playbook %q not found in registry", playbookID))
		return result, nil
	}

	// Validate that asset-related playbooks have a target asset specified.
	if assetRelatedFindingTypes()[pb.FindingType] && assetID == nil {
		result.Valid = false
		result.Issues = append(result.Issues, fmt.Sprintf("playbook %q targets finding type %q which requires a data asset ID", playbookID, pb.FindingType))
	}

	// Validate that identity-related playbooks have a target identity specified.
	if identityRelatedFindingTypes()[pb.FindingType] && identityID == "" {
		result.Issues = append(result.Issues, fmt.Sprintf("playbook %q targets finding type %q; specifying an identity_id improves remediation accuracy", playbookID, pb.FindingType))
		// This is a warning, not a hard failure: identity-related playbooks can still run at asset scope.
	}

	// Validate step configuration integrity.
	if len(pb.Steps) == 0 {
		result.Valid = false
		result.Issues = append(result.Issues, "playbook has no steps defined")
	}

	for i, step := range pb.Steps {
		if step.Action == "" {
			result.Valid = false
			result.Issues = append(result.Issues, fmt.Sprintf("step %d (%s) has no action defined", i+1, step.ID))
		}
		if step.Timeout <= 0 {
			result.Issues = append(result.Issues, fmt.Sprintf("step %d (%s) has no timeout configured; default will apply", i+1, step.ID))
		}
	}

	// Check for approval requirements.
	if pb.RequiresApproval {
		result.Issues = append(result.Issues, "playbook requires manual approval before execution can proceed")
	}

	// Estimate affected scope.
	if assetID != nil {
		result.AssetsAffected = 1
	}
	if identityID != "" {
		result.IdentitiesAffected = 1
	}

	// For asset-related playbooks without an explicit identity target,
	// estimate the number of identities that will be affected.
	if assetID != nil && identityID == "" {
		result.IdentitiesAffected = estimateAffectedIdentities(pb)
	}

	// Estimate risk reduction based on playbook characteristics.
	result.EstimatedRiskReduction = estimateRiskReduction(pb)

	v.logger.Info().
		Str("playbook_id", playbookID).
		Bool("valid", result.Valid).
		Int("issues", len(result.Issues)).
		Float64("estimated_risk_reduction", result.EstimatedRiskReduction).
		Msg("dry run validation completed")

	return result, nil
}

// estimateRiskReduction calculates an estimated risk reduction percentage based on
// the playbook's finding type, step count, and the nature of the remediation actions.
func estimateRiskReduction(pb *model.Playbook) float64 {
	// Base reduction by finding type severity.
	var base float64
	switch pb.FindingType {
	case model.FindingEncryptionMissing:
		base = 35.0
	case model.FindingExposureRisk:
		base = 40.0
	case model.FindingPIIUnprotected:
		base = 38.0
	case model.FindingShadowCopy:
		base = 30.0
	case model.FindingOverprivilegedAccess:
		base = 28.0
	case model.FindingBlastRadiusExcessive:
		base = 32.0
	case model.FindingStaleAccess:
		base = 22.0
	case model.FindingClassificationDrift:
		base = 18.0
	case model.FindingRetentionExpired:
		base = 15.0
	case model.FindingPostureGap:
		base = 12.0
	default:
		base = 10.0
	}

	// Bonus for each step that adds defense-in-depth, up to a cap.
	stepBonus := 0.0
	for _, step := range pb.Steps {
		switch step.Action {
		case model.StepActionEncryptAtRest, model.StepActionEncryptInTransit:
			stepBonus += 5.0
		case model.StepActionRevokeAccess, model.StepActionDowngradeAccess:
			stepBonus += 4.0
		case model.StepActionRestrictNetwork, model.StepActionQuarantine:
			stepBonus += 4.5
		case model.StepActionEnableAuditLog:
			stepBonus += 2.0
		case model.StepActionReclassify:
			stepBonus += 3.0
		case model.StepActionScheduleReview:
			stepBonus += 1.5
		case model.StepActionCreateTicket, model.StepActionNotifyOwner:
			stepBonus += 0.5
		case model.StepActionArchiveData, model.StepActionDeleteData:
			stepBonus += 3.5
		case model.StepActionConfigureBackup:
			stepBonus += 2.5
		}
	}

	// Cap the step bonus at 20 percentage points.
	if stepBonus > 20.0 {
		stepBonus = 20.0
	}

	total := base + stepBonus
	// Cap total at 85% -- no single playbook eliminates all risk.
	if total > 85.0 {
		total = 85.0
	}

	return total
}

// estimateAffectedIdentities provides a conservative estimate of identities affected
// by a playbook that targets an asset without specifying a particular identity.
func estimateAffectedIdentities(pb *model.Playbook) int {
	hasAccessAction := false
	for _, step := range pb.Steps {
		switch step.Action {
		case model.StepActionRevokeAccess, model.StepActionDowngradeAccess, model.StepActionScheduleReview:
			hasAccessAction = true
		}
	}

	if hasAccessAction {
		// Access-related playbooks likely affect multiple identities.
		switch pb.FindingType {
		case model.FindingBlastRadiusExcessive:
			return 15
		case model.FindingOverprivilegedAccess:
			return 8
		case model.FindingStaleAccess:
			return 5
		default:
			return 3
		}
	}

	return 0
}
