// cti-seeder seeds the CTI (Cyber Threat Intelligence) tables in cyber_db
// with realistic demonstration data for development and demo environments.
//
// Usage:
//
//	go run ./cmd/cti-seeder --db-url=postgres://... --tenant-id=<uuid>
package main

import (
	"context"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/observability"
)

// ---------------------------------------------------------------------------
// Static seed data
// ---------------------------------------------------------------------------

type severityDef struct {
	Code      string
	Label     string
	ColorHex  string
	SortOrder int
}

var severities = []severityDef{
	{"critical", "Critical", "#DC2626", 1},
	{"high", "High", "#EA580C", 2},
	{"medium", "Medium", "#CA8A04", 3},
	{"low", "Low", "#2563EB", 4},
	{"informational", "Informational", "#6B7280", 5},
}

type categoryDef struct {
	Code        string
	Label       string
	Description string
	MitreTIDs   []string
}

var categories = []categoryDef{
	{"apt", "Advanced Persistent Threat", "State-sponsored or highly capable adversary conducting long-term operations", []string{"TA0001", "TA0003", "TA0004", "TA0011"}},
	{"ransomware", "Ransomware", "Malware that encrypts data and demands ransom", []string{"TA0002", "TA0040", "TA0005"}},
	{"phishing", "Phishing", "Social engineering via deceptive emails or websites", []string{"TA0001", "TA0043"}},
	{"ddos", "DDoS", "Distributed Denial-of-Service attack", []string{"TA0040"}},
	{"supply_chain", "Supply Chain Attack", "Compromise through trusted third-party software or services", []string{"TA0001", "TA0003"}},
	{"insider_threat", "Insider Threat", "Malicious or negligent action by authorized users", []string{"TA0001", "TA0010"}},
	{"zero_day", "Zero-Day Exploit", "Exploitation of previously unknown vulnerability", []string{"TA0001", "TA0002"}},
	{"wiper", "Wiper / Destructive", "Malware designed to destroy data", []string{"TA0040"}},
	{"botnet", "Botnet", "Network of compromised devices under centralized control", []string{"TA0011", "TA0002"}},
	{"cryptojacking", "Cryptojacking", "Unauthorized cryptocurrency mining", []string{"TA0040"}},
	{"bec_fraud", "Business Email Compromise", "Fraud via impersonation of business executives or partners", []string{"TA0001", "TA0043"}},
	{"credential_theft", "Credential Theft", "Harvesting of usernames and passwords", []string{"TA0006"}},
	{"data_exfil", "Data Exfiltration", "Unauthorized transfer of data out of the organization", []string{"TA0010"}},
	{"espionage", "Espionage", "Intelligence gathering for political or economic advantage", []string{"TA0009", "TA0010"}},
	{"destructive", "Destructive Attack", "Attacks aimed at destroying systems or data", []string{"TA0040"}},
}

type regionDef struct {
	Code       string
	Label      string
	Parent     string // code of parent
	Lat        float64
	Lng        float64
	ISOCountry string
}

var regions = []regionDef{
	// Continents
	{"asia", "Asia", "", 34.0479, 100.6197, ""},
	{"europe", "Europe", "", 54.5260, 15.2551, ""},
	{"north_america", "North America", "", 54.5260, -105.2551, ""},
	{"south_america", "South America", "", -8.7832, -55.4915, ""},
	{"africa", "Africa", "", -8.7832, 34.5085, ""},
	{"oceania", "Oceania", "", -22.7359, 140.0188, ""},
	// Sub-regions
	{"east_asia", "East Asia", "asia", 35.8617, 104.1954, ""},
	{"south_asia", "South Asia", "asia", 20.5937, 78.9629, ""},
	{"southeast_asia", "Southeast Asia", "asia", 4.2105, 101.9758, ""},
	{"middle_east", "Middle East", "asia", 29.3117, 47.4818, ""},
	{"eastern_europe", "Eastern Europe", "europe", 53.9006, 27.5590, ""},
	{"western_europe", "Western Europe", "europe", 46.2276, 2.2137, ""},
	{"northern_europe", "Northern Europe", "europe", 59.9139, 10.7522, ""},
	{"west_africa", "West Africa", "africa", 9.0820, 8.6753, ""},
	{"east_africa", "East Africa", "africa", -1.2921, 36.8219, ""},
	// Countries
	{"cn", "China", "east_asia", 39.9042, 116.4074, "CHN"},
	{"ru", "Russia", "eastern_europe", 55.7558, 37.6173, "RUS"},
	{"us", "United States", "north_america", 38.9072, -77.0369, "USA"},
	{"gb", "United Kingdom", "western_europe", 51.5074, -0.1278, "GBR"},
	{"ir", "Iran", "middle_east", 35.6892, 51.3890, "IRN"},
	{"kp", "North Korea", "east_asia", 39.0392, 125.7625, "PRK"},
	{"br", "Brazil", "south_america", -15.7975, -47.8919, "BRA"},
	{"ng", "Nigeria", "west_africa", 9.0579, 7.4951, "NGA"},
	{"de", "Germany", "western_europe", 52.5200, 13.4050, "DEU"},
	{"fr", "France", "western_europe", 48.8566, 2.3522, "FRA"},
	{"in", "India", "south_asia", 28.6139, 77.2090, "IND"},
	{"kr", "South Korea", "east_asia", 37.5665, 126.9780, "KOR"},
	{"il", "Israel", "middle_east", 31.7683, 35.2137, "ISR"},
	{"sa", "Saudi Arabia", "middle_east", 24.7136, 46.6753, "SAU"},
	{"ae", "UAE", "middle_east", 25.2048, 55.2708, "ARE"},
	{"ua", "Ukraine", "eastern_europe", 50.4501, 30.5234, "UKR"},
	{"pk", "Pakistan", "south_asia", 33.6844, 73.0479, "PAK"},
	{"au", "Australia", "oceania", -33.8688, 151.2093, "AUS"},
	{"jp", "Japan", "east_asia", 35.6762, 139.6503, "JPN"},
	{"sg", "Singapore", "southeast_asia", 1.3521, 103.8198, "SGP"},
	{"nl", "Netherlands", "western_europe", 52.3676, 4.9041, "NLD"},
	{"id", "Indonesia", "southeast_asia", -6.2088, 106.8456, "IDN"},
	{"vn", "Vietnam", "southeast_asia", 21.0285, 105.8542, "VNM"},
	{"tr", "Turkey", "middle_east", 39.9334, 32.8597, "TUR"},
	{"ro", "Romania", "eastern_europe", 44.4268, 26.1025, "ROU"},
	{"ke", "Kenya", "east_africa", -1.2921, 36.8219, "KEN"},
}

