package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	"github.com/clario360/platform/internal/auth"
	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	"github.com/clario360/platform/internal/cyber/vciso/chat/tools"
	"github.com/clario360/platform/internal/events"
)

type Engine struct {
	classifier       *IntentClassifier
	extractor        *EntityExtractor
	contextManager   *ContextManager
	toolRegistry     *tools.ToolRegistry
	toolRouter       *ToolRouter
	formatter        *ResponseFormatter
	suggestionEngine *SuggestionEngine
	conversationRepo *chatrepo.ConversationRepository
	predLogger       *aigovmiddleware.PredictionLogger
	producer         *events.Producer
	logger           zerolog.Logger
	metrics          *VCISOMetrics
	now              func() time.Time
}

func NewEngine(
	classifier *IntentClassifier,
	extractor *EntityExtractor,
	contextManager *ContextManager,
	toolRegistry *tools.ToolRegistry,
	toolRouter *ToolRouter,
	formatter *ResponseFormatter,
	suggestionEngine *SuggestionEngine,
	conversationRepo *chatrepo.ConversationRepository,
	predLogger *aigovmiddleware.PredictionLogger,
	producer *events.Producer,
	metrics *VCISOMetrics,
	logger zerolog.Logger,
) *Engine {
	now := func() time.Time { return time.Now().UTC() }
	if contextManager != nil && contextManager.now != nil {
		now = contextManager.now
	}
	return &Engine{
		classifier:       classifier,
		extractor:        extractor,
		contextManager:   contextManager,
		toolRegistry:     toolRegistry,
		toolRouter:       toolRouter,
		formatter:        formatter,
		suggestionEngine: suggestionEngine,
		conversationRepo: conversationRepo,
		predLogger:       predLogger,
		producer:         producer,
		logger:           logger.With().Str("component", "vciso_engine").Logger(),
		metrics:          metrics,
		now:              now,
	}
}

func (e *Engine) Peek(message string) *chatmodel.ClassificationResult {
	if e.classifier == nil {
		return &chatmodel.ClassificationResult{
			Intent:      "unknown",
			Confidence:  0,
			MatchMethod: "fallback",
			MatchedRule: "classifier unavailable",
			Entities:    map[string]string{},
		}
	}
	return e.classifier.Classify(message)
}

func (e *Engine) GetSuggestions(ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID) ([]chatdto.Suggestion, error) {
	var contextState *chatmodel.ConversationContext
	if conversationID != nil && *conversationID != uuid.Nil && e.conversationRepo != nil {
		conversation, err := e.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID)
		if err == nil && conversation != nil {
			contextState = &conversation.LastContext
		}
	}
	if e.suggestionEngine == nil {
		return nil, nil
	}
	return e.suggestionEngine.GetSuggestions(ctx, tenantID, contextState)
}

