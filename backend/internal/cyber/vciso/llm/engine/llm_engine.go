package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/aigovernance"
	aigovmiddleware "github.com/clario360/platform/internal/aigovernance/middleware"
	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatengine "github.com/clario360/platform/internal/cyber/vciso/chat/engine"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	llmcfg "github.com/clario360/platform/internal/cyber/vciso/llm"
	llmgovernance "github.com/clario360/platform/internal/cyber/vciso/llm/governance"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
	llmprovider "github.com/clario360/platform/internal/cyber/vciso/llm/provider"
	llmrepo "github.com/clario360/platform/internal/cyber/vciso/llm/repository"
	llmtools "github.com/clario360/platform/internal/cyber/vciso/llm/tools"
)

// ---------------------------------------------------------------------------
// Deps — constructor dependency injection container
// ---------------------------------------------------------------------------

// Deps groups every collaborator the engine needs.  Using a struct instead of
// 18 positional constructor args means:
//   - adding a dependency never breaks existing call-sites
//   - test doubles are obvious (just set the field)
//   - the compiler catches missing required fields via linting, not at runtime
type Deps struct {
	Cfg              *llmcfg.Config                    // required
	ConversationRepo *chatrepo.ConversationRepository  // required
	ContextManager   *chatengine.ContextManager        // required
	PromptBuilder    *PromptBuilder                    // required
	ContextCompiler  *ContextCompiler                  // required
	ToolRegistry     *llmtools.Registry                // required
	ToolExecutor     *ToolCallExecutor                 // required
	ResponseSynth    *ResponseSynthesizer              // required
	Hallucination    *HallucinationGuard               // required
	InjectionGuard   *InjectionGuard                   // required
	PIIFilter        *llmgovernance.PIIFilter          // required
	RateLimiter      *llmgovernance.RateLimiter        // required
	AuditRepo        *llmrepo.LLMAuditRepository       // optional (nil = no audit)
	ProviderManager  *llmprovider.Manager              // required
	FallbackHandler  *FallbackHandler                  // required
	PredLogger       *aigovmiddleware.PredictionLogger // optional (nil = no prediction log)
	Metrics          *Metrics                          // optional (nil-safe helpers used)
	Logger           zerolog.Logger
	Now              func() time.Time // optional — defaults to time.Now().UTC
}

// ---------------------------------------------------------------------------
// LLMEngine
// ---------------------------------------------------------------------------

// LLMEngine orchestrates message processing through a deterministic pipeline:
//
//	rate-limit → conversation → sanitize → provider → context → prompt
//	→ tool-loop → grounding → PII → synthesis → persist → audit
//
// Each phase is a separate method operating on a shared processingState,
// making the flow easy to trace, test, and extend.
type LLMEngine struct {
	cfg              *llmcfg.Config
	conversationRepo *chatrepo.ConversationRepository
	contextManager   *chatengine.ContextManager
	promptBuilder    *PromptBuilder
	contextCompiler  *ContextCompiler
	toolRegistry     *llmtools.Registry
	toolExecutor     *ToolCallExecutor
	responseSynth    *ResponseSynthesizer
	hallucination    *HallucinationGuard
	injectionGuard   *InjectionGuard
	piiFilter        *llmgovernance.PIIFilter
	rateLimiter      *llmgovernance.RateLimiter
	auditRepo        *llmrepo.LLMAuditRepository
	providerManager  *llmprovider.Manager
	fallbackHandler  *FallbackHandler
	predLogger       *aigovmiddleware.PredictionLogger
	metrics          *Metrics
	logger           zerolog.Logger
	now              func() time.Time
}

