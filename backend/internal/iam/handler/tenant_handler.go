package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	iamauth "github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/service"
)

type TenantHandler struct {
	tenantSvc *service.TenantService
	logger    zerolog.Logger
}

func NewTenantHandler(tenantSvc *service.TenantService, logger zerolog.Logger) *TenantHandler {
	return &TenantHandler{tenantSvc: tenantSvc, logger: logger}
}

func (h *TenantHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Put("/{id}/status", h.UpdateStatus)
	return r
}

func (h *TenantHandler) List(w http.ResponseWriter, r *http.Request) {
	page, perPage := parsePagination(r)

	tenants, total, err := h.tenantSvc.List(r.Context(), page, perPage)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginatedResponse(tenants, total, page, perPage))
}

func (h *TenantHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateTenantRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.tenantSvc.Create(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *TenantHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	tenantID := urlParam(r, "id")

	// Super-admin can view any tenant; tenant admin can view own tenant
	if currentUser != nil && currentUser.TenantID != tenantID {
		if !iamauth.HasPermission(currentUser.Roles, "*") {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	resp, err := h.tenantSvc.GetByID(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TenantHandler) Update(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	tenantID := urlParam(r, "id")

	// Super-admin can update any tenant; tenant admin can update own tenant
	if currentUser != nil && currentUser.TenantID != tenantID {
		if !iamauth.HasPermission(currentUser.Roles, "*") {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	var req dto.UpdateTenantRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.tenantSvc.Update(r.Context(), tenantID, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *TenantHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	tenantID := urlParam(r, "id")

	// Only super-admin can change tenant status
	if currentUser == nil || !iamauth.HasPermission(currentUser.Roles, "*") {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.UpdateStatusRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	statusReq := &dto.UpdateTenantRequest{Status: &req.Status}
	resp, err := h.tenantSvc.Update(r.Context(), tenantID, statusReq)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
