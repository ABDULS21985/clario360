package correlator

import (
	"sort"
	"time"

	"github.com/clario360/platform/internal/cyber/ueba/model"
)

func filterWithinWindow(signals []model.AnomalySignal, since time.Time) []model.AnomalySignal {
	filtered := make([]model.AnomalySignal, 0, len(signals))
	for _, signal := range signals {
		if signal.EventTimestamp.IsZero() || signal.EventTimestamp.Before(since) {
			continue
		}
		filtered = append(filtered, signal)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		return filtered[i].EventTimestamp.Before(filtered[j].EventTimestamp)
	})
	return filtered
}
