package engine

import (
	"strings"

	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	llmmodel "github.com/clario360/platform/internal/cyber/vciso/llm/model"
)

type ResponseSynthesizer struct{}

func NewResponseSynthesizer() *ResponseSynthesizer {
	return &ResponseSynthesizer{}
}

func (s *ResponseSynthesizer) Synthesize(text string, toolResults []*llmmodel.ToolCallResult, grounding *llmmodel.GroundingResult) (chatmodel.ResponsePayload, *chatdto.ResponseMeta) {
	payload := chatmodel.ResponsePayload{
		Text:     text,
		DataType: "text",
		Actions:  []chatmodel.SuggestedAction{},
		Entities: []chatmodel.EntityReference{},
	}
	meta := &chatdto.ResponseMeta{
		Grounding: "passed",
		Engine:    "llm",
	}
	if grounding != nil && grounding.Status != "" {
		meta.Grounding = grounding.Status
		if grounding.Status == "corrected" && grounding.CorrectedResponse != "" {
			payload.Text = grounding.CorrectedResponse
		}
	}
	for _, result := range toolResults {
		if result == nil {
			continue
		}
		if richer(result.DataType, payload.DataType) {
			payload.DataType = result.DataType
			payload.Data = result.Data
		}
		payload.Actions = mergeActions(payload.Actions, result.Actions)
		payload.Entities = append(payload.Entities, result.Entities...)
	}
	meta.ToolCallsCount = len(toolResults)
	meta.ReasoningSteps = max(meta.ToolCallsCount, 1)
	return payload, meta
}

func richer(candidate, current string) bool {
	order := map[string]int{
		"text":          1,
		"list":          2,
		"kpi":           3,
		"table":         4,
		"chart":         5,
		"dashboard":     6,
		"investigation": 7,
	}
	return order[strings.ToLower(candidate)] > order[strings.ToLower(current)]
}

func mergeActions(current, incoming []chatmodel.SuggestedAction) []chatmodel.SuggestedAction {
	seen := map[string]struct{}{}
	out := make([]chatmodel.SuggestedAction, 0, len(current)+len(incoming))
	for _, item := range append(current, incoming...) {
		key := item.Type + ":" + item.Label
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
		if len(out) == 4 {
			break
		}
	}
	return out
}
