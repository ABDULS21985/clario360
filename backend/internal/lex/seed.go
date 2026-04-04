package lex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/lex/analyzer"
	"github.com/clario360/platform/internal/lex/dto"
	"github.com/clario360/platform/internal/lex/model"
)

var (
	seedTenantID    = mustUUID("22222222-2222-2222-2222-222222222222")
	seedSystemUser  = mustUUID("22222222-2222-2222-2222-222222222201")
	seedReferenceAt = time.Date(2026, time.March, 7, 9, 0, 0, 0, time.UTC)
)

type seedUser struct {
	ID    uuid.UUID
	Name  string
	Title string
}

type seedContractSpec struct {
	Title             string
	Type              model.ContractType
	Description       string
	PartyBName        string
	PartyBEntity      string
	PartyBContact     string
	Department        string
	Tags              []string
	TotalValue        float64
	Currency          string
	PaymentTerms      string
	EffectiveOffset   int
	ExpiryOffset      int
	RenewalNoticeDays int
	AutoRenew         bool
	Status            model.ContractStatus
	Owner             seedUser
	LegalReviewer     seedUser
	ContainsPII       bool
}

type seedClauseBlueprint struct {
	ClauseType   model.ClauseType
	Title        string
	Trigger      string
	SafeBody     string
	RiskKeywords []string
}

func SeedDemoData(ctx context.Context, app *Application, logger zerolog.Logger) (uuid.UUID, error) {
	if app == nil || app.Store == nil || app.Store.DB() == nil {
		return uuid.Nil, fmt.Errorf("lex application is not initialized")
	}
	existing, total, err := app.Store.Contracts.List(ctx, seedTenantID, model.ContractListFilters{Page: 1, PerPage: 1})
	if err != nil {
		return uuid.Nil, err
	}
	if total > 0 && len(existing) > 0 {
		return seedTenantID, nil
	}

	users := seedUsers()
	specs := seedContractSpecs(users)
	blueprints := seedClauseBlueprints()
	recommendations := analyzer.NewRecommendationEngine("Saudi Arabia")

	contracts := make([]*model.Contract, 0, len(specs))
	for idx, spec := range specs {
		contract, err := seedContract(ctx, app, spec, idx, blueprints, recommendations)
		if err != nil {
			return uuid.Nil, err
		}
		contracts = append(contracts, contract)
	}

	if err := applySeedClauseReviews(ctx, app, contracts, users[1].ID, seedReferenceAt); err != nil {
		return uuid.Nil, err
	}
	rules, err := seedComplianceRules(ctx, app, users[0].ID)
	if err != nil {
		return uuid.Nil, err
	}
	if err := seedLegalDocuments(ctx, app, contracts, users[0].ID); err != nil {
		return uuid.Nil, err
	}
	if err := seedComplianceAlerts(ctx, app, contracts, rules, users[1].ID); err != nil {
		return uuid.Nil, err
	}

	logger.Info().
		Str("tenant_id", seedTenantID.String()).
		Int("contracts", len(contracts)).
		Int("rules", len(rules)).
		Int("documents", 8).
		Int("alerts", 10).
		Msg("seeded lex demo dataset")

	return seedTenantID, nil
}

func seedContract(
	ctx context.Context,
	app *Application,
	spec seedContractSpec,
	index int,
	blueprints []seedClauseBlueprint,
	recommendations *analyzer.RecommendationEngine,
) (*model.Contract, error) {
	effectiveDate := normalizeSeedDate(seedReferenceAt.AddDate(0, 0, spec.EffectiveOffset))
	expiryDate := normalizeSeedDate(seedReferenceAt.AddDate(0, 0, spec.ExpiryOffset))
	var renewalDate *time.Time
	if spec.AutoRenew {
		date := normalizeSeedDate(expiryDate.AddDate(0, 0, -spec.RenewalNoticeDays))
		renewalDate = &date
	}

	baseContract := &model.Contract{
		ID:                uuid.NewSHA1(seedTenantID, []byte(fmt.Sprintf("seed-contract-%02d", index+1))),
		TenantID:          seedTenantID,
		Title:             spec.Title,
		Type:              spec.Type,
		Description:       spec.Description,
		PartyAName:        "Clario Holdings Limited",
		PartyBName:        spec.PartyBName,
		PartyBEntity:      ptrString(spec.PartyBEntity),
		PartyBContact:     ptrString(spec.PartyBContact),
		TotalValue:        ptrFloat(spec.TotalValue),
		Currency:          spec.Currency,
		PaymentTerms:      ptrString(spec.PaymentTerms),
		EffectiveDate:     &effectiveDate,
		ExpiryDate:        &expiryDate,
		RenewalDate:       renewalDate,
		AutoRenew:         spec.AutoRenew,
		RenewalNoticeDays: spec.RenewalNoticeDays,
		Status:            model.ContractStatusDraft,
		OwnerUserID:       spec.Owner.ID,
		OwnerName:         spec.Owner.Name,
		LegalReviewerID:   &spec.LegalReviewer.ID,
		LegalReviewerName: ptrString(spec.LegalReviewer.Name),
		Department:        ptrString(spec.Department),
		Tags:              spec.Tags,
		Metadata: map[string]any{
			"portfolio":  spec.Department,
			"seed_index": index + 1,
		},
	}

	clauses, documentText := buildSeedClauses(baseContract, spec, index, blueprints, recommendations)
	req := dto.CreateContractRequest{
		Title:             spec.Title,
		Type:              spec.Type,
		Description:       spec.Description,
		PartyAName:        baseContract.PartyAName,
		PartyBName:        spec.PartyBName,
		PartyBEntity:      ptrString(spec.PartyBEntity),
		PartyBContact:     ptrString(spec.PartyBContact),
		TotalValue:        ptrFloat(spec.TotalValue),
		Currency:          spec.Currency,
		PaymentTerms:      ptrString(spec.PaymentTerms),
		EffectiveDate:     &effectiveDate,
		ExpiryDate:        &expiryDate,
		RenewalDate:       renewalDate,
		AutoRenew:         spec.AutoRenew,
		RenewalNoticeDays: spec.RenewalNoticeDays,
		OwnerUserID:       spec.Owner.ID,
		OwnerName:         spec.Owner.Name,
		LegalReviewerID:   &spec.LegalReviewer.ID,
		LegalReviewerName: ptrString(spec.LegalReviewer.Name),
		Department:        ptrString(spec.Department),
		Tags:              spec.Tags,
		Metadata: map[string]any{
			"seeded":      true,
			"seed_status": spec.Status,
		},
		Document: &dto.FileReference{
			FileID:        uuid.NewSHA1(seedTenantID, []byte(fmt.Sprintf("seed-contract-file-%02d", index+1))),
			FileName:      slugify(spec.Title) + ".txt",
			FileSizeBytes: int64(len(documentText)),
			ContentHash:   contentHash(documentText),
			ExtractedText: documentText,
			ChangeSummary: "Initial seeded contract document.",
		},
	}

	contract, err := app.ContractService.CreateContract(ctx, seedTenantID, seedSystemUser, req)
	if err != nil {
		return nil, fmt.Errorf("create seeded contract %q: %w", spec.Title, err)
	}
	if err := seedContractAnalysis(ctx, app, contract, clauses, documentText); err != nil {
		return nil, fmt.Errorf("seed analysis for %q: %w", spec.Title, err)
	}
	if err := driveSeedContractStatus(ctx, app, contract.ID, spec.Status); err != nil {
		return nil, fmt.Errorf("seed status for %q: %w", spec.Title, err)
	}
	updated, err := app.Store.Contracts.Get(ctx, seedTenantID, contract.ID)
	if err != nil {
		return nil, fmt.Errorf("reload seeded contract %q: %w", spec.Title, err)
	}
	return updated, nil
}

