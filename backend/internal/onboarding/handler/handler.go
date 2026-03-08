package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	iamauth "github.com/clario360/platform/internal/auth"
	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
)

// Handler wires together all onboarding HTTP endpoints.
type Handler struct {
	registrationSvc  *onboardingsvc.RegistrationService
	wizardSvc        *onboardingsvc.WizardService
	invitationSvc    *onboardingsvc.InvitationService
	provisioner      *onboardingsvc.TenantProvisioner
	deprovisioner    *onboardingsvc.TenantDeprovisioner
	provisioningRepo *onboardingrepo.ProvisioningRepository
	logger           zerolog.Logger
}

// New constructs an onboarding Handler.
func New(
	registrationSvc *onboardingsvc.RegistrationService,
	wizardSvc *onboardingsvc.WizardService,
	invitationSvc *onboardingsvc.InvitationService,
	provisioner *onboardingsvc.TenantProvisioner,
	deprovisioner *onboardingsvc.TenantDeprovisioner,
	provisioningRepo *onboardingrepo.ProvisioningRepository,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		registrationSvc:  registrationSvc,
		wizardSvc:        wizardSvc,
		invitationSvc:    invitationSvc,
		provisioner:      provisioner,
		deprovisioner:    deprovisioner,
		provisioningRepo: provisioningRepo,
		logger:           logger.With().Str("handler", "onboarding").Logger(),
	}
}

// PublicRoutes returns routes that do not require authentication.
// Mount at /api/v1/onboarding
func (h *Handler) PublicRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/verify-email", h.VerifyEmail)
	r.Post("/resend-otp", h.ResendOTP)
	r.Get("/status/{tenantId}", h.GetOnboardingStatus)
	// Invitation public endpoints
	r.Get("/invitations/validate", h.ValidateInviteToken)
	r.Post("/invitations/accept", h.AcceptInvitation)
	return r
}

// WizardRoutes returns routes for the onboarding wizard (requires auth).
// Mount at /api/v1/onboarding
func (h *Handler) WizardRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/progress", h.GetWizardProgress)
	r.Post("/organization", h.SaveOrganization)
	r.Post("/branding", h.SaveBranding)
	r.Post("/team", h.SaveTeam)
	r.Post("/suites", h.SaveSuites)
	r.Post("/complete", h.CompleteWizard)
	return r
}

// InvitationRoutes returns authenticated invitation management routes.
// Mount at /api/v1/invitations
func (h *Handler) InvitationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListInvitations)
	r.Post("/", h.CreateBatchInvitations)
	r.Post("/{id}/cancel", h.CancelInvitation)
	r.Post("/{id}/resend", h.ResendInvitation)
	r.Delete("/{id}", h.CancelInvitation)
	return r
}

// AdminRoutes returns admin-only provisioning routes.
// Mount at /api/v1/admin/tenants
func (h *Handler) AdminRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/{id}/provision", h.AdminProvision)
	r.Get("/{id}/provision-status", h.AdminGetProvisionStatus)
	r.Post("/{id}/deprovision", h.AdminDeprovision)
	r.Post("/{id}/reactivate", h.AdminReactivate)
	return r
}

// -------------------------------------------------------------------------
// Public: Registration
// -------------------------------------------------------------------------

// Register handles POST /api/v1/onboarding/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.RegisterRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.registrationSvc.Register(r.Context(), req, ip, userAgent)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// VerifyEmail handles POST /api/v1/onboarding/verify-email
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.VerifyEmailRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.registrationSvc.VerifyEmail(r.Context(), req, ip, userAgent)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// ResendOTP handles POST /api/v1/onboarding/resend-otp
func (h *Handler) ResendOTP(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.ResendOTPRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	if err := h.registrationSvc.ResendOTP(r.Context(), req.Email, ip, userAgent); err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Verification code resent."})
}

