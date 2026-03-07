package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
)

// VCISORepository handles vciso_briefings table operations.
type VCISORepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

// NewVCISORepository creates a new VCISORepository.
func NewVCISORepository(db *pgxpool.Pool, logger zerolog.Logger) *VCISORepository {
	return &VCISORepository{db: db, logger: logger}
}

// SaveBriefing stores a generated executive briefing.
func (r *VCISORepository) SaveBriefing(ctx context.Context, tenantID, generatedBy uuid.UUID, briefingType string, periodStart, periodEnd time.Time, content *model.ExecutiveBriefing, riskScore *float64) (*model.VCISOBriefingRecord, error) {
	id := uuid.New()
	now := time.Now().UTC()

	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal briefing content: %w", err)
	}

	_, err = r.db.Exec(ctx, `
		INSERT INTO vciso_briefings (id, tenant_id, type, period_start, period_end, content, risk_score_at_time, generated_by, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, tenantID, briefingType, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"),
		contentJSON, riskScore, generatedBy, now,
	)
	if err != nil {
		return nil, fmt.Errorf("save briefing: %w", err)
	}

	return &model.VCISOBriefingRecord{
		ID: id, TenantID: tenantID, Type: briefingType,
		PeriodStart: periodStart, PeriodEnd: periodEnd,
		Content: content, RiskScoreAtTime: riskScore,
		GeneratedBy: generatedBy, CreatedAt: now,
	}, nil
}

// ListBriefings retrieves briefing history.
func (r *VCISORepository) ListBriefings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOBriefingHistoryParams) ([]*model.VCISOBriefingRecord, int, error) {
	conds := []string{"tenant_id=$1"}
	args := []interface{}{tenantID}
	i := 2

	if params.Type != nil {
		conds = append(conds, fmt.Sprintf("type=$%d", i))
		args = append(args, *params.Type)
		i++
	}

	where := "WHERE " + strings.Join(conds, " AND ")

	var total int
	_ = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM vciso_briefings "+where, args...).Scan(&total)

	offset := (params.Page - 1) * params.PerPage
	query := fmt.Sprintf(
		`SELECT id, tenant_id, type, period_start, period_end, content, risk_score_at_time, generated_by, created_at
		 FROM vciso_briefings %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		where, i, i+1,
	)
	args = append(args, params.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list briefings: %w", err)
	}
	defer rows.Close()

	var records []*model.VCISOBriefingRecord
	for rows.Next() {
		rec, err := scanBriefing(rows)
		if err != nil {
			return nil, 0, err
		}
		records = append(records, rec)
	}
	return records, total, rows.Err()
}

func scanBriefing(row interface{ Scan(...interface{}) error }) (*model.VCISOBriefingRecord, error) {
	var rec model.VCISOBriefingRecord
	var contentJSON []byte
	var periodStart, periodEnd time.Time
	var riskScore sql.NullFloat64

	err := row.Scan(
		&rec.ID, &rec.TenantID, &rec.Type, &periodStart, &periodEnd,
		&contentJSON, &riskScore, &rec.GeneratedBy, &rec.CreatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("scan briefing: %w", err)
	}

	rec.PeriodStart = periodStart
	rec.PeriodEnd = periodEnd
	if riskScore.Valid {
		rec.RiskScoreAtTime = &riskScore.Float64
	}

	if contentJSON != nil {
		rec.Content = &model.ExecutiveBriefing{}
		if err := json.Unmarshal(contentJSON, rec.Content); err != nil {
			return nil, fmt.Errorf("unmarshal briefing content: %w", err)
		}
	}
	return &rec, nil
}
