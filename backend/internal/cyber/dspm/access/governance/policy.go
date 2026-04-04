package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/access/model"
)

// PolicyRepository loads enabled policies for a tenant.
type PolicyRepository interface {
	ListEnabled(ctx context.Context, tenantID uuid.UUID) ([]model.AccessPolicy, error)
}

// MappingProvider loads active access mappings.
type MappingProvider interface {
	ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.AccessMapping, error)
}

// ProfileProvider loads identity profiles.
type ProfileProvider interface {
	ListActive(ctx context.Context, tenantID uuid.UUID) ([]*model.IdentityProfile, error)
}

// MappingStatusUpdater updates access mapping status.
type MappingStatusUpdater interface {
	UpdateStatus(ctx context.Context, mappingID uuid.UUID, status string) error
}

// PolicyEngine evaluates all enabled policies for a tenant and produces violations.
type PolicyEngine struct {
	policyRepo    PolicyRepository
	mappingRepo   MappingProvider
	profileRepo   ProfileProvider
	statusUpdater MappingStatusUpdater
	logger        zerolog.Logger
}

// NewPolicyEngine creates a new policy evaluation engine.
func NewPolicyEngine(
	policyRepo PolicyRepository,
	mappingRepo MappingProvider,
	profileRepo ProfileProvider,
	statusUpdater MappingStatusUpdater,
	logger zerolog.Logger,
) *PolicyEngine {
	return &PolicyEngine{
		policyRepo:    policyRepo,
		mappingRepo:   mappingRepo,
		profileRepo:   profileRepo,
		statusUpdater: statusUpdater,
		logger:        logger.With().Str("component", "policy_engine").Logger(),
	}
}

