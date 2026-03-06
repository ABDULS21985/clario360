package events

import (
	"context"
	"sync"
)

// HandlerRegistry maps event types to their handlers and supports dispatching
// events to the correct handler based on event type.
type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]EventHandler
}

// NewHandlerRegistry creates a new handler registry.
func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		handlers: make(map[string]EventHandler),
	}
}

// Register adds a handler for the given event type.
func (r *HandlerRegistry) Register(eventType string, handler EventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = handler
}

// RegisterFunc adds a function handler for the given event type.
func (r *HandlerRegistry) RegisterFunc(eventType string, fn func(ctx context.Context, event *Event) error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[eventType] = EventHandlerFunc(fn)
}

// RegisterTyped registers a TypedEventHandler for all event types it declares.
func (r *HandlerRegistry) RegisterTyped(handler TypedEventHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, eventType := range handler.EventTypes() {
		r.handlers[eventType] = handler
	}
}

// Get returns the handler for the given event type, or nil if not found.
func (r *HandlerRegistry) Get(eventType string) EventHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[eventType]
}

// Handle implements EventHandler, routing events to their registered handler by type.
// Returns nil if no handler is registered for the event type (silently skips).
func (r *HandlerRegistry) Handle(ctx context.Context, event *Event) error {
	handler := r.Get(event.Type)
	if handler == nil {
		return nil
	}
	return handler.Handle(ctx, event)
}

// Dispatch is an alias for Handle.
func (r *HandlerRegistry) Dispatch(ctx context.Context, event *Event) error {
	return r.Handle(ctx, event)
}

// EventTypes returns all registered event types.
func (r *HandlerRegistry) EventTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	types := make([]string, 0, len(r.handlers))
	for t := range r.handlers {
		types = append(types, t)
	}
	return types
}

// HasHandler returns true if a handler is registered for the given event type.
func (r *HandlerRegistry) HasHandler(eventType string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.handlers[eventType]
	return ok
}