// NewLLMEngine constructs an engine from the dependency container.
func NewLLMEngine(d Deps) *LLMEngine {
	nowFn := d.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	return &LLMEngine{
		cfg:              d.Cfg,
		conversationRepo: d.ConversationRepo,
		contextManager:   d.ContextManager,
		promptBuilder:    d.PromptBuilder,
		contextCompiler:  d.ContextCompiler,
		toolRegistry:     d.ToolRegistry,
		toolExecutor:     d.ToolExecutor,
		responseSynth:    d.ResponseSynth,
		hallucination:    d.Hallucination,
		injectionGuard:   d.InjectionGuard,
		piiFilter:        d.PIIFilter,
		rateLimiter:      d.RateLimiter,
		auditRepo:        d.AuditRepo,
		providerManager:  d.ProviderManager,
		fallbackHandler:  d.FallbackHandler,
		predLogger:       d.PredLogger,
		metrics:          d.Metrics,
		logger:           d.Logger.With().Str("component", "vciso_llm_engine").Logger(),
		now:              nowFn,
	}
}

// ===========================================================================
// Public API — health, availability, usage (unchanged signatures)
// ===========================================================================

func (e *LLMEngine) Available(ctx context.Context, tenantID uuid.UUID) bool {
	if e == nil || e.providerManager == nil || e.cfg == nil || !e.cfg.Enabled {
		return false
	}
	provider, err := e.providerManager.Resolve(ctx, tenantID)
	if err != nil {
		return false
	}
	status, err := provider.HealthCheck(ctx)
	if err != nil {
		return false
	}
	return status.Status == "healthy" || status.Status == "degraded"
}

func (e *LLMEngine) Health(ctx context.Context, tenantID uuid.UUID) (*llmprovider.HealthStatus, error) {
	provider, err := e.providerManager.Resolve(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderResolveFailed, err)
	}
	return provider.HealthCheck(ctx)
}

func (e *LLMEngine) Usage(ctx context.Context, tenantID uuid.UUID) (*llmmodel.UsageStats, error) {
	return e.auditRepo.UsageStats(ctx, tenantID)
}

// ===========================================================================
// ProcessMessage — pipeline orchestrator
// ===========================================================================

// ProcessMessage is the primary entry-point.  It delegates to discrete phase
// methods and uses the shared processingState to thread intermediate results.
func (e *LLMEngine) ProcessMessage(ctx context.Context, in ProcessMessageInput) (*chatdto.ChatResponse, error) {
	st := newProcessingState(in)

	// ---- Phase 1: Rate limit -------------------------------------------
	if err := e.phaseRateLimit(ctx, st); err != nil {
		return e.fallback(ctx, st, "llm_rate_limit")
	}

	// ---- Phase 2: Conversation load/create -----------------------------
	if err := e.phaseConversation(ctx, st); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConversationLoadFailed, err)
	}

	// ---- Phase 3: Injection guard --------------------------------------
	if err := e.phaseSanitize(ctx, st); err != nil {
		return nil, err
	}
	if st.sanitized.Blocked {
		return e.shortCircuitBlocked(ctx, st)
	}

	// ---- Phase 4: Provider resolution ----------------------------------
	if err := e.phaseProvider(ctx, st); err != nil {
		return e.fallback(ctx, st, "provider_unavailable")
	}

	// ---- Phase 5: Context compilation ----------------------------------
	if err := e.phaseContextCompile(ctx, st); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContextCompileFailed, err)
	}

	// ---- Phase 6: Prompt build -----------------------------------------
	if err := e.phasePromptBuild(ctx, st); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPromptBuildFailed, err)
	}

	// ---- Phase 7: Tool loop (LLM calls + tool execution) ---------------
	if err := e.phaseToolLoop(ctx, st); err != nil {
		return e.fallback(ctx, st, "provider_error")
	}

	// ---- Phase 8: Grounding check --------------------------------------
	e.phaseGrounding(st)

	// ---- Phase 9: PII filter -------------------------------------------
	e.phasePIIFilter(st)

	// ---- Phase 10: Synthesis -------------------------------------------
	e.phaseSynthesis(st)

	// ---- Phase 11: Prediction logging ----------------------------------
	e.phasePredictionLog(ctx, st)

	// ---- Phase 12: Rate-limit consumption ------------------------------
	e.phaseRateLimitConsume(ctx, st)

	// ---- Phase 13: Persist + audit + respond ---------------------------
	resp, err := e.phasePersist(ctx, st)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPersistFailed, err)
	}

	// ---- Observability: record total latency and phase breakdown -------
	e.recordMetrics(st)

	return resp, nil
}

