package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/iam/service"
)

const oauthAccessCookieName = "clario360_access"

type OAuthHandler struct {
	oauthSvc *service.OAuthService
	logger   zerolog.Logger
}

func NewOAuthHandler(oauthSvc *service.OAuthService, logger zerolog.Logger) *OAuthHandler {
	return &OAuthHandler{oauthSvc: oauthSvc, logger: logger}
}

func (h *OAuthHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/authorize", h.Authorize)
	r.Post("/token", h.Token)
	r.Get("/userinfo", h.UserInfo)
	// Social-login provider discovery — always returns the configured provider list.
	// Returns an empty array when no external social providers are configured.
	r.Get("/providers", h.ListSocialProviders)
	return r
}

// SocialAuthRoutes returns routes that require JWT authentication (connections, link).
// Mount these inside the protected route group in main.go.
func (h *OAuthHandler) SocialAuthRoutes() chi.Router {
	r := chi.NewRouter()
	r.Get("/connections", h.ListConnections)
	r.Delete("/link/{provider}", h.UnlinkProvider)
	return r
}

func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	result, err := h.oauthSvc.Authorize(r.Context(), service.OAuthAuthorizeRequest{
		ResponseType:        strings.TrimSpace(r.URL.Query().Get("response_type")),
		ClientID:            strings.TrimSpace(r.URL.Query().Get("client_id")),
		RedirectURI:         strings.TrimSpace(r.URL.Query().Get("redirect_uri")),
		Scope:               strings.TrimSpace(r.URL.Query().Get("scope")),
		State:               strings.TrimSpace(r.URL.Query().Get("state")),
		CodeChallenge:       strings.TrimSpace(r.URL.Query().Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(r.URL.Query().Get("code_challenge_method")),
	}, accessTokenFromRequest(r))
	if err != nil {
		writeOAuthError(w, err)
		return
	}

	redirectURL := result.RedirectURL
	if redirectURL == "" {
		redirectURL = result.LoginRedirectURL
	}
	if redirectURL == "" {
		writeOAuthError(w, &service.OAuthError{Status: http.StatusUnauthorized, Code: "UNAUTHENTICATED", Message: "user must authenticate before authorization"})
		return
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeOAuthError(w, &service.OAuthError{Status: http.StatusBadRequest, Code: "INVALID_REQUEST", Message: "form-encoded request body is required"})
		return
	}

	req := service.OAuthTokenRequest{
		GrantType:    strings.TrimSpace(r.Form.Get("grant_type")),
		Code:         strings.TrimSpace(r.Form.Get("code")),
		RedirectURI:  strings.TrimSpace(r.Form.Get("redirect_uri")),
		ClientID:     strings.TrimSpace(r.Form.Get("client_id")),
		ClientSecret: strings.TrimSpace(r.Form.Get("client_secret")),
		CodeVerifier: strings.TrimSpace(r.Form.Get("code_verifier")),
		RefreshToken: strings.TrimSpace(r.Form.Get("refresh_token")),
	}
	if clientID, clientSecret, ok := r.BasicAuth(); ok {
		if req.ClientID == "" {
			req.ClientID = strings.TrimSpace(clientID)
		}
		if req.ClientSecret == "" {
			req.ClientSecret = strings.TrimSpace(clientSecret)
		}
	}
	resp, err := h.oauthSvc.ExchangeToken(r.Context(), req, getIPAddress(r), r.UserAgent())
	if err != nil {
		writeOAuthError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *OAuthHandler) UserInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.oauthSvc.UserInfo(r.Context(), accessTokenFromRequest(r))
	if err != nil {
		writeOAuthError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (h *OAuthHandler) Discovery(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.oauthSvc.DiscoveryDocument())
}

func (h *OAuthHandler) JWKS(w http.ResponseWriter, r *http.Request) {
	jwks, err := h.oauthSvc.JWKS()
	if err != nil {
		writeOAuthError(w, &service.OAuthError{Status: http.StatusInternalServerError, Code: "INTERNAL_ERROR", Message: "failed to generate jwks"})
		return
	}
	writeJSON(w, http.StatusOK, jwks)
}

// ListSocialProviders handles GET /api/v1/auth/oauth/providers.
// Returns an empty list until social-login provider integration is implemented.
func (h *OAuthHandler) ListSocialProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

// ListConnections handles GET /api/v1/auth/oauth/connections (auth-required).
// Returns the calling user's linked social-login accounts.
// Returns an empty list until social-login provider integration is implemented.
func (h *OAuthHandler) ListConnections(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

// UnlinkProvider handles DELETE /api/v1/auth/oauth/link/{provider} (auth-required).
// Removes a linked social-login account. Returns 404 until social-login is implemented.
func (h *OAuthHandler) UnlinkProvider(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "social login provider linking is not yet configured")
}

func accessTokenFromRequest(r *http.Request) string {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && strings.EqualFold(parts[0], "bearer") {
			return strings.TrimSpace(parts[1])
		}
	}

	if cookie, err := r.Cookie(oauthAccessCookieName); err == nil {
		return strings.TrimSpace(cookie.Value)
	}
	return ""
}

func writeOAuthError(w http.ResponseWriter, err error) {
	if oauthErr, ok := err.(*service.OAuthError); ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(oauthErr.Status)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":   oauthErr.Code,
			"message": oauthErr.Message,
		})
		return
	}
	writeError(w, http.StatusInternalServerError, "internal server error")
}