type sectorDef struct {
	Code        string
	Label       string
	Description string
	NAICS       string
}

var sectors = []sectorDef{
	{"technology", "Technology", "Software, hardware, cloud services, SaaS providers", "5112"},
	{"government", "Government", "Federal, state, local agencies; public administration", "9211"},
	{"financial_services", "Financial Services", "Banks, insurance, fintech, capital markets", "5221"},
	{"healthcare", "Healthcare", "Hospitals, pharma, medical devices, biotech", "6211"},
	{"energy", "Energy", "Oil & gas, utilities, renewables, nuclear", "2111"},
	{"defense", "Defense", "Military contractors, weapons systems, intelligence", "9271"},
	{"critical_infrastructure", "Critical Infrastructure", "Water, power grids, dams, transportation", "2211"},
	{"media", "Media & Entertainment", "News, social media, streaming, gaming", "5121"},
	{"telecom", "Telecommunications", "ISPs, mobile carriers, satellite, fiber", "5171"},
	{"transportation", "Transportation", "Airlines, shipping, rail, logistics", "4811"},
	{"education", "Education", "Universities, K-12, e-learning, research", "6111"},
	{"retail", "Retail", "E-commerce, brick-and-mortar, supply chain", "4521"},
	{"manufacturing", "Manufacturing", "Automotive, electronics, industrial, OT/ICS", "3361"},
}

type sourceDef struct {
	Name          string
	SourceType    string
	URL           string
	Reliability   float64
	PollIntervalS int
}

var dataSources = []sourceDef{
	{"MITRE ATT&CK", "government_feed", "https://attack.mitre.org", 0.95, 86400},
	{"AlienVault OTX", "osint", "https://otx.alienvault.com", 0.80, 3600},
	{"Abuse.ch URLhaus", "osint", "https://urlhaus.abuse.ch", 0.85, 1800},
	{"Recorded Future", "commercial_feed", "https://app.recordedfuture.com", 0.92, 900},
	{"Shodan Monitor", "osint", "https://monitor.shodan.io", 0.78, 3600},
	{"Internal SOC", "internal", "", 0.90, 0},
	{"Dark Web Monitor", "dark_web", "", 0.72, 7200},
	{"Government CERT Feed", "government_feed", "https://cert.example.gov", 0.88, 3600},
}

type actorDef struct {
	Name           string
	Aliases        []string
	ActorType      string
	OriginCountry  string
	Sophistication string
	Motivation     string
	Description    string
	MitreGroupID   string
	RiskScore      float64
}

var actors = []actorDef{
	{"CRIMSON BEAR", []string{"APT-CB", "FancyFerret"}, "state_sponsored", "ru", "advanced", "espionage", "Russian state-sponsored group targeting Western governments and defense sectors", "G0007", 92.5},
	{"SILENT PANDA", []string{"PandaStorm", "APT-SP"}, "state_sponsored", "cn", "advanced", "espionage", "Chinese espionage group focused on technology IP theft and telecom infiltration", "G0096", 89.0},
	{"LAZARUS ECHO", []string{"HiddenCobra-E", "LabyrinthChollima"}, "state_sponsored", "kp", "advanced", "financial_gain", "North Korean group conducting financial theft via SWIFT and cryptocurrency platforms", "G0032", 88.5},
	{"CHARMING KITTEN", []string{"APT-CK", "PhosphorusV"}, "state_sponsored", "ir", "intermediate", "espionage", "Iranian group targeting Middle East energy and government sectors", "G0059", 82.0},
	{"SANDSTORM COLLECTIVE", []string{"VoodooSand"}, "state_sponsored", "ru", "advanced", "disruption", "Destructive attack group specializing in wiper malware against critical infrastructure", "G0034", 94.0},
	{"DARK SYNDICATE", []string{"DarkMoney", "Fin8Plus"}, "cybercriminal", "ru", "intermediate", "financial_gain", "Eastern European ransomware cartel operating RaaS infrastructure", "", 78.0},
	{"PHANTOM SPIDER", []string{"WebSpinner"}, "cybercriminal", "br", "intermediate", "financial_gain", "South American BEC fraud operation targeting multinational corporations", "", 65.0},
	{"NEON WOLF", []string{"WolfPack-N"}, "cybercriminal", "ng", "basic", "financial_gain", "West African credential-harvesting network using commodity phishing kits", "", 55.0},
	{"DIGITAL RESISTANCE", []string{"D-Resist", "AnonLegion"}, "hacktivist", "us", "basic", "ideological", "Hacktivist collective conducting DDoS and defacement for political causes", "", 42.0},
	{"GHOST CIRCUIT", []string{"GC-APT"}, "state_sponsored", "cn", "advanced", "espionage", "Chinese supply-chain attack group targeting semiconductor and 5G infrastructure", "", 91.0},
	{"VIPER CELL", []string{"KSA-Viper"}, "state_sponsored", "ir", "intermediate", "disruption", "Iranian disruptive unit focused on Saudi and Gulf state energy facilities", "", 80.0},
	{"SILENT GLACIER", []string{"Glacier-0"}, "state_sponsored", "ru", "advanced", "espionage", "SVR-linked cyber-espionage unit targeting diplomatic and policy institutions", "G0016", 90.0},
	{"CRYPTO HYDRA", []string{"HydraRaaS"}, "cybercriminal", "ro", "intermediate", "financial_gain", "Ransomware-as-a-Service operator with double-extortion model", "", 72.0},
	{"OCEAN LOTUS REMNANT", []string{"OceanV2"}, "state_sponsored", "vn", "intermediate", "espionage", "Vietnamese APT conducting espionage against ASEAN government targets", "G0050", 68.0},
	{"INSIDER THETA", []string{}, "insider", "us", "basic", "financial_gain", "Represents insider threat scenarios: disgruntled employees or recruited insiders", "", 60.0},
}

