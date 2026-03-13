package engine

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatengine "github.com/clario360/platform/internal/cyber/vciso/chat/engine"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
)

// ---------------------------------------------------------------------------
// Engine identifiers
// ---------------------------------------------------------------------------

// EngineID is a typed constant for engine selection.
type EngineID string

const (
	EngineRuleBased EngineID = "rule_based"
	EngineLLM       EngineID = "llm"
)

// ---------------------------------------------------------------------------
// EngineDecision — the routing verdict
// ---------------------------------------------------------------------------

// EngineDecision captures the full reasoning behind a routing choice.
// It is both the internal return value of Route() and the basis for
// the routing_reason field exposed to clients and dashboards.
type EngineDecision struct {
	Engine        EngineID
	Reason        string
	RuleBasedHint *chatmodel.ClassificationResult
	Confidence    float64

	// Signals records every signal that contributed to the decision,
	// keyed by signal name.  Useful for debugging and A/B analysis.
	Signals map[string]SignalResult
}

// ---------------------------------------------------------------------------
// Routing signals — pluggable scorers
// ---------------------------------------------------------------------------

// SignalResult is the output of a single routing signal.
type SignalResult struct {
	Name     string  // human-readable label
	Score    float64 // positive = favours LLM, negative = favours rule engine
	Reason   string  // one-line explanation
	Computed bool    // false if the signal was skipped (e.g. no conversation context)
}

// RoutingSignal evaluates one dimension of the routing decision.
// Implementations must be safe for concurrent use and should be fast
// (no network calls — use data already loaded into RoutingContext).
type RoutingSignal interface {
	Name() string
	Evaluate(rctx *RoutingContext) SignalResult
}

// RoutingContext bundles everything a signal might need.  It is built
// once per Route() call and passed to every signal.
type RoutingContext struct {
	Message         string
	MessageLower    string // pre-lowered for signal convenience
	Hint            *chatmodel.ClassificationResult
	ConversationCtx *chatmodel.ConversationContext
	LLMAvailable    bool
	PreferEngine    string // raw caller preference, already lowered+trimmed
}

// ---------------------------------------------------------------------------
// Built-in signals
// ---------------------------------------------------------------------------

// confidenceSignal routes high-confidence single-intent messages to the
// rule engine and low-confidence messages to the LLM.
type confidenceSignal struct {
	highThreshold float64 // above this → rule engine
	lowThreshold  float64 // below this → LLM
}

func (s *confidenceSignal) Name() string { return "confidence" }

func (s *confidenceSignal) Evaluate(rctx *RoutingContext) SignalResult {
	conf := rctx.Hint.Confidence
	switch {
	case conf >= s.highThreshold:
		return SignalResult{Name: s.Name(), Score: -1.0, Reason: "high rule-engine confidence", Computed: true}
	case conf < s.lowThreshold:
		return SignalResult{Name: s.Name(), Score: 1.0, Reason: "low rule-engine confidence", Computed: true}
	default:
		return SignalResult{Name: s.Name(), Score: 0.3, Reason: "medium confidence, slight LLM preference", Computed: true}
	}
}

// multiIntentSignal detects messages that contain multiple questions or
// action verbs chained with conjunctions.
type multiIntentSignal struct{}

func (s *multiIntentSignal) Name() string { return "multi_intent" }

func (s *multiIntentSignal) Evaluate(rctx *RoutingContext) SignalResult {
	if isMultiIntent(rctx.MessageLower) {
		return SignalResult{Name: s.Name(), Score: 1.0, Reason: "multiple intents detected", Computed: true}
	}
	return SignalResult{Name: s.Name(), Score: 0, Reason: "single intent", Computed: true}
}

// reasoningSignal detects messages that require explanation, comparison,
// or executive-level synthesis.
type reasoningSignal struct{}

func (s *reasoningSignal) Name() string { return "reasoning" }

