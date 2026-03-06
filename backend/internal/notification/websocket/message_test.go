package websocket

import (
	"encoding/json"
	"testing"
)

func TestNewWSMessage_WithData(t *testing.T) {
	data := ConnectionAckData{UserID: "user-1", SessionID: "sess-1"}
	msg, err := NewWSMessage(MsgTypeConnectionAck, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed WSMessage
	if err := json.Unmarshal(msg, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != MsgTypeConnectionAck {
		t.Errorf("expected type %s, got %s", MsgTypeConnectionAck, parsed.Type)
	}
	if parsed.Timestamp == "" {
		t.Error("expected non-empty timestamp")
	}
	if len(parsed.Data) == 0 {
		t.Error("expected non-empty data")
	}

	var ackData ConnectionAckData
	if err := json.Unmarshal(parsed.Data, &ackData); err != nil {
		t.Fatalf("failed to unmarshal ack data: %v", err)
	}
	if ackData.UserID != "user-1" {
		t.Errorf("expected user-1, got %s", ackData.UserID)
	}
}

func TestNewWSMessage_WithoutData(t *testing.T) {
	msg, err := NewWSMessage("ping", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed WSMessage
	if err := json.Unmarshal(msg, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if parsed.Type != "ping" {
		t.Errorf("expected type ping, got %s", parsed.Type)
	}
	if parsed.Data != nil {
		t.Error("expected nil data for ping")
	}
}

func TestNewWSMessage_UnreadCount(t *testing.T) {
	data := UnreadCountData{Count: 42}
	msg, err := NewWSMessage(MsgTypeUnreadCount, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed WSMessage
	if err := json.Unmarshal(msg, &parsed); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	var countData UnreadCountData
	if err := json.Unmarshal(parsed.Data, &countData); err != nil {
		t.Fatalf("failed to unmarshal count data: %v", err)
	}
	if countData.Count != 42 {
		t.Errorf("expected count 42, got %d", countData.Count)
	}
}
