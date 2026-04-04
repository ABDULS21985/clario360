package models

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	predictmodel "github.com/clario360/platform/internal/cyber/vciso/predict/model"
)

type CampaignAlertSample struct {
	AlertID      uuid.UUID `json:"alert_id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Timestamp    time.Time `json:"timestamp"`
	Embedding    []float64 `json:"embedding"`
	IOCs         []string  `json:"iocs,omitempty"`
	Techniques   []string  `json:"techniques,omitempty"`
	TargetAssets []string  `json:"target_assets,omitempty"`
}

type CampaignDetector struct {
	ModelVersion   string             `json:"model_version"`
	Epsilon        float64            `json:"epsilon"`
	MinPoints      int                `json:"min_points"`
	FeatureWeights map[string]float64 `json:"feature_weights"`
}

func NewCampaignDetector(version string) *CampaignDetector {
	if version == "" {
		version = "campaign-detector-v1"
	}
	return &CampaignDetector{
		ModelVersion: version,
		Epsilon:      0.55,
		MinPoints:    2,
		FeatureWeights: map[string]float64{
			"semantic_similarity": 0.35,
			"time_proximity":      0.20,
			"ioc_overlap":         0.20,
			"technique_overlap":   0.15,
			"target_overlap":      0.10,
		},
	}
}

func (m *CampaignDetector) Train(samples []CampaignAlertSample) error {
	if len(samples) == 0 {
		return fmt.Errorf("at least 1 alert sample is required")
	}
	return nil
}

func (m *CampaignDetector) Detect(samples []CampaignAlertSample) []predictmodel.CampaignCluster {
	if len(samples) == 0 {
		return nil
	}
	clusters := dbscan(samples, m.Epsilon, m.MinPoints, m.distance)
	out := make([]predictmodel.CampaignCluster, 0, len(clusters))
	for idx, cluster := range clusters {
		if len(cluster) < m.MinPoints {
			continue
		}
		sort.SliceStable(cluster, func(i, j int) bool { return cluster[i].Timestamp.Before(cluster[j].Timestamp) })
		featureTotals := map[string]float64{
			"semantic_similarity": 0,
			"time_proximity":      0,
			"ioc_overlap":         0,
			"technique_overlap":   0,
			"target_overlap":      0,
		}
		alertIDs := make([]string, 0, len(cluster))
		titles := make([]string, 0, len(cluster))
		techniques := map[string]struct{}{}
		iocs := map[string]struct{}{}
		for i := 0; i < len(cluster); i++ {
			alertIDs = append(alertIDs, cluster[i].AlertID.String())
			titles = append(titles, cluster[i].Title)
			for _, technique := range cluster[i].Techniques {
				techniques[technique] = struct{}{}
			}
			for _, ioc := range cluster[i].IOCs {
				iocs[ioc] = struct{}{}
			}
			for j := i + 1; j < len(cluster); j++ {
				metrics := m.pairMetrics(cluster[i], cluster[j])
				for key, value := range metrics {
					featureTotals[key] += value
				}
			}
		}
		divisor := float64(max(1, len(cluster)*(len(cluster)-1)/2))
		confidence := 0.0
		for key, total := range featureTotals {
			featureTotals[key] = total / divisor
			confidence += featureTotals[key] * m.FeatureWeights[key]
		}
		out = append(out, predictmodel.CampaignCluster{
			ClusterID:       fmt.Sprintf("campaign-%d", idx+1),
			AlertIDs:        alertIDs,
			AlertTitles:     titles,
			StartAt:         cluster[0].Timestamp,
			EndAt:           cluster[len(cluster)-1].Timestamp,
			Stage:           campaignStage(cluster),
			MITRETechniques: sortedKeys(techniques),
			SharedIOCs:      sortedKeys(iocs),
			ConfidenceInterval: predictmodel.ConfidenceInterval{
				P10: clamp(confidence-0.10, 0, 1),
				P50: clamp(confidence, 0, 1),
				P90: clamp(confidence+0.10, 0, 1),
			},
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ConfidenceInterval.P50 > out[j].ConfidenceInterval.P50
	})
	return out
}

func (m *CampaignDetector) Serialize() ([]byte, error) {
	return json.Marshal(m)
}

func (m *CampaignDetector) Deserialize(payload []byte) error {
	return json.Unmarshal(payload, m)
}

func (m *CampaignDetector) distance(a, b CampaignAlertSample) float64 {
	metrics := m.pairMetrics(a, b)
	score := 0.0
	for key, value := range metrics {
		score += value * m.FeatureWeights[key]
	}
	return 1 - clamp(score, 0, 1)
}

func (m *CampaignDetector) pairMetrics(a, b CampaignAlertSample) map[string]float64 {
	timeDeltaHours := a.Timestamp.Sub(b.Timestamp).Hours()
	if timeDeltaHours < 0 {
		timeDeltaHours = -timeDeltaHours
	}
	return map[string]float64{
		"semantic_similarity": cosineSimilarity(a.Embedding, b.Embedding),
		"time_proximity":      clamp(1-(timeDeltaHours/72), 0, 1),
		"ioc_overlap":         overlapScore(a.IOCs, b.IOCs),
		"technique_overlap":   overlapScore(a.Techniques, b.Techniques),
		"target_overlap":      overlapScore(a.TargetAssets, b.TargetAssets),
	}
}

func dbscan(samples []CampaignAlertSample, epsilon float64, minPoints int, distance func(CampaignAlertSample, CampaignAlertSample) float64) [][]CampaignAlertSample {
	visited := make([]bool, len(samples))
	assigned := make([]bool, len(samples))
	clusters := make([][]CampaignAlertSample, 0)
	for idx := range samples {
		if visited[idx] {
			continue
		}
		visited[idx] = true
		neighbors := regionQuery(samples, idx, epsilon, distance)
		if len(neighbors) < minPoints {
			continue
		}
		cluster := make([]CampaignAlertSample, 0, len(neighbors)+1)
		cluster = append(cluster, samples[idx])
		assigned[idx] = true
		queue := append([]int(nil), neighbors...)
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			if !visited[current] {
				visited[current] = true
				currentNeighbors := regionQuery(samples, current, epsilon, distance)
				if len(currentNeighbors) >= minPoints {
					queue = append(queue, currentNeighbors...)
				}
			}
			if assigned[current] {
				continue
			}
			assigned[current] = true
			cluster = append(cluster, samples[current])
		}
		clusters = append(clusters, cluster)
	}
	return clusters
}

func regionQuery(samples []CampaignAlertSample, index int, epsilon float64, distance func(CampaignAlertSample, CampaignAlertSample) float64) []int {
	out := make([]int, 0, len(samples))
	for idx := range samples {
		if idx == index {
			continue
		}
		if distance(samples[index], samples[idx]) <= epsilon {
			out = append(out, idx)
		}
	}
	return out
}

func overlapScore(a, b []string) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	left := map[string]struct{}{}
	for _, item := range a {
		left[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	right := map[string]struct{}{}
	for _, item := range b {
		right[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}
	intersection := 0.0
	union := float64(len(left))
	for item := range right {
		if _, ok := left[item]; ok {
			intersection++
		} else {
			union++
		}
	}
	if union == 0 {
		return 0
	}
	return intersection / union
}

func campaignStage(cluster []CampaignAlertSample) string {
	if len(cluster) <= 2 {
		return "reconnaissance"
	}
	if len(cluster) <= 4 {
		return "active_attack"
	}
	return "expanded_campaign"
}

func sortedKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		out = append(out, key)
	}
	sort.Strings(out)
	return out
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
