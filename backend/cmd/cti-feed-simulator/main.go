// cti-feed-simulator publishes simulated CTI threat indicators to the
// cyber.cti.feed-ingestion Kafka topic for development and demo use.
//
// Usage:
//
//	go run ./cmd/cti-feed-simulator --interval=5s --events-per-tick=3 --tenant-id=<uuid>
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/cyber/cti"
	"github.com/clario360/platform/internal/events"
	"github.com/clario360/platform/internal/observability"
)

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
	{"Tehran", "ir", 35.6892, 51.3890},
	{"Pyongyang", "kp", 39.0392, 125.7625},
	{"Lagos", "ng", 6.5244, 3.3792},
	{"Sao Paulo", "br", -23.5505, -46.6333},
	{"Bucharest", "ro", 44.4268, 26.1025},
	{"Ho Chi Minh City", "vn", 10.8231, 106.6297},
	{"Istanbul", "tr", 41.0082, 28.9784},
	{"Mumbai", "in", 19.0760, 72.8777},
	{"Karachi", "pk", 24.8607, 67.0011},
	{"Jakarta", "id", -6.2088, 106.8456},
	{"Riyadh", "sa", 24.7136, 46.6753},
	{"Dubai", "ae", 25.2048, 55.2708},
	{"Berlin", "de", 52.5200, 13.4050},
	{"Seoul", "kr", 37.5665, 126.9780},
	{"Singapore", "sg", 1.3521, 103.8198},
	{"Hanoi", "vn", 21.0285, 105.8542},
}

var titles = []string{
	"Spear-phishing email with weaponized PDF detected",
	"C2 callback to known APT infrastructure",
	"Ransomware payload staged via RDP brute-force",
	"DNS tunneling exfiltration attempt",
	"Cobalt Strike beacon downloaded from staging server",
	"Supply-chain package backdoor detected",
	"Credential stuffing attack against VPN portal",
	"Watering-hole redirect to exploit kit",
	"SQL injection probe against public API",
	"SSH brute-force from Tor exit node",
	"Zero-day exploit attempt against web server",
	"DDoS amplification traffic detected",
	"Insider data exfiltration exceeding baseline",
	"Fileless PowerShell execution via WMI",
	"BEC wire-transfer impersonation attempt",
	"Malicious OAuth app requesting excessive scopes",
	"Cryptojacking script injected via compromised CDN",
	"Lateral movement via PsExec detected",
	"Wiper malware signature in network traffic",
	"Data exfiltration via HTTPS to cloud bucket",
}

var categories = []string{
	"apt", "ransomware", "phishing", "ddos", "supply_chain",
	"credential_theft", "data_exfil", "botnet", "zero_day",
	"insider_threat", "bec_fraud", "cryptojacking",
}

var sectors = []string{
	"technology", "government", "financial_services", "healthcare",
	"energy", "defense", "critical_infrastructure", "telecom",
	"education", "retail", "manufacturing", "media",
}

var mitreTechniques = []string{
	"T1566.001", "T1566.002", "T1059.001", "T1059.003",
	"T1071.001", "T1078", "T1190", "T1486",
	"T1021.002", "T1027", "T1003.001", "T1110.003",
	"T1498", "T1189", "T1567.002", "T1195.002",
	"T1561.002", "T1485", "T1528", "T1534",
}

