package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	aigovdto "github.com/clario360/platform/internal/aigovernance/dto"
	"github.com/clario360/platform/internal/aigovernance/repository"
	aigovservice "github.com/clario360/platform/internal/aigovernance/service"
	"github.com/clario360/platform/internal/suiteapi"
)

// maxCompareRuns caps the number of run IDs accepted by CompareRuns to prevent
// expensive fan-out queries.
const maxCompareRuns = 10

type BenchmarkHandler struct {
	svc    *aigovservice.BenchmarkService
	logger zerolog.Logger
}

func NewBenchmarkHandler(svc *aigovservice.BenchmarkService, logger zerolog.Logger) *BenchmarkHandler {
	return &BenchmarkHandler{svc: svc, logger: logger.With().Str("handler", "ai_benchmark").Logger()}
}

// ── Inference Servers ───────────────────────────────────────────────────

func (h *BenchmarkHandler) CreateServer(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	var req aigovdto.CreateInferenceServerRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.CreateServer(r.Context(), tid, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *BenchmarkHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.svc.ListServers(r.Context(), tid, repository.ListServersParams{
		BackendType: r.URL.Query().Get("backend_type"),
		Status:      r.URL.Query().Get("status"),
		Page:        page,
		PerPage:     perPage,
	})
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *BenchmarkHandler) GetServer(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	serverID, err := uuid.Parse(chi.URLParam(r, "serverId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid server id", nil)
		return
	}
	item, err := h.svc.GetServer(r.Context(), tid, serverID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *BenchmarkHandler) UpdateServer(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	serverID, err := uuid.Parse(chi.URLParam(r, "serverId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid server id", nil)
		return
	}
	var req aigovdto.UpdateInferenceServerRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateServer(r.Context(), tid, serverID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *BenchmarkHandler) UpdateServerStatus(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	serverID, err := uuid.Parse(chi.URLParam(r, "serverId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid server id", nil)
		return
	}
	var req aigovdto.UpdateInferenceServerStatusRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if err := h.svc.UpdateServerStatus(r.Context(), tid, serverID, req.Status); err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	// Return the updated server so the frontend receives a full AIInferenceServer object.
	updated, err := h.svc.GetServer(r.Context(), tid, serverID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, updated)
}

func (h *BenchmarkHandler) DeleteServer(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	serverID, err := uuid.Parse(chi.URLParam(r, "serverId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid server id", nil)
		return
	}
	item, err := h.svc.DeleteServer(r.Context(), tid, serverID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	// Return the decommissioned server object so clients see the final state.
	suiteapi.WriteData(w, http.StatusOK, item)
}

// ── Benchmark Suites ────────────────────────────────────────────────────

func (h *BenchmarkHandler) CreateSuite(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	var req aigovdto.CreateBenchmarkSuiteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.CreateSuite(r.Context(), tid, *uid, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *BenchmarkHandler) ListSuites(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.svc.ListSuites(r.Context(), tid, page, perPage)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *BenchmarkHandler) GetSuite(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	suiteID, err := uuid.Parse(chi.URLParam(r, "suiteId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid suite id", nil)
		return
	}
	item, err := h.svc.GetSuite(r.Context(), tid, suiteID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *BenchmarkHandler) UpdateSuite(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	suiteID, err := uuid.Parse(chi.URLParam(r, "suiteId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid suite id", nil)
		return
	}
	var req aigovdto.UpdateBenchmarkSuiteRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.UpdateSuite(r.Context(), tid, *uid, suiteID, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *BenchmarkHandler) DeleteSuite(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	suiteID, err := uuid.Parse(chi.URLParam(r, "suiteId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid suite id", nil)
		return
	}
	if err := h.svc.DeleteSuite(r.Context(), tid, *uid, suiteID); err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ── Benchmark Runs ──────────────────────────────────────────────────────

func (h *BenchmarkHandler) RunBenchmark(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	suiteID, err := uuid.Parse(chi.URLParam(r, "suiteId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid suite id", nil)
		return
	}
	var req aigovdto.RunBenchmarkRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.RunBenchmark(r.Context(), tid, *uid, suiteID, req.ServerID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusAccepted, item)
}

func (h *BenchmarkHandler) ListRuns(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)

	var suiteID *uuid.UUID
	if s := r.URL.Query().Get("suite_id"); s != "" {
		parsed, err := uuid.Parse(s)
		if err != nil {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid suite_id query parameter", nil)
			return
		}
		suiteID = &parsed
	}

	items, total, err := h.svc.ListRuns(r.Context(), tid, suiteID, page, perPage)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *BenchmarkHandler) GetRun(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	runID, err := uuid.Parse(chi.URLParam(r, "runId"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_ID", "invalid run id", nil)
		return
	}
	item, err := h.svc.GetRun(r.Context(), tid, runID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *BenchmarkHandler) CompareRuns(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req aigovdto.CompareRunsRequest
	if !decodeBody(w, r, &req) {
		return
	}
	if len(req.RunIDs) < 2 {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_INPUT", "at least two run IDs are required for comparison", nil)
		return
	}
	if len(req.RunIDs) > maxCompareRuns {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "INVALID_INPUT",
			"too many run IDs; maximum is "+string(rune('0'+maxCompareRuns)), nil)
		return
	}
	item, err := h.svc.CompareRuns(r.Context(), tid, req.RunIDs)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

// ── Cost Models ─────────────────────────────────────────────────────────

func (h *BenchmarkHandler) CreateCostModel(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	uid := userID(r)
	if uid == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "authenticated user required", nil)
		return
	}
	var req aigovdto.CreateComputeCostModelRequest
	if !decodeBody(w, r, &req) {
		return
	}
	item, err := h.svc.CreateCostModel(r.Context(), tid, req)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *BenchmarkHandler) ListCostModels(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	items, err := h.svc.ListCostModels(r.Context(), tid)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *BenchmarkHandler) EstimateCostSavings(w http.ResponseWriter, r *http.Request) {
	tid, ok := tenantID(w, r)
	if !ok {
		return
	}
	var req aigovdto.EstimateCostSavingsRequest
	if !decodeBody(w, r, &req) {
		return
	}
	result, err := h.svc.EstimateCostSavings(r.Context(), tid, req.CPURunID, req.GPURunID)
	if err != nil {
		writeError(h.logger, w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, result)
}