// ===========================================================================
// Pipeline phases
// ===========================================================================

func (e *LLMEngine) phaseRateLimit(ctx context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhaseRateLimit, func() {
		err = e.rateLimiter.Check(ctx, st.input.TenantID, st.input.UserID)
	})
	if err != nil {
		safeInc(e.metrics.FallbackTotal, "rate_limit")
		return fmt.Errorf("%w: %v", ErrRateLimited, err)
	}
	return nil
}

func (e *LLMEngine) phaseConversation(ctx context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhaseConversation, func() {
		st.conversation, st.contextState, st.isNew, err = e.loadOrCreateConversation(
			ctx, st.input.ConversationID, st.input.TenantID, st.input.UserID, st.input.Message,
		)
	})
	return err
}

func (e *LLMEngine) phaseSanitize(_ context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhaseSanitize, func() {
		st.sanitized, err = e.injectionGuard.Sanitize(st.input.Message)
	})
	return err
}

func (e *LLMEngine) phaseProvider(ctx context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhaseProvider, func() {
		st.provider, err = e.providerManager.Resolve(ctx, st.input.TenantID)
	})
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderResolveFailed, err)
	}
	return nil
}

func (e *LLMEngine) phaseContextCompile(ctx context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhaseContext, func() {
		st.compiledCtx, err = e.contextCompiler.Compile(ctx, &st.conversation.ID, st.input.TenantID)
	})
	return err
}

func (e *LLMEngine) phasePromptBuild(ctx context.Context, st *processingState) error {
	var err error
	st.phases.Track(PhasePrompt, func() {
		st.prompt, err = e.promptBuilder.Build(ctx, st.input.TenantID, st.input.UserID, &st.contextState, st.input.Hint)
	})
	if err != nil {
		return err
	}
	st.messages = append(st.compiledCtx.Messages, llmmodel.LLMMessage{
		Role:    "user",
		Content: st.sanitized.Sanitized,
	})
	st.toolSchemas = llmtools.GenerateToolSchemas(e.toolRegistry.List(), st.provider.Name())
	return nil
}

