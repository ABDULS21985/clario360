package websocket

import (
	"context"
	"testing"
	"time"

	"github.com/rs/zerolog"
)

func newTestHub() *Hub {
	logger := zerolog.Nop()
	return NewHub(3, logger)
}

type mockConn struct{}

func TestHub_RegisterAndUnregister(t *testing.T) {
	hub := newTestHub()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go hub.Run(ctx)
	time.Sleep(10 * time.Millisecond)

	if hub.ActiveConnections("tenant-1") != 0 {
		t.Error("expected 0 connections initially")
	}
}

func TestHub_ActiveConnections(t *testing.T) {
	hub := newTestHub()

	hub.mu.Lock()
	hub.clients["tenant-1"] = map[string][]*Client{
		"user-1": {
			{userID: "user-1", tenantID: "tenant-1"},
			{userID: "user-1", tenantID: "tenant-1"},
		},
		"user-2": {
			{userID: "user-2", tenantID: "tenant-1"},
		},
	}
	hub.mu.Unlock()

	if hub.ActiveConnections("tenant-1") != 3 {
		t.Errorf("expected 3 connections, got %d", hub.ActiveConnections("tenant-1"))
	}
	if hub.ActiveUserConnections("tenant-1", "user-1") != 2 {
		t.Errorf("expected 2 connections for user-1, got %d", hub.ActiveUserConnections("tenant-1", "user-1"))
	}
	if hub.ActiveConnections("tenant-2") != 0 {
		t.Errorf("expected 0 connections for tenant-2")
	}
}

func TestHub_SendToUser_NoClients(t *testing.T) {
	hub := newTestHub()
	sent := hub.SendToUser("tenant-1", "user-1", []byte("test"))
	if sent != 0 {
		t.Errorf("expected 0 sent for non-existent user, got %d", sent)
	}
}

func TestHub_BroadcastToTenant_NoClients(t *testing.T) {
	hub := newTestHub()
	sent := hub.BroadcastToTenant("tenant-1", []byte("test"))
	if sent != 0 {
		t.Errorf("expected 0 sent for non-existent tenant, got %d", sent)
	}
}
