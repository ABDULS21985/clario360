package policy

import (
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// EnforcementAction describes the concrete actions to take for a policy violation
// based on its enforcement mode.
type EnforcementAction struct {
	// Action is a human-readable label such as "alert", "auto_remediate", or "block".
	Action string `json:"action"`
	// CreateAlert indicates whether a security alert should be generated.
	CreateAlert bool `json:"create_alert"`
	// CreateRemediation indicates whether an automated remediation work item
	// should be created and dispatched to a playbook executor.
	CreateRemediation bool `json:"create_remediation"`
	// QuarantineAsset indicates whether the asset should be immediately
	// quarantined (network isolation, access revocation, etc.).
	QuarantineAsset bool `json:"quarantine_asset"`
	// PlaybookID is the remediation playbook to execute when CreateRemediation
	// is true. Empty for non-remediation actions.
	PlaybookID string `json:"playbook_id,omitempty"`
}

// Enforcer translates policy violations into enforcement actions based on the
// policy's configured enforcement mode.
type Enforcer struct {
	logger zerolog.Logger
}

// NewEnforcer constructs an Enforcer.
func NewEnforcer(logger zerolog.Logger) *Enforcer {
	return &Enforcer{
		logger: logger.With().Str("component", "policy_enforcer").Logger(),
	}
}

// DetermineAction maps a violation and its enforcement mode to a concrete
// EnforcementAction.
//
// Enforcement modes behave as follows:
//
//   - alert: only creates an alert; no automated remediation or quarantine.
//   - auto_remediate: creates an alert and a remediation work item, dispatching
//     to the playbook identified by the violation's parent policy.
//   - block: creates an alert and immediately quarantines the asset.
func (e *Enforcer) DetermineAction(violation *model.PolicyViolation, enforcement model.PolicyEnforcement) EnforcementAction {
	switch enforcement {
	case model.EnforcementAlert:
		e.logger.Info().
			Str("policy_id", violation.PolicyID.String()).
			Str("asset_id", violation.AssetID.String()).
			Str("enforcement", string(enforcement)).
			Msg("enforcement action: alert only")

		return EnforcementAction{
			Action:      "alert",
			CreateAlert: true,
		}

	case model.EnforcementAutoRemediate:
		// Extract the playbook ID from the violation's enforcement metadata.
		// The playbook ID is carried on the parent DataPolicy; callers should
		// populate it via the policy's AutoPlaybookID field.
		playbookID := extractPlaybookID(violation)

		e.logger.Info().
			Str("policy_id", violation.PolicyID.String()).
			Str("asset_id", violation.AssetID.String()).
			Str("enforcement", string(enforcement)).
			Str("playbook_id", playbookID).
			Msg("enforcement action: auto-remediate")

		return EnforcementAction{
			Action:            "auto_remediate",
			CreateAlert:       true,
			CreateRemediation: true,
			PlaybookID:        playbookID,
		}

	case model.EnforcementBlock:
		e.logger.Warn().
			Str("policy_id", violation.PolicyID.String()).
			Str("asset_id", violation.AssetID.String()).
			Str("enforcement", string(enforcement)).
			Msg("enforcement action: block / quarantine")

		return EnforcementAction{
			Action:          "block",
			CreateAlert:     true,
			QuarantineAsset: true,
		}

	default:
		// Unknown enforcement mode; fail safe by alerting.
		e.logger.Warn().
			Str("policy_id", violation.PolicyID.String()).
			Str("asset_id", violation.AssetID.String()).
			Str("enforcement", string(enforcement)).
			Msg("unknown enforcement mode, defaulting to alert")

		return EnforcementAction{
			Action:      "alert",
			CreateAlert: true,
		}
	}
}

// extractPlaybookID returns the playbook ID embedded in the violation's
// enforcement field. For auto_remediate violations, the caller is expected to
// populate the Enforcement field with a value that may carry a playbook
// reference. If the violation was constructed from a DataPolicy, the playbook
// ID should be set separately. This function returns an empty string as a
// fallback; the caller resolves the playbook from the originating policy.
func extractPlaybookID(_ *model.PolicyViolation) string {
	// PolicyViolation.Enforcement stores the enforcement mode string (e.g.
	// "auto_remediate"), not the playbook ID. The playbook is an attribute of
	// the parent DataPolicy (AutoPlaybookID). Callers that construct
	// EnforcementActions from a full policy+violation pair should pass the
	// playbook ID through the DetermineActionWithPlaybook helper instead.
	return ""
}

// DetermineActionWithPlaybook is a convenience wrapper that attaches an
// explicit playbook ID to auto-remediation actions. For other enforcement modes
// the playbookID parameter is ignored.
func (e *Enforcer) DetermineActionWithPlaybook(violation *model.PolicyViolation, enforcement model.PolicyEnforcement, playbookID string) EnforcementAction {
	action := e.DetermineAction(violation, enforcement)
	if enforcement == model.EnforcementAutoRemediate && playbookID != "" {
		action.PlaybookID = playbookID
	}
	return action
}