func (e *LLMEngine) phaseToolLoop(ctx context.Context, st *processingState) error {
	var loopErr error

	st.phases.Track(PhaseToolLoop, func() {
		maxIter := e.cfg.Safety.MaxToolLoopIterations
		temperature := providerConfigTemperature(
			e.providerManager.GetConfig(st.input.TenantID), e.cfg, st.provider.Name(),
		)

		for iteration := 1; iteration <= maxIter; iteration++ {
			// Honour context cancellation between iterations.
			if err := checkContext(ctx); err != nil {
				loopErr = err
				return
			}

			response, callErr := st.provider.Complete(ctx, &llmprovider.CompletionRequest{
				SystemPrompt: st.prompt.SystemPrompt,
				Messages:     st.messages,
				Tools:        st.toolSchemas,
				MaxTokens:    e.cfg.Tokens.ResponseMax,
				Temperature:  temperature,
			})
			if callErr != nil {
				safeInc(e.metrics.FallbackTotal, "provider_error")
				loopErr = fmt.Errorf("%w: %v", ErrProviderCallFailed, callErr)
				return
			}

			st.promptTokens += response.Usage.PromptTokens
			st.completionTokens += response.Usage.CompletionTokens
			st.toolLoopIters = iteration
			safeInc(e.metrics.CallsTotal, st.provider.Name(), st.provider.Model(), "success")

			// No tool calls → terminal text response.
			if len(response.ToolCalls) == 0 {
				st.finalText = response.Content
				return
			}

			// Execute tool calls and feed results back into messages.
			results, _ := e.toolExecutor.ExecuteAll(ctx, response.ToolCalls, st.input.TenantID, st.input.UserID)
			st.toolResults = append(st.toolResults, results...)
			st.reasoningTrace = append(st.reasoningTrace, llmmodel.ReasoningStep{
				Step:      iteration,
				Action:    "tool_calls",
				Detail:    fmt.Sprintf("%d tool call(s) executed", len(response.ToolCalls)),
				ToolNames: toolNames(response.ToolCalls),
			})
			for idx, result := range results {
				st.messages = append(st.messages, llmmodel.LLMMessage{
					Role:       "tool",
					Name:       response.ToolCalls[idx].FunctionName,
					ToolCallID: response.ToolCalls[idx].ID,
					Content:    marshalToolMessage(result),
				})
			}
		}

		// All iterations consumed without a text response.
		if strings.TrimSpace(st.finalText) == "" {
			st.finalText = fallbackNarrative(st.toolResults)
			e.logger.Warn().
				Int("max_iterations", maxIter).
				Msg("tool loop exhausted without text response; using fallback narrative")
		}
	})

	return loopErr
}

func (e *LLMEngine) phaseGrounding(st *processingState) {
	st.phases.Track(PhaseGrounding, func() {
		st.grounding = e.hallucination.Check(st.finalText, st.toolResults)
		if st.grounding != nil && st.grounding.Status == "blocked" {
			e.logger.Warn().Str("detail", formatGroundingFailure(st.grounding)).Msg("grounding blocked; falling back to tool narrative")
			st.finalText = fallbackNarrative(st.toolResults)
		}
	})
}

func (e *LLMEngine) phasePIIFilter(st *processingState) {
	st.phases.Track(PhasePII, func() {
		st.filteredText, st.piiDetections = e.piiFilter.Filter(st.finalText)
	})
}

func (e *LLMEngine) phaseSynthesis(st *processingState) {
	st.phases.Track(PhaseSynthesis, func() {
		st.payload, st.meta = e.responseSynth.Synthesize(SynthesisInput{
			Text:        st.filteredText,
			ToolResults: st.toolResults,
			Grounding:   st.grounding,
		})
		st.meta.Intent = semanticIntent(st.input.Hint)
		st.meta.Confidence = semanticConfidence(st.input.Hint)
		st.meta.Engine = "llm"
		st.meta.RoutingReason = st.input.RoutingReason
		st.meta.TokensUsed = st.totalTokens()
		st.meta.LatencyMS = int(st.elapsed().Milliseconds())
	})
}

func (e *LLMEngine) phasePredictionLog(ctx context.Context, st *processingState) {
	st.predictionLogID = e.logPrediction(ctx, st.input.TenantID, st.input.Message, st.meta, st.toolResults)
}

func (e *LLMEngine) phaseRateLimitConsume(ctx context.Context, st *processingState) {
	cost := st.provider.EstimateCost(st.promptTokens, st.completionTokens)
	if err := e.rateLimiter.Consume(ctx, st.input.TenantID, st.totalTokens(), cost); err != nil {
		e.logger.Warn().Err(err).Msg("failed to consume llm rate limit")
	}
}

