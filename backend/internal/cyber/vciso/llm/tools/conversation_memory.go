package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type ConversationMemoryTool struct {
	deps *chattools.Dependencies
}

func NewConversationMemoryTool(deps *chattools.Dependencies) *ConversationMemoryTool {
	return &ConversationMemoryTool{deps: deps}
}

func (t *ConversationMemoryTool) Name() string { return "conversation_memory_search" }
func (t *ConversationMemoryTool) Description() string {
	return "Search previous vCISO conversations for a specific topic or entity."
}
func (t *ConversationMemoryTool) RequiredPermissions() []string { return []string{"cyber:read"} }
func (t *ConversationMemoryTool) IsDestructive() bool           { return false }
func (t *ConversationMemoryTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"search_query": stringProp("What to search for"),
		"time_range":   stringProp(timeRangeDescription()),
		"entity_type":  enumString("alert", "asset", "vulnerability", "user", "any"),
	}, "search_query")
}

func (t *ConversationMemoryTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, args map[string]any) (*chattools.ToolResult, error) {
	if t.deps == nil || t.deps.CyberDB == nil {
		return nil, fmt.Errorf("cyber database is unavailable")
	}
	searchQuery := strings.TrimSpace(stringArg(args, "search_query"))
	if searchQuery == "" {
		return nil, fmt.Errorf("search_query is required")
	}
	start, _ := normalizeTimeRange(stringArg(args, "time_range"))
	rows, err := t.deps.CyberDB.Query(ctx, `
		SELECT c.id, COALESCE(c.title, 'Conversation'), m.content, m.created_at
		FROM vciso_messages m
		JOIN vciso_conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = $1
		  AND c.user_id = $2
		  AND m.created_at >= $3
		  AND m.content ILIKE '%' || $4 || '%'
		ORDER BY m.created_at DESC
		LIMIT 5`,
		tenantID, userID, start, searchQuery,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]any, 0, 5)
	entities := make([]chatmodel.EntityReference, 0, 5)
	index := 0
	for rows.Next() {
		var conversationID uuid.UUID
		var title string
		var content string
		var createdAt time.Time
		if err := rows.Scan(&conversationID, &title, &content, &createdAt); err != nil {
			return nil, err
		}
		items = append(items, map[string]any{
			"conversation_id": conversationID,
			"title":           title,
			"excerpt":         truncateSnippet(content, 180),
			"created_at":      createdAt,
		})
		entities = append(entities, entity("conversation", conversationID.String(), title, index))
		index++
	}
	return listResult(
		fmt.Sprintf("Found %d relevant conversation snippets.", len(items)),
		"list",
		map[string]any{"items": items},
		nil,
		entities,
	), nil
}

func truncateSnippet(value string, max int) string {
	value = strings.TrimSpace(value)
	if len(value) <= max {
		return value
	}
	return strings.TrimSpace(value[:max]) + "..."
}
