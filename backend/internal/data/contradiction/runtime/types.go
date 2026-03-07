package runtime

import "github.com/clario360/platform/internal/data/model"

type ModelPair struct {
	ModelA     *model.DataModel
	ModelB     *model.DataModel
	SourceA    *model.DataSource
	SourceB    *model.DataSource
	LinkColumn string
}

type RawContradiction struct {
	Type            model.ContradictionType
	Title           string
	Description     string
	Column          string
	EntityKey       string
	AffectedRecords int
	SampleRecords   []map[string]interface{}
	SourceA         model.ContradictionSource
	SourceB         model.ContradictionSource
	Metadata        map[string]any
	NumericDeltaPct float64
}

