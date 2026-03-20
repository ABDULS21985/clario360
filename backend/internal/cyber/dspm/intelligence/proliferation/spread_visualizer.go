package proliferation

import (
	"sort"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// SpreadVisualizer generates spread trend data from proliferation events
// for visualization in dashboards and reports.
type SpreadVisualizer struct {
	logger zerolog.Logger
}

// NewSpreadVisualizer creates a new spread visualizer instance.
func NewSpreadVisualizer(logger zerolog.Logger) *SpreadVisualizer {
	return &SpreadVisualizer{
		logger: logger.With().Str("component", "spread_visualizer").Logger(),
	}
}

// BuildSpreadTrend generates a time-series of spread trend points over the
// specified number of days. Each point shows the total copies, new copies
// discovered that day, and how many of those new copies were unauthorized.
func (v *SpreadVisualizer) BuildSpreadTrend(events []model.SpreadEvent, days int) []model.SpreadTrendPoint {
	if days <= 0 {
		days = 30
	}

	now := time.Now().UTC()
	startDate := now.AddDate(0, 0, -days)

	// Sort events by detection time.
	sorted := make([]model.SpreadEvent, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].DetectedAt.Before(sorted[j].DetectedAt)
	})

	// Group events by date.
	type dailyCount struct {
		newCopies       int
		unauthorizedNew int
	}
	dailyCounts := make(map[string]*dailyCount)

	for _, event := range sorted {
		if event.DetectedAt.Before(startDate) {
			continue
		}
		dateKey := event.DetectedAt.Format("2006-01-02")
		dc, ok := dailyCounts[dateKey]
		if !ok {
			dc = &dailyCount{}
			dailyCounts[dateKey] = dc
		}
		dc.newCopies++
		if event.Status == model.ProliferationUnauthorized {
			dc.unauthorizedNew++
		}
	}

	// Count events before the window to establish the baseline.
	baselineCopies := 0
	for _, event := range sorted {
		if event.DetectedAt.Before(startDate) {
			baselineCopies++
		}
	}

	// Build trend points for each day in the window.
	var trend []model.SpreadTrendPoint
	runningTotal := baselineCopies

	for i := 0; i <= days; i++ {
		date := startDate.AddDate(0, 0, i)
		dateKey := date.Format("2006-01-02")

		point := model.SpreadTrendPoint{
			Date: dateKey,
		}

		if dc, ok := dailyCounts[dateKey]; ok {
			point.NewCopies = dc.newCopies
			point.UnauthorizedNew = dc.unauthorizedNew
			runningTotal += dc.newCopies
		}

		point.TotalCopies = runningTotal
		trend = append(trend, point)
	}

	v.logger.Debug().
		Int("days", days).
		Int("events", len(events)).
		Int("trend_points", len(trend)).
		Msg("spread trend built")

	return trend
}
