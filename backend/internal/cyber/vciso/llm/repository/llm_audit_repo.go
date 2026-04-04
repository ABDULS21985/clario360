package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

var ErrNotFound = errors.New("not found")

type LLMAuditRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewLLMAuditRepository(db *pgxpool.Pool, logger zerolog.Logger) *LLMAuditRepository {
	return &LLMAuditRepository{
		db:     db,
		logger: logger.With().Str("component", "vciso-llm-audit-repo").Logger(),
	}
}

func (r *LLMAuditRepository) CreateAudit(ctx context.Context, item *llmmodel.AuditLog) error {
	if item == nil {
		return fmt.Errorf("audit log is required")
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	toolCalls := defaultJSON(item.ToolCallsJSON, []byte("[]"))
	reasoning := defaultJSON(item.ReasoningTrace, []byte("[]"))
	return r.db.QueryRow(ctx, `
		INSERT INTO vciso_llm_audit_log (
			id, message_id, conversation_id, tenant_id, user_id, provider, model,
			prompt_tokens, completion_tokens, total_tokens, estimated_cost_usd,
			llm_latency_ms, total_latency_ms, system_prompt_hash, system_prompt_version,
			user_message, context_turns, raw_completion, tool_calls_json, tool_call_count,
			reasoning_trace, grounding_result, pii_detections, injection_flags,
			final_response, prediction_log_id, engine_used, routing_reason, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,
			$8,$9,$10,$11,
			$12,$13,$14,$15,
			$16,$17,$18,$19,$20,
			$21,$22,$23,$24,
			$25,$26,$27,$28,$29
		)
		RETURNING created_at`,
		item.ID, item.MessageID, item.ConversationID, item.TenantID, item.UserID, item.Provider, item.Model,
		item.PromptTokens, item.CompletionTokens, item.TotalTokens, item.EstimatedCostUSD,
		item.LLMLatencyMS, item.TotalLatencyMS, item.SystemPromptHash, item.SystemPromptVersion,
		item.UserMessage, item.ContextTurns, item.RawCompletion, toolCalls, item.ToolCallCount,
		reasoning, item.GroundingResult, item.PIIDetections, item.InjectionFlags,
		item.FinalResponse, item.PredictionLogID, item.EngineUsed, item.RoutingReason, item.CreatedAt,
	).Scan(&item.CreatedAt)
}

func (r *LLMAuditRepository) GetAuditByMessageID(ctx context.Context, tenantID, messageID uuid.UUID) (*llmmodel.AuditLog, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, message_id, conversation_id, tenant_id, user_id, provider, model,
		       prompt_tokens, completion_tokens, total_tokens, estimated_cost_usd,
		       llm_latency_ms, total_latency_ms, system_prompt_hash, system_prompt_version,
		       user_message, context_turns, raw_completion, tool_calls_json, tool_call_count,
		       reasoning_trace, grounding_result, pii_detections, injection_flags,
		       final_response, prediction_log_id, engine_used, COALESCE(routing_reason, ''), created_at
		FROM vciso_llm_audit_log
		WHERE tenant_id = $1 AND message_id = $2
		ORDER BY created_at DESC
		LIMIT 1`,
		tenantID, messageID,
	)
	return scanAudit(row)
}

func (r *LLMAuditRepository) UsageStats(ctx context.Context, tenantID uuid.UUID) (*llmmodel.UsageStats, error) {
	now := time.Now().UTC()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)

	stats := &llmmodel.UsageStats{}
	if err := r.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE created_at >= $2),
			COALESCE(SUM(total_tokens) FILTER (WHERE created_at >= $2), 0),
			COALESCE(SUM(estimated_cost_usd) FILTER (WHERE created_at >= $2), 0),
			COUNT(*) FILTER (WHERE created_at >= $3),
			COALESCE(SUM(estimated_cost_usd) FILTER (WHERE created_at >= $3), 0)
		FROM vciso_llm_audit_log
		WHERE tenant_id = $1`,
		tenantID, dayStart, monthStart,
	).Scan(&stats.CallsToday, &stats.TokensToday, &stats.CostToday, &stats.CallsThisMonth, &stats.CostThisMonth); err != nil {
		return nil, fmt.Errorf("load usage stats: %w", err)
	}
	return stats, nil
}

