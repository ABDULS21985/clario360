package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	cybermodel "github.com/clario360/platform/internal/cyber/model"

	"github.com/clario360/platform/internal/cyber/dspm/remediation/dto"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/exception"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/integration"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/lifecycle"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/model"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/playbook"
	"github.com/clario360/platform/internal/cyber/dspm/remediation/policy"
)

// ---------------------------------------------------------------------------
// Mock: in-memory RemediationRepository
// ---------------------------------------------------------------------------

// mockRemediationRepo is an in-memory implementation that mirrors the methods
// called by RemediationEngine on *repository.RemediationRepository.
type mockRemediationRepo struct {
	mu           sync.RWMutex
	remediations map[uuid.UUID]*model.Remediation
}

func newMockRemediationRepo() *mockRemediationRepo {
	return &mockRemediationRepo{
		remediations: make(map[uuid.UUID]*model.Remediation),
	}
}

func (m *mockRemediationRepo) Create(_ context.Context, rem *model.Remediation) (*model.Remediation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rem.ID == uuid.Nil {
		rem.ID = uuid.New()
	}
	now := time.Now().UTC()
	rem.CreatedAt = now
	rem.UpdatedAt = now
	if len(rem.Steps) == 0 {
		rem.Steps = json.RawMessage("[]")
	}
	if len(rem.ComplianceTags) == 0 {
		rem.ComplianceTags = json.RawMessage("[]")
	}
	if len(rem.PreActionState) == 0 {
		rem.PreActionState = json.RawMessage("{}")
	}
	cp := *rem
	m.remediations[cp.ID] = &cp
	return &cp, nil
}

func (m *mockRemediationRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*model.Remediation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rem, ok := m.remediations[id]
	if !ok || rem.TenantID != tenantID {
		return nil, fmt.Errorf("remediation not found")
	}
	cp := *rem
	return &cp, nil
}

func (m *mockRemediationRepo) Update(_ context.Context, rem *model.Remediation) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.remediations[rem.ID]
	if !ok {
		return fmt.Errorf("remediation not found")
	}
	rem.UpdatedAt = time.Now().UTC()
	cp := *rem
	m.remediations[cp.ID] = &cp
	return nil
}

func (m *mockRemediationRepo) UpdateStatus(_ context.Context, tenantID, id uuid.UUID, status model.RemediationStatus) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rem, ok := m.remediations[id]
	if !ok || rem.TenantID != tenantID {
		return fmt.Errorf("remediation not found")
	}
	rem.Status = status
	rem.UpdatedAt = time.Now().UTC()
	if status.IsTerminal() {
		now := time.Now().UTC()
		rem.CompletedAt = &now
	}
	return nil
}

func (m *mockRemediationRepo) UpdateSteps(_ context.Context, tenantID, id uuid.UUID, steps json.RawMessage, currentStep int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rem, ok := m.remediations[id]
	if !ok || rem.TenantID != tenantID {
		return fmt.Errorf("remediation not found")
	}
	rem.Steps = steps
	rem.CurrentStep = currentStep
	rem.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockRemediationRepo) List(_ context.Context, tenantID uuid.UUID, params *dto.RemediationListParams) ([]model.Remediation, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []model.Remediation
	for _, rem := range m.remediations {
		if rem.TenantID != tenantID {
			continue
		}
		results = append(results, *rem)
	}
	total := len(results)
	// simple pagination
	params.SetDefaults()
	start := (params.Page - 1) * params.PerPage
	if start >= total {
		return []model.Remediation{}, total, nil
	}
	end := start + params.PerPage
	if end > total {
		end = total
	}
	return results[start:end], total, nil
}

func (m *mockRemediationRepo) Stats(_ context.Context, tenantID uuid.UUID) (*model.RemediationStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	stats := &model.RemediationStats{
		ByStatus:      make(map[string]int),
		BySeverity:    make(map[string]int),
		ByFindingType: make(map[string]int),
	}
	for _, rem := range m.remediations {
		if rem.TenantID != tenantID {
			continue
		}
		stats.ByStatus[string(rem.Status)]++
		stats.BySeverity[rem.Severity]++
		stats.ByFindingType[string(rem.FindingType)]++
		switch rem.Status {
		case model.StatusOpen, model.StatusInProgress, model.StatusAwaitingApproval:
			stats.TotalOpen++
			if rem.Severity == "critical" {
				stats.TotalCriticalOpen++
			}
		}
		if rem.Status == model.StatusInProgress {
			stats.TotalInProgress++
		}
		if rem.SLABreached {
			stats.SLABreaches++
		}
	}
	return stats, nil
}

func (m *mockRemediationRepo) FindSLABreached(_ context.Context, tenantID uuid.UUID) ([]model.Remediation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().UTC()
	var results []model.Remediation
	for _, rem := range m.remediations {
		if rem.TenantID != tenantID {
			continue
		}
		if rem.SLABreached {
			continue
		}
		if rem.Status.IsTerminal() {
			continue
		}
		if rem.SLADueAt != nil && rem.SLADueAt.Before(now) {
			results = append(results, *rem)
		}
	}
	return results, nil
}

func (m *mockRemediationRepo) MarkSLABreached(_ context.Context, tenantID, id uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	rem, ok := m.remediations[id]
	if !ok || rem.TenantID != tenantID {
		return fmt.Errorf("remediation not found")
	}
	rem.SLABreached = true
	rem.UpdatedAt = time.Now().UTC()
	return nil
}

func (m *mockRemediationRepo) BurndownData(_ context.Context, _ uuid.UUID, _ int) ([]model.BurndownDataPoint, error) {
	return []model.BurndownDataPoint{}, nil
}

// count returns the total number of remediations stored.
func (m *mockRemediationRepo) count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.remediations)
}

// getAll returns all stored remediations.
func (m *mockRemediationRepo) getAll() []*model.Remediation {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var all []*model.Remediation
	for _, rem := range m.remediations {
		cp := *rem
		all = append(all, &cp)
	}
	return all
}

// ---------------------------------------------------------------------------
// Mock: in-memory HistoryRepository
// ---------------------------------------------------------------------------

type mockHistoryRepo struct {
	mu      sync.RWMutex
	entries []*model.RemediationHistory
}

func newMockHistoryRepo() *mockHistoryRepo {
	return &mockHistoryRepo{
		entries: make([]*model.RemediationHistory, 0),
	}
}

func (m *mockHistoryRepo) Insert(_ context.Context, entry *model.RemediationHistory) (*model.RemediationHistory, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	entry.CreatedAt = time.Now().UTC()
	if len(entry.Details) == 0 {
		entry.Details = json.RawMessage("{}")
	}
	cp := *entry
	m.entries = append(m.entries, &cp)
	return &cp, nil
}

func (m *mockHistoryRepo) ListByRemediation(_ context.Context, tenantID, remediationID uuid.UUID, page, perPage int) ([]model.RemediationHistory, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var matching []model.RemediationHistory
	for _, e := range m.entries {
		if e.TenantID == tenantID && e.RemediationID == remediationID {
			matching = append(matching, *e)
		}
	}
	total := len(matching)
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 || perPage > 200 {
		perPage = 50
	}
	start := (page - 1) * perPage
	if start >= total {
		return []model.RemediationHistory{}, total, nil
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return matching[start:end], total, nil
}

func (m *mockHistoryRepo) GetLastEntry(_ context.Context, tenantID, remediationID uuid.UUID) (*model.RemediationHistory, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var last *model.RemediationHistory
	for _, e := range m.entries {
		if e.TenantID == tenantID && e.RemediationID == remediationID {
			if last == nil || e.CreatedAt.After(last.CreatedAt) {
				cp := *e
				last = &cp
			}
		}
	}
	if last == nil {
		return nil, fmt.Errorf("remediation history not found")
	}
	return last, nil
}

// countForRemediation returns the number of history entries for a specific remediation.
func (m *mockHistoryRepo) countForRemediation(remediationID uuid.UUID) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	count := 0
	for _, e := range m.entries {
		if e.RemediationID == remediationID {
			count++
		}
	}
	return count
}

// findByAction returns history entries matching a specific action for a remediation.
func (m *mockHistoryRepo) findByAction(remediationID uuid.UUID, action model.HistoryAction) []*model.RemediationHistory {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []*model.RemediationHistory
	for _, e := range m.entries {
		if e.RemediationID == remediationID && e.Action == action {
			cp := *e
			results = append(results, &cp)
		}
	}
	return results
}

// ---------------------------------------------------------------------------
// Mock: in-memory PolicyRepository (for PolicyRepo interface used by engine)
// ---------------------------------------------------------------------------

type mockPolicyRepo struct {
	mu       sync.RWMutex
	policies map[uuid.UUID]*model.DataPolicy
}

func newMockPolicyRepo() *mockPolicyRepo {
	return &mockPolicyRepo{
		policies: make(map[uuid.UUID]*model.DataPolicy),
	}
}

func (m *mockPolicyRepo) ListEnabled(_ context.Context, tenantID uuid.UUID) ([]model.DataPolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []model.DataPolicy
	for _, p := range m.policies {
		if p.TenantID == tenantID && p.Enabled {
			results = append(results, *p)
		}
	}
	return results, nil
}

func (m *mockPolicyRepo) UpdateEvaluationResults(_ context.Context, tenantID, id uuid.UUID, violationCount int) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.policies[id]
	if !ok || p.TenantID != tenantID {
		return fmt.Errorf("data policy not found")
	}
	now := time.Now().UTC()
	p.LastEvaluatedAt = &now
	p.ViolationCount = violationCount
	p.UpdatedAt = now
	return nil
}