func (s *reasoningSignal) Evaluate(rctx *RoutingContext) SignalResult {
	if marker := requiresReasoningMarker(rctx.MessageLower); marker != "" {
		return SignalResult{Name: s.Name(), Score: 1.0, Reason: "reasoning marker: " + marker, Computed: true}
	}
	return SignalResult{Name: s.Name(), Score: 0, Reason: "no reasoning markers", Computed: true}
}

// inferenceSignal detects anaphoric references ("it", "that", "those")
// that require conversation context to resolve.
type inferenceSignal struct{}

func (s *inferenceSignal) Name() string { return "inference" }

func (s *inferenceSignal) Evaluate(rctx *RoutingContext) SignalResult {
	if rctx.ConversationCtx == nil || len(rctx.ConversationCtx.LastEntities) == 0 {
		return SignalResult{Name: s.Name(), Score: 0, Reason: "no conversation context", Computed: false}
	}
	if marker := detectAnaphora(rctx.MessageLower); marker != "" {
		return SignalResult{Name: s.Name(), Score: 0.8, Reason: "anaphoric reference: " + marker, Computed: true}
	}
	return SignalResult{Name: s.Name(), Score: 0, Reason: "no anaphoric references", Computed: true}
}

// conversationAffinitySignal keeps a conversation on the same engine it
// started on, reducing context-switching confusion for the user.
type conversationAffinitySignal struct{}

func (s *conversationAffinitySignal) Name() string { return "conversation_affinity" }

func (s *conversationAffinitySignal) Evaluate(rctx *RoutingContext) SignalResult {
	if rctx.ConversationCtx == nil {
		return SignalResult{Name: s.Name(), Score: 0, Reason: "new conversation", Computed: false}
	}
	// Check if prior turns used the LLM engine.
	for _, turn := range rctx.ConversationCtx.Turns {
		if turn.ToolName == "llm" {
			return SignalResult{Name: s.Name(), Score: 0.5, Reason: "conversation previously used LLM", Computed: true}
		}
	}
	return SignalResult{Name: s.Name(), Score: -0.3, Reason: "conversation has been rule-based", Computed: true}
}

// messageLengthSignal gives a mild LLM preference for long messages,
// which tend to be more nuanced than short commands.
type messageLengthSignal struct {
	longThreshold int // word count
}

func (s *messageLengthSignal) Name() string { return "message_length" }

func (s *messageLengthSignal) Evaluate(rctx *RoutingContext) SignalResult {
	words := len(strings.Fields(rctx.Message))
	if words >= s.longThreshold {
		return SignalResult{Name: s.Name(), Score: 0.4, Reason: "long message favours LLM", Computed: true}
	}
	return SignalResult{Name: s.Name(), Score: 0, Reason: "short message", Computed: true}
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	DefaultConfidenceHigh    = 0.85
	DefaultConfidenceLow     = 0.50
	DefaultLLMScoreThreshold = 0.5 // aggregate score above this → LLM
	DefaultLongMessageWords  = 25
)

// RouterOption configures the EngineRouter.
type RouterOption func(*EngineRouter)

// WithConfidenceThresholds sets the high/low boundaries for the
// confidence signal.
func WithConfidenceThresholds(high, low float64) RouterOption {
	return func(r *EngineRouter) {
		if high > 0 {
			r.confidenceHigh = high
		}
		if low > 0 {
			r.confidenceLow = low
		}
	}
}

// WithLLMScoreThreshold sets the aggregate score above which the router
// selects the LLM engine.
func WithLLMScoreThreshold(t float64) RouterOption {
	return func(r *EngineRouter) {
		if t > 0 {
			r.llmScoreThreshold = t
		}
	}
}

// WithExtraSignals appends custom routing signals evaluated after the
// built-in ones.
func WithExtraSignals(signals ...RoutingSignal) RouterOption {
	return func(r *EngineRouter) {
		r.extraSignals = append(r.extraSignals, signals...)
	}
}

