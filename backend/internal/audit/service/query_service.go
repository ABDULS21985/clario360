package service

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/audit/dto"
	"github.com/clario360/platform/internal/audit/metrics"
	"github.com/clario360/platform/internal/audit/model"
	"github.com/clario360/platform/internal/audit/repository"
)

// QueryService handles audit log query operations.
type QueryService struct {
	repo    *repository.AuditRepository
	masking *MaskingService
	logger  zerolog.Logger
}

// NewQueryService creates a new QueryService.
func NewQueryService(repo *repository.AuditRepository, masking *MaskingService, logger zerolog.Logger) *QueryService {
	return &QueryService{
		repo:    repo,
		masking: masking,
		logger:  logger,
	}
}

// Query executes a filtered, paginated audit log query.
func (s *QueryService) Query(ctx context.Context, params *dto.QueryParams, callerRoles []string) (*dto.PaginatedResult, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("list").Observe(time.Since(start).Seconds())
	}()

	filter := repository.QueryFilter{
		TenantID:     params.TenantID,
		UserID:       params.UserID,
		Service:      params.Service,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		DateFrom:     params.DateFrom,
		DateTo:       params.DateTo,
		Search:       params.Search,
		Severity:     params.Severity,
		Sort:         params.Sort,
		Order:        params.Order,
		Limit:        params.PerPage,
		Offset:       params.Offset(),
	}

	entries, total, err := s.repo.Query(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply PII masking based on caller role
	masked := s.masking.MaskEntries(entries, callerRoles)

	metrics.QueryResults.WithLabelValues("list").Add(float64(len(masked)))

	return &dto.PaginatedResult{
		Data: masked,
		Meta: dto.NewPagination(params.Page, params.PerPage, total),
	}, nil
}

// GetByID retrieves a single audit entry by ID.
func (s *QueryService) GetByID(ctx context.Context, tenantID, id string, callerRoles []string) (*model.AuditEntry, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
	}()

	entry, err := s.repo.FindByID(ctx, tenantID, id)
	if err != nil {
		return nil, err
	}
	if entry == nil {
		return nil, nil
	}

	masked := s.masking.MaskEntry(entry, callerRoles)
	return &masked, nil
}

// GetStats returns aggregated statistics for a tenant.
func (s *QueryService) GetStats(ctx context.Context, tenantID string, dateFrom, dateTo time.Time) (*model.AuditStats, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("stats").Observe(time.Since(start).Seconds())
	}()

	return s.repo.GetStats(ctx, tenantID, dateFrom, dateTo)
}

// GetTimeline returns the activity timeline for a specific resource, formatted as
// an AuditTimeline with AuditTimelineEvent entries aligned to the frontend contract.
func (s *QueryService) GetTimeline(ctx context.Context, tenantID, resourceID string, page, perPage int, callerRoles []string) (*model.AuditTimeline, error) {
	start := time.Now()
	defer func() {
		metrics.QueryDuration.WithLabelValues("timeline").Observe(time.Since(start).Seconds())
	}()

	if perPage <= 0 {
		perPage = 50
	}
	if perPage > 200 {
		perPage = 200
	}
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * perPage

	entries, _, err := s.repo.GetTimeline(ctx, tenantID, resourceID, perPage, offset)
	if err != nil {
		return nil, err
	}

	masked := s.masking.MaskEntries(entries, callerRoles)

	timeline := &model.AuditTimeline{
		ResourceID: resourceID,
		Events:     make([]model.AuditTimelineEvent, 0, len(masked)),
	}

	// Populate resource_type and resource_name from first entry
	if len(masked) > 0 {
		timeline.ResourceType = masked[0].ResourceType
		timeline.ResourceName = masked[0].ResourceID // no separate name column; use ID
	}

	for _, e := range masked {
		event := model.AuditTimelineEvent{
			ID:        e.ID,
			Action:    e.Action,
			UserName:  deriveTimelineUserName(e.UserEmail),
			Timestamp: e.CreatedAt.UTC().Format(time.RFC3339),
			Changes:   computeChanges(e.OldValue, e.NewValue),
			Summary:   deriveSummary(e.Action),
		}
		timeline.Events = append(timeline.Events, event)
	}

	return timeline, nil
}

// deriveTimelineUserName extracts a display name from an email address for the timeline view.
func deriveTimelineUserName(email string) string {
	if email == "" {
		return "System"
	}
	for i, c := range email {
		if c == '@' {
			local := email[:i]
			result := make([]byte, 0, len(local))
			capitalize := true
			for _, b := range []byte(local) {
				if b == '.' || b == '_' || b == '-' {
					result = append(result, ' ')
					capitalize = true
				} else if capitalize {
					if b >= 'a' && b <= 'z' {
						b -= 32
					}
					result = append(result, b)
					capitalize = false
				} else {
					result = append(result, b)
				}
			}
			return string(result)
		}
	}
	return email
}

// deriveSummary produces a human-readable summary from an action string.
// e.g. "user.update" → "Updated user"
func deriveSummary(action string) string {
	parts := strings.SplitN(action, ".", 2)
	if len(parts) < 2 {
		return action
	}
	resource := parts[0]
	verb := parts[1]

	switch {
	case strings.HasPrefix(verb, "creat"):
		return "Created " + resource
	case strings.HasPrefix(verb, "updat"), strings.HasPrefix(verb, "modif"), strings.HasPrefix(verb, "edit"):
		return "Updated " + resource
	case strings.HasPrefix(verb, "delet"), strings.HasPrefix(verb, "remov"):
		return "Deleted " + resource
	case strings.HasPrefix(verb, "login"), strings.HasPrefix(verb, "auth"):
		return "Authenticated as " + resource
	case strings.HasPrefix(verb, "export"):
		return "Exported " + resource
	default:
		return strings.Title(strings.ReplaceAll(action, ".", " "))
	}
}

// computeChanges produces a field-level diff between old and new JSON values.
// Returns an empty slice if either value is nil or not a JSON object.
func computeChanges(oldVal, newVal json.RawMessage) []model.AuditChangeRecord {
	if len(oldVal) == 0 || len(newVal) == 0 {
		return []model.AuditChangeRecord{}
	}

	var oldMap, newMap map[string]interface{}
	if err := json.Unmarshal(oldVal, &oldMap); err != nil {
		return []model.AuditChangeRecord{}
	}
	if err := json.Unmarshal(newVal, &newMap); err != nil {
		return []model.AuditChangeRecord{}
	}

	var changes []model.AuditChangeRecord

	// Fields present in old (changed or removed)
	for key, oldV := range oldMap {
		newV, exists := newMap[key]
		if !exists {
			changes = append(changes, model.AuditChangeRecord{
				Field:    key,
				OldValue: oldV,
				NewValue: nil,
			})
		} else if jsonValuesDiffer(oldV, newV) {
			changes = append(changes, model.AuditChangeRecord{
				Field:    key,
				OldValue: oldV,
				NewValue: newV,
			})
		}
	}

	// Fields added in new
	for key, newV := range newMap {
		if _, exists := oldMap[key]; !exists {
			changes = append(changes, model.AuditChangeRecord{
				Field:    key,
				OldValue: nil,
				NewValue: newV,
			})
		}
	}

	return changes
}

// jsonValuesDiffer returns true if two interface{} values differ by JSON equality.
func jsonValuesDiffer(a, b interface{}) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) != string(bb)
}