// addPolicy is a test helper that inserts a policy.
func (m *mockPolicyRepo) addPolicy(p *model.DataPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	cp := *p
	m.policies[cp.ID] = &cp
}

// ---------------------------------------------------------------------------
// Mock: in-memory ExceptionRepository (satisfies exception.ExceptionRepository)
// ---------------------------------------------------------------------------

type mockExceptionRepo struct {
	mu         sync.RWMutex
	exceptions map[uuid.UUID]*model.RiskException
}

func newMockExceptionRepo() *mockExceptionRepo {
	return &mockExceptionRepo{
		exceptions: make(map[uuid.UUID]*model.RiskException),
	}
}

func (m *mockExceptionRepo) Create(_ context.Context, exc *model.RiskException) (*model.RiskException, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if exc.ID == uuid.Nil {
		exc.ID = uuid.New()
	}
	now := time.Now().UTC()
	exc.CreatedAt = now
	exc.UpdatedAt = now
	cp := *exc
	m.exceptions[cp.ID] = &cp
	return &cp, nil
}

func (m *mockExceptionRepo) GetByID(_ context.Context, tenantID, id uuid.UUID) (*model.RiskException, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	exc, ok := m.exceptions[id]
	if !ok || exc.TenantID != tenantID {
		return nil, fmt.Errorf("risk exception not found")
	}
	cp := *exc
	return &cp, nil
}

func (m *mockExceptionRepo) Update(_ context.Context, exc *model.RiskException) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.exceptions[exc.ID]
	if !ok {
		return fmt.Errorf("risk exception not found")
	}
	exc.UpdatedAt = time.Now().UTC()
	cp := *exc
	m.exceptions[cp.ID] = &cp
	return nil
}

func (m *mockExceptionRepo) ListByTenant(_ context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var results []model.RiskException
	for _, exc := range m.exceptions {
		if exc.TenantID == tenantID {
			results = append(results, *exc)
		}
	}
	return results, nil
}

func (m *mockExceptionRepo) FindExpired(_ context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().UTC()
	var results []model.RiskException
	for _, exc := range m.exceptions {
		if exc.TenantID == tenantID && exc.Status == model.ExceptionStatusActive && exc.ExpiresAt.Before(now) {
			results = append(results, *exc)
		}
	}
	return results, nil
}

func (m *mockExceptionRepo) FindNeedingReview(_ context.Context, tenantID uuid.UUID) ([]model.RiskException, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	now := time.Now().UTC()
	var results []model.RiskException
	for _, exc := range m.exceptions {
		if exc.TenantID == tenantID && exc.Status == model.ExceptionStatusActive && exc.NextReviewAt != nil && exc.NextReviewAt.Before(now) {
			results = append(results, *exc)
		}
	}
	return results, nil
}

