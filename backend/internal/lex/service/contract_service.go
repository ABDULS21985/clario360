package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/metrics"
	"github.com/clario360/platform/internal/lex/model"
	"github.com/clario360/platform/internal/lex/repository"
)

var allowedContractTypes = map[model.ContractType]struct{}{
	model.ContractTypeServiceAgreement: {},
	model.ContractTypeNDA:              {},
	model.ContractTypeEmployment:       {},
	model.ContractTypeVendor:           {},
	model.ContractTypeLicense:          {},
	model.ContractTypeLease:            {},
	model.ContractTypePartnership:      {},
	model.ContractTypeConsulting:       {},
	model.ContractTypeProcurement:      {},
	model.ContractTypeSLA:              {},
	model.ContractTypeMOU:              {},
	model.ContractTypeAmendment:        {},
	model.ContractTypeRenewal:          {},
	model.ContractTypeOther:            {},
}

var validTransitions = map[model.ContractStatus]map[model.ContractStatus]struct{}{
	model.ContractStatusDraft: {
		model.ContractStatusInternalReview: {},
		model.ContractStatusCancelled:      {},
	},
	model.ContractStatusInternalReview: {
		model.ContractStatusLegalReview: {},
		model.ContractStatusDraft:       {},
	},
	model.ContractStatusLegalReview: {
		model.ContractStatusNegotiation:    {},
		model.ContractStatusInternalReview: {},
		model.ContractStatusDraft:          {},
	},
	model.ContractStatusNegotiation: {
		model.ContractStatusPendingSignature: {},
		model.ContractStatusCancelled:        {},
		model.ContractStatusDraft:            {},
	},
	model.ContractStatusPendingSignature: {
		model.ContractStatusActive:    {},
		model.ContractStatusCancelled: {},
	},
	model.ContractStatusActive: {
		model.ContractStatusSuspended:  {},
		model.ContractStatusTerminated: {},
		model.ContractStatusExpired:    {},
		model.ContractStatusRenewed:    {},
	},
	model.ContractStatusSuspended: {
		model.ContractStatusActive:     {},
		model.ContractStatusTerminated: {},
	},
	model.ContractStatusExpired: {
		model.ContractStatusRenewed: {},
	},
}

type ContractService struct {
	db         *pgxpool.Pool
	contracts  *repository.ContractRepository
	clauses    *repository.ClauseRepository
	documents  *repository.DocumentRepository
	compliance *repository.ComplianceRepository
	alerts     *repository.AlertRepository
	workflow   *WorkflowService
	analyzer   interface {
		AnalyzeDetailed(contract *model.Contract, text string) (*model.AnalysisResult, error)
	}
	publisher Publisher
	metrics   *metrics.Metrics
	topic     string
	logger    zerolog.Logger
	now       func() time.Time
	predictionLogger *aigovmiddleware.PredictionLogger
}

func NewContractService(
	db *pgxpool.Pool,
	contracts *repository.ContractRepository,
	clauses *repository.ClauseRepository,
	documents *repository.DocumentRepository,
	compliance *repository.ComplianceRepository,
	alerts *repository.AlertRepository,
	workflow *WorkflowService,
	analyzer interface {
		AnalyzeDetailed(contract *model.Contract, text string) (*model.AnalysisResult, error)
	},
	publisher Publisher,
	appMetrics *metrics.Metrics,
	topic string,
	logger zerolog.Logger,
	predictionLogger *aigovmiddleware.PredictionLogger,
) *ContractService {
	return &ContractService{
		db:         db,
		contracts:  contracts,
		clauses:    clauses,
		documents:  documents,
		compliance: compliance,
		alerts:     alerts,
		workflow:   workflow,
		analyzer:   analyzer,
		publisher:  publisherOrNoop(publisher),
		metrics:    appMetrics,
		topic:      topic,
		logger:     logger.With().Str("service", "lex-contracts").Logger(),
		now:        time.Now,
		predictionLogger: predictionLogger,
	}
}

func ValidateContractTransition(currentStatus, newStatus string) error {
	current := model.ContractStatus(strings.TrimSpace(currentStatus))
	next := model.ContractStatus(strings.TrimSpace(newStatus))
	allowed, ok := validTransitions[current]
	if !ok {
		return fmt.Errorf("unsupported current status %q", currentStatus)
	}
	if _, ok := allowed[next]; !ok {
		return fmt.Errorf("invalid contract transition from %s to %s", current, next)
	}
	return nil
}

