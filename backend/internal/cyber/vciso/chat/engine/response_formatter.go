package engine

import (
	"fmt"
	"strings"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	"github.com/clario360/platform/internal/cyber/vciso/chat/tools"
)

type ResponseFormatter struct{}

func NewResponseFormatter() *ResponseFormatter {
	return &ResponseFormatter{}
}

func (f *ResponseFormatter) FormatToolResult(result *tools.ToolResult) chatmodel.ResponsePayload {
	if result == nil {
		return chatmodel.ResponsePayload{Text: "No result was returned.", DataType: "text", Actions: []chatmodel.SuggestedAction{}}
	}
	return chatmodel.ResponsePayload{
		Text:     result.Text,
		Data:     result.Data,
		DataType: result.DataType,
		Actions:  result.Actions,
		Entities: result.Entities,
	}
}

func (f *ResponseFormatter) UnknownIntent(suggestions []chatdto.Suggestion) chatmodel.ResponsePayload {
	lines := []string{
		"I'm not sure I understood that. Here's what I can help with:",
		"",
		"**Security Analysis**",
		`- "What is our risk score?"`,
		`- "Show critical alerts"`,
		`- "Investigate alert {ID}"`,
		"",
		"**Monitoring**",
		`- "Are any pipelines failing?"`,
		`- "Who are the riskiest users?"`,
		`- "MITRE coverage gaps"`,
		"",
		"**Reports & Dashboards**",
		`- "What should I focus on today?"`,
		`- "Build a security dashboard"`,
		`- "Generate executive report"`,
		"",
		"Try asking one of these, or rephrase your question.",
	}
	return chatmodel.ResponsePayload{
		Text:     strings.Join(lines, "\n"),
		DataType: "text",
		Actions:  suggestionActions(suggestions),
		Entities: []chatmodel.EntityReference{},
	}
}

func (f *ResponseFormatter) Clarification(req *ClarificationRequest) chatmodel.ResponsePayload {
	text := "Which item would you like me to use?"
	if req != nil && req.Message != "" {
		text = req.Message
	}
	return chatmodel.ResponsePayload{
		Text:     text,
		DataType: "text",
		Actions:  []chatmodel.SuggestedAction{},
		Entities: []chatmodel.EntityReference{},
	}
}

func (f *ResponseFormatter) PermissionDenied(description string, missing []string) chatmodel.ResponsePayload {
	text := fmt.Sprintf("You don't have permission to %s.", description)
	if len(missing) > 0 {
		text += " Required: " + strings.Join(missing, ", ") + "."
	}
	text += " Contact your admin to request access."
	return chatmodel.ResponsePayload{Text: text, DataType: "text", Actions: []chatmodel.SuggestedAction{}, Entities: []chatmodel.EntityReference{}}
}

func (f *ResponseFormatter) ToolTimeout() chatmodel.ResponsePayload {
	return chatmodel.ResponsePayload{
		Text:     "This query is taking longer than expected. Let me work on it and try again shortly.",
		DataType: "text",
		Actions:  []chatmodel.SuggestedAction{},
		Entities: []chatmodel.EntityReference{},
	}
}

func (f *ResponseFormatter) ToolError(err error) chatmodel.ResponsePayload {
	text := "I encountered an error while processing your request."
	if err != nil {
		text += " " + sanitizeError(err.Error())
	}
	text += " You can try again or rephrase your question."
	return chatmodel.ResponsePayload{Text: text, DataType: "text", Actions: []chatmodel.SuggestedAction{}, Entities: []chatmodel.EntityReference{}}
}

func sanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\n", " ")
	if len(value) > 240 {
		return value[:240] + "..."
	}
	return value
}

func suggestionActions(suggestions []chatdto.Suggestion) []chatmodel.SuggestedAction {
	out := make([]chatmodel.SuggestedAction, 0, len(suggestions))
	for _, suggestion := range suggestions {
		out = append(out, chatmodel.SuggestedAction{
			Label: suggestion.Text,
			Type:  "execute_tool",
			Params: map[string]string{
				"message": suggestion.Text,
			},
		})
	}
	return out
}
