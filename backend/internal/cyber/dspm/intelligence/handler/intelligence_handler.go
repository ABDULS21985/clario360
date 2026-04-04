package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/dto"
	"github.com/clario360/platform/internal/cyber/dspm/intelligence/service"
	"github.com/clario360/platform/internal/middleware"
)

// envelope is a convenience type for JSON response envelopes.
type envelope map[string]any

// IntelligenceHandler handles DSPM advanced intelligence HTTP endpoints.
type IntelligenceHandler struct {
	service *service.IntelligenceService
	logger  zerolog.Logger
}

// NewIntelligenceHandler creates a new IntelligenceHandler.
func NewIntelligenceHandler(svc *service.IntelligenceService, logger zerolog.Logger) *IntelligenceHandler {
	return &IntelligenceHandler{
		service: svc,
		logger:  logger.With().Str("component", "intelligence_handler").Logger(),
	}
}

// --------------------------------------------------------------------------
// Classification
// --------------------------------------------------------------------------

// EnhancedClassification handles GET /dspm/classification/enhanced.
func (h *IntelligenceHandler) EnhancedClassification(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	resp, err := h.service.EnhancedClassification(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("enhanced classification failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "enhanced classification failed")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": resp})
}

// CreateCustomRule handles POST /dspm/classification/custom-rules.
func (h *IntelligenceHandler) CreateCustomRule(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	var req dto.CreateCustomRuleRequest
	if !decodeJSON(w, r, &req) {
		return
	}

	if req.Name == "" || len(req.ColumnPatterns) == 0 || req.Classification == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "name, column_patterns, and classification are required")
		return
	}

	rule, err := h.service.CreateCustomRule(r.Context(), tenantID, &req)
	if err != nil {
		h.logger.Error().Err(err).Msg("create custom rule failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create custom rule")
		return
	}

	writeJSON(w, http.StatusCreated, envelope{"data": rule})
}

// ClassificationHistory handles GET /dspm/classification/history/{assetId}.
func (h *IntelligenceHandler) ClassificationHistory(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	params := &dto.ClassificationHistoryParams{}
	if v := r.URL.Query().Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := r.URL.Query().Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()

	history, total, err := h.service.ClassificationHistory(r.Context(), tenantID, assetID, params)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("classification history failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve classification history")
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		"data":  history,
		"total": total,
		"page":  params.Page,
	})
}

// --------------------------------------------------------------------------
// Lineage
// --------------------------------------------------------------------------

// GetLineageGraph handles GET /dspm/lineage/graph.
func (h *IntelligenceHandler) GetLineageGraph(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	params := &dto.LineageGraphParams{}
	if v := q.Get("classification"); v != "" {
		params.Classification = &v
	}
	if v := q.Get("edge_type"); v != "" {
		params.EdgeType = &v
	}
	if v := q.Get("show_inferred"); v != "" {
		b, _ := strconv.ParseBool(v)
		params.ShowInferred = &b
	}
	if v := q.Get("pii_only"); v != "" {
		b, _ := strconv.ParseBool(v)
		params.PIIOnly = &b
	}

	graph, err := h.service.GetLineageGraph(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("get lineage graph failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve lineage graph")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": graph})
}

// GetUpstream handles GET /dspm/lineage/upstream/{assetId}.
func (h *IntelligenceHandler) GetUpstream(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	depth := 3
	if v := r.URL.Query().Get("depth"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			depth = d
		}
	}

	graph, err := h.service.GetUpstream(r.Context(), tenantID, assetID, depth)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("get upstream failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve upstream lineage")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": graph})
}

// GetDownstream handles GET /dspm/lineage/downstream/{assetId}.
func (h *IntelligenceHandler) GetDownstream(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	depth := 3
	if v := r.URL.Query().Get("depth"); v != "" {
		if d, err := strconv.Atoi(v); err == nil {
			depth = d
		}
	}

	graph, err := h.service.GetDownstream(r.Context(), tenantID, assetID, depth)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("get downstream failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve downstream lineage")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": graph})
}

// GetImpactAnalysis handles GET /dspm/lineage/impact/{assetId}.
func (h *IntelligenceHandler) GetImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	result, err := h.service.GetImpactAnalysis(r.Context(), tenantID, assetID)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("impact analysis failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to perform impact analysis")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": result})
}

