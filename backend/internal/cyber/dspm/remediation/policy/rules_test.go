package policy

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cybermodel "github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// boolPtr returns a pointer to a bool value.
func boolPtr(b bool) *bool { return &b }

// strPtr returns a pointer to a string value.
func strPtr(s string) *string { return &s }

// mustMarshal marshals v to json.RawMessage, panicking on error.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return data
}

func TestEvaluateRuleEncryption(t *testing.T) {
	tests := []struct {
		name            string
		asset           *cybermodel.DSPMDataAsset
		rule            model.EncryptionRule
		expectViolation bool
		expectContains  string
	}{
		{
			name: "encrypted_asset_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "encrypted-db",
				DataClassification: "confidential",
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(true),
			},
			rule: model.EncryptionRule{
				RequireAtRest:     true,
				RequireInTransit:  true,
				ClassificationMin: "confidential",
			},
			expectViolation: false,
		},
		{
			name: "unencrypted_at_rest_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "unencrypted-db",
				DataClassification: "confidential",
				EncryptedAtRest:    boolPtr(false),
				EncryptedInTransit: boolPtr(true),
			},
			rule: model.EncryptionRule{
				RequireAtRest:     true,
				RequireInTransit:  true,
				ClassificationMin: "confidential",
			},
			expectViolation: true,
			expectContains:  "not encrypted at rest",
		},
		{
			name: "unencrypted_in_transit_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "partial-encrypt-db",
				DataClassification: "restricted",
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(false),
			},
			rule: model.EncryptionRule{
				RequireAtRest:     true,
				RequireInTransit:  true,
				ClassificationMin: "confidential",
			},
			expectViolation: true,
			expectContains:  "not encrypted in transit",
		},
		{
			name: "nil_encryption_fields_violate",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "nil-encrypt-db",
				DataClassification: "confidential",
				EncryptedAtRest:    nil,
				EncryptedInTransit: nil,
			},
			rule: model.EncryptionRule{
				RequireAtRest:    true,
				RequireInTransit: true,
			},
			expectViolation: true,
			expectContains:  "not encrypted at rest",
		},
		{
			name: "below_classification_min_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "public-db",
				DataClassification: "public",
				EncryptedAtRest:    boolPtr(false),
				EncryptedInTransit: boolPtr(false),
			},
			rule: model.EncryptionRule{
				RequireAtRest:     true,
				RequireInTransit:  true,
				ClassificationMin: "confidential",
			},
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &model.DataPolicy{
				Category: model.PolicyCategoryEncryption,
				Rule:     mustMarshal(t, tt.rule),
			}

			violated, desc := EvaluateRule(tt.asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, tt.expectContains)
			} else {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRuleClassification(t *testing.T) {
	tests := []struct {
		name            string
		asset           *cybermodel.DSPMDataAsset
		rule            model.ClassificationRule
		expectViolation bool
		expectContains  string
	}{
		{
			name: "pii_asset_below_minimum_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "low-class-pii",
				DataClassification: "internal",
				ContainsPII:        true,
			},
			rule: model.ClassificationRule{
				AutoEscalate:  true,
				PIIImpliesMin: "confidential",
			},
			expectViolation: true,
			expectContains:  "contains PII but is classified as",
		},
		{
			name: "pii_asset_at_minimum_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "proper-class-pii",
				DataClassification: "confidential",
				ContainsPII:        true,
			},
			rule: model.ClassificationRule{
				AutoEscalate:  true,
				PIIImpliesMin: "confidential",
			},
			expectViolation: false,
		},
		{
			name: "non_pii_asset_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "no-pii",
				DataClassification: "public",
				ContainsPII:        false,
			},
			rule: model.ClassificationRule{
				AutoEscalate:  true,
				PIIImpliesMin: "confidential",
			},
			expectViolation: false,
		},
		{
			name: "auto_escalate_disabled_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "no-escalate",
				DataClassification: "public",
				ContainsPII:        true,
			},
			rule: model.ClassificationRule{
				AutoEscalate:  false,
				PIIImpliesMin: "confidential",
			},
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &model.DataPolicy{
				Category: model.PolicyCategoryClassification,
				Rule:     mustMarshal(t, tt.rule),
			}

			violated, desc := EvaluateRule(tt.asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, tt.expectContains)
			} else {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRuleRetention(t *testing.T) {
	tests := []struct {
		name            string
		assetAge        int // days old
		maxDays         int
		scope           []string
		classification  string
		expectViolation bool
	}{
		{
			name:            "fresh_asset_passes",
			assetAge:        10,
			maxDays:         365,
			classification:  "confidential",
			expectViolation: false,
		},
		{
			name:            "expired_asset_violates",
			assetAge:        400,
			maxDays:         365,
			classification:  "confidential",
			expectViolation: true,
		},
		{
			name:            "out_of_scope_classification_passes",
			assetAge:        400,
			maxDays:         365,
			scope:           []string{"restricted"},
			classification:  "public",
			expectViolation: false,
		},
		{
			name:            "in_scope_classification_violates",
			assetAge:        400,
			maxDays:         365,
			scope:           []string{"confidential"},
			classification:  "confidential",
			expectViolation: true,
		},
		{
			name:            "zero_max_days_passes",
			assetAge:        400,
			maxDays:         0,
			classification:  "confidential",
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "test-asset",
				DataClassification: tt.classification,
				CreatedAt:          time.Now().AddDate(0, 0, -tt.assetAge),
			}

			rule := model.RetentionRule{
				MaxDays:             tt.maxDays,
				ClassificationScope: tt.scope,
				Action:              "archive",
			}

			policy := &model.DataPolicy{
				Category: model.PolicyCategoryRetention,
				Rule:     mustMarshal(t, rule),
			}

			violated, desc := EvaluateRule(asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, "exceeding")
			}
			if !tt.expectViolation {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRuleExposure(t *testing.T) {
	tests := []struct {
		name            string
		assetExposure   *string
		maxExposure     string
		classification  string
		classMin        string
		expectViolation bool
		expectContains  string
	}{
		{
			name:            "within_limit_passes",
			assetExposure:   strPtr("internal_only"),
			maxExposure:     "vpn_accessible",
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: false,
		},
		{
			name:            "exceeds_limit_violates",
			assetExposure:   strPtr("internet_facing"),
			maxExposure:     "vpn_accessible",
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: true,
			expectContains:  "exceeds the maximum allowed",
		},
		{
			name:            "below_classification_min_passes",
			assetExposure:   strPtr("internet_facing"),
			maxExposure:     "internal_only",
			classification:  "public",
			classMin:        "confidential",
			expectViolation: false,
		},
		{
			name:            "nil_exposure_passes",
			assetExposure:   nil,
			maxExposure:     "internal_only",
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: false,
		},
		{
			name:            "at_limit_passes",
			assetExposure:   strPtr("vpn_accessible"),
			maxExposure:     "vpn_accessible",
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "test-asset",
				DataClassification: tt.classification,
				NetworkExposure:    tt.assetExposure,
			}

			rule := model.ExposureRule{
				MaxExposure:       tt.maxExposure,
				ClassificationMin: tt.classMin,
			}

			policy := &model.DataPolicy{
				Category: model.PolicyCategoryExposure,
				Rule:     mustMarshal(t, rule),
			}

			violated, desc := EvaluateRule(asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, tt.expectContains)
			} else {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRulePIIProtection(t *testing.T) {
	tests := []struct {
		name            string
		asset           *cybermodel.DSPMDataAsset
		rule            model.PIIProtectionRule
		expectViolation bool
		expectContains  string
	}{
		{
			name: "fully_protected_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "protected-pii",
				ContainsPII:        true,
				PIITypes:           []string{"email", "ssn"},
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(true),
				AccessControlType:  strPtr("rbac"),
				AuditLogging:       boolPtr(true),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: false,
		},
		{
			name: "missing_encryption_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "unencrypted-pii",
				ContainsPII:        true,
				PIITypes:           []string{"email"},
				EncryptedAtRest:    boolPtr(false),
				EncryptedInTransit: boolPtr(true),
				AccessControlType:  strPtr("rbac"),
				AuditLogging:       boolPtr(true),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: true,
			expectContains:  "missing full encryption",
		},
		{
			name: "missing_audit_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "no-audit-pii",
				ContainsPII:        true,
				PIITypes:           []string{"ssn"},
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(true),
				AccessControlType:  strPtr("rbac"),
				AuditLogging:       boolPtr(false),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: true,
			expectContains:  "audit logging not enabled",
		},
		{
			name: "wrong_access_control_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "wrong-acl-pii",
				ContainsPII:        true,
				PIITypes:           []string{"email"},
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(true),
				AccessControlType:  strPtr("basic"),
				AuditLogging:       boolPtr(true),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: true,
			expectContains:  "does not meet required",
		},
		{
			name: "no_access_control_configured_violates",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "nil-acl-pii",
				ContainsPII:        true,
				PIITypes:           []string{"email"},
				EncryptedAtRest:    boolPtr(true),
				EncryptedInTransit: boolPtr(true),
				AccessControlType:  nil,
				AuditLogging:       boolPtr(true),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: true,
			expectContains:  "no access control configured",
		},
		{
			name: "non_pii_asset_passes",
			asset: &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "no-pii-asset",
				ContainsPII:        false,
				EncryptedAtRest:    boolPtr(false),
				EncryptedInTransit: boolPtr(false),
				AuditLogging:       boolPtr(false),
			},
			rule: model.PIIProtectionRule{
				RequireEncryption:    true,
				RequireAccessControl: "rbac",
				RequireAudit:         true,
			},
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &model.DataPolicy{
				Category: model.PolicyCategoryPIIProtection,
				Rule:     mustMarshal(t, tt.rule),
			}

			violated, desc := EvaluateRule(tt.asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, tt.expectContains)
			} else {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRuleAccessReview(t *testing.T) {
	recentReview := time.Now().AddDate(0, 0, -10)
	oldReview := time.Now().AddDate(0, 0, -100)

	tests := []struct {
		name            string
		lastReview      *time.Time
		reviewInterval  int
		classification  string
		classMin        string
		expectViolation bool
		expectContains  string
	}{
		{
			name:            "recent_review_passes",
			lastReview:      &recentReview,
			reviewInterval:  30,
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: false,
		},
		{
			name:            "overdue_review_violates",
			lastReview:      &oldReview,
			reviewInterval:  30,
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: true,
			expectContains:  "last access review was",
		},
		{
			name:            "nil_review_violates",
			lastReview:      nil,
			reviewInterval:  30,
			classification:  "confidential",
			classMin:        "confidential",
			expectViolation: true,
			expectContains:  "has never had an access review",
		},
		{
			name:            "below_classification_min_passes",
			lastReview:      nil,
			reviewInterval:  30,
			classification:  "public",
			classMin:        "confidential",
			expectViolation: false,
		},
		{
			name:            "zero_interval_passes",
			lastReview:      nil,
			reviewInterval:  0,
			classification:  "confidential",
			classMin:        "",
			expectViolation: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "test-asset",
				DataClassification: tt.classification,
				LastAccessReview:   tt.lastReview,
			}

			rule := model.AccessReviewRule{
				ReviewIntervalDays: tt.reviewInterval,
				ClassificationMin:  tt.classMin,
			}

			policy := &model.DataPolicy{
				Category: model.PolicyCategoryAccessReview,
				Rule:     mustMarshal(t, rule),
			}

			violated, desc := EvaluateRule(asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, tt.expectContains)
			} else {
				assert.Empty(t, desc)
			}
		})
	}
}

func TestEvaluateRuleBackup(t *testing.T) {
	tests := []struct {
		name            string
		backupConfigured *bool
		requiredFor     []string
		classification  string
		expectViolation bool
	}{
		{
			name:             "backup_configured_passes",
			backupConfigured: boolPtr(true),
			requiredFor:      []string{"confidential", "restricted"},
			classification:   "confidential",
			expectViolation:  false,
		},
		{
			name:             "backup_not_configured_violates",
			backupConfigured: boolPtr(false),
			requiredFor:      []string{"confidential", "restricted"},
			classification:   "confidential",
			expectViolation:  true,
		},
		{
			name:             "nil_backup_violates",
			backupConfigured: nil,
			requiredFor:      []string{"confidential"},
			classification:   "confidential",
			expectViolation:  true,
		},
		{
			name:             "out_of_scope_passes",
			backupConfigured: boolPtr(false),
			requiredFor:      []string{"restricted"},
			classification:   "public",
			expectViolation:  false,
		},
		{
			name:             "empty_required_for_applies_to_all",
			backupConfigured: boolPtr(false),
			requiredFor:      nil,
			classification:   "public",
			expectViolation:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "test-asset",
				DataClassification: tt.classification,
				BackupConfigured:   tt.backupConfigured,
			}

			rule := model.BackupRule{
				RequiredFor: tt.requiredFor,
			}

			policy := &model.DataPolicy{
				Category: model.PolicyCategoryBackup,
				Rule:     mustMarshal(t, rule),
			}

			violated, desc := EvaluateRule(asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, "does not have backup configured")
			}
		})
	}
}

func TestEvaluateRuleAuditLogging(t *testing.T) {
	tests := []struct {
		name            string
		auditLogging    *bool
		requiredFor     []string
		classification  string
		expectViolation bool
	}{
		{
			name:            "audit_logging_enabled_passes",
			auditLogging:    boolPtr(true),
			requiredFor:     []string{"confidential", "restricted"},
			classification:  "confidential",
			expectViolation: false,
		},
		{
			name:            "audit_logging_disabled_violates",
			auditLogging:    boolPtr(false),
			requiredFor:     []string{"confidential", "restricted"},
			classification:  "confidential",
			expectViolation: true,
		},
		{
			name:            "nil_audit_logging_violates",
			auditLogging:    nil,
			requiredFor:     []string{"confidential"},
			classification:  "confidential",
			expectViolation: true,
		},
		{
			name:            "out_of_scope_passes",
			auditLogging:    boolPtr(false),
			requiredFor:     []string{"restricted"},
			classification:  "public",
			expectViolation: false,
		},
		{
			name:            "empty_required_for_applies_to_all",
			auditLogging:    boolPtr(false),
			requiredFor:     nil,
			classification:  "internal",
			expectViolation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset := &cybermodel.DSPMDataAsset{
				ID:                 uuid.New(),
				AssetName:          "test-asset",
				DataClassification: tt.classification,
				AuditLogging:       tt.auditLogging,
			}

			rule := model.AuditLoggingRule{
				RequiredFor: tt.requiredFor,
			}

			policy := &model.DataPolicy{
				Category: model.PolicyCategoryAuditLogging,
				Rule:     mustMarshal(t, rule),
			}

			violated, desc := EvaluateRule(asset, policy)
			assert.Equal(t, tt.expectViolation, violated)
			if tt.expectViolation {
				assert.Contains(t, desc, "does not have audit logging enabled")
			}
		})
	}
}

func TestEvaluateRuleUnknownCategory(t *testing.T) {
	asset := &cybermodel.DSPMDataAsset{
		ID:        uuid.New(),
		AssetName: "test-asset",
	}

	policy := &model.DataPolicy{
		Category: "unknown_category",
		Rule:     json.RawMessage(`{}`),
	}

	violated, desc := EvaluateRule(asset, policy)
	assert.False(t, violated)
	assert.Empty(t, desc)
}
