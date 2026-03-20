package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/ctem"
	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

type CTEMService struct {
	db             repositoryQueryer
	assessmentRepo *repository.CTEMAssessmentRepository
	findingRepo    *repository.CTEMFindingRepository
	remGroupRepo   *repository.CTEMRemediationGroupRepository
	snapshotRepo   *repository.CTEMSnapshotRepository
	assetRepo      *repository.AssetRepository
	engine         *ctem.CTEMEngine
	scoring        *ctem.ScoringEngine
	producer       *events.Producer
	workflow       ctem.WorkflowLauncher
	logger         zerolog.Logger

	mu         sync.Mutex
	running    map[uuid.UUID]context.CancelFunc
	exportJobs map[string]exportJob
}

type repositoryQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type exportJob struct {
	ID        string
	Status    string
	Format    string
	Path      string
	CreatedAt time.Time
}

func NewCTEMService(
	db repositoryQueryer,
	assessmentRepo *repository.CTEMAssessmentRepository,
	findingRepo *repository.CTEMFindingRepository,
	remGroupRepo *repository.CTEMRemediationGroupRepository,
	snapshotRepo *repository.CTEMSnapshotRepository,
	assetRepo *repository.AssetRepository,
	engine *ctem.CTEMEngine,
	scoring *ctem.ScoringEngine,
	producer *events.Producer,
	workflow ctem.WorkflowLauncher,
	logger zerolog.Logger,
) *CTEMService {
	return &CTEMService{
		db:             db,
		assessmentRepo: assessmentRepo,
		findingRepo:    findingRepo,
		remGroupRepo:   remGroupRepo,
		snapshotRepo:   snapshotRepo,
		assetRepo:      assetRepo,
		engine:         engine,
		scoring:        scoring,
		producer:       producer,
		workflow:       workflow,
		logger:         logger.With().Str("service", "ctem").Logger(),
		running:        make(map[uuid.UUID]context.CancelFunc),
		exportJobs:     make(map[string]exportJob),
	}
}

func (s *CTEMService) CreateAssessment(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreateCTEMAssessmentRequest) (*model.CTEMAssessment, error) {
	assessment, err := s.assessmentRepo.Create(ctx, tenantID, userID, req)
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "cyber.ctem.assessment.created", tenantID.String(), map[string]any{
		"id":         assessment.ID.String(),
		"name":       assessment.Name,
		"scope":      assessment.Scope,
		"created_by": userID.String(),
	})
	if req.Start {
		if err := s.StartAssessment(ctx, tenantID, userID, assessment.ID); err != nil {
			return nil, err
		}
		return s.assessmentRepo.GetByID(ctx, tenantID, assessment.ID)
	}
	return assessment, nil
}

func (s *CTEMService) ListAssessments(ctx context.Context, tenantID uuid.UUID, params *dto.CTEMAssessmentListParams) (*dto.CTEMAssessmentListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.assessmentRepo.List(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	return &dto.CTEMAssessmentListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func (s *CTEMService) GetAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) (*model.CTEMAssessment, error) {
	return s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
}

func (s *CTEMService) UpdateAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID, req *dto.UpdateCTEMAssessmentRequest) (*model.CTEMAssessment, error) {
	return s.assessmentRepo.UpdateDefinition(ctx, tenantID, assessmentID, req)
}

func (s *CTEMService) StartAssessment(ctx context.Context, tenantID, userID, assessmentID uuid.UUID) error {
	assessment, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
	if err != nil {
		return err
	}
	s.publishEvent(ctx, "cyber.ctem.assessment.started", tenantID.String(), map[string]any{
		"id":                   assessmentID.String(),
		"resolved_asset_count": assessment.ResolvedAssetCount,
		"requested_by":         userID.String(),
	})
	if s.producer != nil {
		s.publishEvent(ctx, "cyber.ctem.assessment.run_requested", tenantID.String(), map[string]any{
			"assessment_id": assessmentID.String(),
		})
		return nil
	}
	return s.RunAssessmentAsyncFromEvent(assessmentID)
}

