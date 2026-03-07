package health

import (
	"encoding/json"
	"net/http"
	"time"
)

type statusResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// Register mounts operational health endpoints for the cyber service.
func Register(mux interface {
	HandleFunc(pattern string, handler http.HandlerFunc)
}) {
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, http.StatusOK, "ok")
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		writeStatus(w, http.StatusOK, "ready")
	})
}

func writeStatus(w http.ResponseWriter, status int, value string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(statusResponse{
		Status:    value,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
