package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/classifier"
	"github.com/clario360/platform/internal/cyber/model"
	"github.com/clario360/platform/internal/observability"
)

const (
	defaultSeed             = int64(42)
	assetTargetCount        = 500
	vulnTargetCount         = 200
	relationshipTargetCount = 50
)

type assetSeed struct {
	ID              uuid.UUID
	TenantID        uuid.UUID
	Name            string
	Type            model.AssetType
	IPAddress       string
	Hostname        string
	MACAddress      string
	OS              string
	OSVersion       string
	Owner           string
	Department      string
	Location        string
	Criticality     model.Criticality
	Status          model.AssetStatus
	DiscoveredAt    time.Time
	LastSeenAt      time.Time
	DiscoverySource string
	Metadata        []byte
	Tags            []string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	CreatedBy       uuid.UUID
}

type vulnerabilitySeed struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	AssetID     uuid.UUID
	CVEID       *string
	Title       string
	Description string
	Severity    string
	CVSSScore   *float64
	CVSSVector  *string
	Status      string
	DetectedAt  time.Time
	ResolvedAt  *time.Time
	Source      string
	Remediation string
	Proof       string
	Metadata    []byte
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type relationshipSeed struct {
	ID               uuid.UUID
	TenantID         uuid.UUID
	SourceAssetID    uuid.UUID
	TargetAssetID    uuid.UUID
	RelationshipType string
	Metadata         []byte
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
}

type generator struct {
	rng        *rand.Rand
	classifier *classifier.AssetClassifier
	logger     zerolog.Logger
	now        time.Time
	usedIPs    map[string]struct{}
	usedMACs   map[string]struct{}
	createdBy  uuid.UUID
}

func main() {
	var (
		dbURL        = flag.String("db-url", os.Getenv("CYBER_DB_URL"), "PostgreSQL connection string")
		tenantIDFlag = flag.String("tenant-id", os.Getenv("CYBER_SEED_TENANT_ID"), "Tenant UUID to seed")
		seedValue    = flag.Int64("seed", defaultSeed, "Deterministic random seed")
	)
	flag.Parse()

	if strings.TrimSpace(*dbURL) == "" {
		fmt.Fprintln(os.Stderr, "--db-url or CYBER_DB_URL is required")
		os.Exit(1)
	}

	tenantID := uuid.New()
	if strings.TrimSpace(*tenantIDFlag) != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(*tenantIDFlag))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid tenant id: %v\n", err)
			os.Exit(1)
		}
		tenantID = parsed
	}

	logger := observability.NewLogger("info", "console", "cyber-seeder")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create database pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}

	gen := &generator{
		rng:        rand.New(rand.NewSource(*seedValue)),
		classifier: classifier.NewAssetClassifier(logger),
		logger:     logger,
		now:        time.Now().UTC().Truncate(time.Second),
		usedIPs:    make(map[string]struct{}, assetTargetCount),
		usedMACs:   make(map[string]struct{}, assetTargetCount),
		createdBy:  uuid.New(),
	}

	assets := gen.generateAssets(tenantID)
	vulnerabilities := gen.generateVulnerabilities(tenantID, assets)
	relationships := gen.generateRelationships(tenantID, assets)

	seedCtx, seedCancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer seedCancel()
	if err := insertSeedData(seedCtx, pool, assets, vulnerabilities, relationships); err != nil {
		logger.Fatal().Err(err).Msg("failed to seed cybersecurity inventory")
	}

	logger.Info().
		Str("tenant_id", tenantID.String()).
		Int("assets", len(assets)).
		Int("vulnerabilities", len(vulnerabilities)).
		Int("relationships", len(relationships)).
		Msg("cyber seed completed")
}

func insertSeedData(ctx context.Context, pool *pgxpool.Pool, assets []assetSeed, vulnerabilities []vulnerabilitySeed, relationships []relationshipSeed) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := copyAssets(ctx, tx, assets); err != nil {
		return err
	}
	if err := copyVulnerabilities(ctx, tx, vulnerabilities); err != nil {
		return err
	}
	if err := copyRelationships(ctx, tx, relationships); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit seed transaction: %w", err)
	}
	return nil
}