// Evaluate loads all enabled policies and checks for violations across all mappings.
func (e *PolicyEngine) Evaluate(ctx context.Context, tenantID uuid.UUID) ([]model.PolicyViolation, error) {
	policies, err := e.policyRepo.ListEnabled(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if len(policies) == 0 {
		return nil, nil
	}

	mappings, err := e.mappingRepo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	profiles, err := e.profileRepo.ListActive(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var violations []model.PolicyViolation

	for _, policy := range policies {
		switch policy.PolicyType {
		case "max_idle_days":
			v := e.evaluateMaxIdleDays(ctx, policy, mappings, now)
			violations = append(violations, v...)
		case "classification_restrict":
			v := e.evaluateClassificationRestrict(policy, mappings, now)
			violations = append(violations, v...)
		case "separation_of_duties":
			v := e.evaluateSeparationOfDuties(policy, mappings, now)
			violations = append(violations, v...)
		case "time_bound_access":
			v := e.evaluateTimeBoundAccess(policy, mappings, now)
			violations = append(violations, v...)
		case "blast_radius_limit":
			v := e.evaluateBlastRadiusLimit(policy, profiles, now)
			violations = append(violations, v...)
		case "periodic_review":
			v := e.evaluatePeriodicReview(policy, profiles, now)
			violations = append(violations, v...)
		}
	}

	return violations, nil
}

func (e *PolicyEngine) evaluateMaxIdleDays(ctx context.Context, policy model.AccessPolicy, mappings []*model.AccessMapping, now time.Time) []model.PolicyViolation {
	var cfg model.MaxIdleDaysConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid max_idle_days config")
		return nil
	}

	cutoff := now.Add(-time.Duration(cfg.MaxDays) * 24 * time.Hour)
	minRank := model.ClassificationRank(cfg.ClassificationMin)

	var violations []model.PolicyViolation
	for _, m := range mappings {
		if model.ClassificationRank(m.DataClassification) < minRank {
			continue
		}
		isIdle := m.LastUsedAt == nil || m.LastUsedAt.Before(cutoff)
		if !isIdle {
			continue
		}

		violation := model.PolicyViolation{
			PolicyID:      policy.ID,
			PolicyName:    policy.Name,
			PolicyType:    policy.PolicyType,
			Enforcement:   policy.Enforcement,
			Severity:      policy.Severity,
			IdentityType:  m.IdentityType,
			IdentityID:    m.IdentityID,
			IdentityName:  m.IdentityName,
			MappingID:     &m.ID,
			DataAssetName: m.DataAssetName,
			ViolationType: "idle_permission",
			Description:   fmt.Sprintf("Permission %s on %s idle for more than %d days", m.PermissionType, m.DataAssetName, cfg.MaxDays),
			DetectedAt:    now,
		}
		violations = append(violations, violation)

		// Enforce.
		if cfg.AutoRevoke && policy.Enforcement == "auto_remediate" {
			if err := e.statusUpdater.UpdateStatus(ctx, m.ID, "expired"); err != nil {
				e.logger.Warn().Err(err).Msg("failed to auto-revoke idle mapping")
			}
		}
		if policy.Enforcement == "block" {
			if err := e.statusUpdater.UpdateStatus(ctx, m.ID, "pending_review"); err != nil {
				e.logger.Warn().Err(err).Msg("failed to block idle mapping")
			}
		}
	}
	return violations
}

func (e *PolicyEngine) evaluateClassificationRestrict(policy model.AccessPolicy, mappings []*model.AccessMapping, now time.Time) []model.PolicyViolation {
	var cfg model.ClassificationRestrictConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid classification_restrict config")
		return nil
	}

	allowedSet := make(map[string]bool)
	for _, t := range cfg.AllowedIdentityTypes {
		allowedSet[t] = true
	}

	var violations []model.PolicyViolation
	for _, m := range mappings {
		if m.DataClassification != cfg.Classification {
			continue
		}
		if allowedSet[m.IdentityType] {
			continue
		}
		violations = append(violations, model.PolicyViolation{
			PolicyID:      policy.ID,
			PolicyName:    policy.Name,
			PolicyType:    policy.PolicyType,
			Enforcement:   policy.Enforcement,
			Severity:      policy.Severity,
			IdentityType:  m.IdentityType,
			IdentityID:    m.IdentityID,
			IdentityName:  m.IdentityName,
			MappingID:     &m.ID,
			DataAssetName: m.DataAssetName,
			ViolationType: "unauthorized_access",
			Description:   fmt.Sprintf("%s '%s' is not authorized to access %s data (%s)", m.IdentityType, m.IdentityName, cfg.Classification, m.DataAssetName),
			DetectedAt:    now,
		})
	}
	return violations
}

func (e *PolicyEngine) evaluateSeparationOfDuties(policy model.AccessPolicy, mappings []*model.AccessMapping, now time.Time) []model.PolicyViolation {
	var cfg model.SeparationOfDutiesConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid separation_of_duties config")
		return nil
	}

	minRank := model.ClassificationRank(cfg.ClassificationMin)

	// Group permissions by identity + asset.
	type key struct {
		identityKey string
		assetID     string
	}
	permMap := make(map[key]map[string]*model.AccessMapping)
	for _, m := range mappings {
		if model.ClassificationRank(m.DataClassification) < minRank {
			continue
		}
		k := key{identityKey: m.IdentityType + "|" + m.IdentityID, assetID: m.DataAssetID.String()}
		if permMap[k] == nil {
			permMap[k] = make(map[string]*model.AccessMapping)
		}
		permMap[k][m.PermissionType] = m
	}

	var violations []model.PolicyViolation
	for _, perms := range permMap {
		for _, conflictPair := range cfg.ConflictingPermissions {
			if len(conflictPair) != 2 {
				continue
			}
			m1, has1 := perms[conflictPair[0]]
			_, has2 := perms[conflictPair[1]]
			if has1 && has2 {
				violations = append(violations, model.PolicyViolation{
					PolicyID:      policy.ID,
					PolicyName:    policy.Name,
					PolicyType:    policy.PolicyType,
					Enforcement:   policy.Enforcement,
					Severity:      policy.Severity,
					IdentityType:  m1.IdentityType,
					IdentityID:    m1.IdentityID,
					IdentityName:  m1.IdentityName,
					DataAssetName: m1.DataAssetName,
					ViolationType: "separation_of_duties",
					Description:   fmt.Sprintf("Identity has conflicting permissions [%s, %s] on %s", conflictPair[0], conflictPair[1], m1.DataAssetName),
					DetectedAt:    now,
				})
			}
		}
	}
	return violations
}

