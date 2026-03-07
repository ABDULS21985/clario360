package ctem

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type WorkflowLauncher interface {
	StartRemediation(ctx context.Context, tenantID, userID uuid.UUID, group *model.CTEMRemediationGroup, assessment *model.CTEMAssessment) (string, error)
}

type CTEMEngine struct {
	db            *pgxpool.Pool
	assessmentRepo *repository.CTEMAssessmentRepository
	findingRepo   *repository.CTEMFindingRepository
	snapshotRepo  *repository.CTEMSnapshotRepository
	remGroupRepo  *repository.CTEMRemediationGroupRepository
	assetRepo     *repository.AssetRepository
	vulnRepo      *repository.VulnerabilityRepository
	relationRepo  *repository.RelationshipRepository
	scoring       *ScoringEngine
	producer      *events.Producer
	workflow      WorkflowLauncher
	logger        zerolog.Logger
}

func NewEngine(
	db *pgxpool.Pool,
	assessmentRepo *repository.CTEMAssessmentRepository,
	findingRepo *repository.CTEMFindingRepository,
	snapshotRepo *repository.CTEMSnapshotRepository,
	remGroupRepo *repository.CTEMRemediationGroupRepository,
	assetRepo *repository.AssetRepository,
	vulnRepo *repository.VulnerabilityRepository,
	relationRepo *repository.RelationshipRepository,
	scoring *ScoringEngine,
	producer *events.Producer,
	workflow WorkflowLauncher,
	logger zerolog.Logger,
) *CTEMEngine {
	return &CTEMEngine{
		db:             db,
		assessmentRepo: assessmentRepo,
		findingRepo:    findingRepo,
		snapshotRepo:   snapshotRepo,
		remGroupRepo:   remGroupRepo,
		assetRepo:      assetRepo,
		vulnRepo:       vulnRepo,
		relationRepo:   relationRepo,
		scoring:        scoring,
		producer:       producer,
		workflow:       workflow,
		logger:         logger.With().Str("component", "ctem-engine").Logger(),
	}
}

func (e *CTEMEngine) RunAssessment(ctx context.Context, assessmentID uuid.UUID) error {
	assessment, err := e.assessmentRepo.GetByIDAnyTenant(ctx, assessmentID)
	if err != nil {
		return err
	}

	if assessment.Status != model.CTEMAssessmentStatusCreated &&
		assessment.Status != model.CTEMAssessmentStatusFailed &&
		assessment.Status != model.CTEMAssessmentStatusCancelled {
		return fmt.Errorf("assessment %s is not restartable from status %s", assessment.ID, assessment.Status)
	}

	started := time.Now().UTC()
	if assessment.StartedAt == nil {
		assessment.StartedAt = &started
	}
	assessment.ErrorMessage = nil
	assessment.ErrorPhase = nil

	phases := []struct {
		Name   string
		Status model.CTEMAssessmentStatus
		Run    func(context.Context, *model.CTEMAssessment) error
	}{
		{Name: "scoping", Status: model.CTEMAssessmentStatusScoping, Run: e.runScopingPhase},
		{Name: "discovery", Status: model.CTEMAssessmentStatusDiscovery, Run: e.runDiscovery},
		{Name: "prioritizing", Status: model.CTEMAssessmentStatusPrioritizing, Run: e.runPrioritization},
		{Name: "validating", Status: model.CTEMAssessmentStatusValidating, Run: e.runValidation},
		{Name: "mobilizing", Status: model.CTEMAssessmentStatusMobilizing, Run: e.runMobilization},
	}

	for _, phase := range phases {
		select {
		case <-ctx.Done():
			return e.handleCancellation(context.Background(), assessment, ctx.Err())
		default:
		}

		progress := assessment.Phases[phase.Name]
		if progress.Status == model.CTEMPhaseStatusCompleted {
			continue
		}

		now := time.Now().UTC()
		progress.Status = model.CTEMPhaseStatusRunning
		progress.StartedAt = &now
		progress.Errors = nil
		assessment.Phases[phase.Name] = progress
		assessment.Status = phase.Status
		assessment.CurrentPhase = &phase.Name
		if err := e.assessmentRepo.SaveState(ctx, assessment); err != nil {
			return err
		}
		e.publishEvent(ctx, "cyber.ctem.phase.started", assessment.TenantID.String(), map[string]any{
			"assessment_id": assessment.ID.String(),
			"phase":         phase.Name,
		})

		if err := phase.Run(ctx, assessment); err != nil {
			if errors.Is(err, context.Canceled) {
				return e.handleCancellation(context.Background(), assessment, err)
			}
			errMessage := err.Error()
			progress = assessment.Phases[phase.Name]
			progress.Status = model.CTEMPhaseStatusFailed
			progress.Errors = []string{errMessage}
			assessment.Phases[phase.Name] = progress
			assessment.Status = model.CTEMAssessmentStatusFailed
			assessment.ErrorMessage = &errMessage
			errorPhase := phase.Name
			assessment.ErrorPhase = &errorPhase
			assessment.CurrentPhase = &phase.Name
			_ = e.assessmentRepo.SaveState(context.Background(), assessment)
			e.publishEvent(context.Background(), "cyber.ctem.assessment.failed", assessment.TenantID.String(), map[string]any{
				"id":    assessment.ID.String(),
				"phase": phase.Name,
				"error": errMessage,
			})
			return err
		}

		progress = assessment.Phases[phase.Name]
		completedAt := time.Now().UTC()
		progress.Status = model.CTEMPhaseStatusCompleted
		progress.CompletedAt = &completedAt
		assessment.Phases[phase.Name] = progress
		if err := e.assessmentRepo.SaveState(ctx, assessment); err != nil {
			return err
		}
		e.publishEvent(ctx, "cyber.ctem.phase.completed", assessment.TenantID.String(), map[string]any{
			"assessment_id": assessment.ID.String(),
			"phase":         phase.Name,
			"duration_ms":   durationBetween(progress.StartedAt, progress.CompletedAt),
			"items_processed": progress.ItemsProcessed,
		})
	}

	score, err := e.scoring.CalculateExposureScore(ctx, assessment.TenantID)
	if err != nil {
		return err
	}
	assessment.ExposureScore = &score.Score
	breakdownJSON, _ := json.Marshal(score.Breakdown)
	assessment.ScoreBreakdown = breakdownJSON
	summary, err := e.findingRepo.Summary(ctx, assessment.TenantID, assessment.ID)
	if err != nil {
		return err
	}
	summaryJSON, _ := json.Marshal(summary)
	assessment.FindingsSummary = summaryJSON
	assessment.Status = model.CTEMAssessmentStatusCompleted
	assessment.CurrentPhase = nil
	completedAt := time.Now().UTC()
	assessment.CompletedAt = &completedAt
	duration := completedAt.Sub(*assessment.StartedAt).Milliseconds()
	assessment.DurationMs = &duration
	if err := e.assessmentRepo.SaveState(ctx, assessment); err != nil {
		return err
	}

	assetCount, vulnCount, findingCount, err := e.scoreSnapshotCounts(ctx, assessment)
	if err != nil {
		return err
	}
	if err := e.snapshotRepo.Create(ctx, assessment.TenantID, &assessment.ID, "assessment", score, assetCount, vulnCount, findingCount); err != nil {
		return err
	}
	e.publishEvent(ctx, "cyber.ctem.assessment.completed", assessment.TenantID.String(), map[string]any{
		"id":             assessment.ID.String(),
		"exposure_score": score.Score,
		"finding_count":  findingCount,
		"duration_ms":    duration,
	})
	return nil
}