func (s *CTEMService) RunAssessmentAsyncFromEvent(assessmentID uuid.UUID) error {
	s.mu.Lock()
	if _, exists := s.running[assessmentID]; exists {
		s.mu.Unlock()
		return nil
	}
	runCtx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	s.running[assessmentID] = cancel
	s.mu.Unlock()

	go func() {
		defer s.unregisterAssessment(assessmentID)
		defer cancel()
		if err := s.engine.RunAssessment(runCtx, assessmentID); err != nil && !errors.Is(err, context.Canceled) {
			s.logger.Error().Err(err).Str("assessment_id", assessmentID.String()).Msg("ctem assessment execution failed")
		}
	}()
	return nil
}

func (s *CTEMService) CancelAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	s.mu.Lock()
	cancel := s.running[assessmentID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	assessment, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
	if err != nil {
		return err
	}
	assessment.Status = model.CTEMAssessmentStatusCancelled
	return s.assessmentRepo.SaveState(ctx, assessment)
}

func (s *CTEMService) DeleteAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	return s.assessmentRepo.SoftDelete(ctx, tenantID, assessmentID)
}

func (s *CTEMService) GetPhaseResult(ctx context.Context, tenantID, assessmentID uuid.UUID, phase string) (json.RawMessage, error) {
	assessment, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	progress, ok := assessment.Phases[phase]
	if !ok {
		return json.RawMessage("{}"), nil
	}
	if len(progress.Result) == 0 {
		return json.RawMessage("{}"), nil
	}
	return progress.Result, nil
}

func (s *CTEMService) ValidateAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID, req *dto.ValidateAssessmentRequest) error {
	_, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
	if err != nil {
		return err
	}
	if len(req.Findings) > 0 {
		findings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, assessmentID)
		if err != nil {
			return err
		}
		overrideMap := make(map[uuid.UUID]dto.ValidationFindingOverride)
		for _, override := range req.Findings {
			overrideMap[override.FindingID] = override
		}
		for _, finding := range findings {
			override, ok := overrideMap[finding.ID]
			if !ok {
				continue
			}
			finding.ValidationStatus = override.ValidationStatus
			finding.CompensatingControls = override.CompensatingControls
			finding.ValidationNotes = override.ValidationNotes
			now := time.Now().UTC()
			finding.ValidatedAt = &now
		}
		if err := s.findingRepo.SaveAnalysis(ctx, tenantID, assessmentID, findings); err != nil {
			return err
		}
	}
	return s.engine.RunValidationPhase(ctx, assessmentID)
}

func (s *CTEMService) MobilizeAssessment(ctx context.Context, tenantID, assessmentID uuid.UUID) error {
	if _, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID); err != nil {
		return err
	}
	return s.engine.RunMobilizationPhase(ctx, assessmentID)
}

func (s *CTEMService) ListFindings(ctx context.Context, tenantID, assessmentID uuid.UUID, params *dto.CTEMFindingsListParams) (*dto.CTEMFindingsListResponse, error) {
	params.SetDefaults()
	if err := params.Validate(); err != nil {
		return nil, err
	}
	items, total, err := s.findingRepo.ListByAssessment(ctx, tenantID, assessmentID, params)
	if err != nil {
		return nil, err
	}
	return &dto.CTEMFindingsListResponse{
		Data: items,
		Meta: dto.NewPaginationMeta(params.Page, params.PerPage, total),
	}, nil
}

func (s *CTEMService) GetFinding(ctx context.Context, tenantID, findingID uuid.UUID) (*model.CTEMFinding, error) {
	return s.findingRepo.GetByID(ctx, tenantID, findingID)
}

func (s *CTEMService) UpdateFindingStatus(ctx context.Context, tenantID, findingID, userID uuid.UUID, req *dto.UpdateCTEMFindingStatusRequest) (*model.CTEMFinding, error) {
	finding, err := s.findingRepo.UpdateStatus(ctx, tenantID, findingID, userID, req)
	if err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "cyber.ctem.finding.status_changed", tenantID.String(), map[string]any{
		"finding_id": finding.ID.String(),
		"new_status": finding.Status,
		"changed_by": userID.String(),
	})
	return finding, nil
}

