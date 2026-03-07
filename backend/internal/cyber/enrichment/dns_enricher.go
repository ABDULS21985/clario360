package enrichment

import (
	"context"
	"net"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/model"
)

// DNSEnricher resolves hostnames from IP addresses and vice versa.
type DNSEnricher struct {
	resolver *net.Resolver
	timeout  time.Duration
	logger   zerolog.Logger
}

// NewDNSEnricher creates a DNS enricher.
// timeout is applied per individual DNS lookup.
func NewDNSEnricher(logger zerolog.Logger, timeout time.Duration) *DNSEnricher {
	return &DNSEnricher{
		resolver: net.DefaultResolver,
		timeout:  timeout,
		logger:   logger,
	}
}

// Name implements Enricher.
func (e *DNSEnricher) Name() string { return "dns" }

// Enrich populates Hostname from IP (reverse DNS) or IPAddress from Hostname
// (forward lookup). If both are present it verifies forward-confirmed reverse DNS.
func (e *DNSEnricher) Enrich(ctx context.Context, asset *model.Asset) (*EnrichmentResult, error) {
	result := &EnrichmentResult{EnricherName: e.Name()}

	switch {
	case asset.IPAddress != nil && asset.Hostname == nil:
		// Reverse DNS: IP → hostname
		ctx2, cancel := context.WithTimeout(ctx, e.timeout)
		defer cancel()
		names, err := e.resolver.LookupAddr(ctx2, *asset.IPAddress)
		if err != nil || len(names) == 0 {
			return result, nil // not an error — just no PTR record
		}
		// Strip trailing dot from PTR records
		hostname := trimTrailingDot(names[0])
		asset.Hostname = &hostname
		result.FieldsAdded = append(result.FieldsAdded, "hostname")

	case asset.Hostname != nil && asset.IPAddress == nil:
		// Forward lookup: hostname → IP
		ctx2, cancel := context.WithTimeout(ctx, e.timeout)
		defer cancel()
		addrs, err := e.resolver.LookupHost(ctx2, *asset.Hostname)
		if err != nil || len(addrs) == 0 {
			return result, nil
		}
		ip := addrs[0]
		asset.IPAddress = &ip
		result.FieldsAdded = append(result.FieldsAdded, "ip_address")

	case asset.IPAddress != nil && asset.Hostname != nil:
		// Verify FCRDNS: forward lookup of hostname should include the asset's IP
		ctx2, cancel := context.WithTimeout(ctx, e.timeout)
		defer cancel()
		addrs, err := e.resolver.LookupHost(ctx2, *asset.Hostname)
		if err != nil {
			return result, nil
		}
		for _, addr := range addrs {
			if addr == *asset.IPAddress {
				return result, nil // confirmed
			}
		}
		e.logger.Warn().
			Str("asset_id", asset.ID.String()).
			Str("ip", *asset.IPAddress).
			Str("hostname", *asset.Hostname).
			Msg("FCRDNS mismatch: hostname forward lookup does not include asset IP")
	}

	return result, nil
}

func trimTrailingDot(s string) string {
	if len(s) > 0 && s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	return s
}
