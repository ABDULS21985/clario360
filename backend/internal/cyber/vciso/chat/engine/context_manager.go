package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

// ---------------------------------------------------------------------------
// ClarificationRequest — returned when entity resolution fails
// ---------------------------------------------------------------------------

// ClarificationRequest signals that the engine needs more information
// from the user before it can proceed.
type ClarificationRequest struct {
	EntityType string // the entity key that is missing (e.g. "alert_id")
	Action     string // the action that requires the entity (optional)
	Message    string // user-facing prompt
}

// ---------------------------------------------------------------------------
// Configuration
// ---------------------------------------------------------------------------

const (
	DefaultIdleTimeout    = 30 * time.Minute
	DefaultMaxTurns       = 10
	DefaultMaxFilters     = 20
	DefaultClarifyMessage = "Which %s would you like me to use? You can provide an ID, name, or say 'the first one'."
)

// ContextManagerOption applies a configuration change.
type ContextManagerOption func(*ContextManager)

// WithIdleTimeout overrides the session idle timeout.
func WithIdleTimeout(d time.Duration) ContextManagerOption {
	return func(cm *ContextManager) {
		if d > 0 {
			cm.idleTimeout = d
		}
	}
}

// WithMaxTurns sets the sliding window size for conversation turns.
func WithMaxTurns(n int) ContextManagerOption {
	return func(cm *ContextManager) {
		if n > 0 {
			cm.maxTurns = n
		}
	}
}

// WithContextLogger injects a structured logger.
func WithContextLogger(l zerolog.Logger) ContextManagerOption {
	return func(cm *ContextManager) { cm.logger = l }
}

// WithClock overrides the time source, primarily for tests.
func WithClock(now func() time.Time) ContextManagerOption {
	return func(cm *ContextManager) {
		if now != nil {
			cm.now = now
		}
	}
}

// WithEntityResolvers replaces the built-in entity resolution chain.
func WithEntityResolvers(resolvers ...EntityResolver) ContextManagerOption {
	return func(cm *ContextManager) {
		if len(resolvers) > 0 {
			cm.entityResolvers = resolvers
		}
	}
}

// WithFilterRules replaces the built-in filter carry-over rules.
func WithFilterRules(rules ...FilterRule) ContextManagerOption {
	return func(cm *ContextManager) {
		if len(rules) > 0 {
			cm.filterRules = rules
		}
	}
}

// ---------------------------------------------------------------------------
// EntityResolver — pluggable resolution strategy
// ---------------------------------------------------------------------------

// EntityResolver attempts to resolve a missing entity from available context.
// Implementations are tried in order; the first to return a non-empty value wins.
type EntityResolver interface {
	Name() string
	Resolve(message string, entityType string, entities []chatmodel.EntityReference) string
}

// ---------------------------------------------------------------------------
// Built-in resolvers
// ---------------------------------------------------------------------------

// ordinalResolver handles "the first one", "#2", "the last one", etc.
type ordinalResolver struct{}

func (r *ordinalResolver) Name() string { return "ordinal" }

func (r *ordinalResolver) Resolve(message, entityType string, entities []chatmodel.EntityReference) string {
	filtered := filterByType(entities, entityReferenceType(entityType))
	if len(filtered) == 0 {
		return ""
	}

	lower := normalizeMessage(message)

	// Ordinal patterns ordered by specificity (most specific first).
	ordinals := []struct {
		patterns []string
		index    int // -1 = last
	}{
		{[]string{"the first one", " first", "#1", "number 1", "top one"}, 0},
		{[]string{"the second one", " second", "#2", "number 2"}, 1},
		{[]string{"the third one", " third", "#3", "number 3"}, 2},
		{[]string{"the fourth one", " fourth", "#4"}, 3},
		{[]string{"the fifth one", " fifth", "#5"}, 4},
		{[]string{"the last one", " bottom one", " last"}, -1},
	}

	for _, ord := range ordinals {
		if !containsAny(lower, ord.patterns...) {
			continue
		}
		idx := ord.index
		if idx == -1 {
			idx = len(filtered) - 1
		}
		if idx < len(filtered) {
			return filtered[idx].ID
		}
		return ""
	}

	return ""
}

// anaphoricResolver handles "it", "that", "this one", etc. — resolves
// to the most recent entity of the matching type.
type anaphoricResolver struct{}

func (r *anaphoricResolver) Name() string { return "anaphoric" }

func (r *anaphoricResolver) Resolve(message, entityType string, entities []chatmodel.EntityReference) string {
	filtered := filterByType(entities, entityReferenceType(entityType))
	if len(filtered) == 0 {
		return ""
	}

	lower := normalizeMessage(message)
	if containsAny(lower, "it", "that", "this", "that one", "this one", "the same") {
		return filtered[0].ID
	}
	return ""
}