func (s *CTEMService) ListRemediationGroups(ctx context.Context, tenantID, assessmentID uuid.UUID) ([]*model.CTEMRemediationGroup, error) {
	return s.remGroupRepo.ListByAssessment(ctx, tenantID, assessmentID)
}

func (s *CTEMService) GetRemediationGroup(ctx context.Context, tenantID, groupID uuid.UUID) (*model.CTEMRemediationGroup, []*model.CTEMFinding, error) {
	group, err := s.remGroupRepo.GetByID(ctx, tenantID, groupID)
	if err != nil {
		return nil, nil, err
	}
	findings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, group.AssessmentID)
	if err != nil {
		return nil, nil, err
	}
	filtered := make([]*model.CTEMFinding, 0)
	for _, finding := range findings {
		if finding.RemediationGroupID != nil && *finding.RemediationGroupID == groupID {
			filtered = append(filtered, finding)
		}
	}
	return group, filtered, nil
}

func (s *CTEMService) UpdateRemediationGroupStatus(ctx context.Context, tenantID, groupID uuid.UUID, req *dto.UpdateCTEMRemediationGroupStatusRequest) (*model.CTEMRemediationGroup, error) {
	return s.remGroupRepo.UpdateStatus(ctx, tenantID, groupID, req.Status)
}

func (s *CTEMService) ExecuteRemediationGroup(ctx context.Context, tenantID, userID, groupID uuid.UUID) (*model.CTEMRemediationGroup, error) {
	if s.workflow == nil {
		return nil, fmt.Errorf("workflow remediation execution is not configured")
	}
	group, err := s.remGroupRepo.GetByID(ctx, tenantID, groupID)
	if err != nil {
		return nil, err
	}
	if group.WorkflowInstanceID != nil && *group.WorkflowInstanceID != "" {
		return group, nil
	}
	assessment, err := s.assessmentRepo.GetByID(ctx, tenantID, group.AssessmentID)
	if err != nil {
		return nil, err
	}
	instanceID, err := s.workflow.StartRemediation(ctx, tenantID, userID, group, assessment)
	if err != nil {
		return nil, err
	}
	if err := s.remGroupRepo.UpdateWorkflowInstance(ctx, tenantID, groupID, instanceID); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "cyber.ctem.remediation.triggered", tenantID.String(), map[string]any{
		"group_id":             groupID.String(),
		"workflow_instance_id": instanceID,
	})
	return s.remGroupRepo.GetByID(ctx, tenantID, groupID)
}

