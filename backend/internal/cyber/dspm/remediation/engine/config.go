package engine

import (
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
)

// Config holds configuration for the DSPM remediation engine.
type Config struct {
	// SLAConfig defines severity-to-SLA-hours mappings for remediation deadlines.
	SLAConfig model.SLAConfig

	// EnableAutoRemediation enables the engine to automatically create remediations
	// from policy violations that have enforcement mode set to auto_remediate.
	EnableAutoRemediation bool

	// EnableSLAChecking enables periodic SLA breach detection. When enabled, the
	// scheduler will scan for remediations that have exceeded their SLA deadline
	// and mark them as breached.
	EnableSLAChecking bool

	// ScheduleIntervalMinutes is the interval in minutes between scheduler runs
	// for periodic tasks such as SLA checking, policy evaluation, and exception
	// expiry detection.
	ScheduleIntervalMinutes int

	// MaxConcurrentRemediations limits the number of remediations that can be
	// executing simultaneously. This prevents resource exhaustion when many
	// policy violations are detected at once.
	MaxConcurrentRemediations int
}

// DefaultConfig returns a Config with sensible production defaults.
//
//   - Auto-remediation is enabled so that policy violations with auto_remediate
//     enforcement mode are automatically handled.
//   - SLA checking is enabled to ensure breaches are detected promptly.
//   - The scheduler runs every 15 minutes to balance responsiveness with resource usage.
//   - A maximum of 10 concurrent remediations prevents thundering-herd scenarios.
func DefaultConfig() Config {
	return Config{
		SLAConfig:                 model.DefaultSLAConfig(),
		EnableAutoRemediation:     true,
		EnableSLAChecking:         true,
		ScheduleIntervalMinutes:   15,
		MaxConcurrentRemediations: 10,
	}
}