func (m *mockExceptionRepo) HasActiveException(_ context.Context, tenantID, assetID, policyID uuid.UUID) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, exc := range m.exceptions {
		if exc.TenantID != tenantID {
			continue
		}
		if exc.Status != model.ExceptionStatusActive {
			continue
		}
		if exc.ApprovalStatus != model.ApprovalApproved {
			continue
		}
		if exc.DataAssetID != nil && *exc.DataAssetID == assetID {
			if exc.PolicyID == nil || *exc.PolicyID == policyID {
				return true, nil
			}
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// Mock: AssetLister (satisfies policy.AssetLister and lifecycle.AssetLister)
// ---------------------------------------------------------------------------

type mockAssetLister struct {
	assets []*cybermodel.DSPMDataAsset
}

func newMockAssetLister(assets []*cybermodel.DSPMDataAsset) *mockAssetLister {
	return &mockAssetLister{assets: assets}
}

func (m *mockAssetLister) ListAllActive(_ context.Context, _ uuid.UUID) ([]*cybermodel.DSPMDataAsset, error) {
	return m.assets, nil
}

// ---------------------------------------------------------------------------
// Mock: RemediationUpdater (satisfies exception.RemediationUpdater)
// Delegates to mockRemediationRepo.UpdateStatus
// ---------------------------------------------------------------------------

type mockRemediationUpdater struct {
	repo *mockRemediationRepo
}

func (m *mockRemediationUpdater) UpdateStatus(ctx context.Context, tenantID, remediationID uuid.UUID, status model.RemediationStatus) error {
	return m.repo.UpdateStatus(ctx, tenantID, remediationID, status)
}

// ---------------------------------------------------------------------------
// testHarness holds all mock dependencies and the fully wired real components.
// Since the engine's repository fields are concrete struct pointers (not interfaces),
// we test the engine's logic indirectly by constructing and testing the real
// components (PolicyEngine, ExceptionManager, ExpiryChecker, PlaybookExecutor)
// with in-memory mock dependencies. This exercises the same code paths that the
// engine orchestrates, but without requiring a PostgreSQL database.
// ---------------------------------------------------------------------------

type testHarness struct {
	// Mock repositories
	remRepo       *mockRemediationRepo
	histRepo      *mockHistoryRepo
	policyRepo    *mockPolicyRepo
	exceptionRepo *mockExceptionRepo
	assetLister   *mockAssetLister

	// Real components wired with mocks
	registry         *playbook.Registry
	executor         *playbook.PlaybookExecutor
	validator        *playbook.Validator
	policyEngine     *policy.PolicyEngine
	policyEnforcer   *policy.Enforcer
	exceptionMgr     *exception.ExceptionManager
	expiryChecker    *exception.ExpiryChecker
	retentionEnforcer *lifecycle.RetentionEnforcer
	staleDetector    *lifecycle.StaleDataDetector
	siemExporter     *integration.SIEMExporter
	itsmConnector    *integration.ITSMConnector
	dlpGenerator     *integration.DLPPolicyGenerator

	// Shared identifiers
	tenantID uuid.UUID
	userID   uuid.UUID

	logger zerolog.Logger
}

func newTestHarness(assets []*cybermodel.DSPMDataAsset) *testHarness {
	logger := zerolog.Nop()
	tenantID := uuid.New()
	userID := uuid.New()

	remRepo := newMockRemediationRepo()
	histRepo := newMockHistoryRepo()
	policyRepoMock := newMockPolicyRepo()
	exceptionRepoMock := newMockExceptionRepo()
	assetLister := newMockAssetLister(assets)

	remUpdater := &mockRemediationUpdater{repo: remRepo}

	registry := playbook.NewRegistry()
	executor := playbook.NewPlaybookExecutor(registry, logger)
	validator := playbook.NewValidator(registry, logger)
	policyEng := policy.NewPolicyEngine(assetLister, exceptionRepoMock, logger)
	enforcer := policy.NewEnforcer(logger)
	excMgr := exception.NewExceptionManager(exceptionRepoMock, remUpdater, logger)
	expChecker := exception.NewExpiryChecker(exceptionRepoMock, remUpdater, logger)
	retEnforcer := lifecycle.NewRetentionEnforcer(assetLister, logger)
	staleDet := lifecycle.NewStaleDataDetector(assetLister, logger)
	siemExp := integration.NewSIEMExporter(logger)
	itsmConn := integration.NewITSMConnector(logger)
	dlpGen := integration.NewDLPPolicyGenerator(logger)

	return &testHarness{
		remRepo:           remRepo,
		histRepo:          histRepo,
		policyRepo:        policyRepoMock,
		exceptionRepo:     exceptionRepoMock,
		assetLister:       assetLister,
		registry:          registry,
		executor:          executor,
		validator:         validator,
		policyEngine:      policyEng,
		policyEnforcer:    enforcer,
		exceptionMgr:      excMgr,
		expiryChecker:     expChecker,
		retentionEnforcer: retEnforcer,
		staleDetector:     staleDet,
		siemExporter:      siemExp,
		itsmConnector:     itsmConn,
		dlpGenerator:      dlpGen,
		tenantID:          tenantID,
		userID:            userID,
		logger:            logger,
	}
}

// createRemediation is a helper that simulates engine.CreateRemediation by using
// the mock repos and real playbook registry, following the same logic as the engine.
func (h *testHarness) createRemediation(t *testing.T, req *dto.CreateRemediationRequest) *model.Remediation {
	t.Helper()
	ctx := context.Background()

	require.NoError(t, req.Validate(), "request validation should pass")

	pb, ok := h.registry.Get(req.PlaybookID)
	require.True(t, ok, "playbook %q should exist in registry", req.PlaybookID)

	remSteps := make([]model.RemediationStep, len(pb.Steps))
	for i, step := range pb.Steps {
		remSteps[i] = model.RemediationStep{
			StepID:      step.ID,
			Order:       step.Order,
			Action:      string(step.Action),
			Description: step.Description,
			Params:      step.Params,
			Status:      "pending",
		}
	}

	stepsJSON, err := json.Marshal(remSteps)
	require.NoError(t, err)

	now := time.Now().UTC()
	slaDue := dto.SLADueAt(now, req.Severity)

	var complianceJSON json.RawMessage
	if len(req.ComplianceTags) > 0 {
		complianceJSON, _ = json.Marshal(req.ComplianceTags)
	} else {
		complianceJSON = json.RawMessage("[]")
	}

	initialStatus := model.StatusOpen
	if pb.RequiresApproval {
		initialStatus = model.StatusAwaitingApproval
	}

	rem := &model.Remediation{
		ID:                uuid.New(),
		TenantID:          h.tenantID,
		FindingType:       model.FindingType(req.FindingType),
		FindingID:         req.FindingID,
		DataAssetID:       req.DataAssetID,
		DataAssetName:     req.DataAssetName,
		IdentityID:        req.IdentityID,
		PlaybookID:        req.PlaybookID,
		Title:             req.Title,
		Description:       req.Description,
		Severity:          req.Severity,
		Steps:             stepsJSON,
		CurrentStep:       0,
		TotalSteps:        len(pb.Steps),
		AssignedTo:        req.AssignedTo,
		AssignedTeam:      req.AssignedTeam,
		SLADueAt:          &slaDue,
		SLABreached:       false,
		RollbackAvailable: pb.AutoRollback,
		RolledBack:        false,
		Status:            initialStatus,
		CreatedBy:         &h.userID,
		ComplianceTags:    complianceJSON,
	}

	created, err := h.remRepo.Create(ctx, rem)
	require.NoError(t, err)

	// Record "created" history entry.
	h.recordHistory(t, created.ID, model.HistoryActionCreated, &h.userID, model.ActorTypeUser, map[string]interface{}{
		"playbook_id":    req.PlaybookID,
		"finding_type":   req.FindingType,
		"severity":       req.Severity,
		"total_steps":    len(pb.Steps),
		"initial_status": string(initialStatus),
	})

	return created
}

// executeStep simulates engine.ExecuteStep using mock repos and the real playbook executor.
func (h *testHarness) executeStep(t *testing.T, remediationID uuid.UUID, actorID *uuid.UUID) *model.StepResult {
	t.Helper()
	ctx := context.Background()

	rem, err := h.remRepo.GetByID(ctx, h.tenantID, remediationID)
	require.NoError(t, err)

	require.True(t, rem.Status == model.StatusOpen || rem.Status == model.StatusInProgress,
		"remediation status must be open or in_progress, got %q", rem.Status)
	require.Less(t, rem.CurrentStep, rem.TotalSteps, "all steps already completed")

	pb, ok := h.registry.Get(rem.PlaybookID)
	require.True(t, ok)

	stepResult, err := h.executor.Execute(ctx, pb, rem.CurrentStep)
	require.NoError(t, err)

	// Update steps in the remediation.
	var remSteps []model.RemediationStep
	require.NoError(t, json.Unmarshal(rem.Steps, &remSteps))

	if rem.CurrentStep < len(remSteps) {
		remSteps[rem.CurrentStep].Status = string(stepResult.Status)
		remSteps[rem.CurrentStep].StartedAt = &stepResult.StartedAt
		remSteps[rem.CurrentStep].CompletedAt = stepResult.CompletedAt
		remSteps[rem.CurrentStep].Result = stepResult.Result
		if stepResult.Error != "" {
			remSteps[rem.CurrentStep].Error = stepResult.Error
		}
	}

	updatedStepsJSON, err := json.Marshal(remSteps)
	require.NoError(t, err)
	rem.Steps = updatedStepsJSON

	// Record step history.
	histAction := model.HistoryActionStepCompleted
	if stepResult.Status == model.StepStatusFailed {
		histAction = model.HistoryActionStepFailed
	}
	h.recordHistory(t, rem.ID, histAction, actorID, model.ActorTypeUser, map[string]interface{}{
		"step_id":     stepResult.StepID,
		"step_index":  rem.CurrentStep,
		"action":      stepResult.Action,
		"status":      string(stepResult.Status),
		"duration_ms": stepResult.DurationMs,
	})

	// Advance step on success.
	if stepResult.Status == model.StepStatusCompleted {
		rem.CurrentStep++
	}

	// Transition open -> in_progress.
	if rem.Status == model.StatusOpen && rem.CurrentStep > 0 {
		rem.Status = model.StatusInProgress
		h.recordHistory(t, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeSystem, map[string]interface{}{
			"from_status": string(model.StatusOpen),
			"to_status":   string(model.StatusInProgress),
		})
	}

	// Check if all steps completed.
	if rem.CurrentStep >= rem.TotalSteps && rem.Status != model.StatusFailed {
		rem.Status = model.StatusCompleted
		now := time.Now().UTC()
		rem.CompletedAt = &now
		h.recordHistory(t, rem.ID, model.HistoryActionStatusChanged, actorID, model.ActorTypeSystem, map[string]interface{}{
			"from_status": string(model.StatusInProgress),
			"to_status":   string(model.StatusCompleted),
			"total_steps": rem.TotalSteps,
		})
	}

	err = h.remRepo.Update(ctx, rem)
	require.NoError(t, err)

	return stepResult
}

// recordHistory appends a tamper-evident history entry.
func (h *testHarness) recordHistory(t *testing.T, remediationID uuid.UUID, action model.HistoryAction, actorID *uuid.UUID, actorType model.ActorType, details interface{}) {
	t.Helper()
	ctx := context.Background()

	var detailsJSON json.RawMessage
	if details != nil {
		d, err := json.Marshal(details)
		require.NoError(t, err)
		detailsJSON = d
	} else {
		detailsJSON = json.RawMessage("{}")
	}

	var prevHash string
	lastEntry, err := h.histRepo.GetLastEntry(ctx, h.tenantID, remediationID)
	if err == nil && lastEntry != nil {
		prevHash = lastEntry.EntryHash
	}

	now := time.Now().UTC()
	entryHash := model.ComputeEntryHash(prevHash, action, detailsJSON, now)

	entry := &model.RemediationHistory{
		ID:            uuid.New(),
		TenantID:      h.tenantID,
		RemediationID: remediationID,
		Action:        action,
		ActorID:       actorID,
		ActorType:     actorType,
		Details:       detailsJSON,
		EntryHash:     entryHash,
		PrevHash:      prevHash,
		CreatedAt:     now,
	}

	_, err = h.histRepo.Insert(ctx, entry)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// Test: Full Remediation Cycle
// ---------------------------------------------------------------------------

func TestFullRemediationCycle(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// Create a remediation using the "encrypt-sensitive-data" playbook (5 steps).
	assetID := uuid.New()
	req := &dto.CreateRemediationRequest{
		FindingType:   "encryption_missing",
		DataAssetID:   &assetID,
		DataAssetName: "customer-pii-db",
		PlaybookID:    "encrypt-sensitive-data",
		Title:         "Encrypt Customer PII Database",
		Description:   "Apply encryption to the unencrypted customer PII database",
		Severity:      "high",
		AssignedTeam:  "security-ops",
	}

	rem := h.createRemediation(t, req)

	// Verify initial state.
	assert.Equal(t, model.StatusOpen, rem.Status, "initial status should be open")
	assert.Equal(t, 0, rem.CurrentStep, "initial current step should be 0")
	assert.Equal(t, 5, rem.TotalSteps, "encrypt-sensitive-data playbook has 5 steps")
	assert.NotNil(t, rem.SLADueAt, "SLA due at should be set")
	assert.False(t, rem.SLABreached, "SLA should not be breached initially")
	assert.False(t, rem.RollbackAvailable, "encrypt-sensitive-data does not support rollback")

	// Verify initial history entry.
	assert.Equal(t, 1, h.histRepo.countForRemediation(rem.ID), "should have 1 history entry after creation")
	createdEntries := h.histRepo.findByAction(rem.ID, model.HistoryActionCreated)
	require.Len(t, createdEntries, 1, "should have exactly one 'created' entry")

	// Execute all 5 steps sequentially.
	for i := 0; i < 5; i++ {
		stepResult := h.executeStep(t, rem.ID, &h.userID)
		assert.Equal(t, model.StepStatusCompleted, stepResult.Status,
			"step %d should complete successfully", i)
		assert.NotEmpty(t, stepResult.StepID, "step result should have a step ID")
		assert.NotEmpty(t, stepResult.Action, "step result should have an action")
	}

	// Reload the remediation and verify final state.
	finalRem, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)

	assert.Equal(t, model.StatusCompleted, finalRem.Status, "final status should be completed")
	assert.Equal(t, 5, finalRem.CurrentStep, "current step should be 5 (all done)")
	assert.NotNil(t, finalRem.CompletedAt, "completed_at should be set")

	// Verify step statuses in the steps JSON.
	var finalSteps []model.RemediationStep
	require.NoError(t, json.Unmarshal(finalRem.Steps, &finalSteps))
	for i, step := range finalSteps {
		assert.Equal(t, "completed", step.Status, "step %d should be completed", i)
		assert.NotNil(t, step.StartedAt, "step %d should have started_at", i)
		assert.NotNil(t, step.CompletedAt, "step %d should have completed_at", i)
	}

	// Verify history entries.
	// Expected: 1 created + 5 step_completed + 1 status_changed(open->in_progress) + 1 status_changed(in_progress->completed) = 8
	historyCount := h.histRepo.countForRemediation(rem.ID)
	assert.Equal(t, 8, historyCount, "should have 8 history entries total")

	stepCompletedEntries := h.histRepo.findByAction(rem.ID, model.HistoryActionStepCompleted)
	assert.Len(t, stepCompletedEntries, 5, "should have 5 step_completed entries")

	statusChangedEntries := h.histRepo.findByAction(rem.ID, model.HistoryActionStatusChanged)
	assert.Len(t, statusChangedEntries, 2, "should have 2 status_changed entries")

	// Verify the history chain is retrievable.
	allHistory, total, err := h.histRepo.ListByRemediation(ctx, h.tenantID, rem.ID, 1, 50)
	require.NoError(t, err)
	assert.Equal(t, 8, total)
	assert.Len(t, allHistory, 8)
}

// ---------------------------------------------------------------------------
// Test: Policy Auto-Remediation
// ---------------------------------------------------------------------------

func TestPolicyAutoRemediation(t *testing.T) {
	encryptedFalse := false
	internetFacing := "internet_facing"

	// Create assets that violate an encryption policy.
	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			TenantID:           uuid.New(), // will be overridden by evaluation context
			AssetName:          "unencrypted-customer-db",
			AssetType:          "database",
			DataClassification: "confidential",
			EncryptedAtRest:    &encryptedFalse,
			EncryptedInTransit: &encryptedFalse,
			NetworkExposure:    &internetFacing,
			ContainsPII:        true,
			PIITypes:           []string{"email", "phone"},
			CreatedAt:          time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			ID:                 uuid.New(),
			TenantID:           uuid.New(),
			AssetName:          "analytics-warehouse",
			AssetType:          "data_warehouse",
			DataClassification: "internal",
			EncryptedAtRest:    &encryptedFalse,
			EncryptedInTransit: &encryptedFalse,
			CreatedAt:          time.Now().Add(-60 * 24 * time.Hour),
		},
	}

	h := newTestHarness(assets)
	ctx := context.Background()

	// Create an encryption policy with auto_remediate enforcement.
	encRule := model.EncryptionRule{
		RequireAtRest:     true,
		RequireInTransit:  true,
		ClassificationMin: "internal",
	}
	ruleJSON, err := json.Marshal(encRule)
	require.NoError(t, err)

	pol := &model.DataPolicy{
		ID:              uuid.New(),
		TenantID:        h.tenantID,
		Name:            "Require Encryption for Internal+ Data",
		Description:     "All internal and above classified data must be encrypted at rest and in transit",
		Category:        model.PolicyCategoryEncryption,
		Rule:            ruleJSON,
		Enforcement:     model.EnforcementAutoRemediate,
		AutoPlaybookID:  "encrypt-sensitive-data",
		Severity:        "high",
		Enabled:         true,
		ComplianceFrameworks: []string{"SOC2", "GDPR"},
		ScopeClassification:  []string{},
		ScopeAssetTypes:      []string{},
	}
	h.policyRepo.addPolicy(pol)

	// Evaluate policies.
	policies, err := h.policyRepo.ListEnabled(ctx, h.tenantID)
	require.NoError(t, err)
	require.Len(t, policies, 1, "should have 1 enabled policy")

	violations, err := h.policyEngine.EvaluateAll(ctx, h.tenantID, policies)
	require.NoError(t, err)
	require.Len(t, violations, 2, "should detect 2 violations (both assets lack encryption)")

	// Verify violations contain correct data.
	for _, v := range violations {
		assert.Equal(t, pol.ID, v.PolicyID, "violation should reference the encryption policy")
		assert.Equal(t, "high", v.Severity, "violation severity should match policy severity")
		assert.Equal(t, string(model.EnforcementAutoRemediate), v.Enforcement,
			"violation enforcement should be auto_remediate")
		assert.NotEmpty(t, v.Description, "violation should have a description")
	}

	// Update evaluation results.
	err = h.policyRepo.UpdateEvaluationResults(ctx, h.tenantID, pol.ID, len(violations))
	require.NoError(t, err)

	// For auto_remediate violations, determine action and create remediations.
	policyMap := map[uuid.UUID]*model.DataPolicy{pol.ID: pol}
	autoRemediationCount := 0

	for i := range violations {
		v := &violations[i]
		p, ok := policyMap[v.PolicyID]
		require.True(t, ok)

		action := h.policyEnforcer.DetermineActionWithPlaybook(v, p.Enforcement, p.AutoPlaybookID)
		assert.True(t, action.CreateRemediation, "auto_remediate should set CreateRemediation=true")
		assert.Equal(t, "encrypt-sensitive-data", action.PlaybookID,
			"should use the policy's auto playbook ID")

		if action.CreateRemediation && action.PlaybookID != "" {
			remReq := &dto.CreateRemediationRequest{
				FindingType:    string(model.FindingPolicyViolation),
				DataAssetID:    &v.AssetID,
				DataAssetName:  v.AssetName,
				PlaybookID:     action.PlaybookID,
				Title:          fmt.Sprintf("Auto-remediation: %s - %s", v.PolicyName, v.AssetName),
				Description:    v.Description,
				Severity:       v.Severity,
				ComplianceTags: v.ComplianceFrameworks,
			}
			rem := h.createRemediation(t, remReq)
			assert.Equal(t, model.StatusOpen, rem.Status)
			assert.Equal(t, model.FindingPolicyViolation, rem.FindingType)
			assert.Equal(t, "encrypt-sensitive-data", rem.PlaybookID)
			autoRemediationCount++
		}
	}

	assert.Equal(t, 2, autoRemediationCount, "should create 2 auto-remediations")
	assert.Equal(t, 2, h.remRepo.count(), "repository should contain 2 remediations")

	// Verify that auto-created remediations have compliance tags.
	for _, rem := range h.remRepo.getAll() {
		var tags []string
		err := json.Unmarshal(rem.ComplianceTags, &tags)
		require.NoError(t, err)
		assert.Contains(t, tags, "SOC2", "compliance tags should include SOC2")
		assert.Contains(t, tags, "GDPR", "compliance tags should include GDPR")
	}
}