func (e *LLMEngine) phasePersist(ctx context.Context, st *processingState) (*chatdto.ChatResponse, error) {
	var (
		resp *chatdto.ChatResponse
		err  error
	)
	st.phases.Track(PhasePersist, func() {
		resp, err = e.persistAndRespond(ctx, persistInput{
			conversation:     st.conversation,
			contextState:     &st.contextState,
			originalMessage:  st.input.Message,
			classification:   st.classification,
			entities:         map[string]string{},
			toolResults:      st.toolResults,
			payload:          st.payload,
			sanitized:        st.sanitized,
			grounding:        st.grounding,
			routingReason:    st.input.RoutingReason,
			latency:          st.elapsed(),
			predictionLogID:  st.predictionLogID,
			promptTokens:     st.promptTokens,
			completionTokens: st.completionTokens,
			piiDetections:    st.piiDetections,
			reasoningTrace:   st.reasoningTrace,
			prompt:           st.prompt,
		})
	})
	return resp, err
}

// ===========================================================================
// Short-circuit helpers
// ===========================================================================

// shortCircuitBlocked handles the case where the injection guard blocks the
// message before any LLM call is made.
func (e *LLMEngine) shortCircuitBlocked(ctx context.Context, st *processingState) (*chatdto.ChatResponse, error) {
	payload := chatmodel.ResponsePayload{
		Text:     st.sanitized.Sanitized,
		DataType: "text",
		Actions:  []chatmodel.SuggestedAction{},
		Entities: []chatmodel.EntityReference{},
	}
	return e.persistAndRespond(ctx, persistInput{
		conversation:    st.conversation,
		contextState:    &st.contextState,
		originalMessage: st.input.Message,
		classification:  st.classification,
		entities:        map[string]string{},
		payload:         payload,
		sanitized:       st.sanitized,
		routingReason:   st.input.RoutingReason,
		latency:         st.elapsed(),
	})
}

// fallback delegates to the FallbackHandler with metrics tracking.
func (e *LLMEngine) fallback(ctx context.Context, st *processingState, reason string) (*chatdto.ChatResponse, error) {
	safeInc(e.metrics.FallbackTotal, reason)
	return e.fallbackHandler.Handle(
		ctx, st.input.ConversationID, st.input.TenantID, st.input.UserID,
		st.input.Message, FallbackReason(reason),
	)
}

// ===========================================================================
// Metrics recording
// ===========================================================================

func (e *LLMEngine) recordMetrics(st *processingState) {
	if e == nil || e.metrics == nil || st == nil {
		return
	}
	safeObserveVec(e.metrics.ResponseLatencySeconds, st.elapsed().Seconds(), "llm")
	safeObserve(e.metrics.ToolLoopIterations, float64(maxInt(st.toolLoopIters, 1)))
	safeObserve(e.metrics.ToolCallsPerQuery, float64(len(st.toolResults)))
	if st.provider != nil {
		safeAdd(e.metrics.TokensTotal, float64(st.promptTokens), st.provider.Name(), st.provider.Model(), "prompt")
		safeAdd(e.metrics.TokensTotal, float64(st.completionTokens), st.provider.Name(), st.provider.Model(), "completion")
		safeObserveVec(e.metrics.CallLatencySeconds, st.elapsed().Seconds(), st.provider.Name(), st.provider.Model())
		safeAdd(e.metrics.CostUSDTotal, st.provider.EstimateCost(st.promptTokens, st.completionTokens), st.provider.Name(), st.provider.Model(), st.input.TenantID.String())
	}
	if st.grounding != nil && e.metrics.GroundingResultsTotal != nil {
		e.metrics.GroundingResultsTotal.WithLabelValues(st.grounding.Status).Inc()
	}
}

// ===========================================================================
// Conversation management (unchanged logic, cleaned up)
// ===========================================================================

func (e *LLMEngine) loadOrCreateConversation(
	ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID, firstMessage string,
) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	if conversationID != nil && *conversationID != uuid.Nil {
		conversation, err := e.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID)
		if err == nil && conversation != nil {
			cs := conversation.LastContext
			if e.contextManager != nil && e.contextManager.IsExpired(cs) {
				return e.createConversation(ctx, tenantID, userID, firstMessage)
			}
			return conversation, cs, false, nil
		}
	}
	return e.createConversation(ctx, tenantID, userID, firstMessage)
}