func copyAssets(ctx context.Context, tx pgx.Tx, assets []assetSeed) error {
	rows := make([][]any, 0, len(assets))
	for _, asset := range assets {
		rows = append(rows, []any{
			asset.ID, asset.TenantID, asset.Name, string(asset.Type), asset.IPAddress, asset.Hostname, asset.MACAddress,
			asset.OS, asset.OSVersion, asset.Owner, asset.Department, asset.Location, string(asset.Criticality), string(asset.Status),
			asset.DiscoveredAt, asset.LastSeenAt, asset.DiscoverySource, asset.Metadata, asset.Tags, asset.CreatedBy, asset.CreatedAt, asset.UpdatedAt,
		})
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"assets"}, []string{
		"id", "tenant_id", "name", "type", "ip_address", "hostname", "mac_address", "os", "os_version", "owner",
		"department", "location", "criticality", "status", "discovered_at", "last_seen_at", "discovery_source", "metadata", "tags",
		"created_by", "created_at", "updated_at",
	}, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy assets: %w", err)
	}
	return nil
}

func copyVulnerabilities(ctx context.Context, tx pgx.Tx, vulnerabilities []vulnerabilitySeed) error {
	rows := make([][]any, 0, len(vulnerabilities))
	for _, vuln := range vulnerabilities {
		rows = append(rows, []any{
			vuln.ID, vuln.TenantID, vuln.AssetID, vuln.CVEID, vuln.Title, vuln.Description, vuln.Severity, vuln.CVSSScore,
			vuln.CVSSVector, vuln.Status, vuln.DetectedAt, vuln.ResolvedAt, vuln.Source, vuln.Remediation, vuln.Proof, vuln.Metadata,
			vuln.CreatedAt, vuln.UpdatedAt,
		})
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"vulnerabilities"}, []string{
		"id", "tenant_id", "asset_id", "cve_id", "title", "description", "severity", "cvss_score", "cvss_vector", "status",
		"detected_at", "resolved_at", "source", "remediation", "proof", "metadata", "created_at", "updated_at",
	}, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy vulnerabilities: %w", err)
	}
	return nil
}

func copyRelationships(ctx context.Context, tx pgx.Tx, relationships []relationshipSeed) error {
	rows := make([][]any, 0, len(relationships))
	for _, rel := range relationships {
		rows = append(rows, []any{
			rel.ID, rel.TenantID, rel.SourceAssetID, rel.TargetAssetID, rel.RelationshipType, rel.Metadata, rel.CreatedBy, rel.CreatedAt,
		})
	}
	_, err := tx.CopyFrom(ctx, pgx.Identifier{"asset_relationships"}, []string{
		"id", "tenant_id", "source_asset_id", "target_asset_id", "relationship_type", "metadata", "created_by", "created_at",
	}, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("copy relationships: %w", err)
	}
	return nil
}

func (g *generator) generateAssets(tenantID uuid.UUID) []assetSeed {
	typeCounts := []struct {
		assetType model.AssetType
		count     int
	}{
		{model.AssetTypeServer, 200},
		{model.AssetTypeEndpoint, 150},
		{model.AssetTypeNetworkDevice, 30},
		{model.AssetTypeCloudResource, 40},
		{model.AssetTypeIoTDevice, 20},
		{model.AssetTypeApplication, 25},
		{model.AssetTypeDatabase, 20},
		{model.AssetTypeContainer, 15},
	}

	assets := make([]assetSeed, 0, assetTargetCount)
	for _, item := range typeCounts {
		for index := 0; index < item.count; index++ {
			assets = append(assets, g.newAssetSeed(tenantID, item.assetType, index))
		}
	}
	return assets
}

func (g *generator) newAssetSeed(tenantID uuid.UUID, assetType model.AssetType, index int) assetSeed {
	name, hostname, owner, department, location, tags, osName, osVersion, metadata := g.assetProfile(assetType, index)
	ip := g.nextIP(assetType, department, index)
	mac := g.nextMAC()
	status := g.randomStatus()
	discoverySource := g.randomDiscoverySource(assetType)
	discoveredAt := g.randomPast(365)
	lastSeenAt := discoveredAt.Add(time.Duration(g.rng.Intn(2400)) * time.Hour)
	if lastSeenAt.After(g.now) {
		lastSeenAt = g.now.Add(-time.Duration(g.rng.Intn(48)) * time.Hour)
	}
	if lastSeenAt.Before(discoveredAt) {
		lastSeenAt = discoveredAt.Add(6 * time.Hour)
	}

	asset := &model.Asset{ID: uuid.New(), TenantID: tenantID, Name: name, Type: assetType, Tags: tags, Metadata: metadata, CreatedAt: g.now}
	if hostname != "" {
		asset.Hostname = &hostname
	}
	crit, _, _ := g.classifier.Classify(asset)

	return assetSeed{
		ID:              asset.ID,
		TenantID:        tenantID,
		Name:            name,
		Type:            assetType,
		IPAddress:       ip,
		Hostname:        hostname,
		MACAddress:      mac,
		OS:              osName,
		OSVersion:       osVersion,
		Owner:           owner,
		Department:      department,
		Location:        location,
		Criticality:     crit,
		Status:          status,
		DiscoveredAt:    discoveredAt,
		LastSeenAt:      lastSeenAt,
		DiscoverySource: discoverySource,
		Metadata:        metadata,
		Tags:            tags,
		CreatedBy:       g.createdBy,
		CreatedAt:       g.now,
		UpdatedAt:       g.now,
	}
}