func seedContractAnalysis(ctx context.Context, app *Application, contract *model.Contract, clauses []model.ExtractedClause, text string) error {
	analysis, err := buildSeedAnalysis(contract, clauses, text)
	if err != nil {
		return err
	}
	return database.RunInTx(ctx, app.Store.DB(), func(tx pgx.Tx) error {
		if err := app.Store.Clauses.ReplaceForContract(ctx, tx, contract.TenantID, contract.ID, clauses); err != nil {
			return err
		}
		if err := app.Store.Contracts.InsertAnalysis(ctx, tx, analysis); err != nil {
			return err
		}
		return app.Store.Contracts.UpdateAnalysisFields(ctx, tx, contract.TenantID, contract.ID, analysis.RiskScore, analysis.OverallRisk, model.AnalysisStatusCompleted, analysis.AnalyzedAt)
	})
}

func buildSeedAnalysis(contract *model.Contract, clauses []model.ExtractedClause, text string) (*model.ContractRiskAnalysis, error) {
	found := make(map[model.ClauseType]bool, len(clauses))
	clauseRiskSum := 0.0
	highRiskCount := 0
	for _, clause := range clauses {
		found[clause.ClauseType] = true
		clauseRiskSum += clause.RiskScore
		if clause.RiskLevel == model.RiskLevelCritical || clause.RiskLevel == model.RiskLevelHigh {
			highRiskCount++
		}
	}

	missing := analyzer.NewMissingClauseDetector().Detect(contract.Type, found)
	complianceFlags := analyzer.NewComplianceChecker("Saudi Arabia").Check(contract, clauses, text)
	parties, dates, amounts := analyzer.NewEntityExtractor().Extract(text)

	clauseRiskAvg := 0.0
	if len(clauses) > 0 {
		clauseRiskAvg = clauseRiskSum / float64(len(clauses))
	}
	missingPenalty := float64(len(missing) * 8)
	valueFactor := 0.0
	if contract.TotalValue != nil {
		switch {
		case *contract.TotalValue > 10_000_000:
			valueFactor = 15
		case *contract.TotalValue > 1_000_000:
			valueFactor = 10
		}
	}
	expiryFactor := 0.0
	if contract.ExpiryDate != nil {
		days := int(contract.ExpiryDate.Sub(normalizeSeedDate(seedReferenceAt)).Hours() / 24)
		switch {
		case days <= 7:
			expiryFactor = 20
		case days <= 30:
			expiryFactor = 10
		}
	}
	compliancePenalty := float64(len(complianceFlags) * 5)
	riskScore := clampSeedScore(clauseRiskAvg + missingPenalty + valueFactor + expiryFactor + compliancePenalty)
	riskLevel := model.RiskLevelFromScore(riskScore)

	recommendations := collectSeedRecommendations(clauses, missing, complianceFlags)
	findings := buildSeedFindings(clauses, missing, complianceFlags)
	if len(findings) > 5 {
		findings = findings[:5]
	}

	return &model.ContractRiskAnalysis{
		ID:                  uuid.NewSHA1(contract.ID, []byte("seed-analysis")),
		TenantID:            contract.TenantID,
		ContractID:          contract.ID,
		ContractVersion:     contract.CurrentVersion,
		OverallRisk:         riskLevel,
		RiskScore:           riskScore,
		ClauseCount:         len(clauses),
		HighRiskClauseCount: highRiskCount,
		MissingClauses:      missing,
		KeyFindings:         findings,
		Recommendations:     recommendations,
		ComplianceFlags:     complianceFlags,
		ExtractedParties:    parties,
		ExtractedDates:      dates,
		ExtractedAmounts:    amounts,
		AnalysisDurationMS:  120,
		AnalyzedBy:          "system",
		AnalyzedAt:          seedReferenceAt,
		CreatedAt:           seedReferenceAt,
	}, nil
}

