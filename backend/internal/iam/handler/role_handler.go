package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	iamauth "github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/service"
)

type RoleHandler struct {
	roleSvc *service.RoleService
	logger  zerolog.Logger
}

func NewRoleHandler(roleSvc *service.RoleService, logger zerolog.Logger) *RoleHandler {
	return &RoleHandler{roleSvc: roleSvc, logger: logger}
}

func (h *RoleHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Get("/{roleSlug}/users", h.ListUsersByRole)
	return r
}

// UserRoleRoutes returns routes for /users/{id}/roles endpoints.
func (h *RoleHandler) UserRoleRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/", h.AssignRole)
	r.Delete("/{roleId}", h.RemoveRole)
	return r
}

func (h *RoleHandler) List(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	roles, err := h.roleSvc.List(r.Context(), user.TenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, roles)
}

func (h *RoleHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	roleID := urlParam(r, "id")

	resp, err := h.roleSvc.GetByID(r.Context(), roleID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RoleHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.CreateRoleRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.roleSvc.Create(r.Context(), user.TenantID, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *RoleHandler) Update(w http.ResponseWriter, r *http.Request) {
	roleID := urlParam(r, "id")

	var req dto.UpdateRoleRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.roleSvc.Update(r.Context(), roleID, &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *RoleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	roleID := urlParam(r, "id")

	if err := h.roleSvc.Delete(r.Context(), roleID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "role deleted"})
}

func (h *RoleHandler) AssignRole(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := urlParam(r, "id")

	var req dto.AssignRoleRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.roleSvc.AssignRole(r.Context(), userID, &req, currentUser.TenantID, currentUser.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "role assigned"})
}

func (h *RoleHandler) ListUsersByRole(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	roleSlug := urlParam(r, "roleSlug")

	users, err := h.roleSvc.ListUsersByRole(r.Context(), user.TenantID, roleSlug)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *RoleHandler) InternalUserIDsByRole(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Internal-Service") == "" {
		writeError(w, http.StatusUnauthorized, "internal access required")
		return
	}

	tenantID := strings.TrimSpace(r.URL.Query().Get("tenant_id"))
	roleSlug := strings.TrimSpace(r.URL.Query().Get("role"))
	if tenantID == "" || roleSlug == "" {
		writeError(w, http.StatusBadRequest, "tenant_id and role are required")
		return
	}

	userIDs, err := h.roleSvc.ListUserIDsByRole(r.Context(), tenantID, roleSlug)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string][]string{"user_ids": userIDs})
}

func (h *RoleHandler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	userID := urlParam(r, "id")

	roles, err := h.roleSvc.GetUserRoles(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, roles)
}

func (h *RoleHandler) RemoveRole(w http.ResponseWriter, r *http.Request) {
	userID := urlParam(r, "id")
	roleID := urlParam(r, "roleId")

	if err := h.roleSvc.RemoveRole(r.Context(), userID, roleID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "role removed"})
}
