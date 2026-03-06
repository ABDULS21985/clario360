package websocket

import (
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
)

const (
	sendBufferSize = 256
)

// ClientConfig holds configurable timeouts for client connections.
type ClientConfig struct {
	PingInterval    time.Duration
	PongTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxMessageSize  int64
}

// Client represents a single WebSocket connection.
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    string
	tenantID  string
	sessionID string
	createdAt time.Time
	closed    atomic.Bool
	config    ClientConfig
	logger    zerolog.Logger
}

// NewClient creates a new WebSocket client.
func NewClient(hub *Hub, conn *websocket.Conn, userID, tenantID, sessionID string, cfg ClientConfig, logger zerolog.Logger) *Client {
	return &Client{
		hub:       hub,
		conn:      conn,
		send:      make(chan []byte, sendBufferSize),
		userID:    userID,
		tenantID:  tenantID,
		sessionID: sessionID,
		createdAt: time.Now(),
		config:    cfg,
		logger:    logger.With().Str("user_id", userID).Str("session_id", sessionID).Logger(),
	}
}

// ReadPump reads from the WebSocket connection. Must run as a goroutine.
// At most one goroutine should call ReadPump for a connection.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.Close()
	}()

	c.conn.SetReadLimit(c.config.MaxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(c.config.PongTimeout))
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				c.logger.Debug().Err(err).Msg("ws read error")
			}
			return
		}
		// Currently clients do not send meaningful messages.
	}
}

// WritePump writes messages to the WebSocket connection. Must run as a goroutine.
// At most one goroutine should call WritePump for a connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				// Channel closed.
				_ = c.conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
					time.Now().Add(c.config.WriteTimeout),
				)
				return
			}

			_ = c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				c.logger.Debug().Err(err).Msg("ws write error")
				return
			}

			// Drain queued messages for batch write efficiency.
			n := len(c.send)
			for i := 0; i < n; i++ {
				queuedMsg := <-c.send
				if err := c.conn.WriteMessage(websocket.TextMessage, queuedMsg); err != nil {
					c.logger.Debug().Err(err).Msg("ws batch write error")
					return
				}
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(c.config.WriteTimeout))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.logger.Debug().Err(err).Msg("ws ping error")
				return
			}
		}
	}
}

// Send queues a message for delivery to the client.
// Returns false if the send buffer is full (message dropped).
func (c *Client) Send(msg []byte) bool {
	if c.closed.Load() {
		return false
	}
	select {
	case c.send <- msg:
		return true
	default:
		c.logger.Warn().Msg("ws send buffer full, dropping message")
		return false
	}
}

// Close closes the client connection idempotently.
func (c *Client) Close() {
	if c.closed.CompareAndSwap(false, true) {
		close(c.send)
		_ = c.conn.Close()
	}
}
