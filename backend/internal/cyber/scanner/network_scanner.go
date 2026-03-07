package scanner

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/classifier"
	"github.com/clario360/platform/internal/cyber/enrichment"
	"github.com/clario360/platform/internal/cyber/model"
)

// AssetUpsertRepo is the minimal repository interface needed by NetworkScanner.
type AssetUpsertRepo interface {
	UpsertFromScan(ctx context.Context, tenantID uuid.UUID, d *model.DiscoveredAsset) (uuid.UUID, bool, error)
	GetByID(ctx context.Context, tenantID, assetID uuid.UUID) (*model.Asset, error)
	BulkUpdateCriticality(ctx context.Context, tenantID uuid.UUID, updates map[uuid.UUID]model.Criticality) error
}

// NetworkScanner performs TCP probe-based network asset discovery.
type NetworkScanner struct {
	repo       AssetUpsertRepo
	pipeline   *enrichment.Pipeline
	classifier *classifier.AssetClassifier
	logger     zerolog.Logger
	workers    int
	timeout    time.Duration
	maxIPs     int
	defaultPorts []int
}

// NewNetworkScanner creates a new network scanner.
func NewNetworkScanner(
	repo AssetUpsertRepo,
	pipeline *enrichment.Pipeline,
	cls *classifier.AssetClassifier,
	logger zerolog.Logger,
	workers, timeoutSec, maxIPs int,
	defaultPorts []int,
) *NetworkScanner {
	return &NetworkScanner{
		repo:         repo,
		pipeline:     pipeline,
		classifier:   cls,
		logger:       logger,
		workers:      workers,
		timeout:      time.Duration(timeoutSec) * time.Second,
		maxIPs:       maxIPs,
		defaultPorts: defaultPorts,
	}
}

// Type implements Scanner.
func (s *NetworkScanner) Type() model.ScanType { return model.ScanTypeNetwork }

// Scan performs a network scan over the given CIDR ranges.
func (s *NetworkScanner) Scan(ctx context.Context, cfg *model.ScanConfig) (*model.ScanResult, error) {
	start := time.Now()
	result := &model.ScanResult{Status: model.ScanStatusRunning}

	// Parse targets into IPs
	ips, err := expandCIDRs(cfg.Targets, s.maxIPs)
	if err != nil {
		result.Status = model.ScanStatusFailed
		result.Errors = []string{err.Error()}
		return result, err
	}

	// Determine ports
	ports := s.defaultPorts
	if len(cfg.Ports) > 0 {
		ports = cfg.Ports
	}

	// Validate ports
	for _, p := range ports {
		if p < 1 || p > 65535 {
			result.Status = model.ScanStatusFailed
			result.Errors = []string{fmt.Sprintf("invalid port: %d", p)}
			return result, fmt.Errorf("invalid port %d", p)
		}
	}

	// Semaphore for worker concurrency
	sem := make(chan struct{}, s.workers)
	discovered := make(chan *model.DiscoveredAsset, len(ips))
	var wg sync.WaitGroup
	var errsMu sync.Mutex
	var scanErrors []string

	for _, ip := range ips {
		select {
		case <-ctx.Done():
			break
		default:
		}

		wg.Add(1)
		go func(ipStr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			d, err := s.probeHost(ctx, ipStr, ports)
			if err != nil {
				errsMu.Lock()
				scanErrors = append(scanErrors, fmt.Sprintf("%s: %v", ipStr, err))
				errsMu.Unlock()
				return
			}
			if d != nil {
				discovered <- d
			}
		}(ip.String())
	}

	// Wait for all probes then close the channel
	go func() {
		wg.Wait()
		close(discovered)
	}()

	// Collect results — we need a tenantID which comes from the scan record,
	// but we don't have it here. The service layer passes it in via ctx value.
	tenantID := tenantIDFromCtx(ctx)

	var newCount, updatedCount int
	for d := range discovered {
		if tenantID == uuid.Nil {
			continue
		}
		assetID, isNew, err := s.repo.UpsertFromScan(ctx, tenantID, d)
		if err != nil {
			errsMu.Lock()
			scanErrors = append(scanErrors, fmt.Sprintf("upsert %s: %v", d.IPAddress, err))
			errsMu.Unlock()
			continue
		}
		d.AssetID = assetID
		d.IsNew = isNew
		if isNew {
			newCount++
		} else {
			updatedCount++
		}

		// Enrich and classify newly discovered assets
		asset, err := s.repo.GetByID(ctx, tenantID, assetID)
		if err == nil {
			s.pipeline.Run(ctx, asset)
			crit, _, _ := s.classifier.Classify(asset)
			if crit != asset.Criticality {
				_ = s.repo.BulkUpdateCriticality(ctx, tenantID, map[uuid.UUID]model.Criticality{assetID: crit})
			}
		}
	}

	result.Status = model.ScanStatusCompleted
	result.AssetsDiscovered = newCount + updatedCount
	result.AssetsNew = newCount
	result.AssetsUpdated = updatedCount
	result.DurationMs = time.Since(start).Milliseconds()
	result.Errors = scanErrors
	return result, nil
}

