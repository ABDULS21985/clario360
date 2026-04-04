package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/cyber/vciso/predict/feeds"
	predictmodels "github.com/clario360/platform/internal/cyber/vciso/predict/models"
	predictrepo "github.com/clario360/platform/internal/cyber/vciso/predict/repository"
)

type FeatureStore struct {
	db          *pgxpool.Pool
	features    *predictrepo.FeatureRepository
	ingester    *feeds.ThreatFeedIngester
	cveEnricher *feeds.CVEEnricher
	benchmarker *feeds.IndustryBenchmarker
	darkWeb     *feeds.DarkWebMonitor
	logger      zerolog.Logger
	now         func() time.Time
}

func NewFeatureStore(
	db *pgxpool.Pool,
	featureRepo *predictrepo.FeatureRepository,
	ingester *feeds.ThreatFeedIngester,
	cveEnricher *feeds.CVEEnricher,
	benchmarker *feeds.IndustryBenchmarker,
	darkWeb *feeds.DarkWebMonitor,
	logger zerolog.Logger,
) *FeatureStore {
	return &FeatureStore{
		db:          db,
		features:    featureRepo,
		ingester:    ingester,
		cveEnricher: cveEnricher,
		benchmarker: benchmarker,
		darkWeb:     darkWeb,
		logger:      logger.With().Str("component", "vciso_predict_feature_store").Logger(),
		now:         func() time.Time { return time.Now().UTC() },
	}
}

func (s *FeatureStore) AlertVolumeSamples(ctx context.Context, tenantID uuid.UUID, lookbackDays int) ([]predictmodels.AlertVolumeSample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	if lookbackDays <= 0 {
		lookbackDays = 365
	}
	start := s.now().AddDate(0, 0, -lookbackDays)
	alertCounts, err := s.aggregateDailyCounts(ctx, tenantID, `
		SELECT date_trunc('day', created_at)::date AS day, COUNT(*)::float8
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2
		GROUP BY day
	`, start)
	if err != nil {
		return nil, err
	}
	threatCounts, err := s.aggregateDailyCounts(ctx, tenantID, `
		SELECT date_trunc('day', detected_at)::date AS day, COUNT(*)::float8
		FROM threats
		WHERE tenant_id = $1 AND detected_at >= $2
		GROUP BY day
	`, start)
	if err != nil {
		return nil, err
	}
	assetOnboarding, err := s.aggregateDailyCounts(ctx, tenantID, `
		SELECT date_trunc('day', created_at)::date AS day, COUNT(*)::float8
		FROM assets
		WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2
		GROUP BY day
	`, start)
	if err != nil {
		return nil, err
	}
	detectionChanges, err := s.aggregateDailyCounts(ctx, tenantID, `
		SELECT date_trunc('day', created_at)::date AS day, COUNT(DISTINCT COALESCE(rule_id, gen_random_uuid()))::float8
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2
		GROUP BY day
	`, start)
	if err != nil {
		return nil, err
	}
	darkWebScore := 0.0
	if s.darkWeb != nil {
		if score, scoreErr := s.darkWeb.RiskScore(ctx, tenantID, start); scoreErr == nil {
			darkWebScore = score
		}
	}
	out := make([]predictmodels.AlertVolumeSample, 0, lookbackDays)
	for day := 0; day < lookbackDays; day++ {
		timestamp := dateOnly(start.AddDate(0, 0, day))
		maintenance := 0.0
		holiday := 0.0
		if timestamp.Weekday() == time.Saturday || timestamp.Weekday() == time.Sunday {
			maintenance = 0.25
			holiday = 1.0
		}
		item := predictmodels.AlertVolumeSample{
			Timestamp:         timestamp,
			AlertCount:        alertCounts[timestamp],
			ThreatActivity:    threatCounts[timestamp] + darkWebScore/float64(max(1, lookbackDays)),
			AssetOnboarding:   assetOnboarding[timestamp],
			DetectionChanges:  detectionChanges[timestamp],
			MaintenanceWindow: maintenance,
			Holiday:           holiday,
		}
		out = append(out, item)
		_ = s.persistVector(ctx, tenantID, "alert_volume", "day", nil, item)
	}
	return out, nil
}