func driveSeedContractStatus(ctx context.Context, app *Application, contractID uuid.UUID, target model.ContractStatus) error {
	for _, next := range seedStatusPath(target) {
		if _, err := app.ContractService.UpdateStatus(ctx, seedTenantID, seedSystemUser, contractID, next); err != nil {
			return err
		}
	}
	return nil
}

func applySeedClauseReviews(ctx context.Context, app *Application, contracts []*model.Contract, reviewerID uuid.UUID, reviewedAt time.Time) error {
	allClauses := make([]model.Clause, 0, len(contracts)*3)
	for _, contract := range contracts {
		clauses, err := app.Store.Clauses.ListByContract(ctx, seedTenantID, contract.ID)
		if err != nil {
			return err
		}
		allClauses = append(allClauses, clauses...)
	}

	statuses := make([]model.ClauseReviewStatus, 0, 30)
	for i := 0; i < 20; i++ {
		statuses = append(statuses, model.ClauseReviewReviewed)
	}
	for i := 0; i < 5; i++ {
		statuses = append(statuses, model.ClauseReviewFlagged)
	}
	for i := 0; i < 5; i++ {
		statuses = append(statuses, model.ClauseReviewAccepted)
	}
	notesByStatus := map[model.ClauseReviewStatus]string{
		model.ClauseReviewReviewed: "Reviewed during seed data initialization.",
		model.ClauseReviewFlagged:  "Flagged for manual legal review during demo setup.",
		model.ClauseReviewAccepted: "Accepted during seed review cycle.",
	}

	for idx, status := range statuses {
		clause := allClauses[idx]
		notes := notesByStatus[status]
		if err := app.Store.Clauses.UpdateReview(ctx, app.Store.DB(), seedTenantID, clause.ContractID, clause.ID, status, &reviewerID, notes, reviewedAt); err != nil {
			return err
		}
	}
	return nil
}

func seedComplianceRules(ctx context.Context, app *Application, userID uuid.UUID) ([]model.ComplianceRule, error) {
	requests := []dto.CreateComplianceRuleRequest{
		{
			Name:          "Default expiry warning",
			Description:   "Alert owners when active contracts approach expiry.",
			RuleType:      model.ComplianceRuleExpiryWarning,
			Severity:      model.ComplianceSeverityHigh,
			Config:        map[string]any{"days_before": 30},
			ContractTypes: []string{},
			Enabled:       true,
		},
		{
			Name:          "Missing clause review",
			Description:   "Flag contracts missing mandatory standard clauses.",
			RuleType:      model.ComplianceRuleMissingClause,
			Severity:      model.ComplianceSeverityHigh,
			Config:        map[string]any{},
			ContractTypes: []string{string(model.ContractTypeServiceAgreement), string(model.ContractTypeVendor), string(model.ContractTypeLicense)},
			Enabled:       true,
		},
		{
			Name:          "High risk review gate",
			Description:   "High-risk contracts must reach legal review status.",
			RuleType:      model.ComplianceRuleRiskThreshold,
			Severity:      model.ComplianceSeverityCritical,
			Config:        map[string]any{"min_score": 70, "required_status": string(model.ContractStatusLegalReview)},
			ContractTypes: []string{},
			Enabled:       true,
		},
		{
			Name:          "Review overdue",
			Description:   "Contracts may not remain in internal or legal review beyond the SLA.",
			RuleType:      model.ComplianceRuleReviewOverdue,
			Severity:      model.ComplianceSeverityMedium,
			Config:        map[string]any{"overdue_days": 7},
			ContractTypes: []string{},
			Enabled:       true,
		},
		{
			Name:          "Data protection required",
			Description:   "Contracts involving personal data must include data protection language.",
			RuleType:      model.ComplianceRuleDataProtectionRequired,
			Severity:      model.ComplianceSeverityCritical,
			Config:        map[string]any{},
			ContractTypes: []string{string(model.ContractTypeVendor), string(model.ContractTypeServiceAgreement), string(model.ContractTypeNDA)},
			Enabled:       true,
		},
	}

	rules := make([]model.ComplianceRule, 0, len(requests))
	for _, req := range requests {
		rule, err := app.ComplianceService.CreateRule(ctx, seedTenantID, userID, req)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}
	return rules, nil
}

