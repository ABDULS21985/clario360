package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	llmdto "github.com/clario360/platform/internal/cyber/vciso/llm/dto"
	llmengine "github.com/clario360/platform/internal/cyber/vciso/llm/engine"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
	llmrepo "github.com/clario360/platform/internal/cyber/vciso/llm/repository"
	llmtools "github.com/clario360/platform/internal/cyber/vciso/llm/tools"
	"github.com/clario360/platform/internal/suiteapi"
)

// ---------------------------------------------------------------------------
// Error codes — stable identifiers for clients
// ---------------------------------------------------------------------------

const (
	codeUnauthorised = "UNAUTHORIZED"
	codeForbidden    = "FORBIDDEN"
	codeValidation   = "VALIDATION_ERROR"
	codeNotFound     = "NOT_FOUND"
	codeUnavailable  = "LLM_UNAVAILABLE"
	codeInternal     = "INTERNAL_ERROR"
)

// ---------------------------------------------------------------------------
// LLMHandler
// ---------------------------------------------------------------------------

// LLMHandler exposes the LLM engine's administrative and observability
// endpoints over HTTP.  It is responsible only for:
//   - extracting and validating request parameters
//   - authorisation checks
//   - delegating to the engine / repository
//   - mapping results and errors to HTTP responses
//
// Business logic stays in the engine layer.
type LLMHandler struct {
	engine *llmengine.LLMEngine
	repo   *llmrepo.LLMAuditRepository
	logger zerolog.Logger
}

func NewLLMHandler(
	engine *llmengine.LLMEngine,
	repo *llmrepo.LLMAuditRepository,
	logger zerolog.Logger,
) *LLMHandler {
	return &LLMHandler{
		engine: engine,
		repo:   repo,
		logger: logger.With().Str("component", "vciso_llm_handler").Logger(),
	}
}

// ===========================================================================
// Endpoints
// ===========================================================================

// --- Audit ----------------------------------------------------------------

func (h *LLMHandler) Audit(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := h.requireTenantAndUser(w, r)
	if !ok {
		return
	}

	messageID, err := uuid.Parse(chi.URLParam(r, "message_id"))
	if err != nil {
		h.writeValidation(w, r, "invalid message_id: must be a valid UUID")
		return
	}

	item, err := h.repo.GetAuditByMessageID(r.Context(), tenantID, messageID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	suiteapi.WriteJSON(w, http.StatusOK, h.mapAuditResponse(item))
}

// --- Usage ----------------------------------------------------------------

func (h *LLMHandler) Usage(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := h.requireTenantAndUser(w, r)
	if !ok {
		return
	}

	stats, err := h.engine.Usage(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	suiteapi.WriteJSON(w, http.StatusOK, llmdto.UsageResponse{
		CallsToday:     stats.CallsToday,
		TokensToday:    stats.TokensToday,
		CostToday:      stats.CostToday,
		CallsThisMonth: stats.CallsThisMonth,
		CostThisMonth:  stats.CostThisMonth,
	})
}

// --- Health ---------------------------------------------------------------

func (h *LLMHandler) Health(w http.ResponseWriter, r *http.Request) {
	tenantID, _, ok := h.requireTenantAndUser(w, r)
	if !ok {
		return
	}

	status, err := h.engine.Health(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	suiteapi.WriteJSON(w, http.StatusOK, llmdto.HealthResponse{
		Provider:           status.Provider,
		Model:              status.Model,
		Status:             status.Status,
		LatencyMS:          status.LatencyMS,
		RateLimitRemaining: status.RateLimitRemaining,
	})
}

// --- Config (admin) -------------------------------------------------------

func (h *LLMHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	tenantID, userLabel, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	var req llmdto.UpdateConfigRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		h.writeValidation(w, r, "invalid request body")
		return
	}

	if h.engine == nil || h.engine.ProviderManager() == nil {
		h.writeUnavailable(w, r)
		return
	}

	override := h.engine.ProviderManager().UpdateConfig(tenantID, req)

	h.logAdminAction(r, "update_config", userLabel, tenantID, nil)
	suiteapi.WriteJSON(w, http.StatusOK, override)
}

// --- Prompts (admin) ------------------------------------------------------

func (h *LLMHandler) ListPrompts(w http.ResponseWriter, r *http.Request) {
	if _, _, ok := h.requireAdmin(w, r); !ok {
		return
	}

	items, err := h.repo.ListPrompts(r.Context())
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	response := make([]llmdto.PromptVersionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, h.mapPromptResponse(item))
	}
	suiteapi.WriteJSON(w, http.StatusOK, response)
}

func (h *LLMHandler) CreatePrompt(w http.ResponseWriter, r *http.Request) {
	tenantID, userLabel, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	var req llmdto.PromptVersionRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		h.writeValidation(w, r, "invalid request body")
		return
	}

	// --- Validate fields ---
	if problems := validatePromptRequest(req); len(problems) > 0 {
		h.writeValidation(w, r, strings.Join(problems, "; "))
		return
	}

	item := &llmmodel.SystemPrompt{
		ID:         uuid.New(),
		Version:    strings.TrimSpace(req.Version),
		PromptText: strings.TrimSpace(req.PromptText),
		PromptHash: llmengine.HashPrompt(req.PromptText),
		CreatedBy:  userLabel,
		Active:     false,
	}

	item.ToolSchemas = h.snapshotToolSchemas(tenantID)

	if desc := strings.TrimSpace(req.Description); desc != "" {
		item.Description = &desc
	}

	if err := h.repo.CreatePrompt(r.Context(), item); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logAdminAction(r, "create_prompt", userLabel, tenantID, map[string]any{
		"prompt_id": item.ID,
		"version":   item.Version,
	})

	suiteapi.WriteJSON(w, http.StatusCreated, h.mapPromptResponse(*item))
}

