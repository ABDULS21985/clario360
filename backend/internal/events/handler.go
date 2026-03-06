package events

import "context"

// EventHandler processes a single event.
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
}

// EventHandlerFunc is a function adapter for EventHandler.
type EventHandlerFunc func(ctx context.Context, event *Event) error

// Handle implements EventHandler.
func (f EventHandlerFunc) Handle(ctx context.Context, event *Event) error {
	return f(ctx, event)
}

// HandlerRegistry is a simple registry that maps event types to handlers.
type HandlerRegistry struct {
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
	r.handlers[eventType] = handler
}

// RegisterFunc adds a function handler for the given event type.
func (r *HandlerRegistry) RegisterFunc(eventType string, fn func(ctx context.Context, event *Event) error) {
	r.handlers[eventType] = EventHandlerFunc(fn)
}

// Get returns the handler for the given event type, or nil if not found.
func (r *HandlerRegistry) Get(eventType string) EventHandler {
	return r.handlers[eventType]
}

// Handle implements EventHandler, routing events to their registered handler by type.
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
