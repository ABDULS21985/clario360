// cti-seeder seeds the CTI (Cyber Threat Intelligence) tables in cyber_db
// with realistic demonstration data for development and demo environments.
//
// Usage:
//
//	go run ./cmd/cti-seeder --db-url=postgres://... --tenant-id=<uuid> --purge
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sort"
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
	{"southern_europe", "Southern Europe", "europe", 41.8719, 12.5674, ""},
	{"central_europe", "Central Europe", "europe", 47.4979, 19.0402, ""},
	{"central_asia", "Central Asia", "asia", 43.2220, 76.8512, ""},
	{"northern_america", "Northern America", "north_america", 56.1304, -106.3468, ""},
	{"central_america", "Central America", "north_america", 14.6349, -90.5069, ""},
	{"north_africa", "North Africa", "africa", 30.0444, 31.2357, ""},
	{"west_africa", "West Africa", "africa", 9.0820, 8.6753, ""},
	{"east_africa", "East Africa", "africa", -1.2921, 36.8219, ""},
	{"southern_africa", "Southern Africa", "africa", -26.2041, 28.0473, ""},
	// Countries
	{"cn", "China", "east_asia", 39.9042, 116.4074, "CHN"},
	{"ru", "Russia", "eastern_europe", 55.7558, 37.6173, "RUS"},
	{"us", "United States", "north_america", 38.9072, -77.0369, "USA"},
	{"ca", "Canada", "northern_america", 45.4215, -75.6972, "CAN"},
	{"mx", "Mexico", "central_america", 19.4326, -99.1332, "MEX"},
	{"gb", "United Kingdom", "western_europe", 51.5074, -0.1278, "GBR"},
	{"ir", "Iran", "middle_east", 35.6892, 51.3890, "IRN"},
	{"kp", "North Korea", "east_asia", 39.0392, 125.7625, "PRK"},
	{"br", "Brazil", "south_america", -15.7975, -47.8919, "BRA"},
	{"ng", "Nigeria", "west_africa", 9.0579, 7.4951, "NGA"},
	{"de", "Germany", "western_europe", 52.5200, 13.4050, "DEU"},
	{"fr", "France", "western_europe", 48.8566, 2.3522, "FRA"},
	{"es", "Spain", "southern_europe", 40.4168, -3.7038, "ESP"},
	{"it", "Italy", "southern_europe", 41.9028, 12.4964, "ITA"},
	{"pl", "Poland", "central_europe", 52.2297, 21.0122, "POL"},
	{"in", "India", "south_asia", 28.6139, 77.2090, "IND"},
	{"kr", "South Korea", "east_asia", 37.5665, 126.9780, "KOR"},
	{"il", "Israel", "middle_east", 31.7683, 35.2137, "ISR"},
	{"sa", "Saudi Arabia", "middle_east", 24.7136, 46.6753, "SAU"},
	{"ae", "UAE", "middle_east", 25.2048, 55.2708, "ARE"},
	{"ua", "Ukraine", "eastern_europe", 50.4501, 30.5234, "UKR"},
	{"pk", "Pakistan", "south_asia", 33.6844, 73.0479, "PAK"},
	{"kz", "Kazakhstan", "central_asia", 51.1694, 71.4491, "KAZ"},
	{"au", "Australia", "oceania", -33.8688, 151.2093, "AUS"},
	{"jp", "Japan", "east_asia", 35.6762, 139.6503, "JPN"},
	{"sg", "Singapore", "southeast_asia", 1.3521, 103.8198, "SGP"},
	{"nl", "Netherlands", "western_europe", 52.3676, 4.9041, "NLD"},
	{"id", "Indonesia", "southeast_asia", -6.2088, 106.8456, "IDN"},
	{"vn", "Vietnam", "southeast_asia", 21.0285, 105.8542, "VNM"},
	{"tr", "Turkey", "middle_east", 39.9334, 32.8597, "TUR"},
	{"ro", "Romania", "eastern_europe", 44.4268, 26.1025, "ROU"},
	{"ke", "Kenya", "east_africa", -1.2921, 36.8219, "KEN"},
	{"eg", "Egypt", "north_africa", 30.0444, 31.2357, "EGY"},
	{"za", "South Africa", "southern_africa", -25.7479, 28.2293, "ZAF"},
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
	Code            string
	Name            string
	Status          string
	ActorIdx        int // index into actors
	Description     string
	TTPs            string
	MitreTechniques []string
	TargetSectors   []int // indices into sectors
	TargetRegions   []string
	DaysAgo         int // first_seen offset from now
}

