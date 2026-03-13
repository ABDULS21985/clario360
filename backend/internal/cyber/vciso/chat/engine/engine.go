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
	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	"github.com/clario360/platform/internal/cyber/vciso/chat/tools"
	"github.com/clario360/platform/internal/events"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrEmptyMessage      = errors.New("engine: message is required")
	ErrMessageTooLong    = errors.New("engine: message exceeds maximum length")
	ErrRepoRequired      = errors.New("engine: conversation repository is required")
	ErrToolNotRegistered = errors.New("engine: tool not registered")
	ErrPermissionDenied  = errors.New("engine: permission denied")
	ErrToolTimeout       = errors.New("engine: tool execution timed out")
	ErrToolExecution     = errors.New("engine: tool execution failed")
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	MaxMessageLength     = 2000
	DefaultToolTimeout   = 10 * time.Second
	RuleBasedEngine      = "rule_based"
	UnknownConfidenceMin = 0.30
)

// ---------------------------------------------------------------------------
// EngineDeps — dependency injection container
// ---------------------------------------------------------------------------

// EngineDeps groups every collaborator the rule-based Engine needs.
type EngineDeps struct {
	Classifier       *IntentClassifier
	Extractor        *EntityExtractor
	ContextManager   *ContextManager
	ToolRegistry     *tools.ToolRegistry
	ToolRouter       *ToolRouter
	Formatter        *ResponseFormatter
	SuggestionEngine *SuggestionEngine
	ConversationRepo *chatrepo.ConversationRepository  // required
	PredLogger       *aigovmiddleware.PredictionLogger // optional
	Producer         *events.Producer                  // optional
	Metrics          *VCISOMetrics                     // optional (nil-safe)
	Logger           zerolog.Logger
	Now              func() time.Time // optional — defaults to UTC now
}

// ---------------------------------------------------------------------------
// Engine
// ---------------------------------------------------------------------------

// Engine is the deterministic, rule-based message processing pipeline.
// It classifies user intent, extracts entities, resolves context, executes
// the matching tool, and formats the response.
//
// Pipeline phases:
//
//	validate → conversation → classify → extract → resolve → authorise
//	→ execute → format → persist → respond
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
	eventBus         *eventBus
	logger           zerolog.Logger
	metrics          *VCISOMetrics
	now              func() time.Time
}

func NewEngine(d EngineDeps) *Engine {
	nowFn := d.Now
	if nowFn == nil {
		if d.ContextManager != nil && d.ContextManager.now != nil {
			nowFn = d.ContextManager.now
		} else {
			nowFn = func() time.Time { return time.Now().UTC() }
		}
	}

	return &Engine{
		classifier:       d.Classifier,
		extractor:        d.Extractor,
		contextManager:   d.ContextManager,
		toolRegistry:     d.ToolRegistry,
		toolRouter:       d.ToolRouter,
		formatter:        d.Formatter,
		suggestionEngine: d.SuggestionEngine,
		conversationRepo: d.ConversationRepo,
		predLogger:       d.PredLogger,
		eventBus:         newEventBus(d.Producer, d.Logger),
		logger:           d.Logger.With().Str("component", "vciso_engine").Logger(),
		metrics:          d.Metrics,
		now:              nowFn,
	}
}

// ===========================================================================
// Public API — peek & suggestions
// ===========================================================================

// Peek classifies a message without executing anything.
func (e *Engine) Peek(message string) *chatmodel.ClassificationResult {
	if e.classifier == nil {
		return unknownClassification("classifier unavailable")
	}
	return e.classifier.Classify(message)
}

// GetSuggestions returns contextual suggestions for the current conversation.
func (e *Engine) GetSuggestions(ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID) ([]chatdto.Suggestion, error) {
	if e.suggestionEngine == nil {
		return nil, nil
	}

	var cs *chatmodel.ConversationContext
	if conversationID != nil && *conversationID != uuid.Nil && e.conversationRepo != nil {
		if conv, err := e.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID); err == nil && conv != nil {
			cs = &conv.LastContext
		}
	}

	return e.suggestionEngine.GetSuggestions(ctx, tenantID, cs)
}

// ===========================================================================
// ProcessMessage — pipeline orchestrator
// ===========================================================================

