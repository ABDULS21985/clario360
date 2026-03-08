package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/events"
	intsvc "github.com/clario360/platform/internal/integration/service"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	resp := map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
	if reqID := r.Context().Value("request_id"); reqID != nil {
		resp["error"].(map[string]any)["request_id"] = reqID
	}
	writeJSON(w, status, resp)
}

func actorFromRequest(r *http.Request) *intsvc.AuditActor {
	user := auth.UserFromContext(r.Context())
	if user == nil {
		return nil
	}
	return &intsvc.AuditActor{
		UserID:    user.ID,
		UserEmail: user.Email,
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}
}

func requireAuth(r *http.Request) (*auth.ContextUser, string) {
	user := auth.UserFromContext(r.Context())
	tenantID := auth.TenantFromContext(r.Context())
	return user, tenantID
}

func readBodyAndRestore(r *http.Request, limit int64) ([]byte, error) {
	if limit <= 0 {
		limit = 1 << 20
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, limit))
	if err != nil {
		return nil, err
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}

func publishAuditEvent(ctx context.Context, producer *events.Producer, tenantID, eventType string, actor *intsvc.AuditActor, data map[string]any) {
	if producer == nil || tenantID == "" {
		return
	}
	event, err := events.NewEvent(eventType, "notification-service", tenantID, data)
	if err != nil {
		return
	}
	if actor != nil {
		event.UserID = actor.UserID
		if event.Metadata == nil {
			event.Metadata = map[string]string{}
		}
		if actor.UserEmail != "" {
			event.Metadata["user_email"] = actor.UserEmail
		}
		if actor.IPAddress != "" {
			event.Metadata["ip_address"] = actor.IPAddress
		}
		if actor.UserAgent != "" {
			event.Metadata["user_agent"] = actor.UserAgent
		}
	}
	_ = producer.Publish(ctx, events.Topics.AuditEvents, event)
}

func pointerTime(t time.Time) *time.Time {
	return &t
}