// nameMatchResolver attempts to match an entity by name substring.
type nameMatchResolver struct{}

func (r *nameMatchResolver) Name() string { return "name_match" }

func (r *nameMatchResolver) Resolve(message, entityType string, entities []chatmodel.EntityReference) string {
	filtered := filterByType(entities, entityReferenceType(entityType))
	if len(filtered) == 0 {
		return ""
	}

	lower := normalizeMessage(message)
	for _, e := range filtered {
		if e.Name != "" && strings.Contains(lower, strings.ToLower(e.Name)) {
			return e.ID
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// FilterRule — pluggable filter carry-over logic
// ---------------------------------------------------------------------------

// FilterRule defines how a single filter key is carried over between turns.
type FilterRule interface {
	// Key returns the filter key this rule manages (e.g. "severity").
	Key() string

	// Apply updates extracted based on the current message and prior filters.
	// It may modify conversation.ActiveFilters as a side-effect.
	Apply(message string, extracted map[string]string, conversation *chatmodel.ConversationContext)
}

// ---------------------------------------------------------------------------
// Built-in filter rules
// ---------------------------------------------------------------------------

// simpleCarryoverRule carries a filter forward from the previous turn
// when the current message doesn't specify it.  New values overwrite old.
type simpleCarryoverRule struct {
	key string
}

func (r *simpleCarryoverRule) Key() string { return r.key }

func (r *simpleCarryoverRule) Apply(_ string, extracted map[string]string, conv *chatmodel.ConversationContext) {
	if conv == nil {
		return
	}
	if value, ok := extracted[r.key]; ok {
		conv.ActiveFilters[r.key] = value
	} else if value, ok := conv.ActiveFilters[r.key]; ok {
		extracted[r.key] = value
	}
}

// severityFilterRule handles severity with additive semantics ("and high",
// "also critical") and reset semantics ("all", "any", "everything").
type severityFilterRule struct{}

func (r *severityFilterRule) Key() string { return "severity" }

func (r *severityFilterRule) Apply(message string, extracted map[string]string, conv *chatmodel.ConversationContext) {
	if conv == nil {
		return
	}

	lower := normalizeMessage(message)

	// Reset semantics.
	if containsAny(lower, "all ", "any ", "everything") {
		delete(conv.ActiveFilters, "severity")
		delete(extracted, "severity")
		return
	}

	value, hasNew := extracted["severity"]
	if hasNew {
		// Additive: "and high", "also critical".
		if containsAny(lower, "and ", "also ") {
			extracted["severity"] = appendCSV(conv.ActiveFilters["severity"], value)
		}
		conv.ActiveFilters["severity"] = extracted["severity"]
	} else if prev, ok := conv.ActiveFilters["severity"]; ok {
		extracted["severity"] = prev
	}
}

// timeRangeFilterRule carries start_time and end_time as a pair.
type timeRangeFilterRule struct{}

func (r *timeRangeFilterRule) Key() string { return "start_time" }

func (r *timeRangeFilterRule) Apply(_ string, extracted map[string]string, conv *chatmodel.ConversationContext) {
	if conv == nil {
		return
	}

	if value, ok := extracted["start_time"]; ok {
		conv.ActiveFilters["start_time"] = value
		conv.ActiveFilters["end_time"] = extracted["end_time"]
	} else if value, ok := conv.ActiveFilters["start_time"]; ok {
		extracted["start_time"] = value
		if end := conv.ActiveFilters["end_time"]; end != "" {
			extracted["end_time"] = end
		}
	}
}

// ---------------------------------------------------------------------------
// ContextManager
// ---------------------------------------------------------------------------

// ContextManager owns conversation-scoped state: turn windowing, entity
// resolution, filter carry-over, follow-up intent inference, and session
// expiration.
//
// All mutating methods operate on *ConversationContext passed by pointer.
// The manager itself is stateless and safe for concurrent use.
type ContextManager struct {
	idleTimeout     time.Duration
	maxTurns        int
	now             func() time.Time
	logger          zerolog.Logger
	entityResolvers []EntityResolver
	filterRules     []FilterRule
}

func NewContextManager(opts ...ContextManagerOption) *ContextManager {
	cm := &ContextManager{
		idleTimeout: DefaultIdleTimeout,
		maxTurns:    DefaultMaxTurns,
		now:         func() time.Time { return time.Now().UTC() },
		logger:      zerolog.Nop(),
	}

	for _, opt := range opts {
		opt(cm)
	}

	// Default resolver chain if none injected.
	if len(cm.entityResolvers) == 0 {
		cm.entityResolvers = []EntityResolver{
			&ordinalResolver{},
			&nameMatchResolver{},
			&anaphoricResolver{},
		}
	}

	// Default filter rules if none injected.
	if len(cm.filterRules) == 0 {
		cm.filterRules = []FilterRule{
			&severityFilterRule{},
			&simpleCarryoverRule{key: "status"},
			&timeRangeFilterRule{},
		}
	}

	return cm
}

// ===========================================================================
// Context lifecycle
// ===========================================================================

// NewContext initialises a fresh conversation context.
func (cm *ContextManager) NewContext(conversationID, userID, tenantID uuid.UUID) chatmodel.ConversationContext {
	now := cm.now()
	return chatmodel.ConversationContext{
		ConversationID: conversationID,
		UserID:         userID,
		TenantID:       tenantID,
		Turns:          []chatmodel.Turn{},
		LastEntities:   []chatmodel.EntityReference{},
		ActiveFilters:  map[string]string{},
		StartedAt:      now,
		LastActivityAt: now,
		IdleTimeoutMin: int(cm.idleTimeout / time.Minute),
	}
}

// IsExpired returns true if the context has been idle beyond the timeout.
func (cm *ContextManager) IsExpired(ctx chatmodel.ConversationContext) bool {
	last := ctx.LastActivityAt
	if last.IsZero() {
		last = ctx.StartedAt
	}
	return cm.now().Sub(last) > cm.idleTimeout
}

// ===========================================================================
// Turn management
// ===========================================================================

// AddTurn appends a turn to the conversation context, maintaining a
// sliding window of the most recent maxTurns entries.
func (cm *ContextManager) AddTurn(ctx *chatmodel.ConversationContext, turn chatmodel.Turn) {
	if ctx == nil {
		return
	}

	ctx.Turns = append(ctx.Turns, turn)

	// Trim to sliding window.  Reallocate to release memory from old turns.
	if len(ctx.Turns) > cm.maxTurns {
		trimmed := make([]chatmodel.Turn, cm.maxTurns)
		copy(trimmed, ctx.Turns[len(ctx.Turns)-cm.maxTurns:])
		ctx.Turns = trimmed
	}

	ctx.LastActivityAt = turn.At
}

// ===========================================================================
// Entity resolution
// ===========================================================================

// ResolveEntities attempts to fill a required entity from extracted params,
// the resolver chain, or asks the user for clarification.
func (cm *ContextManager) ResolveEntities(
	message, intent string,
	extracted map[string]string,
	conversation *chatmodel.ConversationContext,
	requiredEntity string,
) (map[string]string, *ClarificationRequest) {
	if extracted == nil {
		extracted = map[string]string{}
	}

	// Already present — nothing to resolve.
	if requiredEntity == "" || extracted[requiredEntity] != "" {
		return extracted, nil
	}

	// No conversation context — can't resolve, ask the user.
	if conversation == nil || len(conversation.LastEntities) == 0 {
		return extracted, cm.clarify(requiredEntity)
	}

	// Walk the resolver chain.
	for _, resolver := range cm.entityResolvers {
		resolved := resolver.Resolve(message, requiredEntity, conversation.LastEntities)
		if resolved != "" {
			extracted[requiredEntity] = resolved
			cm.logger.Debug().
				Str("entity", requiredEntity).
				Str("resolver", resolver.Name()).
				Str("resolved_id", resolved).
				Msg("entity resolved from context")
			return extracted, nil
		}
	}

	// Chain exhausted — ask the user.
	return extracted, cm.clarify(requiredEntity)
}

func (cm *ContextManager) clarify(entityType string) *ClarificationRequest {
	return &ClarificationRequest{
		EntityType: entityType,
		Message:    fmt.Sprintf(DefaultClarifyMessage, humanEntity(entityType)),
	}
}

// ===========================================================================
// Filter carry-over
// ===========================================================================

// ApplyFilterCarryover runs each filter rule against the current message
// and extracted parameters, carrying forward or resetting filters from
// prior turns.
func (cm *ContextManager) ApplyFilterCarryover(
	message string,
	extracted map[string]string,
	conversation *chatmodel.ConversationContext,
) map[string]string {
	if extracted == nil {
		extracted = map[string]string{}
	}
	if conversation == nil {
		return extracted
	}
	if conversation.ActiveFilters == nil {
		conversation.ActiveFilters = map[string]string{}
	}

	for _, rule := range cm.filterRules {
		rule.Apply(message, extracted, conversation)
	}

	// Safety cap: prevent unbounded filter accumulation.
	if len(conversation.ActiveFilters) > DefaultMaxFilters {
		cm.logger.Warn().
			Int("count", len(conversation.ActiveFilters)).
			Msg("active filters exceed cap, trimming oldest")
		cm.trimFilters(conversation)
	}

	return extracted
}

func (cm *ContextManager) trimFilters(conv *chatmodel.ConversationContext) {
	// Keep only keys managed by registered rules.
	managed := make(map[string]struct{}, len(cm.filterRules))
	for _, r := range cm.filterRules {
		managed[r.Key()] = struct{}{}
	}
	// Also keep end_time if start_time is managed (time range pair).
	if _, ok := managed["start_time"]; ok {
		managed["end_time"] = struct{}{}
	}

	for k := range conv.ActiveFilters {
		if _, ok := managed[k]; !ok {
			delete(conv.ActiveFilters, k)
		}
	}
}

// ===========================================================================
// Follow-up intent inference
// ===========================================================================

// InferFollowUpIntent examines the message and conversation context to
// determine if the user is following up on a prior result (e.g. "investigate
// that one" after an alert list).  Returns "" if no follow-up is detected.
func (cm *ContextManager) InferFollowUpIntent(
	message string,
	conversation *chatmodel.ConversationContext,
) string {
	if conversation == nil {
		return ""
	}

	lower := normalizeMessage(message)

	// --- Deictic reference + entity type → specific follow-up intent ---
	if hasDeictic(lower) && len(conversation.LastEntities) > 0 {
		if intent := deicticEntityIntent(lower, conversation.LastEntities[0].Type); intent != "" {
			return intent
		}
	}

	// --- Verb-based follow-up from last turn's intent ---
	if len(conversation.Turns) > 0 {
		lastTurn := conversation.Turns[len(conversation.Turns)-1]
		if intent := verbBasedFollowUp(lower, lastTurn.Intent); intent != "" {
			return intent
		}
	}

	return ""
}

// hasDeictic checks for deictic/demonstrative references.
func hasDeictic(lower string) bool {
	return containsAny(lower,
		"the first one", "the second one", "the third one", "the last one",
		"that one", "this one", "the same", "it", "that", "this",
		"#1", "#2", "#3", "#4", "#5",
	)
}

// deicticEntityIntent maps a deictic reference + entity type to an intent.
func deicticEntityIntent(lower, entityType string) string {
	switch entityType {
	case "alert":
		switch {
		case containsAny(lower, "investigate", "deep dive", "analyze", "analyse", "look into", "dig into"):
			return "investigation_query"
		case containsAny(lower, "remediate", "fix", "contain", "isolate", "patch", "block", "mitigate"):
			return "remediation_query"
		default:
			return "alert_detail"
		}
	case "asset":
		switch {
		case containsAny(lower, "vulnerabilit", "vuln", "scan"):
			return "vulnerability_query"
		default:
			return "asset_lookup"
		}
	case "user":
		return "user_detail"
	}
	return ""
}

// verbBasedFollowUp matches action verbs against the prior turn's intent.
func verbBasedFollowUp(lower, lastIntent string) string {
	switch {
	case containsAny(lower, "details", "tell me more", "about it", "what happened", "more info"):
		return "alert_detail"
	case containsAny(lower, "investigate", "deep dive", "analyze", "analyse"):
		return "investigation_query"
	case containsAny(lower, "remediate", "fix", "contain", "isolate", "patch", "mitigate"):
		return "remediation_query"
	case containsAny(lower, "and high", "and critical", "also high", "also critical",
		"this week", "today", "last week", "all alerts", "past month"):
		if lastIntent == "alert_query" {
			return "alert_query"
		}
	}
	return ""
}

// ===========================================================================
// Pure helpers
// ===========================================================================

func humanEntity(entity string) string {
	names := map[string]string{
		"alert_id":   "alert",
		"asset_name": "asset",
		"asset_ip":   "asset",
		"user_id":    "user",
		"policy_id":  "policy",
	}
	if name, ok := names[entity]; ok {
		return name
	}
	return strings.ReplaceAll(entity, "_", " ")
}

func entityReferenceType(entity string) string {
	types := map[string]string{
		"alert_id":   "alert",
		"asset_name": "asset",
		"asset_ip":   "asset",
		"user_id":    "user",
		"policy_id":  "policy",
	}
	if t, ok := types[entity]; ok {
		return t
	}
	return entity
}

func filterByType(entities []chatmodel.EntityReference, targetType string) []chatmodel.EntityReference {
	out := make([]chatmodel.EntityReference, 0, len(entities))
	for _, e := range entities {
		if e.Type == targetType {
			out = append(out, e)
		}
	}
	return out
}

func containsAny(value string, options ...string) bool {
	for _, opt := range options {
		if strings.Contains(value, opt) {
			return true
		}
	}
	return false
}

func appendCSV(existing, next string) string {
	seen := make(map[string]struct{})
	var out []string
	for _, raw := range strings.Split(existing+","+next, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	return strings.Join(out, ",")
}
