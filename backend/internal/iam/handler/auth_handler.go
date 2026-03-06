package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/service"
)

type AuthHandler struct {
	authSvc *service.AuthService
	logger  zerolog.Logger
}

func NewAuthHandler(authSvc *service.AuthService, logger zerolog.Logger) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, logger: logger}
}

func (h *AuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Post("/refresh", h.Refresh)
	r.Post("/logout", h.Logout)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	r.Post("/verify-mfa", h.VerifyMFA)
	return r
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authSvc.Register(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.authSvc.Login(r.Context(), &req, ip, userAgent)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req dto.RefreshRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ip := getIPAddress(r)
	userAgent := r.UserAgent()

	resp, err := h.authSvc.RefreshToken(r.Context(), &req, ip, userAgent)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var req dto.LogoutRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authSvc.Logout(r.Context(), &req); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "logged out successfully"})
}

func (h *AuthHandler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ForgotPasswordRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Always return success to prevent email enumeration
	_ = h.authSvc.ForgotPassword(r.Context(), &req)

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "if an account exists with that email, a reset link has been sent"})
}

func (h *AuthHandler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req dto.ResetPasswordRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.authSvc.ResetPassword(r.Context(), &req); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "password reset successfully"})
}

func (h *AuthHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyMFARequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.authSvc.VerifyMFA(r.Context(), &req)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