func (e *LLMEngine) createConversation(
	ctx context.Context, tenantID, userID uuid.UUID, firstMessage string,
) (*chatmodel.Conversation, chatmodel.ConversationContext, bool, error) {
	id := uuid.New()
	cs := e.contextManager.NewContext(id, userID, tenantID)
	now := e.now()
	conversation := &chatmodel.Conversation{
		ID:           id,
		TenantID:     tenantID,
		UserID:       userID,
		Title:        conversationTitle(firstMessage),
		Status:       chatmodel.ConversationStatusActive,
		MessageCount: 0,
		LastContext:  cs,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := e.conversationRepo.CreateConversation(ctx, conversation); err != nil {
		return nil, chatmodel.ConversationContext{}, false, err
	}
	return conversation, cs, true, nil
}

// ===========================================================================
// Persist, audit, respond
// ===========================================================================

func (e *LLMEngine) persistAndRespond(ctx context.Context, in persistInput) (*chatdto.ChatResponse, error) {
	now := e.now()

	// --- Update conversation context turns ---
	if e.contextManager != nil {
		e.contextManager.AddTurn(in.contextState, chatmodel.Turn{
			Role: "user", Content: in.originalMessage,
			Intent: in.classification.Intent, Entities: in.entities, At: now,
		})
		entityMap := make(map[string]string, len(in.payload.Entities))
		for _, item := range in.payload.Entities {
			entityMap[item.Type] = item.ID
		}
		e.contextManager.AddTurn(in.contextState, chatmodel.Turn{
			Role: "assistant", Content: in.payload.Text,
			Intent: in.classification.Intent, ToolName: "llm", Entities: entityMap, At: now,
		})
		in.contextState.LastEntities = in.payload.Entities
	}

	// --- Persist user message ---
	userMsg := &chatmodel.Message{
		ID:             uuid.New(),
		ConversationID: in.conversation.ID,
		TenantID:       in.conversation.TenantID,
		Role:           chatmodel.MessageRoleUser,
		Content:        in.originalMessage,
		Intent:         ptrString(in.classification.Intent),
		CreatedAt:      now,
	}
	if err := e.conversationRepo.CreateMessage(ctx, userMsg); err != nil {
		return nil, fmt.Errorf("persist user message: %w", err)
	}

	// --- Persist assistant message ---
	assistantMsg := &chatmodel.Message{
		ID:               uuid.New(),
		ConversationID:   in.conversation.ID,
		TenantID:         in.conversation.TenantID,
		Role:             chatmodel.MessageRoleAssistant,
		Content:          in.payload.Text,
		Intent:           ptrString(in.classification.Intent),
		ToolName:         ptrString("llm"),
		ToolResult:       marshalPayload(in.payload.Data),
		ToolLatencyMS:    ptrInt(int(in.latency.Milliseconds())),
		ResponseType:     ptrString(in.payload.DataType),
		SuggestedActions: in.payload.Actions,
		EntityReferences: in.payload.Entities,
		PredictionLogID:  in.predictionLogID,
		CreatedAt:        now,
	}
	if err := e.conversationRepo.CreateMessage(ctx, assistantMsg); err != nil {
		return nil, fmt.Errorf("persist assistant message: %w", err)
	}

	// --- Update conversation state ---
	if err := e.conversationRepo.UpdateConversationState(
		ctx, in.conversation.ID, in.conversation.TenantID, 2, in.contextState,
	); err != nil {
		return nil, fmt.Errorf("update conversation state: %w", err)
	}

	// --- Audit (fire-and-forget; failure must not block the response) ---
	e.writeAuditLog(ctx, in, assistantMsg.ID, now)

	// --- Build response ---
	return &chatdto.ChatResponse{
		ConversationID: in.conversation.ID,
		MessageID:      assistantMsg.ID,
		Response:       in.payload,
		Intent:         in.classification.Intent,
		Confidence:     semanticConfidence(in.classification),
		Engine:         "llm",
		Meta: &chatdto.ResponseMeta{
			Intent:         in.classification.Intent,
			Confidence:     semanticConfidence(in.classification),
			ToolCallsCount: len(in.toolResults),
			ReasoningSteps: maxInt(len(in.reasoningTrace), 1),
			LatencyMS:      int(in.latency.Milliseconds()),
			TokensUsed:     in.promptTokens + in.completionTokens,
			Grounding:      mapGrounding(in.grounding),
			Engine:         "llm",
			RoutingReason:  in.routingReason,
		},
	}, nil
}

// writeAuditLog writes the LLM audit record.  Errors are logged but never
// propagated — audit failure must not block the user response.
func (e *LLMEngine) writeAuditLog(ctx context.Context, in persistInput, msgID uuid.UUID, now time.Time) {
	if e.auditRepo == nil {
		return
	}

	toolAudit := make([]llmmodel.ToolCallAudit, 0, len(in.toolResults))
	for _, r := range in.toolResults {
		if r == nil {
			continue
		}
		toolAudit = append(toolAudit, llmmodel.ToolCallAudit{
			Name:          r.ToolName,
			ResultSummary: r.Summary,
			Success:       r.Success,
			LatencyMs:     r.LatencyMs,
			CalledAt:      now,
		})
	}
	toolAuditJSON, _ := json.Marshal(toolAudit)

	tenantCfg := e.providerManager.GetConfig(in.conversation.TenantID)

	if err := e.auditRepo.CreateAudit(ctx, &llmmodel.AuditLog{
		MessageID:           msgID,
		ConversationID:      in.conversation.ID,
		TenantID:            in.conversation.TenantID,
		UserID:              in.conversation.UserID,
		Provider:            providerName(tenantCfg, e.cfg),
		Model:               tenantCfg.Model,
		PromptTokens:        in.promptTokens,
		CompletionTokens:    in.completionTokens,
		TotalTokens:         in.promptTokens + in.completionTokens,
		EstimatedCostUSD:    0,
		LLMLatencyMS:        int(in.latency.Milliseconds()),
		TotalLatencyMS:      int(in.latency.Milliseconds()),
		SystemPromptHash:    promptHash(in.prompt),
		SystemPromptVersion: promptVersion(in.prompt, e.cfg),
		UserMessage:         originalOrSanitized(in.sanitized, in.originalMessage),
		ContextTurns:        len(in.contextState.Turns),
		RawCompletion:       in.payload.Text,
		ToolCallsJSON:       toolAuditJSON,
		ToolCallCount:       len(toolAudit),
		ReasoningTrace:      marshalPayload(in.reasoningTrace),
		GroundingResult:     mapGrounding(in.grounding),
		PIIDetections:       in.piiDetections,
		InjectionFlags:      injectionCount(in.sanitized),
		FinalResponse:       in.payload.Text,
		PredictionLogID:     in.predictionLogID,
		EngineUsed:          "llm",
		RoutingReason:       in.routingReason,
	}); err != nil {
		e.logger.Error().Err(err).
			Str("message_id", msgID.String()).
			Msg("failed to write LLM audit log")
	}
}

// ===========================================================================
// Prediction logging
// ===========================================================================

func (e *LLMEngine) logPrediction(
	ctx context.Context, tenantID uuid.UUID, message string,
	meta *chatdto.ResponseMeta, toolResults []*llmmodel.ToolCallResult,
) *uuid.UUID {
	if e.predLogger == nil {
		return nil
	}
	result, err := e.predLogger.Predict(ctx, aigovernance.PredictParams{
		TenantID:  tenantID,
		ModelSlug: "cyber-vciso-llm",
		UseCase:   "conversational_ai",
		Input: map[string]any{
			"message":      message,
			"tool_results": toolResults,
			"meta":         meta,
		},
		InputSummary: map[string]any{
			"tool_calls": len(toolResults),
			"engine":     meta.Engine,
			"grounding":  meta.Grounding,
		},
		ModelFunc: func(ctx context.Context, input any) (*aigovernance.ModelOutput, error) {
			return &aigovernance.ModelOutput{
				Output: map[string]any{
					"tool_calls": len(toolResults),
					"grounding":  meta.Grounding,
					"tokens":     meta.TokensUsed,
				},
				Confidence: meta.Confidence,
				Metadata: map[string]any{
					"routing_reason": meta.RoutingReason,
				},
			}, nil
		},
	})
	if err != nil {
		e.logger.Warn().Err(err).Msg("llm prediction logging failed")
		return nil
	}
	return &result.PredictionLogID
}

// ===========================================================================
// Pure helper functions (package-private)
// ===========================================================================

func hintOrUnknown(hint *chatmodel.ClassificationResult) *chatmodel.ClassificationResult {
	if hint != nil {
		return hint
	}
	return &chatmodel.ClassificationResult{Intent: "semantic_query", Confidence: 0.7}
}

func semanticIntent(hint *chatmodel.ClassificationResult) string {
	if hint == nil || hint.Intent == "" || hint.Intent == "unknown" {
		return "semantic_query"
	}
	return hint.Intent
}

func semanticConfidence(hint *chatmodel.ClassificationResult) float64 {
	if hint == nil || hint.Confidence == 0 {
		return 0.72
	}
	return hint.Confidence
}

func providerConfigTemperature(override llmprovider.TenantOverride, cfg *llmcfg.Config, name string) float64 {
	if override.Temperature > 0 {
		return override.Temperature
	}
	if cfg != nil {
		if pc, ok := cfg.Providers[name]; ok && pc.Temperature > 0 {
			return pc.Temperature
		}
	}
	return 0.1
}

func toolNames(items []llmmodel.LLMToolCall) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.FunctionName)
	}
	return out
}