func (s *ContractService) CreateContract(ctx context.Context, tenantID, userID uuid.UUID, req dto.CreateContractRequest) (*model.Contract, error) {
	req.Normalize()
	if err := validateContractCreate(req); err != nil {
		return nil, err
	}

	contractNumber := normalizeOptionalString(req.ContractNumber)
	if contractNumber == nil {
		generated := fmt.Sprintf("LEX-%s-%s", s.now().UTC().Format("20060102"), strings.ToUpper(uuid.NewString()[:8]))
		contractNumber = &generated
	}

	contract := &model.Contract{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Title:             req.Title,
		ContractNumber:    contractNumber,
		Type:              req.Type,
		Description:       req.Description,
		PartyAName:        req.PartyAName,
		PartyAEntity:      normalizeOptionalString(req.PartyAEntity),
		PartyBName:        req.PartyBName,
		PartyBEntity:      normalizeOptionalString(req.PartyBEntity),
		PartyBContact:     normalizeOptionalString(req.PartyBContact),
		TotalValue:        req.TotalValue,
		Currency:          req.Currency,
		PaymentTerms:      normalizeOptionalString(req.PaymentTerms),
		EffectiveDate:     req.EffectiveDate,
		ExpiryDate:        req.ExpiryDate,
		RenewalDate:       req.RenewalDate,
		AutoRenew:         req.AutoRenew,
		RenewalNoticeDays: req.RenewalNoticeDays,
		Status:            model.ContractStatusDraft,
		OwnerUserID:       req.OwnerUserID,
		OwnerName:         req.OwnerName,
		LegalReviewerID:   req.LegalReviewerID,
		LegalReviewerName: normalizeOptionalString(req.LegalReviewerName),
		RiskLevel:         model.RiskLevelNone,
		AnalysisStatus:    model.AnalysisStatusPending,
		CurrentVersion:    1,
		Department:        normalizeOptionalString(req.Department),
		Tags:              req.Tags,
		Metadata:          req.Metadata,
		CreatedBy:         userID,
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start contract transaction", err)
	}
	defer tx.Rollback(ctx)

	if err := s.contracts.Create(ctx, tx, contract); err != nil {
		return nil, internalError("create contract", err)
	}
	if req.Document != nil {
		version := &model.ContractVersion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ContractID:    contract.ID,
			Version:       1,
			FileID:        req.Document.FileID,
			FileName:      req.Document.FileName,
			FileSizeBytes: req.Document.FileSizeBytes,
			ContentHash:   req.Document.ContentHash,
			ExtractedText: &req.Document.ExtractedText,
			ChangeSummary: normalizeOptionalString(&req.Document.ChangeSummary),
			UploadedBy:    userID,
		}
		if err := s.contracts.InsertVersion(ctx, tx, version); err != nil {
			return nil, internalError("create contract version", err)
		}
		if err := s.contracts.UpdateDocument(ctx, tx, tenantID, contract.ID, req.Document.FileID, req.Document.ExtractedText, 1); err != nil {
			return nil, internalError("attach contract document", err)
		}
		contract.DocumentFileID = &req.Document.FileID
		contract.DocumentText = req.Document.ExtractedText
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit contract create", err)
	}

	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.created", tenantID, &userID, map[string]any{
		"id":                contract.ID,
		"title":             contract.Title,
		"type":              contract.Type,
		"party_b_name":      contract.PartyBName,
		"value":             contract.TotalValue,
		"owner_user_id":     contract.OwnerUserID,
		"legal_reviewer_id": contract.LegalReviewerID,
		"created_by":        contract.CreatedBy,
	}, s.logger)
	return contract, nil
}

func (s *ContractService) ListContracts(ctx context.Context, tenantID uuid.UUID, filters model.ContractListFilters) ([]model.Contract, int, error) {
	return s.contracts.List(ctx, tenantID, filters)
}