func (s *FeatureStore) AssetRiskSamples(ctx context.Context, tenantID uuid.UUID, assetType string) ([]predictmodels.AssetRiskSample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	benchmark, err := s.industryBenchmark(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	query := `
		SELECT
			a.id,
			a.name,
			a.type::text,
			a.criticality::text,
			COUNT(*) FILTER (WHERE v.deleted_at IS NULL AND v.status IN ('open','in_progress') AND v.severity = 'critical')::float8 AS open_critical,
			COUNT(*) FILTER (WHERE v.deleted_at IS NULL AND v.status IN ('open','in_progress') AND v.severity = 'high')::float8 AS open_high,
			EXTRACT(EPOCH FROM (now() - COALESCE(NULLIF(a.metadata->>'last_patch_at','')::timestamptz, a.updated_at))) / 86400.0 AS patch_age_days,
			CASE
				WHEN COALESCE(NULLIF(a.metadata->>'internet_facing','')::boolean, false) THEN 1.0
				WHEN a.ip_address IS NOT NULL THEN 1.0
				ELSE 0.0
			END AS internet_facing,
			COALESCE(alerts.alert_count, 0)::float8 AS historical_alerts,
			COALESCE(NULLIF(a.metadata->>'user_access_count','')::float8, CASE WHEN a.owner IS NOT NULL THEN 5.0 ELSE 1.0 END) AS user_access_count,
			COALESCE(NULLIF(a.metadata->>'data_sensitivity','')::float8,
				CASE a.criticality::text WHEN 'critical' THEN 1.0 WHEN 'high' THEN 0.75 WHEN 'medium' THEN 0.50 ELSE 0.25 END
			) AS data_sensitivity,
			CASE WHEN COALESCE(targeted.targeted_count, 0) > 0 THEN 1.0 ELSE 0.0 END AS targeted_label
		FROM assets a
		LEFT JOIN vulnerabilities v ON v.asset_id = a.id AND v.tenant_id = a.tenant_id
		LEFT JOIN (
			SELECT COALESCE(asset_id, unnest(asset_ids)) AS asset_id, COUNT(*) AS alert_count
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= now() - interval '90 days'
			GROUP BY COALESCE(asset_id, unnest(asset_ids))
		) alerts ON alerts.asset_id = a.id
		LEFT JOIN (
			SELECT COALESCE(asset_id, unnest(asset_ids)) AS asset_id, COUNT(*) AS targeted_count
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= now() - interval '30 days'
			GROUP BY COALESCE(asset_id, unnest(asset_ids))
		) targeted ON targeted.asset_id = a.id
		WHERE a.tenant_id = $1 AND a.deleted_at IS NULL`
	args := []any{tenantID}
	if trimmed := strings.TrimSpace(strings.ToLower(assetType)); trimmed != "" && trimmed != "all" {
		args = append(args, trimmed)
		query += fmt.Sprintf(" AND a.type::text = $%d", len(args))
	}
	query += `
		GROUP BY a.id, a.name, a.type, a.criticality, a.metadata, a.owner, a.updated_at, a.ip_address, alerts.alert_count, targeted.targeted_count
		ORDER BY a.criticality DESC, a.created_at DESC`
	rows, err := s.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("asset risk feature query: %w", err)
	}
	defer rows.Close()
	out := make([]predictmodels.AssetRiskSample, 0, 128)
	for rows.Next() {
		var (
			item            predictmodels.AssetRiskSample
			criticalityText string
		)
		if err := rows.Scan(
			&item.AssetID,
			&item.AssetName,
			&item.AssetType,
			&criticalityText,
			&item.OpenCritical,
			&item.OpenHigh,
			&item.PatchAgeDays,
			&item.InternetFacing,
			&item.HistoricalAlerts,
			&item.UserAccessCount,
			&item.DataSensitivity,
			&item.TargetedLabel,
		); err != nil {
			return nil, err
		}
		item.CriticalityScore = criticalityScore(criticalityText)
		item.IndustrySignal = benchmark.AssetTypePressure[strings.ToLower(item.AssetType)]
		item.TechniqueCoverageGap = math.Max(0, item.IndustrySignal-(item.HistoricalAlerts/10))
		out = append(out, item)
		entityID := item.AssetID.String()
		_ = s.persistVector(ctx, tenantID, "asset_risk", "asset", &entityID, item)
	}
	return out, rows.Err()
}