func (e *Engine) ProcessMessage(ctx context.Context, conversationID *uuid.UUID, tenantID uuid.UUID, userID uuid.UUID, message string, _ string) (*chatdto.ChatResponse, error) {
	message = strings.TrimSpace(message)
	if message == "" {
		return nil, fmt.Errorf("message is required")
	}
	if len(message) > 2000 {
		return nil, fmt.Errorf("message exceeds 2000 characters")
	}

	conversation, contextState, created, err := e.loadOrCreateConversation(ctx, conversationID, tenantID, userID, message)
	if err != nil {
		return nil, err
	}
	if created && e.metrics != nil {
		if e.metrics.ConversationsTotal != nil {
			e.metrics.ConversationsTotal.Inc()
		}
		if e.metrics.ConversationsActive != nil {
			e.metrics.ConversationsActive.Inc()
		}
	}
	if created {
		e.publishEvent(ctx, "com.clario360.cyber.vciso.conversation.started", tenantID, userID, map[string]any{
			"conversation_id": conversation.ID.String(),
			"user_id":         userID.String(),
			"tenant_id":       tenantID.String(),
		})
	}

	classification := e.classifyWithContext(message, &contextState)
	e.observeClassification(classification)
	e.publishEvent(ctx, "com.clario360.cyber.vciso.message.received", tenantID, userID, map[string]any{
		"conversation_id": conversation.ID.String(),
		"user_id":         userID.String(),
		"intent":          classification.Intent,
		"confidence":      classification.Confidence,
		"match_method":    classification.MatchMethod,
	})

	intentPattern := e.intentPattern(classification.Intent)
	entities := map[string]string{}
	if e.extractor != nil {
		entities = e.extractor.Extract(message, classification.Intent)
	}
	for key, value := range classification.Entities {
		entities[key] = value
	}
	resolutionType := "explicit"
	if intentPattern != nil && intentPattern.RequiresEntity {
		explicitValue := entities[intentPattern.EntityType]
		var clarification *ClarificationRequest
		entities, clarification = e.contextManager.ResolveEntities(message, classification.Intent, entities, &contextState, intentPattern.EntityType)
		switch {
		case clarification != nil:
			if e.metrics != nil && e.metrics.ContextResolutionsTotal != nil {
				e.metrics.ContextResolutionsTotal.WithLabelValues("clarification").Inc()
			}
			payload := e.formatter.Clarification(clarification)
			return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, "", nil, 0, nil, payload, nil)
		case explicitValue == "":
			resolutionType = "deictic"
		default:
			resolutionType = "explicit"
		}
		if e.metrics != nil && e.metrics.ContextResolutionsTotal != nil {
			e.metrics.ContextResolutionsTotal.WithLabelValues(resolutionType).Inc()
		}
	}
	if e.contextManager != nil {
		entities = e.contextManager.ApplyFilterCarryover(message, entities, &contextState)
	}

	if classification.Intent == "unknown" || classification.Confidence < 0.30 {
		if e.metrics != nil && e.metrics.UnknownIntentsTotal != nil {
			e.metrics.UnknownIntentsTotal.Inc()
		}
		suggestions, _ := e.GetSuggestions(ctx, &conversation.ID, tenantID, userID)
		e.publishEvent(ctx, "com.clario360.cyber.vciso.unknown_intent", tenantID, userID, map[string]any{
			"conversation_id": conversation.ID.String(),
			"user_id":         userID.String(),
			"message_hash":    hashMessage(message),
		})
		payload := e.formatter.UnknownIntent(suggestions)
		return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, "", nil, 0, nil, payload, nil)
	}

	tool := e.toolRegistry.Get(classification.ToolName)
	if tool == nil {
		payload := e.formatter.ToolError(fmt.Errorf("tool %q is not registered", classification.ToolName))
		return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, classification.ToolName, nil, 0, ptrString("tool is not registered"), payload, nil)
	}

	requiredPerms := tool.RequiredPermissions()
	userPerms := permissionsFromContext(ctx)
	missing := missingPermissions(userPerms, requiredPerms)
	if len(missing) > 0 {
		if e.metrics != nil && e.metrics.PermissionDenialsTotal != nil {
			e.metrics.PermissionDenialsTotal.WithLabelValues(tool.Name()).Inc()
		}
		if e.metrics != nil && e.metrics.ToolExecutionsTotal != nil {
			e.metrics.ToolExecutionsTotal.WithLabelValues(tool.Name(), "permission_denied").Inc()
		}
		e.publishEvent(ctx, "com.clario360.cyber.vciso.permission.denied", tenantID, userID, map[string]any{
			"conversation_id":      conversation.ID.String(),
			"tool_name":            tool.Name(),
			"missing_permissions":  missing,
			"user_id":              userID.String(),
		})
		payload := e.formatter.PermissionDenied(tool.Description(), missing)
		return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, tool.Name(), nil, 0, ptrString(strings.Join(missing, ",")), payload, nil)
	}

	toolResult, latency, toolErr := e.toolRouter.Execute(ctx, tool, tenantID, userID, entities, 10*time.Second)
	if toolErr != nil {
		if errors.Is(toolErr, context.DeadlineExceeded) {
			e.publishEvent(ctx, "com.clario360.cyber.vciso.tool.timeout", tenantID, userID, map[string]any{
				"conversation_id": conversation.ID.String(),
				"tool_name":       tool.Name(),
				"timeout_ms":      10000,
			})
			payload := e.formatter.ToolTimeout()
			return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, tool.Name(), nil, latency, ptrString(toolErr.Error()), payload, nil)
		}
		e.publishEvent(ctx, "com.clario360.cyber.vciso.tool.executed", tenantID, userID, map[string]any{
			"conversation_id": conversation.ID.String(),
			"tool_name":       tool.Name(),
			"latency_ms":      latency.Milliseconds(),
			"success":         false,
			"error":           sanitizeError(toolErr.Error()),
		})
		payload := e.formatter.ToolError(toolErr)
		return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, tool.Name(), nil, latency, ptrString(toolErr.Error()), payload, nil)
	}

	e.publishEvent(ctx, "com.clario360.cyber.vciso.tool.executed", tenantID, userID, map[string]any{
		"conversation_id": conversation.ID.String(),
		"tool_name":       tool.Name(),
		"latency_ms":      latency.Milliseconds(),
		"success":         true,
	})
	if tool.Name() == "dashboard_builder" {
		e.observeDashboard(ctx, tenantID, userID, toolResult)
	}
	if tool.Name() == "remediation" {
		e.observeRemediation(ctx, tenantID, userID, toolResult)
	}

	payload := e.formatter.FormatToolResult(toolResult)
	return e.persistAndRespond(ctx, conversation, &contextState, message, classification, entities, tool.Name(), toolResult, latency, nil, payload, toolResult.Entities)
}

