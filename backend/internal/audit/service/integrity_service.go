package service

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/hash"
	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
)

// IntegrityService handles hash chain verification for audit logs.
type IntegrityService struct {
	repo   *repository.AuditRepository
	logger zerolog.Logger
}

// NewIntegrityService creates a new IntegrityService.
func NewIntegrityService(repo *repository.AuditRepository, logger zerolog.Logger) *IntegrityService {
	return &IntegrityService{repo: repo, logger: logger}
}

// VerifyChain loads entries for a tenant in [startTime, endTime] ordered by created_at ASC,
// recomputes each hash, and verifies chain integrity.
//
// The returned ChainVerificationResult is aligned with the frontend AuditVerificationResult
// interface, providing verified, total_records, verified_records, broken_chain_at,
// first_record, last_record, verification_hash, and verified_at.
func (s *IntegrityService) VerifyChain(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.ChainVerificationResult, error) {
	result := &model.ChainVerificationResult{
		VerifiedAt: time.Now().UTC().Format(time.RFC3339),
	}

	var previousHash string
	var isFirst = true
	var lastVerifiedHash string

	err := s.repo.StreamByTenant(ctx, tenantID, startTime, endTime, func(entry *model.AuditEntry) error {
		result.TotalRecords++

		if isFirst {
			result.FirstRecord = entry.ID
			previousHash = entry.PreviousHash
			isFirst = false
		}

		// Recompute hash
		expectedHash := hash.ComputeEntryHash(entry, previousHash)

		if expectedHash != entry.EntryHash {
			s.logger.Warn().
				Str("tenant_id", tenantID).
				Str("entry_id", entry.ID).
				Str("expected_hash", expectedHash).
				Str("stored_hash", entry.EntryHash).
				Msg("hash chain integrity violation detected")

			result.VerifiedRecords++
			result.BrokenChainAt = &entry.ID
			metrics.HashChainVerifications.WithLabelValues("broken").Inc()
			// Return nil to continue streaming — we count all records
			return nil
		}

		// Also verify the previous_hash link (skip for first entry in range)
		if !isFirst && result.BrokenChainAt == nil && entry.PreviousHash != previousHash {
			s.logger.Warn().
				Str("tenant_id", tenantID).
				Str("entry_id", entry.ID).
				Str("expected_previous", previousHash).
				Str("stored_previous", entry.PreviousHash).
				Msg("previous hash mismatch detected")

			result.VerifiedRecords++
			result.BrokenChainAt = &entry.ID
			metrics.HashChainVerifications.WithLabelValues("broken").Inc()
			return nil
		}

		previousHash = entry.EntryHash
		lastVerifiedHash = entry.EntryHash
		result.LastRecord = entry.ID
		result.VerifiedRecords++
		return nil
	})

	if err != nil {
		return nil, err
	}

	result.Verified = result.BrokenChainAt == nil
	result.VerificationHash = lastVerifiedHash

	if result.Verified {
		metrics.HashChainVerifications.WithLabelValues("ok").Inc()
	}

	s.logger.Info().
		Str("tenant_id", tenantID).
		Bool("verified", result.Verified).
		Int64("total_records", result.TotalRecords).
		Int64("verified_records", result.VerifiedRecords).
		Msg("hash chain verification complete")

	return result, nil
}