func (g *generator) assetProfile(assetType model.AssetType, index int) (string, string, string, string, string, []string, string, string, []byte) {
	departmentPool := []string{"engineering", "finance", "operations", "security", "hr"}
	ownerPool := []string{"alice.johnson", "bruno.kim", "carla.singh", "dina.owens", "emre.tan", "frank.lopez"}
	locationPool := []string{"datacenter-01", "datacenter-02", "hq-floor-03", "branch-nyc", "aws-us-east-1", "azure-east-us"}
	department := departmentPool[index%len(departmentPool)]
	owner := ownerPool[index%len(ownerPool)]
	location := locationPool[index%len(locationPool)]
	tags := []string{department, "internal"}
	openPorts := []int{}
	publicIP := ""
	name := ""
	hostname := ""
	osName := "linux"
	osVersion := "Ubuntu 22.04"

	switch assetType {
	case model.AssetTypeServer:
		patterns := []string{"web-prod-%02d", "api-prod-%02d", "k8s-worker-%02d", "batch-stg-%02d"}
		name = fmt.Sprintf(patterns[index%len(patterns)], index+1)
		hostname = name + ".corp.local"
		openPorts = choosePorts(index, []int{22, 80, 443, 8080, 8443})
		tags = append(tags, envTag(name))
		if strings.Contains(name, "web") || strings.Contains(name, "api") {
			tags = append(tags, "internet-facing")
		}
	case model.AssetTypeEndpoint:
		prefix := []string{"laptop", "desktop"}[index%2]
		name = fmt.Sprintf("%s-%s-%03d", prefix, department, index+1)
		hostname = name + ".corp.local"
		osName, osVersion = chooseEndpointOS(index)
	case model.AssetTypeNetworkDevice:
		switch index % 3 {
		case 0:
			name = fmt.Sprintf("fw-%02d", index+1)
		case 1:
			name = fmt.Sprintf("sw-core-%02d", index+1)
		default:
			name = fmt.Sprintf("router-edge-%02d", index+1)
		}
		hostname = name + ".net.local"
		openPorts = choosePorts(index, []int{22, 161, 443})
		osName, osVersion = chooseNetworkOS(index)
	case model.AssetTypeCloudResource:
		env := []string{"prod", "staging", "dev"}[index%3]
		service := []string{"payments", "search", "analytics", "auth"}[index%4]
		name = fmt.Sprintf("ec2-%s-%s-%02d", env, service, index+1)
		hostname = name + ".compute.internal"
		openPorts = choosePorts(index, []int{22, 443, 8443})
		if env == "prod" {
			tags = append(tags, "production")
		}
		if index%3 == 0 {
			publicIP = fmt.Sprintf("34.%d.%d.%d", 20+index%100, 10+index%200, 5+index%200)
			tags = append(tags, "public")
		}
	case model.AssetTypeIoTDevice:
		name = fmt.Sprintf("sensor-%s-%02d", []string{"plant-a", "warehouse-b", "office-c"}[index%3], index+1)
		hostname = name + ".iot.local"
		openPorts = choosePorts(index, []int{80, 443, 1883})
		osName, osVersion = "linux", []string{"OpenWRT 23", "Yocto 4", "Ubuntu Core 22"}[index%3]
	case model.AssetTypeApplication:
		if index < 5 {
			name = fmt.Sprintf("lb-edge-%02d", index+1)
			tags = append(tags, "internet-facing", "production", "load-balancer")
			openPorts = []int{80, 443, 8443}
		} else {
			name = fmt.Sprintf("app-%s", []string{"payments", "orders", "billing", "risk", "hr-portal"}[index%5])
			openPorts = []int{443, 8080}
		}
		hostname = name + ".apps.local"
	case model.AssetTypeDatabase:
		engine := []string{"postgres", "mysql", "mongodb", "redis"}[index%4]
		name = fmt.Sprintf("db-%s-%02d", engine, index+1)
		hostname = name + ".data.local"
		tags = append(tags, "database")
		openPorts = map[string][]int{"postgres": {5432}, "mysql": {3306}, "mongodb": {27017}, "redis": {6379}}[engine]
		osName, osVersion = "linux", []string{"Ubuntu 22.04", "RHEL 9", "Debian 12"}[index%3]
	case model.AssetTypeContainer:
		name = fmt.Sprintf("pod-%s-%06x", []string{"api", "worker", "ingest", "frontend"}[index%4], index*index+17)
		hostname = name + ".cluster.local"
		tags = append(tags, "containerized")
		openPorts = choosePorts(index, []int{8080, 8443})
		osName, osVersion = "linux", "Alpine 3.18"
	}

	metadataMap := map[string]any{"environment": envFromTags(tags), "open_ports": openPorts}
	if publicIP != "" {
		metadataMap["public_ip"] = publicIP
	}
	metadata, _ := json.Marshal(metadataMap)
	tags = uniqueStrings(tags)
	return name, hostname, owner, department, location, tags, osName, osVersion, metadata
}