// ruleState carries intermediate results between pipeline phases.
type ruleState struct {
	message        string
	conversation   *chatmodel.Conversation
	contextState   chatmodel.ConversationContext
	isNew          bool
	classification *chatmodel.ClassificationResult
	entities       map[string]string
	resolutionType string
	toolName       string
	toolResult     *tools.ToolResult
	toolLatency    time.Duration
	toolError      *string
	payload        chatmodel.ResponsePayload
	respEntities   []chatmodel.EntityReference
}

// ProcessMessage runs the full rule-based pipeline.
func (e *Engine) ProcessMessage(
	ctx context.Context,
	conversationID *uuid.UUID,
	tenantID, userID uuid.UUID,
	message string,
	_ string, // routingHint (reserved, unused)
) (*chatdto.ChatResponse, error) {

	// ---- Phase 1: Validate ---------------------------------------------
	msg, err := e.validateMessage(message)
	if err != nil {
		return nil, err
	}

	// ---- Phase 2: Conversation -----------------------------------------
	st := &ruleState{message: msg}
	if err := e.phaseConversation(ctx, st, conversationID, tenantID, userID); err != nil {
		return nil, err
	}

	// ---- Phase 3: Classify (with context follow-up) --------------------
	e.phaseClassify(st)

	// ---- Phase 4: Extract entities -------------------------------------
	e.phaseExtract(st)

	// ---- Phase 5: Resolve entities + filter carry-over -----------------
	if done, resp, err := e.phaseResolve(ctx, st, tenantID, userID); done {
		return resp, err
	}

	// ---- Phase 6: Handle unknown intent --------------------------------
	if done, resp, err := e.phaseUnknown(ctx, st, tenantID, userID); done {
		return resp, err
	}

	// ---- Phase 7: Look up tool + authorise -----------------------------
	tool, done, resp, err := e.phaseAuthorise(ctx, st, tenantID, userID)
	if done {
		return resp, err
	}

	// ---- Phase 8: Execute tool -----------------------------------------
	if done, resp, err := e.phaseExecute(ctx, st, tool, tenantID, userID); done {
		return resp, err
	}

	// ---- Phase 9: Format + persist + respond ---------------------------
	st.payload = e.formatter.FormatToolResult(st.toolResult)
	st.respEntities = st.toolResult.Entities
	return e.persistAndRespond(ctx, st)
}

// ===========================================================================
// Pipeline phases
// ===========================================================================

func (e *Engine) validateMessage(raw string) (string, error) {
	msg := strings.TrimSpace(raw)
	if msg == "" {
		return "", ErrEmptyMessage
	}
	if len(msg) > MaxMessageLength {
		return "", fmt.Errorf("%w: %d characters (max %d)", ErrMessageTooLong, len(msg), MaxMessageLength)
	}
	return msg, nil
}

func (e *Engine) phaseConversation(
	ctx context.Context, st *ruleState,
	conversationID *uuid.UUID, tenantID, userID uuid.UUID,
) error {
	if e.conversationRepo == nil {
		return ErrRepoRequired
	}

	conv, cs, created, err := e.loadOrCreateConversation(ctx, conversationID, tenantID, userID, st.message)
	if err != nil {
		return err
	}
	st.conversation = conv
	st.contextState = cs
	st.isNew = created

	if created {
		safeVCISOInc(e.metrics, metricsConversationsTotal)
		safeVCISOInc(e.metrics, metricsConversationsActive)
		e.eventBus.Publish(ctx, eventConversationStarted, tenantID, userID, map[string]any{
			"conversation_id": conv.ID.String(),
			"user_id":         userID.String(),
			"tenant_id":       tenantID.String(),
		})
	}

	return nil
}

func (e *Engine) phaseClassify(st *ruleState) {
	st.classification = e.classifyWithContext(st.message, &st.contextState)
	e.observeClassification(st.classification)
}

func (e *Engine) phaseExtract(st *ruleState) {
	st.entities = make(map[string]string)
	if e.extractor != nil {
		st.entities = e.extractor.Extract(st.message, st.classification.Intent)
	}
	// Merge classification-level entities (regex captures) over extraction.
	for k, v := range st.classification.Entities {
		st.entities[k] = v
	}
}

