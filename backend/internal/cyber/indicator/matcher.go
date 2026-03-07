package indicator

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// IndicatorRepository defines the repository methods needed by the matcher.
type IndicatorRepository interface {
	ListActiveByTenant(ctx context.Context, tenantID uuid.UUID) ([]*model.ThreatIndicator, error)
}

// Matcher performs O(1) IOC lookups against normalized security events.
type Matcher struct {
	repo   IndicatorRepository
	logger zerolog.Logger

	mu      sync.RWMutex
	tenants map[uuid.UUID]*tenantIndicatorSet
}

type tenantIndicatorSet struct {
	loadedAt   time.Time
	ipSet      map[string][]*model.ThreatIndicator
	cidrNets   []*CIDRIndicator
	domains    map[string][]*model.ThreatIndicator
	hashes     map[string][]*model.ThreatIndicator
	userAgents map[string][]*model.ThreatIndicator
}

// CIDRIndicator pairs a parsed network with its indicator record.
type CIDRIndicator struct {
	Network   *net.IPNet
	Indicator *model.ThreatIndicator
}

// NewMatcher creates a new tenant-aware indicator matcher.
func NewMatcher(repo IndicatorRepository, logger zerolog.Logger) *Matcher {
	return &Matcher{
		repo:    repo,
		logger:  logger,
		tenants: make(map[uuid.UUID]*tenantIndicatorSet),
	}
}

// Load refreshes the in-memory indicator index for a tenant.
func (m *Matcher) Load(ctx context.Context, tenantID uuid.UUID) error {
	indicators, err := m.repo.ListActiveByTenant(ctx, tenantID)
	if err != nil {
		return err
	}
	set := &tenantIndicatorSet{
		loadedAt:   time.Now().UTC(),
		ipSet:      make(map[string][]*model.ThreatIndicator),
		cidrNets:   make([]*CIDRIndicator, 0),
		domains:    make(map[string][]*model.ThreatIndicator),
		hashes:     make(map[string][]*model.ThreatIndicator),
		userAgents: make(map[string][]*model.ThreatIndicator),
	}
	for _, indicator := range indicators {
		normalizedValue := strings.ToLower(strings.TrimSpace(indicator.Value))
		switch indicator.Type {
		case model.IndicatorTypeIP:
			set.ipSet[normalizedValue] = append(set.ipSet[normalizedValue], indicator)
		case model.IndicatorTypeCIDR:
			if _, network, err := net.ParseCIDR(indicator.Value); err == nil {
				set.cidrNets = append(set.cidrNets, &CIDRIndicator{Network: network, Indicator: indicator})
			}
		case model.IndicatorTypeDomain, model.IndicatorTypeURL:
			set.domains[normalizedValue] = append(set.domains[normalizedValue], indicator)
		case model.IndicatorTypeHashMD5, model.IndicatorTypeHashSHA1, model.IndicatorTypeHashSHA256:
			set.hashes[normalizedValue] = append(set.hashes[normalizedValue], indicator)
		case model.IndicatorTypeUserAgent:
			set.userAgents[normalizedValue] = append(set.userAgents[normalizedValue], indicator)
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tenants[tenantID] = set
	return nil
}

// Match evaluates one event against the loaded indicators for its tenant.
func (m *Matcher) Match(event *model.SecurityEvent) []*model.IndicatorMatch {
	if event == nil {
		return nil
	}
	m.mu.RLock()
	set := m.tenants[event.TenantID]
	m.mu.RUnlock()
	if set == nil {
		return nil
	}

	matches := make([]*model.IndicatorMatch, 0, 4)
	if event.SourceIP != nil {
		matches = append(matches, m.matchIP(set, "source_ip", *event.SourceIP)...)
	}
	if event.DestIP != nil {
		matches = append(matches, m.matchIP(set, "dest_ip", *event.DestIP)...)
	}
	raw := event.RawMap()
	if domain, ok := raw["domain"].(string); ok {
		matches = append(matches, buildIndicatorMatches(set.domains[strings.ToLower(domain)], "raw.domain", domain)...)
	}
	if url, ok := raw["url"].(string); ok {
		matches = append(matches, buildIndicatorMatches(set.domains[strings.ToLower(url)], "raw.url", url)...)
	}
	if event.FileHash != nil {
		hash := strings.ToLower(*event.FileHash)
		matches = append(matches, buildIndicatorMatches(set.hashes[hash], "file_hash", *event.FileHash)...)
	}
	if userAgent, ok := raw["user_agent"].(string); ok {
		matches = append(matches, buildIndicatorMatches(set.userAgents[strings.ToLower(userAgent)], "raw.user_agent", userAgent)...)
	}
	return matches
}

func (m *Matcher) matchIP(set *tenantIndicatorSet, field, value string) []*model.IndicatorMatch {
	matches := buildIndicatorMatches(set.ipSet[strings.ToLower(value)], field, value)
	ip := net.ParseIP(value)
	if ip == nil {
		return matches
	}
	for _, cidr := range set.cidrNets {
		if cidr.Network.Contains(ip) {
			matches = append(matches, &model.IndicatorMatch{
				Indicator: cidr.Indicator,
				Field:     field,
				Value:     value,
			})
		}
	}
	return matches
}

func buildIndicatorMatches(indicators []*model.ThreatIndicator, field, value string) []*model.IndicatorMatch {
	if len(indicators) == 0 {
		return nil
	}
	matches := make([]*model.IndicatorMatch, 0, len(indicators))
	for _, indicator := range indicators {
		matches = append(matches, &model.IndicatorMatch{
			Indicator: indicator,
			Field:     field,
			Value:     value,
		})
	}
	return matches
}
