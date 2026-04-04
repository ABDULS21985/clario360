package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

type DagsterMockConfig struct {
	Version      string
	Repositories []string
	Assets       []DagsterAsset
	Runs         []DagsterRun
}

type DagsterAsset struct {
	KeyPath       []string
	Description   string
	GraphName     string
	ComputeKind   string
	IsPartitioned bool
	PartitionName string
	Metadata      map[string]string
	Dependencies  [][]string
}

type DagsterRun struct {
	RunID        string
	PipelineName string
	Status       string
	User         string
	StartTime    time.Time
	EndTime      time.Time
	RunConfigYAML string
}

func DefaultDagsterMockConfig() DagsterMockConfig {
	return DagsterMockConfig{
		Version:      "1.6.0",
		Repositories: []string{"analytics"},
		Assets: []DagsterAsset{
			{
				KeyPath:     []string{"warehouse", "customers"},
				Description: "Curated customer asset",
				GraphName:   "nightly_ingest",
				ComputeKind: "sql",
				Metadata: map[string]string{
					"owner": "data-platform",
				},
				Dependencies: [][]string{{"raw", "customers"}},
			},
		},
		Runs: []DagsterRun{
			{
				RunID:         "run-1",
				PipelineName:  "nightly_ingest",
				Status:        "SUCCESS",
				User:          "data-engineer",
				StartTime:     time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				EndTime:       time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
				RunConfigYAML: "asset: warehouse.customers",
			},
		},
	}
}

func NewDagsterMockServer(cfg DagsterMockConfig) *httptest.Server {
	if cfg.Version == "" {
		cfg.Version = "1.6.0"
	}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		switch {
		case strings.Contains(payload.Query, "query { version }"):
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"version": cfg.Version}})
		case strings.Contains(payload.Query, "repositoriesOrError"):
			nodes := make([]map[string]any, 0, len(cfg.Repositories))
			for _, repo := range cfg.Repositories {
				nodes = append(nodes, map[string]any{"name": repo})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"repositoriesOrError": map[string]any{"nodes": nodes},
			}})
		case strings.Contains(payload.Query, "assetsOrError"):
			nodes := make([]map[string]any, 0, len(cfg.Assets))
			for _, asset := range cfg.Assets {
				metadataEntries := make([]map[string]any, 0, len(asset.Metadata))
				for label, text := range asset.Metadata {
					metadataEntries = append(metadataEntries, map[string]any{
						"label":      label,
						"__typename": "TextMetadataEntry",
						"text":       text,
					})
				}
				dependencies := make([]map[string]any, 0, len(asset.Dependencies))
				for _, dep := range asset.Dependencies {
					dependencies = append(dependencies, map[string]any{"path": dep})
				}
				node := map[string]any{
					"key":            map[string]any{"path": asset.KeyPath},
					"description":    asset.Description,
					"graphName":      asset.GraphName,
					"computeKind":    asset.ComputeKind,
					"isPartitioned":  asset.IsPartitioned,
					"metadataEntries": metadataEntries,
					"dependencyKeys": dependencies,
				}
				if asset.PartitionName != "" {
					node["partitionDefinition"] = map[string]any{"name": asset.PartitionName}
				}
				nodes = append(nodes, node)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"assetsOrError": map[string]any{"nodes": nodes},
			}})
		case strings.Contains(payload.Query, "runsOrError"):
			results := make([]map[string]any, 0, len(cfg.Runs))
			for _, run := range cfg.Runs {
				results = append(results, map[string]any{
					"runId":        run.RunID,
					"pipelineName": run.PipelineName,
					"status":       run.Status,
					"startTime":    float64(run.StartTime.Unix()),
					"endTime":      float64(run.EndTime.Unix()),
					"tags": []map[string]any{
						{"key": "dagster/user", "value": run.User},
					},
					"runConfigYaml": run.RunConfigYAML,
				})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"runsOrError": map[string]any{"results": results},
			}})
		default:
			http.Error(w, "unexpected query", http.StatusBadRequest)
		}
	}))
}