// phaseResolve handles entity resolution and filter carry-over.
// Returns (true, resp, err) if the pipeline should short-circuit (clarification needed).
func (e *Engine) phaseResolve(
	ctx context.Context, st *ruleState, tenantID, userID uuid.UUID,
) (bool, *chatdto.ChatResponse, error) {
	intentPattern := e.intentPattern(st.classification.Intent)

	if intentPattern != nil && intentPattern.RequiresEntity {
		explicitValue := st.entities[intentPattern.EntityType]
		var clarification *ClarificationRequest

		st.entities, clarification = e.contextManager.ResolveEntities(
			st.message, st.classification.Intent, st.entities,
			&st.contextState, intentPattern.EntityType,
		)

		if clarification != nil {
			safeVCISOLabelled(e.metrics, metricsContextResolutions, "clarification")
			st.payload = e.formatter.Clarification(clarification)
			resp, err := e.persistAndRespond(ctx, st)
			return true, resp, err
		}

		if explicitValue == "" {
			st.resolutionType = "deictic"
		} else {
			st.resolutionType = "explicit"
		}
		safeVCISOLabelled(e.metrics, metricsContextResolutions, st.resolutionType)
	}

	if e.contextManager != nil {
		st.entities = e.contextManager.ApplyFilterCarryover(st.message, st.entities, &st.contextState)
	}

	return false, nil, nil
}

// phaseUnknown handles unrecognised intents.
func (e *Engine) phaseUnknown(
	ctx context.Context, st *ruleState, tenantID, userID uuid.UUID,
) (bool, *chatdto.ChatResponse, error) {
	if st.classification.Intent != "unknown" && st.classification.Confidence >= UnknownConfidenceMin {
		return false, nil, nil
	}

	safeVCISOInc(e.metrics, metricsUnknownIntents)
	suggestions, _ := e.GetSuggestions(ctx, &st.conversation.ID, tenantID, userID)

	e.eventBus.Publish(ctx, eventUnknownIntent, tenantID, userID, map[string]any{
		"conversation_id": st.conversation.ID.String(),
		"user_id":         userID.String(),
		"message_hash":    hashMessage(st.message),
	})

	st.payload = e.formatter.UnknownIntent(suggestions)
	resp, err := e.persistAndRespond(ctx, st)
	return true, resp, err
}

// phaseAuthorise looks up the tool and checks permissions.
func (e *Engine) phaseAuthorise(
	ctx context.Context, st *ruleState, tenantID, userID uuid.UUID,
) (tools.Tool, bool, *chatdto.ChatResponse, error) {
	tool := e.toolRegistry.Get(st.classification.ToolName)
	if tool == nil {
		st.toolName = st.classification.ToolName
		st.toolError = ptrString("tool is not registered")
		st.payload = e.formatter.ToolError(fmt.Errorf("%w: %s", ErrToolNotRegistered, st.classification.ToolName))
		resp, err := e.persistAndRespond(ctx, st)
		return nil, true, resp, err
	}

	st.toolName = tool.Name()

	missing := missingPermissions(permissionsFromContext(ctx), tool.RequiredPermissions())
	if len(missing) > 0 {
		safeVCISOLabelled(e.metrics, metricsPermissionDenials, tool.Name())
		safeVCISOLabelled(e.metrics, metricsToolExecutions, tool.Name(), "permission_denied")

		e.eventBus.Publish(ctx, eventPermissionDenied, tenantID, userID, map[string]any{
			"conversation_id":     st.conversation.ID.String(),
			"tool_name":           tool.Name(),
			"missing_permissions": missing,
			"user_id":             userID.String(),
		})

		st.toolError = ptrString(strings.Join(missing, ","))
		st.payload = e.formatter.PermissionDenied(tool.Description(), missing)
		resp, err := e.persistAndRespond(ctx, st)
		return nil, true, resp, err
	}

	return tool, false, nil, nil
}

// phaseExecute runs the tool and handles errors.
func (e *Engine) phaseExecute(
	ctx context.Context, st *ruleState, tool tools.Tool, tenantID, userID uuid.UUID,
) (bool, *chatdto.ChatResponse, error) {
	result, latency, toolErr := e.toolRouter.Execute(ctx, tool, tenantID, userID, st.entities, DefaultToolTimeout)
	st.toolLatency = latency

	if toolErr != nil {
		st.toolError = ptrString(toolErr.Error())

		if errors.Is(toolErr, context.DeadlineExceeded) {
			e.eventBus.Publish(ctx, eventToolTimeout, tenantID, userID, map[string]any{
				"conversation_id": st.conversation.ID.String(),
				"tool_name":       tool.Name(),
				"timeout_ms":      DefaultToolTimeout.Milliseconds(),
			})
			st.payload = e.formatter.ToolTimeout()
		} else {
			e.eventBus.Publish(ctx, eventToolExecuted, tenantID, userID, map[string]any{
				"conversation_id": st.conversation.ID.String(),
				"tool_name":       tool.Name(),
				"latency_ms":      latency.Milliseconds(),
				"success":         false,
				"error":           sanitizeError(toolErr.Error()),
			})
			st.payload = e.formatter.ToolError(toolErr)
		}

		resp, err := e.persistAndRespond(ctx, st)
		return true, resp, err
	}

	// Success.
	st.toolResult = result

	e.eventBus.Publish(ctx, eventToolExecuted, tenantID, userID, map[string]any{
		"conversation_id": st.conversation.ID.String(),
		"tool_name":       tool.Name(),
		"latency_ms":      latency.Milliseconds(),
		"success":         true,
	})

	e.observeToolSpecific(ctx, tenantID, userID, tool.Name(), result)

	return false, nil, nil
}

