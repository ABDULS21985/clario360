package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
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

// JWTManager handles JWT creation and validation using RS256.
type JWTManager struct {
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	issuer         string
	accessTokenTTL time.Duration
	refreshTTL     time.Duration
}

// NewJWTManager creates a new JWT manager from configuration.
// If RSA PEM keys are provided, they are used. Otherwise, an ephemeral
// 2048-bit RSA key pair is generated (suitable for development only).
func NewJWTManager(cfg config.AuthConfig) (*JWTManager, error) {
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey

	if cfg.RSAPrivateKeyPEM != "" && cfg.RSAPublicKeyPEM != "" {
		privKey, err := parseRSAPrivateKey([]byte(cfg.RSAPrivateKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("parsing RSA private key: %w", err)
		}
		privateKey = privKey

		pubKey, err := parseRSAPublicKey([]byte(cfg.RSAPublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("parsing RSA public key: %w", err)
		}
		publicKey = pubKey
	} else if cfg.RSAPublicKeyPEM != "" {
		pubKey, err := parseRSAPublicKey([]byte(cfg.RSAPublicKeyPEM))
		if err != nil {
			return nil, fmt.Errorf("parsing RSA public key: %w", err)
		}
		publicKey = pubKey
	} else {
		// Generate ephemeral key pair for development
		key, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("generating dev RSA key pair: %w", err)
		}
		privateKey = key
		publicKey = &key.PublicKey
	}

	return &JWTManager{
		privateKey:     privateKey,
		publicKey:      publicKey,
		issuer:         cfg.JWTIssuer,
		accessTokenTTL: cfg.AccessTokenTTL,
		refreshTTL:     cfg.RefreshTokenTTL,
	}, nil
}

// GenerateTokenPair creates a new access/refresh token pair signed with RS256.
func (m *JWTManager) GenerateTokenPair(userID, tenantID, email string, roles []string) (*TokenPair, error) {
	if m.privateKey == nil {
		return nil, fmt.Errorf("JWT manager is configured for validation only")
	}
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

	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, accessClaims)
	accessStr, err := accessToken.SignedString(m.privateKey)
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	refreshClaims := jwt.RegisteredClaims{
		Issuer:    m.issuer,
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
	}

	refreshToken := jwt.NewWithClaims(jwt.SigningMethodRS256, refreshClaims)
	refreshStr, err := refreshToken.SignedString(m.privateKey)
	if err != nil {
		return nil, fmt.Errorf("signing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessStr,
		RefreshToken: refreshStr,
		ExpiresAt:    accessExp,
	}, nil
}

// SignClaims signs arbitrary JWT claims with the manager's private key.
// This is used for standards-based tokens such as OIDC ID tokens while
// keeping all platform signing centralized in one place.
func (m *JWTManager) SignClaims(claims jwt.Claims) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("JWT manager is configured for validation only")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	signed, err := token.SignedString(m.privateKey)
	if err != nil {
		return "", fmt.Errorf("signing token: %w", err)
	}
	return signed, nil
}

// ValidateAccessToken parses and validates an access token, returning the claims.
func (m *JWTManager) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
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
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.publicKey, nil
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

// PublicKey returns the public verification key.
func (m *JWTManager) PublicKey() *rsa.PublicKey {
	return m.publicKey
}

// Issuer returns the configured JWT issuer.
func (m *JWTManager) Issuer() string {
	return m.issuer
}

// AccessTokenTTL returns the configured access-token TTL.
func (m *JWTManager) AccessTokenTTL() time.Duration {
	return m.accessTokenTTL
}

// RefreshTokenTTL returns the configured refresh-token TTL.
func (m *JWTManager) RefreshTokenTTL() time.Duration {
	return m.refreshTTL
}

func parseRSAPrivateKey(pemData []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Try PKCS1 first, then PKCS8
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}

	parsed, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err2 != nil {
		return nil, fmt.Errorf("PKCS1: %v, PKCS8: %v", err, err2)
	}

	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("parsed key is not RSA")
	}
	return rsaKey, nil
}

func parseRSAPublicKey(pemData []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parsing public key: %w", err)
	}

	rsaKey, ok := parsed.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("parsed key is not RSA")
	}
	return rsaKey, nil
}
