package analytics

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/data/model"
	"github.com/clario360/platform/internal/data/repository"
)

type AuditRecorder struct {
	repo   *repository.AnalyticsRepository
	logger zerolog.Logger
}

func NewAuditRecorder(repo *repository.AnalyticsRepository, logger zerolog.Logger) *AuditRecorder {
	return &AuditRecorder{repo: repo, logger: logger}
}

func (r *AuditRecorder) RecordQueryExecution(ctx context.Context, tenantID, userID, modelID, sourceID uuid.UUID, query model.AnalyticsQuery, columnsAccessed, piiColumnsAccessed []string, piiMaskingApplied bool, rowsReturned int, truncated bool, executionTimeMs int64, errorOccurred bool, errorMessage string, savedQueryID *uuid.UUID, ipAddress, userAgent string) {
	filtersJSON, _ := json.Marshal(query.Filters)
	entry := &model.AnalyticsAuditLog{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		UserID:             userID,
		ModelID:            modelID,
		SourceID:           sourceID,
		QueryDefinition:    query,
		ColumnsAccessed:    columnsAccessed,
		FiltersApplied:     filtersJSON,
		PIIColumnsAccessed: piiColumnsAccessed,
		PIIMaskingApplied:  piiMaskingApplied,
		RowsReturned:       rowsReturned,
		Truncated:          truncated,
		ExecutionTimeMs:    &executionTimeMs,
		ErrorOccurred:      errorOccurred,
		SavedQueryID:       savedQueryID,
		ExecutedAt:         time.Now().UTC(),
	}
	if ipAddress = strings.TrimSpace(ipAddress); ipAddress != "" {
		entry.IPAddress = &ipAddress
	}
	if userAgent = strings.TrimSpace(userAgent); userAgent != "" {
		entry.UserAgent = &userAgent
	}
	if strings.TrimSpace(errorMessage) != "" {
		entry.ErrorMessage = &errorMessage
	}
	if err := r.repo.CreateAuditLog(ctx, entry); err != nil {
		r.logger.Error().Err(err).Str("tenant_id", tenantID.String()).Str("model_id", modelID.String()).Msg("failed to record analytics audit log")
	}
}