func (g *generator) generateVulnerabilities(tenantID uuid.UUID, assets []assetSeed) []vulnerabilitySeed {
	weightedAssets := make([]assetSeed, 0, len(assets)*2)
	for _, asset := range assets {
		weightedAssets = append(weightedAssets, asset)
		if asset.Criticality == model.CriticalityCritical || asset.Criticality == model.CriticalityHigh {
			weightedAssets = append(weightedAssets, asset, asset)
		}
	}

	severityBag := severityDistribution(vulnTargetCount)
	statusBag := vulnerabilityStatusDistribution(vulnTargetCount)
	vulns := make([]vulnerabilitySeed, 0, vulnTargetCount)
	usedCVEByAsset := make(map[uuid.UUID]map[string]struct{})
	cvePool := []string{"CVE-2024-3094", "CVE-2023-44487", "CVE-2021-44228", "CVE-2024-3400", "CVE-2023-4966", "CVE-2024-6387", "CVE-2023-3519", "CVE-2024-21762"}
	manualTitles := []string{"Excessive admin exposure", "Weak TLS cipher support", "Sensitive backup share exposed", "Default credentials detected"}

	for index := 0; index < vulnTargetCount; index++ {
		asset := weightedAssets[g.rng.Intn(len(weightedAssets))]
		severity := severityBag[index]
		status := statusBag[index]
		source := "manual"
		var cveID *string
		title := manualTitles[index%len(manualTitles)]
		description := fmt.Sprintf("%s on asset %s requires remediation.", title, asset.Name)
		if index%4 != 0 {
			source = "cve_enrichment"
			candidate := cvePool[index%len(cvePool)]
			for {
				if _, ok := usedCVEByAsset[asset.ID]; !ok {
					usedCVEByAsset[asset.ID] = make(map[string]struct{})
				}
				if _, exists := usedCVEByAsset[asset.ID][candidate]; !exists {
					usedCVEByAsset[asset.ID][candidate] = struct{}{}
					cveID = &candidate
					break
				}
				candidate = cvePool[g.rng.Intn(len(cvePool))]
			}
			title = *cveID + " on " + asset.Name
			description = fmt.Sprintf("Detected %s affecting %s (%s).", *cveID, asset.Name, asset.OSVersion)
		}

		detectedAt := g.randomPast(180)
		var resolvedAt *time.Time
		if status == "resolved" {
			resolved := detectedAt.Add(time.Duration(24+g.rng.Intn(240)) * time.Hour)
			resolvedAt = &resolved
		}
		cvss := cvssForSeverity(severity, g.rng)
		vector := cvssVectorForSeverity(severity)
		proof := fmt.Sprintf("Observed on %s during seeded discovery run.", asset.Name)
		metadata, _ := json.Marshal(map[string]any{"seeded": true, "asset_type": asset.Type})

		vulns = append(vulns, vulnerabilitySeed{ID: uuid.New(), TenantID: tenantID, AssetID: asset.ID, CVEID: cveID, Title: title, Description: description, Severity: severity, CVSSScore: &cvss, CVSSVector: &vector, Status: status, DetectedAt: detectedAt, ResolvedAt: resolvedAt, Source: source, Remediation: "Apply the recommended patch, rotate exposed credentials, and verify compensating controls.", Proof: proof, Metadata: metadata, CreatedAt: g.now, UpdatedAt: g.now})
	}
	return vulns
}

