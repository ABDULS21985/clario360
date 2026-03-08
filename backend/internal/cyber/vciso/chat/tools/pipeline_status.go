package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	datadto "github.com/clario360/platform/internal/data/dto"
	datamodel "github.com/clario360/platform/internal/data/model"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type PipelineStatusTool struct {
	baseTool
}

func NewPipelineStatusTool(deps *Dependencies) *PipelineStatusTool {
	return &PipelineStatusTool{baseTool: newBaseTool(deps)}
}

func (t *PipelineStatusTool) Name() string { return "pipeline_status" }

func (t *PipelineStatusTool) Description() string { return "check data pipeline health and failures" }

func (t *PipelineStatusTool) RequiredPermissions() []string { return []string{"data:read"} }

func (t *PipelineStatusTool) Execute(ctx context.Context, tenantID uuid.UUID, _ uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.DataPipelineRepo == nil {
		return nil, fmt.Errorf("%w: pipeline repository", errToolUnavailable)
	}
	limit := t.parseCount(params, 10, 25)
	items, _, err := t.deps.DataPipelineRepo.List(ctx, tenantID, datadto.ListPipelinesParams{
		Page:    1,
		PerPage: limit,
		Sort:    "updated_at",
		Order:   "desc",
	})
	if err != nil {
		return nil, err
	}
	failures := make([]map[string]any, 0, len(items))
	entities := make([]chatmodel.EntityReference, 0, len(items))
	lines := []string{}
	for idx, item := range items {
		if item.Status != datamodel.PipelineStatusError && (item.LastRunStatus == nil || !strings.EqualFold(*item.LastRunStatus, "failed")) {
			continue
		}
		consecutive := 0
		if t.deps.DataPipelineRunRepo != nil {
			consecutive, _ = t.deps.DataPipelineRunRepo.ConsecutiveFailures(ctx, tenantID, item.ID, 10)
		}
		failures = append(failures, map[string]any{
			"id":                   item.ID,
			"name":                 item.Name,
			"status":               item.Status,
			"last_run_status":      item.LastRunStatus,
			"last_run_error":       item.LastRunError,
			"last_run_at":          item.LastRunAt,
			"consecutive_failures": consecutive,
		})
		lines = append(lines, fmt.Sprintf("%d. %s — last run %s (%d consecutive failures)", len(failures), item.Name, pointerString(item.LastRunStatus, "failed"), consecutive))
		entities = append(entities, entityRef("pipeline", item.ID.String(), item.Name, idx))
	}
	if len(failures) == 0 {
		return makeListResult("No failing pipelines were found.", map[string]any{"pipelines": []any{}}, []chatmodel.SuggestedAction{
			navigateAction("Open pipelines", "/data/pipelines"),
		}, nil), nil
	}
	text := joinLines(
		fmt.Sprintf("There %s **%d** failing %s right now:", map[bool]string{true: "is", false: "are"}[len(failures) == 1], len(failures), maybePlural(len(failures), "pipeline", "pipelines")),
		"",
		strings.Join(lines, "\n"),
	)
	actions := []chatmodel.SuggestedAction{navigateAction("Open pipelines", "/data/pipelines")}
	if len(failures) > 0 {
		actions = append(actions, navigateAction("Open first failing pipeline", "/data/pipelines/"+failures[0]["id"].(uuid.UUID).String()))
	}
	return makeListResult(text, map[string]any{"pipelines": failures}, actions, entities), nil
}

func pointerString(value *string, fallback string) string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return fallback
	}
	return *value
}

