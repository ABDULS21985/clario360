package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	iamauth "github.com/clario360/platform/internal/auth"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.RegisterRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.registrationSvc.Register(r.Context(), req, getIPAddress(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.VerifyEmailRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.registrationSvc.VerifyEmail(r.Context(), req, getIPAddress(r), r.UserAgent())
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.ResendOTPRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.registrationSvc.ResendOTP(r.Context(), req.Email, getIPAddress(r), r.UserAgent()); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Verification code resent."})
}

func (h *Handler) GetOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantId")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser != nil && currentUser.TenantID != tenantID.String() && !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "forbidden")
		return
	}

	status, err := h.provisioningRepo.GetStatus(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}
