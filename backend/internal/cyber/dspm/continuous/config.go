package continuous

import "time"

// Config holds configuration for the continuous DSPM scanning engine.
type Config struct {
	// AtRestScanInterval is how often at-rest scans run (default: 24 hours).
	AtRestScanInterval time.Duration

	// ShadowScanInterval is how often shadow copy detection runs (default: 7 days).
	ShadowScanInterval time.Duration

	// ShadowSimilarityThreshold is the minimum similarity for shadow detection (default: 0.8).
	ShadowSimilarityThreshold float64

	// DriftAlertEnabled controls whether classification drift triggers alerts.
	DriftAlertEnabled bool

	// TransitEncryptionRequired controls whether unencrypted transit raises alerts.
	TransitEncryptionRequired bool

	// PipelineApprovalRequired controls whether unapproved pipelines raise alerts.
	PipelineApprovalRequired bool
}

// DefaultConfig returns sensible defaults for continuous scanning.
func DefaultConfig() Config {
	return Config{
		AtRestScanInterval:        24 * time.Hour,
		ShadowScanInterval:        7 * 24 * time.Hour,
		ShadowSimilarityThreshold: 0.8,
		DriftAlertEnabled:         true,
		TransitEncryptionRequired: true,
		PipelineApprovalRequired:  true,
	}
}
