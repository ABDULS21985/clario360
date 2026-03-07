package strategy

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// managementCIDRs are internal ranges that must not be blocked.
var managementCIDRs = []string{
	"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16",
	"127.0.0.0/8", "169.254.0.0/16",
}

// BlockStrategy handles IP/network block remediations.
type BlockStrategy struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewBlockStrategy creates a BlockStrategy.
func NewBlockStrategy(db *pgxpool.Pool, logger zerolog.Logger) *BlockStrategy {
	return &BlockStrategy{db: db, logger: logger}
}

func (s *BlockStrategy) Type() model.RemediationType { return model.RemediationTypeBlockIP }

func (s *BlockStrategy) DryRun(ctx context.Context, action *model.RemediationAction) (*model.DryRunResult, error) {
	start := time.Now()
	result := &model.DryRunResult{
		Success:          true,
		SimulatedChanges: make([]model.SimulatedChange, 0),
		Warnings:         make([]string, 0),
		Blockers:         make([]string, 0),
		AffectedServices: make([]string, 0),
	}

	targets := action.Plan.BlockTargets
	if len(targets) == 0 {
		result.Blockers = append(result.Blockers, "no block targets specified in plan")
		result.Success = false
		result.DurationMs = time.Since(start).Milliseconds()
		return result, nil
	}

	for _, target := range targets {
		ip := net.ParseIP(target)
		if ip == nil {
			_, _, err := net.ParseCIDR(target)
			if err != nil {
				result.Blockers = append(result.Blockers, fmt.Sprintf("invalid IP/CIDR: %s", target))
				result.Success = false
				continue
			}
		}

		// Safety check: warn about internal management IPs
		if isInternalIP(target) {
			result.Warnings = append(result.Warnings, fmt.Sprintf("WARNING: %s is an internal address — blocking may disrupt management access", target))
		}

		// Check for existing allow-list entries
		var allowCount int
		_ = s.db.QueryRow(ctx,
			"SELECT COUNT(*) FROM threat_indicators WHERE value=$1 AND type='ip' AND active=false AND tenant_id=$2",
			target, action.TenantID,
		).Scan(&allowCount)
		if allowCount > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s is in allow-list (inactive indicator) — review before blocking", target))
		}

		// Check for recent connections from this IP
		var recentEvents int
		_ = s.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM security_events
			WHERE tenant_id=$1 AND (source_ip=$2 OR destination_ip=$2)
			AND occurred_at > now() - interval '24 hours'`,
			action.TenantID, target,
		).Scan(&recentEvents)
		if recentEvents > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s had %d connection events in the last 24h — verify this is not legitimate traffic", target, recentEvents))
		}

		result.SimulatedChanges = append(result.SimulatedChanges, model.SimulatedChange{
			ChangeType:  "ip_block",
			Description: fmt.Sprintf("Block IP/CIDR %s", target),
			AfterValue:  "blocked",
		})
	}

	if len(result.Blockers) == 0 {
		result.Success = true
	}
	result.EstimatedImpact = model.ImpactEstimate{
		Downtime:         "0",
		ServicesAffected: 0,
		RiskLevel:        "low",
		RecommendWindow:  "immediate",
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *BlockStrategy) Execute(ctx context.Context, action *model.RemediationAction) (*model.ExecutionResult, error) {
	start := time.Now()
	result := &model.ExecutionResult{
		StepsTotal:     len(action.Plan.BlockTargets),
		StepResults:    make([]model.StepResult, 0),
		ChangesApplied: make([]model.AppliedChange, 0),
	}

	for i, target := range action.Plan.BlockTargets {
		stepStart := time.Now()
		sr := model.StepResult{
			StepNumber: i + 1,
			Action:     fmt.Sprintf("block %s", target),
		}

		// Activate threat indicator for this IP (creates or activates)
		_, err := s.db.Exec(ctx, `
			INSERT INTO threat_indicators (
				id, tenant_id, threat_id, type, value, confidence, active,
				description, source, created_at, updated_at
			)
			SELECT gen_random_uuid(), $1, NULL, 'ip', $2, 90, true,
			       'Blocked by remediation action', 'remediation', now(), now()
			WHERE NOT EXISTS (
				SELECT 1 FROM threat_indicators WHERE tenant_id=$1 AND value=$2 AND type='ip'
			)`,
			action.TenantID, target,
		)
		if err != nil {
			sr.Status = "failure"
			sr.Error = fmt.Sprintf("failed to create block rule for %s: %v", target, err)
			result.StepResults = append(result.StepResults, sr)
			result.Success = false
			result.DurationMs = time.Since(start).Milliseconds()
			return result, nil
		}
		// Activate any existing indicator
		_, _ = s.db.Exec(ctx,
			"UPDATE threat_indicators SET active=true, updated_at=now() WHERE tenant_id=$1 AND value=$2 AND type='ip'",
			action.TenantID, target,
		)

		sr.Status = "success"
		sr.Output = fmt.Sprintf("Block rule activated for %s", target)
		sr.DurationMs = time.Since(stepStart).Milliseconds()
		result.StepResults = append(result.StepResults, sr)
		result.StepsExecuted++
		result.ChangesApplied = append(result.ChangesApplied, model.AppliedChange{
			ChangeType:  "ip_blocked",
			Description: fmt.Sprintf("IP %s blocked via threat indicator", target),
			NewValue:    "active",
		})
	}

	result.Success = true
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *BlockStrategy) Verify(ctx context.Context, action *model.RemediationAction) (*model.VerificationResult, error) {
	start := time.Now()
	result := &model.VerificationResult{
		Checks: make([]model.VerificationCheck, 0),
	}

	allPassed := true
	for _, target := range action.Plan.BlockTargets {
		var recentEvents int
		_ = s.db.QueryRow(ctx, `
			SELECT COUNT(*) FROM security_events
			WHERE tenant_id=$1 AND (source_ip=$2 OR destination_ip=$2)
			AND occurred_at > now() - interval '1 hour'`,
			action.TenantID, target,
		).Scan(&recentEvents)

		check := model.VerificationCheck{
			Name:     fmt.Sprintf("Traffic check for %s", target),
			Expected: "no security events from blocked IP in last hour",
			Actual:   fmt.Sprintf("%d events detected", recentEvents),
			Passed:   recentEvents == 0,
		}
		if !check.Passed {
			allPassed = false
			check.Notes = "Traffic still detected from blocked IP — block rule may not be effective"
		}
		result.Checks = append(result.Checks, check)
	}

	result.Verified = allPassed
	if !allPassed {
		result.FailureReason = "Traffic still detected from one or more blocked IPs"
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result, nil
}

func (s *BlockStrategy) Rollback(ctx context.Context, action *model.RemediationAction) error {
	for _, target := range action.Plan.BlockTargets {
		_, err := s.db.Exec(ctx,
			"UPDATE threat_indicators SET active=false, updated_at=now() WHERE tenant_id=$1 AND value=$2 AND type='ip'",
			action.TenantID, target,
		)
		if err != nil {
			return fmt.Errorf("deactivate block indicator for %s: %w", target, err)
		}
	}
	return nil
}

func (s *BlockStrategy) CaptureState(ctx context.Context, action *model.RemediationAction) (json.RawMessage, error) {
	type indicatorState struct {
		IP     string `json:"ip"`
		Active bool   `json:"active"`
		Exists bool   `json:"exists"`
	}

	states := make([]indicatorState, 0, len(action.Plan.BlockTargets))
	for _, target := range action.Plan.BlockTargets {
		st := indicatorState{IP: target}
		var active bool
		err := s.db.QueryRow(ctx,
			"SELECT active FROM threat_indicators WHERE tenant_id=$1 AND value=$2 AND type='ip' LIMIT 1",
			action.TenantID, target,
		).Scan(&active)
		if err == nil {
			st.Exists = true
			st.Active = active
		}
		states = append(states, st)
	}
	return json.Marshal(map[string]interface{}{"indicators": states, "captured_at": time.Now().UTC()})
}

func isInternalIP(target string) bool {
	ip := net.ParseIP(target)
	if ip != nil {
		for _, cidr := range managementCIDRs {
			_, network, _ := net.ParseCIDR(cidr)
			if network != nil && network.Contains(ip) {
				return true
			}
		}
		return false
	}
	// For CIDR ranges, check if it's a private CIDR
	_, network, err := net.ParseCIDR(target)
	if err != nil {
		return false
	}
	for _, cidr := range managementCIDRs {
		_, mgmt, _ := net.ParseCIDR(cidr)
		if mgmt != nil && mgmt.Contains(network.IP) {
			return true
		}
	}
	return false
}