// WithRouterLogger injects a structured logger.
func WithRouterLogger(l zerolog.Logger) RouterOption {
	return func(r *EngineRouter) { r.logger = l }
}

// WithRouterMetrics injects the shared metrics collector.
func WithRouterMetrics(m *Metrics) RouterOption {
	return func(r *EngineRouter) { r.metrics = m }
}

// ---------------------------------------------------------------------------
// EngineRouter
// ---------------------------------------------------------------------------

// EngineRouter decides whether a message is handled by the rule-based
// engine or the LLM engine.  The decision is based on:
//
//  1. Explicit caller preference (overrides everything)
//  2. LLM availability (hard gate)
//  3. An aggregate score from pluggable signals (soft decision)
//
// Signals produce scores on a [-1, +1] scale: positive favours LLM,
// negative favours the rule engine.  The scores are summed; if the
// aggregate exceeds llmScoreThreshold the LLM is selected.
//
// This design makes it easy to:
//   - add new routing dimensions without touching existing logic
//   - tune thresholds per deployment via options
//   - inspect exactly why a decision was made (Signals map on EngineDecision)
type EngineRouter struct {
	ruleEngine       *chatengine.Engine
	llmEngine        *LLMEngine
	conversationRepo *chatrepo.ConversationRepository
	logger           zerolog.Logger
	metrics          *Metrics

	// Thresholds
	confidenceHigh    float64
	confidenceLow     float64
	llmScoreThreshold float64

	// Signal chain: built-in + extras
	builtinSignals []RoutingSignal
	extraSignals   []RoutingSignal
}

func NewEngineRouter(
	ruleEngine *chatengine.Engine,
	llmEngine *LLMEngine,
	conversationRepo *chatrepo.ConversationRepository,
	opts ...RouterOption,
) *EngineRouter {
	r := &EngineRouter{
		ruleEngine:        ruleEngine,
		llmEngine:         llmEngine,
		conversationRepo:  conversationRepo,
		logger:            zerolog.Nop(),
		confidenceHigh:    DefaultConfidenceHigh,
		confidenceLow:     DefaultConfidenceLow,
		llmScoreThreshold: DefaultLLMScoreThreshold,
	}

	for _, opt := range opts {
		opt(r)
	}

	// Build the signal chain after options are applied so thresholds
	// are already set.
	r.builtinSignals = []RoutingSignal{
		&confidenceSignal{highThreshold: r.confidenceHigh, lowThreshold: r.confidenceLow},
		&multiIntentSignal{},
		&reasoningSignal{},
		&inferenceSignal{},
		&conversationAffinitySignal{},
		&messageLengthSignal{longThreshold: DefaultLongMessageWords},
	}

	return r
}

// ===========================================================================
// Public API — pass-through conveniences
// ===========================================================================

// Peek classifies a message via the rule engine without executing it.
func (r *EngineRouter) Peek(message string) *chatmodel.ClassificationResult {
	if r.ruleEngine == nil {
		return &chatmodel.ClassificationResult{Intent: "unknown", Confidence: 0}
	}
	return r.ruleEngine.Peek(message)
}

// GetSuggestions returns contextual suggestions from the rule engine.
func (r *EngineRouter) GetSuggestions(
	ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID,
) ([]chatdto.Suggestion, error) {
	if r.ruleEngine == nil {
		return nil, nil
	}
	return r.ruleEngine.GetSuggestions(ctx, conversationID, tenantID, userID)
}

// ===========================================================================
// ProcessMessage — route then execute
// ===========================================================================

