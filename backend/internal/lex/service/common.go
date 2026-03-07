package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	apperrors "github.com/clario360/platform/internal/errors"
	"github.com/clario360/platform/internal/events"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, event *events.Event) error
}

type noopPublisher struct{}

func (noopPublisher) Publish(context.Context, string, *events.Event) error { return nil }

func publisherOrNoop(p Publisher) Publisher {
	if p == nil {
		return noopPublisher{}
	}
	return p
}

func validationError(message string, fields map[string]string) error {
	return apperrors.NewValidation("VALIDATION_ERROR", message, fields)
}

func notFoundError(message string) error {
	return apperrors.NewNotFound("NOT_FOUND", message)
}

func conflictError(message string) error {
	return apperrors.NewConflict("CONFLICT", message)
}

func forbiddenError(message string) error {
	return apperrors.NewForbidden("FORBIDDEN", message)
}

func internalError(message string, err error) error {
	return apperrors.NewInternal("INTERNAL_ERROR", message, err)
}

func writeEvent(ctx context.Context, publisher Publisher, source, topic, eventType string, tenantID uuid.UUID, userID *uuid.UUID, payload any, logger zerolog.Logger) {
	event, err := events.NewEvent(eventType, source, tenantID.String(), payload)
	if err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Msg("build lex event")
		return
	}
	if userID != nil {
		event.UserID = userID.String()
	}
	if err := publisher.Publish(ctx, topic, event); err != nil {
		logger.Error().Err(err).Str("topic", topic).Str("event_type", eventType).Msg("publish lex event")
	}
}

func httpStatus(err error) int {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return http.StatusInternalServerError
}

func clampScore(score float64) float64 {
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeDate(value time.Time) time.Time {
	return time.Date(value.UTC().Year(), value.UTC().Month(), value.UTC().Day(), 0, 0, 0, 0, time.UTC)
}

func changedFields(before, after map[string]any) []string {
	keys := make([]string, 0)
	for key, afterValue := range after {
		if fmt.Sprintf("%v", before[key]) != fmt.Sprintf("%v", afterValue) {
			keys = append(keys, key)
		}
	}
	return keys
}
