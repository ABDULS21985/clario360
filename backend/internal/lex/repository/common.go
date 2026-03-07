package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type Queryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Store struct {
	Contracts  *ContractRepository
	Clauses    *ClauseRepository
	Documents  *DocumentRepository
	Compliance *ComplianceRepository
	Alerts     *AlertRepository
}

func NewStore(db *pgxpool.Pool, logger zerolog.Logger) *Store {
	return &Store{
		Contracts:  NewContractRepository(db, logger),
		Clauses:    NewClauseRepository(db, logger),
		Documents:  NewDocumentRepository(db, logger),
		Compliance: NewComplianceRepository(db, logger),
		Alerts:     NewAlertRepository(db, logger),
	}
}

func queryRowJSON[T any](ctx context.Context, q Queryer, query string, args ...any) (*T, error) {
	var raw []byte
	if err := q.QueryRow(ctx, query, args...).Scan(&raw); err != nil {
		return nil, err
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("unmarshal row json: %w", err)
	}
	return &out, nil
}

func queryListJSON[T any](ctx context.Context, q Queryer, query string, args ...any) ([]T, error) {
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]T, 0)
	for rows.Next() {
		var raw []byte
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var item T
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("unmarshal row json: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func buildWhere(base []string, filters ...string) string {
	conditions := make([]string, 0, len(base)+len(filters))
	conditions = append(conditions, base...)
	for _, filter := range filters {
		if strings.TrimSpace(filter) != "" {
			conditions = append(conditions, filter)
		}
	}
	return strings.Join(conditions, " AND ")
}
