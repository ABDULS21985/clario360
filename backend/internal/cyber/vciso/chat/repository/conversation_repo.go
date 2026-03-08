package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

var ErrNotFound = errors.New("not found")

type ConversationRepository struct {
	db     *pgxpool.Pool
	logger zerolog.Logger
}

func NewConversationRepository(db *pgxpool.Pool, logger zerolog.Logger) *ConversationRepository {
	return &ConversationRepository{
		db:     db,
		logger: logger.With().Str("component", "vciso-conversation-repo").Logger(),
	}
}

func (r *ConversationRepository) CreateConversation(ctx context.Context, conversation *chatmodel.Conversation) error {
	if conversation == nil {
		return fmt.Errorf("conversation is required")
	}
	payload, err := json.Marshal(conversation.LastContext)
	if err != nil {
		return fmt.Errorf("marshal conversation context: %w", err)
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO vciso_conversations (
			id, tenant_id, user_id, title, status, message_count, last_context, last_message_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING created_at, updated_at`,
		conversation.ID,
		conversation.TenantID,
		conversation.UserID,
		conversation.Title,
		conversation.Status,
		conversation.MessageCount,
		payload,
		conversation.LastMessageAt,
		conversation.CreatedAt,
		conversation.UpdatedAt,
	).Scan(&conversation.CreatedAt, &conversation.UpdatedAt)
}

func (r *ConversationRepository) GetConversation(ctx context.Context, tenantID, userID, conversationID uuid.UUID) (*chatmodel.Conversation, error) {
	row := r.db.QueryRow(ctx, `
		SELECT id, tenant_id, user_id, title, status, message_count, last_context, last_message_at, created_at, updated_at
		FROM vciso_conversations
		WHERE tenant_id = $1 AND user_id = $2 AND id = $3 AND status != 'deleted'`,
		tenantID, userID, conversationID,
	)
	return scanConversation(row)
}

func (r *ConversationRepository) ListConversations(ctx context.Context, tenantID, userID uuid.UUID, page, perPage int) ([]chatdto.ConversationListItem, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}
	offset := (page - 1) * perPage
	rows, err := r.db.Query(ctx, `
		SELECT id, title, status, message_count, last_message_at, created_at
		FROM vciso_conversations
		WHERE tenant_id = $1 AND user_id = $2 AND status != 'deleted'
		ORDER BY updated_at DESC
		LIMIT $3 OFFSET $4`,
		tenantID, userID, perPage, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	items := make([]chatdto.ConversationListItem, 0, perPage)
	for rows.Next() {
		var item chatdto.ConversationListItem
		if err := rows.Scan(&item.ID, &item.Title, &item.Status, &item.MessageCount, &item.LastMessageAt, &item.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	var total int
	if err := r.db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM vciso_conversations
		WHERE tenant_id = $1 AND user_id = $2 AND status != 'deleted'`,
		tenantID, userID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count conversations: %w", err)
	}
	return items, total, nil
}

func (r *ConversationRepository) SoftDeleteConversation(ctx context.Context, tenantID, userID, conversationID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE vciso_conversations
		SET status = 'deleted', updated_at = now()
		WHERE tenant_id = $1 AND user_id = $2 AND id = $3 AND status != 'deleted'`,
		tenantID, userID, conversationID,
	)
	if err != nil {
		return fmt.Errorf("delete conversation: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ConversationRepository) UpdateConversationState(ctx context.Context, conversationID, tenantID uuid.UUID, messageDelta int, contextState *chatmodel.ConversationContext) error {
	payload := []byte("{}")
	var err error
	if contextState != nil {
		payload, err = json.Marshal(contextState)
		if err != nil {
			return fmt.Errorf("marshal context: %w", err)
		}
	}
	tag, err := r.db.Exec(ctx, `
		UPDATE vciso_conversations
		SET message_count = message_count + $3,
		    last_context = $4,
		    last_message_at = now(),
		    updated_at = now()
		WHERE id = $1 AND tenant_id = $2 AND status != 'deleted'`,
		conversationID, tenantID, messageDelta, payload,
	)
	if err != nil {
		return fmt.Errorf("update conversation state: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ConversationRepository) CreateMessage(ctx context.Context, message *chatmodel.Message) error {
	if message == nil {
		return fmt.Errorf("message is required")
	}
	extracted, _ := json.Marshal(defaultMap(message.ExtractedEntities))
	toolParams, _ := json.Marshal(defaultMap(message.ToolParams))
	actions, _ := json.Marshal(defaultActions(message.SuggestedActions))
	entities, _ := json.Marshal(defaultEntities(message.EntityReferences))
	if len(message.ToolResult) == 0 {
		message.ToolResult = json.RawMessage(`null`)
	}
	return r.db.QueryRow(ctx, `
		INSERT INTO vciso_messages (
			id, conversation_id, tenant_id, role, content, intent, intent_confidence, match_method,
			matched_pattern, extracted_entities, tool_name, tool_params, tool_result, tool_latency_ms,
			tool_error, response_type, suggested_actions, entity_references, prediction_log_id, created_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,
			$9,$10,$11,$12,$13,$14,
			$15,$16,$17,$18,$19,$20
		)
		RETURNING created_at`,
		message.ID,
		message.ConversationID,
		message.TenantID,
		message.Role,
		message.Content,
		message.Intent,
		message.IntentConfidence,
		message.MatchMethod,
		message.MatchedPattern,
		extracted,
		message.ToolName,
		toolParams,
		message.ToolResult,
		message.ToolLatencyMS,
		message.ToolError,
		message.ResponseType,
		actions,
		entities,
		message.PredictionLogID,
		message.CreatedAt,
	).Scan(&message.CreatedAt)
}

func (r *ConversationRepository) ListMessages(ctx context.Context, tenantID, conversationID uuid.UUID) ([]chatmodel.Message, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, conversation_id, tenant_id, role, content, intent, intent_confidence, match_method,
		       matched_pattern, extracted_entities, tool_name, tool_params, tool_result, tool_latency_ms,
		       tool_error, response_type, suggested_actions, entity_references, prediction_log_id, created_at
		FROM vciso_messages
		WHERE tenant_id = $1 AND conversation_id = $2
		ORDER BY created_at ASC`,
		tenantID, conversationID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()
	items := make([]chatmodel.Message, 0, 32)
	for rows.Next() {
		item, err := scanMessage(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, *item)
	}
	return items, rows.Err()
}

func scanConversation(row interface{ Scan(...any) error }) (*chatmodel.Conversation, error) {
	var (
		item       chatmodel.Conversation
		contextRaw []byte
	)
	if err := row.Scan(
		&item.ID,
		&item.TenantID,
		&item.UserID,
		&item.Title,
		&item.Status,
		&item.MessageCount,
		&contextRaw,
		&item.LastMessageAt,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if len(contextRaw) > 0 {
		if err := json.Unmarshal(contextRaw, &item.LastContext); err != nil {
			return nil, fmt.Errorf("decode conversation context: %w", err)
		}
	}
	if item.LastContext.ActiveFilters == nil {
		item.LastContext.ActiveFilters = map[string]string{}
	}
	return &item, nil
}

func scanMessage(row interface{ Scan(...any) error }) (*chatmodel.Message, error) {
	var (
		item            chatmodel.Message
		extractedRaw    []byte
		toolParamsRaw   []byte
		actionsRaw      []byte
		entitiesRaw     []byte
		predictionLogID *uuid.UUID
	)
	if err := row.Scan(
		&item.ID,
		&item.ConversationID,
		&item.TenantID,
		&item.Role,
		&item.Content,
		&item.Intent,
		&item.IntentConfidence,
		&item.MatchMethod,
		&item.MatchedPattern,
		&extractedRaw,
		&item.ToolName,
		&toolParamsRaw,
		&item.ToolResult,
		&item.ToolLatencyMS,
		&item.ToolError,
		&item.ResponseType,
		&actionsRaw,
		&entitiesRaw,
		&predictionLogID,
		&item.CreatedAt,
	); err != nil {
		return nil, err
	}
	item.PredictionLogID = predictionLogID
	_ = json.Unmarshal(extractedRaw, &item.ExtractedEntities)
	_ = json.Unmarshal(toolParamsRaw, &item.ToolParams)
	_ = json.Unmarshal(actionsRaw, &item.SuggestedActions)
	_ = json.Unmarshal(entitiesRaw, &item.EntityReferences)
	if item.ExtractedEntities == nil {
		item.ExtractedEntities = map[string]string{}
	}
	if item.ToolParams == nil {
		item.ToolParams = map[string]string{}
	}
	return &item, nil
}

func defaultMap(value map[string]string) map[string]string {
	if value == nil {
		return map[string]string{}
	}
	return value
}

func defaultActions(value []chatmodel.SuggestedAction) []chatmodel.SuggestedAction {
	if value == nil {
		return []chatmodel.SuggestedAction{}
	}
	return value
}

func defaultEntities(value []chatmodel.EntityReference) []chatmodel.EntityReference {
	if value == nil {
		return []chatmodel.EntityReference{}
	}
	return value
}

func IsConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
