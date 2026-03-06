package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	iamauth "github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/service"
)

type UserHandler struct {
	userSvc *service.UserService
	logger  zerolog.Logger
}

func NewUserHandler(userSvc *service.UserService, logger zerolog.Logger) *UserHandler {
	return &UserHandler{userSvc: userSvc, logger: logger}
}

func (h *UserHandler) Routes() chi.Router {
	r := chi.NewRouter()

	// /users/me routes (must be before /{id} to avoid conflict)
	r.Get("/me", h.GetProfile)
	r.Put("/me/password", h.ChangePassword)
	r.Post("/me/mfa/enable", h.EnableMFA)
	r.Post("/me/mfa/verify-setup", h.VerifyMFASetup)
	r.Post("/me/mfa/disable", h.DisableMFA)

	// /users CRUD
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
	r.Put("/{id}/status", h.UpdateStatus)

	return r
}

func (h *UserHandler) List(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, perPage := parsePagination(r)
	search := r.URL.Query().Get("search")
	status := r.URL.Query().Get("status")

	users, total, err := h.userSvc.List(r.Context(), user.TenantID, page, perPage, search, status)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginatedResponse(users, total, page, perPage))
}

func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := urlParam(r, "id")

	resp, err := h.userSvc.GetByID(r.Context(), userID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) Update(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := urlParam(r, "id")

	// Allow self-update or admin
	if userID != currentUser.ID && !iamauth.HasPermission(currentUser.Roles, "users:*") {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.UpdateUserRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.userSvc.Update(r.Context(), userID, &req, currentUser.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := urlParam(r, "id")

	if err := h.userSvc.Delete(r.Context(), userID, currentUser.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "user deleted"})
}

func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	userID := urlParam(r, "id")

	var req dto.UpdateStatusRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.userSvc.UpdateStatus(r.Context(), userID, &req, currentUser.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "user status updated"})
}

func (h *UserHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.userSvc.GetByID(r.Context(), currentUser.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.ChangePasswordRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.userSvc.ChangePassword(r.Context(), currentUser.ID, &req); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "password changed successfully"})
}

func (h *UserHandler) EnableMFA(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	resp, err := h.userSvc.EnableMFA(r.Context(), currentUser.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *UserHandler) VerifyMFASetup(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.DisableMFARequest // reuse — same shape: {code: "123456"}
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.userSvc.VerifyMFASetup(r.Context(), currentUser.ID, req.Code); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "MFA enabled successfully"})
}

func (h *UserHandler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dto.DisableMFARequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.userSvc.DisableMFA(r.Context(), currentUser.ID, &req); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "MFA disabled"})
}
