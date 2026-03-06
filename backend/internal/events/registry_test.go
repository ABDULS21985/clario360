package events

import (
	"context"
	"errors"
	"testing"
)

func TestHandlerRegistry_RegisterAndHandle(t *testing.T) {
	registry := NewHandlerRegistry()

	var handled bool
	registry.RegisterFunc("com.clario360.test.event", func(ctx context.Context, event *Event) error {
		handled = true
		return nil
	})

	event := &Event{Type: "com.clario360.test.event", TenantID: "t1"}
	if err := registry.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle failed: %v", err)
	}
	if !handled {
		t.Error("expected handler to be called")
	}
}

func TestHandlerRegistry_UnregisteredType(t *testing.T) {
	registry := NewHandlerRegistry()
	event := &Event{Type: "com.clario360.unknown", TenantID: "t1"}
	err := registry.Handle(context.Background(), event)
	if err != nil {
		t.Errorf("expected nil error for unregistered type, got: %v", err)
	}
}

func TestHandlerRegistry_HandlerError(t *testing.T) {
	registry := NewHandlerRegistry()

	expectedErr := errors.New("handler failed")
	registry.RegisterFunc("com.clario360.fail", func(ctx context.Context, event *Event) error {
		return expectedErr
	})

	event := &Event{Type: "com.clario360.fail", TenantID: "t1"}
	err := registry.Handle(context.Background(), event)
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
}

func TestHandlerRegistry_Dispatch(t *testing.T) {
	registry := NewHandlerRegistry()

	var dispatched bool
	registry.RegisterFunc("com.clario360.dispatch.test", func(ctx context.Context, event *Event) error {
		dispatched = true
		return nil
	})

	event := &Event{Type: "com.clario360.dispatch.test", TenantID: "t1"}
	if err := registry.Dispatch(context.Background(), event); err != nil {
		t.Fatalf("Dispatch failed: %v", err)
	}
	if !dispatched {
		t.Error("expected dispatch handler to be called")
	}
}

func TestHandlerRegistry_EventTypes(t *testing.T) {
	registry := NewHandlerRegistry()
	noop := func(ctx context.Context, event *Event) error { return nil }

	registry.RegisterFunc("type-a", noop)
	registry.RegisterFunc("type-b", noop)
	registry.RegisterFunc("type-c", noop)

	types := registry.EventTypes()
	if len(types) != 3 {
		t.Errorf("expected 3 event types, got %d", len(types))
	}
}

func TestHandlerRegistry_HasHandler(t *testing.T) {
	registry := NewHandlerRegistry()
	noop := func(ctx context.Context, event *Event) error { return nil }

	registry.RegisterFunc("exists", noop)

	if !registry.HasHandler("exists") {
		t.Error("expected HasHandler to return true for registered type")
	}
	if registry.HasHandler("missing") {
		t.Error("expected HasHandler to return false for unregistered type")
	}
}

func TestHandlerRegistry_Register_Overwrite(t *testing.T) {
	registry := NewHandlerRegistry()

	var callCount int
	registry.RegisterFunc("type-x", func(ctx context.Context, event *Event) error {
		callCount = 1
		return nil
	})
	registry.RegisterFunc("type-x", func(ctx context.Context, event *Event) error {
		callCount = 2
		return nil
	})

	event := &Event{Type: "type-x", TenantID: "t1"}
	_ = registry.Handle(context.Background(), event)
	if callCount != 2 {
		t.Errorf("expected second handler to be called (callCount=2), got %d", callCount)
	}
}
