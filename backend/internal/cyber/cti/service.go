package cti

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/auth"
	"github.com/clario360/platform/internal/cyber/service"
	apperrors "github.com/clario360/platform/internal/errors"
	"github.com/clario360/platform/internal/events"
)

// AggEngine is an optional interface for the full aggregation engine.
// When set, the admin refresh endpoint delegates to it instead of raw repo queries.
type AggEngine interface {
	RunFullAggregation(ctx context.Context, tenantID string) error
	RunAllTenants(ctx context.Context) error
}

type AggregationRefreshScope string

const (
	AggregationScopeTenant     AggregationRefreshScope = "tenant"
	AggregationScopeAllTenants AggregationRefreshScope = "all_tenants"
)

// Service implements CTI business logic on top of the repository.
type Service struct {
	repo      Repository
	producer  *events.Producer
	logger    zerolog.Logger
	cache     *refCache
	aggEngine AggEngine
}

func NewService(repo Repository, producer *events.Producer, logger zerolog.Logger) *Service {
	return &Service{repo: repo, producer: producer, logger: logger, cache: newRefCache()}
}

// SetAggregationEngine attaches the full aggregation engine for admin refresh.
func (s *Service) SetAggregationEngine(engine AggEngine) {
	s.aggEngine = engine
}

func ParseAggregationRefreshScope(raw string) (AggregationRefreshScope, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "tenant", "self", "current":
		return AggregationScopeTenant, nil
	case "all", "all_tenants", "all-tenants":
		return AggregationScopeAllTenants, nil
	default:
		return "", apperrors.NewValidation("INVALID_SCOPE", "scope must be one of: tenant, all", map[string]string{
			"scope": "must be 'tenant' or 'all'",
		})
	}
}

// ---------------------------------------------------------------------------
// Context helpers
// ---------------------------------------------------------------------------

func tenantAndUser(ctx context.Context) (uuid.UUID, uuid.UUID, error) {
	t := auth.TenantFromContext(ctx)
	if t == "" {
		return uuid.Nil, uuid.Nil, fmt.Errorf("tenant context required")
	}
	tid, err := uuid.Parse(t)
	if err != nil {
		return uuid.Nil, uuid.Nil, fmt.Errorf("invalid tenant id: %w", err)
	}
	var uid uuid.UUID
	u := auth.UserFromContext(ctx)
	if u != nil {
		uid, _ = uuid.Parse(u.ID)
	}
	return tid, uid, nil
}

func actorFromContext(ctx context.Context) *service.Actor {
	u := auth.UserFromContext(ctx)
	if u == nil {
		return nil
	}
	uid, _ := uuid.Parse(u.ID)
	return &service.Actor{
		UserID:    uid,
		UserName:  u.Email,
		UserEmail: u.Email,
	}
}

// ---------------------------------------------------------------------------
// Event publishing — follows the shared cyber-service pattern from common.go
// ---------------------------------------------------------------------------

// publishDomainEvent publishes to a CTI-specific Kafka topic with actor metadata.
func (s *Service) publishDomainEvent(ctx context.Context, topic, eventType string, tenantID uuid.UUID, data interface{}) {
	if s.producer == nil {
		return
	}
	actor := actorFromContext(ctx)
	event, err := events.NewEvent(eventType, "cyber-service", tenantID.String(), data)
	if err != nil {
		s.logger.Warn().Err(err).Str("type", eventType).Msg("failed to create cti event")
		return
	}
	if actor != nil {
		event.UserID = actor.UserID.String()
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		if actor.UserEmail != "" {
			event.Metadata["user_email"] = actor.UserEmail
		}
	}
	if err := s.producer.Publish(ctx, topic, event); err != nil {
		s.logger.Warn().Err(err).Str("topic", topic).Str("type", eventType).Msg("failed to publish cti event")
	}
}

// publishAuditEvent publishes to the shared audit topic with actor metadata.
func (s *Service) publishAuditEvent(ctx context.Context, eventType string, tenantID uuid.UUID, data interface{}) {
	s.publishDomainEvent(ctx, events.Topics.AuditEvents, eventType, tenantID, data)
}

func (s *Service) publishAlert(ctx context.Context, tenantID uuid.UUID, alert AlertPayload) {
	s.publishDomainEvent(ctx, TopicCTIAlerts, alert.AlertType, tenantID, alert)
}

func (s *Service) triggerAggregation(ctx context.Context, tenantID uuid.UUID, trigger string, entityID string) {
	s.publishDomainEvent(ctx, TopicCTIAggregationTriggers, EventAggregationTriggered, tenantID, map[string]string{
		"trigger":   trigger,
		"entity_id": entityID,
	})
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ---------------------------------------------------------------------------
// Reference data cache (5-minute TTL)
// ---------------------------------------------------------------------------

type refCacheEntry[T any] struct {
	data      []T
	expiresAt time.Time
}

type refCache struct {
	mu         sync.RWMutex
	severities map[uuid.UUID]*refCacheEntry[ThreatSeverityLevel] // keyed by tenantID
	categories map[uuid.UUID]*refCacheEntry[ThreatCategory]
	sectors    map[uuid.UUID]*refCacheEntry[IndustrySector]
	sources    map[uuid.UUID]*refCacheEntry[DataSource]
}

const refCacheTTL = 5 * time.Minute

func newRefCache() *refCache {
	return &refCache{
		severities: make(map[uuid.UUID]*refCacheEntry[ThreatSeverityLevel]),
		categories: make(map[uuid.UUID]*refCacheEntry[ThreatCategory]),
		sectors:    make(map[uuid.UUID]*refCacheEntry[IndustrySector]),
		sources:    make(map[uuid.UUID]*refCacheEntry[DataSource]),
	}
}

func (s *Service) cachedSeverityByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatSeverityLevel, error) {
	items, err := s.cachedSeverities(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Code == code {
			return &items[i], nil
		}
	}
	return nil, apperrors.NewNotFound("CTI_SEVERITY_NOT_FOUND", fmt.Sprintf("severity code %q not found", code))
}