func (h *LLMHandler) ActivatePrompt(w http.ResponseWriter, r *http.Request) {
	tenantID, userLabel, ok := h.requireAdmin(w, r)
	if !ok {
		return
	}

	version := strings.TrimSpace(chi.URLParam(r, "version"))
	if version == "" {
		h.writeValidation(w, r, "version path parameter is required")
		return
	}

	if err := h.repo.ActivatePrompt(r.Context(), version); err != nil {
		h.writeError(w, r, err)
		return
	}

	h.logAdminAction(r, "activate_prompt", userLabel, tenantID, map[string]any{
		"version": version,
	})

	w.WriteHeader(http.StatusNoContent)
}

// ===========================================================================
// Auth helpers
// ===========================================================================

// requireTenantAndUser extracts and validates both IDs from the request
// context.  On failure it writes the HTTP error and returns ok=false.
func (h *LLMHandler) requireTenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, codeUnauthorised, "missing tenant context", nil)
		return uuid.Nil, uuid.Nil, false
	}

	userID, err := suiteapi.UserID(r)
	if err != nil || userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, codeUnauthorised, "missing user context", nil)
		return uuid.Nil, uuid.Nil, false
	}

	return tenantID, *userID, true
}

// requireAdmin validates tenant+user AND checks LLM admin permission.
// Returns the tenant ID and the user label (email or "system") for audit.
func (h *LLMHandler) requireAdmin(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	tenantID, _, ok := h.requireTenantAndUser(w, r)
	if !ok {
		return uuid.Nil, "", false
	}

	if !isLLMAdmin(r) {
		suiteapi.WriteError(w, r, http.StatusForbidden, codeForbidden, "vciso llm admin permission required", nil)
		return uuid.Nil, "", false
	}

	return tenantID, currentUserLabel(r), true
}

// isLLMAdmin checks both claims-based and user-based role sources.
func isLLMAdmin(r *http.Request) bool {
	ctx := r.Context()

	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		if hasAdminPerm(claims.Roles) {
			return true
		}
	}

	if user := auth.UserFromContext(ctx); user != nil {
		return hasAdminPerm(user.Roles)
	}

	return false
}

func hasAdminPerm(roles []string) bool {
	return auth.HasPermission(roles, auth.PermVCISOLLMAdmin) ||
		auth.HasPermission(roles, auth.PermAdminAll)
}

func currentUserLabel(r *http.Request) string {
	if user := auth.UserFromContext(r.Context()); user != nil {
		if email := strings.TrimSpace(user.Email); email != "" {
			return email
		}
	}
	return "system"
}

// ===========================================================================
// Validation
// ===========================================================================

func validatePromptRequest(req llmdto.PromptVersionRequest) []string {
	var problems []string

	if strings.TrimSpace(req.Version) == "" {
		problems = append(problems, "version is required")
	}
	if strings.TrimSpace(req.PromptText) == "" {
		problems = append(problems, "prompt_text is required")
	}
	if len(req.PromptText) > 100_000 {
		problems = append(problems, "prompt_text exceeds maximum length (100,000 chars)")
	}

	return problems
}

// ===========================================================================
// Response mapping (handler ↔ DTO boundary)
// ===========================================================================

