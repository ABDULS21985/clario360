package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	chattools "github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

const (
	defaultMemorySearchLimit  = 10
	maxMemorySearchLimit      = 50
	defaultMemorySearchOffset = 0
	defaultMemoryLookbackDays = 90
)

type ConversationMemoryTool struct {
	deps *chattools.Dependencies
}

func NewConversationMemoryTool(deps *chattools.Dependencies) *ConversationMemoryTool {
	return &ConversationMemoryTool{deps: deps}
}

func (t *ConversationMemoryTool) Name() string {
	return "conversation_memory_search"
}

func (t *ConversationMemoryTool) Description() string {
	return "Search previous vCISO conversations using keyword, optional filters, and ranked retrieval."
}

func (t *ConversationMemoryTool) RequiredPermissions() []string {
	return []string{"cyber:read"}
}

func (t *ConversationMemoryTool) IsDestructive() bool {
	return false
}

func (t *ConversationMemoryTool) Schema() map[string]any {
	return requiredSchema(map[string]any{
		"search_query":           stringProp("What to search for in prior conversations"),
		"time_range":             stringProp(timeRangeDescription()),
		"entity_type":            enumString("alert", "asset", "vulnerability", "user", "conversation", "any"),
		"conversation_id":        stringProp("Optional conversation UUID to scope the search"),
		"limit":                  numberProp("Maximum number of results to return"),
		"offset":                 numberProp("Pagination offset"),
		"include_messages":       boolProp("Whether to include raw matching message snippets"),
		"dedupe_by_conversation": boolProp("If true, return only the best hit per conversation"),
	}, "search_query")
}

func (t *ConversationMemoryTool) Execute(
	ctx context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
	args map[string]any,
) (*chattools.ToolResult, error) {
	if t == nil || t.deps == nil || t.deps.CyberDB == nil {
		return nil, errors.New("conversation memory search is unavailable: cyber database dependency is missing")
	}

	req, err := parseConversationMemoryRequest(args)
	if err != nil {
		return nil, err
	}

	start, end, err := resolveTimeWindow(req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("invalid time_range: %w", err)
	}

	if end.Before(start) {
		return nil, fmt.Errorf("invalid time_range: end time is before start time")
	}

	if req.ConversationID != nil && *req.ConversationID == uuid.Nil {
		return nil, fmt.Errorf("conversation_id cannot be empty")
	}

	rows, err := t.queryMemory(ctx, tenantID, userID, req, start, end)
	if err != nil {
		return nil, fmt.Errorf("query conversation memory: %w", err)
	}
	defer rows.Close()

	results := make([]memorySearchHit, 0, req.Limit*2)
	for rows.Next() {
		var hit memorySearchHit
		if err := rows.Scan(
			&hit.ConversationID,
			&hit.ConversationTitle,
			&hit.MessageID,
			&hit.Content,
			&hit.CreatedAt,
			&hit.RankScore,
		); err != nil {
			return nil, fmt.Errorf("scan conversation memory result: %w", err)
		}

		if !matchesEntityType(req.EntityType, hit.ConversationTitle, hit.Content) {
			continue
		}

		results = append(results, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversation memory results: %w", err)
	}

	if req.DedupeByConversation {
		results = dedupeHitsByConversation(results)
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].RankScore == results[j].RankScore {
			return results[i].CreatedAt.After(results[j].CreatedAt)
		}
		return results[i].RankScore > results[j].RankScore
	})

	total := len(results)
	paged := paginateHits(results, req.Offset, req.Limit)

	items := make([]map[string]any, 0, len(paged))
	entities := make([]chatmodel.EntityReference, 0, len(paged)*2)

	for i, hit := range paged {
		item := map[string]any{
			"conversation_id": hit.ConversationID.String(),
			"title":           defaultString(hit.ConversationTitle, "Conversation"),
			"created_at":      hit.CreatedAt,
			"rank_score":      hit.RankScore,
		}

		if req.IncludeMessages {
			item["message_id"] = hit.MessageID.String()
			item["excerpt"] = buildSnippet(hit.Content, req.SearchQuery, 220)
		}

		items = append(items, item)

		entities = append(entities,
			entity("conversation", hit.ConversationID.String(), defaultString(hit.ConversationTitle, "Conversation"), i),
		)

		if req.IncludeMessages {
			entities = append(entities,
				entity("message", hit.MessageID.String(), buildSnippet(hit.Content, req.SearchQuery, 90), i),
			)
		}
	}

	summary := fmt.Sprintf(
		"Found %d matching conversation memory result(s); returning %d item(s).",
		total,
		len(paged),
	)

	return listResult(
		summary,
		"list",
		map[string]any{
			"items": items,
			"pagination": map[string]any{
				"offset":     req.Offset,
				"limit":      req.Limit,
				"returned":   len(paged),
				"total_hits": total,
				"has_more":   req.Offset+len(paged) < total,
			},
			"filters": map[string]any{
				"search_query":           req.SearchQuery,
				"time_range":             req.TimeRange,
				"entity_type":            req.EntityType,
				"conversation_id":        uuidPtrString(req.ConversationID),
				"include_messages":       req.IncludeMessages,
				"dedupe_by_conversation": req.DedupeByConversation,
			},
		},
		nil,
		entities,
	), nil
}