type campaignDef struct {
	Code           string
	Name           string
	Status         string
	ActorIdx       int // index into actors
	Description    string
	TTPs           string
	MitreTechniques []string
	TargetSectors  []int // indices into sectors
	DaysAgo        int   // first_seen offset from now
}

var campaigns = []campaignDef{
	{"C-2026-0101", "CRIMSON TEMPEST", "active", 0, "Credential-harvesting campaign targeting NATO government portals via spear-phishing", "Spear phishing with weaponized Office docs, lateral movement via PsExec, exfil over DNS", []string{"T1566.001", "T1059.001", "T1071.004", "T1078"}, []int{1, 5}, 45},
	{"C-2026-0102", "SILENT TYPHOON", "active", 1, "Supply-chain compromise targeting cloud SaaS providers to access downstream customers", "Trojanized npm packages, GitHub Actions abuse, cloud API key theft", []string{"T1195.002", "T1059.007", "T1528"}, []int{0, 8}, 30},
	{"C-2026-0103", "PHANTOM VORTEX", "active", 2, "Cryptocurrency exchange exploitation via watering-hole attacks", "Watering-hole on crypto forums, browser exploits, clipboard hijacking", []string{"T1189", "T1185", "T1115"}, []int{2}, 20},
	{"C-2026-0104", "DESERT SHADOW", "active", 3, "Espionage campaign against Gulf state energy companies via VPN exploit chains", "FortiGate CVE exploitation, Cobalt Strike beacons, SMB lateral movement", []string{"T1190", "T1059.003", "T1021.002"}, []int{4, 6}, 15},
	{"C-2026-0105", "IRON GLACIER", "active", 4, "Wiper deployment against Ukrainian critical infrastructure during geopolitical tensions", "CaddyWiper variant, GPO abuse for deployment, MBR destruction", []string{"T1561.002", "T1484.001", "T1485"}, []int{6, 4}, 10},
	{"C-2026-0106", "NEON HARVEST", "active", 5, "Large-scale ransomware campaign against healthcare and education via Citrix Bleed", "Citrix Bleed exploitation, data exfiltration before encryption, leak site pressure", []string{"T1190", "T1486", "T1567.002"}, []int{3, 10}, 7},
	{"C-2026-0107", "SPIDER WEB", "monitoring", 6, "BEC fraud operation impersonating CFOs at multinational firms", "Email domain spoofing, invoice redirection, wire fraud", []string{"T1566.002", "T1534"}, []int{2, 11}, 60},
	{"C-2026-0108", "DIGITAL STORM", "monitoring", 8, "DDoS campaign against media organizations covering controversial geopolitical events", "Amplification attacks, application-layer floods, Telegram coordination", []string{"T1498", "T1499"}, []int{7}, 55},
	{"C-2026-0109", "CIRCUIT BREAKER", "monitoring", 9, "Supply-chain reconnaissance targeting semiconductor fabs", "LinkedIn social engineering, vendor portal credential spray, firmware implants", []string{"T1566.003", "T1110.003", "T1195.003"}, []int{12, 0}, 50},
	{"C-2026-0110", "HYDRA LOCK", "dormant", 12, "Dormant RaaS affiliate program regrouping after law enforcement takedown", "New infrastructure setup, affiliate recruitment, testing new encryptor", []string{"T1486", "T1562.001"}, []int{3, 2}, 80},
	{"C-2026-0111", "OCEAN DRIFT", "dormant", 13, "Low-level espionage probes against ASEAN maritime agencies", "Spear-phishing with RTF exploits, PlugX RAT deployment", []string{"T1566.001", "T1059.005"}, []int{1, 9}, 75},
	{"C-2026-0112", "WOLF TRAP", "resolved", 7, "Resolved credential-harvesting operation taken down via domain seizure", "Bulk phishing kits, auto-generated lookalike domains", []string{"T1566.002", "T1090.002"}, []int{2, 11}, 90},
}

type brandDef struct {
	Name    string
	Domain  string
	Keywords []string
}

var brands = []brandDef{
	{"Clario 360", "clario360.com", []string{"clario", "clario360", "cipher360"}},
	{"Meridian Bank", "meridianbank.example.com", []string{"meridian", "meridianbank"}},
	{"AuroraPay", "aurorapay.example.com", []string{"aurora", "aurorapay", "aurora-pay"}},
	{"NovaCare Health", "novacare.example.com", []string{"novacare", "nova-care"}},
	{"Zenith Energy", "zenithenergy.example.com", []string{"zenith", "zenithenergy"}},
	{"Vertex Defense", "vertexdefense.example.com", []string{"vertex", "vertexdefense"}},
	{"CloudNest SaaS", "cloudnest.example.com", []string{"cloudnest", "cloud-nest"}},
	{"PrimeTech Corp", "primetech.example.com", []string{"primetech", "prime-tech"}},
	{"SafeRoute Logistics", "saferoute.example.com", []string{"saferoute", "safe-route"}},
	{"EduConnect", "educonnect.example.com", []string{"educonnect", "edu-connect"}},
}

// Cities used for threat-event origins
type cityDef struct {
	Name    string
	Country string
	Lat     float64
	Lng     float64
}

var originCities = []cityDef{
	{"Moscow", "ru", 55.7558, 37.6173},
	{"St Petersburg", "ru", 59.9311, 30.3609},
	{"Beijing", "cn", 39.9042, 116.4074},
	{"Shanghai", "cn", 31.2304, 121.4737},
	{"Shenzhen", "cn", 22.5431, 114.0579},
	{"Tehran", "ir", 35.6892, 51.3890},
	{"Pyongyang", "kp", 39.0392, 125.7625},
	{"Lagos", "ng", 6.5244, 3.3792},
	{"Sao Paulo", "br", -23.5505, -46.6333},
	{"Bucharest", "ro", 44.4268, 26.1025},
	{"Ho Chi Minh City", "vn", 10.8231, 106.6297},
	{"Hanoi", "vn", 21.0285, 105.8542},
	{"Istanbul", "tr", 41.0082, 28.9784},
	{"Mumbai", "in", 19.0760, 72.8777},
	{"Karachi", "pk", 24.8607, 67.0011},
	{"Jakarta", "id", -6.2088, 106.8456},
}