var campaigns = []campaignDef{
	{"C-2026-0101", "CRIMSON TEMPEST", "active", 0, "Credential-harvesting campaign targeting NATO government portals via spear-phishing", "Spear phishing with weaponized Office docs, lateral movement via PsExec, exfil over DNS", []string{"T1566.001", "T1059.001", "T1071.004", "T1078"}, []int{1, 5}, []string{"gb", "de", "fr"}, 45},
	{"C-2026-0102", "SILENT TYPHOON", "active", 1, "Supply-chain compromise targeting cloud SaaS providers to access downstream customers", "Trojanized npm packages, GitHub Actions abuse, cloud API key theft", []string{"T1195.002", "T1059.007", "T1528"}, []int{0, 8}, []string{"us", "gb", "sg"}, 30},
	{"C-2026-0103", "PHANTOM VORTEX", "active", 2, "Cryptocurrency exchange exploitation via watering-hole attacks", "Watering-hole on crypto forums, browser exploits, clipboard hijacking", []string{"T1189", "T1185", "T1115"}, []int{2}, []string{"us", "jp", "kr"}, 20},
	{"C-2026-0104", "DESERT SHADOW", "active", 3, "Espionage campaign against Gulf state energy companies via VPN exploit chains", "FortiGate CVE exploitation, Cobalt Strike beacons, SMB lateral movement", []string{"T1190", "T1059.003", "T1021.002"}, []int{4, 6}, []string{"sa", "ae"}, 15},
	{"C-2026-0105", "IRON GLACIER", "active", 4, "Wiper deployment against Ukrainian critical infrastructure during geopolitical tensions", "CaddyWiper variant, GPO abuse for deployment, MBR destruction", []string{"T1561.002", "T1484.001", "T1485"}, []int{6, 4}, []string{"ua", "pl", "ro"}, 10},
	{"C-2026-0106", "NEON HARVEST", "active", 5, "Large-scale ransomware campaign against healthcare and education via Citrix Bleed", "Citrix Bleed exploitation, data exfiltration before encryption, leak site pressure", []string{"T1190", "T1486", "T1567.002"}, []int{3, 10}, []string{"us", "gb", "ca"}, 7},
	{"C-2026-0107", "SPIDER WEB", "monitoring", 6, "BEC fraud operation impersonating CFOs at multinational firms", "Email domain spoofing, invoice redirection, wire fraud", []string{"T1566.002", "T1534"}, []int{2, 11}, []string{"us", "gb", "ng"}, 60},
	{"C-2026-0108", "DIGITAL STORM", "monitoring", 8, "DDoS campaign against media organizations covering controversial geopolitical events", "Amplification attacks, application-layer floods, Telegram coordination", []string{"T1498", "T1499"}, []int{7}, []string{"us", "gb", "fr"}, 55},
	{"C-2026-0109", "CIRCUIT BREAKER", "monitoring", 9, "Supply-chain reconnaissance targeting semiconductor fabs", "LinkedIn social engineering, vendor portal credential spray, firmware implants", []string{"T1566.003", "T1110.003", "T1195.003"}, []int{12, 0}, []string{"kr", "jp", "us"}, 50},
	{"C-2026-0110", "HYDRA LOCK", "dormant", 12, "Dormant RaaS affiliate program regrouping after law enforcement takedown", "New infrastructure setup, affiliate recruitment, testing new encryptor", []string{"T1486", "T1562.001"}, []int{3, 2}, []string{"de", "fr", "it"}, 80},
	{"C-2026-0111", "OCEAN DRIFT", "dormant", 13, "Low-level espionage probes against ASEAN maritime agencies", "Spear-phishing with RTF exploits, PlugX RAT deployment", []string{"T1566.001", "T1059.005"}, []int{1, 9}, []string{"sg", "id", "vn"}, 75},
	{"C-2026-0112", "WOLF TRAP", "resolved", 7, "Resolved credential-harvesting operation taken down via domain seizure", "Bulk phishing kits, auto-generated lookalike domains", []string{"T1566.002", "T1090.002"}, []int{2, 11}, []string{"ng", "gb", "us"}, 90},
}

