package enrichment

import (
	"context"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// Enricher is the interface all enrichers must implement.
type Enricher interface {
	Name() string
	Enrich(ctx context.Context, asset *model.Asset) (*EnrichmentResult, error)
}

// EnrichmentResult carries the outcome of a single enricher run.
type EnrichmentResult struct {
	EnricherName string
	FieldsAdded  []string
	Duration     time.Duration
	Error        error
}

// Pipeline runs a fixed sequence of Enricher implementations against an asset.
// Enrichers run sequentially; if one fails the error is logged and the next
// enricher still runs. The asset is modified in-place by each enricher.
type Pipeline struct {
	enrichers []Enricher
	logger    zerolog.Logger
}

// NewPipeline creates a pipeline with the supplied enrichers.
// Order matters: DNS runs before CVE so that hostname is available for CPE matching.
func NewPipeline(logger zerolog.Logger, enrichers ...Enricher) *Pipeline {
	return &Pipeline{enrichers: enrichers, logger: logger}
}

// Run executes all enrichers against asset and returns the combined results.
// It does NOT persist the asset — callers are responsible for saving.
func (p *Pipeline) Run(ctx context.Context, asset *model.Asset) []EnrichmentResult {
	results := make([]EnrichmentResult, 0, len(p.enrichers))
	for _, e := range p.enrichers {
		start := time.Now()
		result, err := e.Enrich(ctx, asset)
		elapsed := time.Since(start)

		if result == nil {
			result = &EnrichmentResult{EnricherName: e.Name()}
		}
		result.Duration = elapsed
		result.Error = err

		if err != nil {
			p.logger.Warn().
				Err(err).
				Str("enricher", e.Name()).
				Str("asset_id", asset.ID.String()).
				Msg("enricher failed, continuing pipeline")
		} else {
			p.logger.Debug().
				Str("enricher", e.Name()).
				Str("asset_id", asset.ID.String()).
				Strs("fields_added", result.FieldsAdded).
				Dur("duration", elapsed).
				Msg("enricher completed")
		}
		results = append(results, *result)
	}
	return results
}