func (s *FeatureStore) VulnerabilitySamples(ctx context.Context, tenantID uuid.UUID) ([]predictmodels.VulnerabilitySample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	rows, err := s.db.Query(ctx, `
		SELECT
			v.id,
			v.asset_id,
			a.name,
			COALESCE(v.cve_id, ''),
			v.severity::text,
			COALESCE(v.cvss_score, 0)::float8,
			EXTRACT(EPOCH FROM (now() - v.discovered_at)) / 86400.0 AS age_days,
			COALESCE(v.metadata, '{}'::jsonb)
		FROM vulnerabilities v
		JOIN assets a ON a.id = v.asset_id AND a.deleted_at IS NULL
		WHERE v.tenant_id = $1 AND v.deleted_at IS NULL
		ORDER BY v.created_at DESC`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("vulnerability feature query: %w", err)
	}
	defer rows.Close()
	productPrevalence, err := s.assetProductPrevalence(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	out := make([]predictmodels.VulnerabilitySample, 0, 256)
	for rows.Next() {
		var (
			item        predictmodels.VulnerabilitySample
			metadataRaw []byte
			metadata    map[string]any
		)
		if err := rows.Scan(
			&item.VulnerabilityID,
			&item.AssetID,
			&item.AssetName,
			&item.CVEID,
			&item.Severity,
			&item.CVSS,
			&item.AgeDays,
			&metadataRaw,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadataRaw, &metadata)
		if metadata == nil {
			metadata = map[string]any{}
		}
		enrichment := s.cveEnricher.Enrich(metadata, productPrevalence[productKey(metadata, item.CVEID)])
		item.EPSS = enrichment.EPSSScore
		if enrichment.InKnownExploitedList {
			item.KEV = 1
		}
		if enrichment.ProofOfConcept {
			item.ProofOfConcept = 1
		}
		item.SocialMentions = float64(enrichment.SocialMediaMentions)
		item.ProductPrevalence = enrichment.ProductPrevalence
		item.VendorFrequency = vendorFrequency(metadata)
		item.ClassFrequency = vulnClassFrequency(metadata)
		item.ExploitedLabel = exploitedLabel(metadata)
		out = append(out, item)
		entityID := item.VulnerabilityID.String()
		_ = s.persistVector(ctx, tenantID, "vulnerability_exploit", "vulnerability", &entityID, item)
	}
	return out, rows.Err()
}

func (s *FeatureStore) TechniqueTrendSamples(ctx context.Context, tenantID uuid.UUID, lookbackDays int) ([]predictmodels.TechniqueTrendSample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	if lookbackDays <= 0 {
		lookbackDays = 180
	}
	start := s.now().AddDate(0, 0, -lookbackDays)
	rows, err := s.db.Query(ctx, `
		WITH alert_counts AS (
			SELECT date_trunc('day', created_at)::date AS day, COALESCE(mitre_technique_id, 'unknown') AS technique_id, COUNT(*)::float8 AS count
			FROM alerts
			WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= $2
			GROUP BY day, technique_id
		),
		threat_counts AS (
			SELECT date_trunc('day', detected_at)::date AS day, COALESCE(mitre_technique_id, 'unknown') AS technique_id, COUNT(*)::float8 AS count
			FROM threats
			WHERE tenant_id = $1 AND detected_at >= $2
			GROUP BY day, technique_id
		)
		SELECT
			COALESCE(a.technique_id, t.technique_id) AS technique_id,
			COALESCE(a.day, t.day) AS day,
			COALESCE(a.count, 0)::float8 AS internal_count,
			COALESCE(t.count, 0)::float8 AS industry_count
		FROM alert_counts a
		FULL OUTER JOIN threat_counts t ON t.day = a.day AND t.technique_id = a.technique_id
		ORDER BY day ASC`,
		tenantID, start,
	)
	if err != nil {
		return nil, fmt.Errorf("technique trend query: %w", err)
	}
	defer rows.Close()
	out := make([]predictmodels.TechniqueTrendSample, 0, 256)
	for rows.Next() {
		var (
			item      predictmodels.TechniqueTrendSample
			day       time.Time
			technique string
		)
		if err := rows.Scan(&technique, &day, &item.InternalCount, &item.IndustryCount); err != nil {
			return nil, err
		}
		item.TechniqueID = technique
		item.TechniqueName = technique
		item.Timestamp = day
		item.CampaignCorrelation = math.Min(1, (item.InternalCount+item.IndustryCount)/10)
		item.Seasonality = seasonalityValue(day)
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *FeatureStore) InsiderThreatSequences(ctx context.Context, tenantID uuid.UUID, lookbackDays int) (map[string][]predictmodels.InsiderThreatSample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	rows, err := s.db.Query(ctx, `
		SELECT
			a.entity_id,
			COALESCE(a.entity_name, a.entity_id),
			date_trunc('day', a.created_at)::date AS day,
			MAX(a.risk_score_after)::float8 AS risk_score,
			COUNT(*) FILTER (WHERE a.alert_type IN ('possible_credential_compromise', 'unusual_activity'))::float8 AS login_anomalies,
			COUNT(*) FILTER (WHERE a.alert_type = 'possible_data_exfiltration')::float8 AS data_access_trend,
			COUNT(*) FILTER (WHERE a.alert_type IN ('possible_insider_threat', 'possible_privilege_abuse'))::float8 AS after_hours_trend,
			COUNT(*) FILTER (WHERE a.alert_type = 'policy_violation')::float8 AS policy_violations
		FROM ueba_alerts a
		WHERE a.tenant_id = $1
		  AND a.created_at >= now() - make_interval(days => $2)
		GROUP BY a.entity_id, COALESCE(a.entity_name, a.entity_id), day
		ORDER BY a.entity_id, day ASC`,
		tenantID, lookbackDays,
	)
	if err != nil {
		return nil, fmt.Errorf("insider threat feature query: %w", err)
	}
	defer rows.Close()
	out := map[string][]predictmodels.InsiderThreatSample{}
	for rows.Next() {
		var item predictmodels.InsiderThreatSample
		if err := rows.Scan(
			&item.EntityID,
			&item.EntityName,
			&item.Timestamp,
			&item.RiskScore,
			&item.LoginAnomalies,
			&item.DataAccessTrend,
			&item.AfterHoursTrend,
			&item.PolicyViolations,
		); err != nil {
			return nil, err
		}
		item.HREventScore = 0
		item.PeerDeviation = math.Min(1, (item.LoginAnomalies+item.DataAccessTrend+item.PolicyViolations)/10)
		out[item.EntityID] = append(out[item.EntityID], item)
		entityID := item.EntityID
		_ = s.persistVector(ctx, tenantID, "insider_trajectory", "user", &entityID, item)
	}
	return out, rows.Err()
}

func (s *FeatureStore) CampaignSamples(ctx context.Context, tenantID uuid.UUID, lookbackDays int) ([]predictmodels.CampaignAlertSample, error) {
	if s.db == nil {
		return nil, fmt.Errorf("predictive feature store requires a database")
	}
	if lookbackDays <= 0 {
		lookbackDays = 30
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, title, description, created_at, COALESCE(metadata, '{}'::jsonb), COALESCE(mitre_technique_id, ''), COALESCE(asset_ids, '{}'::uuid[])
		FROM alerts
		WHERE tenant_id = $1 AND deleted_at IS NULL AND created_at >= now() - make_interval(days => $2)
		ORDER BY created_at DESC`,
		tenantID, lookbackDays,
	)
	if err != nil {
		return nil, fmt.Errorf("campaign sample query: %w", err)
	}
	defer rows.Close()
	out := make([]predictmodels.CampaignAlertSample, 0, 128)
	for rows.Next() {
		var (
			item        predictmodels.CampaignAlertSample
			metadataRaw []byte
			metadata    map[string]any
			technique   string
			assetIDs    []uuid.UUID
		)
		if err := rows.Scan(
			&item.AlertID,
			&item.Title,
			&item.Description,
			&item.Timestamp,
			&metadataRaw,
			&technique,
			&assetIDs,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(metadataRaw, &metadata)
		item.Embedding = embedText(item.Title + " " + item.Description)
		item.IOCs = extractStringSlice(metadata, "iocs", "domains", "ips", "hashes")
		if strings.TrimSpace(technique) != "" {
			item.Techniques = append(item.Techniques, technique)
		}
		for _, assetID := range assetIDs {
			item.TargetAssets = append(item.TargetAssets, assetID.String())
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *FeatureStore) industryBenchmark(ctx context.Context, tenantID uuid.UUID) (feeds.IndustryBenchmark, error) {
	if s.db == nil || s.benchmarker == nil {
		return feeds.IndustryBenchmark{
			AssetTypePressure:      map[string]float64{},
			TechniquePressure:      map[string]float64{},
			VulnerabilityClassRisk: map[string]float64{},
		}, nil
	}
	rows, err := s.db.Query(ctx, `
		SELECT title, severity::text, COALESCE(mitre_technique_id, ''), COALESCE(metadata, '{}'::jsonb), detected_at
		FROM threats
		WHERE tenant_id = $1
		  AND detected_at >= now() - interval '90 days'`,
		tenantID,
	)
	if err != nil {
		return feeds.IndustryBenchmark{}, fmt.Errorf("industry benchmark query: %w", err)
	}
	defer rows.Close()
	signals := make([]feeds.ThreatFeedSignal, 0, 64)
	for rows.Next() {
		var (
			title       string
			severity    string
			technique   string
			metadataRaw []byte
			metadata    map[string]any
			detectedAt  time.Time
		)
		if err := rows.Scan(&title, &severity, &technique, &metadataRaw, &detectedAt); err != nil {
			return feeds.IndustryBenchmark{}, err
		}
		_ = json.Unmarshal(metadataRaw, &metadata)
		targets := extractStringSlice(metadata, "industries", "targets", "asset_types")
		techniques := []string{}
		if strings.TrimSpace(technique) != "" {
			techniques = append(techniques, technique)
		}
		signals = append(signals, feeds.ThreatFeedSignal{
			Source:       "internal-threats",
			Title:        title,
			Severity:     severity,
			TechniqueIDs: techniques,
			Targets:      targets,
			PublishedAt:  detectedAt,
			Metadata:     metadata,
		})
	}
	return s.benchmarker.Build("tenant", signals), rows.Err()
}

func (s *FeatureStore) assetProductPrevalence(ctx context.Context, tenantID uuid.UUID) (map[string]float64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT lower(COALESCE(metadata->>'vendor', metadata->>'product', 'unknown')), COUNT(*)::float8
		FROM assets
		WHERE tenant_id = $1 AND deleted_at IS NULL
		GROUP BY lower(COALESCE(metadata->>'vendor', metadata->>'product', 'unknown'))`,
		tenantID,
	)
	if err != nil {
		return nil, fmt.Errorf("asset product prevalence query: %w", err)
	}
	defer rows.Close()
	total := 0.0
	counts := map[string]float64{}
	for rows.Next() {
		var key string
		var count float64
		if err := rows.Scan(&key, &count); err != nil {
			return nil, err
		}
		counts[key] = count
		total += count
	}
	if total == 0 {
		total = 1
	}
	for key, count := range counts {
		counts[key] = count / total
	}
	return counts, rows.Err()
}

func (s *FeatureStore) aggregateDailyCounts(ctx context.Context, tenantID uuid.UUID, query string, start time.Time) (map[time.Time]float64, error) {
	rows, err := s.db.Query(ctx, query, tenantID, start)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[time.Time]float64{}
	for rows.Next() {
		var (
			day   time.Time
			count float64
		)
		if err := rows.Scan(&day, &count); err != nil {
			return nil, err
		}
		out[dateOnly(day)] = count
	}
	return out, rows.Err()
}

func (s *FeatureStore) persistVector(ctx context.Context, tenantID uuid.UUID, featureSet, entityType string, entityID *string, vector any) error {
	if s.features == nil {
		return nil
	}
	return s.features.SaveSnapshot(ctx, tenantID, featureSet, entityType, entityID, vector)
}

func criticalityScore(value string) float64 {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical":
		return 1.0
	case "high":
		return 0.75
	case "medium":
		return 0.50
	case "low":
		return 0.25
	default:
		return 0.10
	}
}

func vendorFrequency(metadata map[string]any) float64 {
	key := strings.ToLower(strings.TrimSpace(stringValue(metadata, "vendor_frequency", "vendor_risk")))
	switch key {
	case "critical":
		return 1
	case "high":
		return 0.75
	case "medium":
		return 0.50
	default:
		return floatValue(metadata, "vendor_frequency_score")
	}
}

func vulnClassFrequency(metadata map[string]any) float64 {
	class := strings.ToLower(strings.TrimSpace(stringValue(metadata, "vulnerability_class", "class")))
	switch class {
	case "rce":
		return 1
	case "privilege_escalation":
		return 0.8
	case "sqli":
		return 0.7
	case "xss":
		return 0.5
	default:
		return floatValue(metadata, "class_risk_score")
	}
}

func exploitedLabel(metadata map[string]any) float64 {
	if boolValue(metadata, "exploited_in_wild", "known_exploited", "cisa_kev") {
		return 1
	}
	return 0
}

func productKey(metadata map[string]any, cveID string) string {
	for _, key := range []string{"vendor", "product"} {
		if value := strings.ToLower(strings.TrimSpace(stringValue(metadata, key))); value != "" {
			return value
		}
	}
	return strings.ToLower(strings.TrimSpace(cveID))
}

func stringValue(metadata map[string]any, keys ...string) string {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		if typed, ok := value.(string); ok {
			return typed
		}
	}
	return ""
}

