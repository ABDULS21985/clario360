package handler

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/google/uuid"

	iamauth "github.com/clario360/platform/internal/auth"
	iammodel "github.com/clario360/platform/internal/iam/model"
	onboardingdto "github.com/clario360/platform/internal/onboarding/dto"
	onboardingsvc "github.com/clario360/platform/internal/onboarding/service"
)

const brandingMultipartMaxBytes = 3 << 20

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

	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Type"))), "multipart/form-data") {
		resp, saveErr := h.saveBrandingMultipart(w, r, tenantID, currentUser)
		if saveErr != nil {
			handleServiceError(w, saveErr)
			return
		}
		writeJSON(w, http.StatusOK, resp)
		return
	}

	var req onboardingdto.BrandingRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	var logoFileID *uuid.UUID
	if strings.TrimSpace(req.LogoFileID) != "" {
		parsedLogoID, parseErr := uuid.Parse(strings.TrimSpace(req.LogoFileID))
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "invalid logo_file_id")
			return
		}
		logoFileID = &parsedLogoID
	}

	var primaryColor, accentColor *string
	if req.PrimaryColor != "" {
		primaryColor = &req.PrimaryColor
	}
	if req.AccentColor != "" {
		accentColor = &req.AccentColor
	}

	resp, err := h.wizardSvc.SaveBranding(r.Context(), tenantID, logoFileID, primaryColor, accentColor)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) saveBrandingMultipart(w http.ResponseWriter, r *http.Request, tenantID uuid.UUID, currentUser *iamauth.ContextUser) (*onboardingdto.WizardStepResponse, error) {
	userID, err := uuid.Parse(currentUser.ID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", iammodel.ErrValidation)
	}

	r.Body = http.MaxBytesReader(w, r.Body, brandingMultipartMaxBytes)
	if err := r.ParseMultipartForm(brandingMultipartMaxBytes); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "request body too large") {
			return nil, fmt.Errorf("logo file exceeds the 2MB limit: %w", iammodel.ErrValidation)
		}
		return nil, fmt.Errorf("invalid multipart branding payload: %w", iammodel.ErrValidation)
	}

	var logoFileID *uuid.UUID
	if existingLogoID := strings.TrimSpace(r.FormValue("logo_file_id")); existingLogoID != "" {
		parsedLogoID, parseErr := uuid.Parse(existingLogoID)
		if parseErr != nil {
			return nil, fmt.Errorf("invalid logo_file_id: %w", iammodel.ErrValidation)
		}
		logoFileID = &parsedLogoID
	}

	file, header, err := findMultipartFile(r, "logo", "file")
	if err != nil {
		return nil, err
	}
	if file != nil {
		defer file.Close()
		if h.brandingUploader == nil {
			return nil, fmt.Errorf("logo uploads are not configured: %w", iammodel.ErrValidation)
		}
		uploadedLogoID, uploadErr := h.brandingUploader.UploadLogo(r.Context(), onboardingsvc.BrandingLogoUploadRequest{
			TenantID:    tenantID,
			UserID:      userID,
			File:        file,
			Filename:    header.Filename,
			ContentType: header.Header.Get("Content-Type"),
			IPAddress:   getIPAddress(r),
			UserAgent:   r.UserAgent(),
			Size:        header.Size,
		})
		if uploadErr != nil {
			return nil, uploadErr
		}
		logoFileID = &uploadedLogoID
	}

	var primaryColor, accentColor *string
	if value := strings.TrimSpace(r.FormValue("primary_color")); value != "" {
		primaryColor = &value
	}
	if value := strings.TrimSpace(r.FormValue("accent_color")); value != "" {
		accentColor = &value
	}

	return h.wizardSvc.SaveBranding(r.Context(), tenantID, logoFileID, primaryColor, accentColor)
}

func findMultipartFile(r *http.Request, fieldNames ...string) (multipart.File, *multipart.FileHeader, error) {
	for _, fieldName := range fieldNames {
		file, header, err := r.FormFile(fieldName)
		if err == nil {
			return file, header, nil
		}
		if errors.Is(err, http.ErrMissingFile) || err == io.EOF || strings.Contains(strings.ToLower(err.Error()), "no such file") {
			continue
		}
		return nil, nil, fmt.Errorf("read %s upload: %w", fieldName, iammodel.ErrValidation)
	}
	return nil, nil, nil
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
