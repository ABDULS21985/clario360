package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	iamauth "github.com/clario360/platform/internal/auth"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func (h *Handler) ValidateInviteToken(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == "" {
		writeError(w, http.StatusBadRequest, "token query parameter is required")
		return
	}

	details, err := h.invitationSvc.ValidateToken(r.Context(), token)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, details)
}

func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.AcceptInviteRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.invitationSvc.Accept(r.Context(), req, getIPAddress(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	invitations, err := h.invitationSvc.List(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": invitations})
}

func (h *Handler) CreateBatchInvitations(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}
	userID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user ID in token")
		return
	}
	if !iamauth.HasAnyPermission(currentUser.Roles, iamauth.PermUserWrite, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "insufficient permissions to invite users")
		return
	}

	var req onboardingdto.BatchInviteRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	invitations, err := h.invitationSvc.CreateBatch(r.Context(), tenantID, userID, currentUser.Email, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": invitations, "count": len(invitations)})
}

func (h *Handler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}
	if !iamauth.HasAnyPermission(currentUser.Roles, iamauth.PermUserWrite, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "insufficient permissions to cancel invitations")
		return
	}

	invitationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invitation ID")
		return
	}

	if err := h.invitationSvc.Cancel(r.Context(), tenantID, invitationID); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation cancelled."})
}

func (h *Handler) ResendInvitation(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}
	if !iamauth.HasAnyPermission(currentUser.Roles, iamauth.PermUserWrite, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "insufficient permissions to resend invitations")
		return
	}

	invitationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid invitation ID")
		return
	}

	if err := h.invitationSvc.Resend(r.Context(), tenantID, invitationID); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Invitation resent."})
}
