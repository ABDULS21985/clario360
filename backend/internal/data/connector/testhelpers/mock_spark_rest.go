package testhelpers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"
)

type SparkRESTFixture struct {
	Applications []SparkApplicationFixture
}

type SparkApplicationFixture struct {
	ID         string
	Name       string
	User       string
	StartTime  time.Time
	EndTime    time.Time
	Completed  bool
	Jobs       []SparkJobFixture
	Stages     []SparkStageFixture
	Executors  []SparkExecutorFixture
}

type SparkJobFixture struct {
	JobID             int
	Name              string
	Status            string
	NumTasks          int
	NumCompletedTasks int
}

type SparkStageFixture struct {
	InputBytes   int64
	InputRecords int64
}

type SparkExecutorFixture struct {
	ID            string
	HostPort      string
	TotalTasks    int
	TotalDuration int64
}

func DefaultSparkRESTFixture() SparkRESTFixture {
	return SparkRESTFixture{
		Applications: []SparkApplicationFixture{
			{
				ID:        "app-1",
				Name:      "daily-sales-job",
				User:      "analyst",
				StartTime: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
				EndTime:   time.Date(2026, 3, 8, 10, 5, 0, 0, time.UTC),
				Completed: true,
				Jobs: []SparkJobFixture{
					{JobID: 7, Name: "daily-sales-job", Status: "SUCCEEDED", NumTasks: 4, NumCompletedTasks: 4},
				},
				Stages: []SparkStageFixture{
					{InputBytes: 4096, InputRecords: 1024},
				},
				Executors: []SparkExecutorFixture{
					{ID: "1", HostPort: "executor:1234", TotalTasks: 4, TotalDuration: 3000},
				},
			},
		},
	}
}

func NewSparkRESTServer(fixture SparkRESTFixture) *httptest.Server {
	appIndex := make(map[string]SparkApplicationFixture, len(fixture.Applications))
	for _, app := range fixture.Applications {
		appIndex[app.ID] = app
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
		payload := make([]map[string]any, 0, len(fixture.Applications))
		for _, app := range fixture.Applications {
			payload = append(payload, map[string]any{
				"id":        app.ID,
				"name":      app.Name,
				"sparkUser": app.User,
				"completed": app.Completed,
				"attempts": []map[string]any{
					{
						"startTime": app.StartTime.Format("2006-01-02T15:04:05.000GMT"),
						"endTime":   app.EndTime.Format("2006-01-02T15:04:05.000GMT"),
						"completed": app.Completed,
					},
				},
			})
		}
		_ = json.NewEncoder(w).Encode(payload)
	})
	mux.HandleFunc("/api/v1/applications/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/api/v1/applications/"):]
		parts := splitNonEmpty(path)
		if len(parts) == 0 {
			http.NotFound(w, r)
			return
		}
		app, ok := appIndex[parts[0]]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if len(parts) == 1 {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":        app.ID,
				"name":      app.Name,
				"sparkUser": app.User,
				"completed": app.Completed,
				"attempts": []map[string]any{
					{
						"startTime": app.StartTime.Format("2006-01-02T15:04:05.000GMT"),
						"endTime":   app.EndTime.Format("2006-01-02T15:04:05.000GMT"),
						"completed": app.Completed,
					},
				},
			})
			return
		}
		switch parts[1] {
		case "jobs":
			payload := make([]map[string]any, 0, len(app.Jobs))
			for _, job := range app.Jobs {
				payload = append(payload, map[string]any{
					"jobId":             job.JobID,
					"name":              job.Name,
					"status":            job.Status,
					"numTasks":          job.NumTasks,
					"numCompletedTasks": job.NumCompletedTasks,
				})
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "stages":
			payload := make([]map[string]any, 0, len(app.Stages))
			for _, stage := range app.Stages {
				payload = append(payload, map[string]any{
					"inputBytes":   stage.InputBytes,
					"inputRecords": stage.InputRecords,
				})
			}
			_ = json.NewEncoder(w).Encode(payload)
		case "executors":
			payload := make([]map[string]any, 0, len(app.Executors))
			for _, executor := range app.Executors {
				payload = append(payload, map[string]any{
					"id":            executor.ID,
					"hostPort":      executor.HostPort,
					"totalTasks":    executor.TotalTasks,
					"totalDuration": executor.TotalDuration,
				})
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	})

	return httptest.NewServer(mux)
}

func splitNonEmpty(path string) []string {
	parts := make([]string, 0)
	current := ""
	for _, r := range path {
		if r == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
			continue
		}
		current += string(r)
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
