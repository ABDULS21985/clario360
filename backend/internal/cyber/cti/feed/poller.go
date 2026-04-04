package feed

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/cti"
	"github.com/clario360/platform/internal/events"
)

// MultiTenantPoller coordinates polling for every tenant that has active CTI data sources.
type MultiTenantPoller struct {
	repo     cti.Repository
	producer *events.Producer
	interval time.Duration
	logger   zerolog.Logger
}

func NewMultiTenantPoller(repo cti.Repository, producer *events.Producer, interval time.Duration, logger zerolog.Logger) *MultiTenantPoller {
	return &MultiTenantPoller{
		repo:     repo,
		producer: producer,
		interval: interval,
		logger:   logger.With().Str("component", "cti-feed-poller").Logger(),
	}
}

func (p *MultiTenantPoller) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	p.logger.Info().Dur("interval", p.interval).Msg("CTI feed multi-tenant poller started")
	p.pollTenants(ctx)
	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("CTI feed multi-tenant poller stopped")
			return ctx.Err()
		case <-ticker.C:
			p.pollTenants(ctx)
		}
	}
}

func (p *MultiTenantPoller) pollTenants(ctx context.Context) {
	tenants, err := p.repo.ListPollingTenants(ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("list CTI polling tenants")
		return
	}

	for _, tenantID := range tenants {
		poller := NewPoller(p.repo, p.producer, tenantID, p.interval, p.logger)
		poller.pollAll(ctx)
	}
}

// Poller periodically checks active CTI data sources and publishes raw feed data
// to the feed-ingestion Kafka topic for downstream processing.
type Poller struct {
	repo       cti.Repository
	producer   *events.Producer
	httpClient *http.Client
	tenantID   uuid.UUID
	interval   time.Duration
	logger     zerolog.Logger
}

func NewPoller(repo cti.Repository, producer *events.Producer, tenantID uuid.UUID, interval time.Duration, logger zerolog.Logger) *Poller {
	return &Poller{
		repo:       repo,
		producer:   producer,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		tenantID:   tenantID,
		interval:   interval,
		logger:     logger.With().Str("component", "cti-feed-poller").Logger(),
	}
}

// Run starts the polling loop. Blocks until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) error {
	p.logger.Info().Dur("interval", p.interval).Msg("CTI feed poller started")
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	p.pollAll(ctx)

	for {
		select {
		case <-ctx.Done():
			p.logger.Info().Msg("CTI feed poller stopped")
			return ctx.Err()
		case <-ticker.C:
			p.pollAll(ctx)
		}
	}
}

func (p *Poller) pollAll(ctx context.Context) {
	sources, err := p.repo.ListDataSources(ctx, p.tenantID)
	if err != nil {
		p.logger.Error().Err(err).Msg("list data sources for polling")
		return
	}

	now := time.Now().UTC()
	polled := 0
	for _, src := range sources {
		if !src.IsActive {
			continue
		}
		if src.URL == nil || *src.URL == "" {
			continue
		}
		// Check if poll interval has elapsed
		if src.PollIntervalSecs != nil && *src.PollIntervalSecs > 0 && src.LastPolledAt != nil {
			if now.Sub(*src.LastPolledAt) < time.Duration(*src.PollIntervalSecs)*time.Second {
				continue
			}
		}

		if err := p.pollSource(ctx, src); err != nil {
			p.logger.Warn().Err(err).Str("source", src.Name).Msg("poll source failed")
			continue
		}
		polled++
	}

	if polled > 0 {
		p.logger.Info().Int("sources_polled", polled).Msg("poll cycle complete")
	}
}

func (p *Poller) pollSource(ctx context.Context, src cti.DataSource) error {
	req, err := http.NewRequestWithContext(ctx, "GET", *src.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil // skip non-OK responses silently
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20)) // 10 MB max
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}

	// Validate that the response is JSON — skip HTML error pages, redirects, etc.
	body = bytes.TrimSpace(body)
	if len(body) == 0 || (body[0] != '{' && body[0] != '[') {
		p.logger.Debug().Str("source", src.Name).Msg("skipping non-JSON response from feed")
		return nil
	}

	// Publish raw data to the feed-ingestion topic
	payload := cti.FeedIngestionPayload{
		SourceID:   src.ID.String(),
		SourceName: src.Name,
		SourceType: src.SourceType,
		TenantID:   p.tenantID.String(),
		RawData:    body,
		ReceivedAt: time.Now().UTC(),
	}

	evt, err := events.NewEvent(cti.EventFeedRawIngested, "cyber-service/cti-poller", p.tenantID.String(), payload)
	if err != nil {
		return err
	}
	if err := p.producer.Publish(ctx, cti.TopicCTIFeedIngestion, evt); err != nil {
		return err
	}
	return p.repo.UpdateDataSourceLastPolled(ctx, p.tenantID, src.ID, payload.ReceivedAt)
}