func seedLegalDocuments(ctx context.Context, app *Application, contracts []*model.Contract, userID uuid.UUID) error {
	type docSpec struct {
		Title           string
		Type            model.LegalDocumentType
		Description     string
		Category        string
		Confidentiality model.DocumentConfidentiality
		ContractIndex   int
		Tags            []string
		Content         string
	}
	specs := []docSpec{
		{"Information Security Policy", model.DocumentTypePolicy, "Baseline information security requirements for enterprise contracts.", "IT", model.DocumentConfidentialityInternal, -1, []string{"policy", "security"}, "All vendors must maintain baseline information security safeguards and notify Clario of material incidents."},
		{"Data Retention Policy", model.DocumentTypePolicy, "Records retention standards for legal operations.", "Legal Ops", model.DocumentConfidentialityInternal, -1, []string{"policy", "retention"}, "Retention periods must align with statutory obligations and litigation hold procedures."},
		{"Vendor Due Diligence Policy", model.DocumentTypePolicy, "Vendor onboarding policy for procurement and legal review.", "Procurement", model.DocumentConfidentialityConfidential, 12, []string{"policy", "vendor"}, "Critical vendors require audit rights, insurance validation, and data protection review before activation."},
		{"NDA Template 2026", model.DocumentTypeTemplate, "Approved mutual NDA template.", "Legal", model.DocumentConfidentialityInternal, -1, []string{"template", "nda"}, "This template provides balanced confidentiality, term, and governing law language for standard NDAs."},
		{"Service Agreement Template 2026", model.DocumentTypeTemplate, "Approved services agreement template.", "Legal", model.DocumentConfidentialityInternal, -1, []string{"template", "services"}, "This template includes limitations of liability, termination rights, and service level commitments."},
		{"Board Memo - Litigation Reserve", model.DocumentTypeMemo, "Summary memo for pending litigation reserve impacts.", "Finance", model.DocumentConfidentialityPrivileged, -1, []string{"memo", "litigation"}, "Outside counsel estimates are summarized for board review and reserve planning."},
		{"Memo - Regulatory Update Q1", model.DocumentTypeMemo, "Quarterly regulatory update for commercial contracting.", "Legal", model.DocumentConfidentialityConfidential, -1, []string{"memo", "regulatory"}, "Recent regulatory guidance tightened requirements for breach notice timing and processor oversight."},
		{"Board Resolution - Litigation Hold", model.DocumentTypeResolution, "Board resolution approving enterprise litigation hold procedures.", "Board", model.DocumentConfidentialityPrivileged, -1, []string{"resolution", "board"}, "Resolved that the company shall maintain a documented litigation hold workflow across all regulated documents."},
	}

	for idx, spec := range specs {
		var contractID *uuid.UUID
		if spec.ContractIndex >= 0 && spec.ContractIndex < len(contracts) {
			contractID = &contracts[spec.ContractIndex].ID
		}
		fileName := slugify(spec.Title) + ".txt"
		req := dto.CreateLegalDocumentRequest{
			Title:           spec.Title,
			Type:            spec.Type,
			Description:     spec.Description,
			Category:        ptrString(spec.Category),
			Confidentiality: spec.Confidentiality,
			ContractID:      contractID,
			Tags:            spec.Tags,
			Metadata:        map[string]any{"seeded": true, "document_index": idx + 1},
			Document: &dto.FileReference{
				FileID:        uuid.NewSHA1(seedTenantID, []byte("seed-doc-"+fileName)),
				FileName:      fileName,
				FileSizeBytes: int64(len(spec.Content)),
				ContentHash:   contentHash(spec.Content),
				ChangeSummary: "Initial seeded document version.",
			},
		}
		document, err := app.DocumentService.Create(ctx, seedTenantID, userID, req)
		if err != nil {
			return err
		}
		if idx < 2 {
			updated := spec.Content + "\nUpdated on " + seedReferenceAt.Format("2006-01-02") + " for demo versioning."
			if _, err := app.DocumentService.UploadVersion(ctx, seedTenantID, userID, document.ID, dto.UploadDocumentVersionRequest{
				FileReference: dto.FileReference{
					FileID:        uuid.NewSHA1(seedTenantID, []byte("seed-doc-v2-"+fileName)),
					FileName:      strings.TrimSuffix(fileName, ".txt") + "-v2.txt",
					FileSizeBytes: int64(len(updated)),
					ContentHash:   contentHash(updated),
					ChangeSummary: "Version 2 seeded update.",
				},
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func seedComplianceAlerts(ctx context.Context, app *Application, contracts []*model.Contract, rules []model.ComplianceRule, userID uuid.UUID) error {
	statuses := []model.ComplianceAlertStatus{
		model.ComplianceAlertOpen,
		model.ComplianceAlertOpen,
		model.ComplianceAlertOpen,
		model.ComplianceAlertAcknowledged,
		model.ComplianceAlertAcknowledged,
		model.ComplianceAlertAcknowledged,
		model.ComplianceAlertAcknowledged,
		model.ComplianceAlertResolved,
		model.ComplianceAlertResolved,
		model.ComplianceAlertResolved,
	}
	severities := []model.ComplianceSeverity{
		model.ComplianceSeverityHigh,
		model.ComplianceSeverityCritical,
		model.ComplianceSeverityMedium,
		model.ComplianceSeverityHigh,
		model.ComplianceSeverityMedium,
		model.ComplianceSeverityLow,
		model.ComplianceSeverityHigh,
		model.ComplianceSeverityCritical,
		model.ComplianceSeverityMedium,
		model.ComplianceSeverityLow,
	}
	titles := []string{
		"Managed Cloud Services Agreement expires soon",
		"Data Hosting vendor lacks audit rights",
		"Customer analytics contract requires privacy review",
		"ERP license contract exceeds risk threshold",
		"NDA renewal notice requires confirmation",
		"Consulting engagement missing updated memo attachment",
		"Office facilities vendor review is overdue",
		"Historic NDA compliance issue resolved",
		"Legacy print services closure completed",
		"Resolved memo linkage discrepancy",
	}

	for idx := range titles {
		var ruleID *uuid.UUID
		if len(rules) > 0 {
			value := rules[idx%len(rules)].ID
			ruleID = &value
		}
		contractID := contracts[idx%len(contracts)].ID
		alert := &model.ComplianceAlert{
			ID:          uuid.NewSHA1(seedTenantID, []byte(fmt.Sprintf("seed-alert-%02d", idx+1))),
			TenantID:    seedTenantID,
			RuleID:      ruleID,
			ContractID:  &contractID,
			Title:       titles[idx],
			Description: fmt.Sprintf("Seeded compliance alert %d generated for demo reporting coverage.", idx+1),
			Severity:    severities[idx],
			Status:      statuses[idx],
			Evidence: map[string]any{
				"seeded":      true,
				"contract_id": contractID,
				"alert_index": idx + 1,
			},
		}
		if statuses[idx] == model.ComplianceAlertResolved {
			alert.ResolvedBy = &userID
			resolvedAt := seedReferenceAt.Add(time.Duration(idx+1) * time.Hour)
			alert.ResolvedAt = &resolvedAt
			notes := "Resolved during seeded compliance triage."
			alert.ResolutionNotes = &notes
		}
		if err := app.Store.Alerts.Create(ctx, app.Store.DB(), alert); err != nil {
			return err
		}
	}
	return nil
}

func buildSeedClauses(contract *model.Contract, spec seedContractSpec, index int, blueprints []seedClauseBlueprint, recommendations *analyzer.RecommendationEngine) ([]model.ExtractedClause, string) {
	textParts := []string{
		fmt.Sprintf("This agreement is entered into between %s and %s effective as of %s.", contract.PartyAName, contract.PartyBName, contract.EffectiveDate.UTC().Format("January 2, 2006")),
		fmt.Sprintf("Party A: %s", contract.PartyAName),
		fmt.Sprintf("Party B: %s", contract.PartyBName),
		fmt.Sprintf("The total value of %s %.2f applies under this contract.", contract.Currency, *contract.TotalValue),
		fmt.Sprintf("Expiry date is %s.", contract.ExpiryDate.UTC().Format("January 2, 2006")),
	}
	if contract.RenewalDate != nil {
		textParts = append(textParts, fmt.Sprintf("Renewal date is %s.", contract.RenewalDate.UTC().Format("January 2, 2006")))
	}
	if spec.ContainsPII {
		textParts = append(textParts, "The parties may process personal data and customer account identifiers while performing the services.")
	}

	slotStart := index * 3
	clauses := make([]model.ExtractedClause, 0, 3)
	for sectionIndex := 0; sectionIndex < 3; sectionIndex++ {
		blueprint := blueprints[(slotStart+sectionIndex)%len(blueprints)]
		riskLevel, keywords := seedClauseRiskProfile(blueprint, slotStart+sectionIndex)
		content := renderSeedClauseContent(blueprint, riskLevel, keywords)
		sectionReference := fmt.Sprintf("Section %d", sectionIndex+1)
		recs := recommendations.Recommend(blueprint.ClauseType, riskLevel, keywords, content)
		clauses = append(clauses, model.ExtractedClause{
			ClauseType:           blueprint.ClauseType,
			PrimaryType:          blueprint.ClauseType,
			MatchedTypes:         []model.ClauseType{blueprint.ClauseType},
			Title:                blueprint.Title,
			Content:              content,
			SectionReference:     sectionReference,
			PageNumber:           1,
			RiskLevel:            riskLevel,
			RiskScore:            riskLevel.Score(),
			RiskKeywords:         keywords,
			AnalysisSummary:      fmt.Sprintf("%s cites %s and is tagged %s risk.", sectionReference, strings.ToLower(blueprint.Title), riskLevel),
			Recommendations:      recs,
			ComplianceFlags:      seedClauseComplianceFlags(blueprint.ClauseType, keywords, content),
			ExtractionConfidence: seedConfidence(riskLevel),
			PatternHits:          1,
			FirstMatchOffset:     sectionIndex * 80,
		})
		textParts = append(textParts, fmt.Sprintf("%s %s\n%s", sectionReference, blueprint.Title, content))
	}
	return clauses, strings.Join(textParts, "\n\n")
}

func buildSeedFindings(clauses []model.ExtractedClause, missing []model.ClauseType, complianceFlags []model.ComplianceFlag) []model.RiskFinding {
	findings := make([]model.RiskFinding, 0, len(clauses)+len(missing)+len(complianceFlags))
	for _, clause := range clauses {
		if clause.RiskLevel == model.RiskLevelNone {
			continue
		}
		clauseType := clause.ClauseType
		ref := clause.SectionReference
		findings = append(findings, model.RiskFinding{
			Title:           fmt.Sprintf("%s clause requires attention", blueprintTitle(clause.ClauseType)),
			Description:     clause.AnalysisSummary,
			Severity:        clause.RiskLevel,
			ClauseReference: &ref,
			Recommendation:  strings.Join(clause.Recommendations, " "),
			ClauseType:      &clauseType,
		})
	}
	for _, clauseType := range missing {
		missingType := clauseType
		findings = append(findings, model.RiskFinding{
			Title:          fmt.Sprintf("Missing %s clause", blueprintTitle(clauseType)),
			Description:    "Required standard clause is missing from the seeded contract text.",
			Severity:       model.RiskLevelHigh,
			Recommendation: "Add the missing clause before approval.",
			ClauseType:     &missingType,
		})
	}
	for _, flag := range complianceFlags {
		findings = append(findings, model.RiskFinding{
			Title:           flag.Title,
			Description:     flag.Description,
			Severity:        flag.Severity,
			ClauseReference: flag.ClauseReference,
			Recommendation:  flag.Description,
		})
	}
	sort.SliceStable(findings, func(i, j int) bool {
		if findings[i].Severity.Weight() != findings[j].Severity.Weight() {
			return findings[i].Severity.Weight() > findings[j].Severity.Weight()
		}
		return findings[i].Title < findings[j].Title
	})
	return findings
}

func collectSeedRecommendations(clauses []model.ExtractedClause, missing []model.ClauseType, complianceFlags []model.ComplianceFlag) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(clauses)+len(missing)+len(complianceFlags))
	appendUnique := func(values ...string) {
		for _, value := range values {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			out = append(out, value)
		}
	}
	for _, clause := range clauses {
		appendUnique(clause.Recommendations...)
	}
	for _, clauseType := range missing {
		appendUnique("Insert a standard " + blueprintTitle(clauseType) + " clause before approval.")
	}
	for _, flag := range complianceFlags {
		appendUnique(flag.Description)
	}
	sort.Strings(out)
	return out
}

func seedUsers() []seedUser {
	return []seedUser{
		{mustUUID("22222222-2222-2222-2222-222222222301"), "Aisha Rahman", "Group Legal Director"},
		{mustUUID("22222222-2222-2222-2222-222222222302"), "Omar Haddad", "Senior Legal Counsel"},
		{mustUUID("22222222-2222-2222-2222-222222222303"), "Leila Faris", "Procurement Director"},
		{mustUUID("22222222-2222-2222-2222-222222222304"), "Noura Saleh", "HR Director"},
		{mustUUID("22222222-2222-2222-2222-222222222305"), "Tariq Malik", "CIO"},
		{mustUUID("22222222-2222-2222-2222-222222222306"), "Rana Kassem", "Finance Controller"},
	}
}

func seedContractSpecs(users []seedUser) []seedContractSpec {
	return []seedContractSpec{
		{"Managed Cloud Services Agreement", model.ContractTypeServiceAgreement, "Managed hosting and support for regional production workloads.", "Nimbus Datacenters LLC", "Nimbus Datacenters LLC", "contracts@nimbus.example", "IT", []string{"cloud", "infrastructure"}, 2400000, "SAR", "net_30", -180, 20, 30, false, model.ContractStatusActive, users[4], users[0], true},
		{"ERP Support Services Agreement", model.ContractTypeServiceAgreement, "Application support and enhancement services for the enterprise ERP stack.", "Sahara Systems Ltd.", "Sahara Systems Ltd.", "legal@sahara.example", "Finance", []string{"erp", "support"}, 850000, "SAR", "monthly", -300, 95, 30, false, model.ContractStatusActive, users[5], users[1], false},
		{"Branch Security Services Agreement", model.ContractTypeServiceAgreement, "Guarding and alarm response coverage for branch offices.", "Shield Operations Co.", "Shield Operations Co.", "ops@shield.example", "Operations", []string{"physical-security"}, 420000, "SAR", "net_45", -10, 120, 30, false, model.ContractStatusInternalReview, users[2], users[1], false},
		{"Customer Analytics Implementation Agreement", model.ContractTypeServiceAgreement, "Implementation of customer analytics platform and managed reporting services.", "Bright Metrics FZE", "Bright Metrics FZE", "contracts@brightmetrics.example", "Marketing", []string{"analytics", "implementation"}, 1300000, "SAR", "milestone", -30, 220, 30, false, model.ContractStatusNegotiation, users[4], users[0], true},
		{"Digital Workplace Services Agreement", model.ContractTypeServiceAgreement, "Legacy digital workplace managed services contract retained for reference.", "Vertex Enablement Ltd.", "Vertex Enablement Ltd.", "legal@vertex.example", "IT", []string{"workspace", "legacy"}, 610000, "SAR", "net_30", -400, -15, 30, false, model.ContractStatusExpired, users[4], users[0], false},
		{"Mutual NDA with Falcon Robotics", model.ContractTypeNDA, "Mutual confidentiality arrangement for robotics partnership evaluation.", "Falcon Robotics", "Falcon Robotics LLC", "privacy@falcon.example", "R&D", []string{"nda", "robotics"}, 25000, "SAR", "n/a", -60, 15, 30, false, model.ContractStatusActive, users[0], users[1], true},
		{"Investor NDA with Cedar Capital", model.ContractTypeNDA, "Confidentiality agreement for fundraising diligence.", "Cedar Capital", "Cedar Capital Partners", "dealteam@cedar.example", "Finance", []string{"nda", "investor"}, 15000, "SAR", "n/a", -90, 180, 30, false, model.ContractStatusActive, users[5], users[0], false},
		{"Product Evaluation NDA with Helix Labs", model.ContractTypeNDA, "Evaluation NDA for new product trials.", "Helix Labs", "Helix Labs Inc.", "contracts@helix.example", "Product", []string{"nda", "evaluation"}, 12000, "SAR", "n/a", -45, 210, 30, false, model.ContractStatusActive, users[4], users[1], false},
		{"Historic NDA with Atlas Retail", model.ContractTypeNDA, "Historical confidentiality agreement retained for audit.", "Atlas Retail", "Atlas Retail Group", "legal@atlas.example", "Commercial", []string{"nda", "archive"}, 18000, "SAR", "n/a", -500, -60, 30, false, model.ContractStatusExpired, users[0], users[1], false},
		{"Employment Agreement - Chief Data Officer", model.ContractTypeEmployment, "Executive employment agreement for the chief data officer.", "Dr. Layth Hamdan", "Layth Hamdan", "layth.hamdan@example", "HR", []string{"employment", "executive"}, 1100000, "SAR", "monthly", -300, 400, 30, false, model.ContractStatusActive, users[3], users[0], false},
		{"Employment Agreement - Regional Counsel", model.ContractTypeEmployment, "Regional counsel employment terms and confidentiality obligations.", "Maya Qureshi", "Maya Qureshi", "maya.qureshi@example", "HR", []string{"employment", "legal"}, 900000, "SAR", "monthly", -240, 500, 30, false, model.ContractStatusActive, users[3], users[0], false},
		{"Employment Agreement - Sales Director", model.ContractTypeEmployment, "Sales director incentive and restrictive covenant agreement.", "Karim Najjar", "Karim Najjar", "karim.najjar@example", "HR", []string{"employment", "sales"}, 700000, "SAR", "monthly", -120, 600, 30, false, model.ContractStatusActive, users[3], users[0], false},
		{"Vendor Master Agreement - Data Hosting", model.ContractTypeVendor, "Primary vendor agreement for hosted infrastructure and backup operations.", "Blue Harbor Hosting", "Blue Harbor Hosting", "contracts@blueharbor.example", "IT", []string{"vendor", "hosting"}, 3800000, "SAR", "net_30", -330, 10, 20, true, model.ContractStatusActive, users[4], users[0], true},
		{"Vendor Agreement - Office Facilities", model.ContractTypeVendor, "Facilities management services for corporate offices.", "Crescent Facilities Co.", "Crescent Facilities Co.", "legal@crescentfacilities.example", "Operations", []string{"vendor", "facilities"}, 540000, "SAR", "net_45", -150, 45, 30, false, model.ContractStatusActive, users[2], users[1], false},
		{"Vendor Agreement - Legacy Print Services", model.ContractTypeVendor, "Legacy office print and mailroom services agreement.", "Prime Print Services", "Prime Print Services", "support@primeprint.example", "Operations", []string{"vendor", "legacy"}, 110000, "SAR", "net_30", -540, 60, 30, false, model.ContractStatusTerminated, users[2], users[1], false},
		{"Software License Agreement - ERP", model.ContractTypeLicense, "Enterprise resource planning software license and maintenance.", "Northgate Software", "Northgate Software Ltd.", "licensing@northgate.example", "Finance", []string{"license", "erp"}, 4600000, "SAR", "annual", -270, 200, 45, false, model.ContractStatusActive, users[5], users[0], false},
		{"Software License Agreement - Endpoint Security", model.ContractTypeLicense, "Endpoint security platform license and support.", "Sentinel Works", "Sentinel Works Inc.", "contracts@sentinelworks.example", "IT", []string{"license", "security"}, 2200000, "SAR", "annual", -120, 40, 45, true, model.ContractStatusActive, users[4], users[0], false},
		{"MOU with Riyadh Innovation Hub", model.ContractTypeMOU, "Memorandum of understanding covering joint innovation initiatives.", "Riyadh Innovation Hub", "Riyadh Innovation Hub", "office@innovationhub.example", "Strategy", []string{"mou", "innovation"}, 75000, "SAR", "n/a", -40, 5, 30, false, model.ContractStatusActive, users[0], users[1], false},
		{"MOU with Green Energy Council", model.ContractTypeMOU, "Draft memorandum for sustainability cooperation.", "Green Energy Council", "Green Energy Council", "secretariat@greencouncil.example", "Sustainability", []string{"mou", "sustainability"}, 55000, "SAR", "n/a", 5, 365, 30, false, model.ContractStatusDraft, users[0], users[1], false},
		{"Consulting Agreement - Tax Advisory", model.ContractTypeConsulting, "Tax advisory consulting engagement for regional structuring.", "Apex Advisory LLP", "Apex Advisory LLP", "engagements@apexadvisory.example", "Finance", []string{"consulting", "tax"}, 380000, "SAR", "net_30", -45, 28, 25, true, model.ContractStatusActive, users[5], users[0], false},
	}
}

func seedClauseBlueprints() []seedClauseBlueprint {
	return []seedClauseBlueprint{
		{model.ClauseTypeIndemnification, "Indemnification", "The indemnification clause requires the supplier to indemnify Clario.", "Liability is limited to direct third-party claims caused by breach.", []string{"unlimited", "uncapped", "sole expense", "first dollar", "all claims", "regardless of fault", "broadly defined losses"}},
		{model.ClauseTypeTermination, "Termination", "The termination clause describes when either party may terminate the agreement.", "Termination requires material breach, notice, and a cure period.", []string{"without cause", "immediate", "no notice", "at will", "unilateral", "no cure period", "automatic termination"}},
		{model.ClauseTypeLimitationOfLiability, "Limitation of Liability", "The limitation of liability clause caps aggregate liability.", "Aggregate liability is capped at fees paid in the prior contract year.", []string{"unlimited", "no cap", "no limitation", "excluding consequential", "excluding indirect", "waiver of liability"}},
		{model.ClauseTypeConfidentiality, "Confidentiality", "The confidentiality clause protects proprietary information.", "Confidential information may be used only to perform the agreement and must be returned on request.", []string{"perpetual", "no exceptions", "residual knowledge", "unrestricted use"}},
		{model.ClauseTypeIPOwnership, "IP Ownership", "The intellectual property clause allocates ownership of work product.", "Work product created specifically for Clario is assigned to Clario upon payment.", []string{"vendor retains", "shared ownership", "license back", "non-exclusive", "pre-existing IP", "joint ownership"}},
		{model.ClauseTypeNonCompete, "Non-Compete", "The non-compete clause restricts competitive activity.", "Restrictions apply only to directly competing services in agreed territories for a limited term.", []string{"worldwide", "perpetual", "all industries", "no geographic limit"}},
		{model.ClauseTypePaymentTerms, "Payment Terms", "The payment clause governs invoicing and fees.", "Invoices are payable within thirty days after receipt of an undisputed invoice.", []string{"net 90", "net 120", "upon completion only", "milestone-based only", "no penalty for late payment"}},
		{model.ClauseTypeWarranty, "Warranty", "The warranty clause provides service and compliance assurances.", "Each party warrants it has authority to enter the agreement and will perform services professionally.", []string{"as-is", "no warranty", "disclaims all", "implied warranties excluded"}},
		{model.ClauseTypeForceMajeure, "Force Majeure", "The force majeure clause addresses extraordinary events beyond control.", "Affected obligations are suspended only while the force majeure event continues.", []string{"pandemic excluded", "economic downturn excluded", "no termination right"}},
		{model.ClauseTypeDisputeResolution, "Dispute Resolution", "The dispute resolution clause describes escalation and arbitration.", "The parties will attempt executive escalation before commencing arbitration in Riyadh.", []string{"foreign jurisdiction", "binding arbitration only", "waive right to trial", "vendor's jurisdiction"}},
		{model.ClauseTypeDataProtection, "Data Protection", "The data protection clause governs processing of personal data.", "The processor must notify Clario of breaches, delete data on request, and limit transfers.", []string{"no breach notification", "unlimited processing", "no data deletion", "cross-border transfer unrestricted"}},
		{model.ClauseTypeGoverningLaw, "Governing Law", "The governing law clause states which laws apply.", "This agreement is governed by the laws of the Kingdom of Saudi Arabia.", []string{"foreign law", "vendor's jurisdiction", "new york", "england"}},
		{model.ClauseTypeAssignment, "Assignment", "The assignment clause controls transfers and novation.", "Neither party may assign the agreement without prior written consent except for internal reorganizations.", []string{"freely assignable", "no consent required", "to any affiliate"}},
		{model.ClauseTypeInsurance, "Insurance", "The insurance clause defines minimum coverage requirements.", "The supplier must maintain professional liability and cyber insurance with evidence on request.", []string{"no insurance requirement", "minimum not specified", "coverage not evidenced"}},
		{model.ClauseTypeAuditRights, "Audit Rights", "The audit rights clause grants inspection of records.", "Clario may audit relevant records annually on reasonable notice.", []string{"no audit right", "only with consent", "limited frequency"}},
		{model.ClauseTypeSLA, "Service Levels", "The service level clause sets uptime and response expectations.", "Service levels include uptime targets, response times, and service credits for failures.", []string{"best effort", "no penalty", "no credit", "commercially reasonable"}},
		{model.ClauseTypeAutoRenewal, "Auto Renewal", "The auto renewal clause describes renewal mechanics.", "Renewal requires prior written notice and preserves current commercial terms unless agreed otherwise.", []string{"without notice", "annual renewal", "opt-out only", "price increase on renewal"}},
		{model.ClauseTypeRepresentations, "Representations", "The representations clause captures statements and undertakings.", "Each party represents that its statements are accurate as of signing and during performance.", []string{"unilateral representations", "perpetual", "survive termination indefinitely"}},
		{model.ClauseTypeNonSolicitation, "Non-Solicitation", "The non-solicitation clause restricts poaching of personnel.", "Neither party may solicit the other party's named project staff during the term and for twelve months after.", []string{"perpetual", "worldwide", "all employees", "includes independent contractors"}},
	}
}

func seedStatusPath(target model.ContractStatus) []model.ContractStatus {
	switch target {
	case model.ContractStatusDraft:
		return nil
	case model.ContractStatusInternalReview:
		return []model.ContractStatus{model.ContractStatusInternalReview}
	case model.ContractStatusLegalReview:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview}
	case model.ContractStatusNegotiation:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview, model.ContractStatusNegotiation}
	case model.ContractStatusPendingSignature:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview, model.ContractStatusNegotiation, model.ContractStatusPendingSignature}
	case model.ContractStatusActive:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview, model.ContractStatusNegotiation, model.ContractStatusPendingSignature, model.ContractStatusActive}
	case model.ContractStatusExpired:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview, model.ContractStatusNegotiation, model.ContractStatusPendingSignature, model.ContractStatusActive, model.ContractStatusExpired}
	case model.ContractStatusTerminated:
		return []model.ContractStatus{model.ContractStatusInternalReview, model.ContractStatusLegalReview, model.ContractStatusNegotiation, model.ContractStatusPendingSignature, model.ContractStatusActive, model.ContractStatusTerminated}
	default:
		return nil
	}
}