// ===========================================================================
// Persist + respond
// ===========================================================================

func (e *Engine) persistAndRespond(ctx context.Context, st *ruleState) (*chatdto.ChatResponse, error) {
	predictionLogID := e.logPrediction(ctx, st)
	now := e.now()

	// --- Context turns ---
	e.contextManager.AddTurn(&st.contextState, chatmodel.Turn{
		Role: "user", Content: st.message,
		Intent: st.classification.Intent, Entities: st.entities, At: now,
	})

	entityMap := make(map[string]string, len(st.respEntities))
	for _, item := range st.respEntities {
		entityMap[item.Type] = item.ID
	}
	e.contextManager.AddTurn(&st.contextState, chatmodel.Turn{
		Role: "assistant", Content: st.payload.Text,
		Intent: st.classification.Intent, ToolName: st.toolName, Entities: entityMap, At: now,
	})

	if st.respEntities != nil {
		st.contextState.LastEntities = st.respEntities
	}
	st.contextState.LastActivityAt = now

	// --- Persist user message ---
	userMsg := &chatmodel.Message{
		ID:                uuid.New(),
		ConversationID:    st.conversation.ID,
		TenantID:          st.conversation.TenantID,
		Role:              chatmodel.MessageRoleUser,
		Content:           st.message,
		Intent:            ptrString(st.classification.Intent),
		IntentConfidence:  ptrFloat(st.classification.Confidence),
		MatchMethod:       ptrString(st.classification.MatchMethod),
		MatchedPattern:    ptrString(st.classification.MatchedRule),
		ExtractedEntities: st.entities,
		CreatedAt:         now,
	}
	if err := e.conversationRepo.CreateMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	// --- Persist assistant message ---
	assistantMsg := &chatmodel.Message{
		ID:               uuid.New(),
		ConversationID:   st.conversation.ID,
		TenantID:         st.conversation.TenantID,
		Role:             chatmodel.MessageRoleAssistant,
		Content:          st.payload.Text,
		Intent:           ptrString(st.classification.Intent),
		ToolName:         nilIfEmpty(st.toolName),
		ToolParams:       st.entities,
		ToolResult:       marshalToolResult(st.payload.Data),
		ToolLatencyMS:    toolLatencyPtr(st.toolName, st.toolLatency),
		ToolError:        st.toolError,
		ResponseType:     ptrString(st.payload.DataType),
		SuggestedActions: st.payload.Actions,
		EntityReferences: st.respEntities,
		PredictionLogID:  predictionLogID,
		CreatedAt:        now,
	}
	if err := e.conversationRepo.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("persist assistant message: %w", err)
	}

	// --- Update conversation state ---
	if err := e.conversationRepo.UpdateConversationState(
		ctx, st.conversation.ID, st.conversation.TenantID, 2, &st.contextState,
	); err != nil {
		return nil, fmt.Errorf("update conversation state: %w", err)
	}

	e.observeMessages(st.classification.Intent)

	return &chatdto.ChatResponse{
		ConversationID: st.conversation.ID,
		MessageID:      assistantMsg.ID,
		Response:       st.payload,
		Intent:         st.classification.Intent,
		Confidence:     st.classification.Confidence,
		Engine:         RuleBasedEngine,
		Meta: &chatdto.ResponseMeta{
			Intent:        st.classification.Intent,
			Confidence:    st.classification.Confidence,
			LatencyMS:     int(st.toolLatency.Milliseconds()),
			Grounding:     "passed",
			Engine:        RuleBasedEngine,
			RoutingReason: "deterministic classifier",
		},
	}, nil
}

// ===========================================================================
// Classification with context follow-up
// ===========================================================================

