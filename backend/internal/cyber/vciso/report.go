package vciso

import (
	"context"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// ReportGenerator assembles on-demand security reports from the same structured data as executive briefings.
type ReportGenerator struct {
	briefing *BriefingGenerator
	logger   zerolog.Logger
}

// NewReportGenerator creates a report generator.
func NewReportGenerator(briefing *BriefingGenerator, logger zerolog.Logger) *ReportGenerator {
	return &ReportGenerator{
		briefing: briefing,
		logger:   logger.With().Str("component", "vciso-report").Logger(),
	}
}

// Generate produces a report payload for the requested type.
func (g *ReportGenerator) Generate(ctx context.Context, tenantID uuid.UUID, reportType string, periodDays int) (*model.ExecutiveBriefing, error) {
	// The current stored format is the structured executive briefing schema. The frontend can render
	// the same underlying content differently for executive, technical, and compliance audiences.
	return g.briefing.GenerateExecutiveBriefing(ctx, tenantID, periodDays)
}
