package security

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
)

// GenerateCSRFToken generates a cryptographically random 32-byte token.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// SetCSRFCookie generates a new CSRF token and sets it as a cookie.
func SetCSRFCookie(w http.ResponseWriter, cfg *CSRFConfig) (string, error) {
	token, err := GenerateCSRFToken()
	if err != nil {
		return "", err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     cfg.CookieName,
		Value:    token,
		Path:     "/",
		Domain:   cfg.CookieDomain,
		Secure:   cfg.CookieSecure,
		HttpOnly: false, // Must be readable by JavaScript
		SameSite: cfg.CookieSameSite,
		MaxAge:   cfg.MaxAge,
	})

	return token, nil
}

// RotateCSRFToken generates a new token and replaces the existing cookie.
// Called after: successful login, token refresh, password change.
func RotateCSRFToken(w http.ResponseWriter, cfg *CSRFConfig) error {
	_, err := SetCSRFCookie(w, cfg)
	return err
}
