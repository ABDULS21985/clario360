package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rs/zerolog"
)

type WebhookHandler struct {
	logger zerolog.Logger
}

func NewWebhookHandler(logger zerolog.Logger) *WebhookHandler {
	return &WebhookHandler{
		logger: logger.With().Str("component", "integration_webhook_handler").Logger(),
	}
}

func (h *WebhookHandler) TestReceiver(w http.ResponseWriter, r *http.Request) {
	body, err := readBodyAndRestore(r, 1<<20)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "failed to read webhook payload")
		return
	}

	headers := make(map[string]string, len(r.Header))
	for key, values := range r.Header {
		headers[key] = strings.Join(values, ", ")
	}

	var decoded any
	if len(body) > 0 {
		if err := json.Unmarshal(body, &decoded); err != nil {
			decoded = string(body)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"method":  r.Method,
			"headers": headers,
			"body":    decoded,
		},
	})
}

