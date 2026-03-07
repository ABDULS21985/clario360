package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

var (
	ErrNotFound   = errors.New("not found")
	ErrConflict   = errors.New("conflict")
	ErrValidation = errors.New("validation")
)

type pageMeta struct {
	Limit  int
	Offset int
}

type Store struct {
	db             *pgxpool.Pool
	logger         zerolog.Logger
	Dashboards     *DashboardRepository
	Widgets        *WidgetRepository
	KPIs           *KPIRepository
	KPISnapshots   *KPISnapshotRepository
	Alerts         *AlertRepository
	Reports        *ReportRepository
	ReportSnapshots *ReportSnapshotRepository
	SuiteCache     *SuiteCacheRepository
}

func NewStore(db *pgxpool.Pool, logger zerolog.Logger) *Store {
	store := &Store{
		db:     db,
		logger: logger,
	}
	store.Dashboards = NewDashboardRepository(db, logger)
	store.Widgets = NewWidgetRepository(db, logger)
	store.KPIs = NewKPIRepository(db, logger)
	store.KPISnapshots = NewKPISnapshotRepository(db, logger)
	store.Alerts = NewAlertRepository(db, logger)
	store.Reports = NewReportRepository(db, logger)
	store.ReportSnapshots = NewReportSnapshotRepository(db, logger)
	store.SuiteCache = NewSuiteCacheRepository(db, logger)
	return store
}

func normalizePagination(page, perPage int) pageMeta {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 25
	}
	if perPage > 200 {
		perPage = 200
	}
	return pageMeta{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}
}

func marshalJSON(value any) []byte {
	if value == nil {
		return []byte("{}")
	}
	out, err := json.Marshal(value)
	if err != nil {
		return []byte("{}")
	}
	return out
}

func marshalJSONArray(value any) []byte {
	if value == nil {
		return []byte("[]")
	}
	out, err := json.Marshal(value)
	if err != nil {
		return []byte("[]")
	}
	return out
}

func unmarshalMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func likePattern(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "%"
	}
	return "%" + trimmed + "%"
}

func wrapErr(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}
