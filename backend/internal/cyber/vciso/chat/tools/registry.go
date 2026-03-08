package tools

type ToolRegistry struct {
	items map[string]Tool
}

func NewRegistry(deps *Dependencies) *ToolRegistry {
	items := map[string]Tool{}
	register := func(tool Tool) {
		items[tool.Name()] = tool
	}
	register(NewRiskScoreTool(deps))
	register(NewAlertSummaryTool(deps))
	register(NewAlertDetailTool(deps))
	register(NewAssetLookupTool(deps))
	register(NewVulnerabilitySummaryTool(deps))
	register(NewMITRECoverageTool(deps))
	register(NewUEBASummaryTool(deps))
	register(NewPipelineStatusTool(deps))
	register(NewComplianceScoreTool(deps))
	register(NewRecommendationTool(deps))
	register(NewDashboardBuilderTool(deps))
	register(NewInvestigationTool(deps))
	register(NewTrendAnalysisTool(deps))
	register(NewRemediationTool(deps))
	register(NewReportGeneratorTool(deps))
	return &ToolRegistry{items: items}
}

func (r *ToolRegistry) Get(name string) Tool {
	if r == nil {
		return nil
	}
	return r.items[name]
}

func (r *ToolRegistry) List() []Tool {
	if r == nil {
		return nil
	}
	out := make([]Tool, 0, len(r.items))
	for _, item := range r.items {
		out = append(out, item)
	}
	return out
}