func (s *Service) cachedSeverities(ctx context.Context, tenantID uuid.UUID) ([]ThreatSeverityLevel, error) {
	s.cache.mu.RLock()
	if e, ok := s.cache.severities[tenantID]; ok && time.Now().Before(e.expiresAt) {
		s.cache.mu.RUnlock()
		return e.data, nil
	}
	s.cache.mu.RUnlock()

	items, err := s.repo.ListSeverityLevels(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.severities[tenantID] = &refCacheEntry[ThreatSeverityLevel]{data: items, expiresAt: time.Now().Add(refCacheTTL)}
	s.cache.mu.Unlock()
	return items, nil
}

func (s *Service) cachedCategoryByCode(ctx context.Context, tenantID uuid.UUID, code string) (*ThreatCategory, error) {
	s.cache.mu.RLock()
	if e, ok := s.cache.categories[tenantID]; ok && time.Now().Before(e.expiresAt) {
		s.cache.mu.RUnlock()
		for i := range e.data {
			if e.data[i].Code == code {
				return &e.data[i], nil
			}
		}
		return nil, apperrors.ErrNotFound
	}
	s.cache.mu.RUnlock()

	items, err := s.repo.ListCategories(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.categories[tenantID] = &refCacheEntry[ThreatCategory]{data: items, expiresAt: time.Now().Add(refCacheTTL)}
	s.cache.mu.Unlock()
	for i := range items {
		if items[i].Code == code {
			return &items[i], nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (s *Service) cachedSectorByCode(ctx context.Context, tenantID uuid.UUID, code string) (*IndustrySector, error) {
	s.cache.mu.RLock()
	if e, ok := s.cache.sectors[tenantID]; ok && time.Now().Before(e.expiresAt) {
		s.cache.mu.RUnlock()
		for i := range e.data {
			if e.data[i].Code == code {
				return &e.data[i], nil
			}
		}
		return nil, apperrors.ErrNotFound
	}
	s.cache.mu.RUnlock()

	items, err := s.repo.ListSectors(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.sectors[tenantID] = &refCacheEntry[IndustrySector]{data: items, expiresAt: time.Now().Add(refCacheTTL)}
	s.cache.mu.Unlock()
	for i := range items {
		if items[i].Code == code {
			return &items[i], nil
		}
	}
	return nil, apperrors.ErrNotFound
}

func (s *Service) cachedSourceByName(ctx context.Context, tenantID uuid.UUID, name string) (*DataSource, error) {
	s.cache.mu.RLock()
	if e, ok := s.cache.sources[tenantID]; ok && time.Now().Before(e.expiresAt) {
		s.cache.mu.RUnlock()
		for i := range e.data {
			if e.data[i].Name == name {
				return &e.data[i], nil
			}
		}
		return nil, apperrors.ErrNotFound
	}
	s.cache.mu.RUnlock()

	items, err := s.repo.ListDataSources(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.sources[tenantID] = &refCacheEntry[DataSource]{data: items, expiresAt: time.Now().Add(refCacheTTL)}
	s.cache.mu.Unlock()
	for i := range items {
		if items[i].Name == name {
			return &items[i], nil
		}
	}
	return nil, apperrors.ErrNotFound
}

// ---------------------------------------------------------------------------
// Reference data
// ---------------------------------------------------------------------------

func (s *Service) ListSeverityLevels(ctx context.Context) ([]ThreatSeverityLevel, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSeverityLevels(ctx, tid)
}

func (s *Service) ListCategories(ctx context.Context) ([]ThreatCategory, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListCategories(ctx, tid)
}

func (s *Service) ListRegions(ctx context.Context, parentID *uuid.UUID) ([]GeographicRegion, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListRegions(ctx, tid, parentID)
}

func (s *Service) ListSectors(ctx context.Context) ([]IndustrySector, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListSectors(ctx, tid)
}

func (s *Service) ListDataSources(ctx context.Context) ([]DataSource, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListDataSources(ctx, tid)
}

// ---------------------------------------------------------------------------
// Threat events
// ---------------------------------------------------------------------------

func (s *Service) CreateThreatEvent(ctx context.Context, req CreateThreatEventRequest) (*ThreatEventDetail, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}

	sev, err := s.cachedSeverityByCode(ctx, tid, req.SeverityCode)
	if err != nil {
		return nil, fmt.Errorf("invalid severity_code: %w", err)
	}

	var catID *uuid.UUID
	if req.CategoryCode != nil {
		cat, err := s.cachedCategoryByCode(ctx, tid, *req.CategoryCode)
		if err != nil && !apperrors.IsNotFound(err) {
			return nil, err
		}
		if cat != nil {
			catID = &cat.ID
		}
	}

	var sourceID *uuid.UUID
	if req.SourceName != nil {
		src, err := s.cachedSourceByName(ctx, tid, *req.SourceName)
		if err != nil && !apperrors.IsNotFound(err) {
			return nil, err
		}
		if src != nil {
			sourceID = &src.ID
		}
	}

	var sectorID *uuid.UUID
	if req.TargetSectorCode != nil {
		sec, err := s.cachedSectorByCode(ctx, tid, *req.TargetSectorCode)
		if err != nil && !apperrors.IsNotFound(err) {
			return nil, err
		}
		if sec != nil {
			sectorID = &sec.ID
		}
	}

	now := time.Now().UTC()
	firstSeen := now
	if req.FirstSeenAt != nil {
		firstSeen = *req.FirstSeenAt
	}

	event := &ThreatEvent{
		ID:                uuid.New(),
		TenantID:          tid,
		EventType:         req.EventType,
		Title:             req.Title,
		Description:       req.Description,
		SeverityID:        &sev.ID,
		CategoryID:        catID,
		SourceID:          sourceID,
		SourceReference:   req.SourceReference,
		ConfidenceScore:   req.ConfidenceScore,
		OriginLatitude:    req.OriginLatitude,
		OriginLongitude:   req.OriginLongitude,
		OriginCountryCode: req.OriginCountryCode,
		OriginCity:        req.OriginCity,
		TargetSectorID:    sectorID,
		TargetOrgName:     req.TargetOrgName,
		TargetCountryCode: req.TargetCountryCode,
		IOCType:           req.IOCType,
		IOCValue:          req.IOCValue,
		MitreTechniqueIDs: req.MitreTechniqueIDs,
		RawPayload:        req.RawPayload,
		FirstSeenAt:       firstSeen,
		LastSeenAt:        firstSeen,
		CreatedBy:         &uid,
	}

	if err := s.repo.CreateThreatEvent(ctx, tid, event); err != nil {
		return nil, err
	}

	if len(req.Tags) > 0 {
		_ = s.repo.AddEventTags(ctx, tid, event.ID, req.Tags)
	}

	// Publish to CTI topic
	s.publishDomainEvent(ctx, TopicCTIThreatEvents, EventThreatEventCreated, tid, ThreatEventPayload{
		EventID:         event.ID.String(),
		TenantID:        tid.String(),
		EventType:       event.EventType,
		Title:           event.Title,
		SeverityCode:    req.SeverityCode,
		ConfidenceScore: req.ConfidenceScore,
		OriginCountry:   derefStr(req.OriginCountryCode),
		OriginCity:      derefStr(req.OriginCity),
		IOCType:         derefStr(req.IOCType),
		IOCValue:        derefStr(req.IOCValue),
		Timestamp:       event.FirstSeenAt,
	})
	// Audit log
	s.publishAuditEvent(ctx, EventThreatEventCreated, tid, map[string]interface{}{
		"id": event.ID.String(), "title": event.Title, "severity": req.SeverityCode,
	})

	// Alert escalation for critical events
	if req.SeverityCode == "critical" {
		s.publishAlert(ctx, tid, AlertPayload{
			AlertType:    EventCriticalThreatAlert,
			TenantID:     tid.String(),
			Title:        fmt.Sprintf("CRITICAL: %s", event.Title),
			Description:  fmt.Sprintf("Critical threat event detected: %s", event.EventType),
			SeverityCode: "critical",
			SourceEntity: "threat_event",
			SourceID:     event.ID.String(),
			ActionURL:    fmt.Sprintf("/cyber/cti/events/%s", event.ID.String()),
		})
	}

	// Trigger aggregation refresh
	s.triggerAggregation(ctx, tid, "threat_event_created", event.ID.String())

	return s.repo.GetThreatEvent(ctx, tid, event.ID)
}

func (s *Service) GetThreatEvent(ctx context.Context, eventID uuid.UUID) (*ThreatEventDetail, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	d, err := s.repo.GetThreatEvent(ctx, tid, eventID)
	if err != nil {
		return nil, err
	}
	tags, _ := s.repo.GetEventTags(ctx, tid, eventID)
	d.Tags = tags
	return d, nil
}

func (s *Service) ListThreatEvents(ctx context.Context, f ThreatEventFilters) (*ListResponse[ThreatEventDetail], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListThreatEvents(ctx, tid, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ThreatEventDetail{}
	}
	return &ListResponse[ThreatEventDetail]{
		Data: items,
		Meta: NewPaginationMeta(f.Page, f.PerPage, total),
	}, nil
}

func (s *Service) UpdateThreatEvent(ctx context.Context, eventID uuid.UUID, req UpdateThreatEventRequest) (*ThreatEventDetail, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}

	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ConfidenceScore != nil {
		updates["confidence_score"] = *req.ConfidenceScore
	}
	if req.OriginCountryCode != nil {
		updates["origin_country_code"] = *req.OriginCountryCode
	}
	if req.OriginCity != nil {
		updates["origin_city"] = *req.OriginCity
	}
	if req.TargetCountryCode != nil {
		updates["target_country_code"] = *req.TargetCountryCode
	}
	if req.IOCType != nil {
		updates["ioc_type"] = *req.IOCType
	}
	if req.IOCValue != nil {
		updates["ioc_value"] = *req.IOCValue
	}
	if req.MitreTechniqueIDs != nil {
		updates["mitre_technique_ids"] = req.MitreTechniqueIDs
	}

	if req.SeverityCode != nil {
		sev, err := s.cachedSeverityByCode(ctx, tid, *req.SeverityCode)
		if err != nil {
			return nil, fmt.Errorf("invalid severity_code: %w", err)
		}
		updates["severity_id"] = sev.ID
	}
	if req.CategoryCode != nil {
		cat, err := s.cachedCategoryByCode(ctx, tid, *req.CategoryCode)
		if err != nil && !apperrors.IsNotFound(err) {
			return nil, err
		}
		if cat != nil {
			updates["category_id"] = cat.ID
		}
	}
	if req.TargetSectorCode != nil {
		sec, err := s.cachedSectorByCode(ctx, tid, *req.TargetSectorCode)
		if err != nil && !apperrors.IsNotFound(err) {
			return nil, err
		}
		if sec != nil {
			updates["target_sector_id"] = sec.ID
		}
	}

	updates["updated_by"] = uid

	if err := s.repo.UpdateThreatEvent(ctx, tid, eventID, updates); err != nil {
		return nil, err
	}

	if req.Tags != nil {
		// Replace tags: clear existing then add new
		existing, _ := s.repo.GetEventTags(ctx, tid, eventID)
		for _, t := range existing {
			_ = s.repo.RemoveEventTag(ctx, tid, eventID, t)
		}
		if len(req.Tags) > 0 {
			_ = s.repo.AddEventTags(ctx, tid, eventID, req.Tags)
		}
	}

	s.publishDomainEvent(ctx, TopicCTIThreatEvents, EventThreatEventUpdated, tid, map[string]interface{}{"id": eventID.String()})
	s.publishAuditEvent(ctx, EventThreatEventUpdated, tid, map[string]interface{}{"id": eventID.String()})
	s.triggerAggregation(ctx, tid, "threat_event_updated", eventID.String())

	return s.repo.GetThreatEvent(ctx, tid, eventID)
}

func (s *Service) DeleteThreatEvent(ctx context.Context, eventID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteThreatEvent(ctx, tid, eventID, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTIThreatEvents, EventThreatEventDeleted, tid, map[string]interface{}{"id": eventID.String()})
	s.publishAuditEvent(ctx, EventThreatEventDeleted, tid, map[string]interface{}{"id": eventID.String()})
	s.triggerAggregation(ctx, tid, "threat_event_deleted", eventID.String())
	return nil
}

func (s *Service) MarkEventFalsePositive(ctx context.Context, eventID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.MarkFalsePositive(ctx, tid, eventID, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTIThreatEvents, EventThreatEventFalsePositive, tid, map[string]interface{}{"id": eventID.String()})
	s.publishAuditEvent(ctx, EventThreatEventFalsePositive, tid, map[string]interface{}{"id": eventID.String()})
	s.triggerAggregation(ctx, tid, "threat_event_false_positive", eventID.String())
	return nil
}

func (s *Service) ResolveEvent(ctx context.Context, eventID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.ResolveThreatEvent(ctx, tid, eventID, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTIThreatEvents, EventThreatEventResolved, tid, map[string]interface{}{"id": eventID.String()})
	s.publishAuditEvent(ctx, EventThreatEventResolved, tid, map[string]interface{}{"id": eventID.String()})
	s.triggerAggregation(ctx, tid, "threat_event_resolved", eventID.String())
	return nil
}

// Event tags
func (s *Service) GetEventTags(ctx context.Context, eventID uuid.UUID) ([]string, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetEventTags(ctx, tid, eventID)
}

func (s *Service) AddEventTags(ctx context.Context, eventID uuid.UUID, tags []string) error {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	return s.repo.AddEventTags(ctx, tid, eventID, tags)
}

func (s *Service) RemoveEventTag(ctx context.Context, eventID uuid.UUID, tag string) error {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	return s.repo.RemoveEventTag(ctx, tid, eventID, tag)
}

// ---------------------------------------------------------------------------
// Threat actors
// ---------------------------------------------------------------------------

func (s *Service) CreateThreatActor(ctx context.Context, req CreateThreatActorRequest) (*ThreatActor, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	actor := &ThreatActor{
		ID:                  uuid.New(),
		TenantID:            tid,
		Name:                req.Name,
		Aliases:             req.Aliases,
		ActorType:           req.ActorType,
		OriginCountryCode:   req.OriginCountryCode,
		SophisticationLevel: req.SophisticationLevel,
		PrimaryMotivation:   req.PrimaryMotivation,
		Description:         req.Description,
		MitreGroupID:        req.MitreGroupID,
		ExternalReferences:  req.ExternalReferences,
		IsActive:            true,
		RiskScore:           req.RiskScore,
		FirstObservedAt:     &now,
		LastActivityAt:      &now,
		CreatedBy:           &uid,
	}
	if err := s.repo.CreateThreatActor(ctx, tid, actor); err != nil {
		return nil, err
	}
	s.publishAuditEvent(ctx, "cti.threat_actor.created", tid, map[string]interface{}{
		"id": actor.ID.String(), "name": actor.Name,
	})
	return s.repo.GetThreatActor(ctx, tid, actor.ID)
}

func (s *Service) GetThreatActor(ctx context.Context, actorID uuid.UUID) (*ThreatActor, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetThreatActor(ctx, tid, actorID)
}

func (s *Service) ListThreatActors(ctx context.Context, f ThreatActorFilters) (*ListResponse[ThreatActor], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListThreatActors(ctx, tid, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ThreatActor{}
	}
	return &ListResponse[ThreatActor]{Data: items, Meta: NewPaginationMeta(f.Page, f.PerPage, total)}, nil
}

func (s *Service) UpdateThreatActor(ctx context.Context, actorID uuid.UUID, req UpdateThreatActorRequest) (*ThreatActor, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Aliases != nil {
		updates["aliases"] = req.Aliases
	}
	if req.ActorType != nil {
		updates["actor_type"] = *req.ActorType
	}
	if req.OriginCountryCode != nil {
		updates["origin_country_code"] = *req.OriginCountryCode
	}
	if req.SophisticationLevel != nil {
		updates["sophistication_level"] = *req.SophisticationLevel
	}
	if req.PrimaryMotivation != nil {
		updates["primary_motivation"] = *req.PrimaryMotivation
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.MitreGroupID != nil {
		updates["mitre_group_id"] = *req.MitreGroupID
	}
	if req.RiskScore != nil {
		updates["risk_score"] = *req.RiskScore
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	updates["updated_by"] = uid

	if err := s.repo.UpdateThreatActor(ctx, tid, actorID, updates); err != nil {
		return nil, err
	}
	s.publishAuditEvent(ctx, "cti.threat_actor.updated", tid, map[string]interface{}{"id": actorID.String()})
	return s.repo.GetThreatActor(ctx, tid, actorID)
}

func (s *Service) DeleteThreatActor(ctx context.Context, actorID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteThreatActor(ctx, tid, actorID, uid); err != nil {
		return err
	}
	s.publishAuditEvent(ctx, "cti.threat_actor.deleted", tid, map[string]interface{}{"id": actorID.String()})
	return nil
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

func (s *Service) CreateCampaign(ctx context.Context, req CreateCampaignRequest) (*CampaignDetail, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	sev, err := s.cachedSeverityByCode(ctx, tid, req.SeverityCode)
	if err != nil {
		return nil, fmt.Errorf("invalid severity_code: %w", err)
	}

	var actorID *uuid.UUID
	if req.PrimaryActorID != nil {
		aid, err := uuid.Parse(*req.PrimaryActorID)
		if err != nil {
			return nil, fmt.Errorf("invalid primary_actor_id: %w", err)
		}
		actorID = &aid
	}

	targetSectors, err := ParseUUIDs(req.TargetSectors)
	if err != nil {
		return nil, fmt.Errorf("invalid target sector UUID: %w", err)
	}
	targetRegions, err := ParseUUIDs(req.TargetRegions)
	if err != nil {
		return nil, fmt.Errorf("invalid target region UUID: %w", err)
	}

	c := &Campaign{
		ID:                uuid.New(),
		TenantID:          tid,
		CampaignCode:      req.CampaignCode,
		Name:              req.Name,
		Description:       req.Description,
		Status:            req.Status,
		SeverityID:        &sev.ID,
		PrimaryActorID:    actorID,
		TargetSectors:     targetSectors,
		TargetRegions:     targetRegions,
		TargetDescription: req.TargetDescription,
		MitreTechniqueIDs: req.MitreTechniqueIDs,
		TTPsSummary:       req.TTPsSummary,
		FirstSeenAt:       req.FirstSeenAt,
		CreatedBy:         &uid,
	}
	if err := s.repo.CreateCampaign(ctx, tid, c); err != nil {
		return nil, err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, EventCampaignCreated, tid, CampaignPayload{
		CampaignID:   c.ID.String(),
		TenantID:     tid.String(),
		CampaignCode: c.CampaignCode,
		Name:         c.Name,
		Status:       c.Status,
		SeverityCode: req.SeverityCode,
	})
	s.publishAuditEvent(ctx, EventCampaignCreated, tid, map[string]interface{}{
		"id": c.ID.String(), "name": c.Name, "status": c.Status,
	})
	if req.SeverityCode == "critical" && req.Status == "active" {
		s.publishAlert(ctx, tid, AlertPayload{
			AlertType:    EventCampaignEscalation,
			TenantID:     tid.String(),
			Title:        fmt.Sprintf("CRITICAL CAMPAIGN: %s", c.Name),
			Description:  fmt.Sprintf("Active critical campaign detected: %s", c.CampaignCode),
			SeverityCode: "critical",
			SourceEntity: "campaign",
			SourceID:     c.ID.String(),
			ActionURL:    fmt.Sprintf("/cyber/cti/campaigns/%s", c.ID.String()),
		})
	}
	return s.repo.GetCampaign(ctx, tid, c.ID)
}

func (s *Service) GetCampaign(ctx context.Context, campaignID uuid.UUID) (*CampaignDetail, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetCampaign(ctx, tid, campaignID)
}

func (s *Service) ListCampaigns(ctx context.Context, f CampaignFilters) (*ListResponse[CampaignDetail], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListCampaigns(ctx, tid, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CampaignDetail{}
	}
	return &ListResponse[CampaignDetail]{Data: items, Meta: NewPaginationMeta(f.Page, f.PerPage, total)}, nil
}

func (s *Service) UpdateCampaign(ctx context.Context, campaignID uuid.UUID, req UpdateCampaignRequest) (*CampaignDetail, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	updates := make(map[string]interface{})
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.TargetDescription != nil {
		updates["target_description"] = *req.TargetDescription
	}
	if req.TargetSectors != nil {
		targetSectors, err := ParseUUIDs(req.TargetSectors)
		if err != nil {
			return nil, fmt.Errorf("invalid target sector UUID: %w", err)
		}
		updates["target_sectors"] = targetSectors
	}
	if req.TargetRegions != nil {
		targetRegions, err := ParseUUIDs(req.TargetRegions)
		if err != nil {
			return nil, fmt.Errorf("invalid target region UUID: %w", err)
		}
		updates["target_regions"] = targetRegions
	}
	if req.MitreTechniqueIDs != nil {
		updates["mitre_technique_ids"] = req.MitreTechniqueIDs
	}
	if req.TTPsSummary != nil {
		updates["ttps_summary"] = *req.TTPsSummary
	}
	if req.PrimaryActorID != nil {
		aid, err := uuid.Parse(*req.PrimaryActorID)
		if err != nil {
			return nil, fmt.Errorf("invalid primary_actor_id: %w", err)
		}
		updates["primary_actor_id"] = aid
	}
	if req.SeverityCode != nil {
		sev, err := s.cachedSeverityByCode(ctx, tid, *req.SeverityCode)
		if err != nil {
			return nil, fmt.Errorf("invalid severity_code: %w", err)
		}
		updates["severity_id"] = sev.ID
	}
	updates["updated_by"] = uid

	if err := s.repo.UpdateCampaign(ctx, tid, campaignID, updates); err != nil {
		return nil, err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, EventCampaignUpdated, tid, map[string]interface{}{"id": campaignID.String()})
	s.publishAuditEvent(ctx, EventCampaignUpdated, tid, map[string]interface{}{"id": campaignID.String()})
	return s.repo.GetCampaign(ctx, tid, campaignID)
}

func (s *Service) DeleteCampaign(ctx context.Context, campaignID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteCampaign(ctx, tid, campaignID, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, "cyber.cti.campaign.deleted", tid, map[string]interface{}{"id": campaignID.String()})
	s.publishAuditEvent(ctx, "cyber.cti.campaign.deleted", tid, map[string]interface{}{"id": campaignID.String()})
	return nil
}

func (s *Service) UpdateCampaignStatus(ctx context.Context, campaignID uuid.UUID, status string) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateCampaignStatus(ctx, tid, campaignID, status, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, EventCampaignStatusChanged, tid, map[string]interface{}{
		"id": campaignID.String(), "status": status,
	})
	s.publishAuditEvent(ctx, EventCampaignStatusChanged, tid, map[string]interface{}{
		"id": campaignID.String(), "status": status,
	})
	return nil
}

// Campaign events / IOCs
func (s *Service) LinkEventToCampaign(ctx context.Context, campaignID, eventID uuid.UUID) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.LinkEventToCampaign(ctx, tid, campaignID, eventID, &uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, EventCampaignEventLinked, tid, map[string]interface{}{
		"campaign_id": campaignID.String(), "event_id": eventID.String(),
	})
	s.publishAuditEvent(ctx, EventCampaignEventLinked, tid, map[string]interface{}{
		"campaign_id": campaignID.String(), "event_id": eventID.String(),
	})
	return nil
}

func (s *Service) UnlinkEventFromCampaign(ctx context.Context, campaignID, eventID uuid.UUID) error {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.UnlinkEventFromCampaign(ctx, tid, campaignID, eventID); err != nil {
		return err
	}
	s.publishAuditEvent(ctx, "cyber.cti.campaign.event-unlinked", tid, map[string]interface{}{
		"campaign_id": campaignID.String(), "event_id": eventID.String(),
	})
	return nil
}

func (s *Service) ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, p ListParams) (*ListResponse[ThreatEventDetail], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListCampaignEvents(ctx, tid, campaignID, p)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ThreatEventDetail{}
	}
	return &ListResponse[ThreatEventDetail]{Data: items, Meta: NewPaginationMeta(p.Page, p.PerPage, total)}, nil
}

func (s *Service) CreateCampaignIOC(ctx context.Context, campaignID uuid.UUID, req CreateCampaignIOCRequest) (*CampaignIOC, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	var sourceID *uuid.UUID
	if req.SourceName != nil {
		src, err := s.cachedSourceByName(ctx, tid, *req.SourceName)
		if err == nil {
			sourceID = &src.ID
		}
	}
	now := time.Now().UTC()
	ioc := &CampaignIOC{
		ID:              uuid.New(),
		TenantID:        tid,
		CampaignID:      campaignID,
		IOCType:         req.IOCType,
		IOCValue:        req.IOCValue,
		ConfidenceScore: req.ConfidenceScore,
		FirstSeenAt:     now,
		LastSeenAt:      now,
		IsActive:        true,
		SourceID:        sourceID,
	}
	if err := s.repo.CreateCampaignIOC(ctx, tid, ioc); err != nil {
		return nil, err
	}
	s.publishDomainEvent(ctx, TopicCTICampaigns, EventCampaignIOCAdded, tid, map[string]interface{}{
		"campaign_id": campaignID.String(), "ioc_type": ioc.IOCType, "ioc_value": ioc.IOCValue,
	})
	s.publishAuditEvent(ctx, EventCampaignIOCAdded, tid, map[string]interface{}{
		"campaign_id": campaignID.String(), "ioc_id": ioc.ID.String(),
	})
	return ioc, nil
}

func (s *Service) ListCampaignIOCs(ctx context.Context, campaignID uuid.UUID, p ListParams) (*ListResponse[CampaignIOC], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListCampaignIOCs(ctx, tid, campaignID, p)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []CampaignIOC{}
	}
	return &ListResponse[CampaignIOC]{Data: items, Meta: NewPaginationMeta(p.Page, p.PerPage, total)}, nil
}

func (s *Service) DeleteCampaignIOC(ctx context.Context, iocID uuid.UUID) error {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeleteCampaignIOC(ctx, tid, iocID)
}

// ---------------------------------------------------------------------------
// Brand abuse
// ---------------------------------------------------------------------------

func (s *Service) CreateMonitoredBrand(ctx context.Context, req CreateMonitoredBrandRequest) (*MonitoredBrand, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	brand := &MonitoredBrand{
		ID:            uuid.New(),
		TenantID:      tid,
		BrandName:     req.BrandName,
		DomainPattern: req.DomainPattern,
		Keywords:      req.Keywords,
		IsActive:      true,
		CreatedBy:     &uid,
	}
	if err := s.repo.CreateMonitoredBrand(ctx, tid, brand); err != nil {
		return nil, err
	}
	s.publishAuditEvent(ctx, "cti.brand.created", tid, map[string]interface{}{
		"id": brand.ID.String(), "name": brand.BrandName,
	})
	brands, _ := s.repo.ListMonitoredBrands(ctx, tid)
	for _, b := range brands {
		if b.ID == brand.ID {
			return &b, nil
		}
	}
	return brand, nil
}

func (s *Service) ListMonitoredBrands(ctx context.Context) ([]MonitoredBrand, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.ListMonitoredBrands(ctx, tid)
}

func (s *Service) UpdateMonitoredBrand(ctx context.Context, brandID uuid.UUID, req UpdateMonitoredBrandRequest) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	updates := make(map[string]interface{})
	if req.BrandName != nil {
		updates["brand_name"] = *req.BrandName
	}
	if req.DomainPattern != nil {
		updates["domain_pattern"] = *req.DomainPattern
	}
	if req.Keywords != nil {
		updates["keywords"] = req.Keywords
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	updates["updated_by"] = uid
	return s.repo.UpdateMonitoredBrand(ctx, tid, brandID, updates)
}

func (s *Service) DeleteMonitoredBrand(ctx context.Context, brandID uuid.UUID) error {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	return s.repo.DeleteMonitoredBrand(ctx, tid, brandID)
}

func (s *Service) CreateBrandAbuseIncident(ctx context.Context, req CreateBrandAbuseIncidentRequest) (*BrandAbuseDetail, error) {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	brandID, err := uuid.Parse(req.BrandID)
	if err != nil {
		return nil, fmt.Errorf("invalid brand_id: %w", err)
	}
	var sourceID *uuid.UUID
	if req.SourceName != nil {
		src, err := s.cachedSourceByName(ctx, tid, *req.SourceName)
		if err == nil {
			sourceID = &src.ID
		}
	}
	now := time.Now().UTC()
	inc := &BrandAbuseIncident{
		ID:              uuid.New(),
		TenantID:        tid,
		BrandID:         brandID,
		MaliciousDomain: req.MaliciousDomain,
		AbuseType:       req.AbuseType,
		RiskLevel:       req.RiskLevel,
		SourceID:        sourceID,
		WhoisRegistrant: req.WhoisRegistrant,
		SSLIssuer:       req.SSLIssuer,
		HostingIP:       req.HostingIP,
		HostingASN:      req.HostingASN,
		TakedownStatus:  "detected",
		FirstDetectedAt: now,
		LastDetectedAt:  now,
		CreatedBy:       &uid,
	}
	if err := s.repo.CreateBrandAbuseIncident(ctx, tid, inc); err != nil {
		return nil, err
	}
	s.publishDomainEvent(ctx, TopicCTIBrandAbuse, EventBrandAbuseDetected, tid, BrandAbusePayload{
		IncidentID:      inc.ID.String(),
		TenantID:        tid.String(),
		MaliciousDomain: inc.MaliciousDomain,
		AbuseType:       inc.AbuseType,
		RiskLevel:       inc.RiskLevel,
		TakedownStatus:  inc.TakedownStatus,
	})
	s.publishAuditEvent(ctx, EventBrandAbuseDetected, tid, map[string]interface{}{
		"id": inc.ID.String(), "domain": inc.MaliciousDomain, "risk_level": inc.RiskLevel,
	})
	if inc.RiskLevel == "critical" {
		s.publishAlert(ctx, tid, AlertPayload{
			AlertType:    EventBrandAbuseUrgent,
			TenantID:     tid.String(),
			Title:        fmt.Sprintf("CRITICAL BRAND ABUSE: %s", inc.MaliciousDomain),
			Description:  fmt.Sprintf("Critical brand abuse detected: %s (%s)", inc.AbuseType, inc.MaliciousDomain),
			SeverityCode: "critical",
			SourceEntity: "brand_abuse",
			SourceID:     inc.ID.String(),
			ActionURL:    fmt.Sprintf("/cyber/cti/brand-abuse/%s", inc.ID.String()),
		})
	}
	return s.repo.GetBrandAbuseIncident(ctx, tid, inc.ID)
}

func (s *Service) GetBrandAbuseIncident(ctx context.Context, incidentID uuid.UUID) (*BrandAbuseDetail, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	return s.repo.GetBrandAbuseIncident(ctx, tid, incidentID)
}

func (s *Service) ListBrandAbuseIncidents(ctx context.Context, f BrandAbuseFilters) (*ListResponse[BrandAbuseDetail], error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	items, total, err := s.repo.ListBrandAbuseIncidents(ctx, tid, f)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []BrandAbuseDetail{}
	}
	return &ListResponse[BrandAbuseDetail]{Data: items, Meta: NewPaginationMeta(f.Page, f.PerPage, total)}, nil
}

func (s *Service) UpdateBrandAbuseIncident(ctx context.Context, incidentID uuid.UUID, updates map[string]interface{}) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	updates["updated_by"] = uid
	if err := s.repo.UpdateBrandAbuseIncident(ctx, tid, incidentID, updates); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTIBrandAbuse, EventBrandAbuseUpdated, tid, map[string]interface{}{"id": incidentID.String()})
	s.publishAuditEvent(ctx, EventBrandAbuseUpdated, tid, map[string]interface{}{"id": incidentID.String()})
	return nil
}