func (s *ContractService) GetContract(ctx context.Context, tenantID, id uuid.UUID) (*model.ContractDetail, error) {
	contract, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	clauses, err := s.clauses.ListByContract(ctx, tenantID, id)
	if err != nil {
		return nil, internalError("load clauses", err)
	}
	analysis, err := s.contracts.GetLatestAnalysis(ctx, tenantID, id)
	if err != nil && err != pgx.ErrNoRows {
		return nil, internalError("load analysis", err)
	}
	versions, err := s.contracts.ListVersions(ctx, tenantID, id)
	if err != nil {
		return nil, internalError("load versions", err)
	}
	return &model.ContractDetail{
		Contract:       contract,
		Clauses:        clauses,
		LatestAnalysis: analysis,
		VersionCount:   len(versions),
	}, nil
}

func (s *ContractService) UpdateContract(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UpdateContractRequest) (*model.Contract, error) {
	contract, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	before := map[string]any{
		"title":       contract.Title,
		"type":        contract.Type,
		"description": contract.Description,
		"owner_name":  contract.OwnerName,
		"status":      contract.Status,
	}
	applyContractUpdate(contract, req)
	if err := validateContractForUpdate(contract); err != nil {
		return nil, err
	}
	if err := s.contracts.Update(ctx, s.db, contract); err != nil {
		return nil, internalError("update contract", err)
	}
	after := map[string]any{
		"title":       contract.Title,
		"type":        contract.Type,
		"description": contract.Description,
		"owner_name":  contract.OwnerName,
		"status":      contract.Status,
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.updated", tenantID, &userID, map[string]any{
		"id":             contract.ID,
		"changed_fields": changedFields(before, after),
	}, s.logger)
	return contract, nil
}

func (s *ContractService) DeleteContract(ctx context.Context, tenantID uuid.UUID, id uuid.UUID) error {
	if err := s.contracts.SoftDelete(ctx, tenantID, id); err != nil {
		if err == pgx.ErrNoRows {
			return notFoundError("contract not found")
		}
		return internalError("delete contract", err)
	}
	return nil
}

func (s *ContractService) UploadDocument(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.UploadContractDocumentRequest) ([]model.ContractVersion, error) {
	contract, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	if req.FileID == uuid.Nil || strings.TrimSpace(req.ContentHash) == "" || strings.TrimSpace(req.FileName) == "" {
		return nil, validationError("file_id, file_name, and content_hash are required", map[string]string{
			"file_id":      "required",
			"file_name":    "required",
			"content_hash": "required",
		})
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start upload transaction", err)
	}
	defer tx.Rollback(ctx)

	version := &model.ContractVersion{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ContractID:    contract.ID,
		Version:       contract.CurrentVersion + 1,
		FileID:        req.FileID,
		FileName:      req.FileName,
		FileSizeBytes: req.FileSizeBytes,
		ContentHash:   req.ContentHash,
		ExtractedText: &req.ExtractedText,
		ChangeSummary: normalizeOptionalString(&req.ChangeSummary),
		UploadedBy:    userID,
	}
	if err := s.contracts.InsertVersion(ctx, tx, version); err != nil {
		return nil, internalError("insert contract version", err)
	}
	if err := s.contracts.UpdateDocument(ctx, tx, tenantID, contract.ID, req.FileID, req.ExtractedText, version.Version); err != nil {
		return nil, internalError("update current contract document", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit upload transaction", err)
	}

	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.document_uploaded", tenantID, &userID, map[string]any{
		"id":           contract.ID,
		"version":      version.Version,
		"file_id":      req.FileID,
		"content_hash": req.ContentHash,
	}, s.logger)
	return s.contracts.ListVersions(ctx, tenantID, contract.ID)
}

