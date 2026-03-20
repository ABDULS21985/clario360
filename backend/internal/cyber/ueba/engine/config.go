package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

const configRedisKey = "cyber:ueba:config"

type UEBAConfig struct {
	CycleInterval       time.Duration         `json:"cycle_interval"`
	MaxEventsPerCycle   int                   `json:"max_events_per_cycle"`
	MaxProcessingTime   time.Duration         `json:"max_processing_time"`
	EMAAlpha            float64               `json:"ema_alpha"`
	MinMaturityForAlert model.ProfileMaturity `json:"min_maturity_for_alert"`
	CorrelationWindow   time.Duration         `json:"correlation_window"`
	RiskDecayRatePerDay float64               `json:"risk_decay_rate_per_day"`
	BatchSize           int                   `json:"batch_size"`

	UnusualTimeMatureHighProb   float64 `json:"unusual_time_mature_high_prob"`
	UnusualTimeMatureMediumProb float64 `json:"unusual_time_mature_medium_prob"`
	UnusualTimeBaseHighProb     float64 `json:"unusual_time_base_high_prob"`
	UnusualTimeBaseMediumProb   float64 `json:"unusual_time_base_medium_prob"`

	UnusualVolumeMediumZ   float64 `json:"unusual_volume_medium_z"`
	UnusualVolumeHighZ     float64 `json:"unusual_volume_high_z"`
	UnusualVolumeCriticalZ float64 `json:"unusual_volume_critical_z"`
	UnusualVolumeStddevMin float64 `json:"unusual_volume_stddev_min"`

	FailureSpikeMediumZ   float64 `json:"failure_spike_medium_z"`
	FailureSpikeHighZ     float64 `json:"failure_spike_high_z"`
	FailureSpikeCriticalZ float64 `json:"failure_spike_critical_z"`
	FailureStddevMin      float64 `json:"failure_stddev_min"`
	FailureCriticalCount  float64 `json:"failure_critical_count"`

	BulkRowsMediumMultiplier float64 `json:"bulk_rows_medium_multiplier"`
	BulkRowsHighMultiplier   float64 `json:"bulk_rows_high_multiplier"`
	DDLUnusualThreshold      float64 `json:"ddl_unusual_threshold"`
}

func DefaultConfig() UEBAConfig {
	return UEBAConfig{
		CycleInterval:       5 * time.Minute,
		MaxEventsPerCycle:   10000,
		MaxProcessingTime:   60 * time.Second,
		EMAAlpha:            0.05,
		MinMaturityForAlert: model.ProfileMaturityBaseline,
		CorrelationWindow:   time.Hour,
		RiskDecayRatePerDay: 0.10,
		BatchSize:           500,

		UnusualTimeMatureHighProb:   0.005,
		UnusualTimeMatureMediumProb: 0.02,
		UnusualTimeBaseHighProb:     0.002,
		UnusualTimeBaseMediumProb:   0.01,

		UnusualVolumeMediumZ:   3,
		UnusualVolumeHighZ:     4,
		UnusualVolumeCriticalZ: 5,
		UnusualVolumeStddevMin: 1000,

		FailureSpikeMediumZ:   3,
		FailureSpikeHighZ:     5,
		FailureSpikeCriticalZ: 8,
		FailureStddevMin:      1,
		FailureCriticalCount:  50,

		BulkRowsMediumMultiplier: 5,
		BulkRowsHighMultiplier:   10,
		DDLUnusualThreshold:      0.05,
	}
}

func (c UEBAConfig) Validate() error {
	switch c.MinMaturityForAlert {
	case model.ProfileMaturityBaseline, model.ProfileMaturityMature:
	default:
		return fmt.Errorf("min_maturity_for_alert must be baseline or mature")
	}
	if c.CycleInterval <= 0 {
		return fmt.Errorf("cycle_interval must be positive")
	}
	if c.MaxEventsPerCycle <= 0 {
		return fmt.Errorf("max_events_per_cycle must be positive")
	}
	if c.MaxProcessingTime <= 0 {
		return fmt.Errorf("max_processing_time must be positive")
	}
	if c.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be positive")
	}
	if c.EMAAlpha <= 0 || c.EMAAlpha >= 1 {
		return fmt.Errorf("ema_alpha must be between 0 and 1")
	}
	if c.CorrelationWindow <= 0 {
		return fmt.Errorf("correlation_window must be positive")
	}
	if c.RiskDecayRatePerDay <= 0 || c.RiskDecayRatePerDay >= 1 {
		return fmt.Errorf("risk_decay_rate_per_day must be between 0 and 1")
	}
	if !(c.UnusualTimeMatureHighProb < c.UnusualTimeMatureMediumProb &&
		c.UnusualTimeBaseHighProb < c.UnusualTimeBaseMediumProb) {
		return fmt.Errorf("unusual time high probability must be lower than medium probability")
	}
	if !(c.UnusualVolumeMediumZ < c.UnusualVolumeHighZ && c.UnusualVolumeHighZ < c.UnusualVolumeCriticalZ) {
		return fmt.Errorf("unusual volume z thresholds must increase with severity")
	}
	if !(c.FailureSpikeMediumZ < c.FailureSpikeHighZ && c.FailureSpikeHighZ < c.FailureSpikeCriticalZ) {
		return fmt.Errorf("failure spike z thresholds must increase with severity")
	}
	if !(c.BulkRowsMediumMultiplier < c.BulkRowsHighMultiplier) {
		return fmt.Errorf("bulk row multipliers must increase with severity")
	}
	if c.DDLUnusualThreshold <= 0 || c.DDLUnusualThreshold >= 1 {
		return fmt.Errorf("ddl_unusual_threshold must be between 0 and 1")
	}
	return nil
}

type ConfigStore struct {
	mu     sync.RWMutex
	config UEBAConfig
	redis  *redis.Client
	logger zerolog.Logger
}

func NewConfigStore(ctx context.Context, redisClient *redis.Client, logger zerolog.Logger) *ConfigStore {
	store := &ConfigStore{
		config: DefaultConfig(),
		redis:  redisClient,
		logger: logger.With().Str("component", "ueba-config-store").Logger(),
	}
	_ = store.Reload(ctx)
	return store
}

func (s *ConfigStore) Snapshot() UEBAConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *ConfigStore) Reload(ctx context.Context) error {
	if s.redis == nil {
		return nil
	}
	payload, err := s.redis.Get(ctx, configRedisKey).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return fmt.Errorf("load ueba config: %w", err)
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return fmt.Errorf("decode ueba config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("validate persisted ueba config: %w", err)
	}
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()
	return nil
}

func (s *ConfigStore) Update(ctx context.Context, cfg UEBAConfig) (UEBAConfig, error) {
	if err := cfg.Validate(); err != nil {
		return UEBAConfig{}, err
	}
	if s.redis != nil {
		payload, err := json.Marshal(cfg)
		if err != nil {
			return UEBAConfig{}, fmt.Errorf("encode ueba config: %w", err)
		}
		if err := s.redis.Set(ctx, configRedisKey, payload, 0).Err(); err != nil {
			return UEBAConfig{}, fmt.Errorf("persist ueba config: %w", err)
		}
	}
	s.mu.Lock()
	s.config = cfg
	s.mu.Unlock()
	return cfg, nil
}
