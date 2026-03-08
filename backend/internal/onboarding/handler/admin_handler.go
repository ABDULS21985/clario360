package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	iamauth "github.com/clario360/platform/internal/auth"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
)

func (h *Handler) AdminProvision(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	var req onboardingdto.ManualProvisionRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tenantID, err := uuid.Parse(req.TenantID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	h.startProvisioningAsync(tenantID, "manual admin provision")
	writeJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Provisioning started.",
		"tenant_id": tenantID.String(),
	})
}

func (h *Handler) AdminGetProvisionStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	status, err := h.provisioningRepo.GetStatus(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) AdminDeprovision(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}
	adminID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid admin user ID in token")
		return
	}

	var req onboardingdto.DeprovisionRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.deprovisioner.Deprovision(r.Context(), tenantID, adminID, req); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tenant deprovisioned."})
}

func (h *Handler) AdminReprovision(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	h.startProvisioningAsync(tenantID, "admin reprovision")
	writeJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Reprovisioning started.",
		"tenant_id": tenantID.String(),
	})
}

func (h *Handler) AdminReactivate(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}
	adminID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid admin user ID in token")
		return
	}

	if err := h.deprovisioner.Reactivate(r.Context(), tenantID, adminID); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Tenant reactivated."})
}