func main() {
	var (
		kafkaBrokers  = flag.String("kafka", os.Getenv("KAFKA_BROKERS"), "Kafka bootstrap servers")
		interval      = flag.Duration("interval", 5*time.Second, "Publish interval")
		eventsPerTick = flag.Int("events-per-tick", 3, "Events per interval")
		tenantIDFlag  = flag.String("tenant-id", "aaaaaaaa-0000-0000-0000-000000000001", "Target tenant")
		seedVal       = flag.Int64("seed", 42, "Random seed")
	)
	flag.Parse()

	if *kafkaBrokers == "" {
		*kafkaBrokers = "localhost:9094"
	}

	logger := observability.NewLogger("info", "console", "cti-feed-simulator")

	kafkaCfg := config.KafkaConfig{
		Brokers: strings.Split(*kafkaBrokers, ","),
		GroupID: "cti-feed-simulator",
	}
	producer, err := events.NewProducer(kafkaCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create Kafka producer")
	}

	rng := rand.New(rand.NewSource(*seedVal))
	tenantID := *tenantIDFlag

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info().
		Dur("interval", *interval).
		Int("events_per_tick", *eventsPerTick).
		Str("tenant_id", tenantID).
		Msg("CTI feed simulator started — press Ctrl+C to stop")

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	tickCount := 0
	for {
		select {
		case <-sigCh:
			logger.Info().Msg("shutting down simulator")
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickCount++
			// Burst mode: 5x events every 60 ticks (~5 minutes at 5s interval)
			n := *eventsPerTick
			if tickCount%60 == 0 {
				n *= 5
				logger.Warn().Int("burst_count", n).Msg("BURST MODE — simulating active attack")
			}
			publishBatch(ctx, producer, rng, tenantID, n, logger)
		}
	}
}

func publishBatch(ctx context.Context, producer *events.Producer, rng *rand.Rand, tenantID string, count int, logger zerolog.Logger) {
	for i := 0; i < count; i++ {
		city := originCities[rng.Intn(len(originCities))]
		iocType, iocValue := randomIOC(rng, i)

		// Severity: weighted 15/25/35/25 critical/high/medium/low
		var severity string
		r := rng.Float64()
		switch {
		case r < 0.15:
			severity = "critical"
		case r < 0.40:
			severity = "high"
		case r < 0.75:
			severity = "medium"
		default:
			severity = "low"
		}

		// Build indicators as JSON array (generic JSON adapter format)
		indicator := map[string]interface{}{
			"id":                fmt.Sprintf("sim-%d-%d", time.Now().UnixNano(), i),
			"title":             titles[rng.Intn(len(titles))],
			"severity":          severity,
			"category":          categories[rng.Intn(len(categories))],
			"confidence":        0.4 + rng.Float64()*0.55,
			"ioc_type":          iocType,
			"ioc_value":         iocValue,
			"country":           city.Country,
			"city":              city.Name,
			"sector":            sectors[rng.Intn(len(sectors))],
			"mitre_techniques":  randomMITRE(rng),
			"tags":              []string{"simulated", "feed-simulator"},
		}

		rawData, _ := json.Marshal([]interface{}{indicator})

		payload := cti.FeedIngestionPayload{
			SourceID:   "simulator",
			SourceName: "CTI Feed Simulator",
			SourceType: "json_generic",
			TenantID:   tenantID,
			RawData:    rawData,
			ReceivedAt: time.Now().UTC(),
		}

		evt, err := events.NewEvent(cti.EventFeedRawIngested, "cti-feed-simulator", tenantID, payload)
		if err != nil {
			logger.Error().Err(err).Msg("failed to create event")
			continue
		}

		if err := producer.Publish(ctx, cti.TopicCTIFeedIngestion, evt); err != nil {
			logger.Error().Err(err).Msg("failed to publish event")
			continue
		}

		logger.Info().
			Str("ioc", fmt.Sprintf("%s:%s", iocType, iocValue)).
			Str("severity", severity).
			Str("origin", city.Name).
			Msg("published simulated indicator")
	}
}

func randomIOC(rng *rand.Rand, idx int) (string, string) {
	switch rng.Intn(4) {
	case 0:
		return "ip", fmt.Sprintf("10.%d.%d.%d", rng.Intn(99), rng.Intn(255), 1+rng.Intn(254))
	case 1:
		words := []string{"evil", "malware", "c2", "drop", "stage", "proxy", "phish"}
		return "domain", fmt.Sprintf("%s-%04d.example.net", words[rng.Intn(len(words))], idx)
	case 2:
		return "hash_sha256", fmt.Sprintf("%064x", rng.Uint64())
	default:
		return "url", fmt.Sprintf("https://malicious-%04d.example.net/payload", idx)
	}
}

func randomMITRE(rng *rand.Rand) []string {
	n := 1 + rng.Intn(3)
	out := make([]string, n)
	for i := 0; i < n; i++ {
		out[i] = mitreTechniques[rng.Intn(len(mitreTechniques))]
	}
	return out
}
