package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// ipAPIBaseURL is the ip-api.com free JSON endpoint.
// Free tier limit: 45 requests/minute from a single public IP.
// Supports only HTTP (not HTTPS) on the free plan.
const ipAPIBaseURL = "http://ip-api.com/json"

// ipAPIFields selects only the fields we need to minimise response payload.
const ipAPIFields = "status,message,country,countryCode,region,regionName,city,lat,lon,isp,org,timezone"

// ipAPIResponse maps the ip-api.com JSON response.
type ipAPIResponse struct {
	Status      string  `json:"status"`
	Message     string  `json:"message"` // populated on failure
	Country     string  `json:"country"`
	CountryCode string  `json:"countryCode"`
	Region      string  `json:"region"`
	RegionName  string  `json:"regionName"`
	City        string  `json:"city"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	ISP         string  `json:"isp"`
	Org         string  `json:"org"`
	Timezone    string  `json:"timezone"`
}

// GeoEnricher enriches assets with geolocation data from ip-api.com.
// When enabled=false (default) it is a no-op so the service can run in air-gapped
// environments. The dbPath parameter is accepted for interface compatibility with
// the MaxMind GeoLite2 file-based path but is not used by this HTTP implementation.
type GeoEnricher struct {
	enabled bool
	client  *http.Client
	logger  zerolog.Logger
}

// NewGeoEnricher creates a geolocation enricher.
// When enabled=false the enricher is a no-op and makes no outbound requests.
func NewGeoEnricher(logger zerolog.Logger, _ string, enabled bool) *GeoEnricher {
	return &GeoEnricher{
		enabled: enabled,
		client: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// Name implements Enricher.
func (e *GeoEnricher) Name() string { return "geolocation" }

// Enrich performs an ip-api.com lookup for the asset's IP address and merges the
// resulting country, city, latitude/longitude, ISP, and timezone fields into the
// asset's metadata JSONB blob.
//
// Private/loopback addresses and assets without an IP are silently skipped since
// geolocation data is only meaningful for public IP addresses.
func (e *GeoEnricher) Enrich(ctx context.Context, asset *model.Asset) (*EnrichmentResult, error) {
	result := &EnrichmentResult{EnricherName: e.Name()}

	if !e.enabled {
		return result, nil
	}
	if asset.IPAddress == nil {
		return result, nil
	}

	ip := net.ParseIP(*asset.IPAddress)
	if ip == nil {
		return result, fmt.Errorf("invalid IP address: %s", *asset.IPAddress)
	}
	// Skip RFC-1918 / loopback / link-local — no public geo data available.
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return result, nil
	}

	geoData, err := e.fetchGeoData(ctx, *asset.IPAddress)
	if err != nil {
		e.logger.Warn().
			Err(err).
			Str("asset_id", asset.ID.String()).
			Str("ip", *asset.IPAddress).
			Msg("geolocation lookup failed")
		return result, err
	}

	// Deserialise existing metadata so we can merge rather than overwrite.
	var meta map[string]any
	if len(asset.Metadata) > 0 {
		if err := json.Unmarshal(asset.Metadata, &meta); err != nil {
			// Corrupt metadata should not block enrichment; start fresh.
			e.logger.Warn().Err(err).Str("asset_id", asset.ID.String()).Msg("failed to parse existing metadata for geo merge; resetting")
			meta = make(map[string]any)
		}
	} else {
		meta = make(map[string]any)
	}

	meta["geo"] = map[string]any{
		"country":      geoData.Country,
		"country_code": geoData.CountryCode,
		"region":       geoData.RegionName,
		"city":         geoData.City,
		"latitude":     geoData.Lat,
		"longitude":    geoData.Lon,
		"isp":          geoData.ISP,
		"org":          geoData.Org,
		"timezone":     geoData.Timezone,
	}
	meta["geo_enriched"] = true

	updated, err := json.Marshal(meta)
	if err != nil {
		return result, fmt.Errorf("marshal enriched metadata: %w", err)
	}
	asset.Metadata = updated
	result.FieldsAdded = []string{"geo", "geo_enriched"}

	e.logger.Debug().
		Str("asset_id", asset.ID.String()).
		Str("ip", *asset.IPAddress).
		Str("country", geoData.Country).
		Str("city", geoData.City).
		Float64("lat", geoData.Lat).
		Float64("lon", geoData.Lon).
		Msg("geolocation enrichment completed")

	return result, nil
}

// fetchGeoData calls ip-api.com and returns the parsed geolocation record.
// The response body is capped at 4 KB to guard against runaway reads.
func (e *GeoEnricher) fetchGeoData(ctx context.Context, ip string) (*ipAPIResponse, error) {
	apiURL := fmt.Sprintf("%s/%s?fields=%s", ipAPIBaseURL, ip, ipAPIFields)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create geo request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "clario360-cyber-service/1.0")

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ip-api request failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("ip-api rate limit exceeded (45 requests/minute on free tier)")
	case http.StatusOK:
		// continue
	default:
		return nil, fmt.Errorf("ip-api returned unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return nil, fmt.Errorf("read geo response: %w", err)
	}

	var geoResp ipAPIResponse
	if err := json.Unmarshal(body, &geoResp); err != nil {
		return nil, fmt.Errorf("parse geo response: %w", err)
	}
	if geoResp.Status != "success" {
		return nil, fmt.Errorf("ip-api query failed for %s: %s", ip, geoResp.Message)
	}

	return &geoResp, nil
}
