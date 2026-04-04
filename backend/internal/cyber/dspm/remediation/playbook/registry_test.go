package playbook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// expectedPlaybooks maps each built-in playbook's finding type to its ID.
var expectedPlaybooks = map[model.FindingType]string{
	model.FindingEncryptionMissing:    "encrypt-sensitive-data",
	model.FindingOverprivilegedAccess: "revoke-overprivileged-access",
	model.FindingExposureRisk:         "restrict-network-exposure",
	model.FindingShadowCopy:           "remediate-shadow-copy",
	model.FindingPIIUnprotected:       "enforce-pii-controls",
	model.FindingClassificationDrift:  "handle-classification-drift",
	model.FindingRetentionExpired:     "enforce-data-retention",
	model.FindingBlastRadiusExcessive: "reduce-blast-radius",
	model.FindingPostureGap:           "posture-gap-generic",
	model.FindingStaleAccess:          "stale-access-cleanup",
}

// validStepActions is the set of all valid StepAction values.
var validStepActions = map[model.StepAction]bool{
	model.StepActionEncryptAtRest:    true,
	model.StepActionEncryptInTransit: true,
	model.StepActionRevokeAccess:     true,
	model.StepActionDowngradeAccess:  true,
	model.StepActionRestrictNetwork:  true,
	model.StepActionEnableAuditLog:   true,
	model.StepActionConfigureBackup:  true,
	model.StepActionCreateTicket:     true,
	model.StepActionNotifyOwner:      true,
	model.StepActionQuarantine:       true,
	model.StepActionReclassify:       true,
	model.StepActionScheduleReview:   true,
	model.StepActionArchiveData:      true,
	model.StepActionDeleteData:       true,
}

func TestAllPlaybooksRegistered(t *testing.T) {
	reg := NewRegistry()
	playbooks := reg.List()

	// Verify exactly 10 playbooks are loaded.
	require.Len(t, playbooks, 10, "registry should contain exactly 10 built-in playbooks")

	// Verify each expected finding type has a corresponding playbook.
	for findingType, expectedID := range expectedPlaybooks {
		pb, ok := reg.GetForFindingType(findingType)
		require.True(t, ok, "playbook for finding type %q should be registered", findingType)
		assert.Equal(t, expectedID, pb.ID, "playbook ID mismatch for finding type %q", findingType)
	}
}

func TestGetPlaybook(t *testing.T) {
	reg := NewRegistry()

	for findingType, expectedID := range expectedPlaybooks {
		t.Run(string(findingType), func(t *testing.T) {
			pb, ok := reg.GetForFindingType(findingType)
			require.True(t, ok)
			assert.Equal(t, expectedID, pb.ID)
			assert.Equal(t, findingType, pb.FindingType)
			assert.NotEmpty(t, pb.Name)
			assert.NotEmpty(t, pb.Description)
		})
	}
}

func TestGetPlaybookByID(t *testing.T) {
	reg := NewRegistry()

	for _, id := range expectedPlaybooks {
		t.Run(id, func(t *testing.T) {
			pb, ok := reg.Get(id)
			require.True(t, ok)
			assert.Equal(t, id, pb.ID)
		})
	}
}

func TestGetPlaybookNotFound(t *testing.T) {
	reg := NewRegistry()

	t.Run("unknown_finding_type", func(t *testing.T) {
		pb, ok := reg.GetForFindingType("nonexistent_finding_type")
		assert.False(t, ok)
		assert.Nil(t, pb)
	})

	t.Run("unknown_id", func(t *testing.T) {
		pb, ok := reg.Get("nonexistent-playbook-id")
		assert.False(t, ok)
		assert.Nil(t, pb)
	})

	t.Run("empty_finding_type", func(t *testing.T) {
		pb, ok := reg.GetForFindingType("")
		assert.False(t, ok)
		assert.Nil(t, pb)
	})
}

func TestPlaybookStepCount(t *testing.T) {
	reg := NewRegistry()
	playbooks := reg.List()

	for _, pb := range playbooks {
		t.Run(pb.ID, func(t *testing.T) {
			assert.GreaterOrEqual(t, len(pb.Steps), 2,
				"playbook %q should have at least 2 steps, got %d", pb.ID, len(pb.Steps))
		})
	}
}

func TestPlaybookStepActions(t *testing.T) {
	reg := NewRegistry()
	playbooks := reg.List()

	for _, pb := range playbooks {
		t.Run(pb.ID, func(t *testing.T) {
			for i, step := range pb.Steps {
				assert.NotEmpty(t, step.Action,
					"step %d (%s) in playbook %q should have an action", i+1, step.ID, pb.ID)
				assert.True(t, validStepActions[step.Action],
					"step %d (%s) in playbook %q has invalid action %q", i+1, step.ID, pb.ID, step.Action)
				assert.NotEmpty(t, step.ID,
					"step %d in playbook %q should have an ID", i+1, pb.ID)
				assert.NotEmpty(t, step.Description,
					"step %d (%s) in playbook %q should have a description", i+1, step.ID, pb.ID)
				assert.Greater(t, step.Timeout.Seconds(), 0.0,
					"step %d (%s) in playbook %q should have a positive timeout", i+1, step.ID, pb.ID)
				assert.NotEmpty(t, step.FailureHandling,
					"step %d (%s) in playbook %q should have failure handling", i+1, step.ID, pb.ID)
			}
		})
	}
}

func TestPlaybookStepOrdering(t *testing.T) {
	reg := NewRegistry()
	playbooks := reg.List()

	for _, pb := range playbooks {
		t.Run(pb.ID, func(t *testing.T) {
			for i, step := range pb.Steps {
				assert.Equal(t, i+1, step.Order,
					"step %d (%s) in playbook %q should have order %d, got %d",
					i+1, step.ID, pb.ID, i+1, step.Order)
			}
		})
	}
}

func TestRegistryListSortedByID(t *testing.T) {
	reg := NewRegistry()
	playbooks := reg.List()

	for i := 1; i < len(playbooks); i++ {
		assert.LessOrEqual(t, playbooks[i-1].ID, playbooks[i].ID,
			"playbooks should be sorted by ID: %q should come before %q",
			playbooks[i-1].ID, playbooks[i].ID)
	}
}

func TestPlaybookMetadata(t *testing.T) {
	reg := NewRegistry()

	tests := []struct {
		id               string
		requiresApproval bool
		autoRollback     bool
	}{
		{"encrypt-sensitive-data", false, false},
		{"remediate-shadow-copy", true, true},
		{"enforce-data-retention", true, false},
		{"enforce-pii-controls", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			pb, ok := reg.Get(tt.id)
			require.True(t, ok)
			assert.Equal(t, tt.requiresApproval, pb.RequiresApproval, "requiresApproval mismatch")
			assert.Equal(t, tt.autoRollback, pb.AutoRollback, "autoRollback mismatch")
			assert.Greater(t, pb.EstimatedMinutes, 0, "estimated minutes should be positive")
		})
	}
}