// GetPIIFlow handles GET /dspm/lineage/pii-flow.
func (h *IntelligenceHandler) GetPIIFlow(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	graph, err := h.service.GetPIIFlow(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get PII flow failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve PII flow")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": graph})
}

// --------------------------------------------------------------------------
// AI Security
// --------------------------------------------------------------------------

// ListAIUsage handles GET /dspm/ai/usage.
func (h *IntelligenceHandler) ListAIUsage(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	params := &dto.AIUsageListParams{}
	if v := q.Get("usage_type"); v != "" {
		params.UsageType = &v
	}
	if v := q.Get("risk_level"); v != "" {
		params.RiskLevel = &v
	}
	if v := q.Get("model_slug"); v != "" {
		params.ModelSlug = &v
	}
	if v := q.Get("pii_only"); v != "" {
		b, _ := strconv.ParseBool(v)
		params.PIIOnly = &b
	}
	if v := q.Get("status"); v != "" {
		params.Status = &v
	}
	params.Sort = q.Get("sort")
	params.Order = q.Get("order")
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()

	usages, total, err := h.service.ListAIUsage(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("list AI usage failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list AI usage")
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		"data":  usages,
		"total": total,
		"page":  params.Page,
	})
}

// GetAIUsageByAsset handles GET /dspm/ai/usage/{assetId}.
func (h *IntelligenceHandler) GetAIUsageByAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	usages, err := h.service.GetAIUsageByAsset(r.Context(), tenantID, assetID)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("get AI usage by asset failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve AI usage for asset")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": usages})
}

// GetModelDataGovernance handles GET /dspm/ai/models/{modelSlug}/data.
func (h *IntelligenceHandler) GetModelDataGovernance(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	modelSlug := chi.URLParam(r, "modelSlug")
	if modelSlug == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "model slug is required")
		return
	}

	assessment, err := h.service.GetModelDataGovernance(r.Context(), tenantID, modelSlug)
	if err != nil {
		h.logger.Error().Err(err).Str("model_slug", modelSlug).Msg("model data governance failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve model governance data")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": assessment})
}

// GetAIRiskRanking handles GET /dspm/ai/risk-ranking.
func (h *IntelligenceHandler) GetAIRiskRanking(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	ranking, err := h.service.GetAIRiskRanking(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get AI risk ranking failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve AI risk ranking")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": ranking})
}

// GetAIDashboard handles GET /dspm/ai/dashboard.
func (h *IntelligenceHandler) GetAIDashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	dashboard, err := h.service.GetAIDashboard(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get AI dashboard failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve AI dashboard")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": dashboard})
}

// --------------------------------------------------------------------------
// Financial
// --------------------------------------------------------------------------

// GetPortfolioRisk handles GET /dspm/financial/impact.
func (h *IntelligenceHandler) GetPortfolioRisk(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	risk, err := h.service.GetPortfolioRisk(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get portfolio risk failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve portfolio risk")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": risk})
}

// GetAssetFinancialImpact handles GET /dspm/financial/impact/{assetId}.
func (h *IntelligenceHandler) GetAssetFinancialImpact(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	impact, err := h.service.GetAssetFinancialImpact(r.Context(), tenantID, assetID)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("get asset financial impact failed")
		writeError(w, http.StatusNotFound, "NOT_FOUND", "financial impact not found for asset")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": impact})
}

// GetTopFinancialRisks handles GET /dspm/financial/top-risks.
func (h *IntelligenceHandler) GetTopFinancialRisks(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	limit := 10
	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 {
			limit = l
		}
	}

	risks, err := h.service.GetTopFinancialRisks(r.Context(), tenantID, limit)
	if err != nil {
		h.logger.Error().Err(err).Msg("get top financial risks failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve top financial risks")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": risks})
}

// --------------------------------------------------------------------------
// Compliance
// --------------------------------------------------------------------------

// GetCompliancePosture handles GET /dspm/compliance/posture.
func (h *IntelligenceHandler) GetCompliancePosture(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	postures, err := h.service.GetCompliancePosture(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get compliance posture failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve compliance posture")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": postures})
}

// GetFrameworkPosture handles GET /dspm/compliance/posture/{framework}.
func (h *IntelligenceHandler) GetFrameworkPosture(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	framework := chi.URLParam(r, "framework")
	if framework == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "framework is required")
		return
	}

	posture, err := h.service.GetFrameworkPosture(r.Context(), tenantID, framework)
	if err != nil {
		h.logger.Error().Err(err).Str("framework", framework).Msg("get framework posture failed")
		writeError(w, http.StatusNotFound, "NOT_FOUND", "compliance posture not found for framework")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": posture})
}

