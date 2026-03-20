package engine

import (
	"fmt"
	"strings"
	"sync"
	"time"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

// ---------------------------------------------------------------------------
// Data-type richness ranking (package-level, immutable after init)
// ---------------------------------------------------------------------------

// dataTypeRank defines the visual-richness hierarchy used to decide which
// tool result "wins" when multiple results compete for the response payload.
// Higher rank = richer presentation.  Unknown types default to 0.
var dataTypeRank = map[string]int{
	"text":          1,
	"list":          2,
	"kpi":           3,
	"table":         4,
	"chart":         5,
	"dashboard":     6,
	"investigation": 7,
}

// richer returns true when candidate outranks current in visual richness.
func richer(candidate, current string) bool {
	return dataTypeRank[strings.ToLower(candidate)] > dataTypeRank[strings.ToLower(current)]
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	DefaultMaxActions  = 4
	DefaultMaxEntities = 20
	DefaultEngine      = "llm"
	DefaultGrounding   = "passed"
	DefaultDataType    = "text"
)

// SynthesizerOption applies a configuration change to the synthesizer.
type SynthesizerOption func(*ResponseSynthesizer)

// WithMaxActions caps the number of suggested actions surfaced to the client.
func WithMaxActions(n int) SynthesizerOption {
	return func(s *ResponseSynthesizer) {
		if n > 0 {
			s.maxActions = n
		}
	}
}

// WithMaxEntities caps the number of entity references surfaced to the client.
func WithMaxEntities(n int) SynthesizerOption {
	return func(s *ResponseSynthesizer) {
		if n > 0 {
			s.maxEntities = n
		}
	}
}

// WithEngine overrides the engine label written into ResponseMeta.
func WithEngine(name string) SynthesizerOption {
	return func(s *ResponseSynthesizer) {
		if name != "" {
			s.engine = name
		}
	}
}

// WithLogger injects a structured logger.  When nil, logging is silently
// skipped (safe for tests / benchmarks).
type Logger interface {
	Debug(msg string, keysAndValues ...any)
	Warn(msg string, keysAndValues ...any)
}

func WithLogger(l Logger) SynthesizerOption {
	return func(s *ResponseSynthesizer) { s.log = l }
}

// ---------------------------------------------------------------------------
// ResponseSynthesizer
// ---------------------------------------------------------------------------

// ResponseSynthesizer merges raw LLM text, tool-call results, and grounding
// output into a single client-facing response payload + metadata.
//
// It is safe for concurrent use; all state is read-only after construction.
type ResponseSynthesizer struct {
	maxActions  int
	maxEntities int
	engine      string
	log         Logger
}

// NewResponseSynthesizer creates a synthesizer with sensible defaults.
// Pass Option functions to override behaviour.
func NewResponseSynthesizer(opts ...SynthesizerOption) *ResponseSynthesizer {
	s := &ResponseSynthesizer{
		maxActions:  DefaultMaxActions,
		maxEntities: DefaultMaxEntities,
		engine:      DefaultEngine,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// ---------------------------------------------------------------------------
// Primary API
// ---------------------------------------------------------------------------

// SynthesisInput bundles every signal the synthesizer needs.  Using a struct
// avoids a parameter list that grows every time a new signal is introduced.
type SynthesisInput struct {
	Text        string
	ToolResults []*llmmodel.ToolCallResult
	Grounding   *llmmodel.GroundingResult
}

// Synthesize merges all inputs into a response payload and metadata.
func (s *ResponseSynthesizer) Synthesize(in SynthesisInput) (chatmodel.ResponsePayload, *chatdto.ResponseMeta) {
	start := time.Now()

	payload := s.basePayload(in.Text)
	meta := s.baseMeta()

	s.applyGrounding(&payload, meta, in.Grounding)
	s.applyToolResults(&payload, meta, in.ToolResults)
	s.enforceEntityCap(&payload)

	meta.SynthesisLatencyMs = time.Since(start).Milliseconds()

	s.debug("synthesis complete",
		"data_type", payload.DataType,
		"actions", len(payload.Actions),
		"entities", len(payload.Entities),
		"tool_calls", meta.ToolCallsCount,
		"latency_ms", meta.SynthesisLatencyMs,
	)

	return payload, meta
}

// ---------------------------------------------------------------------------
// Backwards-compatible shim (drop once callers migrate to SynthesisInput)
// ---------------------------------------------------------------------------

// SynthesizeLegacy preserves the original call-site signature so existing
// callers keep compiling while you migrate them one at a time.
func (s *ResponseSynthesizer) SynthesizeLegacy(
	text string,
	toolResults []*llmmodel.ToolCallResult,
	grounding *llmmodel.GroundingResult,
) (chatmodel.ResponsePayload, *chatdto.ResponseMeta) {
	return s.Synthesize(SynthesisInput{
		Text:        text,
		ToolResults: toolResults,
		Grounding:   grounding,
	})
}

// ---------------------------------------------------------------------------
// Internal: base builders
// ---------------------------------------------------------------------------

func (s *ResponseSynthesizer) basePayload(text string) chatmodel.ResponsePayload {
	return chatmodel.ResponsePayload{
		Text:     text,
		DataType: DefaultDataType,
		Actions:  []chatmodel.SuggestedAction{},
		Entities: []chatmodel.EntityReference{},
	}
}

func (s *ResponseSynthesizer) baseMeta() *chatdto.ResponseMeta {
	return &chatdto.ResponseMeta{
		Grounding: DefaultGrounding,
		Engine:    s.engine,
	}
}

// ---------------------------------------------------------------------------
// Internal: grounding
// ---------------------------------------------------------------------------

func (s *ResponseSynthesizer) applyGrounding(
	payload *chatmodel.ResponsePayload,
	meta *chatdto.ResponseMeta,
	g *llmmodel.GroundingResult,
) {
	if g == nil || g.Status == "" {
		return
	}

	meta.Grounding = g.Status

	switch g.Status {
	case "corrected":
		if g.CorrectedResponse != "" {
			payload.Text = g.CorrectedResponse
			s.debug("grounding corrected response")
		} else {
			s.warn("grounding status=corrected but CorrectedResponse is empty")
		}
	case "failed":
		s.warn("grounding check failed", "detail", formatGroundingFailure(g))
	}
}

// ---------------------------------------------------------------------------
// Internal: tool results
// ---------------------------------------------------------------------------

func (s *ResponseSynthesizer) applyToolResults(
	payload *chatmodel.ResponsePayload,
	meta *chatdto.ResponseMeta,
	results []*llmmodel.ToolCallResult,
) {
	var (
		validCount  int
		entitySeen  = make(map[string]struct{})
		actionAccum = make([]chatmodel.SuggestedAction, 0, len(payload.Actions))
	)

	// Seed action accumulator with any pre-existing actions.
	actionSeen := make(map[string]struct{}, s.maxActions)
	for _, a := range payload.Actions {
		key := actionKey(a)
		actionSeen[key] = struct{}{}
		actionAccum = append(actionAccum, a)
	}

	for i, result := range results {
		if result == nil {
			s.debug("skipping nil tool result", "index", i)
			continue
		}
		validCount++

		// Promote data type if the tool produced something richer.
		if richer(result.DataType, payload.DataType) {
			payload.DataType = result.DataType
			payload.Data = result.Data
		}

		// Accumulate actions (deduplicated, capped).
		for _, a := range result.Actions {
			if len(actionAccum) >= s.maxActions {
				break
			}
			key := actionKey(a)
			if _, dup := actionSeen[key]; dup {
				continue
			}
			actionSeen[key] = struct{}{}
			actionAccum = append(actionAccum, a)
		}

		// Accumulate entities (deduplicated).
		for _, e := range result.Entities {
			key := entityKey(e)
			if _, dup := entitySeen[key]; dup {
				continue
			}
			entitySeen[key] = struct{}{}
			payload.Entities = append(payload.Entities, e)
		}
	}

	payload.Actions = actionAccum
	meta.ToolCallsCount = validCount
	meta.ReasoningSteps = reasoningSteps(validCount)
}

// enforceEntityCap trims entities to the configured ceiling and logs a
// warning when truncation occurs so upstream teams can tune extraction.
func (s *ResponseSynthesizer) enforceEntityCap(payload *chatmodel.ResponsePayload) {
	if len(payload.Entities) <= s.maxEntities {
		return
	}
	s.warn("entity count exceeds cap, truncating",
		"count", len(payload.Entities),
		"cap", s.maxEntities,
	)
	payload.Entities = payload.Entities[:s.maxEntities]
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// actionKey produces a deduplication key for a suggested action.
func actionKey(a chatmodel.SuggestedAction) string {
	return a.Type + "\x00" + a.Label
}

// entityKey produces a deduplication key for an entity reference.
// Adjust the fields if your EntityReference model uses different identifiers.
func entityKey(e chatmodel.EntityReference) string {
	return fmt.Sprintf("%s\x00%s\x00%s", e.Type, e.ID, e.Name)
}

// reasoningSteps maps tool-call count to a user-visible "reasoning steps"
// metric.  The heuristic can evolve independently of the rest of synthesis.
func reasoningSteps(toolCalls int) int {
	// Base step (the LLM itself always counts as one reasoning step)
	// plus one step per tool invocation.
	if toolCalls == 0 {
		return 1
	}
	return 1 + toolCalls
}

// ---------------------------------------------------------------------------
// Logging helpers (nil-safe)
// ---------------------------------------------------------------------------

func (s *ResponseSynthesizer) debug(msg string, kv ...any) {
	if s.log != nil {
		s.log.Debug(msg, kv...)
	}
}

func (s *ResponseSynthesizer) warn(msg string, kv ...any) {
	if s.log != nil {
		s.log.Warn(msg, kv...)
	}
}

// ---------------------------------------------------------------------------
// Concurrency-safe registry of custom data-type rankings
// ---------------------------------------------------------------------------

// DataTypeRegistry allows runtime registration of additional data types
// (e.g. from plugins) without recompiling the engine package.
type DataTypeRegistry struct {
	mu    sync.RWMutex
	ranks map[string]int
}

// NewDataTypeRegistry creates a registry pre-seeded with the default ranks.
func NewDataTypeRegistry() *DataTypeRegistry {
	base := make(map[string]int, len(dataTypeRank))
	for k, v := range dataTypeRank {
		base[k] = v
	}
	return &DataTypeRegistry{ranks: base}
}

// Register adds or overrides a data-type rank.  Not safe to call from
// multiple goroutines without the internal mutex — but the mutex is there.
func (r *DataTypeRegistry) Register(dataType string, rank int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.ranks[strings.ToLower(dataType)] = rank
}

// Richer is the registry-aware equivalent of the package-level richer().
func (r *DataTypeRegistry) Richer(candidate, current string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ranks[strings.ToLower(candidate)] > r.ranks[strings.ToLower(current)]
}
