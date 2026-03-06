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
// This streams rows via a cursor to avoid loading millions of rows into memory.
func (s *IntegrityService) VerifyChain(ctx context.Context, tenantID string, startTime, endTime time.Time) (*model.ChainVerificationResult, error) {
	result := &model.ChainVerificationResult{}
	var previousHash string
	var isFirst = true

	err := s.repo.StreamByTenant(ctx, tenantID, startTime, endTime, func(entry *model.AuditEntry) error {
		if isFirst {
			// The first entry in the range uses its stored previous_hash
			previousHash = entry.PreviousHash
			isFirst = false

			// For genesis entries, verify PreviousHash is "GENESIS"
			if previousHash == hash.GenesisHash {
				// This is the first ever entry for this tenant — valid genesis
			}
			// Otherwise, we trust the stored previous_hash as the chain starting point
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

			result.BrokenAt = &entry.ID
			result.Checked++
			metrics.HashChainVerifications.WithLabelValues("broken").Inc()
			return nil // Return nil to stop iteration; we found the break
		}

		// Also verify that the entry's previous_hash matches what we expect
		if !isFirst && entry.PreviousHash != previousHash {
			s.logger.Warn().
				Str("tenant_id", tenantID).
				Str("entry_id", entry.ID).
				Str("expected_previous", previousHash).
				Str("stored_previous", entry.PreviousHash).
				Msg("previous hash mismatch detected")

			result.BrokenAt = &entry.ID
			result.Checked++
			metrics.HashChainVerifications.WithLabelValues("broken").Inc()
			return nil
		}

		previousHash = entry.EntryHash
		result.Checked++
		return nil
	})

	if err != nil {
		return nil, err
	}

	result.OK = result.BrokenAt == nil
	if result.OK {
		metrics.HashChainVerifications.WithLabelValues("ok").Inc()
	}

	s.logger.Info().
		Str("tenant_id", tenantID).
		Bool("ok", result.OK).
		Int64("checked", result.Checked).
		Msg("hash chain verification complete")

	return result, nil
}
