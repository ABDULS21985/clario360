package engine

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type ClarificationRequest struct {
	EntityType string
	Action     string
	Message    string
}

type ContextManager struct {
	idleTimeout time.Duration
	now         func() time.Time
}

func NewContextManager(now func() time.Time, idleTimeout time.Duration) *ContextManager {
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	if idleTimeout <= 0 {
		idleTimeout = 30 * time.Minute
	}
	return &ContextManager{idleTimeout: idleTimeout, now: now}
}

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

func (cm *ContextManager) IsExpired(ctx chatmodel.ConversationContext) bool {
	last := ctx.LastActivityAt
	if last.IsZero() {
		last = ctx.StartedAt
	}
	return cm.now().Sub(last) > cm.idleTimeout
}

func (cm *ContextManager) AddTurn(ctx *chatmodel.ConversationContext, turn chatmodel.Turn) {
	if ctx == nil {
		return
	}
	ctx.Turns = append(ctx.Turns, turn)
	if len(ctx.Turns) > 10 {
		ctx.Turns = append([]chatmodel.Turn(nil), ctx.Turns[len(ctx.Turns)-10:]...)
	}
	ctx.LastActivityAt = turn.At
}

func (cm *ContextManager) ResolveEntities(message string, intent string, extracted map[string]string, conversation *chatmodel.ConversationContext, requiredEntity string) (map[string]string, *ClarificationRequest) {
	if extracted == nil {
		extracted = map[string]string{}
	}
	if requiredEntity == "" || extracted[requiredEntity] != "" {
		return extracted, nil
	}
	if conversation == nil {
		return extracted, &ClarificationRequest{
			EntityType: requiredEntity,
			Message:    fmt.Sprintf("Which %s would you like me to use? You can provide an ID, name, or say 'the first one'.", humanEntity(requiredEntity)),
		}
	}
	resolved := cm.resolveFromLastEntities(message, requiredEntity, conversation.LastEntities)
	if resolved != "" {
		extracted[requiredEntity] = resolved
		return extracted, nil
	}
	return extracted, &ClarificationRequest{
		EntityType: requiredEntity,
		Message:    fmt.Sprintf("Which %s would you like me to use? You can provide an ID, name, or say 'the first one'.", humanEntity(requiredEntity)),
	}
}

func (cm *ContextManager) ApplyFilterCarryover(message string, extracted map[string]string, conversation *chatmodel.ConversationContext) map[string]string {
	if extracted == nil {
		extracted = map[string]string{}
	}
	if conversation == nil {
		return extracted
	}
	if conversation.ActiveFilters == nil {
		conversation.ActiveFilters = map[string]string{}
	}
	lower := normalizeMessage(message)
	if strings.Contains(lower, "all ") || strings.Contains(lower, "any ") || strings.Contains(lower, "everything") {
		delete(conversation.ActiveFilters, "severity")
		delete(extracted, "severity")
	}
	if value, ok := extracted["severity"]; ok {
		if strings.Contains(lower, "and ") || strings.Contains(lower, "also ") {
			extracted["severity"] = appendCSV(conversation.ActiveFilters["severity"], value)
		}
		conversation.ActiveFilters["severity"] = extracted["severity"]
	} else if value, ok := conversation.ActiveFilters["severity"]; ok {
		extracted["severity"] = value
	}
	if value, ok := extracted["status"]; ok {
		conversation.ActiveFilters["status"] = value
	} else if value, ok := conversation.ActiveFilters["status"]; ok {
		extracted["status"] = value
	}
	if value, ok := extracted["start_time"]; ok {
		conversation.ActiveFilters["start_time"] = value
		conversation.ActiveFilters["end_time"] = extracted["end_time"]
	} else if value, ok := conversation.ActiveFilters["start_time"]; ok {
		extracted["start_time"] = value
		if endValue := conversation.ActiveFilters["end_time"]; endValue != "" {
			extracted["end_time"] = endValue
		}
	}
	return extracted
}

func (cm *ContextManager) resolveFromLastEntities(message, requiredEntity string, items []chatmodel.EntityReference) string {
	if len(items) == 0 {
		return ""
	}
	targetType := entityReferenceType(requiredEntity)
	filtered := make([]chatmodel.EntityReference, 0, len(items))
	for _, item := range items {
		if item.Type == targetType {
			filtered = append(filtered, item)
		}
	}
	if len(filtered) == 0 {
		return ""
	}
	lower := normalizeMessage(message)
	switch {
	case containsAny(lower, "the first one", " first", "#1", "number 1", "top one"):
		return filtered[0].ID
	case containsAny(lower, "the second one", " second", "#2", "number 2") && len(filtered) > 1:
		return filtered[1].ID
	case containsAny(lower, "the third one", " third", "#3") && len(filtered) > 2:
		return filtered[2].ID
	case containsAny(lower, "the last one", " bottom one", " last") && len(filtered) > 0:
		return filtered[len(filtered)-1].ID
	case containsAny(lower, "it", "that", "this", "that one", "the same"):
		return filtered[0].ID
	default:
		return ""
	}
}

func inferFollowUpIntent(message string, conversation *chatmodel.ConversationContext) string {
	if conversation == nil {
		return ""
	}
	lower := normalizeMessage(message)
	hasDeicticReference := containsAny(lower,
		"the first one", "the second one", "the third one", "the last one",
		"that one", "this one", "the same", "it", "that", "this", "#1", "#2", "#3",
	)
	if hasDeicticReference && len(conversation.LastEntities) > 0 {
		switch conversation.LastEntities[0].Type {
		case "alert":
			switch {
			case containsAny(lower, "investigate", "deep dive", "analyze", "look into", "dig into"):
				return "investigation_query"
			case containsAny(lower, "remediate", "fix", "contain", "isolate", "patch", "block"):
				return "remediation_query"
			default:
				return "alert_detail"
			}
		case "asset":
			return "asset_lookup"
		}
	}
	if len(conversation.Turns) == 0 {
		return ""
	}
	lastTurn := conversation.Turns[len(conversation.Turns)-1]
	switch {
	case containsAny(lower, "details", "tell me more", "about it", "what happened"):
		return "alert_detail"
	case containsAny(lower, "investigate", "deep dive", "analyze"):
		return "investigation_query"
	case containsAny(lower, "remediate", "fix", "contain", "isolate", "patch"):
		return "remediation_query"
	case containsAny(lower, "and high", "and critical", "also high", "also critical", "this week", "today", "last week", "all alerts"):
		if lastTurn.Intent == "alert_query" {
			return "alert_query"
		}
	}
	return ""
}

func humanEntity(entity string) string {
	switch entity {
	case "alert_id":
		return "alert"
	case "asset_name":
		return "asset"
	default:
		return entity
	}
}

func entityReferenceType(entity string) string {
	switch entity {
	case "alert_id":
		return "alert"
	case "asset_name", "asset_ip":
		return "asset"
	default:
		return entity
	}
}

func containsAny(value string, options ...string) bool {
	for _, option := range options {
		if strings.Contains(value, option) {
			return true
		}
	}
	return false
}

func appendCSV(existing, next string) string {
	seen := map[string]struct{}{}
	values := []string{}
	for _, raw := range strings.Split(existing+","+next, ",") {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if _, ok := seen[raw]; ok {
			continue
		}
		seen[raw] = struct{}{}
		values = append(values, raw)
	}
	return strings.Join(values, ",")
}