var targetCountries = []string{"us", "gb", "de", "fr", "sa", "ae", "il", "jp", "kr", "au", "sg", "nl", "in"}

var eventTitles = []string{
	"Spear-phishing email with weaponized attachment detected",
	"Credential-stuffing attack against VPN portal observed",
	"C2 beacon callback to known APT infrastructure identified",
	"Ransomware payload delivery via RDP brute-force",
	"DNS tunneling exfiltration attempt blocked",
	"Watering-hole redirect to exploit kit observed",
	"Lateral movement via PsExec from compromised endpoint",
	"Cobalt Strike stager downloaded from staging server",
	"Cryptojacking script injected into web application",
	"DDoS amplification traffic targeting public API gateway",
	"Insider data download exceeding baseline by 500%",
	"Supply-chain package with embedded backdoor detected",
	"Zero-day exploit attempt against unpatched web server",
	"Wiper malware signature matched in network traffic",
	"BEC wire-transfer request impersonating CFO",
	"Malicious OAuth application requesting excessive scopes",
	"SSH brute-force from Tor exit node",
	"SQL injection probe against customer-facing database",
	"Fileless PowerShell execution via WMI persistence",
	"Exfiltration via HTTPS to cloud storage bucket",
}

var mitreTechniques = []string{
	"T1566.001", "T1566.002", "T1059.001", "T1059.003", "T1059.005", "T1059.007",
	"T1071.001", "T1071.004", "T1078", "T1190", "T1195.002", "T1486",
	"T1021.002", "T1053.005", "T1547.001", "T1027", "T1140",
	"T1055", "T1003.001", "T1562.001", "T1070.004", "T1110.003",
	"T1498", "T1499", "T1189", "T1185", "T1115", "T1534",
	"T1561.002", "T1485", "T1567.002", "T1528", "T1090.002",
}

var abuseTypes = []string{
	"credential_phishing", "invoice_fraud", "tech_support_scam",
	"identity_theft", "oauth_phishing", "payment_fraud",
	"spear_phishing", "data_harvesting", "typosquatting", "lookalike_site",
}

var takedownStatuses = []string{
	"detected", "detected", "detected", "reported",
	"takedown_requested", "taken_down", "monitoring", "false_positive",
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	var (
		dbURL        = flag.String("db-url", os.Getenv("CYBER_DB_URL"), "PostgreSQL connection string")
		tenantIDFlag = flag.String("tenant-id", os.Getenv("CYBER_SEED_TENANT_ID"), "Tenant UUID to seed")
		seedValue    = flag.Int64("seed", 42, "Deterministic random seed")
	)
	flag.Parse()

	if strings.TrimSpace(*dbURL) == "" {
		fmt.Fprintln(os.Stderr, "--db-url or CYBER_DB_URL is required")
		os.Exit(1)
	}

	tenantID, err := uuid.Parse("aaaaaaaa-0000-0000-0000-000000000001") // default dev tenant
	if strings.TrimSpace(*tenantIDFlag) != "" {
		tenantID, err = uuid.Parse(strings.TrimSpace(*tenantIDFlag))
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid tenant id: %v\n", err)
			os.Exit(1)
		}
	}

	logger := observability.NewLogger("info", "console", "cti-seeder")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, *dbURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create database pool")
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}

	rng := rand.New(rand.NewSource(*seedValue))
	now := time.Now().UTC().Truncate(time.Second)

	s := &seeder{
		pool:     pool,
		logger:   logger,
		rng:      rng,
		tenantID: tenantID,
		now:      now,
		// ID maps populated during seeding
		severityIDs: make(map[string]uuid.UUID),
		categoryIDs: make(map[string]uuid.UUID),
		regionIDs:   make(map[string]uuid.UUID),
		sectorIDs:   make(map[string]uuid.UUID),
		sourceIDs:   make(map[string]uuid.UUID),
		actorIDs:    make([]uuid.UUID, 0),
		campaignIDs: make([]uuid.UUID, 0),
		eventIDs:    make([]uuid.UUID, 0),
		brandIDs:    make([]uuid.UUID, 0),
	}

	logger.Info().
		Str("tenant_id", tenantID.String()).
		Int64("seed", *seedValue).
		Msg("starting CTI data seeding")

	if err := s.run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("seeding failed")
	}

	logger.Info().Msg("CTI data seeding completed successfully")
}

// ---------------------------------------------------------------------------
// Seeder
// ---------------------------------------------------------------------------

type seeder struct {
	pool     *pgxpool.Pool
	logger   zerolog.Logger
	rng      *rand.Rand
	tenantID uuid.UUID
	now      time.Time

	severityIDs map[string]uuid.UUID
	categoryIDs map[string]uuid.UUID
	regionIDs   map[string]uuid.UUID
	sectorIDs   map[string]uuid.UUID
	sourceIDs   map[string]uuid.UUID
	actorIDs    []uuid.UUID
	campaignIDs []uuid.UUID
	eventIDs    []uuid.UUID
	brandIDs    []uuid.UUID
}

func (s *seeder) run(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// Set tenant for RLS bypass (superuser/service account)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", s.tenantID.String())); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	steps := []struct {
		name string
		fn   func(context.Context, pgx.Tx) error
	}{
		{"severity_levels", s.seedSeverities},
		{"threat_categories", s.seedCategories},
		{"geographic_regions", s.seedRegions},
		{"industry_sectors", s.seedSectors},
		{"data_sources", s.seedSources},
		{"threat_actors", s.seedActors},
		{"campaigns", s.seedCampaigns},
		{"threat_events", s.seedEvents},
		{"campaign_iocs", s.seedCampaignIOCs},
		{"campaign_events", s.seedCampaignEvents},
		{"monitored_brands", s.seedBrands},
		{"brand_abuse_incidents", s.seedBrandAbuse},
		{"geo_threat_summary", s.seedGeoSummary},
		{"sector_threat_summary", s.seedSectorSummary},
		{"executive_snapshot", s.seedExecSnapshot},
	}

	for _, step := range steps {
		s.logger.Info().Str("step", step.name).Msg("seeding")
		if err := step.fn(ctx, tx); err != nil {
			return fmt.Errorf("seed %s: %w", step.name, err)
		}
	}

	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Reference table seeders
