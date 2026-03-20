package integration

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/events"
)

func NewCyberRuntime(ctx context.Context, cfg *config.Config, reg prometheus.Registerer, producer *events.Producer, logger zerolog.Logger) (*Runtime, error) {
	return NewRuntime(ctx, cfg, reg, producer, logger)
}