func (s *ContractService) AnalyzeContract(ctx context.Context, tenantID, id uuid.UUID) (*model.AnalysisResult, error) {
	contract, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	if strings.TrimSpace(contract.DocumentText) == "" {
		return nil, validationError("contract document text is required for analysis", map[string]string{"document_text": "missing"})
	}
	if err := s.contracts.SetAnalysisStatus(ctx, tenantID, id, model.AnalysisStatusAnalyzing); err != nil {
		return nil, internalError("mark contract analyzing", err)
	}

	result, err := s.analyzer.AnalyzeDetailed(contract, contract.DocumentText)
	if err != nil {
		_ = s.contracts.SetAnalysisStatus(ctx, tenantID, id, model.AnalysisStatusFailed)
		return nil, internalError("analyze contract", err)
	}
	s.recordGovernedPredictions(ctx, contract, result)

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start analysis transaction", err)
	}
	defer tx.Rollback(ctx)
	if err := s.contracts.InsertAnalysis(ctx, tx, result.Analysis); err != nil {
		return nil, internalError("store contract analysis", err)
	}
	if err := s.clauses.ReplaceForContract(ctx, tx, tenantID, id, result.Clauses); err != nil {
		return nil, internalError("store extracted clauses", err)
	}
	if err := s.contracts.UpdateAnalysisFields(ctx, tx, tenantID, id, result.Analysis.RiskScore, result.Analysis.OverallRisk, model.AnalysisStatusCompleted, result.Analysis.AnalyzedAt); err != nil {
		return nil, internalError("update contract risk fields", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit analysis transaction", err)
	}

	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.analyzed", tenantID, nil, map[string]any{
		"id":                contract.ID,
		"risk_level":        result.Analysis.OverallRisk,
		"risk_score":        result.Analysis.RiskScore,
		"clause_count":      result.Analysis.ClauseCount,
		"missing_count":     len(result.Analysis.MissingClauses),
		"created_by":        contract.CreatedBy,
		"owner_user_id":     contract.OwnerUserID,
		"legal_reviewer_id": contract.LegalReviewerID,
	}, s.logger)
	for _, clause := range result.Clauses {
		if clause.RiskLevel != model.RiskLevelCritical && clause.RiskLevel != model.RiskLevelHigh {
			continue
		}
		writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.clause.risk_flagged", tenantID, nil, map[string]any{
			"contract_id":   contract.ID,
			"clause_type":   clause.ClauseType,
			"risk_level":    clause.RiskLevel,
			"section_ref":   clause.SectionReference,
			"risk_keywords": clause.RiskKeywords,
			"owner_user_id": contract.OwnerUserID,
		}, s.logger)
	}
	return result, nil
}

func (s *ContractService) recordGovernedPredictions(ctx context.Context, contract *model.Contract, result *model.AnalysisResult) {
	if s.predictionLogger == nil || contract == nil || result == nil || result.Analysis == nil {
		return
	}
	contractID := contract.ID
	input := map[string]any{
		"contract_id":      contract.ID.String(),
		"contract_type":    contract.Type,
		"document_length":  len(contract.DocumentText),
		"current_version":  contract.CurrentVersion,
	}
	_, _ = s.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     contract.TenantID,
		ModelSlug:    "lex-clause-extractor",
		UseCase:      "clause_extraction",
		EntityType:   "contract",
		EntityID:     &contractID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     result.Clauses,
				Confidence: clauseExtractionConfidence(result.Clauses),
				Metadata: map[string]any{
					"matched_rules": clauseTypes(result.Clauses),
					"clause_count":  len(result.Clauses),
					"high_risk":     result.Analysis.HighRiskClauseCount,
				},
			}, nil
		},
	})
	componentScores, componentWeights := lexRiskComponents(contract, result.Analysis)
	_, _ = s.predictionLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:     contract.TenantID,
		ModelSlug:    "lex-risk-analyzer",
		UseCase:      "contract_risk_analysis",
		EntityType:   "contract",
		EntityID:     &contractID,
		Input:        input,
		InputSummary: input,
		ModelFunc: func(context.Context, any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output:     result.Analysis,
				Confidence: riskAnalysisConfidence(result.Analysis),
				Metadata: map[string]any{
					"component_scores":  componentScores,
					"component_weights": componentWeights,
					"overall_score":     result.Analysis.RiskScore,
				},
			}, nil
		},
	})
}

func clauseExtractionConfidence(clauses []model.ExtractedClause) float64 {
	if len(clauses) == 0 {
		return 0.65
	}
	total := 0.0
	for _, item := range clauses {
		total += item.ExtractionConfidence
	}
	return total / float64(len(clauses))
}

