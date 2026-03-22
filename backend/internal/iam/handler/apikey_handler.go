package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	iamauth "github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/iam/dto"
	"github.com/clario360/platform/internal/iam/service"
)

type APIKeyHandler struct {
	keySvc *service.APIKeyService
	logger zerolog.Logger
}

func NewAPIKeyHandler(keySvc *service.APIKeyService, logger zerolog.Logger) *APIKeyHandler {
	return &APIKeyHandler{keySvc: keySvc, logger: logger}
}

func (h *APIKeyHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Post("/{id}/rotate", h.Rotate)
	r.Delete("/{id}", h.Revoke)
	return r
}

func (h *APIKeyHandler) List(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	page, perPage := parsePagination(r)
	search := r.URL.Query().Get("search")
	// Support multi-value status: ?status=active&status=revoked
	statuses := r.URL.Query()["status"]
	status := strings.Join(statuses, ",")
	sort := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

	items, total, err := h.keySvc.ListPaginated(r.Context(), user.TenantID, page, perPage, search, status, sort, order)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, paginatedResponse(items, total, page, perPage))
}

func (h *APIKeyHandler) Create(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req service.CreateAPIKeyRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := h.keySvc.Create(r.Context(), user.TenantID, &req, user.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, resp)
}

func (h *APIKeyHandler) Rotate(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID := urlParam(r, "id")

	resp, err := h.keySvc.Rotate(r.Context(), keyID, user.TenantID, user.ID)
	if err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *APIKeyHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	user := iamauth.UserFromContext(r.Context())
	if user == nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	keyID := urlParam(r, "id")

	if err := h.keySvc.Revoke(r.Context(), keyID, user.TenantID, user.ID); err != nil {
		handleServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.MessageResponse{Message: "api key revoked"})
}
