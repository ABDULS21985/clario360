package watchers

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// TransitWatcher monitors data in transit for encryption and authorization compliance.
type TransitWatcher struct {
	db        *pgxpool.Pool
	dataDB    *pgxpool.Pool
	alertRepo *repository.AlertRepository
	producer  *events.Producer
	logger    zerolog.Logger
	cancel    context.CancelFunc

	requireEncryption bool
	requireApproval   bool
}

// NewTransitWatcher creates a transit security watcher.
func NewTransitWatcher(
	db *pgxpool.Pool,
	dataDB *pgxpool.Pool,
	alertRepo *repository.AlertRepository,
	producer *events.Producer,
	requireEncryption, requireApproval bool,
	logger zerolog.Logger,
) *TransitWatcher {
	return &TransitWatcher{
		db:                db,
		dataDB:            dataDB,
		alertRepo:         alertRepo,
		producer:          producer,
		logger:            logger.With().Str("watcher", "transit").Logger(),
		requireEncryption: requireEncryption,
		requireApproval:   requireApproval,
	}
}

func (w *TransitWatcher) Name() string { return "transit" }

func (w *TransitWatcher) Start(ctx context.Context) error {
	ctx, w.cancel = context.WithCancel(ctx)
	<-ctx.Done()
	return nil
}

func (w *TransitWatcher) Stop() error {
	if w.cancel != nil {
		w.cancel()
	}
	return nil
}

// HandlePipelineRunning processes a pipeline running event to check transit security.
func (w *TransitWatcher) HandlePipelineRunning(ctx context.Context, evt *events.Event) error {
	var data EventData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		return fmt.Errorf("unmarshal pipeline event: %w", err)
	}

	if data.Status != "running" {
		return nil
	}

	tenantID, err := uuid.Parse(evt.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant ID: %w", err)
	}

	w.logger.Info().
		Str("pipeline_id", data.PipelineID.String()).
		Msg("checking transit security for running pipeline")

	// Check source encryption
	sourceEncrypted := w.checkConnectionEncryption(ctx, tenantID, data.SourceID)

	// Check target encryption
	targetEncrypted := w.checkConnectionEncryption(ctx, tenantID, data.TargetID)

	// Check pipeline approval
	hasApproval := w.checkPipelineApproval(ctx, tenantID, data.PipelineID)

	now := time.Now().UTC()

	// Alert for unencrypted source
	if w.requireEncryption && !sourceEncrypted && data.SourceID != nil {
		w.createTransitAlert(ctx, tenantID, data, "high",
			fmt.Sprintf("Data in transit without encryption: source connection for pipeline %s", data.PipelineName),
			"Pipeline is extracting data from source without TLS/SSL encryption enabled.",
			"source", data.SourceID, now)
	}

	// Alert for unencrypted target
	if w.requireEncryption && !targetEncrypted && data.TargetID != nil {
		w.createTransitAlert(ctx, tenantID, data, "high",
			fmt.Sprintf("Data in transit without encryption: target connection for pipeline %s", data.PipelineName),
			"Pipeline is loading data to target without TLS/SSL encryption enabled.",
			"target", data.TargetID, now)
	}

	// Alert for unapproved pipeline
	if w.requireApproval && !hasApproval {
		w.createTransitAlert(ctx, tenantID, data, "medium",
			fmt.Sprintf("Unapproved data transfer detected: pipeline %s", data.PipelineName),
			"Pipeline execution has no associated workflow approval record.",
			"approval", nil, now)
	}

	// Log transit event
	if w.producer != nil {
		transitEvt, _ := events.NewEvent("cyber.dspm.transit.checked", "cyber-service", tenantID.String(), map[string]interface{}{
			"pipeline_id":      data.PipelineID.String(),
			"pipeline_name":    data.PipelineName,
			"source_encrypted": sourceEncrypted,
			"target_encrypted": targetEncrypted,
			"has_approval":     hasApproval,
		})
		if transitEvt != nil {
			_ = w.producer.Publish(ctx, events.Topics.DSPMEvents, transitEvt)
		}
	}

	return nil
}