// probeHost probes a single IP address on the given ports.
// Returns nil if the host does not respond on any port.
func (s *NetworkScanner) probeHost(ctx context.Context, ipStr string, ports []int) (*model.DiscoveredAsset, error) {
	var openPorts []int
	banners := make(map[int]string)

	for _, port := range ports {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		addr := net.JoinHostPort(ipStr, fmt.Sprintf("%d", port))
		conn, err := net.DialTimeout("tcp", addr, s.timeout)
		if err != nil {
			continue
		}
		openPorts = append(openPorts, port)

		// Banner grab
		_ = conn.SetReadDeadline(time.Now().Add(time.Second))
		buf := make([]byte, 256)
		n, _ := conn.Read(buf)
		conn.Close()
		if n > 0 {
			banners[port] = string(buf[:n])
		}
	}

	if len(openPorts) == 0 {
		return nil, nil // host down or no ports open
	}

	d := &model.DiscoveredAsset{
		IPAddress: ipStr,
		OpenPorts: openPorts,
		Banners:   banners,
		AssetType: inferAssetType(openPorts),
	}

	// Reverse DNS for hostname
	ctx2, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	names, err := net.DefaultResolver.LookupAddr(ctx2, ipStr)
	if err == nil && len(names) > 0 {
		hostname := trimDot(names[0])
		d.Hostname = &hostname
	}

	// OS inference from banners
	d.OS, d.OSVersion = inferOS(banners)

	return d, nil
}

// inferAssetType maps open ports to the most likely AssetType.
func inferAssetType(ports []int) model.AssetType {
	portSet := make(map[int]bool, len(ports))
	for _, p := range ports {
		portSet[p] = true
	}
	switch {
	case portSet[3389]:
		return model.AssetTypeEndpoint // Windows RDP
	case portSet[161]:
		return model.AssetTypeNetworkDevice // SNMP
	case portSet[3306] || portSet[5432] || portSet[1433] || portSet[27017]:
		return model.AssetTypeDatabase
	default:
		return model.AssetTypeServer
	}
}

// inferOS tries to determine OS and version from banner strings.
func inferOS(banners map[int]string) (*string, *string) {
	for _, banner := range banners {
		b := strings.ToLower(banner)
		switch {
		case strings.Contains(b, "ubuntu"):
			os := "linux"
			ver := "Ubuntu"
			return &os, &ver
		case strings.Contains(b, "debian"):
			os := "linux"
			ver := "Debian"
			return &os, &ver
		case strings.Contains(b, "openssh"):
			os := "linux"
			return &os, nil
		case strings.Contains(b, "microsoft-iis"):
			os := "windows"
			ver := "IIS"
			return &os, &ver
		case strings.Contains(b, "windows"):
			os := "windows"
			return &os, nil
		}
	}
	return nil, nil
}

// expandCIDRs parses CIDR notation strings and returns individual IP addresses.
// Returns an error if the total exceeds maxIPs.
func expandCIDRs(cidrs []string, maxIPs int) ([]net.IP, error) {
	var all []net.IP
	for _, cidr := range cidrs {
		ips, err := expandCIDR(cidr)
		if err != nil {
			return nil, err
		}
		all = append(all, ips...)
		if len(all) > maxIPs {
			return nil, fmt.Errorf("scan scope too large: %d IPs exceeds maximum %d", len(all), maxIPs)
		}
	}
	return all, nil
}

func expandCIDR(cidr string) ([]net.IP, error) {
	ip, network, err := net.ParseCIDR(cidr)
	if err != nil {
		// Single IP (no prefix)?
		parsed := net.ParseIP(cidr)
		if parsed == nil {
			return nil, fmt.Errorf("invalid CIDR or IP: %s", cidr)
		}
		return []net.IP{parsed}, nil
	}

	// Reject too-large CIDRs (/0 through /7)
	ones, bits := network.Mask.Size()
	if ones < 8 && bits == 32 {
		return nil, fmt.Errorf("CIDR %s is too large (minimum /8)", cidr)
	}

	// For /32 return the single host
	if ones == 32 {
		return []net.IP{ip.Mask(network.Mask)}, nil
	}

	var ips []net.IP
	cur := cloneIP(network.IP)
	for network.Contains(cur) {
		// Skip network address and broadcast for subnets
		if !isNetworkOrBroadcast(cur, network) {
			// Skip loopback and reserved
			if !cur.IsLoopback() && !cur.Equal(net.IPv4zero) {
				c := cloneIP(cur)
				ips = append(ips, c)
			}
		}
		if !incrementIP(cur) {
			break
		}
	}
	return ips, nil
}

func cloneIP(ip net.IP) net.IP {
	clone := make(net.IP, len(ip))
	copy(clone, ip)
	return clone
}

func incrementIP(ip net.IP) bool {
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			return true
		}
	}
	return false // overflow
}

func isNetworkOrBroadcast(ip net.IP, network *net.IPNet) bool {
	ones, bits := network.Mask.Size()
	if ones == bits {
		return false // /32 — no network/broadcast
	}
	// Network address: all host bits zero
	masked := ip.Mask(network.Mask)
	if masked.Equal(network.IP) {
		return true
	}
	// Broadcast: all host bits one
	if len(network.IP) == 4 {
		n := binary.BigEndian.Uint32(network.IP.To4())
		m := binary.BigEndian.Uint32([]byte(network.Mask))
		bcast := n | ^m
		ipUint := binary.BigEndian.Uint32(ip.To4())
		if ipUint == bcast {
			return true
		}
	}
	return false
}

func trimDot(s string) string {
	if len(s) > 0 && s[len(s)-1] == '.' {
		return s[:len(s)-1]
	}
	return s
}

// ctxTenantIDKey is the context key for tenant ID passed to the scanner.
type ctxTenantIDKey struct{}

// WithTenantID stores tenantID in ctx for the scanner to retrieve.
func WithTenantID(ctx context.Context, tenantID uuid.UUID) context.Context {
	return context.WithValue(ctx, ctxTenantIDKey{}, tenantID)
}

func tenantIDFromCtx(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(ctxTenantIDKey{}).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}