func (g *generator) generateRelationships(tenantID uuid.UUID, assets []assetSeed) []relationshipSeed {
	byType := map[model.AssetType][]assetSeed{}
	loadBalancers := make([]assetSeed, 0)
	for _, asset := range assets {
		byType[asset.Type] = append(byType[asset.Type], asset)
		if contains(asset.Tags, "load-balancer") {
			loadBalancers = append(loadBalancers, asset)
		}
	}

	rels := make([]relationshipSeed, 0, relationshipTargetCount)
	seen := make(map[string]struct{}, relationshipTargetCount)
	appendRel := func(source, target assetSeed, relationshipType string, metadata map[string]any) {
		if len(rels) >= relationshipTargetCount || source.ID == target.ID {
			return
		}
		key := source.ID.String() + ":" + target.ID.String() + ":" + relationshipType
		if _, exists := seen[key]; exists {
			return
		}
		payload, _ := json.Marshal(metadata)
		seen[key] = struct{}{}
		rels = append(rels, relationshipSeed{ID: uuid.New(), TenantID: tenantID, SourceAssetID: source.ID, TargetAssetID: target.ID, RelationshipType: relationshipType, Metadata: payload, CreatedBy: g.createdBy, CreatedAt: g.now})
	}

	for index, server := range byType[model.AssetTypeServer] {
		if len(rels) >= 20 || len(byType[model.AssetTypeDatabase]) == 0 {
			break
		}
		database := byType[model.AssetTypeDatabase][index%len(byType[model.AssetTypeDatabase])]
		appendRel(server, database, string(model.RelationshipDependsOn), map[string]any{"seeded": true, "reason": "application data dependency"})
	}
	for index, container := range byType[model.AssetTypeContainer] {
		if len(rels) >= 35 || len(byType[model.AssetTypeServer]) == 0 {
			break
		}
		server := byType[model.AssetTypeServer][index%len(byType[model.AssetTypeServer])]
		appendRel(container, server, string(model.RelationshipRunsOn), map[string]any{"seeded": true, "platform": "kubernetes"})
	}
	for index, application := range byType[model.AssetTypeApplication] {
		if len(rels) >= 45 || len(byType[model.AssetTypeServer]) == 0 {
			break
		}
		server := byType[model.AssetTypeServer][(index*3)%len(byType[model.AssetTypeServer])]
		appendRel(application, server, string(model.RelationshipRunsOn), map[string]any{"seeded": true, "runtime": "app-hosting"})
	}
	for index := 0; index < len(loadBalancers) && len(rels) < relationshipTargetCount; index++ {
		server := byType[model.AssetTypeServer][(index*5)%len(byType[model.AssetTypeServer])]
		appendRel(loadBalancers[index], server, string(model.RelationshipLoadBalances), map[string]any{"seeded": true, "protocol": "https"})
	}

	for len(rels) < relationshipTargetCount {
		source := assets[g.rng.Intn(len(assets))]
		target := assets[g.rng.Intn(len(assets))]
		typeValue := []string{string(model.RelationshipConnectsTo), string(model.RelationshipManagedBy), string(model.RelationshipBacksUp)}[g.rng.Intn(3)]
		appendRel(source, target, typeValue, map[string]any{"seeded": true})
	}

	return rels
}

func (g *generator) nextIP(assetType model.AssetType, department string, index int) string {
	subnetBase := map[model.AssetType]int{model.AssetTypeServer: 1, model.AssetTypeEndpoint: 2, model.AssetTypeNetworkDevice: 3, model.AssetTypeCloudResource: 4, model.AssetTypeIoTDevice: 5, model.AssetTypeApplication: 6, model.AssetTypeDatabase: 7, model.AssetTypeContainer: 8}[assetType]
	departmentOffset := map[string]int{"engineering": 10, "finance": 20, "operations": 30, "security": 40, "hr": 50}[department]
	thirdOctet := subnetBase + departmentOffset/10
	for {
		host := 10 + (index % 220) + g.rng.Intn(20)
		ip := fmt.Sprintf("10.0.%d.%d", thirdOctet, host)
		if _, exists := g.usedIPs[ip]; !exists {
			g.usedIPs[ip] = struct{}{}
			return ip
		}
		index++
	}
}

