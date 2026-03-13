package feeds

import "strings"

type IndustryBenchmark struct {
	Industry               string             `json:"industry"`
	TechniquePressure      map[string]float64 `json:"technique_pressure"`
	AssetTypePressure      map[string]float64 `json:"asset_type_pressure"`
	VulnerabilityClassRisk map[string]float64 `json:"vulnerability_class_risk"`
}

type IndustryBenchmarker struct{}

func NewIndustryBenchmarker() *IndustryBenchmarker {
	return &IndustryBenchmarker{}
}

func (b *IndustryBenchmarker) Build(industry string, feedSignals []ThreatFeedSignal) IndustryBenchmark {
	benchmark := IndustryBenchmark{
		Industry:               strings.TrimSpace(strings.ToLower(industry)),
		TechniquePressure:      map[string]float64{},
		AssetTypePressure:      map[string]float64{},
		VulnerabilityClassRisk: map[string]float64{},
	}
	for _, item := range feedSignals {
		weight := severityWeight(item.Severity)
		for _, technique := range item.TechniqueIDs {
			benchmark.TechniquePressure[technique] += weight
		}
		for _, target := range item.Targets {
			target = strings.TrimSpace(strings.ToLower(target))
			if target == "" {
				continue
			}
			benchmark.AssetTypePressure[target] += weight
		}
		if class, ok := item.Metadata["vulnerability_class"].(string); ok {
			benchmark.VulnerabilityClassRisk[strings.ToLower(strings.TrimSpace(class))] += weight
		}
	}
	return benchmark
}