func (e *CTEMEngine) UpdatePhaseProgress(ctx context.Context, assessment *model.CTEMAssessment, phase string, processed, total int) error {
	progress := assessment.Phases[phase]
	progress.ItemsProcessed = processed
	progress.ItemsTotal = total
	assessment.Phases[phase] = progress
	if err := e.assessmentRepo.SaveState(ctx, assessment); err != nil {
		return err
	}
	e.publishEvent(ctx, "cyber.ctem.phase.progress", assessment.TenantID.String(), map[string]any{
		"assessment_id":  assessment.ID.String(),
		"phase":          phase,
		"items_processed": processed,
		"items_total":    total,
	})
	return nil
}

func (e *CTEMEngine) handleCancellation(ctx context.Context, assessment *model.CTEMAssessment, reason error) error {
	assessment.Status = model.CTEMAssessmentStatusCancelled
	assessment.CurrentPhase = nil
	if assessment.StartedAt != nil {
		duration := time.Since(*assessment.StartedAt).Milliseconds()
		assessment.DurationMs = &duration
	}
	if reason != nil {
		message := reason.Error()
		assessment.ErrorMessage = &message
	}
	if err := e.assessmentRepo.SaveState(ctx, assessment); err != nil {
		return err
	}
	e.publishEvent(ctx, "cyber.ctem.assessment.cancelled", assessment.TenantID.String(), map[string]any{
		"id": assessment.ID.String(),
	})
	return context.Canceled
}

func (e *CTEMEngine) publishEvent(ctx context.Context, eventType, tenantID string, data any) {
	if e.producer == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = e.producer.Publish(ctx, events.Topics.CtemEvents, events.NewEventRaw(eventType, "cyber-service", tenantID, payload))
}

func (e *CTEMEngine) scoreSnapshotCounts(ctx context.Context, assessment *model.CTEMAssessment) (int, int, int, error) {
	var assetCount int
	if err := e.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM assets
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		assessment.TenantID,
	).Scan(&assetCount); err != nil {
		return 0, 0, 0, err
	}

	var vulnCount int
	if err := e.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM vulnerabilities
		WHERE tenant_id = $1 AND deleted_at IS NULL AND status IN ('open','in_progress')`,
		assessment.TenantID,
	).Scan(&vulnCount); err != nil {
		return 0, 0, 0, err
	}

	var findingCount int
	if err := e.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM ctem_findings
		WHERE tenant_id = $1 AND assessment_id = $2`,
		assessment.TenantID, assessment.ID,
	).Scan(&findingCount); err != nil {
		return 0, 0, 0, err
	}
	return assetCount, vulnCount, findingCount, nil
}

func durationBetween(startedAt, completedAt *time.Time) int64 {
	if startedAt == nil || completedAt == nil {
		return 0
	}
	return completedAt.Sub(*startedAt).Milliseconds()
}