func seedClauseRiskProfile(blueprint seedClauseBlueprint, slot int) (model.RiskLevel, []string) {
	switch {
	case slot < 8:
		keywords := append([]string(nil), blueprint.RiskKeywords...)
		if len(keywords) > 5 {
			keywords = keywords[:5]
		}
		if blueprint.ClauseType == model.ClauseTypeLimitationOfLiability && len(keywords) > 0 {
			return model.RiskLevelCritical, keywords
		}
		return model.RiskLevelHigh, keywords
	case slot < 23:
		return model.RiskLevelMedium, firstKeywords(blueprint.RiskKeywords, 3)
	case slot < 40:
		return model.RiskLevelLow, firstKeywords(blueprint.RiskKeywords, 1)
	default:
		return model.RiskLevelNone, nil
	}
}

func renderSeedClauseContent(blueprint seedClauseBlueprint, riskLevel model.RiskLevel, keywords []string) string {
	if len(keywords) == 0 {
		return blueprint.Trigger + " " + blueprint.SafeBody
	}
	return fmt.Sprintf("%s %s Risk considerations include %s.", blueprint.Trigger, blueprint.SafeBody, strings.Join(keywords, ", "))
}

func seedClauseComplianceFlags(clauseType model.ClauseType, keywords []string, content string) []string {
	lower := strings.ToLower(strings.Join(keywords, " ")) + " " + strings.ToLower(content)
	flags := []string{}
	if clauseType == model.ClauseTypeDataProtection && strings.Contains(lower, "cross-border transfer unrestricted") {
		flags = append(flags, "cross_border_transfer_unrestricted")
	}
	if clauseType == model.ClauseTypeGoverningLaw && strings.Contains(lower, "foreign law") {
		flags = append(flags, "foreign_governing_law")
	}
	if clauseType == model.ClauseTypeAutoRenewal && strings.Contains(lower, "price increase on renewal") {
		flags = append(flags, "auto_renewal_notice")
	}
	return flags
}

