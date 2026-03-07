package darkdata

import (
	"sort"
	"strings"

	"github.com/clario360/platform/internal/data/discovery"
	"github.com/clario360/platform/internal/data/model"
)

type DarkDataClassifier struct{}

func NewClassifier() *DarkDataClassifier {
	return &DarkDataClassifier{}
}

func (c *DarkDataClassifier) Classify(name string, columns []string) (bool, []string, *model.DataClassification) {
	discovered := make([]model.DiscoveredColumn, 0, len(columns))
	for _, column := range columns {
		discovered = append(discovered, model.DiscoveredColumn{
			Name: column,
		})
	}
	if len(discovered) == 0 && strings.TrimSpace(name) != "" {
		discovered = append(discovered, model.DiscoveredColumn{Name: name})
	}
	discovered = discovery.DetectPII(discovered)

	piiTypes := make([]string, 0)
	classification := model.DataClassificationPublic
	containsPII := false
	seen := make(map[string]struct{})
	for _, column := range discovered {
		if column.InferredPII && column.InferredPIIType != "" {
			containsPII = true
			if _, ok := seen[column.InferredPIIType]; !ok {
				seen[column.InferredPIIType] = struct{}{}
				piiTypes = append(piiTypes, column.InferredPIIType)
			}
		}
		if rankClassification(column.InferredClass) > rankClassification(classification) {
			classification = column.InferredClass
		}
	}
	sort.Strings(piiTypes)
	return containsPII, piiTypes, &classification
}

func rankClassification(value model.DataClassification) int {
	switch value {
	case model.DataClassificationRestricted:
		return 4
	case model.DataClassificationConfidential:
		return 3
	case model.DataClassificationInternal:
		return 2
	default:
		return 1
	}
}
