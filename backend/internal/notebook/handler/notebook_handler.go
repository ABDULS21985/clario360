package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	nbdto "github.com/clario360/platform/internal/notebook/dto"
	nbmodel "github.com/clario360/platform/internal/notebook/model"
	nbservice "github.com/clario360/platform/internal/notebook/service"
)

type NotebookHandler struct {
	service *nbservice.NotebookService
	logger  zerolog.Logger
}

func NewNotebookHandler(service *nbservice.NotebookService, logger zerolog.Logger) *NotebookHandler {
	return &NotebookHandler{service: service, logger: logger}
}

func (h *NotebookHandler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/profiles", h.ListProfiles)
	r.Get("/templates", h.ListTemplates)
	r.Get("/servers", h.ListServers)
	r.Post("/servers", h.StartServer)
	r.Delete("/servers/{id}", h.StopServer)
	r.Get("/servers/{id}/status", h.GetServerStatus)
	r.Post("/servers/{id}/copy-template", h.CopyTemplate)
	r.Post("/activity", h.RecordActivity)
	return r
}

func (h *NotebookHandler) ListProfiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.ListProfiles(actorFromContext(r)))
}

func (h *NotebookHandler) ListTemplates(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.ListTemplates())
}

func (h *NotebookHandler) ListServers(w http.ResponseWriter, r *http.Request) {
	servers, err := h.service.ListServers(r.Context(), actorFromContext(r))
	if err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, servers)
}

func (h *NotebookHandler) StartServer(w http.ResponseWriter, r *http.Request) {
	var req nbdto.StartServerRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	server, err := h.service.StartServer(r.Context(), actorFromContext(r), req.Profile)
	if err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, server)
}

func (h *NotebookHandler) StopServer(w http.ResponseWriter, r *http.Request) {
	if err := h.service.StopServer(r.Context(), actorFromContext(r), chi.URLParam(r, "id"), r.URL.Query().Get("reason")); err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, nbdto.MessageResponse{Message: "notebook server stopped"})
}

func (h *NotebookHandler) GetServerStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetServerStatus(r.Context(), actorFromContext(r), chi.URLParam(r, "id"))
	if err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *NotebookHandler) CopyTemplate(w http.ResponseWriter, r *http.Request) {
	var req nbdto.CopyTemplateRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.CopyTemplate(r.Context(), actorFromContext(r), chi.URLParam(r, "id"), req.TemplateID)
	if err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *NotebookHandler) RecordActivity(w http.ResponseWriter, r *http.Request) {
	var req nbdto.ActivityRequest
	if err := parseBody(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.service.RecordActivity(r.Context(), actorFromContext(r), req); err != nil {
		handleNotebookError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, nbdto.MessageResponse{Message: "notebook activity recorded"})
}

func actorFromContext(r *http.Request) nbmodel.Actor {
	user := auth.MustUserFromContext(r.Context())
	return nbmodel.Actor{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Roles:    user.Roles,
	}
}

func handleNotebookError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, nbmodel.ErrInvalidProfile), errors.Is(err, nbmodel.ErrTemplateNotFound), errors.Is(err, nbmodel.ErrActivityInvalid):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, nbmodel.ErrProfileForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, nbmodel.ErrServerRunning):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, nbmodel.ErrServerNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusBadGateway, err.Error())
	}
}
