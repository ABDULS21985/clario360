package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"

	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/model"
)

var validate = validator.New(validator.WithRequiredStructEnabled())

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"code":    inferErrorCode(status, message),
		"message": message,
	})
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

	if err := validate.Struct(dst); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			msgs := make([]string, len(ve))
			for i, fe := range ve {
				msgs[i] = fmt.Sprintf("field '%s' %s", fe.Field(), fe.Tag())
			}
			return fmt.Errorf("validation: %s", strings.Join(msgs, "; "))
		}
		return fmt.Errorf("validation: %w", err)
	}
	return nil
}

func urlParam(r *http.Request, key string) string {
	return chi.URLParam(r, key)
}

func parsePagination(r *http.Request) (page, perPage int) {
	page = 1
	perPage = 20

	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if pp := r.URL.Query().Get("per_page"); pp != "" {
		if v, err := strconv.Atoi(pp); err == nil && v > 0 && v <= 100 {
			perPage = v
		}
	}
	return
}

func paginatedResponse(data any, total, page, perPage int) dto.PaginatedResponse {
	lastPage := total / perPage
	if total%perPage != 0 {
		lastPage++
	}
	if lastPage < 1 {
		lastPage = 1
	}
	return dto.PaginatedResponse{
		Data: data,
		Meta: dto.PaginationMeta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: lastPage,
		},
	}
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
	case errors.Is(err, model.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, model.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, model.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, model.ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, model.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, model.ErrAccountLocked):
		writeError(w, http.StatusTooManyRequests, err.Error())
	case errors.Is(err, model.ErrMFARequired):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, model.ErrInvalidMFA):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, model.ErrInvalidToken):
		writeError(w, http.StatusUnauthorized, err.Error())
	case errors.Is(err, model.ErrSystemRole):
		writeError(w, http.StatusForbidden, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func inferErrorCode(status int, message string) string {
	lower := strings.ToLower(message)
	switch {
	case status == http.StatusBadRequest:
		return "VALIDATION_ERROR"
	case status == http.StatusUnauthorized && strings.Contains(lower, "invalid credentials"):
		return "INVALID_CREDENTIALS"
	case status == http.StatusUnauthorized && strings.Contains(lower, "invalid mfa"):
		return "MFA_INVALID"
	case status == http.StatusUnauthorized && strings.Contains(lower, "token"):
		return "TOKEN_EXPIRED"
	case status == http.StatusForbidden && strings.Contains(lower, "mfa"):
		return "MFA_REQUIRED"
	case status == http.StatusForbidden && strings.Contains(lower, "suspend"):
		return "ACCOUNT_SUSPENDED"
	case status == http.StatusTooManyRequests && strings.Contains(lower, "locked"):
		return "ACCOUNT_LOCKED"
	case status == http.StatusConflict && strings.Contains(lower, "email"):
		return "EMAIL_TAKEN"
	case status == http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case status == http.StatusForbidden:
		return "FORBIDDEN"
	case status == http.StatusNotFound:
		return "NOT_FOUND"
	case status == http.StatusConflict:
		return "CONFLICT"
	default:
		return "INTERNAL_ERROR"
	}
}
