package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatengine "github.com/clario360/platform/internal/cyber/vciso/chat/engine"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

// ---------------------------------------------------------------------------
// Sentinel errors
// ---------------------------------------------------------------------------

var (
	ErrFallbackExhausted = errors.New("fallback: all strategies exhausted")
	ErrFallbackDisabled  = errors.New("fallback: handler disabled by policy")
)

// ---------------------------------------------------------------------------
// Constants & defaults
// ---------------------------------------------------------------------------

const (
	DefaultFallbackNote = "I'm providing a simplified response while my advanced reasoning is temporarily unavailable."

	// Engine label stamped onto every fallback response.
	fallbackEngine = "fallback"
)

// ---------------------------------------------------------------------------
// FallbackReason — known reasons for entering the fallback path
// ---------------------------------------------------------------------------

// FallbackReason is a typed string so callers can't silently mistype a reason.
type FallbackReason string

const (
	ReasonRateLimit           FallbackReason = "llm_rate_limit"
	ReasonProviderUnavailable FallbackReason = "provider_unavailable"
	ReasonProviderError       FallbackReason = "provider_error"
	ReasonGroundingBlocked    FallbackReason = "grounding_blocked"
	ReasonTimeout             FallbackReason = "timeout"
	ReasonUnknown             FallbackReason = "unknown"
)

// ---------------------------------------------------------------------------
// ReasonPolicy — per-reason behaviour tuning
// ---------------------------------------------------------------------------

// ReasonPolicy controls how the handler behaves for a specific fallback
// reason.  Missing policies fall back to default behaviour.
type ReasonPolicy struct {
	// Note overrides the degradation notice prepended to the response.
	// Empty string = use the default note.
	Note string

	// Suppress, when true, omits the degradation notice entirely.
	// Useful for reasons like rate-limiting where the user already
	// knows something is constrained.
	SuppressNote bool

	// Disabled, when true, causes the handler to return
	// ErrFallbackDisabled instead of attempting the rule engine.
	Disabled bool

	// Timeout overrides the context deadline for the fallback call.
	// Zero = no override (inherits parent context).
	Timeout time.Duration
}

// ---------------------------------------------------------------------------
// FallbackStrategy interface
// ---------------------------------------------------------------------------

// FallbackStrategy abstracts a single fallback mechanism.  The default is
// the rule engine, but you can chain additional strategies (cached responses,
// static responses, etc.) via WithExtraStrategies.
type FallbackStrategy interface {
	// Name returns a human-readable label for logging/metrics.
	Name() string

	// Handle attempts to produce a response.  Returning a non-nil error
	// signals the chain to try the next strategy.
	Handle(ctx context.Context, conversationID *uuid.UUID, tenantID, userID uuid.UUID, message string) (*chatdto.ChatResponse, error)
}

// ---------------------------------------------------------------------------
// ruleEngineStrategy — wraps the existing rule engine
// ---------------------------------------------------------------------------

type ruleEngineStrategy struct {
	engine *chatengine.Engine
}

func (s *ruleEngineStrategy) Name() string { return "rule_engine" }

func (s *ruleEngineStrategy) Handle(
	ctx context.Context, conversationID *uuid.UUID,
	tenantID, userID uuid.UUID, message string,
) (*chatdto.ChatResponse, error) {
	return s.engine.ProcessMessage(ctx, conversationID, tenantID, userID, message, "rule_based")
}

// ---------------------------------------------------------------------------
// staticStrategy — last-resort hardcoded response
// ---------------------------------------------------------------------------

type staticStrategy struct {
	text string
}

func (s *staticStrategy) Name() string { return "static" }

