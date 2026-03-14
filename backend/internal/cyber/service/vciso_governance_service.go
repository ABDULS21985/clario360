package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

// VCISOGovernanceService handles business logic for vCISO governance features.
type VCISOGovernanceService struct {
	repo     *repository.VCISOGovernanceRepository
	producer *events.Producer
	logger   zerolog.Logger
}

// NewVCISOGovernanceService creates a new VCISOGovernanceService.
func NewVCISOGovernanceService(
	repo *repository.VCISOGovernanceRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *VCISOGovernanceService {
	return &VCISOGovernanceService{
		repo:     repo,
		producer: producer,
		logger:   logger.With().Str("service", "vciso-governance").Logger(),
	}
}

// ─── Risks ──────────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListRisks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListRisks(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISORiskEntry{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateRisk(ctx context.Context, tenantID uuid.UUID, req *dto.CreateRiskRequest) (*model.VCISORiskEntry, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	item := &model.VCISORiskEntry{
		Title: req.Title, Description: req.Description, Category: req.Category,
		Department: req.Department, InherentScore: req.InherentScore, ResidualScore: req.ResidualScore,
		Likelihood: req.Likelihood, Impact: req.Impact, Status: req.Status, Treatment: req.Treatment,
		OwnerID: dto.ParseOptionalUUID(req.OwnerID), OwnerName: req.OwnerName,
		ReviewDate: req.ReviewDate, BusinessServices: req.BusinessServices,
		Controls: req.Controls, Tags: req.Tags, TreatmentPlan: req.TreatmentPlan,
		AcceptanceRationale: req.AcceptanceRationale, AcceptanceExpiry: req.AcceptanceExpiry,
	}
	if item.BusinessServices == nil {
		item.BusinessServices = []string{}
	}
	if item.Controls == nil {
		item.Controls = []string{}
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.Status == "" {
		item.Status = "open"
	}
	if err := s.repo.CreateRisk(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) GetRisk(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISORiskEntry, error) {
	return s.repo.GetRisk(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdateRisk(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateRiskRequest) (*model.VCISORiskEntry, error) {
	if err := s.repo.UpdateRisk(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetRisk(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) DeleteRisk(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteRisk(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) RiskStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISORiskStats, error) {
	return s.repo.RiskStats(ctx, tenantID)
}

// ─── Policies ───────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListPolicies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListPolicies(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOPolicy{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreatePolicy(ctx context.Context, tenantID uuid.UUID, req *dto.CreatePolicyRequest) (*model.VCISOPolicy, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	ownerID, _ := uuid.Parse(req.OwnerID)
	item := &model.VCISOPolicy{
		Title: req.Title, Domain: req.Domain, Version: req.Version, Status: req.Status,
		Content: req.Content, OwnerID: ownerID, OwnerName: req.OwnerName,
		ReviewDue: req.ReviewDue, Tags: req.Tags,
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	if err := s.repo.CreatePolicy(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) GetPolicy(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOPolicy, error) {
	return s.repo.GetPolicy(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdatePolicy(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePolicyRequest) (*model.VCISOPolicy, error) {
	if err := s.repo.UpdatePolicy(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetPolicy(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) DeletePolicy(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeletePolicy(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdatePolicyStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdatePolicyStatusRequest) (*model.VCISOPolicy, error) {
	if err := s.repo.UpdatePolicyStatus(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetPolicy(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) PolicyStats(ctx context.Context, tenantID uuid.UUID) (*dto.GovernanceListResponse, error) {
	return s.repo.PolicyStats(ctx, tenantID)
}

func (s *VCISOGovernanceService) GeneratePolicy(ctx context.Context, tenantID uuid.UUID, domain string) (string, error) {
	if domain == "" {
		domain = "Information Security"
	}
	return s.repo.GeneratePolicyContent(ctx, tenantID, domain)
}

// ─── Policy Exceptions ──────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListPolicyExceptions(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListPolicyExceptions(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOPolicyException{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreatePolicyException(ctx context.Context, tenantID, userID uuid.UUID, req *dto.CreatePolicyExceptionRequest, userName string) (*model.VCISOPolicyException, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	policyID, err := uuid.Parse(req.PolicyID)
	if err != nil {
		return nil, fmt.Errorf("invalid policy_id: %w", err)
	}
	item := &model.VCISOPolicyException{
		PolicyID: policyID, PolicyTitle: "", Title: req.Title,
		Description: req.Description, Justification: req.Justification,
		CompensatingControls: req.CompensatingControls, ExpiresAt: req.ExpiresAt,
		RequestedByName: userName,
	}
	if err := s.repo.CreatePolicyException(ctx, tenantID, userID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) DecidePolicyException(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.DecidePolicyExceptionRequest, userName string) error {
	return s.repo.DecidePolicyException(ctx, tenantID, id, userID, req, userName)
}

// ─── Vendors ────────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListVendors(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListVendors(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOVendor{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateVendor(ctx context.Context, tenantID uuid.UUID, req *dto.CreateVendorRequest) (*model.VCISOVendor, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	item := &model.VCISOVendor{
		Name: req.Name, Category: req.Category, RiskTier: req.RiskTier,
		Status: req.Status, RiskScore: req.RiskScore, NextReviewDate: req.NextReviewDate,
		ContactName: req.ContactName, ContactEmail: req.ContactEmail,
		ServicesProvided: req.ServicesProvided, DataShared: req.DataShared,
		ComplianceFrameworks: req.ComplianceFrameworks,
		ControlsMet: req.ControlsMet, ControlsTotal: req.ControlsTotal,
	}
	if item.ServicesProvided == nil {
		item.ServicesProvided = []string{}
	}
	if item.DataShared == nil {
		item.DataShared = []string{}
	}
	if item.ComplianceFrameworks == nil {
		item.ComplianceFrameworks = []string{}
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := s.repo.CreateVendor(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) GetVendor(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOVendor, error) {
	return s.repo.GetVendor(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdateVendor(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateVendorRequest) (*model.VCISOVendor, error) {
	if err := s.repo.UpdateVendor(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetVendor(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) DeleteVendor(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteVendor(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdateVendorStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateVendorStatusRequest) (*model.VCISOVendor, error) {
	if err := s.repo.UpdateVendorStatus(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetVendor(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) VendorStats(ctx context.Context, tenantID uuid.UUID) (*dto.VendorStatsResponse, error) {
	return s.repo.VendorStats(ctx, tenantID)
}

// ─── Questionnaires ─────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListQuestionnaires(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListQuestionnaires(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOQuestionnaire{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateQuestionnaire(ctx context.Context, tenantID uuid.UUID, req *dto.CreateQuestionnaireRequest) (*model.VCISOQuestionnaire, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	item := &model.VCISOQuestionnaire{
		Title: req.Title, Type: req.Type, Status: req.Status,
		VendorID: dto.ParseOptionalUUID(req.VendorID), VendorName: req.VendorName,
		TotalQuestions: req.TotalQuestions, DueDate: req.DueDate,
		AssignedTo: dto.ParseOptionalUUID(req.AssignedTo), AssignedToName: req.AssignedToName,
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	if err := s.repo.CreateQuestionnaire(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateQuestionnaire(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateQuestionnaireRequest) error {
	return s.repo.UpdateQuestionnaire(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) UpdateQuestionnaireStatus(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateQuestionnaireStatusRequest) error {
	if req.Status == "completed" && req.CompletedAt == nil {
		now := time.Now().UTC().Format(time.RFC3339)
		req.CompletedAt = &now
	}
	return s.repo.UpdateQuestionnaireStatus(ctx, tenantID, id, req)
}

// ─── Evidence ───────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListEvidence(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListEvidence(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOEvidence{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateEvidence(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEvidenceRequest) (*model.VCISOEvidence, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	collectedAt, _ := time.Parse(time.RFC3339, req.CollectedAt)
	if collectedAt.IsZero() {
		collectedAt = time.Now().UTC()
	}
	var expiresAt *time.Time
	if req.ExpiresAt != nil {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err == nil {
			expiresAt = &t
		}
	}
	item := &model.VCISOEvidence{
		Title: req.Title, Description: req.Description, Type: req.Type,
		Source: req.Source, Status: "pending",
		Frameworks: req.Frameworks, ControlIDs: req.ControlIDs,
		FileName: req.FileName, FileSize: req.FileSize, FileURL: req.FileURL,
		CollectedAt: collectedAt, ExpiresAt: expiresAt, CollectorName: req.CollectorName,
	}
	if item.Frameworks == nil {
		item.Frameworks = []string{}
	}
	if item.ControlIDs == nil {
		item.ControlIDs = []string{}
	}
	if err := s.repo.CreateEvidence(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) GetEvidence(ctx context.Context, tenantID, id uuid.UUID) (*model.VCISOEvidence, error) {
	return s.repo.GetEvidence(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) UpdateEvidence(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEvidenceRequest) (*model.VCISOEvidence, error) {
	if err := s.repo.UpdateEvidence(ctx, tenantID, id, req); err != nil {
		return nil, err
	}
	return s.repo.GetEvidence(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) DeleteEvidence(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteEvidence(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) VerifyEvidence(ctx context.Context, tenantID, id, userID uuid.UUID, status string) (*model.VCISOEvidence, error) {
	if err := s.repo.VerifyEvidence(ctx, tenantID, id, userID, status); err != nil {
		return nil, err
	}
	return s.repo.GetEvidence(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) EvidenceStats(ctx context.Context, tenantID uuid.UUID) (*model.VCISOEvidenceStats, error) {
	return s.repo.EvidenceStats(ctx, tenantID)
}

// ─── Maturity ───────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListMaturityAssessments(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListMaturityAssessments(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOMaturityAssessment{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateMaturityAssessment(ctx context.Context, tenantID uuid.UUID, req *dto.CreateMaturityAssessmentRequest) (*model.VCISOMaturityAssessment, error) {
	if req.Framework == "" {
		return nil, fmt.Errorf("framework is required")
	}
	assessedAt, _ := time.Parse(time.RFC3339, req.AssessedAt)
	if assessedAt.IsZero() {
		assessedAt = time.Now().UTC()
	}
	dims := make([]model.VCISOMaturityDimension, len(req.Dimensions))
	for idx, d := range req.Dimensions {
		dims[idx] = model.VCISOMaturityDimension{
			Name: d.Name, Category: d.Category,
			CurrentLevel: d.CurrentLevel, TargetLevel: d.TargetLevel, Score: d.Score,
			Findings: d.Findings, Recommendations: d.Recommendations,
		}
	}
	item := &model.VCISOMaturityAssessment{
		Framework: req.Framework, Status: req.Status,
		OverallScore: req.OverallScore, OverallLevel: req.OverallLevel,
		Dimensions: dims, AssessorName: req.AssessorName, AssessedAt: assessedAt,
	}
	if item.Status == "" {
		item.Status = "completed"
	}
	if err := s.repo.CreateMaturityAssessment(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

// ─── Benchmarks ─────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListBenchmarks(ctx context.Context, tenantID uuid.UUID, params *dto.BenchmarkListParams) ([]model.VCISOBenchmark, error) {
	return s.repo.ListBenchmarks(ctx, tenantID, params)
}

// ─── Budget ─────────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListBudgetItems(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListBudgetItems(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOBudgetItem{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateBudgetItem(ctx context.Context, tenantID uuid.UUID, req *dto.CreateBudgetItemRequest) (*model.VCISOBudgetItem, error) {
	if req.Title == "" {
		return nil, fmt.Errorf("title is required")
	}
	item := &model.VCISOBudgetItem{
		Title: req.Title, Category: req.Category, Type: req.Type,
		Amount: req.Amount, Currency: req.Currency, Status: req.Status,
		RiskReductionEstimate: req.RiskReductionEstimate, Priority: req.Priority,
		Justification: req.Justification, LinkedRiskIDs: req.LinkedRiskIDs,
		LinkedRecommendationIDs: req.LinkedRecommendationIDs,
		FiscalYear: req.FiscalYear, Quarter: req.Quarter, OwnerName: req.OwnerName,
	}
	if item.LinkedRiskIDs == nil {
		item.LinkedRiskIDs = []string{}
	}
	if item.LinkedRecommendationIDs == nil {
		item.LinkedRecommendationIDs = []string{}
	}
	if item.Currency == "" {
		item.Currency = "USD"
	}
	if item.Status == "" {
		item.Status = "proposed"
	}
	if err := s.repo.CreateBudgetItem(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateBudgetItem(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateBudgetItemRequest) error {
	return s.repo.UpdateBudgetItem(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) DeleteBudgetItem(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteBudgetItem(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) BudgetSummary(ctx context.Context, tenantID uuid.UUID) (*dto.BudgetSummaryResponse, error) {
	return s.repo.BudgetSummary(ctx, tenantID)
}

// ─── Awareness ──────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListAwarenessPrograms(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListAwarenessPrograms(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOAwarenessProgram{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateAwarenessProgram(ctx context.Context, tenantID uuid.UUID, req *dto.CreateAwarenessProgramRequest) (*model.VCISOAwarenessProgram, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	item := &model.VCISOAwarenessProgram{
		Name: req.Name, Type: req.Type, Status: req.Status,
		TotalUsers: req.TotalUsers, CompletedUsers: req.CompletedUsers,
		PassedUsers: req.PassedUsers, FailedUsers: req.FailedUsers,
		StartDate: req.StartDate, EndDate: req.EndDate,
	}
	if item.Status == "" {
		item.Status = "planned"
	}
	if err := s.repo.CreateAwarenessProgram(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateAwarenessProgram(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateAwarenessProgramRequest) error {
	return s.repo.UpdateAwarenessProgram(ctx, tenantID, id, req)
}

// ─── IAM Findings ───────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListIAMFindings(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListIAMFindings(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOIAMFinding{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) UpdateIAMFinding(ctx context.Context, tenantID, id uuid.UUID, req *dto.UpdateIAMFindingRequest) error {
	return s.repo.UpdateIAMFinding(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) IAMFindingSummary(ctx context.Context, tenantID uuid.UUID) (*model.VCISOIAMSummary, error) {
	return s.repo.IAMFindingSummary(ctx, tenantID)
}

// ─── Escalation Rules ───────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListEscalationRules(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListEscalationRules(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOEscalationRule{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateEscalationRule(ctx context.Context, tenantID uuid.UUID, req *dto.CreateEscalationRuleRequest) (*model.VCISOEscalationRule, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	item := &model.VCISOEscalationRule{
		Name: req.Name, Description: req.Description,
		TriggerType: req.TriggerType, TriggerCondition: req.TriggerCondition,
		EscalationTarget: req.EscalationTarget, TargetContacts: req.TargetContacts,
		NotificationChannels: req.NotificationChannels, Enabled: req.Enabled,
	}
	if item.TargetContacts == nil {
		item.TargetContacts = []string{}
	}
	if item.NotificationChannels == nil {
		item.NotificationChannels = []string{}
	}
	if err := s.repo.CreateEscalationRule(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateEscalationRule(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateEscalationRuleRequest) error {
	return s.repo.UpdateEscalationRule(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) DeleteEscalationRule(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteEscalationRule(ctx, tenantID, id)
}

// ─── Playbooks ──────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListPlaybooks(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListPlaybooks(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOPlaybook{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreatePlaybook(ctx context.Context, tenantID uuid.UUID, req *dto.CreatePlaybookRequest) (*model.VCISOPlaybook, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	ownerID, _ := uuid.Parse(req.OwnerID)
	item := &model.VCISOPlaybook{
		Name: req.Name, Scenario: req.Scenario, Status: req.Status,
		NextTestDate: req.NextTestDate, OwnerID: ownerID, OwnerName: req.OwnerName,
		StepsCount: req.StepsCount, Dependencies: req.Dependencies,
		RTOHours: req.RTOHours, RPOHours: req.RPOHours,
	}
	if item.Dependencies == nil {
		item.Dependencies = []string{}
	}
	if item.Status == "" {
		item.Status = "draft"
	}
	if err := s.repo.CreatePlaybook(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdatePlaybook(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreatePlaybookRequest) error {
	return s.repo.UpdatePlaybook(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) DeletePlaybook(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeletePlaybook(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) SimulatePlaybook(ctx context.Context, tenantID, id uuid.UUID, result string) error {
	return s.repo.SimulatePlaybook(ctx, tenantID, id, result)
}

// ─── Obligations ────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListObligations(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListObligations(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISORegulatoryObligation{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateObligation(ctx context.Context, tenantID uuid.UUID, req *dto.CreateObligationRequest) (*model.VCISORegulatoryObligation, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	item := &model.VCISORegulatoryObligation{
		Name: req.Name, Type: req.Type, Jurisdiction: req.Jurisdiction,
		Description: req.Description, Requirements: req.Requirements, Status: req.Status,
		MappedControls: req.MappedControls, TotalRequirements: req.TotalRequirements,
		MetRequirements: req.MetRequirements, OwnerID: dto.ParseOptionalUUID(req.OwnerID),
		OwnerName: req.OwnerName, EffectiveDate: req.EffectiveDate, ReviewDate: req.ReviewDate,
	}
	if item.Requirements == nil {
		item.Requirements = []string{}
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := s.repo.CreateObligation(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateObligation(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateObligationRequest) error {
	return s.repo.UpdateObligation(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) DeleteObligation(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteObligation(ctx, tenantID, id)
}

// ─── Control Tests ──────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListControlTests(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListControlTests(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOControlTest{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateControlTest(ctx context.Context, tenantID uuid.UUID, req *dto.CreateControlTestRequest) (*model.VCISOControlTest, error) {
	if req.ControlID == "" {
		return nil, fmt.Errorf("control_id is required")
	}
	item := &model.VCISOControlTest{
		ControlID: req.ControlID, ControlName: req.ControlName, Framework: req.Framework,
		TestType: req.TestType, Result: req.Result, TesterName: req.TesterName,
		TestDate: req.TestDate, NextTestDate: req.NextTestDate, Findings: req.Findings,
		EvidenceIDs: req.EvidenceIDs,
	}
	if item.EvidenceIDs == nil {
		item.EvidenceIDs = []string{}
	}
	if err := s.repo.CreateControlTest(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

// ─── Control Dependencies ───────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListControlDependencies(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListControlDependencies(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

// ─── Integrations ───────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListIntegrations(ctx context.Context, tenantID uuid.UUID) ([]*model.VCISOIntegration, error) {
	return s.repo.ListIntegrations(ctx, tenantID)
}

func (s *VCISOGovernanceService) CreateIntegration(ctx context.Context, tenantID uuid.UUID, req *dto.CreateIntegrationRequest) (*model.VCISOIntegration, error) {
	if req.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	item := &model.VCISOIntegration{
		Name: req.Name, Type: req.Type, Provider: req.Provider,
		Status: req.Status, SyncFrequency: req.SyncFrequency,
		Config: req.Config,
	}
	if item.Config == nil {
		item.Config = make(map[string]interface{})
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := s.repo.CreateIntegration(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateIntegration(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateIntegrationRequest) error {
	return s.repo.UpdateIntegration(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) DeleteIntegration(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.DeleteIntegration(ctx, tenantID, id)
}

func (s *VCISOGovernanceService) SyncIntegration(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.SyncIntegration(ctx, tenantID, id)
}

// ─── Control Ownership ──────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListControlOwnerships(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListControlOwnerships(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOControlOwnership{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) CreateControlOwnership(ctx context.Context, tenantID uuid.UUID, req *dto.CreateControlOwnershipRequest) (*model.VCISOControlOwnership, error) {
	if req.ControlID == "" {
		return nil, fmt.Errorf("control_id is required")
	}
	ownerID, _ := uuid.Parse(req.OwnerID)
	item := &model.VCISOControlOwnership{
		ControlID: req.ControlID, ControlName: req.ControlName, Framework: req.Framework,
		OwnerID: ownerID, OwnerName: req.OwnerName,
		DelegateID: dto.ParseOptionalUUID(req.DelegateID), DelegateName: req.DelegateName,
		Status: req.Status, NextReviewDate: req.NextReviewDate,
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := s.repo.CreateControlOwnership(ctx, tenantID, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *VCISOGovernanceService) UpdateControlOwnership(ctx context.Context, tenantID, id uuid.UUID, req *dto.CreateControlOwnershipRequest) error {
	return s.repo.UpdateControlOwnership(ctx, tenantID, id, req)
}

func (s *VCISOGovernanceService) MarkControlOwnershipReviewed(ctx context.Context, tenantID, id uuid.UUID) error {
	return s.repo.MarkControlOwnershipReviewed(ctx, tenantID, id)
}

// ─── Approvals ──────────────────────────────────────────────────────────────

func (s *VCISOGovernanceService) ListApprovals(ctx context.Context, tenantID uuid.UUID, params *dto.VCISOGovernanceListParams) (*dto.GovernanceListResponse, error) {
	params.SetDefaults()
	items, total, err := s.repo.ListApprovals(ctx, tenantID, params)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*model.VCISOApprovalRequest{}
	}
	return dto.NewGovernanceListResponse(items, params.Page, params.PerPage, total), nil
}

func (s *VCISOGovernanceService) DecideApproval(ctx context.Context, tenantID, id, userID uuid.UUID, req *dto.UpdateApprovalRequest) error {
	return s.repo.DecideApproval(ctx, tenantID, id, userID, req)
}