func (s *Service) UpdateTakedownStatus(ctx context.Context, incidentID uuid.UUID, status string) error {
	tid, uid, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateTakedownStatus(ctx, tid, incidentID, status, uid); err != nil {
		return err
	}
	s.publishDomainEvent(ctx, TopicCTIBrandAbuse, EventBrandAbuseTakedownChanged, tid, map[string]interface{}{
		"id": incidentID.String(), "status": status,
	})
	s.publishAuditEvent(ctx, EventBrandAbuseTakedownChanged, tid, map[string]interface{}{
		"id": incidentID.String(), "status": status,
	})
	return nil
}

// ---------------------------------------------------------------------------
// Dashboard
// ---------------------------------------------------------------------------

func (s *Service) GetGlobalThreatMap(ctx context.Context, period string) (*GlobalThreatMapResponse, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	hotspots, err := s.repo.GetGeoThreatMap(ctx, tid, period)
	if err != nil {
		return nil, err
	}
	if hotspots == nil {
		hotspots = []GeoThreatSummary{}
	}
	var totalEvents int64
	for _, h := range hotspots {
		totalEvents += int64(h.TotalCount)
	}
	return &GlobalThreatMapResponse{Hotspots: hotspots, TotalEvents: totalEvents, Period: period}, nil
}

