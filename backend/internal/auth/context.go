package auth

import "context"

type contextKey string

const (
	userContextKey   contextKey = "auth_user"
	tenantContextKey contextKey = "auth_tenant"
	claimsContextKey contextKey = "auth_claims"
)

// ContextUser holds authenticated user information extracted from the JWT.
type ContextUser struct {
	ID       string
	TenantID string
	Email    string
	Roles    []string
}

// WithUser stores the authenticated user in the context.
func WithUser(ctx context.Context, user *ContextUser) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// UserFromContext retrieves the authenticated user from the context.
// Returns nil if no user is set (unauthenticated request).
func UserFromContext(ctx context.Context) *ContextUser {
	user, _ := ctx.Value(userContextKey).(*ContextUser)
	return user
}

// WithTenantID stores the tenant ID in the context.
func WithTenantID(ctx context.Context, tenantID string) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenantID)
}

// TenantFromContext retrieves the tenant ID from the context.
func TenantFromContext(ctx context.Context) string {
	tenantID, _ := ctx.Value(tenantContextKey).(string)
	return tenantID
}

// WithClaims stores the full JWT claims in the context.
func WithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext retrieves the full JWT claims from the context.
func ClaimsFromContext(ctx context.Context) *Claims {
	claims, _ := ctx.Value(claimsContextKey).(*Claims)
	return claims
}

// MustUserFromContext retrieves the authenticated user or panics.
// Use only in handlers where auth middleware is guaranteed to have run.
func MustUserFromContext(ctx context.Context) *ContextUser {
	user := UserFromContext(ctx)
	if user == nil {
		panic("auth: user not found in context — auth middleware not applied")
	}
	return user
}