// GetOnboardingStatus handles GET /api/v1/onboarding/status/{tenantId}
func (h *Handler) GetOnboardingStatus(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := chi.URLParam(r, "tenantId")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	// Restrict: only the tenant itself or a super admin may call this.
	currentUser := iamauth.UserFromContext(r.Context())
	if currentUser != nil && currentUser.TenantID != tenantID.String() {
		if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
			writeError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	status, err := h.provisioningRepo.GetStatus(r.Context(), tenantID)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

// -------------------------------------------------------------------------
// Public: Invitations
// -------------------------------------------------------------------------

// ValidateInviteToken handles GET /api/v1/onboarding/invitations/validate?token=...
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

// AcceptInvitation handles POST /api/v1/onboarding/invitations/accept
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	var req onboardingdto.AcceptInviteRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.invitationSvc.Accept(r.Context(), req, ip, userAgent)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

// -------------------------------------------------------------------------
// Authenticated: Wizard
// -------------------------------------------------------------------------

// GetWizardProgress handles GET /api/v1/onboarding/progress
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

// SaveOrganization handles POST /api/v1/onboarding/organization
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

// SaveBranding handles POST /api/v1/onboarding/branding
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
		c := req.PrimaryColor
		primaryColor = &c
	}
	if req.AccentColor != "" {
		c := req.AccentColor
		accentColor = &c
	}

	resp, err := h.wizardSvc.SaveBranding(r.Context(), tenantID, nil, primaryColor, accentColor)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SaveTeam handles POST /api/v1/onboarding/team
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

	// Use the email from the JWT as the inviter's display name fallback.
	inviterName := currentUser.Email

	resp, err := h.wizardSvc.SaveTeam(r.Context(), tenantID, userID, inviterName, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// SaveSuites handles POST /api/v1/onboarding/suites
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

// CompleteWizard handles POST /api/v1/onboarding/complete
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

// -------------------------------------------------------------------------
// Authenticated: Invitation management
// -------------------------------------------------------------------------

// ListInvitations handles GET /api/v1/invitations
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

// CreateBatchInvitations handles POST /api/v1/invitations
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

	// Require user:write or admin to invite.
	if !iamauth.HasAnyPermission(currentUser.Roles, iamauth.PermUserWrite, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "insufficient permissions to invite users")
		return
	}

	var req onboardingdto.BatchInviteRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	inviterName := currentUser.Email

	invitations, err := h.invitationSvc.CreateBatch(r.Context(), tenantID, userID, inviterName, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": invitations, "count": len(invitations)})
}

// CancelInvitation handles DELETE /api/v1/invitations/{id} and POST /api/v1/invitations/{id}/cancel
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

	invitationIDStr := chi.URLParam(r, "id")
	invitationID, err := uuid.Parse(invitationIDStr)
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

// ResendInvitation handles POST /api/v1/invitations/{id}/resend
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

	invitationIDStr := chi.URLParam(r, "id")
	invitationID, err := uuid.Parse(invitationIDStr)
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

// -------------------------------------------------------------------------
// Admin: Provisioning
// -------------------------------------------------------------------------

// AdminProvision handles POST /api/v1/admin/tenants/{id}/provision
func (h *Handler) AdminProvision(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid tenant ID")
		return
	}

	go func() {
		if provErr := h.provisioner.Provision(r.Context(), tenantID); provErr != nil {
			h.logger.Error().Err(provErr).Str("tenant_id", tenantID.String()).Msg("manual admin provision failed")
		}
	}()

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message":   "Provisioning started.",
		"tenant_id": tenantID.String(),
	})
}

// AdminGetProvisionStatus handles GET /api/v1/admin/tenants/{id}/provision-status
func (h *Handler) AdminGetProvisionStatus(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
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

// AdminDeprovision handles POST /api/v1/admin/tenants/{id}/deprovision
func (h *Handler) AdminDeprovision(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
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

// AdminReactivate handles POST /api/v1/admin/tenants/{id}/reactivate
func (h *Handler) AdminReactivate(w http.ResponseWriter, r *http.Request) {
	currentUser := iamauth.MustUserFromContext(r.Context())
	if !iamauth.HasPermission(currentUser.Roles, iamauth.PermAdminAll) {
		writeError(w, http.StatusForbidden, "super admin access required")
		return
	}

	tenantIDStr := chi.URLParam(r, "id")
	tenantID, err := uuid.Parse(tenantIDStr)
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

// -------------------------------------------------------------------------
// Helpers (mirrors the IAM handler helpers)
// -------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func parseBody(r *http.Request, dst any) error {
	if r.Body == nil {
		return fmt.Errorf("request body is required")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid request body: %w", err)
	}
	return nil
}

func getIPAddress(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, iammodel.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, iammodel.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, iammodel.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, iammodel.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, iammodel.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, iammodel.ErrAccountLocked):
		writeError(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, iammodel.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, onboardingmodel.ErrExpiredInvitation):
		writeError(w, http.StatusGone, err.Error())
	case errors.Is(err, onboardingmodel.ErrInvitationUsed):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}
