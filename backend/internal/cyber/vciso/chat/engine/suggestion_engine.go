package engine

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	chatdto "github.com/clario360/platform/internal/cyber/vciso/chat/dto"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	"github.com/clario360/platform/internal/cyber/vciso/chat/tools"
	datadto "github.com/clario360/platform/internal/data/dto"
	datamodel "github.com/clario360/platform/internal/data/model"
)

type SuggestionEngine struct {
	deps    *tools.Dependencies
	metrics *VCISOMetrics
	logger  zerolog.Logger
	now     func() time.Time
}

func NewSuggestionEngine(deps *tools.Dependencies, metrics *VCISOMetrics, logger zerolog.Logger) *SuggestionEngine {
	now := func() time.Time { return time.Now().UTC() }
	if deps != nil && deps.Now != nil {
		now = deps.Now
	}
	return &SuggestionEngine{
		deps:    deps,
		metrics: metrics,
		logger:  logger.With().Str("component", "vciso_suggestion_engine").Logger(),
		now:     now,
	}
}

func (s *SuggestionEngine) GetSuggestions(ctx context.Context, tenantID uuid.UUID, conversationCtx *chatmodel.ConversationContext) ([]chatdto.Suggestion, error) {
	items := make([]chatdto.Suggestion, 0, 6)
	add := func(item chatdto.Suggestion) {
		if strings.TrimSpace(item.Text) == "" {
			return
		}
		for _, existing := range items {
			if existing.Text == item.Text {
				return
			}
		}
		items = append(items, item)
	}

	if conversationCtx != nil && len(conversationCtx.LastEntities) > 0 {
		entity := conversationCtx.LastEntities[0]
		switch entity.Type {
		case "alert":
			add(chatdto.Suggestion{
				Text:     "Investigate " + entity.Name,
				Category: "security",
				Priority: 95,
				Reason:   "Follow up on the alert we were discussing",
			})
		case "asset":
			add(chatdto.Suggestion{
				Text:     "Tell me about " + entity.Name,
				Category: "security",
				Priority: 95,
				Reason:   "Continue investigating the asset from the last response",
			})
		}
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	if s.deps != nil && s.deps.AlertService != nil {
		count, err := s.deps.AlertService.Count(timeoutCtx, tenantID, alertListParams([]string{"critical"}, []string{"new", "acknowledged", "investigating"}, nil, nil, 1), nil)
		if err == nil && count > 0 {
			add(chatdto.Suggestion{
				Text:     "Show critical alerts",
				Category: "security",
				Priority: 100,
				Reason:   pluralizeCount(count, "unresolved critical alert", "unresolved critical alerts"),
			})
		}
	}

	if s.deps != nil && s.deps.RiskService != nil {
		score, err := s.deps.RiskService.GetCurrentScore(timeoutCtx, tenantID)
		if err == nil && score != nil && score.OverallScore > 70 {
			add(chatdto.Suggestion{
				Text:     "What should I focus on today?",
				Category: "security",
				Priority: 90,
				Reason:   "Risk score is " + formatScoreReason(score.OverallScore, score.Grade),
			})
		}
	}

	if s.deps != nil && s.deps.DataPipelineRepo != nil {
		pipelines, _, err := s.deps.DataPipelineRepo.List(timeoutCtx, tenantID, datadto.ListPipelinesParams{Page: 1, PerPage: 10, Sort: "updated_at", Order: "desc"})
		if err == nil {
			failed := 0
			for _, item := range pipelines {
				if item.Status == datamodel.PipelineStatusError || (item.LastRunStatus != nil && strings.EqualFold(*item.LastRunStatus, "failed")) {
					failed++
				}
			}
			if failed > 0 {
				add(chatdto.Suggestion{
					Text:     "Are any pipelines failing?",
					Category: "data",
					Priority: 85,
					Reason:   pluralizeCount(failed, "pipeline in failed state", "pipelines in failed state"),
				})
			}
		}
	}

	if s.deps != nil && s.deps.LexComplianceService != nil {
		score, err := s.deps.LexComplianceService.GetScore(timeoutCtx, tenantID)
		if err == nil && score != nil && score.OpenAlerts > 0 {
			add(chatdto.Suggestion{
				Text:     "What's our compliance status?",
				Category: "compliance",
				Priority: 75,
				Reason:   pluralizeCount(score.OpenAlerts, "open compliance alert", "open compliance alerts"),
			})
		}
	}

	if s.deps != nil && s.deps.UEBAService != nil {
		items, err := s.deps.UEBAService.GetRiskRanking(timeoutCtx, tenantID, 5)
		if err == nil {
			highRisk := 0
			for _, item := range items {
				if item.RiskScore > 70 {
					highRisk++
				}
			}
			if highRisk > 0 {
				add(chatdto.Suggestion{
					Text:     "Who are the riskiest users?",
					Category: "security",
					Priority: 70,
					Reason:   pluralizeCount(highRisk, "user with risk score above 70", "users with risk score above 70"),
				})
			}
		}
	}

	add(chatdto.Suggestion{
		Text:     "What is our risk score?",
		Category: "security",
		Priority: 50,
		Reason:   "Core security posture overview",
	})
	add(chatdto.Suggestion{
		Text:     "Generate executive report",
		Category: "general",
		Priority: 40,
		Reason:   "Create a board-ready security summary",
	})

	sort.SliceStable(items, func(i, j int) bool { return items[i].Priority > items[j].Priority })
	if len(items) > 5 {
		items = items[:5]
	}
	if s.metrics != nil && s.metrics.SuggestionsServedTotal != nil {
		s.metrics.SuggestionsServedTotal.Inc()
	}
	return items, nil
}

func alertListParams(severities, statuses []string, dateFrom, dateTo *time.Time, perPage int) *cyberdto.AlertListParams {
	params := &cyberdto.AlertListParams{
		Severities: severities,
		Statuses:   statuses,
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Page:       1,
		PerPage:    perPage,
		Sort:       "created_at",
		Order:      "desc",
	}
	params.SetDefaults()
	if perPage > 0 {
		params.PerPage = perPage
	}
	return params
}

func pluralizeCount(count int, singular, plural string) string {
	if count == 1 {
		return "1 " + singular
	}
	return strings.TrimSpace(strings.Join([]string{strconvItoa(count), plural}, " "))
}

func formatScoreReason(score float64, grade string) string {
	return fmt.Sprintf("%.1f/100 (Grade %s)", score, grade)
}

func strconvItoa(value int) string {
	return strconv.Itoa(value)
}
