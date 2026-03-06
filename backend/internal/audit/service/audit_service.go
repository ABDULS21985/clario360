package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/hash"
	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
	"github.com/clario360/platform/internal/events"
)

// AuditService handles core audit log ingestion with hash chain computation,
// deduplication, and batch insertion.
type AuditService struct {
	repo    *repository.AuditRepository
	rdb     *redis.Client
	logger  zerolog.Logger

	batchSize   int
	batchWindow time.Duration

	mu      sync.Mutex
	buffer  []model.AuditEntry
	flushCh chan struct{}
	done    chan struct{}
}

// NewAuditService creates a new AuditService.
func NewAuditService(
	repo *repository.AuditRepository,
	rdb *redis.Client,
	logger zerolog.Logger,
	batchSize int,
	batchWindow time.Duration,
) *AuditService {
	s := &AuditService{
		repo:        repo,
		rdb:         rdb,
		logger:      logger,
		batchSize:   batchSize,
		batchWindow: batchWindow,
		buffer:      make([]model.AuditEntry, 0, batchSize),
		flushCh:     make(chan struct{}, 1),
		done:        make(chan struct{}),
	}
	return s
}

// Start begins the background batch flusher goroutine.
func (s *AuditService) Start(ctx context.Context) {
	go s.runFlusher(ctx)
}

// Ingest adds an audit entry to the buffer for batch insertion.
func (s *AuditService) Ingest(entry model.AuditEntry) {
	s.mu.Lock()
	s.buffer = append(s.buffer, entry)
	shouldFlush := len(s.buffer) >= s.batchSize
	s.mu.Unlock()

	if shouldFlush {
		s.triggerFlush()
	}
}

// Flush forces an immediate flush of the current buffer.
func (s *AuditService) Flush(ctx context.Context) error {
	s.mu.Lock()
	batch := s.buffer
	s.buffer = make([]model.AuditEntry, 0, s.batchSize)
	s.mu.Unlock()

	if len(batch) == 0 {
		return nil
	}

	return s.processBatch(ctx, batch)
}

// Stop flushes remaining entries and stops the flusher.
func (s *AuditService) Stop(ctx context.Context) error {
	close(s.done)
	return s.Flush(ctx)
}

func (s *AuditService) runFlusher(ctx context.Context) {
	ticker := time.NewTicker(s.batchWindow)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.done:
			return
		case <-ticker.C:
			if err := s.Flush(ctx); err != nil {
				s.logger.Error().Err(err).Msg("periodic flush failed")
			}
		case <-s.flushCh:
			if err := s.Flush(ctx); err != nil {
				s.logger.Error().Err(err).Msg("triggered flush failed")
			}
		}
	}
}

func (s *AuditService) triggerFlush() {
	select {
	case s.flushCh <- struct{}{}:
	default:
	}
}

// processBatch groups entries by tenant, computes hash chains, and batch inserts.
func (s *AuditService) processBatch(ctx context.Context, batch []model.AuditEntry) error {
	start := time.Now()
	metrics.BatchSize.Observe(float64(len(batch)))

	// Group entries by tenant
	tenantGroups := make(map[string][]model.AuditEntry)
	for i := range batch {
		tid := batch[i].TenantID
		tenantGroups[tid] = append(tenantGroups[tid], batch[i])
	}

	// Process each tenant's entries: sort by event time, compute hash chain
	var allEntries []model.AuditEntry
	for tenantID, entries := range tenantGroups {
		// Sort by created_at within tenant
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].CreatedAt.Before(entries[j].CreatedAt)
		})

		// Get last known hash for this tenant
		previousHash, err := s.getLastHash(ctx, tenantID)
		if err != nil {
			s.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("failed to get last hash")
			previousHash = hash.GenesisHash
		}

		// Compute hash chain for each entry
		for i := range entries {
			entries[i].PreviousHash = previousHash
			entries[i].EntryHash = hash.ComputeEntryHash(&entries[i], previousHash)
			previousHash = entries[i].EntryHash
		}

		// Update chain state
		lastEntry := entries[len(entries)-1]
		if err := s.updateChainState(ctx, tenantID, lastEntry.ID, lastEntry.EntryHash, lastEntry.CreatedAt); err != nil {
			s.logger.Error().Err(err).Str("tenant_id", tenantID).Msg("failed to update chain state")
		}

		allEntries = append(allEntries, entries...)
	}

	// Batch insert all entries
	inserted, err := s.repo.BatchInsert(ctx, allEntries)
	if err != nil {
		metrics.EventsIngested.WithLabelValues("error").Add(float64(len(allEntries)))
		return fmt.Errorf("batch insert: %w", err)
	}

	duplicates := int64(len(allEntries)) - inserted
	metrics.EventsIngested.WithLabelValues("ok").Add(float64(inserted))
	metrics.EventsIngested.WithLabelValues("duplicate").Add(float64(duplicates))
	metrics.BatchInsertDuration.Observe(time.Since(start).Seconds())

	s.logger.Debug().
		Int64("inserted", inserted).
		Int64("duplicates", duplicates).
		Int("batch_total", len(allEntries)).
		Dur("duration", time.Since(start)).
		Msg("batch insert completed")

	return nil
}

// getLastHash retrieves the last hash for a tenant from Redis, falling back to DB.
func (s *AuditService) getLastHash(ctx context.Context, tenantID string) (string, error) {
	key := fmt.Sprintf("audit:chain:%s", tenantID)

	// Try Redis first
	val, err := s.rdb.Get(ctx, key).Result()
	if err == nil {
		var state chainCacheEntry
		if err := json.Unmarshal([]byte(val), &state); err == nil {
			return state.LastHash, nil
		}
	}

	// Fall back to DB
	chainState, err := s.repo.GetChainState(ctx, tenantID)
	if err != nil {
		return hash.GenesisHash, err
	}
	if chainState == nil {
		return hash.GenesisHash, nil
	}

	// Backfill Redis
	s.cacheChainState(ctx, tenantID, chainState.LastEntryID, chainState.LastHash, chainState.LastCreated)

	return chainState.LastHash, nil
}

// updateChainState updates both Redis and DB with the new chain state.
func (s *AuditService) updateChainState(ctx context.Context, tenantID, entryID, entryHash string, createdAt time.Time) error {
	// Update DB
	cs := &model.ChainState{
		TenantID:    tenantID,
		LastEntryID: entryID,
		LastHash:    entryHash,
		LastCreated: createdAt,
	}
	if err := s.repo.UpsertChainState(ctx, cs); err != nil {
		return err
	}

	// Update Redis
	s.cacheChainState(ctx, tenantID, entryID, entryHash, createdAt)
	return nil
}

func (s *AuditService) cacheChainState(ctx context.Context, tenantID, entryID, lastHash string, createdAt time.Time) {
	key := fmt.Sprintf("audit:chain:%s", tenantID)
	entry := chainCacheEntry{
		LastHash:    lastHash,
		LastEntryID: entryID,
		LastCreated: createdAt.Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(entry)
	if err != nil {
		s.logger.Warn().Err(err).Msg("failed to marshal chain cache entry")
		return
	}
	s.rdb.Set(ctx, key, data, 24*time.Hour)
}

type chainCacheEntry struct {
	LastHash    string `json:"last_hash"`
	LastEntryID string `json:"last_entry_id"`
	LastCreated string `json:"last_created_at"`
}

// IngestFromEvent creates an AuditEntry from a mapped event and ingests it.
func (s *AuditService) IngestFromEvent(entry model.AuditEntry) {
	if entry.ID == "" {
		entry.ID = events.GenerateUUID()
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	s.Ingest(entry)
}