func (s *Service) GetSectorThreatOverview(ctx context.Context, period string) (*SectorThreatResponse, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	sectors, err := s.repo.GetSectorThreatSummary(ctx, tid, period)
	if err != nil {
		return nil, err
	}
	if sectors == nil {
		sectors = []SectorThreatSummary{}
	}
	return &SectorThreatResponse{Sectors: sectors, Period: period}, nil
}

func (s *Service) GetExecutiveDashboard(ctx context.Context) (*ExecutiveDashboardResponse, error) {
	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return nil, err
	}
	snapshot, err := s.repo.GetExecutiveSnapshot(ctx, tid)
	if err != nil {
		return nil, err
	}

	campaigns, _, _ := s.repo.ListCampaigns(ctx, tid, CampaignFilters{
		Statuses: []string{"active"}, Sort: "first_seen_at", Order: "desc", Page: 1, PerPage: 5,
	})
	if campaigns == nil {
		campaigns = []CampaignDetail{}
	}

	brandAbuse, _, _ := s.repo.ListBrandAbuseIncidents(ctx, tid, BrandAbuseFilters{
		RiskLevels: []string{"critical"}, Sort: "first_detected_at", Order: "desc", Page: 1, PerPage: 5,
	})
	if brandAbuse == nil {
		brandAbuse = []BrandAbuseDetail{}
	}

	sectors, _ := s.repo.GetSectorThreatSummary(ctx, tid, "7d")
	if sectors == nil {
		sectors = []SectorThreatSummary{}
	}
	if len(sectors) > 5 {
		sectors = sectors[:5]
	}

	events, _, _ := s.repo.ListThreatEvents(ctx, tid, ThreatEventFilters{
		Sort: "first_seen_at", Order: "desc", Page: 1, PerPage: 10,
	})
	if events == nil {
		events = []ThreatEventDetail{}
	}

	return &ExecutiveDashboardResponse{
		Snapshot:       *snapshot,
		TopCampaigns:   campaigns,
		CriticalBrands: brandAbuse,
		TopSectors:     sectors,
		RecentEvents:   events,
	}, nil
}

