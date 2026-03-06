package websocket

import (
	"encoding/json"
	"time"
)

// WSMessage is the envelope for all WebSocket messages.
type WSMessage struct {
	Type      string          `json:"type"`
	Data      json.RawMessage `json:"data,omitempty"`
	Timestamp string          `json:"timestamp"`
}

// Message type constants.
const (
	MsgTypeConnectionAck  = "connection.ack"
	MsgTypeNotificationNew = "notification.new"
	MsgTypeNotificationRead = "notification.read"
	MsgTypeUnreadCount    = "unread.count"
	MsgTypeError          = "error"
)

// NewWSMessage creates a new WebSocket message with a timestamp.
func NewWSMessage(msgType string, data interface{}) ([]byte, error) {
	var dataRaw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		dataRaw = b
	}

	msg := WSMessage{
		Type:      msgType,
		Data:      dataRaw,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	return json.Marshal(msg)
}

// ConnectionAckData is the payload for a connection.ack message.
type ConnectionAckData struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

// UnreadCountData is the payload for an unread.count message.
type UnreadCountData struct {
	Count int64 `json:"count"`
}

// ErrorData is the payload for an error message.
type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