// ---------------------------------------------------------------------------

func (s *seeder) seedSeverities(ctx context.Context, tx pgx.Tx) error {
	for _, sv := range severities {
		id := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_threat_severity_levels (id, tenant_id, code, label, color_hex, sort_order)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (tenant_id, code) DO NOTHING`,
			id, s.tenantID, sv.Code, sv.Label, sv.ColorHex, sv.SortOrder)
		if err != nil {
			return err
		}
		s.severityIDs[sv.Code] = id
	}
	// Re-read in case ON CONFLICT skipped
	rows, err := tx.Query(ctx, `SELECT id, code FROM cti_threat_severity_levels WHERE tenant_id = $1`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		s.severityIDs[code] = id
	}
	return rows.Err()
}

func (s *seeder) seedCategories(ctx context.Context, tx pgx.Tx) error {
	for _, c := range categories {
		id := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_threat_categories (id, tenant_id, code, label, description, mitre_tactic_ids)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (tenant_id, code) DO NOTHING`,
			id, s.tenantID, c.Code, c.Label, c.Description, c.MitreTIDs)
		if err != nil {
			return err
		}
		s.categoryIDs[c.Code] = id
	}
	rows, err := tx.Query(ctx, `SELECT id, code FROM cti_threat_categories WHERE tenant_id = $1`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		s.categoryIDs[code] = id
	}
	return rows.Err()
}

func (s *seeder) seedRegions(ctx context.Context, tx pgx.Tx) error {
	// First pass: insert all without parent
	for _, r := range regions {
		id := uuid.New()
		var isoPtr *string
		if r.ISOCountry != "" {
			isoPtr = &r.ISOCountry
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_geographic_regions (id, tenant_id, code, label, latitude, longitude, iso_country_code)
			VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT (tenant_id, code) DO NOTHING`,
			id, s.tenantID, r.Code, r.Label, r.Lat, r.Lng, isoPtr)
		if err != nil {
			return err
		}
		s.regionIDs[r.Code] = id
	}
	// Re-read
	rows, err := tx.Query(ctx, `SELECT id, code FROM cti_geographic_regions WHERE tenant_id = $1`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		s.regionIDs[code] = id
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Second pass: set parent_region_id
	for _, r := range regions {
		if r.Parent == "" {
			continue
		}
		parentID, ok := s.regionIDs[r.Parent]
		if !ok {
			continue
		}
		selfID := s.regionIDs[r.Code]
		_, err := tx.Exec(ctx, `UPDATE cti_geographic_regions SET parent_region_id = $1 WHERE id = $2`, parentID, selfID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedSectors(ctx context.Context, tx pgx.Tx) error {
	for _, sc := range sectors {
		id := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_industry_sectors (id, tenant_id, code, label, description, naics_code)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (tenant_id, code) DO NOTHING`,
			id, s.tenantID, sc.Code, sc.Label, sc.Description, sc.NAICS)
		if err != nil {
			return err
		}
		s.sectorIDs[sc.Code] = id
	}
	rows, err := tx.Query(ctx, `SELECT id, code FROM cti_industry_sectors WHERE tenant_id = $1`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			return err
		}
		s.sectorIDs[code] = id
	}
	return rows.Err()
}

