package tools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	actarepo "github.com/clario360/platform/internal/acta/repository"
	actaservice "github.com/clario360/platform/internal/acta/service"
	"github.com/clario360/platform/internal/auth"
	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	cybersvc "github.com/clario360/platform/internal/cyber/service"
	uebasvc "github.com/clario360/platform/internal/cyber/ueba/service"
	chatmodel "github.com/clario360/platform/internal/cyber/vciso/chat/model"
	datarepo "github.com/clario360/platform/internal/data/repository"
	"github.com/clario360/platform/internal/events"
	lexrepo "github.com/clario360/platform/internal/lex/repository"
	lexservice "github.com/clario360/platform/internal/lex/service"
	visusmodel "github.com/clario360/platform/internal/visus/model"
	visusrepo "github.com/clario360/platform/internal/visus/repository"
	visusservice "github.com/clario360/platform/internal/visus/service"
)

var errToolUnavailable = errors.New("tool dependency is unavailable")

type Dependencies struct {
	CyberDB              *pgxpool.Pool
	AlertService         *cybersvc.AlertService
	AlertRepo            *cyberrepo.AlertRepository
	AssetService         *cybersvc.AssetService
	AssetRepo            *cyberrepo.AssetRepository
	VulnerabilityService *cybersvc.VulnerabilityService
	RiskService          *cybersvc.RiskService
	RuleService          *cybersvc.RuleService
	UEBAService          *uebasvc.UEBAService
	VCISOService         *cybersvc.VCISOService
	RemediationService   *cybersvc.RemediationService

	DataPool            *pgxpool.Pool
	DataPipelineRepo    *datarepo.PipelineRepository
	DataPipelineRunRepo *datarepo.PipelineRunRepository

	ActaPool              *pgxpool.Pool
	ActaStore             *actarepo.Store
	ActaDashboardService  *actaservice.DashboardService
	ActaComplianceService *actaservice.ComplianceService

	LexPool              *pgxpool.Pool
	LexContractRepo      *lexrepo.ContractRepository
	LexAlertRepo         *lexrepo.AlertRepository
	LexDocumentRepo      *lexrepo.DocumentRepository
	LexClauseRepo        *lexrepo.ClauseRepository
	LexComplianceRepo    *lexrepo.ComplianceRepository
	LexComplianceService *lexservice.ComplianceService
	LexDashboardService  *lexservice.DashboardService

	VisusPool              *pgxpool.Pool
	VisusDashboardRepo     *visusrepo.DashboardRepository
	VisusWidgetRepo        *visusrepo.WidgetRepository
	VisusKPIRepo           *visusrepo.KPIRepository
	VisusSnapshotRepo      *visusrepo.KPISnapshotRepository
	VisusAlertRepo         *visusrepo.AlertRepository
	VisusDashboardService  *visusservice.DashboardService
	VisusWidgetService     *visusservice.WidgetService

	Producer *events.Producer
	Logger   zerolog.Logger
	Now      func() time.Time
}

type baseTool struct {
	deps *Dependencies
}

func newBaseTool(deps *Dependencies) baseTool {
	if deps != nil && deps.Now == nil {
		deps.Now = func() time.Time { return time.Now().UTC() }
	}
	return baseTool{deps: deps}
}

func (b baseTool) now() time.Time {
	if b.deps != nil && b.deps.Now != nil {
		return b.deps.Now()
	}
	return time.Now().UTC()
}

func (b baseTool) logger() zerolog.Logger {
	if b.deps == nil {
		return zerolog.Nop()
	}
	return b.deps.Logger
}

func (b baseTool) actorFromContext(ctx context.Context, userID uuid.UUID) *cybersvc.Actor {
	claims := auth.ClaimsFromContext(ctx)
	email := ""
	if claims != nil {
		email = claims.Email
	}
	return &cybersvc.Actor{
		UserID:    userID,
		UserName:  email,
		UserEmail: email,
	}
}

func (b baseTool) requireCyberAlertID(ctx context.Context, tenantID uuid.UUID, value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, fmt.Errorf("alert_id is required")
	}
	if parsed, err := uuid.Parse(value); err == nil {
		return parsed, nil
	}
	if b.deps == nil || b.deps.CyberDB == nil {
		return uuid.Nil, fmt.Errorf("%w: cyber database", errToolUnavailable)
	}
	if strings.HasPrefix(value, "#") {
		needle := strings.TrimPrefix(value, "#")
		var id uuid.UUID
		err := b.deps.CyberDB.QueryRow(ctx, `
			SELECT id
			FROM alerts
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND (
				metadata ->> 'legacy_id' = $2 OR
				title ILIKE '%' || $3 || '%'
			  )
			ORDER BY created_at DESC
			LIMIT 1`,
			tenantID, needle, value,
		).Scan(&id)
		if err == nil {
			return id, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return uuid.Nil, fmt.Errorf("alert %s was not found", value)
		}
		return uuid.Nil, fmt.Errorf("resolve alert %s: %w", value, err)
	}
	rows, err := b.deps.CyberDB.Query(ctx, `
		SELECT id
		FROM alerts
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND id::text ILIKE $2 || '%'
		ORDER BY created_at DESC
		LIMIT 2`,
		tenantID, strings.ToLower(value),
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("resolve alert %s: %w", value, err)
	}
	defer rows.Close()
	var matches []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if scanErr := rows.Scan(&id); scanErr != nil {
			return uuid.Nil, scanErr
		}
		matches = append(matches, id)
	}
	if err := rows.Err(); err != nil {
		return uuid.Nil, err
	}
	switch len(matches) {
	case 0:
		return uuid.Nil, fmt.Errorf("alert %s was not found", value)
	case 1:
		return matches[0], nil
	default:
		return uuid.Nil, fmt.Errorf("alert %s matched multiple alerts; use the full UUID", value)
	}
}

