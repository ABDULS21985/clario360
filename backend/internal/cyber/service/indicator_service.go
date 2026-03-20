package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/mail"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/clario360/platform/internal/cyber/dto"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/cyber/repository"
	"github.com/clario360/platform/internal/events"
)

var (
	domainPattern = regexp.MustCompile(`^(?i)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$`)
	md5Pattern    = regexp.MustCompile(`^[a-fA-F0-9]{32}$`)
	sha1Pattern   = regexp.MustCompile(`^[a-fA-F0-9]{40}$`)
	sha256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)
)

type geoLookupResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message"`
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	Timezone    string  `json:"timezone"`
	AS          string  `json:"as"`
}

// CreateIndicator creates or upserts a standalone IOC.
func (s *ThreatService) CreateIndicator(ctx context.Context, tenantID, userID uuid.UUID, actor *Actor, req *dto.StandaloneIndicatorRequest) (*model.ThreatIndicator, error) {
	if err := s.validateStandaloneIndicator(ctx, tenantID, req); err != nil {
		return nil, err
	}

	item, err := s.indicatorRepo.Create(ctx, &model.ThreatIndicator{
		TenantID:    tenantID,
		ThreatID:    req.ThreatID,
		Type:        req.Type,
		Value:       strings.TrimSpace(req.Value),
		Description: strings.TrimSpace(req.Description),
		Severity:    req.Severity,
		Source:      coalesceSource(req.Source),
		Confidence:  normalizeIndicatorConfidence(req.Confidence),
		Active:      true,
		ExpiresAt:   req.ExpiresAt,
		Tags:        normalizeStrings(req.Tags),
		Metadata:    req.Metadata,
		CreatedBy:   &userID,
	})
	if err != nil {
		return nil, err
	}

	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.indicator.created", tenantID, actor, map[string]interface{}{
		"indicator_id": item.ID.String(),
		"threat_id":    uuidPtrString(item.ThreatID),
		"type":         item.Type,
		"value":        item.Value,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.created", tenantID, actor, item)

	return item, nil
}

// GetIndicator loads one IOC with any linked threat context.
func (s *ThreatService) GetIndicator(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor) (*model.ThreatIndicator, error) {
	item, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.viewed", tenantID, actor, map[string]interface{}{
		"id": item.ID.String(),
	})
	return item, nil
}

// UpdateIndicator edits a standalone IOC.
func (s *ThreatService) UpdateIndicator(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor, req *dto.StandaloneIndicatorRequest) (*model.ThreatIndicator, error) {
	if err := s.validateStandaloneIndicator(ctx, tenantID, req); err != nil {
		return nil, err
	}

	before, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return nil, err
	}

	after, err := s.indicatorRepo.Update(ctx, &model.ThreatIndicator{
		ID:          indicatorID,
		TenantID:    tenantID,
		ThreatID:    req.ThreatID,
		Type:        req.Type,
		Value:       strings.TrimSpace(req.Value),
		Description: strings.TrimSpace(req.Description),
		Severity:    req.Severity,
		Source:      coalesceSource(req.Source),
		Confidence:  normalizeIndicatorConfidence(req.Confidence),
		Active:      before.Active,
		FirstSeenAt: before.FirstSeenAt,
		LastSeenAt:  time.Now().UTC(),
		ExpiresAt:   req.ExpiresAt,
		Tags:        normalizeStrings(req.Tags),
		Metadata:    req.Metadata,
		CreatedBy:   before.CreatedBy,
		CreatedAt:   before.CreatedAt,
	})
	if err != nil {
		return nil, err
	}

	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.indicator.updated", tenantID, actor, map[string]interface{}{
		"indicator_id": after.ID.String(),
		"threat_id":    uuidPtrString(after.ThreatID),
		"type":         after.Type,
		"value":        after.Value,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.updated", tenantID, actor, map[string]interface{}{
		"before": before,
		"after":  after,
	})

	return after, nil
}

