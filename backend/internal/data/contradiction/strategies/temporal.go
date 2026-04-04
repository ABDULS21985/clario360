package strategies

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/clario360/platform/internal/data/connector"
	cruntime "github.com/clario360/platform/internal/data/contradiction/runtime"
	"github.com/clario360/platform/internal/data/model"
)

type TemporalStrategy struct{}

func NewTemporalStrategy() interface{ Type() string; Detect(context.Context, cruntime.ModelPair, connector.Connector, connector.Connector) ([]cruntime.RawContradiction, error) } {
	return &TemporalStrategy{}
}

func (s *TemporalStrategy) Type() string { return "temporal" }

func (s *TemporalStrategy) Detect(ctx context.Context, pair cruntime.ModelPair, connA, connB connector.Connector) ([]cruntime.RawContradiction, error) {
	result := make([]cruntime.RawContradiction, 0)
	if pair.ModelA.SourceTable != nil {
		rows, err := connA.FetchData(ctx, *pair.ModelA.SourceTable, connector.FetchParams{BatchSize: 10000})
		if err != nil {
			return nil, err
		}
		samples := make([]map[string]interface{}, 0)
		for _, row := range rows.Rows {
			if updated, ok := parseTime(row["updated_at"]); ok && time.Since(updated) > 365*24*time.Hour {
				status := strings.ToLower(fmt.Sprint(row["status"]))
				if status != "closed" && status != "archived" {
					samples = append(samples, map[string]interface{}{"updated_at": updated, "status": status})
				}
			}
		}
		if len(samples) > 0 {
			result = append(result, cruntime.RawContradiction{
				Type:            model.ContradictionTypeTemporal,
				Title:           "Stale active records",
				Description:     "records appear stale but remain active.",
				Column:          "updated_at",
				AffectedRecords: len(samples),
				SampleRecords:   samples,
				SourceA:         model.ContradictionSource{SourceID: &pair.SourceA.ID, SourceName: pair.SourceA.Name, ModelID: &pair.ModelA.ID, ModelName: pair.ModelA.DisplayName},
				SourceB:         model.ContradictionSource{SourceID: &pair.SourceA.ID, SourceName: pair.SourceA.Name, ModelID: &pair.ModelA.ID, ModelName: pair.ModelA.DisplayName},
			})
		}
	}
	return result, nil
}
