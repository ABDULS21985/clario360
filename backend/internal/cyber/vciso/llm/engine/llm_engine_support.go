package engine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
	llmprovider "github.com/clario360/platform/internal/cyber/vciso/llm/provider"
	llmtools "github.com/clario360/platform/internal/cyber/vciso/llm/tools"
)

var (
	ErrRateLimited            = errors.New("llm rate limit exceeded")
	ErrConversationLoadFailed = errors.New("llm conversation load failed")
	ErrProviderResolveFailed  = errors.New("llm provider resolve failed")
	ErrContextCompileFailed   = errors.New("llm context compile failed")
	ErrPromptBuildFailed      = errors.New("llm prompt build failed")
	ErrProviderCallFailed     = errors.New("llm provider call failed")
	ErrPersistFailed          = errors.New("llm persist failed")
	ErrContextCancelled       = errors.New("llm context cancelled")
	ErrLLMUnavailable         = errors.New("llm engine unavailable")
)

type ProcessMessageInput struct {
	ConversationID *uuid.UUID
	TenantID       uuid.UUID
	UserID         uuid.UUID
	Message        string
	Hint           *chatmodel.ClassificationResult
	RoutingReason  string
}

type persistInput struct {
	conversation     *chatmodel.Conversation
	contextState     *chatmodel.ConversationContext
	originalMessage  string
	classification   *chatmodel.ClassificationResult
	entities         map[string]string
	toolResults      []*llmmodel.ToolCallResult
	payload          chatmodel.ResponsePayload
	sanitized        *SanitizedMessage
	grounding        *llmmodel.GroundingResult
	routingReason    string
	latency          time.Duration
	predictionLogID  *uuid.UUID
	promptTokens     int
	completionTokens int
	piiDetections    int
	reasoningTrace   []llmmodel.ReasoningStep
	prompt           *LLMPrompt
}

type processingState struct {
	input            ProcessMessageInput
	classification   *chatmodel.ClassificationResult
	startedAt        time.Time
	phases           *phaseTracker
	conversation     *chatmodel.Conversation
	contextState     chatmodel.ConversationContext
	isNew            bool
	sanitized        *SanitizedMessage
	provider         llmprovider.LLMProvider
	compiledCtx      *CompiledContext
	prompt           *LLMPrompt
	messages         []llmmodel.LLMMessage
	toolSchemas      []llmmodel.ToolSchema
	finalText        string
	filteredText     string
	toolResults      []*llmmodel.ToolCallResult
	reasoningTrace   []llmmodel.ReasoningStep
	grounding        *llmmodel.GroundingResult
	payload          chatmodel.ResponsePayload
	meta             *chatdto.ResponseMeta
	predictionLogID  *uuid.UUID
	promptTokens     int
	completionTokens int
	piiDetections    int
	toolLoopIters    int
}

func newProcessingState(input ProcessMessageInput) *processingState {
	return &processingState{
		input:          input,
		classification: hintOrUnknown(input.Hint),
		startedAt:      time.Now().UTC(),
		phases:         newPhaseTracker(),
		compiledCtx:    &CompiledContext{},
		messages:       []llmmodel.LLMMessage{},
		toolSchemas:    []llmmodel.ToolSchema{},
		toolResults:    []*llmmodel.ToolCallResult{},
		reasoningTrace: []llmmodel.ReasoningStep{},
	}
}

func (s *processingState) elapsed() time.Duration {
	if s == nil {
		return 0
	}
	return time.Since(s.startedAt)
}

func (s *processingState) totalTokens() int {
	if s == nil {
		return 0
	}
	return s.promptTokens + s.completionTokens
}

type phase string

const (
	PhaseRateLimit    phase = "rate_limit"
	PhaseConversation phase = "conversation"
	PhaseSanitize     phase = "sanitize"
	PhaseProvider     phase = "provider"
	PhaseContext      phase = "context"
	PhasePrompt       phase = "prompt"
	PhaseToolLoop     phase = "tool_loop"
	PhaseGrounding    phase = "grounding"
	PhasePII          phase = "pii_filter"
	PhaseSynthesis    phase = "synthesis"
	PhasePersist      phase = "persist"
)

type phaseTracker struct {
	durations map[phase]time.Duration
}

func newPhaseTracker() *phaseTracker {
	return &phaseTracker{durations: make(map[phase]time.Duration)}
}

func (t *phaseTracker) Track(name phase, fn func()) {
	if t == nil || fn == nil {
		return
	}
	start := time.Now()
	fn()
	t.durations[name] += time.Since(start)
}

func (t *phaseTracker) Map() map[phase]time.Duration {
	if t == nil {
		return nil
	}
	out := make(map[phase]time.Duration, len(t.durations))
	for key, value := range t.durations {
		out[key] = value
	}
	return out
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return ErrContextCancelled
	}
	return nil
}

func safeInc(counter *prometheus.CounterVec, labels ...string) {
	if counter == nil {
		return
	}
	counter.WithLabelValues(labels...).Inc()
}

func safeAdd(counter *prometheus.CounterVec, value float64, labels ...string) {
	if counter == nil {
		return
	}
	counter.WithLabelValues(labels...).Add(value)
}

func safeObserve(hist prometheus.Observer, value float64) {
	if hist == nil {
		return
	}
	hist.Observe(value)
}

func safeObserveVec(hist *prometheus.HistogramVec, value float64, labels ...string) {
	if hist == nil {
		return
	}
	hist.WithLabelValues(labels...).Observe(value)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func marshalPayload(data any) json.RawMessage {
	payload, _ := json.Marshal(data)
	return payload
}

func marshalToolMessage(result *llmmodel.ToolCallResult) string {
	payload, _ := json.Marshal(map[string]any{
		"tool_name": result.ToolName,
		"success":   result.Success,
		"summary":   result.Summary,
		"data":      result.Data,
		"error":     result.Error,
	})
	return string(payload)
}

func HashPrompt(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func DefaultPromptText() string {
	return defaultSystemPrompt
}

func (e *LLMEngine) ProviderManager() *llmprovider.Manager {
	if e == nil {
		return nil
	}
	return e.providerManager
}

func (e *LLMEngine) ToolRegistry() *llmtools.Registry {
	if e == nil {
		return nil
	}
	return e.toolRegistry
}
