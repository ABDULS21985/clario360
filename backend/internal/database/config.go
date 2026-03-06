package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig holds pgxpool connection configuration.
type PoolConfig struct {
	URL               string        // postgres://...
	MinConns          int           // default 5
	MaxConns          int           // default 20
	MaxConnLife       time.Duration // default 1h
	MaxConnIdle       time.Duration // default 30m
	HealthCheckPeriod time.Duration // default 1m
}

// Connect creates a pgxpool.Pool from the given configuration.
// It parses the URL, applies pool settings, connects, and runs Ping() to verify.
// Returns an error if the connection fails (fail-fast).
func Connect(ctx context.Context, cfg PoolConfig) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing database URL: %w", err)
	}

	if cfg.MinConns > 0 {
		poolCfg.MinConns = int32(cfg.MinConns)
	} else {
		poolCfg.MinConns = 5
	}
	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = int32(cfg.MaxConns)
	} else {
		poolCfg.MaxConns = 20
	}
	if cfg.MaxConnLife > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLife
	} else {
		poolCfg.MaxConnLifetime = 1 * time.Hour
	}
	if cfg.MaxConnIdle > 0 {
		poolCfg.MaxConnIdleTime = cfg.MaxConnIdle
	} else {
		poolCfg.MaxConnIdleTime = 30 * time.Minute
	}
	if cfg.HealthCheckPeriod > 0 {
		poolCfg.HealthCheckPeriod = cfg.HealthCheckPeriod
	} else {
		poolCfg.HealthCheckPeriod = 1 * time.Minute
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return pool, nil
}
