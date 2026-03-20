package handler

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	notifcfg "github.com/clario360/platform/internal/notification/config"
	"github.com/clario360/platform/internal/notification/repository"
	"github.com/clario360/platform/internal/notification/websocket"
)

// WebSocketHandler handles WebSocket upgrade and auth.
type WebSocketHandler struct {
	hub       *websocket.Hub
	jwtMgr    *auth.JWTManager
	notifRepo *repository.NotificationRepository
	cfg       *notifcfg.Config
	upgrader  gorillaWS.Upgrader
	logger    zerolog.Logger
}

// NewWebSocketHandler creates a new WebSocketHandler.
func NewWebSocketHandler(
	hub *websocket.Hub,
	jwtMgr *auth.JWTManager,
	notifRepo *repository.NotificationRepository,
	cfg *notifcfg.Config,
	logger zerolog.Logger,
) *WebSocketHandler {
	h := &WebSocketHandler{
		hub:       hub,
		jwtMgr:    jwtMgr,
		notifRepo: notifRepo,
		cfg:       cfg,
		logger:    logger.With().Str("component", "ws_handler").Logger(),
	}

	h.upgrader = gorillaWS.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if cfg.Environment == "development" {
				return true
			}
			return h.isAllowedOrigin(r)
		},
	}

	return h
}

// HandleWebSocket handles GET /ws/v1/notifications?token=<JWT>.
func (h *WebSocketHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	identity, err := h.authenticate(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	// Upgrade connection.
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Error().Err(err).Msg("ws upgrade failed")
		return
	}

	sessionID := identity.SessionID
	if sessionID == "" {
		sessionID = time.Now().Format("20060102150405")
	}

	clientCfg := websocket.ClientConfig{
		PingInterval:   time.Duration(h.cfg.WSPingIntervalSec) * time.Second,
		PongTimeout:    time.Duration(h.cfg.WSPongTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(h.cfg.WSWriteTimeoutSec) * time.Second,
		MaxMessageSize: h.cfg.WSMaxMessageSizeBytes,
	}

	client := websocket.NewClient(h.hub, conn, identity.UserID, identity.TenantID, sessionID, clientCfg, h.logger)

	h.hub.Register(client)

	// Send connection ack.
	ackMsg, _ := websocket.NewWSMessage(websocket.MsgTypeConnectionAck, websocket.ConnectionAckData{
		UserID:    identity.UserID,
		SessionID: sessionID,
	})
	client.Send(ackMsg)

	// Send current unread count.
	if count, err := h.notifRepo.UnreadCount(r.Context(), identity.TenantID, identity.UserID); err == nil {
		countMsg, _ := websocket.NewWSMessage(websocket.MsgTypeUnreadCount, websocket.UnreadCountData{Count: count})
		client.Send(countMsg)
	}

	// Start read/write pumps.
	go client.WritePump()
	go client.ReadPump()
}

type wsIdentity struct {
	UserID    string
	TenantID  string
	Email     string
	SessionID string
}

func (h *WebSocketHandler) authenticate(r *http.Request) (*wsIdentity, error) {
	if token := strings.TrimSpace(r.URL.Query().Get("token")); token != "" {
		claims, err := h.jwtMgr.ValidateAccessToken(token)
		if err != nil {
			return nil, fmt.Errorf("invalid token")
		}
		return &wsIdentity{
			UserID:    claims.UserID,
			TenantID:  claims.TenantID,
			Email:     claims.Email,
			SessionID: claims.ID,
		}, nil
	}

	userID := strings.TrimSpace(r.Header.Get("X-User-ID"))
	tenantID := strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	if userID == "" || tenantID == "" {
		return nil, fmt.Errorf("missing token")
	}

	return &wsIdentity{
		UserID:    userID,
		TenantID:  tenantID,
		Email:     strings.TrimSpace(r.Header.Get("X-User-Email")),
		SessionID: strings.TrimSpace(r.Header.Get("X-Request-ID")),
	}, nil
}

func (h *WebSocketHandler) isAllowedOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	for _, allowed := range h.cfg.WSAllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}
