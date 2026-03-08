package engine

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type ToolRouter struct {
	registry *tools.ToolRegistry
	metrics  *VCISOMetrics
	logger   zerolog.Logger
}

func NewToolRouter(registry *tools.ToolRegistry, metrics *VCISOMetrics, logger zerolog.Logger) *ToolRouter {
	return &ToolRouter{
		registry: registry,
		metrics:  metrics,
		logger:   logger.With().Str("component", "vciso_tool_router").Logger(),
	}
}

func (r *ToolRouter) Get(name string) tools.Tool {
	if r == nil || r.registry == nil {
		return nil
	}
	return r.registry.Get(name)
}

func (r *ToolRouter) Execute(ctx context.Context, tool tools.Tool, tenantID uuid.UUID, userID uuid.UUID, params map[string]string, timeout time.Duration) (*tools.ToolResult, time.Duration, error) {
	if tool == nil {
		return nil, 0, errors.New("tool is required")
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	result, err := tool.Execute(toolCtx, tenantID, userID, params)
	latency := time.Since(start)

	if r.metrics != nil && r.metrics.ToolLatencySeconds != nil {
		r.metrics.ToolLatencySeconds.WithLabelValues(tool.Name()).Observe(latency.Seconds())
	}
	if err != nil {
		status := "error"
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(toolCtx.Err(), context.DeadlineExceeded) {
			status = "timeout"
			if r.metrics != nil && r.metrics.ToolTimeoutsTotal != nil {
				r.metrics.ToolTimeoutsTotal.WithLabelValues(tool.Name()).Inc()
			}
		}
		if r.metrics != nil && r.metrics.ToolExecutionsTotal != nil {
			r.metrics.ToolExecutionsTotal.WithLabelValues(tool.Name(), status).Inc()
		}
		return nil, latency, err
	}
	if r.metrics != nil && r.metrics.ToolExecutionsTotal != nil {
		r.metrics.ToolExecutionsTotal.WithLabelValues(tool.Name(), "success").Inc()
	}
	return result, latency, nil
}