func (s *seeder) seedSources(ctx context.Context, tx pgx.Tx) error {
	for _, ds := range dataSources {
		id := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_data_sources (id, tenant_id, name, source_type, url, reliability_score, poll_interval_seconds, is_active, last_polled_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8) ON CONFLICT (tenant_id, name) DO NOTHING`,
			id, s.tenantID, ds.Name, ds.SourceType, ds.URL, ds.Reliability, ds.PollIntervalS,
			s.now.Add(-time.Duration(s.rng.Intn(3600))*time.Second))
		if err != nil {
			return err
		}
		s.sourceIDs[ds.Name] = id
	}
	rows, err := tx.Query(ctx, `SELECT id, name FROM cti_data_sources WHERE tenant_id = $1`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id uuid.UUID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}
		s.sourceIDs[name] = id
	}
	return rows.Err()
}

// ---------------------------------------------------------------------------
// Threat actors
// ---------------------------------------------------------------------------

func (s *seeder) seedActors(ctx context.Context, tx pgx.Tx) error {
	sourceKeys := sourceIDSlice(s.sourceIDs)
	for _, a := range actors {
		id := uuid.New()
		regionID := s.regionIDs[a.OriginCountry]
		firstObs := s.now.Add(-time.Duration(180+s.rng.Intn(365*3)) * 24 * time.Hour)
		lastAct := s.now.Add(-time.Duration(s.rng.Intn(30)) * 24 * time.Hour)
		var mitrePtr *string
		if a.MitreGroupID != "" {
			mitrePtr = &a.MitreGroupID
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_threat_actors (id, tenant_id, name, aliases, actor_type, origin_country_code, origin_region_id,
				sophistication_level, primary_motivation, description, first_observed_at, last_activity_at,
				mitre_group_id, is_active, risk_score, created_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
			id, s.tenantID, a.Name, a.Aliases, a.ActorType, a.OriginCountry, regionID,
			a.Sophistication, a.Motivation, a.Description, firstObs, lastAct,
			mitrePtr, true, a.RiskScore, s.pickSource(sourceKeys))
		if err != nil {
			return err
		}
		s.actorIDs = append(s.actorIDs, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Campaigns
// ---------------------------------------------------------------------------

func (s *seeder) seedCampaigns(ctx context.Context, tx pgx.Tx) error {
	sectorCodes := []string{}
	for _, sc := range sectors {
		sectorCodes = append(sectorCodes, sc.Code)
	}

	for _, c := range campaigns {
		id := uuid.New()
		firstSeen := s.now.Add(-time.Duration(c.DaysAgo) * 24 * time.Hour)
		var lastSeen *time.Time
		ls := s.now.Add(-time.Duration(s.rng.Intn(maxInt(c.DaysAgo, 1))) * 24 * time.Hour)
		lastSeen = &ls

		var resolvedAt *time.Time
		if c.Status == "resolved" {
			ra := s.now.Add(-time.Duration(s.rng.Intn(10)+1) * 24 * time.Hour)
			resolvedAt = &ra
		}

		// Map sector indices to IDs
		var targetSectorIDs []uuid.UUID
		for _, si := range c.TargetSectors {
			if si < len(sectorCodes) {
				if sid, ok := s.sectorIDs[sectorCodes[si]]; ok {
					targetSectorIDs = append(targetSectorIDs, sid)
				}
			}
		}

		// Severity: active campaigns mostly critical/high
		sevCode := "high"
		switch {
		case c.Status == "active" && s.rng.Float64() < 0.5:
			sevCode = "critical"
		case c.Status == "monitoring":
			sevCode = "medium"
		case c.Status == "dormant":
			sevCode = "low"
		}

		_, err := tx.Exec(ctx, `
			INSERT INTO cti_campaigns (id, tenant_id, campaign_code, name, description, status, severity_id,
				primary_actor_id, target_sectors, target_description, mitre_technique_ids, ttps_summary,
				first_seen_at, last_seen_at, resolved_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			id, s.tenantID, c.Code, c.Name, c.Description, c.Status, s.severityIDs[sevCode],
			s.actorIDs[c.ActorIdx], targetSectorIDs, c.Description, c.MitreTechniques, c.TTPs,
			firstSeen, lastSeen, resolvedAt)
		if err != nil {
			return err
		}
		s.campaignIDs = append(s.campaignIDs, id)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Threat events (500+)
// ---------------------------------------------------------------------------

func (s *seeder) seedEvents(ctx context.Context, tx pgx.Tx) error {
	eventTypes := []string{"indicator_sighting", "attack_attempt", "vulnerability_exploit", "malware_detection", "anomaly", "policy_violation"}
	catCodes := []string{}
	for _, c := range categories {
		catCodes = append(catCodes, c.Code)
	}
	sourceKeys := sourceIDSlice(s.sourceIDs)

	const total = 550
	batch := &pgx.Batch{}

	for i := 0; i < total; i++ {
		id := uuid.New()

		// Temporal distribution: 40% last 7d, 30% days 8-30, 30% days 31-90
		var daysAgo int
		r := s.rng.Float64()
		switch {
		case r < 0.40:
			daysAgo = s.rng.Intn(7)
		case r < 0.70:
			daysAgo = 7 + s.rng.Intn(23)
		default:
			daysAgo = 30 + s.rng.Intn(60)
		}
		firstSeen := s.now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(s.rng.Intn(86400))*time.Second)
		lastSeen := firstSeen.Add(time.Duration(s.rng.Intn(maxInt(daysAgo*24, 1))) * time.Hour)

		// Severity distribution: 15% crit, 25% high, 35% med, 25% low
		var sevCode string
		sr := s.rng.Float64()
		switch {
		case sr < 0.15:
			sevCode = "critical"
		case sr < 0.40:
			sevCode = "high"
		case sr < 0.75:
			sevCode = "medium"
		default:
			sevCode = "low"
		}

		catCode := catCodes[s.rng.Intn(len(catCodes))]
		evtType := eventTypes[s.rng.Intn(len(eventTypes))]
		title := eventTitles[s.rng.Intn(len(eventTitles))]
		city := originCities[s.rng.Intn(len(originCities))]
		tgtCountry := targetCountries[s.rng.Intn(len(targetCountries))]
		sectorCode := sectors[s.rng.Intn(len(sectors))].Code
		conf := 0.40 + s.rng.Float64()*0.55 // 0.40 - 0.95
		srcKey := sourceKeys[s.rng.Intn(len(sourceKeys))]

		// IOC
		iocType, iocValue := s.randomIOC(i)

		// MITRE subset (1-3 techniques)
		nTech := 1 + s.rng.Intn(3)
		techs := make([]string, nTech)
		for t := 0; t < nTech; t++ {
			techs[t] = mitreTechniques[s.rng.Intn(len(mitreTechniques))]
		}

		regionID := s.regionIDs[city.Country]
		sectorID := s.sectorIDs[sectorCode]

		batch.Queue(`
			INSERT INTO cti_threat_events (id, tenant_id, event_type, title, description, severity_id, category_id,
				source_id, confidence_score, origin_latitude, origin_longitude, origin_country_code, origin_city,
				origin_region_id, target_sector_id, target_country_code, ioc_type, ioc_value,
				mitre_technique_ids, first_seen_at, last_seen_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
			id, s.tenantID, evtType, title, "Auto-generated CTI event for demo", s.severityIDs[sevCode], s.categoryIDs[catCode],
			s.sourceIDs[srcKey], math.Round(conf*100)/100, city.Lat, city.Lng, city.Country, city.Name,
			regionID, sectorID, tgtCountry, iocType, iocValue,
			techs, firstSeen, lastSeen)

		s.eventIDs = append(s.eventIDs, id)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < total; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("event %d: %w", i, err)
		}
	}
	return nil
}

func (s *seeder) randomIOC(idx int) (string, string) {
	switch s.rng.Intn(5) {
	case 0: // IP
		return "ip", fmt.Sprintf("10.%d.%d.%d", s.rng.Intn(255), s.rng.Intn(255), 1+s.rng.Intn(254))
	case 1: // domain
		words := []string{"evil", "malware", "phish", "exploit", "c2", "drop", "stage", "proxy"}
		return "domain", fmt.Sprintf("%s-%s-%04d.example.net", words[s.rng.Intn(len(words))], words[s.rng.Intn(len(words))], idx)
	case 2: // hash_sha256
		h := fmt.Sprintf("%064x", s.rng.Uint64())
		return "hash_sha256", h[:64]
	case 3: // url
		return "url", fmt.Sprintf("https://malicious-%04d.example.net/payload/%d", idx, s.rng.Intn(9999))
	default: // cve
		return "cve", fmt.Sprintf("CVE-2026-%05d", 10000+s.rng.Intn(89999))
	}
}

// ---------------------------------------------------------------------------
// Campaign IOCs (200+)
// ---------------------------------------------------------------------------

func (s *seeder) seedCampaignIOCs(ctx context.Context, tx pgx.Tx) error {
	sourceKeys := sourceIDSlice(s.sourceIDs)
	batch := &pgx.Batch{}
	count := 0

	for ci, cid := range s.campaignIDs {
		// 15-25 IOCs per campaign
		n := 15 + s.rng.Intn(11)
		for j := 0; j < n; j++ {
			iocType, iocValue := s.randomIOC(ci*100 + j)
			conf := 0.50 + s.rng.Float64()*0.45
			daysAgo := s.rng.Intn(60)
			first := s.now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(s.rng.Intn(86400))*time.Second)
			last := first.Add(time.Duration(s.rng.Intn(maxInt(daysAgo*24, 1))) * time.Hour)
			srcKey := sourceKeys[s.rng.Intn(len(sourceKeys))]

			batch.Queue(`
				INSERT INTO cti_campaign_iocs (tenant_id, campaign_id, ioc_type, ioc_value, confidence_score,
					first_seen_at, last_seen_at, is_active, source_id)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
				s.tenantID, cid, iocType, iocValue, math.Round(conf*100)/100,
				first, last, s.rng.Float64() > 0.2, s.sourceIDs[srcKey])
			count++
		}
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("ioc %d: %w", i, err)
		}
	}
	s.logger.Info().Int("count", count).Msg("campaign IOCs inserted")
	return nil
}