func (s *CTEMService) BuildReport(ctx context.Context, tenantID, assessmentID uuid.UUID) (*model.CTEMReport, error) {
	assessment, err := s.assessmentRepo.GetByID(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	findings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	groups, err := s.remGroupRepo.ListByAssessment(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	score, err := s.scoring.CalculateExposureScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return ctem.AssembleReport(assessment, findings, groups, *score), nil
}

func (s *CTEMService) BuildExecutiveSummary(ctx context.Context, tenantID, assessmentID uuid.UUID) (map[string]any, error) {
	report, err := s.BuildReport(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"assessment_id":     report.Assessment.ID,
		"executive_summary": report.ExecutiveSummary,
		"exposure_score":    report.ExposureScore,
		"generated_at":      report.GeneratedAt,
	}, nil
}

func (s *CTEMService) ExportReport(ctx context.Context, tenantID, assessmentID uuid.UUID, req *dto.CTEMReportExportRequest) (*dto.CTEMReportExportResponse, error) {
	report, err := s.BuildReport(ctx, tenantID, assessmentID)
	if err != nil {
		return nil, err
	}
	jobID := uuid.NewString()
	job := exportJob{
		ID:        jobID,
		Status:    "queued",
		Format:    req.Format,
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.exportJobs[jobID] = job
	s.mu.Unlock()

	go func() {
		path, exportErr := writeReportExport(jobID, req.Format, report)
		s.mu.Lock()
		defer s.mu.Unlock()
		updated := s.exportJobs[jobID]
		if exportErr != nil {
			updated.Status = "failed"
		} else {
			updated.Status = "completed"
			updated.Path = path
		}
		s.exportJobs[jobID] = updated
	}()

	return &dto.CTEMReportExportResponse{JobID: jobID, Status: "queued"}, nil
}

func (s *CTEMService) Dashboard(ctx context.Context, tenantID uuid.UUID) (*model.CTEMDashboard, error) {
	score, err := s.scoring.CalculateExposureScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	history, err := s.snapshotRepo.History(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -90))
	if err != nil {
		return nil, err
	}
	latest, _ := s.assessmentRepo.LatestCompleted(ctx, tenantID)
	findingsByPriority, findingsBySeverity, findingsByType, findingsByStatus, topAssets, topPaths, remediationStats, compliance, err := s.dashboardFromLatestAssessment(ctx, tenantID, latest)
	if err != nil {
		return nil, err
	}
	remediationRate, mttr, err := s.remediationMetrics(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	active, lastDate, err := s.activeAssessmentStats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	return &model.CTEMDashboard{
		ExposureScore:           *score,
		ExposureScoreTrend:      history,
		FindingsByPriorityGroup: findingsByPriority,
		FindingsBySeverity:      findingsBySeverity,
		FindingsByType:          findingsByType,
		FindingsByStatus:        findingsByStatus,
		RemediationRate:         remediationRate,
		MeanTimeToRemediate:     mttr,
		TopExposedAssets:        topAssets,
		TopAttackPaths:          topPaths,
		ActiveAssessments:       active,
		LastAssessmentDate:      lastDate,
		RemediationGroupStats:   remediationStats,
		ComplianceSummary:       compliance,
	}, nil
}

func (s *CTEMService) CurrentExposureScore(ctx context.Context, tenantID uuid.UUID) (*model.ExposureScore, error) {
	return s.scoring.CalculateExposureScore(ctx, tenantID)
}

func (s *CTEMService) ExposureHistory(ctx context.Context, tenantID uuid.UUID, days int) ([]model.TimeSeriesPoint, error) {
	if days == 0 {
		days = 90
	}
	return s.snapshotRepo.History(ctx, tenantID, time.Now().UTC().AddDate(0, 0, -days))
}

func (s *CTEMService) ForceCalculateExposureScore(ctx context.Context, tenantID uuid.UUID) (*model.ExposureScore, error) {
	score, err := s.scoring.CalculateExposureScore(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	assetCount, vulnCount, findingCount, err := s.countExposureInputs(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if err := s.snapshotRepo.Create(ctx, tenantID, nil, "manual", score, assetCount, vulnCount, findingCount); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, "cyber.ctem.exposure_score.updated", tenantID.String(), map[string]any{
		"tenant_id": tenantID.String(),
		"new_score": score.Score,
	})
	return score, nil
}

func (s *CTEMService) CompareAssessments(ctx context.Context, tenantID, currentID, otherID uuid.UUID) (*model.CTEMAssessmentComparison, error) {
	current, err := s.assessmentRepo.GetByID(ctx, tenantID, currentID)
	if err != nil {
		return nil, err
	}
	previous, err := s.assessmentRepo.GetByID(ctx, tenantID, otherID)
	if err != nil {
		return nil, err
	}
	currentFindings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, currentID)
	if err != nil {
		return nil, err
	}
	previousFindings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, otherID)
	if err != nil {
		return nil, err
	}
	return compareAssessmentSets(current, previous, currentFindings, previousFindings), nil
}

func (s *CTEMService) unregisterAssessment(assessmentID uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.running, assessmentID)
}

func (s *CTEMService) publishEvent(ctx context.Context, eventType, tenantID string, data any) {
	if s.producer == nil {
		return
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_ = s.producer.Publish(ctx, events.Topics.CtemEvents, events.NewEventRaw(eventType, "cyber-service", tenantID, payload))
}

func (s *CTEMService) countExposureInputs(ctx context.Context, tenantID uuid.UUID) (int, int, int, error) {
	var assetCount, vulnCount, findingCount int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM assets WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID).Scan(&assetCount); err != nil {
		return 0, 0, 0, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM vulnerabilities WHERE tenant_id = $1 AND deleted_at IS NULL AND status IN ('open','in_progress')`, tenantID).Scan(&vulnCount); err != nil {
		return 0, 0, 0, err
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM ctem_findings WHERE tenant_id = $1 AND status = 'open'`, tenantID).Scan(&findingCount); err != nil {
		return 0, 0, 0, err
	}
	return assetCount, vulnCount, findingCount, nil
}

func (s *CTEMService) dashboardFromLatestAssessment(ctx context.Context, tenantID uuid.UUID, latest *model.CTEMAssessment) (map[int]int, map[string]int, map[string]int, map[string]int, []model.AssetExposureSummary, []model.AttackPathSummary, model.RemediationGroupStats, model.ComplianceSummary, error) {
	if latest == nil {
		return map[int]int{}, map[string]int{}, map[string]int{}, map[string]int{}, []model.AssetExposureSummary{}, []model.AttackPathSummary{}, model.RemediationGroupStats{ByStatus: map[string]int{}, ByType: map[string]int{}}, model.ComplianceSummary{}, nil
	}
	findings, err := s.findingRepo.ListAllByAssessment(ctx, tenantID, latest.ID)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, model.RemediationGroupStats{}, model.ComplianceSummary{}, err
	}
	groups, err := s.remGroupRepo.ListByAssessment(ctx, tenantID, latest.ID)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, model.RemediationGroupStats{}, model.ComplianceSummary{}, err
	}
	byPriority := map[int]int{}
	bySeverity := map[string]int{}
	byType := map[string]int{}
	byStatus := map[string]int{}
	for _, finding := range findings {
		byPriority[finding.PriorityGroup]++
		bySeverity[finding.Severity]++
		byType[string(finding.Type)]++
		byStatus[string(finding.Status)]++
	}
	topAssets := topAssetExposureSummaries(findings)
	topPaths := topAttackPathSummaries(findings)
	remStats := remediationGroupStats(groups)
	compliance := complianceSummary(findings, groups)
	return byPriority, bySeverity, byType, byStatus, topAssets, topPaths, remStats, compliance, nil
}