func (e *Engine) loadOrCreateConversation(ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID, firstMessage string) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	if e.conversationRepo == nil {
		return nil, chatmodel.ConversationContext{}, false, fmt.Errorf("conversation repository is required")
	}
	if conversationID != nil && *conversationID != uuid.Nil {
		conversation, err := e.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID)
		if err != nil {
			return nil, chatmodel.ConversationContext{}, false, err
		}
		contextState := conversation.LastContext
		if contextState.ActiveFilters == nil {
			contextState.ActiveFilters = map[string]string{}
		}
		if e.contextManager != nil && e.contextManager.IsExpired(contextState) {
			return e.createConversation(ctx, tenantID, userID, firstMessage)
		}
		return conversation, contextState, false, nil
	}
	return e.createConversation(ctx, tenantID, userID, firstMessage)
}

func (e *Engine) createConversation(ctx context.Context, tenantID, userID uuid.UUID, firstMessage string) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	id := uuid.New()
	contextState := e.contextManager.NewContext(id, userID, tenantID)
	now := e.now()
	conversation := &chatmodel.Conversation{
		ID:            id,
		TenantID:      tenantID,
		UserID:        userID,
		Title:         conversationTitle(firstMessage),
		Status:        chatmodel.ConversationStatusActive,
		MessageCount:  0,
		LastContext:   contextState,
		LastMessageAt: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := e.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, chatmodel.ConversationContext{}, false, err
	}
	return conversation, contextState, true, nil
}

func (e *Engine) classifyWithContext(message string, conversation *chatmodel.ConversationContext) *chatmodel.ClassificationResult {
	if e.classifier == nil {
		return &chatmodel.ClassificationResult{
			Intent:      "unknown",
			Confidence:  0,
			MatchMethod: "fallback",
			MatchedRule: "classifier unavailable",
			Entities:    map[string]string{},
		}
	}
	result := e.classifier.Classify(message)
	if result.Intent != "unknown" || conversation == nil {
		return result
	}
	followUp := inferFollowUpIntent(message, conversation)
	if followUp == "" {
		return result
	}
	pattern := e.intentPattern(followUp)
	if pattern == nil {
		return result
	}
	return &chatmodel.ClassificationResult{
		Intent:      pattern.Intent,
		ToolName:    pattern.ToolName,
		Confidence:  0.60,
		MatchMethod: "fallback",
		MatchedRule: "context_follow_up:" + followUp,
		Entities:    map[string]string{},
	}
}

