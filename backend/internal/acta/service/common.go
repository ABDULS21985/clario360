package service

import (
	"context"
	"errors"
	"fmt"
	"math"
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

func publishEvent(ctx context.Context, publisher Publisher, source, topic, eventType string, tenantID uuid.UUID, userID *uuid.UUID, payload any, logger zerolog.Logger) {
	event, err := events.NewEvent(eventType, source, tenantID.String(), payload)
	if err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Msg("failed to build acta event")
		return
	}
	if userID != nil {
		event.UserID = userID.String()
	}
	if err := publisher.Publish(ctx, topic, event); err != nil {
		logger.Error().Err(err).Str("event_type", eventType).Str("topic", topic).Msg("failed to publish acta event")
	}
}

func computeQuorumRequired(activeMembers int, quorumType string, percentage int, fixedCount *int) (int, error) {
	switch quorumType {
	case "percentage":
		if activeMembers <= 0 {
			return 0, fmt.Errorf("active members must be greater than zero")
		}
		if percentage < 1 || percentage > 100 {
			return 0, fmt.Errorf("quorum percentage must be between 1 and 100")
		}
		return int(math.Ceil(float64(activeMembers*percentage) / 100.0)), nil
	case "fixed_count":
		if fixedCount == nil || *fixedCount < 1 {
			return 0, fmt.Errorf("fixed count quorum must be set")
		}
		return *fixedCount, nil
	default:
		return 0, fmt.Errorf("unsupported quorum type %q", quorumType)
	}
}

func quorumMet(required, present int) bool {
	return present >= required
}

func normalizeString(input string) string {
	return strings.TrimSpace(input)
}

func businessDaysBetween(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC)
	days := 0
	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			days++
		}
	}
	if days > 0 {
		days--
	}
	return days
}

func severityWeight(severity string) float64 {
	switch severity {
	case "critical":
		return 3
	case "high":
		return 2
	case "medium":
		return 1
	case "low":
		return 0.5
	default:
		return 1
	}
}

func ptr[T any](value T) *T {
	return &value
}

func unwrapAppError(err error) *apperrors.AppError {
	var appErr *apperrors.AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return nil
}
