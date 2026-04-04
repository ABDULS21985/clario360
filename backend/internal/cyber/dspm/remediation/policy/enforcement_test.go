package policy

import (
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

func newTestEnforcer() *Enforcer {
	logger := zerolog.Nop()
	return NewEnforcer(logger)
}

func newTestViolation(severity string) *model.PolicyViolation {
	return &model.PolicyViolation{
		PolicyID:       uuid.New(),
		PolicyName:     "test-policy",
		Category:       "encryption",
		AssetID:        uuid.New(),
		AssetName:      "test-asset",
		AssetType:      "database",
		Classification: "confidential",
		Severity:       severity,
		Description:    "test violation description",
		Enforcement:    "alert",
	}
}

func TestDetermineActionAlert(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("high")

	action := enforcer.DetermineAction(violation, model.EnforcementAlert)

	assert.Equal(t, "alert", action.Action)
	assert.True(t, action.CreateAlert, "alert mode should create an alert")
	assert.False(t, action.CreateRemediation, "alert mode should not create remediation")
	assert.False(t, action.QuarantineAsset, "alert mode should not quarantine")
	assert.Empty(t, action.PlaybookID, "alert mode should have no playbook")
}

func TestDetermineActionAutoRemediate(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("high")

	action := enforcer.DetermineAction(violation, model.EnforcementAutoRemediate)

	assert.Equal(t, "auto_remediate", action.Action)
	assert.True(t, action.CreateAlert, "auto_remediate should create an alert")
	assert.True(t, action.CreateRemediation, "auto_remediate should create remediation")
	assert.False(t, action.QuarantineAsset, "auto_remediate should not quarantine")
}

func TestDetermineActionBlock(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("critical")

	action := enforcer.DetermineAction(violation, model.EnforcementBlock)

	assert.Equal(t, "block", action.Action)
	assert.True(t, action.CreateAlert, "block mode should create an alert")
	assert.False(t, action.CreateRemediation, "block mode should not create remediation")
	assert.True(t, action.QuarantineAsset, "block mode should quarantine the asset")
	assert.Empty(t, action.PlaybookID, "block mode should have no playbook")
}

func TestDetermineActionUnknownEnforcementDefaultsToAlert(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("medium")

	action := enforcer.DetermineAction(violation, model.PolicyEnforcement("unknown_mode"))

	assert.Equal(t, "alert", action.Action, "unknown enforcement should default to alert")
	assert.True(t, action.CreateAlert)
	assert.False(t, action.CreateRemediation)
	assert.False(t, action.QuarantineAsset)
}

func TestEnforcementPriority(t *testing.T) {
	enforcer := newTestEnforcer()

	severities := []string{"low", "medium", "high", "critical"}

	for _, sev := range severities {
		t.Run("severity_"+sev, func(t *testing.T) {
			violation := newTestViolation(sev)

			// Alert mode.
			alertAction := enforcer.DetermineAction(violation, model.EnforcementAlert)
			assert.Equal(t, "alert", alertAction.Action)
			assert.True(t, alertAction.CreateAlert)

			// Block mode.
			blockAction := enforcer.DetermineAction(violation, model.EnforcementBlock)
			assert.Equal(t, "block", blockAction.Action)
			assert.True(t, blockAction.QuarantineAsset)

			// Auto-remediate mode.
			remAction := enforcer.DetermineAction(violation, model.EnforcementAutoRemediate)
			assert.Equal(t, "auto_remediate", remAction.Action)
			assert.True(t, remAction.CreateRemediation)
		})
	}
}

func TestDetermineActionWithPlaybook(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("high")

	t.Run("auto_remediate_with_playbook", func(t *testing.T) {
		action := enforcer.DetermineActionWithPlaybook(
			violation,
			model.EnforcementAutoRemediate,
			"encrypt-sensitive-data",
		)

		assert.Equal(t, "auto_remediate", action.Action)
		assert.True(t, action.CreateRemediation)
		assert.Equal(t, "encrypt-sensitive-data", action.PlaybookID)
	})

	t.Run("alert_ignores_playbook", func(t *testing.T) {
		action := enforcer.DetermineActionWithPlaybook(
			violation,
			model.EnforcementAlert,
			"encrypt-sensitive-data",
		)

		assert.Equal(t, "alert", action.Action)
		assert.Empty(t, action.PlaybookID, "alert mode should ignore playbook ID")
	})

	t.Run("block_ignores_playbook", func(t *testing.T) {
		action := enforcer.DetermineActionWithPlaybook(
			violation,
			model.EnforcementBlock,
			"encrypt-sensitive-data",
		)

		assert.Equal(t, "block", action.Action)
		assert.Empty(t, action.PlaybookID, "block mode should ignore playbook ID")
	})

	t.Run("auto_remediate_empty_playbook", func(t *testing.T) {
		action := enforcer.DetermineActionWithPlaybook(
			violation,
			model.EnforcementAutoRemediate,
			"",
		)

		assert.Equal(t, "auto_remediate", action.Action)
		assert.True(t, action.CreateRemediation)
		// PlaybookID will be empty because extractPlaybookID returns "" and the
		// empty string is not overridden.
		assert.Empty(t, action.PlaybookID)
	})
}

func TestEnforcementActionFieldConsistency(t *testing.T) {
	enforcer := newTestEnforcer()
	violation := newTestViolation("medium")

	tests := []struct {
		name        string
		enforcement model.PolicyEnforcement
		wantAction  string
		wantAlert   bool
		wantRemed   bool
		wantBlock   bool
	}{
		{"alert", model.EnforcementAlert, "alert", true, false, false},
		{"auto_remediate", model.EnforcementAutoRemediate, "auto_remediate", true, true, false},
		{"block", model.EnforcementBlock, "block", true, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := enforcer.DetermineAction(violation, tt.enforcement)
			assert.Equal(t, tt.wantAction, action.Action)
			assert.Equal(t, tt.wantAlert, action.CreateAlert)
			assert.Equal(t, tt.wantRemed, action.CreateRemediation)
			assert.Equal(t, tt.wantBlock, action.QuarantineAsset)
		})
	}
}
