package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	chatengine "github.com/clario360/platform/internal/cyber/vciso/chat/engine"
	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	"github.com/clario360/platform/internal/suiteapi"
)

type ChatHandler struct {
	engine           *chatengine.Engine
	conversationRepo *chatrepo.ConversationRepository
	logger           zerolog.Logger
}

func NewChatHandler(engine *chatengine.Engine, conversationRepo *chatrepo.ConversationRepository, logger zerolog.Logger) *ChatHandler {
	return &ChatHandler{
		engine:           engine,
		conversationRepo: conversationRepo,
		logger:           logger.With().Str("component", "vciso_chat_handler").Logger(),
	}
}

func (h *ChatHandler) Chat(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var req chatdto.ChatRequest
	if err := suiteapi.DecodeJSON(r, &req); err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body", nil)
		return
	}
	req.Message = strings.TrimSpace(req.Message)
	switch {
	case req.Message == "":
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "message is required", nil)
		return
	case len(req.Message) > 2000:
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "message exceeds 2000 characters", nil)
		return
	}
	resp, err := h.engine.ProcessMessage(r.Context(), req.ConversationID, tenantID, userID, req.Message)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, resp)
}

func (h *ChatHandler) ListConversations(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	page, perPage := suiteapi.ParsePagination(r)
	items, total, err := h.conversationRepo.ListConversations(r.Context(), tenantID, userID, page, perPage)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WritePaginated(w, http.StatusOK, items, page, perPage, total)
}

func (h *ChatHandler) GetConversation(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	conversationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid conversation id", nil)
		return
	}
	conversation, err := h.conversationRepo.GetConversation(r.Context(), tenantID, userID, conversationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	messages, err := h.conversationRepo.ListMessages(r.Context(), tenantID, conversationID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	out := chatdto.ConversationDetail{
		ID:            conversation.ID,
		Title:         conversation.Title,
		Status:        string(conversation.Status),
		MessageCount:  conversation.MessageCount,
		LastMessageAt: conversation.LastMessageAt,
		CreatedAt:     conversation.CreatedAt,
		Messages:      make([]chatdto.ConversationMessage, 0, len(messages)),
	}
	for _, item := range messages {
		out.Messages = append(out.Messages, mapConversationMessage(item))
	}
	suiteapi.WriteJSON(w, http.StatusOK, out)
}

func (h *ChatHandler) DeleteConversation(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	conversationID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid conversation id", nil)
		return
	}
	if err := h.conversationRepo.SoftDeleteConversation(r.Context(), tenantID, userID, conversationID); err != nil {
		h.writeError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ChatHandler) Suggestions(w http.ResponseWriter, r *http.Request) {
	tenantID, userID, ok := requireTenantAndUser(w, r)
	if !ok {
		return
	}
	var conversationID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("conversation_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "invalid conversation_id", nil)
			return
		}
		conversationID = &parsed
	}
	items, err := h.engine.GetSuggestions(r.Context(), conversationID, tenantID, userID)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	suiteapi.WriteJSON(w, http.StatusOK, chatdto.SuggestionResponse{Suggestions: items})
}

func (h *ChatHandler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, chatrepo.ErrNotFound):
		suiteapi.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "conversation not found", nil)
	case strings.Contains(strings.ToLower(err.Error()), "message is required"),
		strings.Contains(strings.ToLower(err.Error()), "message exceeds"):
		suiteapi.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), nil)
	default:
		h.logger.Error().Err(err).Msg("vciso chat request failed")
		suiteapi.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", sanitizeError(err.Error()), nil)
	}
}

func requireTenantAndUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	tenantID, err := suiteapi.TenantID(r)
	if err != nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "missing tenant context", nil)
		return uuid.Nil, uuid.Nil, false
	}
	userID, err := suiteapi.UserID(r)
	if err != nil || userID == nil {
		suiteapi.WriteError(w, r, http.StatusUnauthorized, "UNAUTHORIZED", "missing user context", nil)
		return uuid.Nil, uuid.Nil, false
	}
	return tenantID, *userID, true
}

func mapConversationMessage(item chatmodel.Message) chatdto.ConversationMessage {
	var toolResult any
	if len(item.ToolResult) > 0 && string(item.ToolResult) != "null" {
		_ = json.Unmarshal(item.ToolResult, &toolResult)
	}
	return chatdto.ConversationMessage{
		ID:           item.ID,
		Role:         string(item.Role),
		Content:      item.Content,
		Intent:       item.Intent,
		ResponseType: item.ResponseType,
		Actions:      item.SuggestedActions,
		ToolResult:   toolResult,
		CreatedAt:    item.CreatedAt,
	}
}

func sanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "request failed"
	}
	if len(value) > 240 {
		return fmt.Sprintf("%s...", value[:240])
	}
	return value
}
