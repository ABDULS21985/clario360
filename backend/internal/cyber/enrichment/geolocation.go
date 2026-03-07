package enrichment

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// GeoEnricher enriches assets with geolocation data from a MaxMind GeoLite2 database.
// If enabled is false (default) it is a no-op so the service can run without the DB file.
type GeoEnricher struct {
	dbPath  string
	enabled bool
	logger  zerolog.Logger
}

// NewGeoEnricher creates a geolocation enricher.
// When enabled=false the enricher is a no-op.
func NewGeoEnricher(logger zerolog.Logger, dbPath string, enabled bool) *GeoEnricher {
	return &GeoEnricher{dbPath: dbPath, enabled: enabled, logger: logger}
}

// Name implements Enricher.
func (e *GeoEnricher) Name() string { return "geolocation" }

// Enrich adds geo metadata (country, city, lat/lon) to the asset's metadata JSONB.
// If the GeoLite2 database is not enabled or not readable this is a no-op.
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
	// Skip private/loopback IPs — no geolocation data available
	if ip.IsLoopback() || ip.IsPrivate() {
		return result, nil
	}

	// In production this would open the MaxMind database using oschwald/geoip2-golang.
	// We return a stub result here so the service compiles and runs without the .mmdb file.
	// To enable: add github.com/oschwald/geoip2-golang to go.mod and implement the lookup.
	e.logger.Debug().
		Str("asset_id", asset.ID.String()).
		Str("ip", *asset.IPAddress).
		Msg("geolocation enrichment: MaxMind DB integration not enabled in this build")

	// Merge a placeholder geo entry into the existing metadata
	var meta map[string]any
	if len(asset.Metadata) > 0 {
		if err := json.Unmarshal(asset.Metadata, &meta); err != nil {
			meta = make(map[string]any)
		}
	} else {
		meta = make(map[string]any)
	}
	meta["geo_enriched"] = false
	updated, err := json.Marshal(meta)
	if err != nil {
		return result, err
	}
	asset.Metadata = updated

	return result, nil
}