// ProcessMessage decides the engine, delegates, and stamps routing metadata.
func (r *EngineRouter) ProcessMessage(
	ctx context.Context,
	conversationID *uuid.UUID,
	tenantID, userID uuid.UUID,
	message string,
	preferEngine string,
) (*chatdto.ChatResponse, error) {
	decision := r.Route(ctx, conversationID, tenantID, userID, message, preferEngine)

	r.recordRouting(decision)

	switch decision.Engine {
	case EngineRuleBased:
		resp, err := r.ruleEngine.ProcessMessage(ctx, conversationID, tenantID, userID, message, string(EngineRuleBased))
		if err != nil {
			return nil, err
		}
		if resp.Meta != nil {
			resp.Meta.RoutingReason = decision.Reason
		}
		return resp, nil

	default:
		return r.llmEngine.ProcessMessage(ctx, ProcessMessageInput{
			ConversationID: conversationID,
			TenantID:       tenantID,
			UserID:         userID,
			Message:        message,
			Hint:           decision.RuleBasedHint,
			RoutingReason:  decision.Reason,
		})
	}
}

// ===========================================================================
// Route — the decision algorithm
// ===========================================================================

// Route evaluates all signals and returns the routing decision.  It does
// NOT execute the message — call ProcessMessage for that.
func (r *EngineRouter) Route(
	ctx context.Context,
	conversationID *uuid.UUID,
	tenantID, userID uuid.UUID,
	message string,
	preferEngine string,
) EngineDecision {
	start := time.Now()
	prefer := strings.ToLower(strings.TrimSpace(preferEngine))

	// ---- Hard overrides (no signal evaluation needed) -------------------

	// Caller explicitly wants rule engine, or LLM is nil.
	if prefer == string(EngineRuleBased) || r.llmEngine == nil {
		return r.decide(EngineRuleBased, "explicit_preference_or_llm_disabled", nil, 1.0, nil)
	}

	hint := r.Peek(message)
	llmAvailable := r.llmEngine.Available(ctx, tenantID)

	// Caller explicitly wants LLM.
	if prefer == string(EngineLLM) {
		if !llmAvailable {
			return r.decide(EngineRuleBased, "llm_unavailable_despite_preference", hint, 1.0, nil)
		}
		return r.decide(EngineLLM, "explicit_llm_preference", hint, 1.0, nil)
	}

	// LLM not available — no choice.
	if !llmAvailable {
		return r.decide(EngineRuleBased, "llm_unavailable", hint, 1.0, nil)
	}

	// ---- Build routing context -----------------------------------------

	rctx := &RoutingContext{
		Message:         message,
		MessageLower:    strings.ToLower(strings.TrimSpace(message)),
		Hint:            hint,
		ConversationCtx: r.loadConversationContext(ctx, conversationID, tenantID, userID),
		LLMAvailable:    true,
		PreferEngine:    prefer,
	}

	// ---- Evaluate signals ----------------------------------------------

	signals := make(map[string]SignalResult, len(r.builtinSignals)+len(r.extraSignals))
	var aggregate float64

	for _, sig := range r.builtinSignals {
		result := sig.Evaluate(rctx)
		signals[result.Name] = result
		aggregate += result.Score
	}
	for _, sig := range r.extraSignals {
		result := sig.Evaluate(rctx)
		signals[result.Name] = result
		aggregate += result.Score
	}

	// ---- Make decision based on aggregate score ------------------------

	var (
		engine EngineID
		reason string
	)

	if aggregate >= r.llmScoreThreshold {
		engine = EngineLLM
		reason = r.dominantReason(signals, true)
	} else {
		engine = EngineRuleBased
		reason = r.dominantReason(signals, false)
	}

	confidence := hint.Confidence
	if engine == EngineLLM && confidence < 0.6 {
		confidence = 0.6
	}

	decision := EngineDecision{
		Engine:        engine,
		Reason:        reason,
		RuleBasedHint: hint,
		Confidence:    confidence,
		Signals:       signals,
	}

	r.logDecision(decision, aggregate, time.Since(start))

	return decision
}

// ---------------------------------------------------------------------------
// Internals
// ---------------------------------------------------------------------------

func (r *EngineRouter) decide(
	engine EngineID, reason string,
	hint *chatmodel.ClassificationResult, confidence float64,
	signals map[string]SignalResult,
) EngineDecision {
	return EngineDecision{
		Engine:        engine,
		Reason:        reason,
		RuleBasedHint: hint,
		Confidence:    confidence,
		Signals:       signals,
	}
}