// ---------------------------------------------------------------------------
// Test: Exception Workflow
// ---------------------------------------------------------------------------

func TestExceptionWorkflow(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// Create a remediation.
	assetID := uuid.New()
	req := &dto.CreateRemediationRequest{
		FindingType:   "posture_gap",
		DataAssetID:   &assetID,
		DataAssetName: "test-asset",
		PlaybookID:    "posture-gap-generic",
		Title:         "Fix Posture Gap",
		Description:   "Address security posture gap",
		Severity:      "medium",
	}
	rem := h.createRemediation(t, req)
	assert.Equal(t, model.StatusOpen, rem.Status)

	// Request an exception for this remediation.
	requesterID := uuid.New()
	excReq := &dto.CreateExceptionRequest{
		ExceptionType:        string(model.ExceptionPostureFinding),
		RemediationID:        &rem.ID,
		DataAssetID:          &assetID,
		Justification:        "Business-critical system cannot be taken offline for remediation",
		BusinessReason:       "Revenue impact of $50K per hour of downtime",
		CompensatingControls: "Enhanced monitoring, WAF rules, quarterly security review",
		RiskScore:            45.0,
		RiskLevel:            "medium",
		ExpiresAt:            time.Now().Add(90 * 24 * time.Hour),
		ReviewIntervalDays:   30,
	}

	exc, err := h.exceptionMgr.Request(ctx, h.tenantID, requesterID, excReq)
	require.NoError(t, err)
	require.NotNil(t, exc)

	// Verify exception is in pending state.
	assert.Equal(t, model.ApprovalPending, exc.ApprovalStatus, "exception should be pending")
	assert.Equal(t, model.ExceptionStatusActive, exc.Status, "exception status should be active")
	assert.Equal(t, requesterID, exc.RequestedBy, "requested_by should match")
	assert.Nil(t, exc.ApprovedBy, "approved_by should be nil before approval")

	// Verify the remediation is still open (not yet approved).
	remBefore, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusOpen, remBefore.Status, "remediation should still be open")

	// Approve the exception (by a different user).
	approverID := uuid.New()
	approvedExc, err := h.exceptionMgr.Approve(ctx, h.tenantID, exc.ID, approverID)
	require.NoError(t, err)
	require.NotNil(t, approvedExc)

	assert.Equal(t, model.ApprovalApproved, approvedExc.ApprovalStatus, "exception should be approved")
	assert.NotNil(t, approvedExc.ApprovedBy, "approved_by should be set")
	assert.Equal(t, approverID, *approvedExc.ApprovedBy, "approved_by should be the approver")
	assert.NotNil(t, approvedExc.ApprovedAt, "approved_at should be set")
	assert.Equal(t, 1, approvedExc.ReviewCount, "review count should be 1")

	// Verify the linked remediation is now in exception_granted status.
	remAfterApproval, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusExceptionGranted, remAfterApproval.Status,
		"remediation should be exception_granted after exception approval")

	// Simulate exception expiry by setting the ExpiresAt to the past.
	excInRepo, err := h.exceptionRepo.GetByID(ctx, h.tenantID, exc.ID)
	require.NoError(t, err)
	excInRepo.ExpiresAt = time.Now().Add(-1 * time.Hour)
	err = h.exceptionRepo.Update(ctx, excInRepo)
	require.NoError(t, err)

	// Run the expiry checker.
	expiredCount, err := h.expiryChecker.Run(ctx, h.tenantID)
	require.NoError(t, err)
	assert.Equal(t, 1, expiredCount, "should expire 1 exception")

	// Verify the exception is now expired.
	expiredExc, err := h.exceptionRepo.GetByID(ctx, h.tenantID, exc.ID)
	require.NoError(t, err)
	assert.Equal(t, model.ExceptionStatusExpired, expiredExc.Status, "exception should be expired")
	assert.Equal(t, model.ApprovalExpired, expiredExc.ApprovalStatus, "approval status should be expired")

	// Verify the remediation re-opens.
	remAfterExpiry, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusOpen, remAfterExpiry.Status,
		"remediation should re-open after exception expiry")
}

// ---------------------------------------------------------------------------
// Test: SLA Breach Detection
// ---------------------------------------------------------------------------