func floatValue(metadata map[string]any, keys ...string) float64 {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case float64:
			return typed
		case int:
			return float64(typed)
		case string:
			if strings.TrimSpace(typed) == "true" {
				return 1
			}
		}
	}
	return 0
}

func boolValue(metadata map[string]any, keys ...string) bool {
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case bool:
			return typed
		case string:
			lower := strings.ToLower(strings.TrimSpace(typed))
			return lower == "true" || lower == "yes" || lower == "1"
		case float64:
			return typed > 0
		}
	}
	return false
}

func extractStringSlice(metadata map[string]any, keys ...string) []string {
	values := map[string]struct{}{}
	appendValue := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		values[value] = struct{}{}
	}
	for _, key := range keys {
		value, ok := metadata[key]
		if !ok || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			for _, part := range strings.Split(typed, ",") {
				appendValue(part)
			}
		case []string:
			for _, item := range typed {
				appendValue(item)
			}
		case []any:
			for _, item := range typed {
				if text, ok := item.(string); ok {
					appendValue(text)
				}
			}
		}
	}
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func seasonalityValue(value time.Time) float64 {
	switch value.Month() {
	case time.October, time.November, time.December:
		return 1.0
	case time.January, time.February:
		return 0.7
	default:
		return 0.5
	}
}

func embedText(text string) []float64 {
	vector := make([]float64, 12)
	tokens := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(tokens) == 0 {
		return vector
	}
	for _, token := range tokens {
		hasher := fnv.New32a()
		_, _ = hasher.Write([]byte(token))
		idx := int(hasher.Sum32()) % len(vector)
		vector[idx]++
	}
	norm := 0.0
	for _, value := range vector {
		norm += value * value
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return vector
	}
	for idx := range vector {
		vector[idx] /= norm
	}
	return vector
}
