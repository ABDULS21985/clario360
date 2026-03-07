package scanner

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// CloudScanner discovers assets from cloud provider APIs (AWS/GCP/Azure).
// The current implementation is a structural skeleton; full cloud provider
// SDK integration requires service account credentials to be configured
// per environment and is out of scope for the base build.
type CloudScanner struct {
	repo   AssetUpsertRepo
	logger zerolog.Logger
}

// NewCloudScanner creates a CloudScanner.
func NewCloudScanner(repo AssetUpsertRepo, logger zerolog.Logger) *CloudScanner {
	return &CloudScanner{repo: repo, logger: logger}
}

// Type implements Scanner.
func (s *CloudScanner) Type() model.ScanType { return model.ScanTypeCloud }

// Scan initiates a cloud provider asset discovery.
// Full implementation requires AWS SDK (aws-sdk-go-v2), GCP client, or Azure SDK
// and appropriate IAM credentials. Returns an informative error so the caller
// knows the feature needs configuration, rather than silently returning empty results.
func (s *CloudScanner) Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error) {
	if len(cfg.Targets) == 0 {
		return &model.ScanResult{Status: model.ScanStatusFailed, Errors: []string{"no cloud targets specified"}},
			fmt.Errorf("cloud scan requires at least one account/project ID in targets")
	}

	s.logger.Info().
		Strs("targets", cfg.Targets).
		Msg("cloud scanner: feature requires cloud provider SDK configuration")

	// In a production deployment this would:
	// 1. Parse cfg.Options["provider"] → "aws"|"gcp"|"azure"
	// 2. Load provider credentials from env/secrets manager
	// 3. Call EC2/GCE/Azure Compute list-instances APIs
	// 4. Map each instance to a model.DiscoveredAsset
	// 5. Upsert each asset via s.repo.UpsertFromScan()

	return &model.ScanResult{
		Status: model.ScanStatusCompleted,
		Errors: []string{"cloud provider SDK integration not configured — no assets discovered"},
	}, nil
}