type conversationMemoryRequest struct {
	SearchQuery          string
	TimeRange            string
	EntityType           string
	ConversationID       *uuid.UUID
	Limit                int
	Offset               int
	IncludeMessages      bool
	DedupeByConversation bool
}

type memorySearchHit struct {
	ConversationID    uuid.UUID
	ConversationTitle string
	MessageID         uuid.UUID
	Content           string
	CreatedAt         time.Time
	RankScore         float64
}

func parseConversationMemoryRequest(args map[string]any) (*conversationMemoryRequest, error) {
	searchQuery := strings.TrimSpace(stringArg(args, "search_query"))
	if searchQuery == "" {
		return nil, fmt.Errorf("search_query is required")
	}

	limit := intArg(args, "limit", defaultMemorySearchLimit)
	if limit <= 0 {
		limit = defaultMemorySearchLimit
	}
	if limit > maxMemorySearchLimit {
		limit = maxMemorySearchLimit
	}

	offset := intArg(args, "offset", defaultMemorySearchOffset)
	if offset < 0 {
		offset = 0
	}

	entityType := strings.ToLower(strings.TrimSpace(stringArg(args, "entity_type")))
	if entityType == "" {
		entityType = "any"
	}
	if !isAllowedEntityType(entityType) {
		return nil, fmt.Errorf("unsupported entity_type %q", entityType)
	}

	var conversationID *uuid.UUID
	if raw := strings.TrimSpace(stringArg(args, "conversation_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("conversation_id must be a valid UUID")
		}
		conversationID = &parsed
	}

	return &conversationMemoryRequest{
		SearchQuery:          searchQuery,
		TimeRange:            strings.TrimSpace(stringArg(args, "time_range")),
		EntityType:           entityType,
		ConversationID:       conversationID,
		Limit:                limit,
		Offset:               offset,
		IncludeMessages:      boolArg(args, "include_messages", true),
		DedupeByConversation: boolArg(args, "dedupe_by_conversation", true),
	}, nil
}

func (t *ConversationMemoryTool) queryMemory(
	ctx context.Context,
	tenantID uuid.UUID,
	userID uuid.UUID,
	req *conversationMemoryRequest,
	start time.Time,
	end time.Time,
) (pgx.Rows, error) {
	// Hybrid SQL approach:
	// 1. full-text search for ranking
	// 2. fallback substring match to catch partial/non-token matches
	//
	// Recommended DB indexes:
	// - GIN(to_tsvector('simple', coalesce(content,'')))
	// - pg_trgm extension + GIN(content gin_trgm_ops)
	//
	baseSQL := `
		SELECT
			c.id,
			COALESCE(c.title, 'Conversation') AS title,
			m.id,
			m.content,
			m.created_at,
			(
				ts_rank_cd(
					to_tsvector('simple', COALESCE(m.content, '')),
					websearch_to_tsquery('simple', $5)
				)
				+
				CASE
					WHEN m.content ILIKE '%' || $6 || '%' THEN 0.25
					ELSE 0
				END
			) AS rank_score
		FROM vciso_messages m
		JOIN vciso_conversations c ON c.id = m.conversation_id
		WHERE c.tenant_id = $1
		  AND c.user_id = $2
		  AND m.created_at >= $3
		  AND m.created_at <= $4
		  AND (
				to_tsvector('simple', COALESCE(m.content, '')) @@ websearch_to_tsquery('simple', $5)
				OR m.content ILIKE '%' || $6 || '%'
		  )
	`

	args := []any{
		tenantID,
		userID,
		start,
		end,
		req.SearchQuery,
		req.SearchQuery,
	}

	if req.ConversationID != nil {
		baseSQL += " AND c.id = $7"
		args = append(args, *req.ConversationID)
	}

	baseSQL += `
		ORDER BY rank_score DESC, m.created_at DESC
		LIMIT $8
	`

	// Fetch a wider candidate window before in-memory dedupe + pagination.
	candidateLimit := req.Limit * 5
	if candidateLimit < 25 {
		candidateLimit = 25
	}
	if candidateLimit > 250 {
		candidateLimit = 250
	}
	args = append(args, candidateLimit)

	return t.deps.CyberDB.Query(ctx, baseSQL, args...)
}

func resolveTimeWindow(raw string) (time.Time, time.Time, error) {
	now := time.Now().UTC()

	if strings.TrimSpace(raw) == "" {
		return now.AddDate(0, 0, -defaultMemoryLookbackDays), now, nil
	}

	start, end, err := normalizeTimeRangeExpanded(raw, now)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	return start.UTC(), end.UTC(), nil
}