func (s *CTEMService) remediationMetrics(ctx context.Context, tenantID uuid.UUID) (float64, map[string]float64, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status = 'remediated' AND status_changed_at >= now() - interval '90 days')::float,
			COUNT(*) FILTER (WHERE created_at >= now() - interval '90 days')::float,
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0) FILTER (WHERE severity = 'critical' AND status = 'remediated' AND status_changed_at >= now() - interval '90 days'),
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0) FILTER (WHERE severity = 'high' AND status = 'remediated' AND status_changed_at >= now() - interval '90 days'),
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0) FILTER (WHERE severity = 'medium' AND status = 'remediated' AND status_changed_at >= now() - interval '90 days'),
			AVG(EXTRACT(EPOCH FROM (status_changed_at - created_at)) / 86400.0) FILTER (WHERE severity = 'low' AND status = 'remediated' AND status_changed_at >= now() - interval '90 days')
		FROM ctem_findings
		WHERE tenant_id = $1`,
		tenantID,
	)
	var remediated, total float64
	var critical, high, medium, low *float64
	if err := row.Scan(&remediated, &total, &critical, &high, &medium, &low); err != nil {
		return 0, nil, err
	}
	rate := 0.0
	if total > 0 {
		rate = (remediated / total) * 100
	}
	mttr := map[string]float64{
		"critical": derefFloat64(critical),
		"high":     derefFloat64(high),
		"medium":   derefFloat64(medium),
		"low":      derefFloat64(low),
	}
	return rate, mttr, nil
}

func (s *CTEMService) activeAssessmentStats(ctx context.Context, tenantID uuid.UUID) (int, *time.Time, error) {
	row := s.db.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE status IN ('scoping','discovery','prioritizing','validating','mobilizing'))::int,
			MAX(completed_at)
		FROM ctem_assessments
		WHERE tenant_id = $1 AND deleted_at IS NULL`,
		tenantID,
	)
	var active int
	var last *time.Time
	if err := row.Scan(&active, &last); err != nil {
		return 0, nil, err
	}
	return active, last, nil
}