func TestSLABreach(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// Create a critical remediation with SLA = 4 hours (per DefaultSLAConfig).
	assetID := uuid.New()
	critReq := &dto.CreateRemediationRequest{
		FindingType:   "encryption_missing",
		DataAssetID:   &assetID,
		DataAssetName: "critical-asset",
		PlaybookID:    "encrypt-sensitive-data",
		Title:         "Critical: Encrypt Data",
		Description:   "Encrypt critical data asset",
		Severity:      "critical",
	}
	critRem := h.createRemediation(t, critReq)

	// Verify SLA is set to 4 hours from now.
	require.NotNil(t, critRem.SLADueAt)
	expectedSLAHours := model.DefaultSLAConfig().SLAHoursForSeverity("critical")
	assert.Equal(t, 4, expectedSLAHours, "critical SLA should be 4 hours")

	// SLA is in the future, so no breach yet.
	breachCount, err := h.remRepo.FindSLABreached(ctx, h.tenantID)
	require.NoError(t, err)
	assert.Empty(t, breachCount, "no SLA breaches should exist initially")

	// Create a low severity remediation with SLA = 168 hours (7 days).
	lowReq := &dto.CreateRemediationRequest{
		FindingType:   "posture_gap",
		DataAssetID:   &assetID,
		DataAssetName: "low-priority-asset",
		PlaybookID:    "posture-gap-generic",
		Title:         "Low: Posture Gap",
		Description:   "Address low priority posture gap",
		Severity:      "low",
	}
	lowRem := h.createRemediation(t, lowReq)

	require.NotNil(t, lowRem.SLADueAt)
	lowSLAHours := model.DefaultSLAConfig().SLAHoursForSeverity("low")
	assert.Equal(t, 168, lowSLAHours, "low SLA should be 168 hours (7 days)")

	// Simulate SLA breach by moving the critical remediation's SLA to the past.
	critInRepo, err := h.remRepo.GetByID(ctx, h.tenantID, critRem.ID)
	require.NoError(t, err)
	pastSLA := time.Now().Add(-1 * time.Hour)
	critInRepo.SLADueAt = &pastSLA
	err = h.remRepo.Update(ctx, critInRepo)
	require.NoError(t, err)

	// Check for breaches.
	breached, err := h.remRepo.FindSLABreached(ctx, h.tenantID)
	require.NoError(t, err)
	require.Len(t, breached, 1, "should find 1 SLA breach")
	assert.Equal(t, critRem.ID, breached[0].ID, "breached remediation should be the critical one")

	// Mark breach.
	err = h.remRepo.MarkSLABreached(ctx, h.tenantID, critRem.ID)
	require.NoError(t, err)

	// Record SLA breach history.
	h.recordHistory(t, critRem.ID, model.HistoryActionSLABreached, nil, model.ActorTypeScheduler, map[string]interface{}{
		"sla_due_at": pastSLA,
		"severity":   "critical",
		"status":     string(critRem.Status),
	})

	// Verify the remediation is flagged.
	critAfterBreach, err := h.remRepo.GetByID(ctx, h.tenantID, critRem.ID)
	require.NoError(t, err)
	assert.True(t, critAfterBreach.SLABreached, "critical remediation should be flagged as SLA breached")

	// Verify no double-counting: running FindSLABreached again should return empty.
	breachedAgain, err := h.remRepo.FindSLABreached(ctx, h.tenantID)
	require.NoError(t, err)
	assert.Empty(t, breachedAgain, "should not find already-marked breaches")

	// Verify SLA breach history entry.
	slaEntries := h.histRepo.findByAction(critRem.ID, model.HistoryActionSLABreached)
	require.Len(t, slaEntries, 1, "should have 1 SLA breach history entry")

	// Verify the low-severity remediation is NOT breached.
	lowAfter, err := h.remRepo.GetByID(ctx, h.tenantID, lowRem.ID)
	require.NoError(t, err)
	assert.False(t, lowAfter.SLABreached, "low severity remediation should not be breached")
}

// ---------------------------------------------------------------------------
// Test: Rollback
// ---------------------------------------------------------------------------

func TestRollback(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// Use "remediate-shadow-copy" playbook which has AutoRollback=true and RequiresApproval=true.
	assetID := uuid.New()
	req := &dto.CreateRemediationRequest{
		FindingType:   "shadow_copy",
		DataAssetID:   &assetID,
		DataAssetName: "shadow-copy-db",
		PlaybookID:    "remediate-shadow-copy",
		Title:         "Remediate Shadow Copy",
		Description:   "Quarantine and remediate unauthorized shadow copy",
		Severity:      "high",
	}
	rem := h.createRemediation(t, req)

	// This playbook requires approval, so initial status is awaiting_approval.
	assert.Equal(t, model.StatusAwaitingApproval, rem.Status,
		"shadow copy playbook requires approval, so initial status should be awaiting_approval")
	assert.True(t, rem.RollbackAvailable, "remediate-shadow-copy supports auto-rollback")

	// Approve the remediation.
	approverID := uuid.New()
	remApproved, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	remApproved.Status = model.StatusInProgress
	err = h.remRepo.Update(ctx, remApproved)
	require.NoError(t, err)

	h.recordHistory(t, rem.ID, model.HistoryActionStatusChanged, &approverID, model.ActorTypeUser, map[string]interface{}{
		"from_status": string(model.StatusAwaitingApproval),
		"to_status":   string(model.StatusInProgress),
		"approved_by": approverID.String(),
	})

	// Execute 2 of the 4 steps.
	for i := 0; i < 2; i++ {
		result := h.executeStep(t, rem.ID, &h.userID)
		assert.Equal(t, model.StepStatusCompleted, result.Status)
	}

	// Verify partial progress.
	partialRem, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusInProgress, partialRem.Status)
	assert.Equal(t, 2, partialRem.CurrentStep, "should be on step 2")

	// Set some pre_action_state for the rollback audit trail.
	partialRem.PreActionState = json.RawMessage(`{"original_access_paths": 6, "original_classification": "internal"}`)
	err = h.remRepo.Update(ctx, partialRem)
	require.NoError(t, err)

	// Rollback.
	rollbackRem, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	require.True(t, rollbackRem.RollbackAvailable, "rollback must be available")

	previousStatus := rollbackRem.Status
	rollbackRem.Status = model.StatusRolledBack
	rollbackRem.RolledBack = true
	err = h.remRepo.Update(ctx, rollbackRem)
	require.NoError(t, err)

	// Record rollback history with pre_action_state.
	historyDetails := map[string]interface{}{
		"from_status": string(previousStatus),
		"to_status":   string(model.StatusRolledBack),
		"reason":      "Owner reported service disruption caused by quarantine",
	}
	if len(rollbackRem.PreActionState) > 0 {
		historyDetails["pre_action_state"] = json.RawMessage(rollbackRem.PreActionState)
	}
	h.recordHistory(t, rem.ID, model.HistoryActionRolledBack, &h.userID, model.ActorTypeUser, historyDetails)

	// Verify final state.
	finalRem, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusRolledBack, finalRem.Status, "status should be rolled_back")
	assert.True(t, finalRem.RolledBack, "rolled_back flag should be true")

	// Verify rollback history entry exists.
	rollbackEntries := h.histRepo.findByAction(rem.ID, model.HistoryActionRolledBack)
	require.Len(t, rollbackEntries, 1, "should have 1 rolled_back history entry")

	// Verify the rolled_back entry includes pre_action_state.
	var rollbackDetails map[string]interface{}
	err = json.Unmarshal(rollbackEntries[0].Details, &rollbackDetails)
	require.NoError(t, err)
	assert.Contains(t, rollbackDetails, "pre_action_state", "rollback history should include pre_action_state")
	assert.Contains(t, rollbackDetails, "reason", "rollback history should include reason")
	assert.Equal(t, string(model.StatusRolledBack), rollbackDetails["to_status"])
}

// ---------------------------------------------------------------------------
// Test: Playbook Registry and Executor Integration
// ---------------------------------------------------------------------------

func TestPlaybookRegistryAndExecutor(t *testing.T) {
	logger := zerolog.Nop()
	registry := playbook.NewRegistry()
	executor := playbook.NewPlaybookExecutor(registry, logger)
	ctx := context.Background()

	// Verify all 10 built-in playbooks are registered.
	allPlaybooks := registry.List()
	assert.Len(t, allPlaybooks, 10, "should have 10 built-in playbooks")

	// Test each playbook can be retrieved by ID and executed.
	expectedPlaybooks := []struct {
		id          string
		findingType model.FindingType
		minSteps    int
	}{
		{"encrypt-sensitive-data", model.FindingEncryptionMissing, 5},
		{"revoke-overprivileged-access", model.FindingOverprivilegedAccess, 4},
		{"restrict-network-exposure", model.FindingExposureRisk, 4},
		{"remediate-shadow-copy", model.FindingShadowCopy, 4},
		{"enforce-pii-controls", model.FindingPIIUnprotected, 5},
		{"handle-classification-drift", model.FindingClassificationDrift, 4},
		{"enforce-data-retention", model.FindingRetentionExpired, 3},
		{"reduce-blast-radius", model.FindingBlastRadiusExcessive, 4},
		{"posture-gap-generic", model.FindingPostureGap, 4},
		{"stale-access-cleanup", model.FindingStaleAccess, 4},
	}

	for _, expected := range expectedPlaybooks {
		t.Run(expected.id, func(t *testing.T) {
			pb, ok := registry.Get(expected.id)
			require.True(t, ok, "playbook %q should be in registry", expected.id)
			assert.Equal(t, expected.findingType, pb.FindingType)
			assert.GreaterOrEqual(t, len(pb.Steps), expected.minSteps)

			// Verify GetForFindingType works.
			byType, ok := registry.GetForFindingType(expected.findingType)
			assert.True(t, ok, "should find playbook by finding type %q", expected.findingType)
			assert.Equal(t, expected.id, byType.ID)

			// Execute the first step.
			result, err := executor.Execute(ctx, pb, 0)
			require.NoError(t, err)
			assert.Equal(t, model.StepStatusCompleted, result.Status,
				"first step of %q should complete", expected.id)
			assert.NotEmpty(t, result.StepID)
			assert.NotEmpty(t, result.Action)
			assert.GreaterOrEqual(t, result.DurationMs, int64(0))
		})
	}
}

