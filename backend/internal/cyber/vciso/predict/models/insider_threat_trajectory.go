package models

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"
)

type InsiderThreatSample struct {
	EntityID         string    `json:"entity_id"`
	EntityName       string    `json:"entity_name"`
	Timestamp        time.Time `json:"timestamp"`
	RiskScore        float64   `json:"risk_score"`
	LoginAnomalies   float64   `json:"login_anomalies"`
	DataAccessTrend  float64   `json:"data_access_trend"`
	AfterHoursTrend  float64   `json:"after_hours_trend"`
	PolicyViolations float64   `json:"policy_violations"`
	HREventScore     float64   `json:"hr_event_score"`
	PeerDeviation    float64   `json:"peer_deviation"`
}

type InsiderThreatTrajectoryModel struct {
	ModelVersion     string             `json:"model_version"`
	InputWeights     map[string]float64 `json:"input_weights"`
	ForgetWeights    map[string]float64 `json:"forget_weights"`
	OutputWeights    map[string]float64 `json:"output_weights"`
	CandidateWeights map[string]float64 `json:"candidate_weights"`
	BiasInput        float64            `json:"bias_input"`
	BiasForget       float64            `json:"bias_forget"`
	BiasOutput       float64            `json:"bias_output"`
	BiasCandidate    float64            `json:"bias_candidate"`
	Residuals        []float64          `json:"residuals"`
}

func NewInsiderThreatTrajectoryModel(version string) *InsiderThreatTrajectoryModel {
	if version == "" {
		version = "insider-trajectory-v1"
	}
	return &InsiderThreatTrajectoryModel{
		ModelVersion: version,
		InputWeights: map[string]float64{
			"risk_score":        0.02,
			"login_anomalies":   0.20,
			"data_access_trend": 0.15,
			"after_hours_trend": 0.10,
			"policy_violations": 0.18,
			"hr_event_score":    0.25,
			"peer_deviation":    0.15,
		},
		ForgetWeights: map[string]float64{
			"risk_score":        -0.01,
			"login_anomalies":   0.10,
			"data_access_trend": 0.08,
			"after_hours_trend": 0.05,
			"policy_violations": 0.10,
			"hr_event_score":    0.12,
			"peer_deviation":    0.08,
		},
		OutputWeights: map[string]float64{
			"risk_score":        0.02,
			"login_anomalies":   0.15,
			"data_access_trend": 0.12,
			"after_hours_trend": 0.10,
			"policy_violations": 0.12,
			"hr_event_score":    0.18,
			"peer_deviation":    0.12,
		},
		CandidateWeights: map[string]float64{
			"risk_score":        0.03,
			"login_anomalies":   0.25,
			"data_access_trend": 0.18,
			"after_hours_trend": 0.15,
			"policy_violations": 0.20,
			"hr_event_score":    0.30,
			"peer_deviation":    0.16,
		},
		BiasInput:     -0.10,
		BiasForget:    0.15,
		BiasOutput:    -0.05,
		BiasCandidate: 0.0,
	}
}

func (m *InsiderThreatTrajectoryModel) Train(sequences map[string][]InsiderThreatSample) error {
	if len(sequences) == 0 {
		return fmt.Errorf("at least 1 insider trajectory is required")
	}
	m.Residuals = m.Residuals[:0]
	for _, series := range sequences {
		if len(series) < 3 {
			continue
		}
		sort.SliceStable(series, func(i, j int) bool { return series[i].Timestamp.Before(series[j].Timestamp) })
		for idx := 2; idx < len(series); idx++ {
			predicted, _, _ := m.Predict(series[:idx], 1, 80)
			m.Residuals = append(m.Residuals, series[idx].RiskScore-predicted)
		}
	}
	if len(m.Residuals) == 0 {
		m.Residuals = []float64{5}
	}
	return nil
}

func (m *InsiderThreatTrajectoryModel) Predict(sequence []InsiderThreatSample, horizonDays int, threshold float64) (float64, bool, *int) {
	if len(sequence) == 0 {
		return 0, false, nil
	}
	if horizonDays <= 0 {
		horizonDays = 30
	}
	if threshold <= 0 {
		threshold = 80
	}
	sort.SliceStable(sequence, func(i, j int) bool { return sequence[i].Timestamp.Before(sequence[j].Timestamp) })
	cell := 0.0
	hidden := 0.0
	riskSeries := make([]float64, 0, len(sequence))
	for _, sample := range sequence {
		gates := m.gates(sample)
		cell = gates.forget*cell + gates.input*gates.candidate
		hidden = gates.output * math.Tanh(cell)
		riskSeries = append(riskSeries, sample.RiskScore)
	}
	last := sequence[len(sequence)-1]
	projected := last.RiskScore
	seriesSlope := slope(riskSeries)
	accelerating := seriesSlope > 1.0
	var daysToThreshold *int
	for day := 1; day <= horizonDays; day++ {
		synthetic := last
		synthetic.RiskScore = projected
		if accelerating {
			synthetic.LoginAnomalies += 0.05
			synthetic.DataAccessTrend += 0.03
			synthetic.AfterHoursTrend += 0.02
			synthetic.PeerDeviation += 0.02
		}
		gates := m.gates(synthetic)
		cell = gates.forget*cell + gates.input*gates.candidate
		hidden = gates.output * math.Tanh(cell)
		projected = clamp(projected+hidden*12+seriesSlope, 0, 100)
		if daysToThreshold == nil && projected >= threshold {
			value := day
			daysToThreshold = &value
		}
	}
	return projected, accelerating, daysToThreshold
}

func (m *InsiderThreatTrajectoryModel) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m *InsiderThreatTrajectoryModel) Deserialize(payload []byte) error {
	return json.Unmarshal(payload, m)
}

type insiderGates struct {
	input     float64
	forget    float64
	output    float64
	candidate float64
}

func (m *InsiderThreatTrajectoryModel) gates(sample InsiderThreatSample) insiderGates {
	input := logistic(m.BiasInput + weightedInsider(sample, m.InputWeights))
	forget := logistic(m.BiasForget + weightedInsider(sample, m.ForgetWeights))
	output := logistic(m.BiasOutput + weightedInsider(sample, m.OutputWeights))
	candidate := math.Tanh(m.BiasCandidate + weightedInsider(sample, m.CandidateWeights))
	return insiderGates{
		input:     input,
		forget:    forget,
		output:    output,
		candidate: candidate,
	}
}

func weightedInsider(sample InsiderThreatSample, weights map[string]float64) float64 {
	total := 0.0
	total += sample.RiskScore * weights["risk_score"]
	total += sample.LoginAnomalies * weights["login_anomalies"]
	total += sample.DataAccessTrend * weights["data_access_trend"]
	total += sample.AfterHoursTrend * weights["after_hours_trend"]
	total += sample.PolicyViolations * weights["policy_violations"]
	total += sample.HREventScore * weights["hr_event_score"]
	total += sample.PeerDeviation * weights["peer_deviation"]
	return total
}
