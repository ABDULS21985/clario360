package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/clario360/platform/internal/workflow/model"
)

// writeJSON serializes v as JSON and writes it to the response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a structured JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
}

// parseBody decodes the JSON request body into dst.
func parseBody(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

// urlParam extracts a URL parameter by name from the chi route context.
func urlParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

// parsePagination extracts page and page_size from query parameters with defaults.
func parsePagination(r *http.Request) (page, pageSize int) {
	page = 1
	pageSize = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	for _, key := range []string{"page_size", "per_page"} {
		if ps := r.URL.Query().Get(key); ps != "" {
			if v, err := strconv.Atoi(ps); err == nil && v > 0 && v <= 100 {
				pageSize = v
				break
			}
		}
	}
	return
}

// handleServiceError maps domain errors to appropriate HTTP responses.
func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, model.ErrConflict):
		writeError(w, http.StatusConflict, "CONFLICT", err.Error())
	case errors.Is(err, model.ErrConcurrencyConfl):
		writeError(w, http.StatusConflict, "CONCURRENCY_CONFLICT", err.Error())
	case errors.Is(err, model.ErrTaskNotClaimable):
		writeError(w, http.StatusConflict, "TASK_NOT_CLAIMABLE", err.Error())
	case errors.Is(err, model.ErrTaskNotOwned):
		writeError(w, http.StatusForbidden, "TASK_NOT_OWNED", err.Error())
	default:
		msg := err.Error()
		if isValidationError(msg) {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
		} else {
			writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
		}
	}
}

// isValidationError heuristically detects validation-style error messages.
func isValidationError(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "validation") ||
		strings.Contains(lower, "required") ||
		strings.Contains(lower, "invalid") ||
		strings.Contains(lower, "must be")
}