// DeleteIndicator removes an IOC from the tenant.
func (s *ThreatService) DeleteIndicator(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor) error {
	item, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return err
	}
	if err := s.indicatorRepo.Delete(ctx, tenantID, indicatorID); err != nil {
		return err
	}

	_ = publishEvent(ctx, s.producer, events.Topics.ThreatEvents, "cyber.indicator.updated", tenantID, actor, map[string]interface{}{
		"indicator_id": indicatorID.String(),
		"deleted":      true,
		"value":        item.Value,
	})
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.deleted", tenantID, actor, map[string]interface{}{
		"id":    indicatorID.String(),
		"value": item.Value,
	})
	return nil
}

// IndicatorStats returns aggregated IOC metrics.
func (s *ThreatService) IndicatorStats(ctx context.Context, tenantID uuid.UUID, actor *Actor) (*model.IndicatorStats, error) {
	stats, err := s.indicatorRepo.Stats(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.stats_viewed", tenantID, actor, map[string]interface{}{})
	return stats, nil
}

// IndicatorEnrichment returns best-effort enrichment data for one IOC.
func (s *ThreatService) IndicatorEnrichment(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor) (*model.IndicatorEnrichment, error) {
	item, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return nil, err
	}

	result := &model.IndicatorEnrichment{}
	if len(item.Metadata) > 0 {
		var meta map[string]interface{}
		if err := json.Unmarshal(item.Metadata, &meta); err == nil {
			result.Metadata = meta
			if score, ok := floatFromAny(meta["reputation_score"]); ok {
				result.ReputationScore = &score
			}
			if cves, ok := stringSliceFromAny(meta["cves"]); ok {
				result.CVEs = cves
			}
		}
	}

	switch item.Type {
	case model.IndicatorTypeIP:
		result.DNS = enrichIPDNS(ctx, item.Value)
		if geo := enrichGeolocation(ctx, item.Value); geo != nil {
			result.Geolocation = geo
		}
	case model.IndicatorTypeCIDR:
		if slash := strings.Index(item.Value, "/"); slash > 0 {
			if geo := enrichGeolocation(ctx, item.Value[:slash]); geo != nil {
				result.Geolocation = geo
			}
		}
	case model.IndicatorTypeDomain:
		result.DNS = enrichDomainDNS(ctx, item.Value)
		result.WHOIS = map[string]interface{}{
			"domain": item.Value,
			"note":   "WHOIS lookup not configured",
		}
	case model.IndicatorTypeURL:
		if parsed, err := neturl.Parse(item.Value); err == nil && parsed.Hostname() != "" {
			result.DNS = enrichDomainDNS(ctx, parsed.Hostname())
		}
		result.WHOIS = map[string]interface{}{
			"url":  item.Value,
			"note": "WHOIS lookup not configured",
		}
	}

	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.enrichment_viewed", tenantID, actor, map[string]interface{}{
		"id": indicatorID.String(),
	})
	return result, nil
}

// IndicatorMatches returns recent alerts and events related to an IOC.
func (s *ThreatService) IndicatorMatches(ctx context.Context, tenantID, indicatorID uuid.UUID, actor *Actor) ([]*model.IndicatorDetectionMatch, error) {
	item, err := s.indicatorRepo.GetByID(ctx, tenantID, indicatorID)
	if err != nil {
		return nil, err
	}
	matches, err := s.indicatorRepo.ListMatches(ctx, tenantID, item, 25)
	if err != nil {
		return nil, err
	}
	_ = publishAuditEvent(ctx, s.producer, "cyber.indicator.matches_viewed", tenantID, actor, map[string]interface{}{
		"id":    indicatorID.String(),
		"count": len(matches),
	})
	return matches, nil
}