func (e *PolicyEngine) evaluateTimeBoundAccess(policy model.AccessPolicy, mappings []*model.AccessMapping, now time.Time) []model.PolicyViolation {
	var cfg model.TimeBoundAccessConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid time_bound_access config")
		return nil
	}

	minRank := model.ClassificationRank(cfg.ClassificationMin)

	var violations []model.PolicyViolation
	for _, m := range mappings {
		if model.ClassificationRank(m.DataClassification) < minRank {
			continue
		}
		if m.ExpiresAt != nil {
			continue // Already time-bound.
		}
		violations = append(violations, model.PolicyViolation{
			PolicyID:      policy.ID,
			PolicyName:    policy.Name,
			PolicyType:    policy.PolicyType,
			Enforcement:   policy.Enforcement,
			Severity:      policy.Severity,
			IdentityType:  m.IdentityType,
			IdentityID:    m.IdentityID,
			IdentityName:  m.IdentityName,
			MappingID:     &m.ID,
			DataAssetName: m.DataAssetName,
			ViolationType: "indefinite_grant",
			Description:   fmt.Sprintf("Indefinite %s grant on %s %s — should be time-bound (max %d days)", m.PermissionType, m.DataClassification, m.DataAssetName, cfg.MaxGrantDays),
			DetectedAt:    now,
		})
	}
	return violations
}

func (e *PolicyEngine) evaluateBlastRadiusLimit(policy model.AccessPolicy, profiles []*model.IdentityProfile, now time.Time) []model.PolicyViolation {
	var cfg model.BlastRadiusLimitConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid blast_radius_limit config")
		return nil
	}

	var violations []model.PolicyViolation
	for _, p := range profiles {
		if p.BlastRadiusScore <= cfg.MaxScore {
			continue
		}
		violations = append(violations, model.PolicyViolation{
			PolicyID:      policy.ID,
			PolicyName:    policy.Name,
			PolicyType:    policy.PolicyType,
			Enforcement:   policy.Enforcement,
			Severity:      cfg.AlertSeverity,
			IdentityType:  p.IdentityType,
			IdentityID:    p.IdentityID,
			IdentityName:  p.IdentityName,
			ViolationType: "blast_radius_exceeded",
			Description:   fmt.Sprintf("Identity blast radius score %.1f exceeds threshold %.1f", p.BlastRadiusScore, cfg.MaxScore),
			DetectedAt:    now,
		})
	}
	return violations
}

func (e *PolicyEngine) evaluatePeriodicReview(policy model.AccessPolicy, profiles []*model.IdentityProfile, now time.Time) []model.PolicyViolation {
	var cfg model.PeriodicReviewConfig
	if err := json.Unmarshal(policy.RuleConfig, &cfg); err != nil {
		e.logger.Warn().Err(err).Str("policy", policy.Name).Msg("invalid periodic_review config")
		return nil
	}

	var violations []model.PolicyViolation
	for _, p := range profiles {
		if p.NextReviewDue == nil || p.NextReviewDue.After(now) {
			continue
		}
		daysOverdue := int(now.Sub(*p.NextReviewDue).Hours() / 24)
		violations = append(violations, model.PolicyViolation{
			PolicyID:      policy.ID,
			PolicyName:    policy.Name,
			PolicyType:    policy.PolicyType,
			Enforcement:   policy.Enforcement,
			Severity:      policy.Severity,
			IdentityType:  p.IdentityType,
			IdentityID:    p.IdentityID,
			IdentityName:  p.IdentityName,
			ViolationType: "review_overdue",
			Description:   fmt.Sprintf("Access review overdue by %d days for %s", daysOverdue, p.IdentityName),
			DetectedAt:    now,
		})
	}
	return violations
}