func (b baseTool) parseCount(params map[string]string, fallback, max int) int {
	if fallback <= 0 {
		fallback = 5
	}
	value := fallback
	if raw := strings.TrimSpace(params["count"]); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			value = parsed
		}
	}
	if max > 0 && value > max {
		return max
	}
	return value
}

func (b baseTool) parseStartEnd(params map[string]string, fallbackDays int) (time.Time, time.Time) {
	end := b.now()
	start := end.AddDate(0, 0, -fallbackDays)
	if raw := strings.TrimSpace(params["start_time"]); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			start = parsed
		}
	}
	if raw := strings.TrimSpace(params["end_time"]); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			end = parsed
		}
	}
	if end.Before(start) {
		return end, start
	}
	return start, end
}

func formatSeverityIcon(severity string) string {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return "🔴"
	case "high":
		return "🟠"
	case "medium":
		return "🟡"
	case "low":
		return "🔵"
	default:
		return "⚪"
	}
}

func severityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func friendlyTimeAgo(now, value time.Time) string {
	if value.IsZero() {
		return "unknown time"
	}
	delta := now.Sub(value)
	switch {
	case delta < time.Minute:
		return "just now"
	case delta < time.Hour:
		return fmt.Sprintf("%dm ago", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(delta.Hours()))
	case delta < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(delta.Hours()/24))
	default:
		return value.Format("2006-01-02 15:04 MST")
	}
}

func confidencePercent(value float64) string {
	if value <= 1 {
		return fmt.Sprintf("%.0f%%", value*100)
	}
	return fmt.Sprintf("%.0f%%", value)
}

func entityRef(entityType, id, name string, index int) chatmodel.EntityReference {
	return chatmodel.EntityReference{
		Type:  entityType,
		ID:    id,
		Name:  name,
		Index: index,
	}
}

func navigateAction(label, url string) chatmodel.SuggestedAction {
	return chatmodel.SuggestedAction{
		Label: label,
		Type:  "navigate",
		Params: map[string]string{
			"url": url,
		},
	}
}

func messageAction(label, message string) chatmodel.SuggestedAction {
	return chatmodel.SuggestedAction{
		Label: label,
		Type:  "execute_tool",
		Params: map[string]string{
			"message": message,
		},
	}
}

func confirmMessageAction(label, message, warning string) chatmodel.SuggestedAction {
	return chatmodel.SuggestedAction{
		Label: label,
		Type:  "confirm",
		Params: map[string]string{
			"message": message,
			"warning": warning,
		},
	}
}

func joinLines(lines ...string) string {
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			filtered = append(filtered, "")
			continue
		}
		filtered = append(filtered, line)
	}
	return strings.Join(filtered, "\n")
}

func clampCount(value, floor, ceiling int) int {
	if value < floor {
		return floor
	}
	if ceiling > 0 && value > ceiling {
		return ceiling
	}
	return value
}

func maybePlural(count int, singular, plural string) string {
	if count == 1 {
		return singular
	}
	return plural
}

func csvValues(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		values = append(values, part)
	}
	return values
}

func normalizeSeveritySet(value string) string {
	items := csvValues(value)
	sort.SliceStable(items, func(i, j int) bool {
		return severityRank(items[i]) > severityRank(items[j])
	})
	return strings.Join(items, ",")
}

func assetSearchParams(query string) *cyberdto.AssetListParams {
	params := &cyberdto.AssetListParams{
		Page:    1,
		PerPage: 10,
		Sort:    "created_at",
		Order:   "desc",
	}
	if trimmed := strings.TrimSpace(query); trimmed != "" {
		params.Search = &trimmed
	}
	return params
}

func alertListParams(severities []string, statuses []string, start *time.Time, end *time.Time, perPage int) *cyberdto.AlertListParams {
	params := &cyberdto.AlertListParams{
		Severities: severities,
		Statuses:   statuses,
		Page:       1,
		PerPage:    clampCount(perPage, 1, 100),
		Sort:       "created_at",
		Order:      "desc",
	}
	if start != nil {
		params.DateFrom = start
	}
	if end != nil {
		params.DateTo = end
	}
	return params
}

func riskDirection(delta float64) string {
	switch {
	case delta > 0.5:
		return "worsened"
	case delta < -0.5:
		return "improved"
	default:
		return "stayed stable"
	}
}

func sourceIPLabel(value string) string {
	ip := net.ParseIP(strings.TrimSpace(value))
	if ip == nil {
		return value
	}
	if ip.IsPrivate() {
		return value + " (private)"
	}
	return value
}

func dashboardKPIWidget(dashboardID, tenantID uuid.UUID, title string, kpiID uuid.UUID, position visusmodel.WidgetPosition) visusmodel.Widget {
	return visusmodel.Widget{
		DashboardID: dashboardID,
		TenantID:    tenantID,
		Title:       title,
		Type:        visusmodel.WidgetTypeGauge,
		Position:    position,
		Config: map[string]any{
			"kpi_id": kpiID.String(),
		},
		RefreshIntervalSeconds: 300,
	}
}

func makeListResult(text string, data any, actions []chatmodel.SuggestedAction, entities []chatmodel.EntityReference) *ToolResult {
	return &ToolResult{
		Text:     text,
		Data:     data,
		DataType: "list",
		Actions:  actions,
		Entities: entities,
	}
}
