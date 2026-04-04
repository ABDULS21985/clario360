package proliferation

import (
	"sort"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dspm/intelligence/model"
)

// classificationSeverity maps classification levels to severity ranks.
// Higher values indicate more sensitive data.
var classificationSeverity = map[string]int{
	"public":       0,
	"internal":     1,
	"confidential": 2,
	"restricted":   3,
}

// DriftAnalyzer detects and categorizes classification drift in data assets
// by analyzing classification history records over time.
type DriftAnalyzer struct {
	logger zerolog.Logger
}

// NewDriftAnalyzer creates a new drift analyzer instance.
func NewDriftAnalyzer(logger zerolog.Logger) *DriftAnalyzer {
	return &DriftAnalyzer{
		logger: logger.With().Str("component", "drift_analyzer").Logger(),
	}
}

// Analyze processes a series of classification history records and identifies
// significant classification drifts. It groups changes by asset and detects
// the type of drift (escalation, de-escalation, PII addition/removal, or
// reclassification).
func (d *DriftAnalyzer) Analyze(history []model.ClassificationHistory) []model.ClassificationDrift {
	if len(history) == 0 {
		return nil
	}

	// Group history by asset.
	assetHistory := make(map[string][]model.ClassificationHistory)
	for _, h := range history {
		key := h.DataAssetID.String()
		assetHistory[key] = append(assetHistory[key], h)
	}

	var drifts []model.ClassificationDrift

	for _, records := range assetHistory {
		// Sort records by creation time ascending.
		sort.Slice(records, func(i, j int) bool {
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		})

		drift := d.analyzeAssetDrift(records)
		if drift != nil && len(drift.DriftEvents) > 0 {
			drifts = append(drifts, *drift)
		}
	}

	d.logger.Info().
		Int("history_records", len(history)).
		Int("assets_with_drift", len(drifts)).
		Msg("drift analysis complete")

	return drifts
}

// analyzeAssetDrift processes classification history for a single asset
// and produces drift events.
func (d *DriftAnalyzer) analyzeAssetDrift(records []model.ClassificationHistory) *model.ClassificationDrift {
	if len(records) < 1 {
		return nil
	}

	drift := &model.ClassificationDrift{
		AssetID:   records[0].DataAssetID,
		AssetName: "", // Will be populated by caller if needed.
	}

	for _, record := range records {
		// Skip initial classifications (no drift).
		if record.OldClassification == "" || record.ChangeType == model.ChangeTypeInitial {
			continue
		}

		changeType := determineChangeType(record)

		event := model.DriftEvent{
			OldClassification: record.OldClassification,
			NewClassification: record.NewClassification,
			ChangeType:        changeType,
			DetectedAt:        record.CreatedAt,
			Confidence:        record.Confidence,
		}

		drift.DriftEvents = append(drift.DriftEvents, event)
	}

	return drift
}

// determineChangeType classifies the nature of a classification change.
func determineChangeType(record model.ClassificationHistory) string {
	// Check for PII-related changes first.
	oldHasPII := len(record.OldPIITypes) > 0
	newHasPII := len(record.NewPIITypes) > 0

	if !oldHasPII && newHasPII {
		return string(model.ChangeTypePIIAdded)
	}
	if oldHasPII && !newHasPII {
		return string(model.ChangeTypePIIRemoved)
	}

	// Check for classification level changes.
	oldSeverity := classificationSeverity[strings.ToLower(record.OldClassification)]
	newSeverity := classificationSeverity[strings.ToLower(record.NewClassification)]

	if newSeverity > oldSeverity {
		return string(model.ChangeTypeEscalation)
	}
	if newSeverity < oldSeverity {
		return string(model.ChangeTypeDeescalation)
	}

	// Same severity level but different classification.
	return string(model.ChangeTypeReclassification)
}