func (s *staticStrategy) Handle(
	_ context.Context, conversationID *uuid.UUID,
	_ uuid.UUID, _ uuid.UUID, _ string,
) (*chatdto.ChatResponse, error) {
	convID := uuid.Nil
	if conversationID != nil {
		convID = *conversationID
	}
	return &chatdto.ChatResponse{
		ConversationID: convID,
		MessageID:      uuid.New(),
		Response: chatmodel.ResponsePayload{
			Text:     s.text,
			DataType: "text",
			Actions:  []chatmodel.SuggestedAction{},
			Entities: []chatmodel.EntityReference{},
		},
		Engine: fallbackEngine,
		Meta: &chatdto.ResponseMeta{
			Engine:    fallbackEngine,
			Grounding: "passed",
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

// FallbackOption configures the FallbackHandler.
type FallbackOption func(*FallbackHandler)

// WithFallbackNote overrides the default degradation notice.
func WithFallbackNote(note string) FallbackOption {
	return func(h *FallbackHandler) {
		if note != "" {
			h.defaultNote = note
		}
	}
}

// WithReasonPolicy registers a per-reason policy.
func WithReasonPolicy(reason FallbackReason, policy ReasonPolicy) FallbackOption {
	return func(h *FallbackHandler) {
		h.policies[reason] = policy
	}
}

// WithExtraStrategies appends additional fallback strategies after the
// rule engine in the chain.  They are tried in order; the first to
// return a non-nil response wins.
func WithExtraStrategies(strategies ...FallbackStrategy) FallbackOption {
	return func(h *FallbackHandler) {
		h.chain = append(h.chain, strategies...)
	}
}

// WithStaticFallback appends a last-resort static-text strategy.
func WithStaticFallback(text string) FallbackOption {
	return func(h *FallbackHandler) {
		h.chain = append(h.chain, &staticStrategy{text: text})
	}
}

// WithFallbackLogger injects a structured logger.
func WithFallbackLogger(l zerolog.Logger) FallbackOption {
	return func(h *FallbackHandler) { h.logger = l }
}

// WithFallbackMetrics injects the shared metrics collector.
func WithFallbackMetrics(m *Metrics) FallbackOption {
	return func(h *FallbackHandler) { h.metrics = m }
}

// ---------------------------------------------------------------------------
// FallbackHandler
// ---------------------------------------------------------------------------

// FallbackHandler manages graceful degradation when the primary LLM path
// fails.  It runs a chain of strategies (rule engine → extras → static)
// and decorates the winning response with a degradation notice and
// fallback metadata.
//
// Design goals:
//   - Never lose the user's message: something always responds
//   - Make degradation visible to both the user (note) and the system (meta)
//   - Per-reason policy control for ops flexibility
//   - Full observability: counters, per-reason breakdowns, chain depth
type FallbackHandler struct {
	chain       []FallbackStrategy
	defaultNote string
	policies    map[FallbackReason]ReasonPolicy
	metrics     *Metrics
	logger      zerolog.Logger

	// Counters — lock-free, suitable for high-throughput paths.
	totalFallbacks atomic.Int64
	totalExhausted atomic.Int64
}

func NewFallbackHandler(ruleEngine *chatengine.Engine, opts ...FallbackOption) *FallbackHandler {
	h := &FallbackHandler{
		chain: []FallbackStrategy{
			&ruleEngineStrategy{engine: ruleEngine},
		},
		defaultNote: DefaultFallbackNote,
		policies:    make(map[FallbackReason]ReasonPolicy),
		logger:      zerolog.Nop(),
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// ---------------------------------------------------------------------------
// Handle — primary entry-point
// ---------------------------------------------------------------------------

// Handle attempts to produce a response through the fallback chain.
// The reason parameter controls per-reason policy and is stamped into
// the response metadata for downstream analytics.
func (h *FallbackHandler) Handle(
	ctx context.Context,
	conversationID *uuid.UUID,
	tenantID, userID uuid.UUID,
	message string,
	reason FallbackReason,
) (*chatdto.ChatResponse, error) {
	start := time.Now()
	h.totalFallbacks.Add(1)

	log := h.logger.With().
		Str("reason", string(reason)).
		Str("tenant_id", tenantID.String()).
		Logger()

	// --- Check reason policy --------------------------------------------
	policy := h.policyFor(reason)
	if policy.Disabled {
		log.Warn().Msg("fallback disabled by policy")
		return nil, fmt.Errorf("%w: reason=%s", ErrFallbackDisabled, reason)
	}

	// --- Apply per-reason timeout if configured -------------------------
	if policy.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, policy.Timeout)
		defer cancel()
	}

	// --- Walk the strategy chain ----------------------------------------
	var (
		resp     *chatdto.ChatResponse
		lastErr  error
		usedName string
	)

	for _, strategy := range h.chain {
		// Honour context cancellation between strategies.
		if ctx.Err() != nil {
			lastErr = ctx.Err()
			break
		}

		candidate, err := strategy.Handle(ctx, conversationID, tenantID, userID, message)
		if err != nil {
			log.Warn().Err(err).
				Str("strategy", strategy.Name()).
				Msg("fallback strategy failed, trying next")
			lastErr = err
			continue
		}

		if candidate != nil {
			resp = candidate
			usedName = strategy.Name()
			break
		}
	}

	if resp == nil {
		h.totalExhausted.Add(1)
		safeInc(h.metrics.FallbackTotal, "exhausted")
		log.Error().Err(lastErr).Msg("all fallback strategies exhausted")
		return nil, fmt.Errorf("%w: last error: %v", ErrFallbackExhausted, lastErr)
	}

	// --- Decorate the response ------------------------------------------
	h.decorate(resp, reason, policy)

	// --- Metrics / logging ----------------------------------------------
	latency := time.Since(start)
	safeInc(h.metrics.FallbackTotal, string(reason))

	log.Info().
		Str("strategy_used", usedName).
		Dur("latency", latency).
		Msg("fallback response served")

	return resp, nil
}

// ---------------------------------------------------------------------------
// HandleLegacy — backwards-compatible shim accepting a plain string reason.
// Drop once all callers migrate to typed FallbackReason.
// ---------------------------------------------------------------------------

func (h *FallbackHandler) HandleLegacy(
	ctx context.Context,
	conversationID *uuid.UUID,
	tenantID, userID uuid.UUID,
	message string,
	reason string,
) (*chatdto.ChatResponse, error) {
	return h.Handle(ctx, conversationID, tenantID, userID, message, FallbackReason(reason))
}

// ---------------------------------------------------------------------------
// Response decoration
// ---------------------------------------------------------------------------

// decorate stamps fallback metadata and prepends the degradation notice.
func (h *FallbackHandler) decorate(
	resp *chatdto.ChatResponse, reason FallbackReason, policy ReasonPolicy,
) {
	// --- Engine label ---
	resp.Engine = fallbackEngine

	if resp.Meta == nil {
		resp.Meta = &chatdto.ResponseMeta{}
	}
	resp.Meta.Engine = fallbackEngine
	resp.Meta.RoutingReason = string(reason)
	resp.Meta.Grounding = "passed"

	// --- Degradation note ---
	if policy.SuppressNote {
		return
	}

	note := h.resolveNote(policy)
	if note != "" && !strings.Contains(resp.Response.Text, note) {
		resp.Response.Text = note + "\n\n" + resp.Response.Text
	}
}

// resolveNote returns the note to prepend, checking policy override first.
func (h *FallbackHandler) resolveNote(policy ReasonPolicy) string {
	if policy.Note != "" {
		return policy.Note
	}
	return h.defaultNote
}

// ---------------------------------------------------------------------------
// Policy resolution
// ---------------------------------------------------------------------------

func (h *FallbackHandler) policyFor(reason FallbackReason) ReasonPolicy {
	if p, ok := h.policies[reason]; ok {
		return p
	}
	return ReasonPolicy{} // zero-value = all defaults
}

// ---------------------------------------------------------------------------
// Observability accessors
// ---------------------------------------------------------------------------

// Stats returns point-in-time counters for health dashboards or tests.
type FallbackStats struct {
	TotalFallbacks int64
	TotalExhausted int64
}

func (h *FallbackHandler) Stats() FallbackStats {
	return FallbackStats{
		TotalFallbacks: h.totalFallbacks.Load(),
		TotalExhausted: h.totalExhausted.Load(),
	}
}