func clauseTypes(clauses []model.ExtractedClause) []string {
	out := make([]string, 0, len(clauses))
	seen := make(map[string]struct{}, len(clauses))
	for _, item := range clauses {
		key := string(item.ClauseType)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	return out
}

func lexRiskComponents(contract *model.Contract, analysis *model.ContractRiskAnalysis) (map[string]any, map[string]any) {
	componentScores := map[string]any{
		"clause_risk":    float64(analysis.HighRiskClauseCount) * 10,
		"missing_clause": float64(len(analysis.MissingClauses) * 8),
		"compliance":     float64(len(analysis.ComplianceFlags) * 5),
	}
	valueFactor := 0.0
	if contract.TotalValue != nil {
		switch {
		case *contract.TotalValue > 10_000_000:
			valueFactor = 15
		case *contract.TotalValue > 1_000_000:
			valueFactor = 10
		}
	}
	componentScores["value"] = valueFactor
	expiryFactor := 0.0
	if contract.ExpiryDate != nil {
		days := int(contract.ExpiryDate.UTC().Sub(time.Now().UTC()).Hours() / 24)
		switch {
		case days <= 7:
			expiryFactor = 20
		case days <= 30:
			expiryFactor = 10
		}
	}
	componentScores["expiry"] = expiryFactor
	componentWeights := map[string]any{
		"clause_risk":    1.0,
		"missing_clause": 1.0,
		"value":          1.0,
		"expiry":         1.0,
		"compliance":     1.0,
	}
	return componentScores, componentWeights
}

func riskAnalysisConfidence(analysis *model.ContractRiskAnalysis) float64 {
	if analysis == nil {
		return 0.5
	}
	switch analysis.OverallRisk {
	case model.RiskLevelCritical:
		return 0.95
	case model.RiskLevelHigh:
		return 0.90
	case model.RiskLevelMedium:
		return 0.82
	default:
		return 0.75
	}
}

func (s *ContractService) GetAnalysis(ctx context.Context, tenantID, id uuid.UUID) (*model.ContractRiskAnalysis, error) {
	analysis, err := s.contracts.GetLatestAnalysis(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("analysis not found")
		}
		return nil, internalError("get analysis", err)
	}
	return analysis, nil
}

func (s *ContractService) UpdateStatus(ctx context.Context, tenantID, userID, id uuid.UUID, status model.ContractStatus) (*model.Contract, error) {
	contract, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	if err := ValidateContractTransition(string(contract.Status), string(status)); err != nil {
		return nil, validationError(err.Error(), map[string]string{"status": "invalid transition"})
	}
	prev := contract.Status
	now := s.now().UTC()
	var signedDate *time.Time
	if status == model.ContractStatusActive && contract.SignedDate == nil {
		value := normalizeDate(now)
		signedDate = &value
	}
	if err := s.contracts.UpdateStatus(ctx, s.db, tenantID, id, &prev, status, &userID, now, signedDate); err != nil {
		return nil, internalError("update contract status", err)
	}
	updated, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		return nil, internalError("reload contract", err)
	}
	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.status_changed", tenantID, &userID, map[string]any{
		"id":         updated.ID,
		"old_status": prev,
		"new_status": status,
		"changed_by": userID,
	}, s.logger)
	return updated, nil
}

func (s *ContractService) ListVersions(ctx context.Context, tenantID, id uuid.UUID) ([]model.ContractVersion, error) {
	return s.contracts.ListVersions(ctx, tenantID, id)
}

