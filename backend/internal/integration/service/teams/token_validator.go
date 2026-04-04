package teams

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	ServiceURL string `json:"serviceurl"`
	jwt.RegisteredClaims
}

var (
	jwksMu       sync.Mutex
	jwksCache    map[string]*rsa.PublicKey
	jwksCachedAt time.Time
)

func ValidateTeamsToken(r *http.Request, botAppID string) (*Claims, error) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(authHeader), "bearer ") {
		return nil, fmt.Errorf("missing teams bearer token")
	}
	tokenStr := strings.TrimSpace(authHeader[len("Bearer "):])

	parser := jwt.NewParser()
	unverifiedClaims := &Claims{}
	token, _, err := parser.ParseUnverified(tokenStr, unverifiedClaims)
	if err != nil {
		return nil, fmt.Errorf("parse teams token header: %w", err)
	}
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, fmt.Errorf("teams token missing kid")
	}

	keys, err := fetchTeamsJWKS(r.Context())
	if err != nil {
		return nil, err
	}
	publicKey, ok := keys[kid]
	if !ok {
		return nil, fmt.Errorf("teams jwks key not found")
	}

	parsedToken, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		return publicKey, nil
	}, jwt.WithAudience(botAppID), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil {
		return nil, fmt.Errorf("validate teams token: %w", err)
	}
	claims, ok := parsedToken.Claims.(*Claims)
	if !ok || !parsedToken.Valid {
		return nil, fmt.Errorf("invalid teams token claims")
	}

	validIssuer := claims.Issuer == "https://api.botframework.com" || strings.HasPrefix(claims.Issuer, "https://sts.windows.net/")
	if !validIssuer {
		return nil, fmt.Errorf("invalid teams token issuer")
	}
	if claims.ServiceURL == "" {
		return nil, fmt.Errorf("teams token missing serviceurl")
	}
	return claims, nil
}

func fetchTeamsJWKS(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	jwksMu.Lock()
	if len(jwksCache) > 0 && time.Since(jwksCachedAt) < 24*time.Hour {
		cached := jwksCache
		jwksMu.Unlock()
		return cached, nil
	}
	jwksMu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://login.botframework.com/v1/.well-known/openidconfiguration", nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("botframework openid config returned %d", resp.StatusCode)
	}

	var openID struct {
		JWKSURI string `json:"jwks_uri"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&openID); err != nil {
		return nil, err
	}
	if openID.JWKSURI == "" {
		return nil, fmt.Errorf("botframework jwks_uri missing")
	}

	jwksReq, err := http.NewRequestWithContext(ctx, http.MethodGet, openID.JWKSURI, nil)
	if err != nil {
		return nil, err
	}
	jwksResp, err := http.DefaultClient.Do(jwksReq)
	if err != nil {
		return nil, err
	}
	defer jwksResp.Body.Close()
	if jwksResp.StatusCode >= 400 {
		return nil, fmt.Errorf("teams jwks returned %d", jwksResp.StatusCode)
	}

	var payload struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(jwksResp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(payload.Keys))
	for _, item := range payload.Keys {
		publicKey, err := jwkToPublicKey(item.N, item.E)
		if err != nil {
			continue
		}
		keys[item.Kid] = publicKey
	}

	jwksMu.Lock()
	jwksCache = keys
	jwksCachedAt = time.Now().UTC()
	jwksMu.Unlock()
	return keys, nil
}

func jwkToPublicKey(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}

	e := 0
	for _, b := range eBytes {
		e = e<<8 + int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("invalid exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nBytes),
		E: e,
	}, nil
}