func (h *LLMHandler) mapAuditResponse(item *llmmodel.AuditLog) llmdto.AuditResponse {
	toolCalls := make([]llmmodel.ToolCallAudit, 0)
	_ = json.Unmarshal(item.ToolCallsJSON, &toolCalls)

	reasoning := make([]llmmodel.ReasoningStep, 0)
	_ = json.Unmarshal(item.ReasoningTrace, &reasoning)

	return llmdto.AuditResponse{
		MessageID:        item.MessageID,
		Provider:         item.Provider,
		Model:            item.Model,
		PromptTokens:     item.PromptTokens,
		CompletionTokens: item.CompletionTokens,
		TotalTokens:      item.TotalTokens,
		ToolCalls:        toolCalls,
		ReasoningTrace:   reasoning,
		GroundingResult:  item.GroundingResult,
		EngineUsed:       item.EngineUsed,
		RoutingReason:    item.RoutingReason,
		CreatedAt:        item.CreatedAt,
	}
}

func (h *LLMHandler) mapPromptResponse(item llmmodel.SystemPrompt) llmdto.PromptVersionResponse {
	description := ""
	if item.Description != nil {
		description = *item.Description
	}
	return llmdto.PromptVersionResponse{
		ID:          item.ID,
		Version:     item.Version,
		Description: description,
		Active:      item.Active,
		CreatedBy:   item.CreatedBy,
		CreatedAt:   item.CreatedAt,
	}
}

// ===========================================================================
// Tool schema snapshot
// ===========================================================================

// snapshotToolSchemas captures the current tool schema set for embedding in
// a prompt version.  Returns nil (not an error) if the engine or registry
// is unavailable — a prompt can exist without embedded schemas.
func (h *LLMHandler) snapshotToolSchemas(tenantID uuid.UUID) json.RawMessage {
	if h.engine == nil {
		return nil
	}

	registry := h.engine.ToolRegistry()
	if registry == nil {
		return nil
	}

	pm := h.engine.ProviderManager()
	if pm == nil {
		return nil
	}

	providerName := pm.GetConfig(tenantID).Provider
	if providerName == "" {
		providerName = "openai"
	}

	schemas, err := json.Marshal(llmtools.GenerateToolSchemas(registry.List(), providerName))
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to marshal tool schemas for prompt snapshot")
		return nil
	}

	return schemas
}

// ===========================================================================
// Error writing — centralised HTTP error mapping
// ===========================================================================

// writeError maps domain/repository errors to appropriate HTTP responses.
// Uses errors.Is for sentinel matching so wrapped errors still resolve.
func (h *LLMHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, llmrepo.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, codeNotFound, "resource not found", nil)

	case errors.Is(err, llmengine.ErrRateLimited):
		suiteapi.WriteError(w, r, http.StatusTooManyRequests, "RATE_LIMITED", "rate limit exceeded, try again later", nil)

	case errors.Is(err, llmengine.ErrProviderResolveFailed),
		errors.Is(err, llmengine.ErrProviderCallFailed):
		h.logger.Warn().Err(err).Msg("llm provider error")
		suiteapi.WriteError(w, r, http.StatusBadGateway, "PROVIDER_ERROR", "upstream LLM provider error", nil)

	case errors.Is(err, llmengine.ErrContextCancelled):
		// Client disconnected — log but don't try to write a response.
		h.logger.Debug().Msg("request context cancelled")

	default:
		h.logger.Error().Err(err).Msg("vciso llm request failed")
		suiteapi.WriteError(w, r, http.StatusInternalServerError, codeInternal, "request failed", nil)
	}
}

func (h *LLMHandler) writeValidation(w http.ResponseWriter, r *http.Request, detail string) {
	suiteapi.WriteError(w, r, http.StatusBadRequest, codeValidation, detail, nil)
}

func (h *LLMHandler) writeUnavailable(w http.ResponseWriter, r *http.Request) {
	suiteapi.WriteError(w, r, http.StatusServiceUnavailable, codeUnavailable, "llm engine is unavailable", nil)
}

// ===========================================================================
// Admin audit logging
// ===========================================================================

// logAdminAction writes a structured log entry for every admin mutation.
// This is intentionally handler-level (not engine-level) because it captures
// HTTP-specific context: who, from where, which endpoint.
func (h *LLMHandler) logAdminAction(
	r *http.Request, action, actor string, tenantID uuid.UUID, extra map[string]any,
) {
	event := h.logger.Info().
		Str("admin_action", action).
		Str("actor", actor).
		Str("tenant_id", tenantID.String()).
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr).
		Time("at", time.Now().UTC())

	for k, v := range extra {
		event = event.Interface(k, v)
	}

	event.Msg("admin action performed")
}

