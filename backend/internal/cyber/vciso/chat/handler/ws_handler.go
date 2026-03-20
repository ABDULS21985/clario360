package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/google/uuid"
	gorillaWS "github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

type WebSocketHandler struct {
	engine   ChatEngine
	jwtMgr   *auth.JWTManager
	upgrader gorillaWS.Upgrader
	logger   zerolog.Logger
}

type wsInboundMessage struct {
	Type           string     `json:"type"`
	ConversationID *uuid.UUID `json:"conversation_id,omitempty"`
	Content        string     `json:"content"`
	PreferEngine   string     `json:"prefer_engine,omitempty"`
}

func NewWebSocketHandler(engine ChatEngine, jwtMgr *auth.JWTManager, logger zerolog.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		engine: engine,
		jwtMgr: jwtMgr,
		logger: logger.With().Str("component", "vciso_ws_handler").Logger(),
		upgrader: gorillaWS.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	claims, err := h.authenticate(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("websocket upgrade failed")
		return
	}
	defer conn.Close()

	ctx := auth.WithClaims(r.Context(), claims)
	ctx = auth.WithTenantID(ctx, claims.TenantID)
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
		Roles:    claims.Roles,
	})

	tenantID, err := uuid.Parse(claims.TenantID)
	if err != nil {
		_ = conn.WriteJSON(map[string]any{"type": "error", "code": "INVALID_TOKEN", "message": "invalid tenant"})
		return
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		_ = conn.WriteJSON(map[string]any{"type": "error", "code": "INVALID_TOKEN", "message": "invalid user"})
		return
	}

	writer := &wsWriter{conn: conn}
	_ = writer.Write(map[string]any{"type": "system", "content": "Connected to vCISO. How can I help?"})
	if suggestions, err := h.engine.GetSuggestions(ctx, nil, tenantID, userID); err == nil {
		_ = writer.Write(map[string]any{"type": "suggestions", "data": suggestions})
	}

	for {
		var inbound wsInboundMessage
		if err := conn.ReadJSON(&inbound); err != nil {
			return
		}
		if inbound.Type != "message" {
			_ = writer.Write(map[string]any{"type": "error", "code": "INVALID_MESSAGE", "message": "unsupported websocket message type"})
			continue
		}
		_ = writer.Write(map[string]any{"type": "status", "status": "classifying"})
		peek := h.engine.Peek(inbound.Content)
		if peek.ToolName != "" {
			_ = writer.Write(map[string]any{"type": "status", "status": "executing", "tool": peek.ToolName})
		}
		resp, err := h.engine.ProcessMessage(withRequestAuth(context.Background(), claims), inbound.ConversationID, tenantID, userID, inbound.Content, inbound.PreferEngine)
		if err != nil {
			_ = writer.Write(map[string]any{"type": "error", "code": "PROCESSING_ERROR", "message": err.Error()})
			continue
		}
		_ = writer.Write(map[string]any{
			"type":            "response",
			"conversation_id": resp.ConversationID,
			"message_id":      resp.MessageID,
			"text":            resp.Response.Text,
			"data":            resp.Response.Data,
			"data_type":       resp.Response.DataType,
			"actions":         resp.Response.Actions,
			"entities":        resp.Response.Entities,
			"intent":          resp.Intent,
			"confidence":      resp.Confidence,
			"engine":          resp.Engine,
			"meta":            resp.Meta,
		})
		suggestions, err := h.engine.GetSuggestions(withRequestAuth(context.Background(), claims), &resp.ConversationID, tenantID, userID)
		if err == nil {
			_ = writer.Write(map[string]any{"type": "suggestions", "data": suggestions})
		}
	}
}

func (h *WebSocketHandler) authenticate(r *http.Request) (*auth.Claims, error) {
	if h.jwtMgr == nil {
		return nil, context.Canceled
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
			token = strings.TrimSpace(authHeader[7:])
		}
	}
	if token != "" {
		return h.jwtMgr.ValidateAccessToken(token)
	}

	// Fallback: accept gateway-injected headers — the gateway already validated
	// the JWT and strips the token before dialing the backend WebSocket.
	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if userID == "" || tenantID == "" {
		return nil, context.Canceled
	}
	var roles []string
	if rolesHeader := strings.TrimSpace(r.Header.Get("X-User-Roles")); rolesHeader != "" {
		roles = strings.Split(rolesHeader, ",")
	}
	return &auth.Claims{
		UserID:   userID,
		TenantID: tenantID,
		Email:    strings.TrimSpace(r.Header.Get("X-User-Email")),
		Roles:    roles,
	}, nil
}

func withRequestAuth(ctx context.Context, claims *auth.Claims) context.Context {
	ctx = auth.WithClaims(ctx, claims)
	ctx = auth.WithTenantID(ctx, claims.TenantID)
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       claims.UserID,
		TenantID: claims.TenantID,
		Email:    claims.Email,
		Roles:    claims.Roles,
	})
	return ctx
}

type wsWriter struct {
	conn *gorillaWS.Conn
	mu   sync.Mutex
}

func (w *wsWriter) Write(payload any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(payload)
}

func (m wsInboundMessage) String() string {
	payload, _ := json.Marshal(m)
	return string(payload)
}
