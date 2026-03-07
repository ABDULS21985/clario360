package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/model"
)

type DBTX interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type Store struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewStore(db *pgxpool.Pool, logger zerolog.Logger) *Store {
	return &Store{db: db, logger: logger}
}

func (s *Store) DB() *pgxpool.Pool {
	return s.db
}

type rowScanner interface {
	Scan(dest ...any) error
}

func marshalJSON(value any) ([]byte, error) {
	if value == nil {
		return []byte("{}"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func marshalJSONArray(value any) ([]byte, error) {
	if value == nil {
		return []byte("[]"), nil
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

func decodeJSONMap(raw []byte) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = map[string]any{}
	}
	return out, nil
}

func decodeExtractedActions(raw []byte) ([]model.ExtractedAction, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var out []model.ExtractedAction
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeMeetingAttachments(raw any) ([]model.MeetingAttachment, error) {
	switch typed := raw.(type) {
	case nil:
		return nil, nil
	case []byte:
		if len(typed) == 0 {
			return nil, nil
		}
		var out []model.MeetingAttachment
		if err := json.Unmarshal(typed, &out); err != nil {
			return nil, err
		}
		return out, nil
	case string:
		if typed == "" {
			return nil, nil
		}
		return decodeMeetingAttachments([]byte(typed))
	default:
		buf, err := json.Marshal(typed)
		if err != nil {
			return nil, err
		}
		return decodeMeetingAttachments(buf)
	}
}

func attachmentMetadata(metadata map[string]any) []model.MeetingAttachment {
	if metadata == nil {
		return nil
	}
	value, ok := metadata["attachments"]
	if !ok {
		return nil
	}
	attachments, err := decodeMeetingAttachments(value)
	if err != nil {
		return nil
	}
	return attachments
}

func setAttachmentMetadata(metadata map[string]any, attachments []model.MeetingAttachment) map[string]any {
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["attachments"] = attachments
	return metadata
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if !AsPgError(err, &pgErr) {
		return ""
	}
	return pgErr.Code
}

func AsPgError(err error, target **pgconn.PgError) bool {
	return errors.As(err, target)
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func ptr[T any](value T) *T {
	return &value
}

func notFoundError(entity string, id uuid.UUID) error {
	return fmt.Errorf("%s %s not found", entity, id)
}