// ---------------------------------------------------------------------------
// Test: Playbook Validator Dry Run
// ---------------------------------------------------------------------------

func TestPlaybookValidatorDryRun(t *testing.T) {
	logger := zerolog.Nop()
	registry := playbook.NewRegistry()
	validator := playbook.NewValidator(registry, logger)
	ctx := context.Background()

	t.Run("valid asset-related playbook", func(t *testing.T) {
		assetID := uuid.New()
		result, err := validator.DryRun(ctx, "encrypt-sensitive-data", &assetID, "")
		require.NoError(t, err)
		assert.True(t, result.Valid, "should be valid with asset ID")
		assert.Equal(t, 1, result.AssetsAffected)
		assert.Greater(t, result.EstimatedRiskReduction, 0.0)
	})

	t.Run("asset-related playbook without asset ID", func(t *testing.T) {
		result, err := validator.DryRun(ctx, "encrypt-sensitive-data", nil, "")
		require.NoError(t, err)
		assert.False(t, result.Valid, "should be invalid without asset ID for asset-related playbook")
		assert.NotEmpty(t, result.Issues)
	})

	t.Run("unknown playbook", func(t *testing.T) {
		result, err := validator.DryRun(ctx, "nonexistent-playbook", nil, "")
		require.NoError(t, err)
		assert.False(t, result.Valid, "should be invalid for unknown playbook")
		assert.NotEmpty(t, result.Issues)
	})

	t.Run("approval-required playbook", func(t *testing.T) {
		assetID := uuid.New()
		result, err := validator.DryRun(ctx, "remediate-shadow-copy", &assetID, "")
		require.NoError(t, err)
		assert.True(t, result.Valid)
		// Should have a note about approval requirement.
		foundApprovalNote := false
		for _, issue := range result.Issues {
			if len(issue) > 0 {
				foundApprovalNote = true
			}
		}
		assert.True(t, foundApprovalNote || len(result.Issues) > 0,
			"should have issues/notes for approval-required playbook")
	})
}

// ---------------------------------------------------------------------------
// Test: Policy Evaluation with Exception Suppression
// ---------------------------------------------------------------------------

func TestPolicyEvaluationWithExceptionSuppression(t *testing.T) {
	encryptedFalse := false
	encryptedTrue := true

	assetWithException := uuid.New()
	assetWithoutException := uuid.New()

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 assetWithException,
			AssetName:          "excepted-asset",
			AssetType:          "database",
			DataClassification: "confidential",
			EncryptedAtRest:    &encryptedFalse,
			EncryptedInTransit: &encryptedFalse,
			CreatedAt:          time.Now(),
		},
		{
			ID:                 assetWithoutException,
			AssetName:          "not-excepted-asset",
			AssetType:          "database",
			DataClassification: "confidential",
			EncryptedAtRest:    &encryptedFalse,
			EncryptedInTransit: &encryptedTrue,
			CreatedAt:          time.Now(),
		},
	}

	h := newTestHarness(assets)
	ctx := context.Background()

	// Create a policy.
	policyID := uuid.New()
	encRule := model.EncryptionRule{RequireAtRest: true, ClassificationMin: "confidential"}
	ruleJSON, _ := json.Marshal(encRule)

	pol := &model.DataPolicy{
		ID:                  policyID,
		TenantID:            h.tenantID,
		Name:                "Require Encryption At Rest",
		Category:            model.PolicyCategoryEncryption,
		Rule:                ruleJSON,
		Enforcement:         model.EnforcementAlert,
		Severity:            "high",
		Enabled:             true,
		ScopeClassification: []string{},
		ScopeAssetTypes:     []string{},
		ComplianceFrameworks: []string{},
	}
	h.policyRepo.addPolicy(pol)

	// Create an active, approved exception for assetWithException.
	approverID := uuid.New()
	now := time.Now().UTC()
	excToInsert := &model.RiskException{
		ID:             uuid.New(),
		TenantID:       h.tenantID,
		ExceptionType:  model.ExceptionEncryptionGap,
		DataAssetID:    &assetWithException,
		PolicyID:       &policyID,
		Justification:  "Legacy system, encryption planned for next quarter",
		RequestedBy:    h.userID,
		ApprovedBy:     &approverID,
		ApprovalStatus: model.ApprovalApproved,
		ApprovedAt:     &now,
		ExpiresAt:      time.Now().Add(90 * 24 * time.Hour),
		Status:         model.ExceptionStatusActive,
		RiskScore:      30.0,
		RiskLevel:      "medium",
		ReviewIntervalDays: 30,
	}
	_, err := h.exceptionRepo.Create(ctx, excToInsert)
	require.NoError(t, err)

	// Evaluate policies.
	policies, _ := h.policyRepo.ListEnabled(ctx, h.tenantID)
	violations, err := h.policyEngine.EvaluateAll(ctx, h.tenantID, policies)
	require.NoError(t, err)

	// Only the non-excepted asset should appear in violations.
	assert.Len(t, violations, 1, "should have 1 violation (the excepted asset should be suppressed)")
	if len(violations) > 0 {
		assert.Equal(t, assetWithoutException, violations[0].AssetID,
			"violation should be for the non-excepted asset")
	}
}

// ---------------------------------------------------------------------------
// Test: History Chain Integrity
// ---------------------------------------------------------------------------

func TestHistoryChainIntegrity(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	remediationID := uuid.New()

	// Record a sequence of history entries.
	actions := []model.HistoryAction{
		model.HistoryActionCreated,
		model.HistoryActionStepCompleted,
		model.HistoryActionStepCompleted,
		model.HistoryActionStatusChanged,
		model.HistoryActionAssigned,
	}

	for i, action := range actions {
		h.recordHistory(t, remediationID, action, &h.userID, model.ActorTypeUser, map[string]interface{}{
			"step_index": i,
		})
		// Small sleep to ensure distinct timestamps for ordering.
		time.Sleep(1 * time.Millisecond)
	}

	// Retrieve all entries.
	entries, total, err := h.histRepo.ListByRemediation(ctx, h.tenantID, remediationID, 1, 100)
	require.NoError(t, err)
	assert.Equal(t, 5, total, "should have 5 history entries")
	require.Len(t, entries, 5)

	// Verify hash chain integrity.
	for i := 1; i < len(entries); i++ {
		assert.Equal(t, entries[i-1].EntryHash, entries[i].PrevHash,
			"entry %d PrevHash should match entry %d EntryHash", i, i-1)
	}

	// The first entry should have an empty PrevHash.
	assert.Empty(t, entries[0].PrevHash, "first entry should have empty PrevHash")

	// All entries should have non-empty EntryHash.
	for i, entry := range entries {
		assert.NotEmpty(t, entry.EntryHash, "entry %d should have a non-empty EntryHash", i)
	}
}

// ---------------------------------------------------------------------------
// Test: SIEM Integration
// ---------------------------------------------------------------------------

func TestSIEMExportIntegration(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	violations := []model.PolicyViolation{
		{
			PolicyID:             uuid.New(),
			PolicyName:           "Encryption Required",
			Category:             "encryption",
			AssetID:              uuid.New(),
			AssetName:            "customer-db",
			AssetType:            "database",
			Classification:       "confidential",
			Severity:             "critical",
			Description:          "Missing encryption at rest",
			Enforcement:          "auto_remediate",
			ComplianceFrameworks: []string{"SOC2", "PCI-DSS"},
		},
		{
			PolicyID:       uuid.New(),
			PolicyName:     "Network Exposure",
			Category:       "exposure",
			AssetID:        uuid.New(),
			AssetName:      "api-gateway",
			AssetType:      "service",
			Classification: "internal",
			Severity:       "high",
			Description:    "Internet-facing asset exceeds policy",
			Enforcement:    "alert",
		},
	}

	events, err := h.siemExporter.ExportFindings(ctx, h.tenantID, violations)
	require.NoError(t, err)
	require.Len(t, events, 2, "should export 2 SIEM events")

	// Verify event fields.
	assert.Equal(t, h.tenantID, events[0].TenantID)
	assert.Equal(t, "critical", events[0].Severity)
	assert.Equal(t, "encryption", events[0].FindingType)
	assert.NotEmpty(t, events[0].RecommendedAction)
	assert.Equal(t, integration.SIEMFormatJSON, events[0].Format)

	// Test CEF format output.
	cef := h.siemExporter.FormatCEF(events[0])
	assert.Contains(t, cef, "CEF:0|Clario360|DSPM|1.0|")
	assert.Contains(t, cef, "encryption")

	// Test JSON format output.
	jsonData, err := h.siemExporter.FormatJSON(events[1])
	require.NoError(t, err)
	assert.Contains(t, string(jsonData), "exposure")
}

// ---------------------------------------------------------------------------
// Test: ITSM Connector Integration
// ---------------------------------------------------------------------------