func (g *generator) nextMAC() string {
	for {
		bytes := []byte{0x02, byte(g.rng.Intn(256)), byte(g.rng.Intn(256)), byte(g.rng.Intn(256)), byte(g.rng.Intn(256)), byte(g.rng.Intn(256))}
		mac := fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", bytes[0], bytes[1], bytes[2], bytes[3], bytes[4], bytes[5])
		if _, exists := g.usedMACs[mac]; !exists {
			g.usedMACs[mac] = struct{}{}
			return mac
		}
	}
}

func (g *generator) randomStatus() model.AssetStatus {
	value := g.rng.Intn(100)
	switch {
	case value < 88:
		return model.AssetStatusActive
	case value < 95:
		return model.AssetStatusInactive
	case value < 98:
		return model.AssetStatusUnknown
	default:
		return model.AssetStatusDecommissioned
	}
}

func (g *generator) randomDiscoverySource(assetType model.AssetType) string {
	sources := []string{"manual", "network_scan", "import"}
	if assetType == model.AssetTypeCloudResource {
		sources = append(sources, "cloud_scan")
	}
	if assetType == model.AssetTypeEndpoint {
		sources = append(sources, "agent")
	}
	return sources[g.rng.Intn(len(sources))]
}

func (g *generator) randomPast(maxDaysAgo int) time.Time {
	daysAgo := g.rng.Intn(maxDaysAgo) + 1
	hoursAgo := g.rng.Intn(24)
	minutesAgo := g.rng.Intn(60)
	return g.now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(hoursAgo)*time.Hour - time.Duration(minutesAgo)*time.Minute)
}

func choosePorts(index int, base []int) []int {
	ports := append([]int(nil), base...)
	if len(ports) > 2 && index%3 == 0 {
		return ports[:2]
	}
	return ports
}

func chooseEndpointOS(index int) (string, string) {
	options := []struct{ os, version string }{{"windows", "Windows 11"}, {"windows", "Server 2022"}, {"macos", "macOS 14"}, {"linux", "Ubuntu 22.04"}}
	choice := options[index%len(options)]
	return choice.os, choice.version
}

func chooseNetworkOS(index int) (string, string) {
	options := []string{"Cisco IOS XE", "FortiOS 7.4", "Arista EOS 4.31"}
	return "network_os", options[index%len(options)]
}

func envTag(name string) string {
	switch {
	case strings.Contains(name, "prod"):
		return "production"
	case strings.Contains(name, "stg") || strings.Contains(name, "staging"):
		return "staging"
	default:
		return "development"
	}
}

func envFromTags(tags []string) string {
	for _, tag := range tags {
		switch tag {
		case "production", "staging", "development":
			return tag
		}
	}
	return "development"
}

func severityDistribution(total int) []string {
	counts := map[string]int{"critical": total * 10 / 100, "high": total * 20 / 100, "medium": total * 40 / 100, "low": total - (total*10/100 + total*20/100 + total*40/100)}
	result := make([]string, 0, total)
	for _, severity := range []string{"critical", "high", "medium", "low"} {
		for index := 0; index < counts[severity]; index++ {
			result = append(result, severity)
		}
	}
	return result
}

func vulnerabilityStatusDistribution(total int) []string {
	counts := map[string]int{"open": total * 60 / 100, "in_progress": total * 20 / 100, "mitigated": total * 15 / 100, "resolved": total - (total*60/100 + total*20/100 + total*15/100)}
	result := make([]string, 0, total)
	for _, status := range []string{"open", "in_progress", "mitigated", "resolved"} {
		for index := 0; index < counts[status]; index++ {
			result = append(result, status)
		}
	}
	return result
}

func cvssForSeverity(severity string, rng *rand.Rand) float64 {
	switch severity {
	case "critical":
		return 9.0 + rng.Float64()
	case "high":
		return 7.0 + rng.Float64()*1.9
	case "medium":
		return 4.0 + rng.Float64()*2.9
	default:
		return 0.1 + rng.Float64()*3.8
	}
}

func cvssVectorForSeverity(severity string) string {
	switch severity {
	case "critical":
		return "CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:C/C:H/I:H/A:H"
	case "high":
		return "CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:H/A:L"
	case "medium":
		return "CVSS:3.1/AV:N/AC:H/PR:L/UI:R/S:U/C:L/I:L/A:L"
	default:
		return "CVSS:3.1/AV:L/AC:H/PR:L/UI:R/S:U/C:L/I:N/A:N"
	}
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