func fallbackNarrative(toolResults []*llmmodel.ToolCallResult) string {
	if len(toolResults) == 0 {
		return "I couldn't verify enough data to answer confidently."
	}
	lines := make([]string, 0, len(toolResults))
	for _, r := range toolResults {
		if r == nil {
			continue
		}
		lines = append(lines, r.Summary)
	}
	return strings.Join(lines, "\n")
}

func mapGrounding(result *llmmodel.GroundingResult) string {
	if result == nil || result.Status == "" {
		return "passed"
	}
	return result.Status
}

func injectionCount(msg *SanitizedMessage) int {
	if msg == nil {
		return 0
	}
	return len(msg.Flags)
}

func ptrString(v string) *string {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return &v
}

func ptrInt(v int) *int { return &v }

func conversationTitle(v string) string {
	v = strings.TrimSpace(v)
	if len(v) <= 80 {
		return v
	}
	return strings.TrimSpace(v[:80])
}

func providerName(override llmprovider.TenantOverride, cfg *llmcfg.Config) string {
	if override.Provider != "" {
		return override.Provider
	}
	if cfg != nil {
		return cfg.DefaultProvider
	}
	return "openai"
}

func promptHash(p *LLMPrompt) string {
	if p == nil {
		return ""
	}
	return p.Hash
}

func promptVersion(p *LLMPrompt, cfg *llmcfg.Config) string {
	if p != nil && p.Version != "" {
		return p.Version
	}
	if cfg != nil {
		return cfg.Prompt.ActiveVersion
	}
	return "v1.0"
}

func originalOrSanitized(msg *SanitizedMessage, fallback string) string {
	if msg == nil {
		return fallback
	}
	if strings.TrimSpace(msg.Original) != "" {
		return msg.Original
	}
	if strings.TrimSpace(msg.Sanitized) != "" {
		return msg.Sanitized
	}
	return fallback
}