func compareAssessmentSets(current, previous *model.CTEMAssessment, currentFindings, previousFindings []*model.CTEMFinding) *model.CTEMAssessmentComparison {
	currentMap := make(map[string]*model.CTEMFinding)
	previousMap := make(map[string]*model.CTEMFinding)
	for _, finding := range currentFindings {
		currentMap[matchingKey(finding)] = finding
	}
	for _, finding := range previousFindings {
		previousMap[matchingKey(finding)] = finding
	}
	newFindings := make([]model.FindingSummary, 0)
	resolvedFindings := make([]model.FindingSummary, 0)
	unchanged := 0
	worsened := 0
	for key, currentFinding := range currentMap {
		previousFinding, exists := previousMap[key]
		if !exists {
			newFindings = append(newFindings, summarizeFinding(currentFinding))
			continue
		}
		if currentFinding.Status == previousFinding.Status {
			unchanged++
		}
		if currentFinding.PriorityScore > previousFinding.PriorityScore || ctemSeverityRank(currentFinding.Severity) > ctemSeverityRank(previousFinding.Severity) {
			worsened++
		}
	}
	for key, previousFinding := range previousMap {
		if _, exists := currentMap[key]; !exists || currentMap[key].Status == model.CTEMFindingStatusRemediated {
			resolvedFindings = append(resolvedFindings, summarizeFinding(previousFinding))
		}
	}

	return &model.CTEMAssessmentComparison{
		Current: model.CTEMAssessmentComparisonSide{
			ID:            current.ID,
			Name:          current.Name,
			ExposureScore: current.ExposureScore,
			Findings:      summaryCounts(currentFindings),
		},
		Previous: model.CTEMAssessmentComparisonSide{
			ID:            previous.ID,
			Name:          previous.Name,
			ExposureScore: previous.ExposureScore,
			Findings:      summaryCounts(previousFindings),
		},
		Delta: model.CTEMAssessmentDelta{
			ScoreChange:       round2(derefFloat(current.ExposureScore) - derefFloat(previous.ExposureScore)),
			ScoreDirection:    scoreDirection(derefFloat(current.ExposureScore), derefFloat(previous.ExposureScore)),
			FindingsNew:       len(newFindings),
			FindingsResolved:  len(resolvedFindings),
			FindingsUnchanged: unchanged,
			FindingsWorsened:  worsened,
			NewFindings:       newFindings,
			ResolvedFindings:  resolvedFindings,
		},
	}
}

func topAssetExposureSummaries(findings []*model.CTEMFinding) []model.AssetExposureSummary {
	type accumulator struct {
		summary model.AssetExposureSummary
	}
	acc := make(map[uuid.UUID]*accumulator)
	for _, finding := range findings {
		if finding.PrimaryAssetID == nil {
			continue
		}
		entry := acc[*finding.PrimaryAssetID]
		if entry == nil {
			entry = &accumulator{summary: model.AssetExposureSummary{AssetID: *finding.PrimaryAssetID}}
			acc[*finding.PrimaryAssetID] = entry
		}
		entry.summary.FindingCount++
		if finding.PriorityScore > entry.summary.HighestScore {
			entry.summary.HighestScore = finding.PriorityScore
		}
		var evidence map[string]any
		_ = json.Unmarshal(finding.Evidence, &evidence)
		if name, ok := evidence["asset_name"].(string); ok {
			entry.summary.AssetName = name
		}
	}
	out := make([]model.AssetExposureSummary, 0, len(acc))
	for _, entry := range acc {
		out = append(out, entry.summary)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FindingCount == out[j].FindingCount {
			return out[i].HighestScore > out[j].HighestScore
		}
		return out[i].FindingCount > out[j].FindingCount
	})
	if len(out) > 10 {
		out = out[:10]
	}
	return out
}