func (r *LLMAuditRepository) ListPrompts(ctx context.Context) ([]llmmodel.SystemPrompt, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, version, prompt_text, prompt_hash, tool_schemas, description, created_by, active, created_at
		FROM vciso_llm_system_prompts
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list llm prompts: %w", err)
	}
	defer rows.Close()

	items := make([]llmmodel.SystemPrompt, 0)
	for rows.Next() {
		item, err := scanPrompt(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func (r *LLMAuditRepository) GetActivePrompt(ctx context.Context) (*llmmodel.SystemPrompt, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, version, prompt_text, prompt_hash, tool_schemas, description, created_by, active, created_at
		FROM vciso_llm_system_prompts
		WHERE active = true
		LIMIT 1`)
	return scanPrompt(row)
}

func (r *LLMAuditRepository) CreatePrompt(ctx context.Context, item *llmmodel.SystemPrompt) error {
	if item == nil {
		return fmt.Errorf("prompt is required")
	}
	if item.ID == uuid.Nil {
		item.ID = uuid.New()
	}
	if item.CreatedAt.IsZero() {
		item.CreatedAt = time.Now().UTC()
	}
	toolSchemas := defaultJSON(item.ToolSchemas, []byte("[]"))
	return r.db.QueryRow(ctx, `
		INSERT INTO vciso_llm_system_prompts (
			id, version, prompt_text, prompt_hash, tool_schemas, description, created_by, active, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		RETURNING created_at`,
		item.ID, item.Version, item.PromptText, item.PromptHash, toolSchemas, item.Description, item.CreatedBy, item.Active, item.CreatedAt,
	).Scan(&item.CreatedAt)
}

func (r *LLMAuditRepository) ActivatePrompt(ctx context.Context, version string) error {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `UPDATE vciso_llm_system_prompts SET active = false WHERE active = true`); err != nil {
		return fmt.Errorf("deactivate prompts: %w", err)
	}
	tag, err := tx.Exec(ctx, `UPDATE vciso_llm_system_prompts SET active = true WHERE version = $1`, version)
	if err != nil {
		return fmt.Errorf("activate prompt: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (r *LLMAuditRepository) EnsurePrompt(ctx context.Context, item *llmmodel.SystemPrompt) error {
	if item == nil {
		return nil
	}
	if _, err := r.GetActivePrompt(ctx); err == nil {
		return nil
	} else if !errors.Is(err, ErrNotFound) {
		return err
	}
	item.Active = true
	return r.CreatePrompt(ctx, item)
}

func (r *LLMAuditRepository) GetRateLimit(ctx context.Context, tenantID uuid.UUID, defaults llmcfg.RateLimitDefaults) (*llmmodel.RateLimitRecord, error) {
	if err := r.ensureRateLimit(ctx, tenantID, defaults); err != nil {
		return nil, err
	}
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, max_calls_per_minute, max_calls_per_hour, max_calls_per_day,
		       max_tokens_per_day, max_cost_per_day_usd, current_calls_minute, current_calls_hour,
		       current_calls_day, current_tokens_day, current_cost_day_usd, minute_reset_at,
		       hour_reset_at, day_reset_at, updated_at
		FROM vciso_llm_rate_limits
		WHERE tenant_id = $1`,
		tenantID,
	)
	return scanRateLimit(row)
}

func (r *LLMAuditRepository) SaveRateLimit(ctx context.Context, item *llmmodel.RateLimitRecord) error {
	if item == nil {
		return fmt.Errorf("rate limit record is required")
	}
	_, err := r.db.Exec(ctx, `
		UPDATE vciso_llm_rate_limits
		SET max_calls_per_minute = $2,
		    max_calls_per_hour = $3,
		    max_calls_per_day = $4,
		    max_tokens_per_day = $5,
		    max_cost_per_day_usd = $6,
		    current_calls_minute = $7,
		    current_calls_hour = $8,
		    current_calls_day = $9,
		    current_tokens_day = $10,
		    current_cost_day_usd = $11,
		    minute_reset_at = $12,
		    hour_reset_at = $13,
		    day_reset_at = $14,
		    updated_at = now()
		WHERE tenant_id = $1`,
		item.TenantID, item.MaxCallsPerMinute, item.MaxCallsPerHour, item.MaxCallsPerDay,
		item.MaxTokensPerDay, item.MaxCostPerDayUSD, item.CurrentCallsMinute, item.CurrentCallsHour,
		item.CurrentCallsDay, item.CurrentTokensDay, item.CurrentCostDayUSD, item.MinuteResetAt,
		item.HourResetAt, item.DayResetAt,
	)
	if err != nil {
		return fmt.Errorf("save rate limit: %w", err)
	}
	return nil
}

func (r *LLMAuditRepository) ensureRateLimit(ctx context.Context, tenantID uuid.UUID, defaults llmcfg.RateLimitDefaults) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO vciso_llm_rate_limits (
			tenant_id, max_calls_per_minute, max_calls_per_hour, max_calls_per_day,
			max_tokens_per_day, max_cost_per_day_usd
		) VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (tenant_id) DO NOTHING`,
		tenantID, defaults.MaxCallsPerMinute, defaults.MaxCallsPerHour, defaults.MaxCallsPerDay,
		defaults.MaxTokensPerDay, defaults.MaxCostPerDayUSD,
	)
	if err != nil {
		return fmt.Errorf("ensure rate limit: %w", err)
	}
	return nil
}

func scanAudit(row interface{ Scan(...any) error }) (*llmmodel.AuditLog, error) {
	var item llmmodel.AuditLog
	var routingReason string
	if err := row.Scan(
		&item.ID, &item.MessageID, &item.ConversationID, &item.TenantID, &item.UserID, &item.Provider, &item.Model,
		&item.PromptTokens, &item.CompletionTokens, &item.TotalTokens, &item.EstimatedCostUSD,
		&item.LLMLatencyMS, &item.TotalLatencyMS, &item.SystemPromptHash, &item.SystemPromptVersion,
		&item.UserMessage, &item.ContextTurns, &item.RawCompletion, &item.ToolCallsJSON, &item.ToolCallCount,
		&item.ReasoningTrace, &item.GroundingResult, &item.PIIDetections, &item.InjectionFlags,
		&item.FinalResponse, &item.PredictionLogID, &item.EngineUsed, &routingReason, &item.CreatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	item.RoutingReason = routingReason
	return &item, nil
}

func scanPrompt(row interface{ Scan(...any) error }) (*llmmodel.SystemPrompt, error) {
	var item llmmodel.SystemPrompt
	if err := row.Scan(&item.ID, &item.Version, &item.PromptText, &item.PromptHash, &item.ToolSchemas, &item.Description, &item.CreatedBy, &item.Active, &item.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func scanRateLimit(row interface{ Scan(...any) error }) (*llmmodel.RateLimitRecord, error) {
	var item llmmodel.RateLimitRecord
	if err := row.Scan(
		&item.ID, &item.TenantID, &item.MaxCallsPerMinute, &item.MaxCallsPerHour, &item.MaxCallsPerDay,
		&item.MaxTokensPerDay, &item.MaxCostPerDayUSD, &item.CurrentCallsMinute, &item.CurrentCallsHour,
		&item.CurrentCallsDay, &item.CurrentTokensDay, &item.CurrentCostDayUSD, &item.MinuteResetAt,
		&item.HourResetAt, &item.DayResetAt, &item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &item, nil
}

func defaultJSON(value json.RawMessage, fallback []byte) []byte {
	if len(value) == 0 {
		return fallback
	}
	return value
}
