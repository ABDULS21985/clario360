package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/events"
)

var (
	ErrValidation = errors.New("validation failed")
	ErrForbidden  = errors.New("forbidden")
)

type Publisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

func publishEvent(ctx context.Context, publisher Publisher, tenantID uuid.UUID, eventType string, payload any) error {
	if publisher == nil {
		return nil
	}
	event, err := events.NewEvent(eventType, "visus-service", tenantID.String(), payload)
	if err != nil {
		return err
	}
	return publisher.Publish(ctx, events.Topics.VisusEvents, event)
}

func requireName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: name is required", ErrValidation)
	}
	return nil
}