// GetComplianceGaps handles GET /dspm/compliance/gaps.
func (h *IntelligenceHandler) GetComplianceGaps(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	q := r.URL.Query()
	params := &dto.ComplianceGapParams{}
	if v := q.Get("framework"); v != "" {
		params.Framework = &v
	}
	if v := q.Get("severity"); v != "" {
		params.Severity = &v
	}
	if v := q.Get("page"); v != "" {
		params.Page, _ = strconv.Atoi(v)
	}
	if v := q.Get("per_page"); v != "" {
		params.PerPage, _ = strconv.Atoi(v)
	}
	params.SetDefaults()

	gaps, total, err := h.service.GetComplianceGaps(r.Context(), tenantID, params)
	if err != nil {
		h.logger.Error().Err(err).Msg("get compliance gaps failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve compliance gaps")
		return
	}

	writeJSON(w, http.StatusOK, envelope{
		"data":  gaps,
		"total": total,
		"page":  params.Page,
	})
}

// GetResidencyAnalysis handles GET /dspm/compliance/residency.
func (h *IntelligenceHandler) GetResidencyAnalysis(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	violations, err := h.service.GetResidencyAnalysis(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get residency analysis failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to perform residency analysis")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": violations})
}

// GenerateAuditReport handles POST /dspm/compliance/audit-report/{framework}.
func (h *IntelligenceHandler) GenerateAuditReport(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	framework := chi.URLParam(r, "framework")
	if framework == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "framework is required")
		return
	}

	report, err := h.service.GenerateAuditReport(r.Context(), tenantID, framework)
	if err != nil {
		h.logger.Error().Err(err).Str("framework", framework).Msg("generate audit report failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to generate audit report")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": report})
}

// --------------------------------------------------------------------------
// Proliferation
// --------------------------------------------------------------------------

// GetProliferationOverview handles GET /dspm/proliferation/overview.
func (h *IntelligenceHandler) GetProliferationOverview(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	overview, err := h.service.GetProliferationOverview(r.Context(), tenantID)
	if err != nil {
		h.logger.Error().Err(err).Msg("get proliferation overview failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve proliferation overview")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": overview})
}

// GetAssetProliferation handles GET /dspm/proliferation/{assetId}.
func (h *IntelligenceHandler) GetAssetProliferation(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := requireTenant(w, r)
	if !ok {
		return
	}

	assetID, ok := parseUUIDParam(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}

	prolif, err := h.service.GetAssetProliferation(r.Context(), tenantID, assetID)
	if err != nil {
		h.logger.Error().Err(err).Str("asset_id", assetID.String()).Msg("get asset proliferation failed")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve asset proliferation")
		return
	}

	writeJSON(w, http.StatusOK, envelope{"data": prolif})
}

// --------------------------------------------------------------------------
// Helper functions
// --------------------------------------------------------------------------

// requireTenant extracts and validates the tenant ID from the request context.
func requireTenant(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	tenantStr := auth.TenantFromContext(r.Context())
	if tenantStr == "" {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "tenant context is required")
		return uuid.Nil, false
	}
	tenantID, err := uuid.Parse(tenantStr)
	if err != nil {
		writeError(w, http.StatusForbidden, "FORBIDDEN", "invalid tenant ID")
		return uuid.Nil, false
	}
	return tenantID, true
}

// writeJSON serializes the given value as JSON and writes it to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a JSON error response.
func writeError(w http.ResponseWriter, status int, code, message string) {
	if status >= http.StatusInternalServerError {
		code = "INTERNAL_ERROR"
		message = "internal server error"
	}
	writeJSON(w, status, map[string]any{
		"code":       code,
		"message":    message,
		"request_id": w.Header().Get(middleware.RequestIDHeader),
	})
}

// parseUUIDParam parses a string as a UUID, writing an error response on failure.
func parseUUIDParam(w http.ResponseWriter, raw string) (uuid.UUID, bool) {
	id, err := uuid.Parse(raw)
	if err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("invalid UUID: %s", raw))
		return uuid.Nil, false
	}
	return id, true
}

// decodeJSON reads and decodes the request body as JSON.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 4<<20) // 4MB
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "request body must be valid JSON")
		return false
	}
	return true
}
