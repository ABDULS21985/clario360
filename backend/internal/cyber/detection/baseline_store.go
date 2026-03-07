package detection

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

const baselineTTL = 90 * 24 * time.Hour

// BaselineStore persists anomaly baselines in Redis with an in-memory fallback.
type BaselineStore struct {
	redis  *redis.Client
	logger zerolog.Logger
	mu     sync.RWMutex
	local  map[string]*Baseline
}

// Baseline holds the running moments for a group value.
type Baseline struct {
	Mean        float64   `json:"mean"`
	Variance    float64   `json:"variance"`
	Count       int64     `json:"count"`
	LastUpdated time.Time `json:"last_updated"`
}

// StdDev returns the baseline standard deviation.
func (b *Baseline) StdDev() float64 {
	if b == nil || b.Variance <= 0 {
		return 0
	}
	return math.Sqrt(b.Variance)
}

// NewBaselineStore creates a new baseline store.
func NewBaselineStore(redisClient *redis.Client, logger zerolog.Logger) *BaselineStore {
	return &BaselineStore{
		redis:  redisClient,
		logger: logger,
		local:  make(map[string]*Baseline),
	}
}

// GetBaseline returns the stored baseline for a tenant/rule/group.
func (s *BaselineStore) GetBaseline(ctx context.Context, tenantID, ruleID uuid.UUID, groupValue string) (*Baseline, error) {
	key := baselineKey(tenantID, ruleID, groupValue)
	if s.redis != nil {
		payload, err := s.redis.Get(ctx, key).Result()
		if err == nil {
			var baseline Baseline
			if err := json.Unmarshal([]byte(payload), &baseline); err != nil {
				return nil, fmt.Errorf("unmarshal baseline: %w", err)
			}
			return &baseline, nil
		}
		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("get baseline: %w", err)
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if baseline, ok := s.local[key]; ok {
		copyValue := *baseline
		return &copyValue, nil
	}
	return &Baseline{}, nil
}

// StoreBaseline persists the given baseline.
func (s *BaselineStore) StoreBaseline(ctx context.Context, tenantID, ruleID uuid.UUID, groupValue string, baseline *Baseline) error {
	if baseline == nil {
		baseline = &Baseline{}
	}
	key := baselineKey(tenantID, ruleID, groupValue)
	payload, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	if s.redis != nil {
		if err := s.redis.Set(ctx, key, payload, baselineTTL).Err(); err != nil {
			return fmt.Errorf("set baseline: %w", err)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyValue := *baseline
	s.local[key] = &copyValue
	return nil
}

// UpdateBaseline updates the baseline using Welford's online algorithm.
func (s *BaselineStore) UpdateBaseline(ctx context.Context, tenantID, ruleID uuid.UUID, groupValue string, newValue float64) (*Baseline, error) {
	baseline, err := s.GetBaseline(ctx, tenantID, ruleID, groupValue)
	if err != nil {
		return nil, err
	}
	if baseline == nil {
		baseline = &Baseline{}
	}
	baseline.Count++
	delta := newValue - baseline.Mean
	baseline.Mean += delta / float64(baseline.Count)
	delta2 := newValue - baseline.Mean
	baseline.Variance += (delta*delta2 - baseline.Variance) / float64(baseline.Count)
	baseline.LastUpdated = time.Now().UTC()
	if err := s.StoreBaseline(ctx, tenantID, ruleID, groupValue, baseline); err != nil {
		return nil, err
	}
	return baseline, nil
}

func baselineKey(tenantID, ruleID uuid.UUID, groupValue string) string {
	return fmt.Sprintf("baseline:%s:%s:%s", tenantID, ruleID, groupValue)
}
