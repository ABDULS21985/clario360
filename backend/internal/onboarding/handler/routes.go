package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	onboardingrepo "github.com/clario360/platform/internal/onboarding/repository"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
)

type Handler struct {
	registrationSvc  *onboardingsvc.RegistrationService
	wizardSvc        *onboardingsvc.WizardService
	invitationSvc    *onboardingsvc.InvitationService
	provisioner      *onboardingsvc.TenantProvisioner
	deprovisioner    *onboardingsvc.TenantDeprovisioner
	brandingUploader onboardingsvc.BrandingAssetUploader
	provisioningRepo *onboardingrepo.ProvisioningRepository
	logger           zerolog.Logger
}

func New(
	registrationSvc *onboardingsvc.RegistrationService,
	wizardSvc *onboardingsvc.WizardService,
	invitationSvc *onboardingsvc.InvitationService,
	provisioner *onboardingsvc.TenantProvisioner,
	deprovisioner *onboardingsvc.TenantDeprovisioner,
	brandingUploader onboardingsvc.BrandingAssetUploader,
	provisioningRepo *onboardingrepo.ProvisioningRepository,
	logger zerolog.Logger,
) *Handler {
	return &Handler{
		registrationSvc:  registrationSvc,
		wizardSvc:        wizardSvc,
		invitationSvc:    invitationSvc,
		provisioner:      provisioner,
		deprovisioner:    deprovisioner,
		brandingUploader: brandingUploader,
		provisioningRepo: provisioningRepo,
		logger:           logger.With().Str("handler", "onboarding").Logger(),
	}
}

func (h *Handler) PublicOnboardingRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/verify-email", h.VerifyEmail)
	r.Post("/resend-otp", h.ResendOTP)
	r.Get("/status/{tenantId}", h.GetOnboardingStatus)
	return r
}

func (h *Handler) WizardRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.GetWizardProgress)
	r.Post("/organization", h.SaveOrganization)
	r.Post("/branding", h.SaveBranding)
	r.Post("/team", h.SaveTeam)
	r.Post("/suites", h.SaveSuites)
	r.Post("/complete", h.CompleteWizard)
	return r
}

func (h *Handler) PublicInvitationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/validate", h.ValidateInviteToken)
	r.Post("/accept", h.AcceptInvitation)
	return r
}

func (h *Handler) InvitationRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.ListInvitations)
	r.Get("/stats", h.GetStats)
	r.Post("/", h.CreateBatchInvitations)
	r.Post("/resend/{id}", h.ResendInvitation)
	r.Delete("/{id}", h.CancelInvitation)
	return r
}

func (h *Handler) AdminRoutes() chi.Router {
	r := chi.NewRouter()
	r.Post("/provision", h.AdminProvision)
	r.Get("/{id}/provision-status", h.AdminGetProvisionStatus)
	r.Post("/{id}/deprovision", h.AdminDeprovision)
	r.Post("/{id}/reprovision", h.AdminReprovision)
	r.Post("/{id}/reactivate", h.AdminReactivate)
	return r
}

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

func (h *Handler) startProvisioningAsync(tenantID uuid.UUID, reason string) {
	if h.provisioner == nil {
		h.logger.Error().
			Str("tenant_id", tenantID.String()).
			Str("reason", reason).
			Msg("provisioner not configured")
		return
	}

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := h.provisioner.Provision(ctx, tenantID); err != nil {
			h.logger.Error().
				Err(err).
				Str("tenant_id", tenantID.String()).
				Str("reason", reason).
				Msg("async provisioning failed")
		}
	}()
}