type brandDef struct {
	Name     string
	Domain   string
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

var targetCountries = []string{"us", "gb", "de", "fr", "sa", "ae", "il", "jp", "kr", "au", "sg", "nl", "in", "ca", "it", "es", "pl", "za"}

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
		purge        = flag.Bool("purge", false, "Delete existing CTI rows for the tenant before seeding")
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
		pool:       pool,
		logger:     logger,
		rng:        rng,
		tenantID:   tenantID,
		purge:      *purge,
		seedUserID: deterministicID(tenantID, "principal", "cti-seeder"),
		now:        now,
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
		Bool("purge", *purge).
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
	pool       *pgxpool.Pool
	logger     zerolog.Logger
	rng        *rand.Rand
	tenantID   uuid.UUID
	purge      bool
	seedUserID uuid.UUID
	now        time.Time

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

	// Set tenant for RLS-scoped writes.
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.current_tenant_id = '%s'", s.tenantID.String())); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", s.tenantID.String())); err != nil {
		return fmt.Errorf("set tenant compatibility key: %w", err)
	}

	if s.purge {
		s.logger.Info().Str("tenant_id", s.tenantID.String()).Msg("purging CTI tenant data before seeding")
		if err := s.purgeTenantData(ctx, tx); err != nil {
			return fmt.Errorf("purge tenant data: %w", err)
		}
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

func (s *seeder) purgeTenantData(ctx context.Context, tx pgx.Tx) error {
	tables := []string{
		"cti_executive_snapshot",
		"cti_sector_threat_summary",
		"cti_geo_threat_summary",
		"cti_brand_abuse_incidents",
		"cti_monitored_brands",
		"cti_campaign_events",
		"cti_campaign_iocs",
		"cti_threat_event_tags",
		"cti_threat_events",
		"cti_campaigns",
		"cti_threat_actors",
		"cti_data_sources",
		"cti_industry_sectors",
		"cti_geographic_regions",
		"cti_threat_categories",
		"cti_threat_severity_levels",
	}

	for _, table := range tables {
		if _, err := tx.Exec(ctx, fmt.Sprintf("DELETE FROM %s WHERE tenant_id = $1", table), s.tenantID); err != nil {
			return fmt.Errorf("delete from %s: %w", table, err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// Reference table seeders
// ---------------------------------------------------------------------------

func (s *seeder) seedSeverities(ctx context.Context, tx pgx.Tx) error {
	for _, sv := range severities {
		id := deterministicID(s.tenantID, "severity", sv.Code)
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_threat_severity_levels (id, tenant_id, code, label, color_hex, sort_order, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (tenant_id, code) DO UPDATE SET
				label = EXCLUDED.label,
				color_hex = EXCLUDED.color_hex,
				sort_order = EXCLUDED.sort_order,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by`,
			id, s.tenantID, sv.Code, sv.Label, sv.ColorHex, sv.SortOrder, s.seedUserID, s.seedUserID)
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
		id := deterministicID(s.tenantID, "category", c.Code)
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_threat_categories (id, tenant_id, code, label, description, mitre_tactic_ids, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (tenant_id, code) DO UPDATE SET
				label = EXCLUDED.label,
				description = EXCLUDED.description,
				mitre_tactic_ids = EXCLUDED.mitre_tactic_ids,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by`,
			id, s.tenantID, c.Code, c.Label, c.Description, c.MitreTIDs, s.seedUserID, s.seedUserID)
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
		id := deterministicID(s.tenantID, "region", r.Code)
		var isoPtr *string
		if r.ISOCountry != "" {
			isoPtr = &r.ISOCountry
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_geographic_regions (id, tenant_id, code, label, latitude, longitude, iso_country_code, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
			ON CONFLICT (tenant_id, code) DO UPDATE SET
				label = EXCLUDED.label,
				latitude = EXCLUDED.latitude,
				longitude = EXCLUDED.longitude,
				iso_country_code = EXCLUDED.iso_country_code,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by`,
			id, s.tenantID, r.Code, r.Label, r.Lat, r.Lng, isoPtr, s.seedUserID, s.seedUserID)
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
		_, err := tx.Exec(ctx, `
			UPDATE cti_geographic_regions
			SET parent_region_id = $1, updated_at = NOW(), updated_by = $3
			WHERE id = $2`,
			parentID, selfID, s.seedUserID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *seeder) seedSectors(ctx context.Context, tx pgx.Tx) error {
	for _, sc := range sectors {
		id := deterministicID(s.tenantID, "sector", sc.Code)
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_industry_sectors (id, tenant_id, code, label, description, naics_code, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
			ON CONFLICT (tenant_id, code) DO UPDATE SET
				label = EXCLUDED.label,
				description = EXCLUDED.description,
				naics_code = EXCLUDED.naics_code,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by`,
			id, s.tenantID, sc.Code, sc.Label, sc.Description, sc.NAICS, s.seedUserID, s.seedUserID)
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
		id := deterministicID(s.tenantID, "source", ds.Name)
		_, err := tx.Exec(ctx, `
			INSERT INTO cti_data_sources (id, tenant_id, name, source_type, url, reliability_score, poll_interval_seconds, is_active, last_polled_at, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,true,$8,$9,$10)
			ON CONFLICT (tenant_id, name) DO UPDATE SET
				source_type = EXCLUDED.source_type,
				url = EXCLUDED.url,
				reliability_score = EXCLUDED.reliability_score,
				poll_interval_seconds = EXCLUDED.poll_interval_seconds,
				is_active = EXCLUDED.is_active,
				last_polled_at = EXCLUDED.last_polled_at,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by`,
			id, s.tenantID, ds.Name, ds.SourceType, ds.URL, ds.Reliability, ds.PollIntervalS,
			s.now.Add(-time.Duration(s.rng.Intn(3600))*time.Second), s.seedUserID, s.seedUserID)
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
	s.actorIDs = s.actorIDs[:0]
	for _, a := range actors {
		id := deterministicID(s.tenantID, "actor", a.Name)
		var existingID uuid.UUID
		err := tx.QueryRow(ctx, `SELECT id FROM cti_threat_actors WHERE tenant_id = $1 AND name = $2 LIMIT 1`, s.tenantID, a.Name).Scan(&existingID)
		if err == nil {
			id = existingID
		} else if err != pgx.ErrNoRows {
			return err
		}
		regionID := s.regionIDs[a.OriginCountry]
		firstObs := s.now.Add(-time.Duration(180+s.rng.Intn(365*3)) * 24 * time.Hour)
		lastAct := s.now.Add(-time.Duration(s.rng.Intn(30)) * 24 * time.Hour)
		var mitrePtr *string
		if a.MitreGroupID != "" {
			mitrePtr = &a.MitreGroupID
		}
		_, err = tx.Exec(ctx, `
				INSERT INTO cti_threat_actors (id, tenant_id, name, aliases, actor_type, origin_country_code, origin_region_id,
					sophistication_level, primary_motivation, description, first_observed_at, last_activity_at,
					mitre_group_id, is_active, risk_score, created_by, updated_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				aliases = EXCLUDED.aliases,
				actor_type = EXCLUDED.actor_type,
				origin_country_code = EXCLUDED.origin_country_code,
				origin_region_id = EXCLUDED.origin_region_id,
				sophistication_level = EXCLUDED.sophistication_level,
				primary_motivation = EXCLUDED.primary_motivation,
				description = EXCLUDED.description,
				first_observed_at = EXCLUDED.first_observed_at,
				last_activity_at = EXCLUDED.last_activity_at,
				mitre_group_id = EXCLUDED.mitre_group_id,
				is_active = EXCLUDED.is_active,
				risk_score = EXCLUDED.risk_score,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by,
				deleted_at = NULL`,
			id, s.tenantID, a.Name, a.Aliases, a.ActorType, a.OriginCountry, regionID,
			a.Sophistication, a.Motivation, a.Description, firstObs, lastAct,
			mitrePtr, true, a.RiskScore, s.seedUserID, s.seedUserID)
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
	s.campaignIDs = s.campaignIDs[:0]
	sectorCodes := []string{}
	for _, sc := range sectors {
		sectorCodes = append(sectorCodes, sc.Code)
	}

	for _, c := range campaigns {
		id := deterministicID(s.tenantID, "campaign", c.Code)
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
		var targetRegionIDs []uuid.UUID
		for _, code := range c.TargetRegions {
			if rid, ok := s.regionIDs[code]; ok {
				targetRegionIDs = append(targetRegionIDs, rid)
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

		err := tx.QueryRow(ctx, `
			INSERT INTO cti_campaigns (id, tenant_id, campaign_code, name, description, status, severity_id,
				primary_actor_id, target_sectors, target_regions, target_description, mitre_technique_ids, ttps_summary,
				first_seen_at, last_seen_at, resolved_at, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)
			ON CONFLICT (tenant_id, campaign_code) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				status = EXCLUDED.status,
				severity_id = EXCLUDED.severity_id,
				primary_actor_id = EXCLUDED.primary_actor_id,
				target_sectors = EXCLUDED.target_sectors,
				target_regions = EXCLUDED.target_regions,
				target_description = EXCLUDED.target_description,
				mitre_technique_ids = EXCLUDED.mitre_technique_ids,
				ttps_summary = EXCLUDED.ttps_summary,
				first_seen_at = EXCLUDED.first_seen_at,
				last_seen_at = EXCLUDED.last_seen_at,
				resolved_at = EXCLUDED.resolved_at,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by,
				deleted_at = NULL
			RETURNING id`,
			id, s.tenantID, c.Code, c.Name, c.Description, c.Status, s.severityIDs[sevCode],
			s.actorIDs[c.ActorIdx], targetSectorIDs, targetRegionIDs, c.Description, c.MitreTechniques, c.TTPs,
			firstSeen, lastSeen, resolvedAt, s.seedUserID, s.seedUserID).Scan(&id)
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
	s.eventIDs = s.eventIDs[:0]

	for i := 0; i < total; i++ {
		id := deterministicID(s.tenantID, "event", fmt.Sprintf("%03d", i))

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
		seedRef := fmt.Sprintf("seed:event:%03d", i)
		payload, err := json.Marshal(map[string]any{
			"generator":   "cti-seeder",
			"seed_key":    seedRef,
			"event_index": i,
		})
		if err != nil {
			return err
		}
		existingID, err := s.findThreatEventID(ctx, tx, seedRef, title, iocType, iocValue, firstSeen)
		if err != nil {
			return err
		}
		if existingID != nil {
			id = *existingID
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO cti_threat_events (id, tenant_id, event_type, title, description, severity_id, category_id,
				source_id, source_reference, confidence_score, origin_latitude, origin_longitude, origin_country_code, origin_city,
				origin_region_id, target_sector_id, target_country_code, ioc_type, ioc_value,
				mitre_technique_ids, raw_payload, first_seen_at, last_seen_at, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25)
			ON CONFLICT (id) DO UPDATE SET
				event_type = EXCLUDED.event_type,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				severity_id = EXCLUDED.severity_id,
				category_id = EXCLUDED.category_id,
				source_id = EXCLUDED.source_id,
				source_reference = EXCLUDED.source_reference,
				confidence_score = EXCLUDED.confidence_score,
				origin_latitude = EXCLUDED.origin_latitude,
				origin_longitude = EXCLUDED.origin_longitude,
				origin_country_code = EXCLUDED.origin_country_code,
				origin_city = EXCLUDED.origin_city,
				origin_region_id = EXCLUDED.origin_region_id,
				target_sector_id = EXCLUDED.target_sector_id,
				target_country_code = EXCLUDED.target_country_code,
				ioc_type = EXCLUDED.ioc_type,
				ioc_value = EXCLUDED.ioc_value,
				mitre_technique_ids = EXCLUDED.mitre_technique_ids,
				raw_payload = EXCLUDED.raw_payload,
				first_seen_at = EXCLUDED.first_seen_at,
				last_seen_at = EXCLUDED.last_seen_at,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by,
				deleted_at = NULL`,
			id, s.tenantID, evtType, title, "Auto-generated CTI event for demo", s.severityIDs[sevCode], s.categoryIDs[catCode],
			s.sourceIDs[srcKey], seedRef, math.Round(conf*100)/100, city.Lat, city.Lng, city.Country, city.Name,
			regionID, sectorID, tgtCountry, iocType, iocValue,
			techs, payload, firstSeen, lastSeen, s.seedUserID, s.seedUserID)
		if err != nil {
			return fmt.Errorf("event %d: %w", i, err)
		}
		s.eventIDs = append(s.eventIDs, id)
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
	count := 0

	for ci, cid := range s.campaignIDs {
		const n = 20
		for j := 0; j < n; j++ {
			id := deterministicID(s.tenantID, "campaign-ioc", fmt.Sprintf("%02d", ci), fmt.Sprintf("%02d", j))
			iocType, iocValue := s.randomIOC(ci*100 + j)
			conf := 0.50 + s.rng.Float64()*0.45
			daysAgo := s.rng.Intn(60)
			first := s.now.Add(-time.Duration(daysAgo)*24*time.Hour - time.Duration(s.rng.Intn(86400))*time.Second)
			last := first.Add(time.Duration(s.rng.Intn(maxInt(daysAgo*24, 1))) * time.Hour)
			srcKey := sourceKeys[s.rng.Intn(len(sourceKeys))]
			existingID, err := s.findCampaignIOCID(ctx, tx, cid, iocType, iocValue)
			if err != nil {
				return err
			}
			if existingID != nil {
				id = *existingID
			}
			_, err = tx.Exec(ctx, `
				INSERT INTO cti_campaign_iocs (id, tenant_id, campaign_id, ioc_type, ioc_value, confidence_score,
					first_seen_at, last_seen_at, is_active, source_id, created_by, updated_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
				ON CONFLICT (id) DO UPDATE SET
					campaign_id = EXCLUDED.campaign_id,
					ioc_type = EXCLUDED.ioc_type,
					ioc_value = EXCLUDED.ioc_value,
					confidence_score = EXCLUDED.confidence_score,
					first_seen_at = EXCLUDED.first_seen_at,
					last_seen_at = EXCLUDED.last_seen_at,
					is_active = EXCLUDED.is_active,
					source_id = EXCLUDED.source_id,
					updated_at = NOW(),
					updated_by = EXCLUDED.updated_by`,
				id, s.tenantID, cid, iocType, iocValue, math.Round(conf*100)/100,
				first, last, s.rng.Float64() > 0.2, s.sourceIDs[srcKey], s.seedUserID, s.seedUserID)
			if err != nil {
				return fmt.Errorf("ioc %d: %w", count, err)
			}
			count++
		}
	}
	s.logger.Info().Int("count", count).Msg("campaign IOCs inserted")
	return nil
}

// ---------------------------------------------------------------------------
// Campaign-event links (300+)
// ---------------------------------------------------------------------------

func (s *seeder) seedCampaignEvents(ctx context.Context, tx pgx.Tx) error {
	count := 0

	for ci, cid := range s.campaignIDs {
		const n = 32
		for j := 0; j < n && j < len(s.eventIDs); j++ {
			eid := s.eventIDs[(ci*41+j*17)%len(s.eventIDs)]
			linkID := deterministicID(s.tenantID, "campaign-event", cid.String(), eid.String())
			linkedAt := s.now.Add(-time.Duration((ci+j)%14) * 24 * time.Hour)
			_, err := tx.Exec(ctx, `
				INSERT INTO cti_campaign_events (id, tenant_id, campaign_id, event_id, linked_at, linked_by, created_at, updated_at, created_by, updated_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
				ON CONFLICT (tenant_id, campaign_id, event_id) DO UPDATE SET
					linked_at = EXCLUDED.linked_at,
					linked_by = EXCLUDED.linked_by,
					updated_at = NOW(),
					updated_by = EXCLUDED.updated_by`,
				linkID, s.tenantID, cid, eid, linkedAt, s.seedUserID, linkedAt, linkedAt, s.seedUserID, s.seedUserID)
			if err != nil {
				return fmt.Errorf("campaign_event %d: %w", count, err)
			}
			count++
		}
	}
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
	s.brandIDs = s.brandIDs[:0]
	for _, b := range brands {
		id := deterministicID(s.tenantID, "brand", b.Name)
		err := tx.QueryRow(ctx, `
			INSERT INTO cti_monitored_brands (id, tenant_id, brand_name, domain_pattern, keywords, is_active, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,true,$6,$7)
			ON CONFLICT (tenant_id, brand_name) DO UPDATE SET
				domain_pattern = EXCLUDED.domain_pattern,
				keywords = EXCLUDED.keywords,
				is_active = EXCLUDED.is_active,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by
			RETURNING id`,
			id, s.tenantID, b.Name, b.Domain, b.Keywords, s.seedUserID, s.seedUserID).Scan(&id)
		if err != nil {
			return err
		}
		s.brandIDs = append(s.brandIDs, id)
	}
	return nil
}

func (s *seeder) seedBrandAbuse(ctx context.Context, tx pgx.Tx) error {
	count := 45

	sslIssuers := []string{"Let's Encrypt", "Comodo", "DigiCert", "Self-Signed", "Unknown CA"}
	riskLevels := []string{"critical", "critical", "high", "high", "high", "medium", "medium", "medium", "low"}
	whoisRegistrants := []string{"Privacy Protect LLC", "NameShield Holdings", "NorthBridge Domains", "Blue Harbor Registration", "Red Maple Proxy"}
	hostingASNs := []string{"AS13335", "AS16509", "AS15169", "AS8075", "AS9009", "AS32787"}
	incidentRegions := []string{"us", "gb", "de", "fr", "ng", "za", "sg", "ae", "jp", "es", "it", "ca"}
	sourceKeys := sourceIDSlice(s.sourceIDs)

	for i := 0; i < count; i++ {
		id := deterministicID(s.tenantID, "brand-abuse", fmt.Sprintf("%03d", i))
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
		regionCode := incidentRegions[s.rng.Intn(len(incidentRegions))]
		regionID := s.regionIDs[regionCode]
		whoisCreated := firstDet.AddDate(0, 0, -(7 + s.rng.Intn(90)))
		screenshotID := deterministicID(s.tenantID, "brand-abuse-shot", domain)

		var tdReqAt, tdAt *time.Time
		if tdStatus == "takedown_requested" || tdStatus == "taken_down" {
			t := firstDet.Add(time.Duration(1+s.rng.Intn(5)) * 24 * time.Hour)
			tdReqAt = &t
		}
		if tdStatus == "taken_down" {
			t := firstDet.Add(time.Duration(3+s.rng.Intn(10)) * 24 * time.Hour)
			tdAt = &t
		}
		existingID, err := s.findBrandAbuseIncidentID(ctx, tx, domain)
		if err != nil {
			return err
		}
		if existingID != nil {
			id = *existingID
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO cti_brand_abuse_incidents (id, tenant_id, brand_id, malicious_domain, abuse_type, risk_level,
				region_id, detection_count, source_id, whois_registrant, whois_created_date, ssl_issuer, hosting_ip,
				hosting_asn, screenshot_file_id, takedown_status, takedown_requested_at, taken_down_at,
				first_detected_at, last_detected_at, created_by, updated_by)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::inet,$14,$15,$16,$17,$18,$19,$20,$21,$22)
			ON CONFLICT (id) DO UPDATE SET
				brand_id = EXCLUDED.brand_id,
				malicious_domain = EXCLUDED.malicious_domain,
				abuse_type = EXCLUDED.abuse_type,
				risk_level = EXCLUDED.risk_level,
				region_id = EXCLUDED.region_id,
				detection_count = EXCLUDED.detection_count,
				source_id = EXCLUDED.source_id,
				whois_registrant = EXCLUDED.whois_registrant,
				whois_created_date = EXCLUDED.whois_created_date,
				ssl_issuer = EXCLUDED.ssl_issuer,
				hosting_ip = EXCLUDED.hosting_ip,
				hosting_asn = EXCLUDED.hosting_asn,
				screenshot_file_id = EXCLUDED.screenshot_file_id,
				takedown_status = EXCLUDED.takedown_status,
				takedown_requested_at = EXCLUDED.takedown_requested_at,
				taken_down_at = EXCLUDED.taken_down_at,
				first_detected_at = EXCLUDED.first_detected_at,
				last_detected_at = EXCLUDED.last_detected_at,
				updated_at = NOW(),
				updated_by = EXCLUDED.updated_by,
				deleted_at = NULL`,
			id, s.tenantID, brandID, domain, abuseType, risk,
			regionID, 1+s.rng.Intn(50), s.sourceIDs[srcKey], whoisRegistrants[s.rng.Intn(len(whoisRegistrants))], whoisCreated,
			sslIssuers[s.rng.Intn(len(sslIssuers))], ip, hostingASNs[s.rng.Intn(len(hostingASNs))], screenshotID,
			tdStatus, tdReqAt, tdAt, firstDet, lastDet, s.seedUserID, s.seedUserID)
		if err != nil {
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

	count := 0
	categoryCodes := make([]string, 0, len(categories))
	for _, c := range categories {
		categoryCodes = append(categoryCodes, c.Code)
	}

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
			topCategoryID := s.categoryIDs[categoryCodes[s.rng.Intn(len(categoryCodes))]]
			_, err := tx.Exec(ctx, `
				INSERT INTO cti_geo_threat_summary (tenant_id, country_code, city, latitude, longitude, region_id,
					severity_critical_count, severity_high_count, severity_medium_count, severity_low_count,
					total_count, top_category_id, top_threat_type, period_start, period_end, created_by, updated_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
				ON CONFLICT (tenant_id, country_code, city, period_start, period_end) DO UPDATE SET
					latitude = EXCLUDED.latitude,
					longitude = EXCLUDED.longitude,
					region_id = EXCLUDED.region_id,
					severity_critical_count = EXCLUDED.severity_critical_count,
					severity_high_count = EXCLUDED.severity_high_count,
					severity_medium_count = EXCLUDED.severity_medium_count,
					severity_low_count = EXCLUDED.severity_low_count,
					total_count = EXCLUDED.total_count,
					top_category_id = EXCLUDED.top_category_id,
					top_threat_type = EXCLUDED.top_threat_type,
					computed_at = NOW(),
					updated_at = NOW(),
					updated_by = EXCLUDED.updated_by`,
				s.tenantID, city.Country, city.Name, city.Lat, city.Lng, regionID,
				crit, high, med, low, total, topCategoryID, "malware_detection",
				p.start, p.end, s.seedUserID, s.seedUserID)
			if err != nil {
				return fmt.Errorf("geo_summary %d: %w", count, err)
			}
			count++
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
			_, err := tx.Exec(ctx, `
				INSERT INTO cti_sector_threat_summary (tenant_id, sector_id, severity_critical_count,
					severity_high_count, severity_medium_count, severity_low_count, total_count,
					period_start, period_end, created_by, updated_by)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
				ON CONFLICT (tenant_id, sector_id, period_start, period_end) DO UPDATE SET
					severity_critical_count = EXCLUDED.severity_critical_count,
					severity_high_count = EXCLUDED.severity_high_count,
					severity_medium_count = EXCLUDED.severity_medium_count,
					severity_low_count = EXCLUDED.severity_low_count,
					total_count = EXCLUDED.total_count,
					computed_at = NOW(),
					updated_at = NOW(),
					updated_by = EXCLUDED.updated_by`,
				s.tenantID, sectorID, crit, high, med, low, total, p.start, p.end, s.seedUserID, s.seedUserID)
			if err != nil {
				return fmt.Errorf("sector_summary %d: %w", count, err)
			}
			count++
		}
	}
	s.logger.Info().Int("count", count).Msg("sector summaries inserted")
	return nil
}

func (s *seeder) seedExecSnapshot(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		WITH event_stats AS (
			SELECT
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '24 hours')::integer AS total_events_24h,
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '7 days')::integer AS total_events_7d,
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '30 days')::integer AS total_events_30d,
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '7 days' AND sl.code = 'critical')::integer AS critical_events_7d,
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '7 days' AND sl.code = 'high')::integer AS high_events_7d
			FROM cti_threat_events e
			LEFT JOIN cti_threat_severity_levels sl ON sl.id = e.severity_id
			WHERE e.tenant_id = $1
		),
		campaign_stats AS (
			SELECT
				count(*) FILTER (WHERE c.deleted_at IS NULL AND c.status = 'active')::integer AS active_campaigns_count,
				count(*) FILTER (WHERE c.deleted_at IS NULL AND c.status = 'active' AND sl.code = 'critical')::integer AS critical_campaigns_count
			FROM cti_campaigns c
			LEFT JOIN cti_threat_severity_levels sl ON sl.id = c.severity_id
			WHERE c.tenant_id = $1
		),
		ioc_stats AS (
			SELECT count(*)::integer AS total_iocs
			FROM cti_campaign_iocs
			WHERE tenant_id = $1
		),
		brand_stats AS (
			SELECT
				count(*) FILTER (WHERE deleted_at IS NULL AND risk_level = 'critical')::integer AS brand_abuse_critical_count,
				count(*) FILTER (WHERE deleted_at IS NULL)::integer AS brand_abuse_total_count
			FROM cti_brand_abuse_incidents
			WHERE tenant_id = $1
		),
		top_sector AS (
			SELECT target_sector_id
			FROM cti_threat_events
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND target_sector_id IS NOT NULL
			GROUP BY target_sector_id
			ORDER BY count(*) DESC, target_sector_id
			LIMIT 1
		),
		top_country AS (
			SELECT origin_country_code
			FROM cti_threat_events
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
			  AND origin_country_code IS NOT NULL
			GROUP BY origin_country_code
			ORDER BY count(*) DESC, origin_country_code
			LIMIT 1
		),
		detect_metrics AS (
			SELECT round(avg(EXTRACT(EPOCH FROM GREATEST(last_seen_at - first_seen_at, INTERVAL '0 seconds')) / 3600.0)::numeric, 2) AS mean_time_to_detect_hours
			FROM cti_threat_events
			WHERE tenant_id = $1
			  AND deleted_at IS NULL
		),
		response_metrics AS (
			SELECT round(avg(hours)::numeric, 2) AS mean_time_to_respond_hours
			FROM (
				SELECT EXTRACT(EPOCH FROM GREATEST(resolved_at - first_seen_at, INTERVAL '0 seconds')) / 3600.0 AS hours
				FROM cti_campaigns
				WHERE tenant_id = $1
				  AND deleted_at IS NULL
				  AND resolved_at IS NOT NULL
				UNION ALL
				SELECT EXTRACT(EPOCH FROM GREATEST(taken_down_at - first_detected_at, INTERVAL '0 seconds')) / 3600.0 AS hours
				FROM cti_brand_abuse_incidents
				WHERE tenant_id = $1
				  AND deleted_at IS NULL
				  AND taken_down_at IS NOT NULL
			) durations
		),
		trend_stats AS (
			SELECT
				count(*) FILTER (WHERE deleted_at IS NULL AND first_seen_at > NOW() - INTERVAL '7 days')::integer AS current_events_7d,
				count(*) FILTER (
					WHERE deleted_at IS NULL
					  AND first_seen_at <= NOW() - INTERVAL '7 days'
					  AND first_seen_at > NOW() - INTERVAL '14 days'
				)::integer AS previous_events_7d
			FROM cti_threat_events
			WHERE tenant_id = $1
		)
		INSERT INTO cti_executive_snapshot (
			tenant_id, total_events_24h, total_events_7d, total_events_30d,
			active_campaigns_count, critical_campaigns_count, total_iocs,
			brand_abuse_critical_count, brand_abuse_total_count,
			top_targeted_sector_id, top_threat_origin_country,
			mean_time_to_detect_hours, mean_time_to_respond_hours,
			risk_score_overall, trend_direction, trend_percentage,
			created_by, updated_by
		)
		SELECT
			$1,
			es.total_events_24h,
			es.total_events_7d,
			es.total_events_30d,
			cs.active_campaigns_count,
			cs.critical_campaigns_count,
			is1.total_iocs,
			bs.brand_abuse_critical_count,
			bs.brand_abuse_total_count,
			(SELECT target_sector_id FROM top_sector),
			(SELECT origin_country_code FROM top_country),
			dm.mean_time_to_detect_hours,
			rm.mean_time_to_respond_hours,
			round(LEAST(
				100::numeric,
				(es.critical_events_7d::numeric * 1.50) +
				(es.high_events_7d::numeric * 0.75) +
				(cs.active_campaigns_count::numeric * 4.00) +
				(cs.critical_campaigns_count::numeric * 6.00) +
				(bs.brand_abuse_critical_count::numeric * 2.50) +
				(is1.total_iocs::numeric * 0.05)
			), 2),
			CASE
				WHEN ts.previous_events_7d = 0 AND ts.current_events_7d = 0 THEN 'stable'
				WHEN ts.previous_events_7d = 0 THEN 'increasing'
				WHEN abs(((ts.current_events_7d - ts.previous_events_7d)::numeric / ts.previous_events_7d::numeric) * 100) < 5 THEN 'stable'
				WHEN ts.current_events_7d > ts.previous_events_7d THEN 'increasing'
				ELSE 'decreasing'
			END,
			CASE
				WHEN ts.previous_events_7d = 0 AND ts.current_events_7d = 0 THEN 0
				WHEN ts.previous_events_7d = 0 THEN 100
				ELSE round(((ts.current_events_7d - ts.previous_events_7d)::numeric / ts.previous_events_7d::numeric) * 100, 2)
			END,
			$2,
			$2
		FROM event_stats es
		CROSS JOIN campaign_stats cs
		CROSS JOIN ioc_stats is1
		CROSS JOIN brand_stats bs
		CROSS JOIN detect_metrics dm
		CROSS JOIN response_metrics rm
		CROSS JOIN trend_stats ts
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
			computed_at = NOW(),
			updated_at = NOW(),
			updated_by = EXCLUDED.updated_by`,
		s.tenantID, s.seedUserID)
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
	sort.Strings(keys)
	return keys
}

func (s *seeder) pickSource(keys []string) uuid.UUID {
	return s.sourceIDs[keys[s.rng.Intn(len(keys))]]
}

func deterministicID(tenantID uuid.UUID, kind string, parts ...string) uuid.UUID {
	payload := tenantID.String() + "|" + kind
	if len(parts) > 0 {
		payload += "|" + strings.Join(parts, "|")
	}
	return uuid.NewSHA1(uuid.NameSpaceURL, []byte(payload))
}

func (s *seeder) findThreatEventID(ctx context.Context, tx pgx.Tx, sourceReference, title, iocType, iocValue string, firstSeen time.Time) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM cti_threat_events
		WHERE tenant_id = $1
		  AND deleted_at IS NULL
		  AND (
			source_reference = $2
			OR (
				source_reference IS NULL
				AND title = $3
				AND ioc_type = $4
				AND ioc_value = $5
				AND first_seen_at = $6
				AND description = 'Auto-generated CTI event for demo'
			)
		  )
		ORDER BY created_at
		LIMIT 1`,
		s.tenantID, sourceReference, title, iocType, iocValue, firstSeen).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *seeder) findCampaignIOCID(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID, iocType, iocValue string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM cti_campaign_iocs
		WHERE tenant_id = $1 AND campaign_id = $2 AND ioc_type = $3 AND ioc_value = $4
		ORDER BY created_at
		LIMIT 1`,
		s.tenantID, campaignID, iocType, iocValue).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func (s *seeder) findBrandAbuseIncidentID(ctx context.Context, tx pgx.Tx, domain string) (*uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT id
		FROM cti_brand_abuse_incidents
		WHERE tenant_id = $1 AND malicious_domain = $2
		ORDER BY created_at
		LIMIT 1`,
		s.tenantID, domain).Scan(&id)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
