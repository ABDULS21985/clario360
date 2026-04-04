package feed

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/cti"
	"github.com/clario360/platform/internal/cyber/cti/feed/adapters"
	"github.com/clario360/platform/internal/events"
)

// Ingester processes FeedIngestionPayload messages from the feed-ingestion Kafka topic.
// It normalizes raw data, enriches indicators, persists them, and publishes downstream events.
type Ingester struct {
	repo       cti.Repository
	normalizer *Normalizer
	enricher   *Enricher
	producer   *events.Producer
	logger     zerolog.Logger
}

func NewIngester(repo cti.Repository, normalizer *Normalizer, enricher *Enricher, producer *events.Producer, logger zerolog.Logger) *Ingester {
	return &Ingester{
		repo:       repo,
		normalizer: normalizer,
		enricher:   enricher,
		producer:   producer,
		logger:     logger.With().Str("component", "cti-ingester").Logger(),
	}
}

// HandleFeedEvent implements events.EventHandler for the feed-ingestion topic.
func (ing *Ingester) HandleFeedEvent(ctx context.Context, event *events.Event) error {
	var payload cti.FeedIngestionPayload
	if err := json.Unmarshal(event.Data, &payload); err != nil {
		return err
	}

	tenantID, err := uuid.Parse(payload.TenantID)
	if err != nil {
		return err
	}

	// Look up source
	src, err := ing.repo.GetSourceByName(ctx, tenantID, payload.SourceName)
	if err != nil {
		ing.logger.Warn().Str("source", payload.SourceName).Msg("unknown data source, using generic adapter")
	}

	sourceType := payload.SourceType
	if src != nil {
		sourceType = src.SourceType
	}

	// Normalize
	indicators, err := ing.normalizer.Normalize(ctx, sourceType, payload.RawData)
	if err != nil {
		return err
	}

	// Enrich + persist (idempotent: deduplicate by source_reference)
	created, updated, correlated := 0, 0, 0
	for i := range indicators {
		ing.enricher.Enrich(&indicators[i])

		// Duplicate detection: check if this indicator was already ingested
		if src != nil && indicators[i].ExternalRef != "" {
			existing, _ := ing.repo.FindThreatEventBySourceRef(ctx, tenantID, src.ID, indicators[i].ExternalRef)
			if existing != nil {
				// Already ingested — update last_seen_at only
				_ = ing.repo.UpdateThreatEventLastSeen(ctx, tenantID, existing.ID)
				updated++
				continue
			}
		}

		eventID, err := ing.persistIndicator(ctx, tenantID, src, &indicators[i])
		if err != nil {
			ing.logger.Warn().Err(err).Str("ref", indicators[i].ExternalRef).Msg("persist indicator failed")
			continue
		}
		created++

		// IOC correlation: check if this IOC matches a known campaign
		if indicators[i].IOCType != "" && indicators[i].IOCValue != "" {
			matches, _ := ing.repo.FindMatchingCampaignIOCs(ctx, tenantID, indicators[i].IOCType, indicators[i].IOCValue)
			if len(matches) > 0 {
				correlated++
				for _, m := range matches {
					_ = ing.repo.LinkEventToCampaign(ctx, tenantID, m.CampaignID, eventID, nil)
				}
			}
		}
	}

	ing.logger.Info().
		Str("source", payload.SourceName).
		Int("total", len(indicators)).
		Int("created", created).
		Int("deduplicated", updated).
		Int("correlated", correlated).
		Msg("feed ingestion batch complete")

	return nil
}

func (ing *Ingester) persistIndicator(ctx context.Context, tenantID uuid.UUID, src *cti.DataSource, ind *adapters.NormalizedIndicator) (uuid.UUID, error) {
	// Resolve severity
	sev, err := ing.repo.GetSeverityByCode(ctx, tenantID, ind.SeverityCode)
	if err != nil {
		return uuid.Nil, err
	}

	var catID *uuid.UUID
	if ind.CategoryCode != "" {
		cat, err := ing.repo.GetCategoryByCode(ctx, tenantID, ind.CategoryCode)
		if err == nil {
			catID = &cat.ID
		}
	}

	var sectorID *uuid.UUID
	if ind.TargetSectorCode != "" {
		sec, err := ing.repo.GetSectorByCode(ctx, tenantID, ind.TargetSectorCode)
		if err == nil {
			sectorID = &sec.ID
		}
	}

	var sourceID *uuid.UUID
	if src != nil {
		sourceID = &src.ID
	}

	firstSeen := ind.FirstSeen
	if firstSeen.IsZero() {
		firstSeen = time.Now().UTC()
	}

	var countryPtr, cityPtr, iocTypePtr, iocValuePtr *string
	if ind.OriginCountryCode != "" {
		countryPtr = &ind.OriginCountryCode
	}
	if ind.OriginCity != "" {
		cityPtr = &ind.OriginCity
	}
	if ind.IOCType != "" {
		iocTypePtr = &ind.IOCType
	}
	if ind.IOCValue != "" {
		iocValuePtr = &ind.IOCValue
	}
	var latPtr, lngPtr *float64
	if ind.Latitude != 0 {
		latPtr = &ind.Latitude
	}
	if ind.Longitude != 0 {
		lngPtr = &ind.Longitude
	}

	var refPtr *string
	if ind.ExternalRef != "" {
		refPtr = &ind.ExternalRef
	}

	event := cti.ThreatEvent{
		ID:                uuid.New(),
		TenantID:          tenantID,
		EventType:         "indicator_sighting",
		Title:             ind.Title,
		SeverityID:        &sev.ID,
		CategoryID:        catID,
		SourceID:          sourceID,
		SourceReference:   refPtr,
		ConfidenceScore:   ind.ConfidenceScore,
		OriginLatitude:    latPtr,
		OriginLongitude:   lngPtr,
		OriginCountryCode: countryPtr,
		OriginCity:        cityPtr,
		TargetSectorID:    sectorID,
		IOCType:           iocTypePtr,
		IOCValue:          iocValuePtr,
		MitreTechniqueIDs: ind.MITRETechniques,
		FirstSeenAt:       firstSeen,
		LastSeenAt:        ind.LastSeen,
	}

	if ind.Description != "" {
		event.Description = &ind.Description
	}

	if err := ing.repo.CreateThreatEvent(ctx, tenantID, &event); err != nil {
		return uuid.Nil, err
	}

	if len(ind.Tags) > 0 {
		_ = ing.repo.AddEventTags(ctx, tenantID, event.ID, ind.Tags)
	}

	// Publish downstream
	if ing.producer != nil {
		evt, _ := events.NewEvent(cti.EventThreatEventCreated, "cyber-service/cti-ingester", tenantID.String(), cti.ThreatEventPayload{
			EventID:         event.ID.String(),
			TenantID:        tenantID.String(),
			EventType:       event.EventType,
			Title:           event.Title,
			SeverityCode:    ind.SeverityCode,
			ConfidenceScore: ind.ConfidenceScore,
			OriginCountry:   ind.OriginCountryCode,
			OriginCity:      ind.OriginCity,
			Timestamp:       event.FirstSeenAt,
		})
		if evt != nil {
			_ = ing.producer.Publish(ctx, cti.TopicCTIThreatEvents, evt)
		}
	}

	return event.ID, nil
}