func TestITSMConnectorIntegration(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	now := time.Now().UTC()
	slaDue := now.Add(24 * time.Hour)
	assetID := uuid.New()

	rem := &model.Remediation{
		ID:            uuid.New(),
		TenantID:      h.tenantID,
		FindingType:   model.FindingEncryptionMissing,
		DataAssetID:   &assetID,
		DataAssetName: "customer-db",
		PlaybookID:    "encrypt-sensitive-data",
		Title:         "Encrypt customer database",
		Description:   "Apply AES-256-GCM encryption at rest",
		Severity:      "high",
		Status:        model.StatusOpen,
		SLADueAt:      &slaDue,
		CurrentStep:   0,
		TotalSteps:    5,
		AssignedTeam:  "security-ops",
		CreatedAt:     now,
	}

	ticket, err := h.itsmConnector.CreateTicket(ctx, h.tenantID, rem)
	require.NoError(t, err)
	require.NotNil(t, ticket)

	assert.NotEmpty(t, ticket.ExternalTicketID, "ticket should have an external ID")
	assert.Contains(t, ticket.ExternalTicketID, "DSPM-", "ticket ID should start with DSPM-")
	assert.NotEmpty(t, ticket.URL, "ticket should have a URL")
	assert.Equal(t, "open", ticket.Status, "ticket status should be open")

	// Verify deterministic ticket ID (same input = same ID).
	ticket2, err := h.itsmConnector.CreateTicket(ctx, h.tenantID, rem)
	require.NoError(t, err)
	assert.Equal(t, ticket.ExternalTicketID, ticket2.ExternalTicketID,
		"ticket ID should be deterministic for the same remediation")

	// Verify severity-to-priority mapping.
	assert.Equal(t, "P2", integration.SeverityToPriority("high"))
	assert.Equal(t, "P1", integration.SeverityToPriority("critical"))
	assert.Equal(t, "P3", integration.SeverityToPriority("medium"))
	assert.Equal(t, "P4", integration.SeverityToPriority("low"))
}

// ---------------------------------------------------------------------------
// Test: DLP Policy Generation Integration
// ---------------------------------------------------------------------------

func TestDLPPolicyGenerationIntegration(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	assets := []cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "customer-profiles",
			DataClassification: "confidential",
			ContainsPII:        true,
			PIITypes:           []string{"email", "phone", "credit_card"},
		},
		{
			ID:                 uuid.New(),
			AssetName:          "health-records",
			DataClassification: "restricted",
			ContainsPII:        true,
			PIITypes:           []string{"health_data", "ssn"},
		},
		{
			ID:                 uuid.New(),
			AssetName:          "public-docs",
			DataClassification: "public",
			ContainsPII:        false,
			PIITypes:           nil,
		},
	}

	rules, err := h.dlpGenerator.Generate(ctx, h.tenantID, assets)
	require.NoError(t, err)

	// Should generate rules for: email, phone, credit_card, health_data, ssn = 5 unique PII types.
	assert.Len(t, rules, 5, "should generate 5 DLP rules for 5 unique PII types")

	// Verify specific PII types are covered.
	piiTypesFound := make(map[string]bool)
	for _, rule := range rules {
		piiTypesFound[rule.PIIType] = true
		assert.NotEmpty(t, rule.ID, "rule should have an ID")
		assert.NotEmpty(t, rule.DataIdentifier, "rule should have a data identifier")
		assert.NotEmpty(t, rule.Channels, "rule should have channels")
		assert.NotEmpty(t, rule.Actions, "rule should have actions")
		assert.NotEmpty(t, rule.Description, "rule should have a description")
	}

	assert.True(t, piiTypesFound["email"], "should have email rule")
	assert.True(t, piiTypesFound["credit_card"], "should have credit_card rule")
	assert.True(t, piiTypesFound["health_data"], "should have health_data rule")
	assert.True(t, piiTypesFound["ssn"], "should have ssn rule")
}

// ---------------------------------------------------------------------------
// Test: Lifecycle Components Integration
// ---------------------------------------------------------------------------

func TestLifecycleComponentsIntegration(t *testing.T) {
	oldAsset := &cybermodel.DSPMDataAsset{
		ID:                 uuid.New(),
		AssetName:          "stale-archive",
		DataClassification: "confidential",
		CreatedAt:          time.Now().Add(-200 * 24 * time.Hour), // 200 days old
	}
	recentAsset := &cybermodel.DSPMDataAsset{
		ID:                 uuid.New(),
		AssetName:          "recent-data",
		DataClassification: "internal",
		CreatedAt:          time.Now().Add(-10 * 24 * time.Hour), // 10 days old
	}
	neverScannedAsset := &cybermodel.DSPMDataAsset{
		ID:                 uuid.New(),
		AssetName:          "never-scanned",
		DataClassification: "restricted",
		CreatedAt:          time.Now().Add(-100 * 24 * time.Hour), // 100 days old, never scanned
	}

	assets := []*cybermodel.DSPMDataAsset{oldAsset, recentAsset, neverScannedAsset}
	h := newTestHarness(assets)
	ctx := context.Background()

	t.Run("retention enforcement", func(t *testing.T) {
		// Set max retention to 90 days for confidential data.
		violations, err := h.retentionEnforcer.Evaluate(ctx, h.tenantID, 90, []string{"confidential"})
		require.NoError(t, err)
		require.Len(t, violations, 1, "should find 1 retention violation")
		assert.Equal(t, oldAsset.ID, violations[0].AssetID)
		assert.Greater(t, violations[0].DaysOverdue, 0)
		assert.Equal(t, "confidential", violations[0].Classification)
	})

	t.Run("stale data detection", func(t *testing.T) {
		// Assets with no LastScannedAt or LastScannedAt > 90 days ago are stale.
		findings, err := h.staleDetector.Detect(ctx, h.tenantID)
		require.NoError(t, err)

		// oldAsset (200 days, no scan) and neverScannedAsset (100 days, no scan) should be detected.
		// recentAsset (10 days) should NOT be stale (creation date within threshold).
		staleIDs := make(map[uuid.UUID]bool)
		for _, f := range findings {
			staleIDs[f.AssetID] = true
			assert.NotEmpty(t, f.Confidence, "finding should have confidence level")
			assert.NotEmpty(t, f.Recommendation, "finding should have recommendation")
		}

		assert.True(t, staleIDs[oldAsset.ID], "200-day-old asset should be stale")
		assert.True(t, staleIDs[neverScannedAsset.ID], "never-scanned 100-day asset should be stale")
	})
}

// ---------------------------------------------------------------------------
// Test: Exception Rejection
// ---------------------------------------------------------------------------

func TestExceptionRejection(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// Create a remediation.
	assetID := uuid.New()
	rem := h.createRemediation(t, &dto.CreateRemediationRequest{
		FindingType:   "encryption_missing",
		DataAssetID:   &assetID,
		DataAssetName: "test-asset",
		PlaybookID:    "encrypt-sensitive-data",
		Title:         "Test Remediation",
		Description:   "Test description",
		Severity:      "high",
	})

	// Request an exception.
	requesterID := uuid.New()
	exc, err := h.exceptionMgr.Request(ctx, h.tenantID, requesterID, &dto.CreateExceptionRequest{
		ExceptionType:      string(model.ExceptionEncryptionGap),
		RemediationID:      &rem.ID,
		DataAssetID:        &assetID,
		Justification:      "Need more time to encrypt",
		RiskScore:          60.0,
		RiskLevel:          "high",
		ExpiresAt:          time.Now().Add(30 * 24 * time.Hour),
		ReviewIntervalDays: 14,
	})
	require.NoError(t, err)
	assert.Equal(t, model.ApprovalPending, exc.ApprovalStatus)

	// Reject the exception.
	rejecterID := uuid.New()
	rejected, err := h.exceptionMgr.Reject(ctx, h.tenantID, exc.ID, rejecterID, "Risk too high, must remediate immediately")
	require.NoError(t, err)

	assert.Equal(t, model.ApprovalRejected, rejected.ApprovalStatus, "should be rejected")
	assert.Equal(t, "Risk too high, must remediate immediately", rejected.RejectionReason)
	assert.NotNil(t, rejected.ApprovedBy, "approved_by should be set (to rejecter)")

	// Verify the remediation status is unchanged (still open).
	remAfter, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusOpen, remAfter.Status,
		"remediation should remain open after exception rejection")

	// Verify self-approval is rejected.
	exc2, err := h.exceptionMgr.Request(ctx, h.tenantID, requesterID, &dto.CreateExceptionRequest{
		ExceptionType:      string(model.ExceptionEncryptionGap),
		DataAssetID:        &assetID,
		Justification:      "Another request",
		RiskScore:          30.0,
		RiskLevel:          "low",
		ExpiresAt:          time.Now().Add(30 * 24 * time.Hour),
		ReviewIntervalDays: 30,
	})
	require.NoError(t, err)

	_, err = h.exceptionMgr.Approve(ctx, h.tenantID, exc2.ID, requesterID)
	assert.Error(t, err, "self-approval should be rejected")
	assert.Contains(t, err.Error(), "approver cannot be the same as the requester")
}

// ---------------------------------------------------------------------------
// Test: Remediation with Approval Required
// ---------------------------------------------------------------------------

