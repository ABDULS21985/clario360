package handler

import (
	"net/http"
	"strings"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/acta/dto"
	"github.com/clario360/platform/internal/acta/model"
	"github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/suiteapi"
)

type ActionItemHandler struct {
	baseHandler
	service *service.ActionItemService
}

func NewActionItemHandler(service *service.ActionItemService, logger zerolog.Logger) *ActionItemHandler {
	return &ActionItemHandler{
		baseHandler: baseHandler{logger: logger},
		service:     service,
	}
}

func (h *ActionItemHandler) Create(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateActionItemRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.CreateActionItem(r.Context(), tenantID, userID, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusCreated, item)
}

func (h *ActionItemHandler) List(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	filters := model.ActionItemFilters{
		Page:    page,
		PerPage: perPage,
	}
	committeeID, err := parseOptionalUUID(r.URL.Query().Get("committee_id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid committee_id", nil)
		return
	}
	meetingID, err := parseOptionalUUID(r.URL.Query().Get("meeting_id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid meeting_id", nil)
		return
	}
	assigneeID, err := parseOptionalUUID(r.URL.Query().Get("assignee_id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid assignee_id", nil)
		return
	}
	overdueOnly, err := parseOptionalBool(r.URL.Query().Get("overdue"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid overdue flag", nil)
		return
	}
	filters.CommitteeID = committeeID
	filters.MeetingID = meetingID
	filters.AssigneeID = assigneeID
	if overdueOnly != nil {
		filters.OverdueOnly = *overdueOnly
	}
	filters.Search = strings.TrimSpace(r.URL.Query().Get("search"))
	filters.Sort = strings.TrimSpace(r.URL.Query().Get("sort"))
	filters.Order = strings.TrimSpace(r.URL.Query().Get("order"))
	rawStatuses := suiteapi.ParseCSVParam(r, "status")
	if len(rawStatuses) > 0 {
		filters.Statuses = make([]model.ActionItemStatus, 0, len(rawStatuses))
		for _, status := range rawStatuses {
			filters.Statuses = append(filters.Statuses, model.ActionItemStatus(status))
		}
	}
	items, total, err := h.service.ListActionItems(r.Context(), tenantID, filters)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *ActionItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	item, err := h.service.GetActionItem(r.Context(), tenantID, id)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ActionItemHandler) Update(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateActionItemRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.UpdateActionItem(r.Context(), tenantID, userID, id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ActionItemHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.UpdateActionItemStatusRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.UpdateStatus(r.Context(), tenantID, userID, id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ActionItemHandler) Extend(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	id, err := suiteapi.UUIDParam(r, "id")
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
		return
	}
	var req dto.ExtendActionItemRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	item, err := h.service.ExtendDueDate(r.Context(), tenantID, userID, id, req)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, item)
}

func (h *ActionItemHandler) Overdue(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	limit := 100
	if parsed, err := parseOptionalInt(r.URL.Query().Get("limit")); err == nil && parsed != nil && *parsed > 0 {
		limit = *parsed
	} else if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid limit", nil)
		return
	}
	items, err := h.service.ListOverdue(r.Context(), tenantID, limit)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ActionItemHandler) My(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := h.tenantAndUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListMyActionItems(r.Context(), tenantID, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, items)
}

func (h *ActionItemHandler) Stats(w http.ResponseWriter, r *http.Request) {
	tenantID, ok := h.tenantID(w, r)
	if !ok {
		return
	}
	stats, err := h.service.Stats(r.Context(), tenantID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteData(w, http.StatusOK, stats)
}