// ---------------------------------------------------------------------------
// Campaign-event links (300+)
// ---------------------------------------------------------------------------

func (s *seeder) seedCampaignEvents(ctx context.Context, tx pgx.Tx) error {
	batch := &pgx.Batch{}
	count := 0
	used := make(map[string]struct{})

	for _, cid := range s.campaignIDs {
		// 25-40 events per campaign
		n := 25 + s.rng.Intn(16)
		for j := 0; j < n && j < len(s.eventIDs); j++ {
			eid := s.eventIDs[s.rng.Intn(len(s.eventIDs))]
			key := cid.String() + ":" + eid.String()
			if _, exists := used[key]; exists {
				continue
			}
			used[key] = struct{}{}

			batch.Queue(`
				INSERT INTO cti_campaign_events (tenant_id, campaign_id, event_id)
				VALUES ($1,$2,$3) ON CONFLICT DO NOTHING`,
				s.tenantID, cid, eid)
			count++
		}
	}

	br := tx.SendBatch(ctx, batch)
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			return fmt.Errorf("campaign_event %d: %w", i, err)
		}
	}
	br.Close()
	s.logger.Info().Int("count", count).Msg("campaign-event links inserted")

	// Update event_count on campaigns
	_, err := tx.Exec(ctx, `
		UPDATE cti_campaigns c SET event_count = (
			SELECT count(*) FROM cti_campaign_events ce WHERE ce.campaign_id = c.id AND ce.tenant_id = c.tenant_id
		) WHERE c.tenant_id = $1`, s.tenantID)
	if err != nil {
		return fmt.Errorf("update event_count: %w", err)
	}

	// Update ioc_count
	_, err = tx.Exec(ctx, `
		UPDATE cti_campaigns c SET ioc_count = (
			SELECT count(*) FROM cti_campaign_iocs ci WHERE ci.campaign_id = c.id AND ci.tenant_id = c.tenant_id
		) WHERE c.tenant_id = $1`, s.tenantID)
	return err
}

// ---------------------------------------------------------------------------
// Brand abuse
// ---------------------------------------------------------------------------