// ---------------------------------------------------------------------------
// Aggregation refresh
// ---------------------------------------------------------------------------

func (s *Service) RefreshAggregations(ctx context.Context, scope AggregationRefreshScope) error {
	if scope == AggregationScopeAllTenants {
		user := auth.UserFromContext(ctx)
		if user == nil || !auth.HasPermission(user.Roles, auth.PermAdminAll) {
			return apperrors.NewForbidden("FORBIDDEN", "super-admin permission required for all-tenant CTI aggregation refresh")
		}
		if s.aggEngine == nil {
			return apperrors.NewInternal("CTI_AGG_ENGINE_UNAVAILABLE", "cti aggregation engine unavailable", fmt.Errorf("all-tenant refresh requires aggregation engine"))
		}
		return s.aggEngine.RunAllTenants(ctx)
	}

	tid, _, err := tenantAndUser(ctx)
	if err != nil {
		return err
	}

	// Delegate to the full aggregation engine when available — it handles
	// all 4 periods (24h/7d/30d/90d), trend calculation, risk scoring,
	// MTTD/MTTR, and Prometheus metric emission.
	if s.aggEngine != nil {
		return s.aggEngine.RunFullAggregation(ctx, tid.String())
	}

	// Fallback: direct repo queries (no 90d, no trends, no risk score)
	now := time.Now().UTC()
	periods := []struct{ start, end time.Time }{
		{now.Add(-24 * time.Hour), now},
		{now.Add(-7 * 24 * time.Hour), now},
		{now.Add(-30 * 24 * time.Hour), now},
	}
	for _, p := range periods {
		if err := s.repo.RefreshGeoThreatSummary(ctx, tid, p.start, p.end); err != nil {
			s.logger.Error().Err(err).Msg("refresh geo summary failed")
		}
		if err := s.repo.RefreshSectorThreatSummary(ctx, tid, p.start, p.end); err != nil {
			s.logger.Error().Err(err).Msg("refresh sector summary failed")
		}
	}
	return s.repo.RefreshExecutiveSnapshot(ctx, tid)
}
