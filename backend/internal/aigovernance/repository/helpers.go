package repository

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
)

var ErrNotFound = errors.New("ai governance record not found")

func nullString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func nullJSON(value []byte, fallback string) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(fallback)
	}
	return json.RawMessage(value)
}

func ptrTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copyValue := value.UTC()
	return &copyValue
}

func ptrUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
}

func loggerWithRepo(logger zerolog.Logger, name string) zerolog.Logger {
	return logger.With().Str("repository", name).Logger()
}

func rowNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