func seedConfidence(level model.RiskLevel) float64 {
	switch level {
	case model.RiskLevelCritical, model.RiskLevelHigh:
		return 0.95
	case model.RiskLevelMedium:
		return 0.85
	case model.RiskLevelLow:
		return 0.70
	default:
		return 0.70
	}
}

func firstKeywords(values []string, limit int) []string {
	if len(values) == 0 || limit <= 0 {
		return nil
	}
	if len(values) < limit {
		limit = len(values)
	}
	out := make([]string, 0, limit)
	for _, value := range values[:limit] {
		out = append(out, value)
	}
	return out
}

func blueprintTitle(clauseType model.ClauseType) string {
	return strings.ReplaceAll(string(clauseType), "_", " ")
}

func contentHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func slugify(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(" ", "-", "/", "-", "&", "and", ",", "", ".", "", "'", "")
	value = replacer.Replace(value)
	for strings.Contains(value, "--") {
		value = strings.ReplaceAll(value, "--", "-")
	}
	return strings.Trim(value, "-")
}

func ptrString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func ptrFloat(value float64) *float64 {
	return &value
}

func normalizeSeedDate(value time.Time) time.Time {
	utc := value.UTC()
	return time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
}

func clampSeedScore(score float64) float64 {
	switch {
	case score < 0:
		return 0
	case score > 100:
		return 100
	default:
		return score
	}
}
