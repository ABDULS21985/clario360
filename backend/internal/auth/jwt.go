package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/clario360/platform/internal/config"
)

// Claims are the JWT claims used by Clario 360.
type Claims struct {
	jwt.RegisteredClaims
	UserID      string   `json:"uid"`
	TenantID    string   `json:"tid"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"perms,omitempty"`
}

// TokenPair holds an access token and a refresh token.
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// JWTManager handles JWT creation and validation.
type JWTManager struct {
	secret         []byte
	issuer         string
	accessTokenTTL time.Duration
	refreshTTL     time.Duration
}

// NewJWTManager creates a new JWT manager from configuration.
func NewJWTManager(cfg config.AuthConfig) *JWTManager {
	return &JWTManager{
		secret:         []byte(cfg.JWTSecret),
		issuer:         cfg.JWTIssuer,
		accessTokenTTL: cfg.AccessTokenTTL,
		refreshTTL:     cfg.RefreshTokenTTL,
	}
}

// GenerateTokenPair creates a new access/refresh token pair.
func (m *JWTManager) GenerateTokenPair(userID, tenantID, email string, roles []string) (*TokenPair, error) {
	now := time.Now()
	accessExp := now.Add(m.accessTokenTTL)

	accessClaims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(accessExp),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:   userID,
		TenantID: tenantID,
		Email:    email,
		Roles:    roles,
	}

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshClaims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(m.secret)
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp,
	}, nil
}

// ValidateAccessToken parses and validates an access token, returning the claims.
func (m *JWTManager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// ValidateRefreshToken parses and validates a refresh token.
func (m *JWTManager) ValidateRefreshToken(tokenStr string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwt.RegisteredClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return "", fmt.Errorf("parsing refresh token: %w", err)
	}

	claims, ok := token.Claims.(*jwt.RegisteredClaims)
	if !ok || !token.Valid {
		return "", fmt.Errorf("invalid refresh token")
	}

	return claims.Subject, nil
}