func normalizeTimeRangeExpanded(raw string, now time.Time) (time.Time, time.Time, error) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return now.AddDate(0, 0, -defaultMemoryLookbackDays), now, nil
	}

	switch value {
	case "24h", "last_24_hours", "last24hours":
		return now.Add(-24 * time.Hour), now, nil
	case "7d", "last_7_days", "last7days":
		return now.AddDate(0, 0, -7), now, nil
	case "30d", "last_30_days", "last30days":
		return now.AddDate(0, 0, -30), now, nil
	case "90d", "last_90_days", "last90days":
		return now.AddDate(0, 0, -90), now, nil
	case "180d", "last_180_days", "last180days":
		return now.AddDate(0, 0, -180), now, nil
	case "1y", "last_1_year", "last1year":
		return now.AddDate(-1, 0, 0), now, nil
	}

	// Support format: "2026-01-01,2026-03-01"
	if strings.Contains(value, ",") {
		parts := strings.SplitN(value, ",", 2)
		if len(parts) != 2 {
			return time.Time{}, time.Time{}, fmt.Errorf("expected comma-separated start and end dates")
		}

		start, err := time.Parse("2006-01-02", strings.TrimSpace(parts[0]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid start date")
		}
		end, err := time.Parse("2006-01-02", strings.TrimSpace(parts[1]))
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid end date")
		}

		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		return start, end, nil
	}

	// Support numeric days e.g. "14"
	if d, err := strconv.Atoi(value); err == nil && d > 0 {
		return now.AddDate(0, 0, -d), now, nil
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unsupported time_range format")
}

func isAllowedEntityType(value string) bool {
	switch value {
	case "alert", "asset", "vulnerability", "user", "conversation", "any":
		return true
	default:
		return false
	}
}

func matchesEntityType(entityType, title, content string) bool {
	if entityType == "" || entityType == "any" {
		return true
	}

	haystack := strings.ToLower(title + " " + content)

	switch entityType {
	case "alert":
		return containsAny(haystack, "alert", "incident", "detection", "ioc", "siem")
	case "asset":
		return containsAny(haystack, "asset", "host", "endpoint", "server", "device")
	case "vulnerability":
		return containsAny(haystack, "cve", "vulnerability", "cvss", "patch", "exploit")
	case "user":
		return containsAny(haystack, "user", "employee", "identity", "account", "login")
	case "conversation":
		return true
	default:
		return true
	}
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func dedupeHitsByConversation(hits []memorySearchHit) []memorySearchHit {
	best := make(map[uuid.UUID]memorySearchHit, len(hits))
	order := make([]uuid.UUID, 0, len(hits))

	for _, hit := range hits {
		existing, ok := best[hit.ConversationID]
		if !ok {
			best[hit.ConversationID] = hit
			order = append(order, hit.ConversationID)
			continue
		}

		if hit.RankScore > existing.RankScore ||
			(hit.RankScore == existing.RankScore && hit.CreatedAt.After(existing.CreatedAt)) {
			best[hit.ConversationID] = hit
		}
	}

	out := make([]memorySearchHit, 0, len(best))
	for _, id := range order {
		out = append(out, best[id])
	}
	return out
}

func paginateHits(hits []memorySearchHit, offset, limit int) []memorySearchHit {
	if offset >= len(hits) {
		return []memorySearchHit{}
	}
	end := offset + limit
	if end > len(hits) {
		end = len(hits)
	}
	return hits[offset:end]
}

func buildSnippet(content, query string, maxRunes int) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}

	lowerContent := strings.ToLower(content)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))

	if lowerQuery == "" {
		return truncateRunes(content, maxRunes)
	}

	idx := strings.Index(lowerContent, lowerQuery)
	if idx < 0 {
		return truncateRunes(content, maxRunes)
	}

	runes := []rune(content)
	queryRunes := []rune(query)

	// Convert byte index to rune index safely
	runeIndex := 0
	byteCount := 0
	for i, r := range runes {
		if byteCount >= idx {
			runeIndex = i
			break
		}
		byteCount += utf8.RuneLen(r)
	}

	start := runeIndex - 40
	if start < 0 {
		start = 0
	}
	end := runeIndex + len(queryRunes) + 120
	if end > len(runes) {
		end = len(runes)
	}

	snippet := strings.TrimSpace(string(runes[start:end]))
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(runes) {
		snippet += "..."
	}
	return snippet
}

func truncateRunes(value string, max int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return strings.TrimSpace(string(runes[:max])) + "..."
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uuidPtrString(v *uuid.UUID) any {
	if v == nil {
		return nil
	}
	return v.String()
}

func stringArg(args map[string]any, key string) string {
	raw, ok := args[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
	}
}

func intArg(args map[string]any, key string, fallback int) int {
	raw, ok := args[key]
	if !ok || raw == nil {
		return fallback
	}
	switch v := raw.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return fallback
		}
		return n
	default:
		return fallback
	}
}

// func boolArg(args map[string]any, key string, fallback bool) bool {
// 	raw, ok := args[key]
// 	if !ok || raw == nil {
// 		return fallback
// 	}
// 	switch v := raw.(type) {
// 	case bool:
// 		return v
// 	case string:
// 		switch strings.ToLower(strings.TrimSpace(v)) {
// 		case "true", "1", "yes":
// 			return true
// 		case "false", "0", "no":
// 			return false
// 		default:
// 			return fallback
// 		}
// 	default:
// 		return fallback
// 	}
// }