// checkConnectionEncryption checks if a data source connection uses TLS/SSL.
func (w *TransitWatcher) checkConnectionEncryption(ctx context.Context, tenantID uuid.UUID, sourceID *uuid.UUID) bool {
	if sourceID == nil || w.dataDB == nil {
		return true // Fail open when unknown
	}

	var config json.RawMessage
	err := w.dataDB.QueryRow(ctx, `
		SELECT connection_config FROM data_sources
		WHERE tenant_id = $1 AND id = $2 AND deleted_at IS NULL
	`, tenantID, *sourceID).Scan(&config)
	if err != nil {
		return true // Fail open on error
	}

	var connConfig map[string]interface{}
	if err := json.Unmarshal(config, &connConfig); err != nil {
		return true
	}

	// Check various SSL/TLS indicators
	if ssl, ok := connConfig["ssl"].(bool); ok {
		return ssl
	}
	if sslMode, ok := connConfig["sslmode"].(string); ok {
		return sslMode != "disable" && sslMode != ""
	}
	if tls, ok := connConfig["tls"].(bool); ok {
		return tls
	}
	if tls, ok := connConfig["tls_enabled"].(bool); ok {
		return tls
	}
	if scheme, ok := connConfig["url"].(string); ok {
		return len(scheme) > 8 && scheme[:8] == "https://"
	}

	return true // Default to true if can't determine
}

// checkPipelineApproval checks if the pipeline has a workflow approval.
func (w *TransitWatcher) checkPipelineApproval(ctx context.Context, tenantID, pipelineID uuid.UUID) bool {
	if w.dataDB == nil {
		return true
	}

	var count int
	err := w.dataDB.QueryRow(ctx, `
		SELECT COUNT(*) FROM workflow_instances
		WHERE tenant_id = $1
		  AND reference_id = $2
		  AND status = 'completed'
		  AND created_at > NOW() - INTERVAL '30 days'
	`, tenantID, pipelineID).Scan(&count)
	if err != nil {
		return true // Fail open on error
	}
	return count > 0
}

func (w *TransitWatcher) createTransitAlert(ctx context.Context, tenantID uuid.UUID, data EventData, severity, title, description, checkType string, assetID *uuid.UUID, now time.Time) {
	metadata, _ := json.Marshal(map[string]interface{}{
		"pipeline_id":   data.PipelineID.String(),
		"pipeline_name": data.PipelineName,
		"check_type":    checkType,
	})

	alert := &model.Alert{
		ID:          uuid.New(),
		TenantID:    tenantID,
		Title:       title,
		Description: description,
		Severity:    model.Severity(severity),
		Status:      model.AlertStatusNew,
		Source:      "dspm_transit",
		AssetID:     assetID,
		Explanation: model.AlertExplanation{
			Summary: title,
			Reason:  description,
			Evidence: []model.AlertEvidence{
				{Label: "Pipeline", Field: "pipeline_id", Value: data.PipelineID.String(), Description: "Pipeline executing data transfer"},
				{Label: "Check Type", Field: "check_type", Value: checkType, Description: "Type of transit security check that failed"},
			},
			RecommendedActions: transitRecommendations(checkType),
		},
		ConfidenceScore: 0.95,
		EventCount:      1,
		FirstEventAt:    now,
		LastEventAt:     now,
		Tags:            []string{"dspm", "transit", checkType},
		Metadata:        metadata,
	}

	if assetID != nil {
		alert.AssetIDs = []uuid.UUID{*assetID}
	}

	if _, err := w.alertRepo.Create(ctx, alert); err != nil {
		w.logger.Error().Err(err).Msg("create transit alert")
	}
}

func transitRecommendations(checkType string) []string {
	switch checkType {
	case "source", "target":
		return []string{
			"Enable TLS/SSL on the data source connection.",
			"Update connection configuration to use encrypted transport.",
			"Verify SSL certificates are valid and trusted.",
		}
	case "approval":
		return []string{
			"Submit the pipeline for workflow approval before execution.",
			"Review and approve the data transfer through the governance workflow.",
			"Consider pausing the pipeline until approval is obtained.",
		}
	default:
		return []string{"Review the pipeline configuration and security settings."}
	}
}
