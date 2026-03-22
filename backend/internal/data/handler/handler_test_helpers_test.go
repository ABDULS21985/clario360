package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
)

var (
	testTenantID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	testUserID   = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	testLogger   = zerolog.Nop()
)

// authRequest creates an *http.Request with tenantID + userID injected into the context.
func authRequest(method, path string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewBuffer(body))
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	ctx := r.Context()
	ctx = auth.WithTenantID(ctx, testTenantID.String())
	ctx = auth.WithUser(ctx, &auth.ContextUser{
		ID:       testUserID.String(),
		TenantID: testTenantID.String(),
		Email:    "test@example.com",
		Roles:    []string{"admin"},
	})
	return r.WithContext(ctx)
}

// authRequestWithID creates an *http.Request with auth context AND chi URL param "id".
func authRequestWithID(method, path string, id uuid.UUID, body []byte) *http.Request {
	r := authRequest(method, path, body)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id.String())
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// authRequestWithParams creates an *http.Request with auth context and arbitrary chi URL params.
func authRequestWithParams(method, path string, params map[string]string, body []byte) *http.Request {
	r := authRequest(method, path, body)
	rctx := chi.NewRouteContext()
	for k, v := range params {
		rctx.URLParams.Add(k, v)
	}
	ctx := context.WithValue(r.Context(), chi.RouteCtxKey, rctx)
	return r.WithContext(ctx)
}

// unauthRequest creates an *http.Request WITHOUT any auth context.
func unauthRequest(method, path string, body []byte) *http.Request {
	if body != nil {
		return httptest.NewRequest(method, path, bytes.NewBuffer(body))
	}
	return httptest.NewRequest(method, path, nil)
}
