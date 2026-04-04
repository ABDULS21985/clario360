package tools

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	datadto "github.com/clario360/platform/internal/data/dto"
	datamodel "github.com/clario360/platform/internal/data/model"
)

type RecommendationTool struct {
	baseTool
}

type recommendationItem struct {
	Priority   int    `json:"priority"`
	Category   string `json:"category"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Severity   string `json:"severity"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Score      int    `json:"-"`
}

func NewRecommendationTool(deps *Dependencies) *RecommendationTool {
	return &RecommendationTool{baseTool: newBaseTool(deps)}
}

func (t *RecommendationTool) Name() string { return "recommendation" }

func (t *RecommendationTool) Description() string {
	return "get prioritized recommendations on what to focus on"
}

func (t *RecommendationTool) RequiredPermissions() []string {
	return []string{"cyber:read", "data:read", "acta:read", "lex:read"}
}

func (t *RecommendationTool) Execute(ctx context.Context, tenantID uuid.UUID, userID uuid.UUID, _ map[string]string) (*ToolResult, error) {
	type state struct {
		criticalAlerts []map[string]any
		pipelines      []map[string]any
		contracts      []map[string]any
		ueba           []map[string]any
		actionItems    []map[string]any
		compliance     []map[string]any
		warnings       []string
	}
	current := &state{}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	timeoutCtx, cancel := context.WithTimeout(gctx, 5*time.Second)
	defer cancel()
	actor := t.actorFromContext(ctx, userID)

	g.Go(func() error {
		if t.deps == nil || t.deps.AlertService == nil {
			return nil
		}
		resp, err := t.deps.AlertService.ListAlerts(timeoutCtx, tenantID, alertListParams([]string{"critical"}, []string{"new", "acknowledged"}, nil, nil, 5), actor)
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "critical alerts")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range resp.Data {
			current.criticalAlerts = append(current.criticalAlerts, map[string]any{
				"id":       item.ID.String(),
				"title":    item.Title,
				"severity": item.Severity,
				"status":   item.Status,
				"created":  item.CreatedAt,
			})
		}
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		if t.deps == nil || t.deps.DataPipelineRepo == nil {
			return nil
		}
		items, _, err := t.deps.DataPipelineRepo.List(timeoutCtx, tenantID, datadto.ListPipelinesParams{Page: 1, PerPage: 6, Sort: "updated_at", Order: "desc"})
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "pipelines")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range items {
			if item.Status == datamodel.PipelineStatusError || (item.LastRunStatus != nil && strings.EqualFold(*item.LastRunStatus, "failed")) {
				consecutive := 0
				if t.deps.DataPipelineRunRepo != nil {
					consecutive, _ = t.deps.DataPipelineRunRepo.ConsecutiveFailures(timeoutCtx, tenantID, item.ID, 10)
				}
				current.pipelines = append(current.pipelines, map[string]any{
					"id":          item.ID.String(),
					"name":        item.Name,
					"consecutive": consecutive,
				})
			}
		}
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		if t.deps == nil || t.deps.LexContractRepo == nil {
			return nil
		}
		items, err := t.deps.LexContractRepo.ListExpiring(timeoutCtx, tenantID, 7)
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "contracts")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range items {
			current.contracts = append(current.contracts, map[string]any{
				"id":    item.ID.String(),
				"title": item.Title,
				"days":  item.DaysUntilExpiry,
			})
		}
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		if t.deps == nil || t.deps.UEBAService == nil {
			return nil
		}
		items, err := t.deps.UEBAService.GetRiskRanking(timeoutCtx, tenantID, 5)
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "ueba")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range items {
			if item.RiskScore >= 70 {
				current.ueba = append(current.ueba, map[string]any{
					"id":    item.EntityID,
					"name":  item.EntityName,
					"score": item.RiskScore,
				})
			}
		}
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		if t.deps == nil || t.deps.ActaStore == nil {
			return nil
		}
		items, err := t.deps.ActaStore.ListOverdueActionItems(timeoutCtx, tenantID, 3)
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "action items")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range items {
			current.actionItems = append(current.actionItems, map[string]any{
				"id":    item.ID.String(),
				"title": item.Title,
				"due":   item.DueDate,
			})
		}
		mu.Unlock()
		return nil
	})
	g.Go(func() error {
		if t.deps == nil || t.deps.LexComplianceService == nil {
			return nil
		}
		items, _, err := t.deps.LexComplianceService.ListAlerts(timeoutCtx, tenantID, "open", "critical", 1, 3)
		if err != nil {
			mu.Lock()
			current.warnings = append(current.warnings, "compliance alerts")
			mu.Unlock()
			return nil
		}
		mu.Lock()
		for _, item := range items {
			current.compliance = append(current.compliance, map[string]any{
				"id":    item.ID.String(),
				"title": item.Title,
			})
		}
		mu.Unlock()
		return nil
	})
	_ = g.Wait()

	recs := make([]recommendationItem, 0, 8)
	if len(current.criticalAlerts) > 0 {
		first := current.criticalAlerts[0]
		recs = append(recs, recommendationItem{
			Priority:   1,
			Category:   "security",
			Title:      fmt.Sprintf("Resolve %d critical security alerts", len(current.criticalAlerts)),
			Detail:     fmt.Sprintf("Most urgent: '%s'.", mapString(first, "title")),
			Severity:   "critical",
			EntityType: "alert",
			EntityID:   mapString(first, "id"),
			Score:      100,
		})
	}
	if len(current.pipelines) > 0 {
		first := current.pipelines[0]
		consecutive := mapInt(first, "consecutive")
		score := 50
		if consecutive >= 3 {
			score = 80
		}
		recs = append(recs, recommendationItem{
			Category:   "data",
			Title:      fmt.Sprintf("Fix failing pipeline '%s'", mapString(first, "name")),
			Detail:     fmt.Sprintf("%d consecutive failures detected.", consecutive),
			Severity:   "high",
			EntityType: "pipeline",
			EntityID:   mapString(first, "id"),
			Score:      score,
		})
	}
	if len(current.contracts) > 0 {
		first := current.contracts[0]
		days := mapInt(first, "days")
		score := 45
		if days < 3 {
			score = 75
		}
		recs = append(recs, recommendationItem{
			Category:   "legal",
			Title:      fmt.Sprintf("Review expiring contract '%s'", mapString(first, "title")),
			Detail:     fmt.Sprintf("Expires in %d days.", days),
			Severity:   "medium",
			EntityType: "contract",
			EntityID:   mapString(first, "id"),
			Score:      score,
		})
	}
	if len(current.ueba) > 0 {
		first := current.ueba[0]
		uebaScore := mapFloat64(first, "score")
		score := 40
		if uebaScore > 80 {
			score = 70
		}
		recs = append(recs, recommendationItem{
			Category:   "ueba",
			Title:      fmt.Sprintf("Investigate high-risk entity '%s'", mapString(first, "name")),
			Detail:     fmt.Sprintf("Risk score %.1f/100.", uebaScore),
			Severity:   "high",
			EntityType: "user",
			EntityID:   mapString(first, "id"),
			Score:      score,
		})
	}
	if len(current.actionItems) > 0 {
		first := current.actionItems[0]
		dueDate := mapTime(first, "due")
		recs = append(recs, recommendationItem{
			Category:   "governance",
			Title:      fmt.Sprintf("Address overdue action item '%s'", mapString(first, "title")),
			Detail:     fmt.Sprintf("Due date: %s.", dueDate.Format("2006-01-02")),
			Severity:   "medium",
			EntityType: "action_item",
			EntityID:   mapString(first, "id"),
			Score:      65,
		})
	}
	if len(current.compliance) > 0 {
		first := current.compliance[0]
		recs = append(recs, recommendationItem{
			Category:   "compliance",
			Title:      fmt.Sprintf("Address compliance gap '%s'", mapString(first, "title")),
			Detail:     "Critical compliance alert is still open.",
			Severity:   "medium",
			EntityType: "compliance_alert",
			EntityID:   mapString(first, "id"),
			Score:      60,
		})
	}
	if len(recs) == 0 {
		return makeListResult("Everything looks good right now. I did not find any urgent cross-suite priorities.", map[string]any{"items": []recommendationItem{}}, []chatmodel.SuggestedAction{
			messageAction("Check risk score", "What is our risk score?"),
			messageAction("Generate executive report", "Generate executive report"),
		}, nil), nil
	}
	sort.SliceStable(recs, func(i, j int) bool { return recs[i].Score > recs[j].Score })
	if len(recs) > 5 {
		recs = recs[:5]
	}
	lines := []string{"Here are your top priorities for today:", ""}
	entities := make([]chatmodel.EntityReference, 0, len(recs))
	actions := make([]chatmodel.SuggestedAction, 0, len(recs))
	for idx := range recs {
		recs[idx].Priority = idx + 1
		lines = append(lines, fmt.Sprintf("%d. %s **%s** — %s", idx+1, formatSeverityIcon(recs[idx].Severity), recs[idx].Title, recs[idx].Detail))
		entities = append(entities, entityRef(recs[idx].EntityType, recs[idx].EntityID, recs[idx].Title, idx))
		switch recs[idx].EntityType {
		case "alert":
			actions = append(actions, messageAction("Investigate alert", "Investigate alert "+recs[idx].EntityID))
		case "pipeline":
			actions = append(actions, navigateAction("Open pipeline", "/data/pipelines/"+recs[idx].EntityID))
		case "contract":
			actions = append(actions, navigateAction("Open contract", "/lex/contracts/"+recs[idx].EntityID))
		case "user":
			actions = append(actions, navigateAction("Open UEBA", "/cyber/ueba"))
		default:
			actions = append(actions, messageAction("Show recommendations", "What should I focus on today?"))
		}
	}
	if len(current.warnings) > 0 {
		lines = append(lines, "", "Some data sources were unavailable: "+strings.Join(current.warnings, ", ")+".")
	}
	return makeListResult(strings.Join(lines, "\n"), recs, actions, entities), nil
}
