package cti

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	gorillaWS "github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

// WSHub manages WebSocket connections grouped by tenant for real-time CTI event broadcasting.
type WSHub struct {
	clients  map[string]map[*gorillaWS.Conn]bool
	mu       sync.RWMutex
	logger   zerolog.Logger
	upgrader gorillaWS.Upgrader
}

func NewWSHub(logger zerolog.Logger) *WSHub {
	return &WSHub{
		clients: make(map[string]map[*gorillaWS.Conn]bool),
		logger:  logger.With().Str("component", "cti-ws-hub").Logger(),
		upgrader: gorillaWS.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     func(r *http.Request) bool { return true },
		},
	}
}

// HandleWebSocket upgrades an HTTP connection to WebSocket for a tenant.
// It reads the tenant from context (when behind auth middleware) or from
// gateway-injected X-Tenant-ID header (when proxied via the API gateway).
func (h *WSHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	tenantStr := auth.TenantFromContext(r.Context())
	if tenantStr == "" {
		// Fallback: gateway-injected header (gateway already validated JWT).
		tenantStr = strings.TrimSpace(r.Header.Get("X-Tenant-ID"))
	}
	if tenantStr == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Warn().Err(err).Msg("websocket upgrade failed")
		return
	}

	h.register(tenantStr, conn)
	defer h.unregister(tenantStr, conn)

	h.logger.Info().Str("tenant_id", tenantStr).Msg("CTI WebSocket client connected")

	// Read loop — keep connection alive, handle client pings
	conn.SetReadDeadline(time.Now().Add(120 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	h.logger.Info().Str("tenant_id", tenantStr).Msg("CTI WebSocket client disconnected")
}

// Broadcast sends a typed message to all connected clients of the given tenant.
func (h *WSHub) Broadcast(tenantID string, eventType string, data json.RawMessage) {
	h.mu.RLock()
	clients, ok := h.clients[tenantID]
	if !ok || len(clients) == 0 {
		h.mu.RUnlock()
		return
	}
	// Copy client set under read lock
	conns := make([]*gorillaWS.Conn, 0, len(clients))
	for c := range clients {
		conns = append(conns, c)
	}
	h.mu.RUnlock()

	msg, _ := json.Marshal(map[string]interface{}{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})

	for _, conn := range conns {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := conn.WriteMessage(gorillaWS.TextMessage, msg); err != nil {
			h.logger.Debug().Err(err).Msg("ws write failed, removing client")
			h.unregister(tenantID, conn)
			conn.Close()
		}
	}
}

// ClientCount returns the number of connected clients for a tenant.
func (h *WSHub) ClientCount(tenantID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[tenantID])
}

func (h *WSHub) register(tenantID string, conn *gorillaWS.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[tenantID] == nil {
		h.clients[tenantID] = make(map[*gorillaWS.Conn]bool)
	}
	h.clients[tenantID][conn] = true
}

func (h *WSHub) unregister(tenantID string, conn *gorillaWS.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if clients, ok := h.clients[tenantID]; ok {
		delete(clients, conn)
		if len(clients) == 0 {
			delete(h.clients, tenantID)
		}
	}
}
