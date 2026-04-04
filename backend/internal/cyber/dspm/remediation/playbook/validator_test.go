package playbook

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestValidator() *Validator {
	logger := zerolog.Nop()
	reg := NewRegistry()
	return NewValidator(reg, logger)
}

func TestDryRunValid(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// encrypt-sensitive-data is an asset-related playbook.
	result, err := v.DryRun(ctx, "encrypt-sensitive-data", &assetID, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Valid, "dry run should be valid for a correct playbook + asset")
	assert.Equal(t, 1, result.AssetsAffected, "should affect exactly 1 asset")
	assert.Greater(t, result.EstimatedRiskReduction, 0.0,
		"estimated risk reduction should be positive")
}

func TestDryRunValidWithIdentity(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// revoke-overprivileged-access is an identity-related playbook.
	result, err := v.DryRun(ctx, "revoke-overprivileged-access", &assetID, "identity-123")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Valid)
	assert.Equal(t, 1, result.AssetsAffected)
	assert.Equal(t, 1, result.IdentitiesAffected)
	assert.Greater(t, result.EstimatedRiskReduction, 0.0)
}

func TestDryRunMissingAsset(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()

	// encrypt-sensitive-data requires an asset ID because it targets
	// FindingEncryptionMissing (an asset-related finding type).
	result, err := v.DryRun(ctx, "encrypt-sensitive-data", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid, "dry run should be invalid when asset-related playbook has no asset ID")
	assert.NotEmpty(t, result.Issues)

	foundIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "requires a data asset ID") {
			foundIssue = true
			break
		}
	}
	assert.True(t, foundIssue, "issues should mention missing data asset ID, got: %v", result.Issues)
}

func TestDryRunMissingIdentityWarning(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// revoke-overprivileged-access targets FindingOverprivilegedAccess (identity-related).
	// Missing identity_id should produce a warning (not a hard failure).
	result, err := v.DryRun(ctx, "revoke-overprivileged-access", &assetID, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	// The result should still be valid (warning only).
	assert.True(t, result.Valid, "missing identity_id should be a warning, not a failure")

	foundWarning := false
	for _, issue := range result.Issues {
		if contains(issue, "identity_id improves remediation accuracy") {
			foundWarning = true
			break
		}
	}
	assert.True(t, foundWarning, "issues should contain identity_id warning, got: %v", result.Issues)
}

func TestDryRunPlaybookNotFound(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	result, err := v.DryRun(ctx, "nonexistent-playbook", &assetID, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.False(t, result.Valid, "dry run should be invalid for unknown playbook")
	assert.NotEmpty(t, result.Issues)

	foundIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "not found in registry") {
			foundIssue = true
			break
		}
	}
	assert.True(t, foundIssue, "issues should mention playbook not found, got: %v", result.Issues)
}

func TestDryRunApprovalRequired(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// remediate-shadow-copy requires approval.
	result, err := v.DryRun(ctx, "remediate-shadow-copy", &assetID, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.True(t, result.Valid)

	foundApprovalIssue := false
	for _, issue := range result.Issues {
		if contains(issue, "requires manual approval") {
			foundApprovalIssue = true
			break
		}
	}
	assert.True(t, foundApprovalIssue,
		"issues should mention approval requirement, got: %v", result.Issues)
}

func TestDryRunStepsValidation(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// All built-in playbooks have valid steps; verifying they pass validation.
	for _, pbID := range expectedPlaybooks {
		t.Run(pbID, func(t *testing.T) {
			result, err := v.DryRun(ctx, pbID, &assetID, "some-identity")
			require.NoError(t, err)
			require.NotNil(t, result)

			// No step-level issues (e.g. "has no action defined") should appear.
			for _, issue := range result.Issues {
				assert.NotContains(t, issue, "has no action defined",
					"playbook %q should have valid step actions", pbID)
			}
		})
	}
}

func TestDryRunCancelledContext(t *testing.T) {
	v := newTestValidator()
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	assetID := uuid.New()
	result, err := v.DryRun(ctx, "encrypt-sensitive-data", &assetID, "")
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "context cancelled")
}

func TestDryRunEstimatedIdentities(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// reduce-blast-radius targets FindingBlastRadiusExcessive.
	// When asset is provided but no identity, it estimates affected identities.
	result, err := v.DryRun(ctx, "reduce-blast-radius", &assetID, "")
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, 1, result.AssetsAffected)
	assert.Equal(t, 15, result.IdentitiesAffected,
		"blast radius playbook should estimate 15 affected identities")
}

func TestDryRunRiskReductionCapped(t *testing.T) {
	v := newTestValidator()
	ctx := context.Background()
	assetID := uuid.New()

	// Verify that no single playbook exceeds 85% risk reduction.
	for _, pbID := range expectedPlaybooks {
		t.Run(pbID, func(t *testing.T) {
			result, err := v.DryRun(ctx, pbID, &assetID, "identity-test")
			require.NoError(t, err)
			require.NotNil(t, result)
			assert.LessOrEqual(t, result.EstimatedRiskReduction, 85.0,
				"risk reduction for %q should not exceed 85%%", pbID)
		})
	}
}

// contains checks if a string contains a substring.
func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
