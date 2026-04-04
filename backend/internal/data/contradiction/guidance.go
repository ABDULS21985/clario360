package contradiction

import (
	"fmt"
	"strings"

	"github.com/clario360/platform/internal/data/model"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
)

func GenerateGuidance(raw cruntime.RawContradiction, modelA, modelB *model.DataModel, sourceA, sourceB *model.DataSource) (string, *string) {
	authoritative := determineAuthoritativeSource(sourceA, sourceB)
	switch raw.Type {
	case model.ContradictionTypeLogical:
		return fmt.Sprintf(
			"The value of '%s' differs between %s and %s.\n\nRecommended actions:\n1. Determine which source is authoritative for this field.\n2. Correct the non-authoritative source.\n3. Consider establishing a master data management process.\n\nLikely authoritative source: %s.",
			raw.Column,
			sourceA.Name,
			sourceB.Name,
			authoritative,
		), stringPtr(authoritative)
	case model.ContradictionTypeSemantic:
		return fmt.Sprintf(
			"The data violates a semantic business rule in %s.\n\nRecommended actions:\n1. Verify the source record.\n2. Correct the source if needed.\n3. Add a preventative quality rule.",
			sourceA.Name,
		), stringPtr(authoritative)
	case model.ContradictionTypeAnalytical:
		return fmt.Sprintf(
			"The aggregated values for '%s' differ between %s and %s.\n\nRecommended actions:\n1. Check for missing or duplicate records.\n2. Verify that both sources cover the same time period.\n3. Reconcile the difference with record-level comparisons.",
			raw.Column,
			sourceA.Name,
			sourceB.Name,
		), stringPtr(authoritative)
	default:
		return fmt.Sprintf(
			"The record appears stale or inconsistent between %s and %s.\n\nRecommended actions:\n1. Confirm recent source updates.\n2. Re-run synchronization.\n3. Review whether the stale state is intentional.",
			sourceA.Name,
			sourceB.Name,
		), stringPtr(authoritative)
	}
}

func determineAuthoritativeSource(sourceA, sourceB *model.DataSource) string {
	switch {
	case sourceA.LastSyncedAt != nil && sourceB.LastSyncedAt != nil:
		if sourceA.LastSyncedAt.After(*sourceB.LastSyncedAt) {
			return sourceA.Name
		}
		if sourceB.LastSyncedAt.After(*sourceA.LastSyncedAt) {
			return sourceB.Name
		}
	}
	if sourceA.TotalRowCount != nil && sourceB.TotalRowCount != nil {
		if *sourceA.TotalRowCount > *sourceB.TotalRowCount {
			return sourceA.Name
		}
		if *sourceB.TotalRowCount > *sourceA.TotalRowCount {
			return sourceB.Name
		}
	}
	return strings.TrimSpace(sourceA.Name)
}

func stringPtr(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	copyValue := value
	return &copyValue
}
