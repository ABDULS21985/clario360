package handler

import (
	"net/http"

	"github.com/google/uuid"

	iamauth "github.com/clario360/platform/internal/auth"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func (h *Handler) GetWizardProgress(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	progress, err := h.wizardSvc.GetProgress(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, progress)
}

func (h *Handler) SaveOrganization(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	var req onboardingdto.OrganizationDetailsRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.wizardSvc.SaveOrganization(r.Context(), tenantID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) SaveBranding(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	var req onboardingdto.BrandingRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var primaryColor, accentColor *string
	if req.PrimaryColor != "" {
		primaryColor = &req.PrimaryColor
	}
	if req.AccentColor != "" {
		accentColor = &req.AccentColor
	}

	resp, err := h.wizardSvc.SaveBranding(r.Context(), tenantID, nil, primaryColor, accentColor)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) SaveTeam(w http.ResponseWriter, r *http.Request) {
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

	var req onboardingdto.TeamStepRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.wizardSvc.SaveTeam(r.Context(), tenantID, userID, currentUser.Email, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) SaveSuites(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	var req onboardingdto.SuitesStepRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.wizardSvc.SaveSuites(r.Context(), tenantID, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) CompleteWizard(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	tenantID, err := uuid.Parse(currentUser.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID in token")
		return
	}

	resp, err := h.wizardSvc.Complete(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}