func (e *Engine) persistAndRespond(
	ctx context.Context,
	conversation *chatmodel.Conversation,
	contextState *chatmodel.ConversationContext,
	message string,
	classification *chatmodel.ClassificationResult,
	entities map[string]string,
	toolName string,
	toolResult *tools.ToolResult,
	latency time.Duration,
	toolError *string,
	payload chatmodel.ResponsePayload,
	responseEntities []chatmodel.EntityReference,
) (*chatdto.ChatResponse, error) {
	predictionLogID := e.logPrediction(ctx, conversation.TenantID, message, classification, entities, toolName, latency, payload, responseEntities)
	now := e.now()

	e.contextManager.AddTurn(contextState, chatmodel.Turn{
		Role:     "user",
		Content:  message,
		Intent:   classification.Intent,
		ToolName: "",
		Entities: entities,
		At:       now,
	})
	entityMap := map[string]string{}
	for _, item := range responseEntities {
		entityMap[item.Type] = item.ID
	}
	e.contextManager.AddTurn(contextState, chatmodel.Turn{
		Role:     "assistant",
		Content:  payload.Text,
		Intent:   classification.Intent,
		ToolName: toolName,
		Entities: entityMap,
		At:       now,
	})
	if responseEntities != nil {
		contextState.LastEntities = responseEntities
	}
	contextState.LastActivityAt = now

	userMessage := &chatmodel.Message{
		ID:                uuid.New(),
		ConversationID:    conversation.ID,
		TenantID:          conversation.TenantID,
		Role:              chatmodel.MessageRoleUser,
		Content:           message,
		Intent:            ptrString(classification.Intent),
		IntentConfidence:  ptrFloat(classification.Confidence),
		MatchMethod:       ptrString(classification.MatchMethod),
		MatchedPattern:    ptrString(classification.MatchedRule),
		ExtractedEntities: entities,
		CreatedAt:         now,
	}
	if err := e.conversationRepo.CreateMessage(ctx, userMessage); err != nil {
		return nil, err
	}

	assistantMessage := &chatmodel.Message{
		ID:               uuid.New(),
		ConversationID:   conversation.ID,
		TenantID:         conversation.TenantID,
		Role:             chatmodel.MessageRoleAssistant,
		Content:          payload.Text,
		Intent:           ptrString(classification.Intent),
		ToolName:         nilIfEmpty(toolName),
		ToolParams:       entities,
		ToolResult:       marshalToolResult(payload.Data),
		ToolLatencyMS:    ptrInt(int(latency.Milliseconds())),
		ToolError:        toolError,
		ResponseType:     ptrString(payload.DataType),
		SuggestedActions: payload.Actions,
		EntityReferences: responseEntities,
		PredictionLogID:  predictionLogID,
		CreatedAt:        now,
	}
	if assistantMessage.ToolLatencyMS != nil && *assistantMessage.ToolLatencyMS == 0 && toolName == "" {
		assistantMessage.ToolLatencyMS = nil
	}
	if err := e.conversationRepo.CreateMessage(ctx, assistantMessage); err != nil {
		return nil, err
	}
	if err := e.conversationRepo.UpdateConversationState(ctx, conversation.ID, conversation.TenantID, 2, contextState); err != nil {
		return nil, err
	}

	e.observeMessages(classification.Intent)

	return &chatdto.ChatResponse{
		ConversationID: conversation.ID,
		MessageID:      assistantMessage.ID,
		Response:       payload,
		Intent:         classification.Intent,
		Confidence:     classification.Confidence,
		Engine:         "rule_based",
		Meta: &chatdto.ResponseMeta{
			Intent:        classification.Intent,
			Confidence:    classification.Confidence,
			LatencyMS:     int(latency.Milliseconds()),
			Grounding:     "passed",
			Engine:        "rule_based",
			RoutingReason: "deterministic classifier",
		},
	}, nil
}

func (e *Engine) logPrediction(
	ctx context.Context,
	tenantID uuid.UUID,
	message string,
	classification *chatmodel.ClassificationResult,
	entities map[string]string,
	toolName string,
	latency time.Duration,
	payload chatmodel.ResponsePayload,
	responseEntities []chatmodel.EntityReference,
) *uuid.UUID {
	if e.predLogger == nil || classification == nil {
		return nil
	}
	result, err := e.predLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:  tenantID,
		ModelSlug: "cyber-vciso-classifier",
		UseCase:   "conversational_ai",
		Input: map[string]any{
			"message":       message,
			"intent":        classification.Intent,
			"confidence":    classification.Confidence,
			"match_method":  classification.MatchMethod,
			"matched_rule":  classification.MatchedRule,
			"entities":      entities,
		},
		InputSummary: map[string]any{
			"intent":       classification.Intent,
			"match_method": classification.MatchMethod,
			"entity_count": len(entities),
		},
		ModelFunc: func(ctx context.Context, input any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"tool_name":       toolName,
					"tool_latency_ms": latency.Milliseconds(),
					"response_type":   payload.DataType,
					"entity_count":    len(responseEntities),
					"action_count":    len(payload.Actions),
				},
				Confidence: classification.Confidence,
				Metadata: map[string]any{
					"matched_rule": classification.MatchedRule,
				},
			}, nil
		},
	})
	if err != nil {
		e.logger.Warn().Err(err).Str("intent", classification.Intent).Msg("vciso prediction logging failed")
		return nil
	}
	return &result.PredictionLogID
}

func (e *Engine) intentPattern(intent string) *chatmodel.IntentPattern {
	if e.classifier == nil {
		return nil
	}
	for _, item := range e.classifier.Intents() {
		if item.Intent == intent {
			return item
		}
	}
	return nil
}

func (e *Engine) observeClassification(result *chatmodel.ClassificationResult) {
	if e.metrics == nil || result == nil {
		return
	}
	if e.metrics.IntentClassifiedTotal != nil {
		e.metrics.IntentClassifiedTotal.WithLabelValues(result.Intent, result.MatchMethod).Inc()
	}
	if e.metrics.IntentConfidence != nil {
		e.metrics.IntentConfidence.WithLabelValues(result.Intent).Observe(result.Confidence)
	}
}