// dominantReason picks the signal with the highest absolute score in the
// winning direction and builds a human-readable reason string.
func (r *EngineRouter) dominantReason(signals map[string]SignalResult, forLLM bool) string {
	var (
		bestName  string
		bestScore float64
	)

	for _, sig := range signals {
		if !sig.Computed {
			continue
		}
		absScore := sig.Score
		if !forLLM {
			absScore = -absScore // flip: most negative = strongest rule-engine signal
		}
		if absScore > bestScore {
			bestScore = absScore
			bestName = sig.Name
		}
	}

	if bestName == "" {
		if forLLM {
			return "aggregate_score_favours_llm"
		}
		return "aggregate_score_favours_rules"
	}

	if forLLM {
		return bestName + "_favours_llm"
	}
	return bestName + "_favours_rules"
}

func (r *EngineRouter) loadConversationContext(
	ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID,
) *chatmodel.ConversationContext {
	if r.conversationRepo == nil || conversationID == nil || *conversationID == uuid.Nil {
		return nil
	}
	conversation, err := r.conversationRepo.GetConversation(ctx, tenantID, userID, *conversationID)
	if err != nil || conversation == nil {
		return nil
	}
	return &conversation.LastContext
}

// ---------------------------------------------------------------------------
// Observability
// ---------------------------------------------------------------------------

func (r *EngineRouter) logDecision(d EngineDecision, aggregate float64, latency time.Duration) {
	event := r.logger.Debug().
		Str("engine", string(d.Engine)).
		Str("reason", d.Reason).
		Float64("confidence", d.Confidence).
		Float64("aggregate_score", aggregate).
		Dur("routing_latency", latency)

	// Append computed signal scores.
	for name, sig := range d.Signals {
		if sig.Computed {
			event = event.Float64("signal_"+name, sig.Score)
		}
	}

	event.Msg("routing decision made")
}

func (r *EngineRouter) recordRouting(d EngineDecision) {
	if r.metrics == nil {
		return
	}
	// Increment a counter partitioned by engine and reason.
	safeInc(r.metrics.CallsTotal, string(d.Engine), "router", d.Reason)
}

// ---------------------------------------------------------------------------
// Heuristic helpers (package-private, used by built-in signals)
// ---------------------------------------------------------------------------

// isMultiIntent detects messages with multiple questions or chained
// action verbs.
func isMultiIntent(lower string) bool {
	if strings.Count(lower, "?") > 1 {
		return true
	}

	actionVerbs := []string{
		"show", "build", "investigate", "compare",
		"generate", "explain", "list", "create", "analyse", "analyze",
	}
	verbCount := 0
	for _, verb := range actionVerbs {
		if strings.Contains(lower, verb) {
			verbCount++
		}
	}

	hasChaining := strings.Contains(lower, " and ") ||
		strings.Contains(lower, " also ") ||
		strings.Contains(lower, " plus ") ||
		strings.Contains(lower, " then ") ||
		strings.Contains(lower, ", then ")

	return hasChaining && verbCount >= 2
}

// requiresReasoningMarker returns the first reasoning marker found, or "".
func requiresReasoningMarker(lower string) string {
	markers := []string{
		"why", "explain", "reason", "cause", "because",
		"what if", "compare", "versus", "better", "worse",
		"should i", "recommend", "board briefing", "executive summary",
		"trade-off", "tradeoff", "pros and cons", "risk assessment",
		"root cause", "correlat",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return m
		}
	}
	return ""
}

// detectAnaphora returns the first anaphoric reference found, or "".
func detectAnaphora(lower string) string {
	markers := []string{
		"it", "that", "those", "them", "their",
		"the first one", "the second one", "the same",
		"this one", "the above", "the previous",
	}
	for _, m := range markers {
		if strings.Contains(lower, m) {
			return m
		}
	}
	return ""
}