func (s *ContractService) RenewContract(ctx context.Context, tenantID, userID, id uuid.UUID, req dto.RenewContractRequest) (*model.Contract, error) {
	original, err := s.contracts.Get(ctx, tenantID, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, notFoundError("contract not found")
		}
		return nil, internalError("load contract", err)
	}
	if original.Status != model.ContractStatusActive && original.Status != model.ContractStatusExpired {
		return nil, validationError("only active or expired contracts can be renewed", map[string]string{"status": "invalid"})
	}
	startDate := req.NewEffectiveDate
	if startDate == nil {
		if original.ExpiryDate != nil {
			value := normalizeDate(original.ExpiryDate.AddDate(0, 0, 1))
			startDate = &value
		} else {
			value := normalizeDate(s.now())
			startDate = &value
		}
	}
	value := original.TotalValue
	if req.NewValue != nil {
		value = req.NewValue
	}

	contractNumber := fmt.Sprintf("LEX-RNW-%s", strings.ToUpper(uuid.NewString()[:8]))
	renewal := &model.Contract{
		ID:                uuid.New(),
		TenantID:          tenantID,
		Title:             original.Title + " (Renewal)",
		ContractNumber:    &contractNumber,
		Type:              original.Type,
		Description:       original.Description,
		PartyAName:        original.PartyAName,
		PartyAEntity:      original.PartyAEntity,
		PartyBName:        original.PartyBName,
		PartyBEntity:      original.PartyBEntity,
		PartyBContact:     original.PartyBContact,
		TotalValue:        value,
		Currency:          original.Currency,
		PaymentTerms:      original.PaymentTerms,
		EffectiveDate:     startDate,
		ExpiryDate:        &req.NewExpiryDate,
		RenewalDate:       nil,
		AutoRenew:         original.AutoRenew,
		RenewalNoticeDays: original.RenewalNoticeDays,
		Status:            model.ContractStatusDraft,
		OwnerUserID:       original.OwnerUserID,
		OwnerName:         original.OwnerName,
		LegalReviewerID:   original.LegalReviewerID,
		LegalReviewerName: original.LegalReviewerName,
		RiskLevel:         model.RiskLevelNone,
		AnalysisStatus:    model.AnalysisStatusPending,
		CurrentVersion:    1,
		ParentContractID:  &original.ID,
		Department:        original.Department,
		Tags:              append([]string(nil), original.Tags...),
		Metadata:          original.Metadata,
		CreatedBy:         userID,
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, internalError("start renewal transaction", err)
	}
	defer tx.Rollback(ctx)
	if err := s.contracts.Create(ctx, tx, renewal); err != nil {
		return nil, internalError("create renewal contract", err)
	}
	latestVersion, err := s.contracts.GetLatestVersion(ctx, tenantID, original.ID)
	if err != nil && err != pgx.ErrNoRows {
		return nil, internalError("load original version", err)
	}
	if latestVersion != nil {
		newVersion := &model.ContractVersion{
			ID:            uuid.New(),
			TenantID:      tenantID,
			ContractID:    renewal.ID,
			Version:       1,
			FileID:        latestVersion.FileID,
			FileName:      latestVersion.FileName,
			FileSizeBytes: latestVersion.FileSizeBytes,
			ContentHash:   latestVersion.ContentHash,
			ExtractedText: latestVersion.ExtractedText,
			ChangeSummary: normalizeOptionalString(&req.ChangeSummary),
			UploadedBy:    userID,
		}
		if err := s.contracts.InsertVersion(ctx, tx, newVersion); err != nil {
			return nil, internalError("copy renewal version", err)
		}
		text := ""
		if latestVersion.ExtractedText != nil {
			text = *latestVersion.ExtractedText
		}
		if err := s.contracts.UpdateDocument(ctx, tx, tenantID, renewal.ID, latestVersion.FileID, text, 1); err != nil {
			return nil, internalError("attach renewal document", err)
		}
	}
	prev := original.Status
	now := s.now().UTC()
	if err := s.contracts.UpdateStatus(ctx, tx, tenantID, original.ID, &prev, model.ContractStatusRenewed, &userID, now, nil); err != nil {
		return nil, internalError("mark original renewed", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, internalError("commit renewal", err)
	}

	writeEvent(ctx, s.publisher, "lex-service", s.topic, "com.clario360.lex.contract.renewed", tenantID, &userID, map[string]any{
		"original_id":      original.ID,
		"new_id":           renewal.ID,
		"new_expiry_date":  req.NewExpiryDate,
	}, s.logger)
	return renewal, nil
}

func (s *ContractService) ListExpiring(ctx context.Context, tenantID uuid.UUID, horizonDays int) ([]model.ExpiringContractSummary, error) {
	return s.contracts.ListExpiring(ctx, tenantID, horizonDays)
}

func (s *ContractService) Stats(ctx context.Context, tenantID uuid.UUID) (*model.ContractStats, error) {
	return s.contracts.Stats(ctx, tenantID)
}

func (s *ContractService) SearchContracts(ctx context.Context, tenantID uuid.UUID, query string, page, perPage int) ([]model.ContractSummary, int, error) {
	return s.contracts.Search(ctx, tenantID, query, page, perPage)
}

