package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	chatrepo "github.com/clario360/platform/internal/cyber/vciso/chat/repository"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

type CompiledContext struct {
	Messages   []llmmodel.LLMMessage
	ContextTurns int
}

type ContextCompiler struct {
	conversationRepo *chatrepo.ConversationRepository
	budget           TokenBudget
	metrics          *Metrics
}

func NewContextCompiler(conversationRepo *chatrepo.ConversationRepository, budget TokenBudget, metrics *Metrics) *ContextCompiler {
	return &ContextCompiler{conversationRepo: conversationRepo, budget: budget, metrics: metrics}
}

func (c *ContextCompiler) Compile(ctx context.Context, conversationID *uuid.UUID, tenantID uuid.UUID) (*CompiledContext, error) {
	compiled := &CompiledContext{Messages: []llmmodel.LLMMessage{}}
	if c == nil || c.conversationRepo == nil || conversationID == nil || *conversationID == uuid.Nil {
		return compiled, nil
	}
	messages, err := c.conversationRepo.ListMessages(ctx, tenantID, *conversationID)
	if err != nil {
		return nil, err
	}
	if len(messages) == 0 {
		return compiled, nil
	}
	compiled.ContextTurns = min(len(messages), 10)
	start := max(len(messages)-10, 0)
	if start > 0 {
		summary := summarizeMessages(messages[:start])
		if summary != "" {
			compiled.Messages = append(compiled.Messages, llmmodel.LLMMessage{
				Role:    "assistant",
				Content: "Conversation summary: " + summary,
			})
		}
	}
	tokens := 0
	for _, item := range messages[start:] {
		role := string(item.Role)
		if role == "system" {
			continue
		}
		msg := llmmodel.LLMMessage{Role: role, Content: item.Content}
		messageTokens := estimateTokens(item.Content)
		if tokens+messageTokens > c.budget.ConversationHistoryBudget && len(compiled.Messages) > 0 {
			break
		}
		compiled.Messages = append(compiled.Messages, msg)
		tokens += messageTokens
	}
	if c.metrics != nil && c.metrics.ContextTokensUsed != nil {
		c.metrics.ContextTokensUsed.Observe(float64(tokens))
	}
	return compiled, nil
}

func summarizeMessages(items []chatrepo.MessageCompat) string {
	_ = items
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func summarizeMessages(messages interface{}) string {
	switch typed := messages.(type) {
	case []string:
		return strings.Join(typed, " ")
	default:
		return ""
	}
}

func summarizeConversation(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return fmt.Sprintf("%d earlier turns summarized.", len(lines))
}