func topAttackPathSummaries(findings []*model.CTEMFinding) []model.AttackPathSummary {
	out := make([]model.AttackPathSummary, 0)
	for _, finding := range findings {
		if finding.Type != model.CTEMFindingTypeAttackPath {
			continue
		}
		var hops []map[string]any
		_ = json.Unmarshal(finding.AttackPath, &hops)
		entry := model.AttackPathSummary{
			FindingID:  finding.ID,
			Title:      finding.Title,
			Score:      finding.PriorityScore,
			PathLength: derefInt(finding.AttackPathLength),
		}
		if len(hops) > 0 {
			if name, ok := hops[0]["asset_name"].(string); ok {
				entry.EntryAsset = name
			}
			if name, ok := hops[len(hops)-1]["asset_name"].(string); ok {
				entry.TargetAsset = name
			}
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if len(out) > 5 {
		out = out[:5]
	}
	return out
}

func remediationGroupStats(groups []*model.CTEMRemediationGroup) model.RemediationGroupStats {
	stats := model.RemediationGroupStats{ByStatus: map[string]int{}, ByType: map[string]int{}}
	for _, group := range groups {
		stats.Total++
		stats.ByStatus[string(group.Status)]++
		stats.ByType[string(group.Type)]++
		if group.Status == model.CTEMRemediationGroupInProgress {
			stats.InProgress++
		}
		if group.Status == model.CTEMRemediationGroupCompleted {
			stats.Completed++
		}
	}
	return stats
}

func complianceSummary(findings []*model.CTEMFinding, groups []*model.CTEMRemediationGroup) model.ComplianceSummary {
	validated := 0
	acceptedRisk := 0
	immediate := 0
	overdueGroups := 0
	for _, finding := range findings {
		if finding.ValidationStatus == model.CTEMValidationValidated || finding.ValidationStatus == model.CTEMValidationCompensated {
			validated++
		}
		if finding.Status == model.CTEMFindingStatusAcceptedRisk {
			acceptedRisk++
		}
		if finding.PriorityGroup == 1 && finding.Status == model.CTEMFindingStatusOpen {
			immediate++
		}
	}
	for _, group := range groups {
		if group.TargetDate != nil && group.TargetDate.Before(time.Now().UTC()) && group.Status != model.CTEMRemediationGroupCompleted {
			overdueGroups++
		}
	}
	percent := 0.0
	if len(findings) > 0 {
		percent = (float64(validated) / float64(len(findings))) * 100
	}
	return model.ComplianceSummary{
		ValidatedPercent:      percent,
		AcceptedRiskFindings:  acceptedRisk,
		ImmediateOpenFindings: immediate,
		OverdueGroups:         overdueGroups,
	}
}

func matchingKey(finding *model.CTEMFinding) string {
	asset := ""
	if finding.PrimaryAssetID != nil {
		asset = finding.PrimaryAssetID.String()
	}
	if len(finding.CVEIDs) > 0 {
		return strings.Join([]string{string(finding.Type), asset, finding.CVEIDs[0]}, "|")
	}
	return strings.Join([]string{string(finding.Type), asset, strings.ToLower(finding.Title)}, "|")
}

func summarizeFinding(finding *model.CTEMFinding) model.FindingSummary {
	return model.FindingSummary{
		ID:            finding.ID,
		Title:         finding.Title,
		Type:          string(finding.Type),
		Severity:      finding.Severity,
		PriorityScore: finding.PriorityScore,
	}
}

func summaryCounts(findings []*model.CTEMFinding) map[string]int {
	out := map[string]int{}
	for _, finding := range findings {
		out[finding.Severity]++
	}
	return out
}

func scoreDirection(current, previous float64) string {
	switch {
	case current < previous:
		return "improved"
	case current > previous:
		return "worsened"
	default:
		return "unchanged"
	}
}

func ctemSeverityRank(severity string) int {
	switch severity {
	case "critical":
		return 5
	case "high":
		return 4
	case "medium":
		return 3
	case "low":
		return 2
	default:
		return 1
	}
}

func derefFloat(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefFloat64(value *float64) float64 {
	if value == nil {
		return 0
	}
	return *value
}

func derefInt(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func writeReportExport(jobID, format string, report *model.CTEMReport) (string, error) {
	reportJSON, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	path := filepath.Join(os.TempDir(), fmt.Sprintf("ctem-report-%s.%s", jobID, format))
	switch format {
	case "pdf":
		return path, os.WriteFile(path, simplePDF(report.ExecutiveSummary, string(reportJSON)), 0o600)
	case "docx":
		return path, simpleDOCX(path, report.ExecutiveSummary, string(reportJSON))
	default:
		return "", fmt.Errorf("unsupported export format %q", format)
	}
}

func simplePDF(summary, body string) []byte {
	content := fmt.Sprintf("BT /F1 12 Tf 40 780 Td (%s) Tj T* (%s) Tj ET", escapePDFText(summary), escapePDFText(truncate(body, 400)))
	var buffer bytes.Buffer
	buffer.WriteString("%PDF-1.4\n")
	offsets := []int{}
	writeObj := func(obj string) {
		offsets = append(offsets, buffer.Len())
		buffer.WriteString(obj)
	}
	writeObj("1 0 obj << /Type /Catalog /Pages 2 0 R >> endobj\n")
	writeObj("2 0 obj << /Type /Pages /Kids [3 0 R] /Count 1 >> endobj\n")
	writeObj("3 0 obj << /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >> endobj\n")
	writeObj(fmt.Sprintf("4 0 obj << /Length %d >> stream\n%s\nendstream endobj\n", len(content), content))
	writeObj("5 0 obj << /Type /Font /Subtype /Type1 /BaseFont /Helvetica >> endobj\n")
	xrefStart := buffer.Len()
	buffer.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for _, offset := range offsets {
		buffer.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	buffer.WriteString(fmt.Sprintf("trailer << /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", xrefStart))
	return buffer.Bytes()
}

func simpleDOCX(path, summary, body string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := zip.NewWriter(file)
	defer writer.Close()

	writeFile := func(name, content string) error {
		entry, err := writer.Create(name)
		if err != nil {
			return err
		}
		_, err = entry.Write([]byte(content))
		return err
	}

	if err := writeFile("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`); err != nil {
		return err
	}
	if err := writeFile("_rels/.rels", `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`); err != nil {
		return err
	}
	document := struct {
		XMLName xml.Name `xml:"w:document"`
		XmlnsW  string   `xml:"xmlns:w,attr"`
		Body    struct {
			Paragraphs []struct {
				Run struct {
					Text string `xml:"w:t"`
				} `xml:"w:r"`
			} `xml:"w:p"`
		} `xml:"w:body"`
	}{XmlnsW: "http://schemas.openxmlformats.org/wordprocessingml/2006/main"}
	for _, text := range []string{summary, truncate(body, 4000)} {
		paragraph := struct {
			Run struct {
				Text string `xml:"w:t"`
			} `xml:"w:r"`
		}{}
		paragraph.Run.Text = text
		document.Body.Paragraphs = append(document.Body.Paragraphs, paragraph)
	}
	payload, err := xml.Marshal(document)
	if err != nil {
		return err
	}
	return writeFile("word/document.xml", xml.Header+string(payload))
}

func escapePDFText(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\n", " ")
	return replacer.Replace(value)
}

func truncate(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[:length]
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
