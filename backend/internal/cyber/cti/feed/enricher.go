package feed

import (
	"strings"

	"github.com/clario360/platform/internal/cyber/cti/feed/adapters"
)

// GeoLocation holds resolved geographic data for an IP address.
type GeoLocation struct {
	Country   string
	City      string
	Latitude  float64
	Longitude float64
}

// GeoResolver resolves IP addresses to geographic locations.
type GeoResolver interface {
	Resolve(ip string) (*GeoLocation, error)
}

// Enricher adds geographic, actor, and severity context to normalized indicators.
type Enricher struct {
	geo GeoResolver
}

func NewEnricher(geo GeoResolver) *Enricher {
	return &Enricher{geo: geo}
}

// Enrich enhances an indicator with geographic and heuristic data.
func (e *Enricher) Enrich(ind *adapters.NormalizedIndicator) {
	// GeoIP resolution for IP-type IOCs without existing geo data
	if ind.IOCType == "ip" && ind.OriginCountryCode == "" && ind.IOCValue != "" && e.geo != nil {
		if loc, err := e.geo.Resolve(ind.IOCValue); err == nil && loc != nil {
			ind.OriginCountryCode = loc.Country
			ind.OriginCity = loc.City
			ind.Latitude = loc.Latitude
			ind.Longitude = loc.Longitude
		}
	}

	// Severity auto-boost for high-confidence APT/ransomware
	if ind.ConfidenceScore > 0.8 {
		cat := strings.ToLower(ind.CategoryCode)
		if (cat == "apt" || cat == "ransomware" || cat == "zero_day") && ind.SeverityCode == "medium" {
			ind.SeverityCode = "high"
			ind.Tags = append(ind.Tags, "auto-escalated")
		}
	}

	// Default severity if missing
	if ind.SeverityCode == "" {
		ind.SeverityCode = "medium"
	}
}

// ---------------------------------------------------------------------------
// DevGeoResolver — for local development without MaxMind
// ---------------------------------------------------------------------------

type DevGeoResolver struct {
	prefixes map[string]*GeoLocation
}

func NewDevGeoResolver() *DevGeoResolver {
	return &DevGeoResolver{
		prefixes: map[string]*GeoLocation{
			"10.55.": {Country: "ru", City: "Moscow", Latitude: 55.7558, Longitude: 37.6173},
			"10.59.": {Country: "ru", City: "St Petersburg", Latitude: 59.9311, Longitude: 30.3609},
			"10.39.": {Country: "cn", City: "Beijing", Latitude: 39.9042, Longitude: 116.4074},
			"10.31.": {Country: "cn", City: "Shanghai", Latitude: 31.2304, Longitude: 121.4737},
			"10.35.": {Country: "ir", City: "Tehran", Latitude: 35.6892, Longitude: 51.3890},
			"10.38.": {Country: "kp", City: "Pyongyang", Latitude: 39.0392, Longitude: 125.7625},
			"10.6.":  {Country: "ng", City: "Lagos", Latitude: 6.5244, Longitude: 3.3792},
			"10.23.": {Country: "br", City: "Sao Paulo", Latitude: -23.5505, Longitude: -46.6333},
			"10.44.": {Country: "ro", City: "Bucharest", Latitude: 44.4268, Longitude: 26.1025},
			"10.10.": {Country: "vn", City: "Ho Chi Minh City", Latitude: 10.8231, Longitude: 106.6297},
			"10.41.": {Country: "tr", City: "Istanbul", Latitude: 41.0082, Longitude: 28.9784},
			"10.19.": {Country: "in", City: "Mumbai", Latitude: 19.0760, Longitude: 72.8777},
			"10.24.": {Country: "sa", City: "Riyadh", Latitude: 24.7136, Longitude: 46.6753},
			"10.25.": {Country: "ae", City: "Dubai", Latitude: 25.2048, Longitude: 55.2708},
			"10.40.": {Country: "us", City: "New York", Latitude: 40.7128, Longitude: -74.0060},
			"10.51.": {Country: "gb", City: "London", Latitude: 51.5074, Longitude: -0.1278},
			"10.52.": {Country: "de", City: "Berlin", Latitude: 52.5200, Longitude: 13.4050},
			"10.48.": {Country: "fr", City: "Paris", Latitude: 48.8566, Longitude: 2.3522},
		},
	}
}

func (r *DevGeoResolver) Resolve(ip string) (*GeoLocation, error) {
	for prefix, loc := range r.prefixes {
		if strings.HasPrefix(ip, prefix) {
			return loc, nil
		}
	}
	return nil, nil
}