func (s *ThreatService) validateStandaloneIndicator(ctx context.Context, tenantID uuid.UUID, req *dto.StandaloneIndicatorRequest) error {
	if req == nil {
		return repository.ErrInvalidInput
	}
	if !req.Type.IsValid() || !req.Severity.IsValid() || req.Severity == model.SeverityInfo {
		return repository.ErrInvalidInput
	}
	switch coalesceSource(req.Source) {
	case "manual", "stix_feed", "osint", "internal", "vendor":
	default:
		return repository.ErrInvalidInput
	}
	if err := validateIndicatorValue(req.Type, strings.TrimSpace(req.Value)); err != nil {
		return err
	}
	if req.ThreatID != nil {
		if _, err := s.threatRepo.GetByID(ctx, tenantID, *req.ThreatID); err != nil {
			return err
		}
	}
	return nil
}

func validateIndicatorValue(indicatorType model.IndicatorType, value string) error {
	if value == "" {
		return repository.ErrInvalidInput
	}
	switch indicatorType {
	case model.IndicatorTypeIP:
		if net.ParseIP(value) == nil {
			return fmt.Errorf("invalid IP indicator value")
		}
	case model.IndicatorTypeCIDR:
		if _, _, err := net.ParseCIDR(value); err != nil {
			return fmt.Errorf("invalid CIDR indicator value")
		}
	case model.IndicatorTypeDomain:
		if !domainPattern.MatchString(value) {
			return fmt.Errorf("invalid domain indicator value")
		}
	case model.IndicatorTypeURL:
		parsed, err := neturl.ParseRequestURI(value)
		if err != nil || parsed.Host == "" {
			return fmt.Errorf("invalid URL indicator value")
		}
	case model.IndicatorTypeEmail:
		if _, err := mail.ParseAddress(value); err != nil {
			return fmt.Errorf("invalid email indicator value")
		}
	case model.IndicatorTypeHashMD5:
		if !md5Pattern.MatchString(value) {
			return fmt.Errorf("invalid MD5 indicator value")
		}
	case model.IndicatorTypeHashSHA1:
		if !sha1Pattern.MatchString(value) {
			return fmt.Errorf("invalid SHA1 indicator value")
		}
	case model.IndicatorTypeHashSHA256:
		if !sha256Pattern.MatchString(value) {
			return fmt.Errorf("invalid SHA256 indicator value")
		}
	case model.IndicatorTypeCertificate, model.IndicatorTypeRegistryKey, model.IndicatorTypeUserAgent:
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("indicator value is required")
		}
	}
	return nil
}

func enrichIPDNS(ctx context.Context, value string) map[string]interface{} {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	names, err := net.DefaultResolver.LookupAddr(ctx, value)
	if err != nil || len(names) == 0 {
		return nil
	}
	return map[string]interface{}{
		"reverse": names,
	}
}

func enrichDomainDNS(ctx context.Context, value string) map[string]interface{} {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	hosts, err := net.DefaultResolver.LookupHost(ctx, value)
	if err != nil || len(hosts) == 0 {
		return nil
	}
	return map[string]interface{}{
		"addresses": hosts,
	}
}

func enrichGeolocation(ctx context.Context, value string) map[string]interface{} {
	ip := net.ParseIP(value)
	if ip == nil || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ip-api.com/json/"+value+"?fields=status,message,country,countryCode,regionName,city,lat,lon,isp,org,timezone,as", nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil
	}

	var geo geoLookupResponse
	if err := json.Unmarshal(body, &geo); err != nil || geo.Status != "success" {
		return nil
	}

	return map[string]interface{}{
		"country":      geo.Country,
		"country_code": geo.CountryCode,
		"region":       geo.RegionName,
		"city":         geo.City,
		"latitude":     geo.Lat,
		"longitude":    geo.Lon,
		"isp":          geo.ISP,
		"org":          geo.Org,
		"asn":          geo.AS,
		"timezone":     geo.Timezone,
	}
}

func floatFromAny(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	default:
		return 0, false
	}
}

func stringSliceFromAny(value interface{}) ([]string, bool) {
	raw, ok := value.([]interface{})
	if !ok {
		return nil, false
	}
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		text, ok := item.(string)
		if ok && strings.TrimSpace(text) != "" {
			items = append(items, strings.TrimSpace(text))
		}
	}
	return items, len(items) > 0
}