func validateContractCreate(req dto.CreateContractRequest) error {
	fields := map[string]string{}
	if req.Title == "" {
		fields["title"] = "required"
	}
	if _, ok := allowedContractTypes[req.Type]; !ok {
		fields["type"] = "invalid"
	}
	if req.PartyAName == "" {
		fields["party_a_name"] = "required"
	}
	if req.PartyBName == "" {
		fields["party_b_name"] = "required"
	}
	if req.OwnerUserID == uuid.Nil {
		fields["owner_user_id"] = "required"
	}
	if req.OwnerName == "" {
		fields["owner_name"] = "required"
	}
	if req.RenewalNoticeDays < 0 {
		fields["renewal_notice_days"] = "must be >= 0"
	}
	if len(fields) > 0 {
		return validationError("invalid contract request", fields)
	}
	return nil
}

func validateContractForUpdate(contract *model.Contract) error {
	if contract == nil {
		return validationError("contract is required", map[string]string{"contract": "required"})
	}
	if strings.TrimSpace(contract.Title) == "" {
		return validationError("title is required", map[string]string{"title": "required"})
	}
	if _, ok := allowedContractTypes[contract.Type]; !ok {
		return validationError("type is invalid", map[string]string{"type": "invalid"})
	}
	return nil
}

func applyContractUpdate(contract *model.Contract, req dto.UpdateContractRequest) {
	if req.Title != nil {
		contract.Title = strings.TrimSpace(*req.Title)
	}
	if req.ContractNumber != nil {
		contract.ContractNumber = normalizeOptionalString(req.ContractNumber)
	}
	if req.Type != nil {
		contract.Type = *req.Type
	}
	if req.Description != nil {
		contract.Description = strings.TrimSpace(*req.Description)
	}
	if req.PartyAName != nil {
		contract.PartyAName = strings.TrimSpace(*req.PartyAName)
	}
	if req.PartyAEntity != nil {
		contract.PartyAEntity = normalizeOptionalString(req.PartyAEntity)
	}
	if req.PartyBName != nil {
		contract.PartyBName = strings.TrimSpace(*req.PartyBName)
	}
	if req.PartyBEntity != nil {
		contract.PartyBEntity = normalizeOptionalString(req.PartyBEntity)
	}
	if req.PartyBContact != nil {
		contract.PartyBContact = normalizeOptionalString(req.PartyBContact)
	}
	if req.TotalValue != nil {
		contract.TotalValue = req.TotalValue
	}
	if req.Currency != nil {
		trimmed := strings.ToUpper(strings.TrimSpace(*req.Currency))
		if trimmed != "" {
			contract.Currency = trimmed
		}
	}
	if req.PaymentTerms != nil {
		contract.PaymentTerms = normalizeOptionalString(req.PaymentTerms)
	}
	if req.EffectiveDate != nil {
		contract.EffectiveDate = req.EffectiveDate
	}
	if req.ExpiryDate != nil {
		contract.ExpiryDate = req.ExpiryDate
	}
	if req.RenewalDate != nil {
		contract.RenewalDate = req.RenewalDate
	}
	if req.AutoRenew != nil {
		contract.AutoRenew = *req.AutoRenew
	}
	if req.RenewalNoticeDays != nil {
		contract.RenewalNoticeDays = *req.RenewalNoticeDays
	}
	if req.SignedDate != nil {
		contract.SignedDate = req.SignedDate
	}
	if req.OwnerUserID != nil {
		contract.OwnerUserID = *req.OwnerUserID
	}
	if req.OwnerName != nil {
		contract.OwnerName = strings.TrimSpace(*req.OwnerName)
	}
	if req.LegalReviewerID != nil {
		contract.LegalReviewerID = req.LegalReviewerID
	}
	if req.LegalReviewerName != nil {
		contract.LegalReviewerName = normalizeOptionalString(req.LegalReviewerName)
	}
	if req.Department != nil {
		contract.Department = normalizeOptionalString(req.Department)
	}
	if req.Tags != nil {
		contract.Tags = dto.NormalizeTags(req.Tags)
	}
	if req.Metadata != nil {
		contract.Metadata = req.Metadata
	}
}