func TestRemediationWithApprovalRequired(t *testing.T) {
	h := newTestHarness(nil)
	ctx := context.Background()

	// The "remediate-shadow-copy" and "enforce-data-retention" playbooks require approval.
	assetID := uuid.New()
	rem := h.createRemediation(t, &dto.CreateRemediationRequest{
		FindingType:   "retention_expired",
		DataAssetID:   &assetID,
		DataAssetName: "expired-data",
		PlaybookID:    "enforce-data-retention",
		Title:         "Archive Expired Data",
		Description:   "Archive data that exceeded retention policy",
		Severity:      "medium",
	})

	assert.Equal(t, model.StatusAwaitingApproval, rem.Status,
		"enforce-data-retention requires approval, initial status should be awaiting_approval")

	// Verify the playbook has RequiresApproval set.
	pb, ok := h.registry.Get("enforce-data-retention")
	require.True(t, ok)
	assert.True(t, pb.RequiresApproval, "enforce-data-retention should require approval")

	// Approve the remediation (simulating engine.ApproveRemediation).
	remApproved, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	require.Equal(t, model.StatusAwaitingApproval, remApproved.Status)

	remApproved.Status = model.StatusInProgress
	err = h.remRepo.Update(ctx, remApproved)
	require.NoError(t, err)

	// Now execute all 3 steps.
	for i := 0; i < 3; i++ {
		result := h.executeStep(t, rem.ID, &h.userID)
		assert.Equal(t, model.StepStatusCompleted, result.Status)
	}

	// Verify completion.
	final, err := h.remRepo.GetByID(ctx, h.tenantID, rem.ID)
	require.NoError(t, err)
	assert.Equal(t, model.StatusCompleted, final.Status)
	assert.NotNil(t, final.CompletedAt)
}

// ---------------------------------------------------------------------------
// Test: SLA Calculations for Different Severities
// ---------------------------------------------------------------------------

func TestSLACalculations(t *testing.T) {
	slaConfig := model.DefaultSLAConfig()

	tests := []struct {
		severity      string
		expectedHours int
	}{
		{"critical", 4},
		{"high", 24},
		{"medium", 72},
		{"low", 168},
		{"unknown", 72}, // defaults to medium
	}

	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			hours := slaConfig.SLAHoursForSeverity(tt.severity)
			assert.Equal(t, tt.expectedHours, hours)

			// Verify SLADueAt calculation.
			now := time.Now().UTC()
			dueAt := dto.SLADueAt(now, tt.severity)
			expected := now.Add(time.Duration(tt.expectedHours) * time.Hour)
			// Allow 1 second tolerance for clock skew.
			assert.WithinDuration(t, expected, dueAt, 1*time.Second,
				"SLA due at for %s should be %d hours from now", tt.severity, tt.expectedHours)
		})
	}
}

// ---------------------------------------------------------------------------
// Test: Remediation Status Terminal States
// ---------------------------------------------------------------------------

func TestRemediationStatusTerminalStates(t *testing.T) {
	terminalStatuses := []model.RemediationStatus{
		model.StatusCompleted,
		model.StatusCancelled,
		model.StatusRolledBack,
		model.StatusExceptionGranted,
	}

	nonTerminalStatuses := []model.RemediationStatus{
		model.StatusOpen,
		model.StatusInProgress,
		model.StatusAwaitingApproval,
		model.StatusFailed,
	}

	for _, s := range terminalStatuses {
		assert.True(t, s.IsTerminal(), "%q should be terminal", s)
	}

	for _, s := range nonTerminalStatuses {
		assert.False(t, s.IsTerminal(), "%q should NOT be terminal", s)
	}
}

// ---------------------------------------------------------------------------
// Test: Multiple Policy Violations and Enforcement Actions
// ---------------------------------------------------------------------------

func TestMultiplePolicyEnforcementActions(t *testing.T) {
	logger := zerolog.Nop()
	enforcer := policy.NewEnforcer(logger)

	violation := &model.PolicyViolation{
		PolicyID:  uuid.New(),
		AssetID:   uuid.New(),
		AssetName: "test-asset",
		Severity:  "high",
	}

	t.Run("alert enforcement", func(t *testing.T) {
		action := enforcer.DetermineAction(violation, model.EnforcementAlert)
		assert.Equal(t, "alert", action.Action)
		assert.True(t, action.CreateAlert)
		assert.False(t, action.CreateRemediation)
		assert.False(t, action.QuarantineAsset)
	})

	t.Run("auto_remediate enforcement", func(t *testing.T) {
		action := enforcer.DetermineActionWithPlaybook(violation, model.EnforcementAutoRemediate, "encrypt-sensitive-data")
		assert.Equal(t, "auto_remediate", action.Action)
		assert.True(t, action.CreateAlert)
		assert.True(t, action.CreateRemediation)
		assert.False(t, action.QuarantineAsset)
		assert.Equal(t, "encrypt-sensitive-data", action.PlaybookID)
	})

	t.Run("block enforcement", func(t *testing.T) {
		action := enforcer.DetermineAction(violation, model.EnforcementBlock)
		assert.Equal(t, "block", action.Action)
		assert.True(t, action.CreateAlert)
		assert.False(t, action.CreateRemediation)
		assert.True(t, action.QuarantineAsset)
	})
}

// ---------------------------------------------------------------------------
// Test: Finding Type and Status Validity
// ---------------------------------------------------------------------------

func TestFindingTypeValidity(t *testing.T) {
	validTypes := model.ValidFindingTypes()
	assert.Len(t, validTypes, 11, "should have 11 valid finding types")

	for _, ft := range validTypes {
		assert.True(t, ft.IsValid(), "%q should be valid", ft)
	}

	invalid := model.FindingType("nonexistent_type")
	assert.False(t, invalid.IsValid(), "unknown finding type should be invalid")
}

func TestRemediationStatusValidity(t *testing.T) {
	validStatuses := model.ValidStatuses()
	assert.Len(t, validStatuses, 8, "should have 8 valid statuses")

	for _, s := range validStatuses {
		assert.True(t, s.IsValid(), "%q should be valid", s)
	}

	invalid := model.RemediationStatus("nonexistent_status")
	assert.False(t, invalid.IsValid(), "unknown status should be invalid")
}

// ---------------------------------------------------------------------------
// Test: Policy Evaluation Across Multiple Policy Categories
// ---------------------------------------------------------------------------

func TestPolicyEvaluationMultipleCategories(t *testing.T) {
	encFalse := false
	internetFacing := "internet_facing"

	assets := []*cybermodel.DSPMDataAsset{
		{
			ID:                 uuid.New(),
			AssetName:          "sensitive-db",
			AssetType:          "database",
			DataClassification: "confidential",
			EncryptedAtRest:    &encFalse,
			NetworkExposure:    &internetFacing,
			ContainsPII:        true,
			PIITypes:           []string{"email"},
			AuditLogging:       &encFalse,
			CreatedAt:          time.Now(),
		},
	}

	h := newTestHarness(assets)
	ctx := context.Background()

	// Add encryption policy.
	encRule, _ := json.Marshal(model.EncryptionRule{RequireAtRest: true, ClassificationMin: "confidential"})
	h.policyRepo.addPolicy(&model.DataPolicy{
		ID: uuid.New(), TenantID: h.tenantID, Name: "Encryption Policy",
		Category: model.PolicyCategoryEncryption, Rule: encRule,
		Enforcement: model.EnforcementAlert, Severity: "high", Enabled: true,
		ScopeClassification: []string{}, ScopeAssetTypes: []string{}, ComplianceFrameworks: []string{},
	})

	// Add exposure policy.
	expRule, _ := json.Marshal(model.ExposureRule{MaxExposure: "vpn_accessible", ClassificationMin: "confidential"})
	h.policyRepo.addPolicy(&model.DataPolicy{
		ID: uuid.New(), TenantID: h.tenantID, Name: "Exposure Policy",
		Category: model.PolicyCategoryExposure, Rule: expRule,
		Enforcement: model.EnforcementAlert, Severity: "critical", Enabled: true,
		ScopeClassification: []string{}, ScopeAssetTypes: []string{}, ComplianceFrameworks: []string{},
	})

	// Add audit logging policy.
	auditRule, _ := json.Marshal(model.AuditLoggingRule{RequiredFor: []string{"confidential", "restricted"}})
	h.policyRepo.addPolicy(&model.DataPolicy{
		ID: uuid.New(), TenantID: h.tenantID, Name: "Audit Logging Policy",
		Category: model.PolicyCategoryAuditLogging, Rule: auditRule,
		Enforcement: model.EnforcementAlert, Severity: "medium", Enabled: true,
		ScopeClassification: []string{}, ScopeAssetTypes: []string{}, ComplianceFrameworks: []string{},
	})

	policies, _ := h.policyRepo.ListEnabled(ctx, h.tenantID)
	require.Len(t, policies, 3, "should have 3 enabled policies")

	violations, err := h.policyEngine.EvaluateAll(ctx, h.tenantID, policies)
	require.NoError(t, err)

	// The single asset should violate all 3 policies.
	assert.Len(t, violations, 3, "sensitive-db should violate encryption, exposure, and audit logging policies")

	categories := make(map[string]bool)
	for _, v := range violations {
		categories[v.Category] = true
	}
	assert.True(t, categories["encryption"], "should have encryption violation")
	assert.True(t, categories["exposure"], "should have exposure violation")
	assert.True(t, categories["audit_logging"], "should have audit logging violation")
}

// Ensure the interface wrappers compile.
var _ exception.ExceptionRepository = (*mockExceptionRepo)(nil)
var _ exception.RemediationUpdater = (*mockRemediationUpdater)(nil)
var _ policy.AssetLister = (*mockAssetLister)(nil)
var _ policy.ExceptionChecker = (*mockExceptionRepo)(nil)
var _ lifecycle.AssetLister = (*mockAssetLister)(nil)
