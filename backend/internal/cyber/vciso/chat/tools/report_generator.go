package tools

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
)

type ReportGeneratorTool struct {
	baseTool
}

func NewReportGeneratorTool(deps *Dependencies) *ReportGeneratorTool {
	return &ReportGeneratorTool{baseTool: newBaseTool(deps)}
}

func (t *ReportGeneratorTool) Name() string { return "report_generator" }

func (t *ReportGeneratorTool) Description() string { return "generate an executive security report" }

func (t *ReportGeneratorTool) RequiredPermissions() []string { return []string{"cyber:read"} }

func (t *ReportGeneratorTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, params map[string]string) (*ToolResult, error) {
	if t.deps == nil || t.deps.VCISOService == nil {
		return nil, fmt.Errorf("%w: vciso service", errToolUnavailable)
	}
	start, end := t.parseStartEnd(params, 30)
	days := int(end.Sub(start).Hours()/24) + 1
	if days <= 0 {
		days = 30
	}
	reportType := "executive"
	if params["framework"] != "" {
		reportType = "compliance"
	}
	resp, err := t.deps.VCISOService.GenerateReport(ctx, tenantID, userID, &cyberdto.VCISOReportRequest{
		Type:       reportType,
		PeriodDays: days,
	}, t.actorFromContext(ctx, userID))
	if err != nil {
		return nil, err
	}
	return &ToolResult{
		Text: fmt.Sprintf("I generated a **%s** report covering the last **%d days**. Job ID: `%s`.", reportType, days, resp.JobID),
		Data: map[string]any{
			"job_id":      resp.JobID,
			"status":      resp.Status,
			"report_type": reportType,
			"period_days": days,
		},
		DataType: "list",
		Actions: []chatmodel.SuggestedAction{
			messageAction("Show recommendations", "What should I focus on today?"),
			messageAction("Build dashboard", "Build me a dashboard showing alerts and risk"),
		},
		Entities: nil,
	}, nil
}
