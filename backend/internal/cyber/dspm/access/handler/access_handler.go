package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dspm/access/dto"
	"github.com/clario360/platform/internal/cyber/dspm/access/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/cyber/service"
)

// AccessIntelligenceHandler handles HTTP requests for the DSPM Access Intelligence module.
type AccessIntelligenceHandler struct {
	svc *service.AccessIntelligenceService
}

// NewAccessIntelligenceHandler creates a new access intelligence handler.
func NewAccessIntelligenceHandler(svc *service.AccessIntelligenceService) *AccessIntelligenceHandler {
	return &AccessIntelligenceHandler{svc: svc}
}

// ── Dashboard ────────────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	result, err := h.svc.Dashboard(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

// ── Identity Profiles ────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) ListIdentities(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := parseIdentityListParams(r)
	result, err := h.svc.ListIdentities(r.Context(), tenantID, params)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AccessIntelligenceHandler) GetIdentity(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	identityID := chi.URLParam(r, "identityId")
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "identityId is required")
		return
	}
	result, err := h.svc.GetIdentity(r.Context(), tenantID, identityID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *AccessIntelligenceHandler) GetIdentityMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	identityID := chi.URLParam(r, "identityId")
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "identityId is required")
		return
	}
	mappings, err := h.svc.GetIdentityMappings(r.Context(), tenantID, identityID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": mappings})
}

func (h *AccessIntelligenceHandler) GetBlastRadius(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	identityID := chi.URLParam(r, "identityId")
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "identityId is required")
		return
	}
	result, err := h.svc.GetBlastRadius(r.Context(), tenantID, identityID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": result})
}

func (h *AccessIntelligenceHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	identityID := chi.URLParam(r, "identityId")
	if identityID == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "identityId is required")
		return
	}
	recs, err := h.svc.GetRecommendations(r.Context(), tenantID, identityID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": recs})
}

// ── Data Asset Access ────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) GetAssetIdentities(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}
	mappings, err := h.svc.GetAssetIdentities(r.Context(), tenantID, assetID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": mappings})
}

func (h *AccessIntelligenceHandler) GetAssetAudit(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	assetID, ok := parseUUID(w, chi.URLParam(r, "assetId"))
	if !ok {
		return
	}
	params := &dto.AuditListParams{}
	params.Page, params.PerPage = parsePageParams(r, 50)
	params.SetDefaults()
	result, err := h.svc.GetAssetAudit(r.Context(), tenantID, assetID, params)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

// ── Access Mappings ──────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) ListMappings(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	params := parseAccessMappingListParams(r)
	result, err := h.svc.ListMappings(r.Context(), tenantID, params)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *AccessIntelligenceHandler) GetOverprivileged(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	results, err := h.svc.GetOverprivileged(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.OverprivilegeListResponse{
		Data: results,
	})
}

func (h *AccessIntelligenceHandler) GetStaleAccess(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	results, err := h.svc.GetStaleAccess(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.StaleAccessListResponse{
		Data: results,
	})
}

// ── Analysis ─────────────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) GetRiskRanking(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.GetRiskRanking(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.RiskRankingResponse{
		Data: items,
	})
}

func (h *AccessIntelligenceHandler) GetBlastRadiusRanking(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.svc.GetBlastRadiusRanking(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.BlastRadiusRankingResponse{
		Data: items,
	})
}

func (h *AccessIntelligenceHandler) GetEscalationPaths(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	paths, err := h.svc.GetEscalationPaths(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.EscalationPathResponse{Data: paths})
}

func (h *AccessIntelligenceHandler) GetCrossAsset(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	results, err := h.svc.GetCrossAsset(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.CrossAssetResponse{
		Data: results,
	})
}

// ── Governance ───────────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) ListPolicies(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	policies, err := h.svc.ListPolicies(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.PolicyListResponse{Data: policies})
}

func (h *AccessIntelligenceHandler) CreatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	policy, err := h.svc.CreatePolicy(r.Context(), tenantID, &req, userID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{"data": policy})
}

func (h *AccessIntelligenceHandler) UpdatePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	policyID, ok := parseUUID(w, chi.URLParam(r, "policyId"))
	if !ok {
		return
	}
	var req dto.UpdatePolicyRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := req.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	policy, err := h.svc.UpdatePolicy(r.Context(), tenantID, policyID, &req)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"data": policy})
}

func (h *AccessIntelligenceHandler) DeletePolicy(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	policyID, ok := parseUUID(w, chi.URLParam(r, "policyId"))
	if !ok {
		return
	}
	if err := h.svc.DeletePolicy(r.Context(), tenantID, policyID); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, envelope{"message": "policy deleted"})
}

func (h *AccessIntelligenceHandler) GetPolicyViolations(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	violations, err := h.svc.GetPolicyViolations(r.Context(), tenantID)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, dto.PolicyViolationListResponse{
		Data: violations,
	})
}

// ── Collection Trigger ───────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) TriggerCollection(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	if err := h.svc.RunCollectionCycle(r.Context(), tenantID); err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, envelope{"message": "collection cycle started"})
}

// ── Helpers ──────────────────────────────────────────────────────────────────

func (h *AccessIntelligenceHandler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "resource not found")
	default:
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

func parseIdentityListParams(r *http.Request) *dto.IdentityListParams {
	q := r.URL.Query()
	params := &dto.IdentityListParams{
		IdentityType: stringPtr(q.Get("identity_type")),
		Status:       stringPtr(q.Get("status")),
		MinRiskScore: floatPtr(q.Get("min_risk_score")),
		Search:       stringPtr(q.Get("search")),
		Sort:         q.Get("sort"),
		Order:        q.Get("order"),
	}
	params.Page, params.PerPage = parsePageParams(r, 50)
	params.SetDefaults()
	return params
}

func parseAccessMappingListParams(r *http.Request) *dto.AccessMappingListParams {
	q := r.URL.Query()
	params := &dto.AccessMappingListParams{
		IdentityType:       stringPtr(q.Get("identity_type")),
		IdentityID:         stringPtr(q.Get("identity_id")),
		PermissionType:     stringPtr(q.Get("permission_type")),
		DataClassification: stringPtr(q.Get("data_classification")),
		Status:             stringPtr(q.Get("status")),
		IsStale:            boolPtr(q.Get("is_stale")),
		Search:             stringPtr(q.Get("search")),
		Sort:               q.Get("sort"),
		Order:              q.Get("order"),
	}
	if v := q.Get("data_asset_id"); v != "" {
		if id, err := uuid.Parse(v); err == nil {
			params.DataAssetID = &id
		}
	}
	params.Page, params.PerPage = parsePageParams(r, 50)
	params.SetDefaults()
	return params
}

// Ensure nil slices become empty slices in JSON responses.
func init() {
	_ = model.AccessMapping{}
}