func (s *seeder) seedBrands(ctx context.Context, tx pgx.Tx) error {
	for _, b := range brands {
		id := uuid.New()
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_monitored_brands (id, tenant_id, brand_name, domain_pattern, keywords, is_active)
			VALUES ($1,$2,$3,$4,$5,true) ON CONFLICT (tenant_id, brand_name) DO NOTHING`,
			id, s.tenantID, b.Name, b.Domain, b.Keywords)
		if err != nil {
			return err
		}
		s.brandIDs = append(s.brandIDs, id)
	}
	// Re-read
	rows, err := tx.Query(ctx, `SELECT id FROM cti_monitored_brands WHERE tenant_id = $1 ORDER BY created_at`, s.tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()
	s.brandIDs = s.brandIDs[:0]
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		s.brandIDs = append(s.brandIDs, id)
	}
	return rows.Err()
}

func (s *seeder) seedBrandAbuse(ctx context.Context, tx pgx.Tx) error {
	batch := &pgx.Batch{}
	count := 45

	sslIssuers := []string{"Let's Encrypt", "Comodo", "DigiCert", "Self-Signed", "Unknown CA"}
	riskLevels := []string{"critical", "critical", "high", "high", "high", "medium", "medium", "medium", "low"}
	sourceKeys := sourceIDSlice(s.sourceIDs)

	for i := 0; i < count; i++ {
		brandID := s.brandIDs[s.rng.Intn(len(s.brandIDs))]
		abuseType := abuseTypes[s.rng.Intn(len(abuseTypes))]
		risk := riskLevels[s.rng.Intn(len(riskLevels))]
		tdStatus := takedownStatuses[s.rng.Intn(len(takedownStatuses))]
		daysAgo := s.rng.Intn(60)
		firstDet := s.now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(s.rng.Intn(86400))*time.Second)
		lastDet := firstDet.Add(time.Duration(s.rng.Intn(maxInt(daysAgo*24, 1))) * time.Hour)
		srcKey := sourceKeys[s.rng.Intn(len(sourceKeys))]

		domain := fmt.Sprintf("clari0-%s-%04d.example.net", abuseType[:4], i)
		ip := fmt.Sprintf("203.0.113.%d", 1+s.rng.Intn(254))

		var tdReqAt, tdAt *time.Time
		if tdStatus == "takedown_requested" || tdStatus == "taken_down" {
			t := firstDet.Add(time.Duration(1+s.rng.Intn(5)) * 24 * time.Hour)
			tdReqAt = &t
		}
		if tdStatus == "taken_down" {
			t := firstDet.Add(time.Duration(3+s.rng.Intn(10)) * 24 * time.Hour)
			tdAt = &t
		}

		batch.Queue(`
			INSERT INTO cti_brand_abuse_incidents (tenant_id, brand_id, malicious_domain, abuse_type, risk_level,
				detection_count, source_id, ssl_issuer, hosting_ip, takedown_status,
				takedown_requested_at, taken_down_at, first_detected_at, last_detected_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::inet,$10,$11,$12,$13,$14)`,
			s.tenantID, brandID, domain, abuseType, risk,
			1+s.rng.Intn(50), s.sourceIDs[srcKey], sslIssuers[s.rng.Intn(len(sslIssuers))], ip, tdStatus,
			tdReqAt, tdAt, firstDet, lastDet)
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("brand_abuse %d: %w", i, err)
		}
	}
	s.logger.Info().Int("count", count).Msg("brand abuse incidents inserted")
	return nil
}

// ---------------------------------------------------------------------------
// Aggregation / dashboard tables
// ---------------------------------------------------------------------------

func (s *seeder) seedGeoSummary(ctx context.Context, tx pgx.Tx) error {
	periods := []struct {
		label string
		start time.Time
		end   time.Time
	}{
		{"24h", s.now.Add(-24 * time.Hour), s.now},
		{"7d", s.now.Add(-7 * 24 * time.Hour), s.now},
		{"30d", s.now.Add(-30 * 24 * time.Hour), s.now},
	}

	batch := &pgx.Batch{}
	count := 0

	for _, p := range periods {
		for _, city := range originCities {
			crit := s.rng.Intn(8)
			high := s.rng.Intn(15)
			med := s.rng.Intn(25)
			low := s.rng.Intn(20)
			total := crit + high + med + low
			if total == 0 {
				total = 1
				med = 1
			}
			regionID := s.regionIDs[city.Country]

			batch.Queue(`
				INSERT INTO cti_geo_threat_summary (tenant_id, country_code, city, latitude, longitude, region_id,
					severity_critical_count, severity_high_count, severity_medium_count, severity_low_count,
					total_count, top_threat_type, period_start, period_end)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
				ON CONFLICT (tenant_id, country_code, city, period_start, period_end) DO NOTHING`,
				s.tenantID, city.Country, city.Name, city.Lat, city.Lng, regionID,
				crit, high, med, low, total, "malware_detection",
				p.start, p.end)
			count++
		}
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("geo_summary %d: %w", i, err)
		}
	}
	s.logger.Info().Int("count", count).Msg("geo summaries inserted")
	return nil
}

func (s *seeder) seedSectorSummary(ctx context.Context, tx pgx.Tx) error {
	periods := []struct {
		start time.Time
		end   time.Time
	}{
		{s.now.Add(-24 * time.Hour), s.now},
		{s.now.Add(-7 * 24 * time.Hour), s.now},
		{s.now.Add(-30 * 24 * time.Hour), s.now},
	}

	batch := &pgx.Batch{}
	count := 0

	for _, p := range periods {
		for _, sc := range sectors {
			sectorID := s.sectorIDs[sc.Code]
			crit := s.rng.Intn(10)
			high := s.rng.Intn(20)
			med := s.rng.Intn(30)
			low := s.rng.Intn(25)
			total := crit + high + med + low
			if total == 0 {
				total = 1
				med = 1
			}

			batch.Queue(`
				INSERT INTO cti_sector_threat_summary (tenant_id, sector_id, severity_critical_count,
					severity_high_count, severity_medium_count, severity_low_count, total_count,
					period_start, period_end)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
				ON CONFLICT (tenant_id, sector_id, period_start, period_end) DO NOTHING`,
				s.tenantID, sectorID, crit, high, med, low, total, p.start, p.end)
			count++
		}
	}

	br := tx.SendBatch(ctx, batch)
	defer br.Close()
	for i := 0; i < count; i++ {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("sector_summary %d: %w", i, err)
		}
	}
	s.logger.Info().Int("count", count).Msg("sector summaries inserted")
	return nil
}

func (s *seeder) seedExecSnapshot(ctx context.Context, tx pgx.Tx) error {
	techSectorID := s.sectorIDs["technology"]
	_, err := tx.Exec(ctx, `
		INSERT INTO cti_executive_snapshot (tenant_id, total_events_24h, total_events_7d, total_events_30d,
			active_campaigns_count, critical_campaigns_count, total_iocs, brand_abuse_critical_count,
			brand_abuse_total_count, top_targeted_sector_id, top_threat_origin_country,
			mean_time_to_detect_hours, mean_time_to_respond_hours, risk_score_overall,
			trend_direction, trend_percentage)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)
		ON CONFLICT (tenant_id) DO UPDATE SET
			total_events_24h = EXCLUDED.total_events_24h,
			total_events_7d = EXCLUDED.total_events_7d,
			total_events_30d = EXCLUDED.total_events_30d,
			active_campaigns_count = EXCLUDED.active_campaigns_count,
			critical_campaigns_count = EXCLUDED.critical_campaigns_count,
			total_iocs = EXCLUDED.total_iocs,
			brand_abuse_critical_count = EXCLUDED.brand_abuse_critical_count,
			brand_abuse_total_count = EXCLUDED.brand_abuse_total_count,
			top_targeted_sector_id = EXCLUDED.top_targeted_sector_id,
			top_threat_origin_country = EXCLUDED.top_threat_origin_country,
			mean_time_to_detect_hours = EXCLUDED.mean_time_to_detect_hours,
			mean_time_to_respond_hours = EXCLUDED.mean_time_to_respond_hours,
			risk_score_overall = EXCLUDED.risk_score_overall,
			trend_direction = EXCLUDED.trend_direction,
			trend_percentage = EXCLUDED.trend_percentage,
			computed_at = NOW()`,
		s.tenantID, 82, 310, 550, 6, 3, 240, 8, 45,
		techSectorID, "cn", 2.40, 18.75, 73.50, "increasing", 12.30)
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func sourceIDSlice(m map[string]uuid.UUID) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func (s *seeder) pickSource(keys []string) uuid.UUID {
	return s.sourceIDs[keys[s.rng.Intn(len(keys))]]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