func (e *Engine) observeMessages(intent string) {
	if e.metrics == nil || e.metrics.MessagesTotal == nil {
		return
	}
	if intent == "" {
		intent = "unknown"
	}
	e.metrics.MessagesTotal.WithLabelValues("user", intent).Inc()
	e.metrics.MessagesTotal.WithLabelValues("assistant", intent).Inc()
}

func (e *Engine) observeDashboard(ctx context.Context, tenantID, userID uuid.UUID, result *tools.ToolResult) {
	if e.metrics == nil || e.metrics.DashboardsCreatedTotal == nil || result == nil {
		return
	}
	e.metrics.DashboardsCreatedTotal.Inc()
	if data, ok := result.Data.(map[string]any); ok {
		dashboardID, _ := data["dashboard_id"]
		widgets, _ := data["widgets"]
		e.publishEvent(ctx, "com.clario360.cyber.vciso.dashboard.created", tenantID, userID, map[string]any{
			"dashboard_id": dashboardID,
			"widget_count": countSlice(widgets),
			"user_id":      userID.String(),
			"description":  result.Text,
		})
	}
}

func (e *Engine) observeRemediation(ctx context.Context, tenantID, userID uuid.UUID, result *tools.ToolResult) {
	if result == nil {
		return
	}
	if data, ok := result.Data.(map[string]any); ok {
		e.publishEvent(ctx, "com.clario360.cyber.vciso.remediation.triggered", tenantID, userID, map[string]any{
			"remediation_id": stringifyAny(data["id"]),
			"alert_id":       firstEntityID(result.Entities, "alert"),
			"user_id":        userID.String(),
		})
	}
}

func (e *Engine) publishEvent(ctx context.Context, eventType string, tenantID, userID uuid.UUID, payload map[string]any) {
	if e.producer == nil {
		return
	}
	tenant := tenantID
	if tenant == uuid.Nil && payload != nil {
		if value, ok := payload["tenant_id"].(string); ok {
			if parsed, err := uuid.Parse(value); err == nil {
				tenant = parsed
			}
		}
	}
	if tenant == uuid.Nil {
		return
	}
	event, err := events.NewEvent(eventType, "cyber-service", tenant.String(), payload)
	if err != nil {
		return
	}
	if userID != uuid.Nil {
		event.UserID = userID.String()
	}
	_ = e.producer.Publish(ctx, events.Topics.VCISOEvents, event)
}

func conversationTitle(message string) string {
	value := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")
	if value == "" {
		return "New conversation"
	}
	if len(value) <= 100 {
		return value
	}
	trimmed := value[:100]
	if idx := strings.LastIndex(trimmed, " "); idx > 20 {
		trimmed = trimmed[:idx]
	}
	return strings.TrimSpace(trimmed)
}

func permissionsFromContext(ctx context.Context) []string {
	values := make([]string, 0)
	if claims := auth.ClaimsFromContext(ctx); claims != nil {
		values = append(values, claims.Permissions...)
		for _, role := range claims.Roles {
			normalized := strings.ReplaceAll(role, "-", "_")
			values = append(values, auth.RolePermissions[normalized]...)
		}
	}
	if user := auth.UserFromContext(ctx); user != nil {
		for _, role := range user.Roles {
			normalized := strings.ReplaceAll(role, "-", "_")
			values = append(values, auth.RolePermissions[normalized]...)
		}
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func missingPermissions(have, required []string) []string {
	missing := make([]string, 0)
	for _, need := range required {
		if !permissionGranted(have, need) {
			missing = append(missing, need)
		}
	}
	return missing
}

func permissionGranted(have []string, required string) bool {
	for _, item := range have {
		if item == auth.PermAdminAll || item == required {
			return true
		}
		if strings.HasSuffix(item, ":*") && strings.HasPrefix(required, strings.TrimSuffix(item, "*")) {
			return true
		}
	}
	return false
}

func marshalToolResult(data any) json.RawMessage {
	if data == nil {
		return json.RawMessage(`null`)
	}
	payload, err := json.Marshal(data)
	if err != nil {
		return json.RawMessage(`null`)
	}
	return payload
}

func hashMessage(message string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(message)))
	return hex.EncodeToString(sum[:])
}

func ptrString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func ptrFloat(value float64) *float64 { return &value }

func ptrInt(value int) *int { return &value }

func nilIfEmpty(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func stringifyAny(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	default:
		return fmt.Sprint(typed)
	}
}

func firstEntityID(items []chatmodel.EntityReference, entityType string) string {
	for _, item := range items {
		if item.Type == entityType {
			return item.ID
		}
	}
	return ""
}

func countSlice(value any) int {
	switch typed := value.(type) {
	case []any:
		return len(typed)
	case []map[string]any:
		return len(typed)
	default:
		return 0
	}
}