func (e *Engine) classifyWithContext(message string, cs *chatmodel.ConversationContext) *chatmodel.ClassificationResult {
	if e.classifier == nil {
		return unknownClassification("classifier unavailable")
	}

	result := e.classifier.Classify(message)
	if result.Intent != "unknown" || cs == nil {
		return result
	}

	followUp := e.contextManager.InferFollowUpIntent(message, cs)
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

// ===========================================================================
// Conversation lifecycle
// ===========================================================================

func (e *Engine) loadOrCreateConversation(
	ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID, firstMessage string,
) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	if conversationID != nil && *conversationID != uuid.Nil {
		conv, err := e.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID)
		if err != nil {
			return nil, chatmodel.ConversationContext{}, false, err
		}
		cs := conv.LastContext
		if cs.ActiveFilters == nil {
			cs.ActiveFilters = map[string]string{}
		}
		if e.contextManager != nil && e.contextManager.IsExpired(cs) {
			return e.createConversation(ctx, tenantID, userID, firstMessage)
		}
		return conv, cs, false, nil
	}
	return e.createConversation(ctx, tenantID, userID, firstMessage)
}

func (e *Engine) createConversation(
	ctx context.Context, tenantID, userID uuid.UUID, firstMessage string,
) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	id := uuid.New()
	cs := e.contextManager.NewContext(id, userID, tenantID)
	now := e.now()
	conv := &chatmodel.Conversation{
		ID:            id,
		TenantID:      tenantID,
		UserID:        userID,
		Title:         conversationTitle(firstMessage),
		Status:        chatmodel.ConversationStatusActive,
		MessageCount:  0,
		LastContext:   cs,
		LastMessageAt: nil,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := e.conversationRepo.CreateConversation(ctx, conv); err != nil {
		return nil, chatmodel.ConversationContext{}, false, err
	}
	return conv, cs, true, nil
}

// ===========================================================================
// Prediction logging
// ===========================================================================

func (e *Engine) logPrediction(ctx context.Context, st *ruleState) *uuid.UUID {
	if e.predLogger == nil || st.classification == nil {
		return nil
	}
	result, err := e.predLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:  st.conversation.TenantID,
		ModelSlug: "cyber-vciso-classifier",
		UseCase:   "conversational_ai",
		Input: map[string]any{
			"message":      st.message,
			"intent":       st.classification.Intent,
			"confidence":   st.classification.Confidence,
			"match_method": st.classification.MatchMethod,
			"matched_rule": st.classification.MatchedRule,
			"entities":     st.entities,
		},
		InputSummary: map[string]any{
			"intent":       st.classification.Intent,
			"match_method": st.classification.MatchMethod,
			"entity_count": len(st.entities),
		},
		ModelFunc: func(_ context.Context, _ any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"tool_name":       st.toolName,
					"tool_latency_ms": st.toolLatency.Milliseconds(),
					"response_type":   st.payload.DataType,
					"entity_count":    len(st.respEntities),
					"action_count":    len(st.payload.Actions),
				},
				Confidence: st.classification.Confidence,
				Metadata:   map[string]any{"matched_rule": st.classification.MatchedRule},
			}, nil
		},
	})
	if err != nil {
		e.logger.Warn().Err(err).
			Str("intent", st.classification.Intent).
			Msg("vciso prediction logging failed")
		return nil
	}
	return &result.PredictionLogID
}

// ===========================================================================
// Observability helpers
// ===========================================================================

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

func (e *Engine) observeToolSpecific(ctx context.Context, tenantID, userID uuid.UUID, toolName string, result *tools.ToolResult) {
	if result == nil {
		return
	}
	switch toolName {
	case "dashboard_builder":
		if e.metrics != nil && e.metrics.DashboardsCreatedTotal != nil {
			e.metrics.DashboardsCreatedTotal.Inc()
		}
		if data, ok := result.Data.(map[string]any); ok {
			e.eventBus.Publish(ctx, eventDashboardCreated, tenantID, userID, map[string]any{
				"dashboard_id": data["dashboard_id"],
				"widget_count": countSlice(data["widgets"]),
				"user_id":      userID.String(),
				"description":  result.Text,
			})
		}
	case "remediation":
		if data, ok := result.Data.(map[string]any); ok {
			e.eventBus.Publish(ctx, eventRemediationTriggered, tenantID, userID, map[string]any{
				"remediation_id": stringifyAny(data["id"]),
				"alert_id":       firstEntityID(result.Entities, "alert"),
				"user_id":        userID.String(),
			})
		}
	}
}

// ===========================================================================
// Event bus — thin wrapper around the events.Producer
// ===========================================================================

// Event type constants — centralised to prevent typos and enable search.
const (
	eventConversationStarted  = "com.clario360.cyber.vciso.conversation.started"
	eventMessageReceived      = "com.clario360.cyber.vciso.message.received"
	eventUnknownIntent        = "com.clario360.cyber.vciso.unknown_intent"
	eventPermissionDenied     = "com.clario360.cyber.vciso.permission.denied"
	eventToolExecuted         = "com.clario360.cyber.vciso.tool.executed"
	eventToolTimeout          = "com.clario360.cyber.vciso.tool.timeout"
	eventDashboardCreated     = "com.clario360.cyber.vciso.dashboard.created"
	eventRemediationTriggered = "com.clario360.cyber.vciso.remediation.triggered"
)

type eventBus struct {
	producer *events.Producer
	logger   zerolog.Logger
}

func newEventBus(producer *events.Producer, logger zerolog.Logger) *eventBus {
	return &eventBus{producer: producer, logger: logger}
}

// Publish fires an event.  It is nil-safe and never returns an error —
// event publishing must never block the response path.
func (eb *eventBus) Publish(ctx context.Context, eventType string, tenantID, userID uuid.UUID, payload map[string]any) {
	if eb == nil || eb.producer == nil || tenantID == uuid.Nil {
		return
	}

	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), payload)
	if err != nil {
		eb.logger.Debug().Err(err).Str("event_type", eventType).Msg("failed to create event")
		return
	}
	if userID != uuid.Nil {
		event.UserID = userID.String()
	}
	if err := eb.producer.Publish(ctx, events.Topics.VCISOEvents, event); err != nil {
		eb.logger.Debug().Err(err).Str("event_type", eventType).Msg("failed to publish event")
	}
}

// ===========================================================================
// Metrics abstraction — nil-safe increment helpers
// ===========================================================================

// Metric key constants to avoid string typos in call-sites.
const (
	metricsConversationsTotal  = "conversations_total"
	metricsConversationsActive = "conversations_active"
	metricsUnknownIntents      = "unknown_intents_total"
	metricsContextResolutions  = "context_resolutions_total"
	metricsPermissionDenials   = "permission_denials_total"
	metricsToolExecutions      = "tool_executions_total"
)

// safeVCISOInc increments a simple counter on VCISOMetrics if non-nil.
func safeVCISOInc(m *VCISOMetrics, name string) {
	if m == nil {
		return
	}
	switch name {
	case metricsConversationsTotal:
		if m.ConversationsTotal != nil {
			m.ConversationsTotal.Inc()
		}
	case metricsConversationsActive:
		if m.ConversationsActive != nil {
			m.ConversationsActive.Inc()
		}
	case metricsUnknownIntents:
		if m.UnknownIntentsTotal != nil {
			m.UnknownIntentsTotal.Inc()
		}
	}
}

// safeVCISOLabelled increments a labelled counter on VCISOMetrics.
func safeVCISOLabelled(m *VCISOMetrics, name string, labels ...string) {
	if m == nil {
		return
	}
	switch name {
	case metricsContextResolutions:
		if m.ContextResolutionsTotal != nil && len(labels) > 0 {
			m.ContextResolutionsTotal.WithLabelValues(labels[0]).Inc()
		}
	case metricsPermissionDenials:
		if m.PermissionDenialsTotal != nil && len(labels) > 0 {
			m.PermissionDenialsTotal.WithLabelValues(labels[0]).Inc()
		}
	case metricsToolExecutions:
		if m.ToolExecutionsTotal != nil && len(labels) >= 2 {
			m.ToolExecutionsTotal.WithLabelValues(labels[0], labels[1]).Inc()
		}
	}
}

// ===========================================================================
// Pure helpers (package-private)
// ===========================================================================

func unknownClassification(reason string) *chatmodel.ClassificationResult {
	return &chatmodel.ClassificationResult{
		Intent: "unknown", Confidence: 0,
		MatchMethod: "fallback", MatchedRule: reason,
		Entities: map[string]string{},
	}
}

func toolLatencyPtr(toolName string, latency time.Duration) *int {
	if toolName == "" {
		return nil
	}
	ms := int(latency.Milliseconds())
	return &ms
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

// func sanitizeError(msg string) string {
// 	if len(msg) > 200 {
// 		return msg[:200] + "..."
// 	}
// 	return msg
// }
