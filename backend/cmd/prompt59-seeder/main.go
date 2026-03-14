package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultCyberDBURL = "postgres://clario:clario_dev_pass@localhost:5432/cyber_db?sslmode=disable"
	defaultDataDBURL  = "postgres://clario:clario_dev_pass@localhost:5432/data_db?sslmode=disable"
	defaultTenantID   = "aaaaaaaa-0000-0000-0000-000000000001"
	defaultActorID    = "bbbbbbbb-0000-0000-0000-000000000001"
	seedKey           = "prompt59"
)

var (
	crmSourceID          = mustUUID("10000000-0000-0000-0000-000000000001")
	warehouseSourceID    = mustUUID("10000000-0000-0000-0000-000000000002")
	martSourceID         = mustUUID("10000000-0000-0000-0000-000000000003")
	shadowCopySourceID   = mustUUID("10000000-0000-0000-0000-000000000004")
	upstreamPipelineID   = mustUUID("20000000-0000-0000-0000-000000000001")
	downstreamPipelineID = mustUUID("20000000-0000-0000-0000-000000000002")
	upstreamRunID        = mustUUID("30000000-0000-0000-0000-000000000001")
	downstreamRunID      = mustUUID("30000000-0000-0000-0000-000000000002")
	initialAccessAlertID = mustUUID("40000000-0000-0000-0000-000000000001")
	credentialAlertID    = mustUUID("40000000-0000-0000-0000-000000000002")
	lateralAlertID       = mustUUID("40000000-0000-0000-0000-000000000003")
	alertTimelineIDOne   = mustUUID("41000000-0000-0000-0000-000000000001")
	alertTimelineIDTwo   = mustUUID("41000000-0000-0000-0000-000000000002")
	alertTimelineIDThree = mustUUID("41000000-0000-0000-0000-000000000003")
	uebaProfileID        = mustUUID("42000000-0000-0000-0000-000000000001")
	uebaAlertID          = mustUUID("42000000-0000-0000-0000-000000000002")
	dspmWarehouseID      = mustUUID("43000000-0000-0000-0000-000000000001")
	dspmShadowCopyID     = mustUUID("43000000-0000-0000-0000-000000000002")
	dspmScanID           = mustUUID("43000000-0000-0000-0000-000000000003")
	assetRelOneID        = mustUUID("44000000-0000-0000-0000-000000000001")
	assetRelTwoID        = mustUUID("44000000-0000-0000-0000-000000000002")
	pipelineDependencyID = mustUUID("45000000-0000-0000-0000-000000000001")
	sourceLineageEdgeID  = mustUUID("45000000-0000-0000-0000-000000000002")
)

func main() {
	cyberDBURL := flag.String("cyber-db-url", firstNonEmpty(os.Getenv("CYBER_DB_URL"), defaultCyberDBURL), "Cyber DB connection string")
	dataDBURL := flag.String("data-db-url", firstNonEmpty(os.Getenv("DATA_DB_URL"), defaultDataDBURL), "Data DB connection string")
	tenantIDRaw := flag.String("tenant-id", firstNonEmpty(os.Getenv("PROMPT59_SEED_TENANT_ID"), defaultTenantID), "Tenant UUID to seed")
	actorIDRaw := flag.String("actor-id", firstNonEmpty(os.Getenv("PROMPT59_SEED_ACTOR_ID"), defaultActorID), "Actor/user UUID for created_by fields")
	flag.Parse()

	tenantID := mustParseFlagUUID("tenant-id", *tenantIDRaw)
	actorID := mustParseFlagUUID("actor-id", *actorIDRaw)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	cyberPool := newPool(ctx, *cyberDBURL)
	defer cyberPool.Close()
	dataPool := newPool(ctx, *dataDBURL)
	defer dataPool.Close()

	now := time.Now().UTC().Truncate(time.Second)

	if err := seedCyber(ctx, cyberPool, tenantID, actorID, now); err != nil {
		fmt.Fprintf(os.Stderr, "seed cyber: %v\n", err)
		os.Exit(1)
	}
	if err := seedData(ctx, dataPool, tenantID, actorID, now); err != nil {
		fmt.Fprintf(os.Stderr, "seed data: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Prompt 59 seed applied for tenant %s\n", tenantID)
	fmt.Printf("Security RCA alert ID: %s\n", lateralAlertID)
	fmt.Printf("Pipeline RCA run ID:   %s\n", downstreamRunID)
	fmt.Printf("DSPM shadow sources:   %s, %s\n", warehouseSourceID, shadowCopySourceID)
}

func seedCyber(ctx context.Context, pool *pgxpool.Pool, tenantID, actorID uuid.UUID, now time.Time) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return err
	}

	if err := deleteCyberSeed(ctx, tx); err != nil {
		return err
	}

	crmCreated := now.Add(-4 * time.Hour)
	initialAccessAt := now.Add(-70 * time.Minute)
	credentialAt := now.Add(-38 * time.Minute)
	lateralAt := now.Add(-12 * time.Minute)
	uebaAt := now.Add(-10 * time.Minute)
	scanStartedAt := now.Add(-3 * time.Hour)
	scanCompletedAt := scanStartedAt.Add(5 * time.Minute)
	lastScannedAt := scanCompletedAt

	assets := []struct {
		id          uuid.UUID
		name        string
		assetType   string
		ipAddress   string
		hostname    string
		location    string
		criticality string
		department  string
		owner       *uuid.UUID
		createdAt   time.Time
	}{
		{crmSourceID, "customer-api-gateway", "application", "203.0.113.10", "customer-api-gateway", "DMZ", "critical", "engineering", &actorID, crmCreated},
		{warehouseSourceID, "customer-ledger-warehouse", "database", "10.20.1.10", "customer-ledger-warehouse", "Data Zone", "critical", "data", &actorID, crmCreated},
		{martSourceID, "executive-risk-mart", "database", "10.20.1.20", "executive-risk-mart", "Analytics Zone", "high", "data", &actorID, crmCreated},
		{shadowCopySourceID, "finance-export-shadow", "database", "198.51.100.77", "finance-export-shadow", "External Share", "high", "finance", &actorID, crmCreated},
	}
	for _, asset := range assets {
		meta := mustJSON(map[string]any{
			"seed_key": seedKey,
			"seeded":   true,
		})
		_, err = tx.Exec(ctx, `
			INSERT INTO assets (
				id, tenant_id, name, type, ip_address, hostname, owner, department, location,
				criticality, status, discovered_at, last_seen_at, discovery_source, metadata, tags,
				created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, 'active', $11, $12, 'manual', $13, $14,
				$15, $16, $16
			)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				type = EXCLUDED.type,
				ip_address = EXCLUDED.ip_address,
				hostname = EXCLUDED.hostname,
				owner = EXCLUDED.owner,
				department = EXCLUDED.department,
				location = EXCLUDED.location,
				criticality = EXCLUDED.criticality,
				status = EXCLUDED.status,
				last_seen_at = EXCLUDED.last_seen_at,
				metadata = EXCLUDED.metadata,
				tags = EXCLUDED.tags,
				updated_at = EXCLUDED.updated_at
		`,
			asset.id, tenantID, asset.name, asset.assetType, asset.ipAddress, asset.hostname, asset.owner, asset.department,
			asset.location, asset.criticality, asset.createdAt, now, meta, []string{"prompt59", "seeded"}, actorID, asset.createdAt,
		)
		if err != nil {
			return fmt.Errorf("upsert asset %s: %w", asset.name, err)
		}
	}

	relationships := []struct {
		id          uuid.UUID
		sourceID    uuid.UUID
		targetID    uuid.UUID
		relType     string
		description string
	}{
		{assetRelOneID, crmSourceID, warehouseSourceID, "depends_on", "Public customer API depends on the customer ledger warehouse."},
		{assetRelTwoID, warehouseSourceID, martSourceID, "depends_on", "Risk mart is derived from the customer ledger warehouse."},
	}
	for _, rel := range relationships {
		_, err = tx.Exec(ctx, `
			INSERT INTO asset_relationships (
				id, tenant_id, source_asset_id, target_asset_id, relationship_type, metadata, created_at, created_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (tenant_id, source_asset_id, target_asset_id, relationship_type) DO UPDATE SET
				metadata = EXCLUDED.metadata,
				created_by = EXCLUDED.created_by
		`,
			rel.id, tenantID, rel.sourceID, rel.targetID, rel.relType,
			mustJSON(map[string]any{"seed_key": seedKey, "description": rel.description}), now, actorID,
		)
		if err != nil {
			return fmt.Errorf("insert asset relationship %s: %w", rel.id, err)
		}
	}

	alerts := []struct {
		id            uuid.UUID
		title         string
		description   string
		severity      string
		status        string
		source        string
		assetID       uuid.UUID
		assetIDs      []uuid.UUID
		confidence    float64
		techniqueID   string
		techniqueName string
		tacticID      string
		tacticName    string
		eventCount    int
		createdAt     time.Time
		metadata      map[string]any
	}{
		{
			id:            initialAccessAlertID,
			title:         "Exploit attempt against public customer API",
			description:   "Repeated crafted requests targeted the exposed customer API login and profile routes.",
			severity:      "high",
			status:        "acknowledged",
			source:        "waf",
			assetID:       crmSourceID,
			assetIDs:      []uuid.UUID{crmSourceID},
			confidence:    0.82,
			techniqueID:   "T1190",
			techniqueName: "Exploit Public-Facing Application",
			tacticID:      "TA0001",
			tacticName:    "Initial Access",
			eventCount:    17,
			createdAt:     initialAccessAt,
			metadata: map[string]any{
				"seed_key":  seedKey,
				"seeded":    true,
				"user_id":   actorID.String(),
				"source_ip": "203.0.113.10",
			},
		},
		{
			id:            credentialAlertID,
			title:         "Credential abuse spike from exposed API session",
			description:   "A burst of brute-force and token reuse attempts followed the initial exploit path.",
			severity:      "high",
			status:        "investigating",
			source:        "iam",
			assetID:       crmSourceID,
			assetIDs:      []uuid.UUID{crmSourceID, warehouseSourceID},
			confidence:    0.88,
			techniqueID:   "T1110",
			techniqueName: "Brute Force",
			tacticID:      "TA0006",
			tacticName:    "Credential Access",
			eventCount:    24,
			createdAt:     credentialAt,
			metadata: map[string]any{
				"seed_key":  seedKey,
				"seeded":    true,
				"user_id":   actorID.String(),
				"source_ip": "203.0.113.10",
			},
		},
		{
			id:            lateralAlertID,
			title:         "Lateral movement into customer ledger warehouse",
			description:   "Remote-service activity pivoted from the API tier into the warehouse path containing regulated customer data.",
			severity:      "critical",
			status:        "new",
			source:        "edr",
			assetID:       crmSourceID,
			assetIDs:      []uuid.UUID{crmSourceID, warehouseSourceID},
			confidence:    0.93,
			techniqueID:   "T1021",
			techniqueName: "Remote Services",
			tacticID:      "TA0008",
			tacticName:    "Lateral Movement",
			eventCount:    11,
			createdAt:     lateralAt,
			metadata: map[string]any{
				"seed_key":  seedKey,
				"seeded":    true,
				"user_id":   actorID.String(),
				"source_ip": "203.0.113.10",
			},
		},
	}
	for _, alert := range alerts {
		explanation := mustJSON(map[string]any{
			"summary": fmt.Sprintf("%s was correlated into the seeded Prompt 59 RCA chain.", alert.title),
			"reason":  alert.description,
			"evidence": []map[string]any{
				{"label": "Technique", "field": "mitre_technique_id", "value": alert.techniqueID, "description": alert.techniqueName},
				{"label": "Source IP", "field": "source_ip", "value": "203.0.113.10", "description": "Common attacker IP across the seeded chain."},
			},
			"matched_conditions": []string{"prompt59_seed_chain", strings.ToLower(strings.ReplaceAll(alert.techniqueName, " ", "_"))},
			"confidence_factors": []map[string]any{
				{"factor": "multi_event_correlation", "impact": 0.18, "description": "Multiple correlated events were observed across the same asset and IP."},
				{"factor": "regulated_data_path", "impact": 0.11, "description": "Activity touched a warehouse with restricted regulated data."},
			},
			"recommended_actions": []string{
				"Restrict public API exposure.",
				"Rotate any credentials observed during the incident window.",
				"Review warehouse access and quarantine copied data stores.",
			},
		})
		_, err = tx.Exec(ctx, `
			INSERT INTO alerts (
				id, tenant_id, title, description, severity, status, source, asset_id, asset_ids,
				explanation, confidence_score, mitre_tactic_id, mitre_tactic_name, mitre_technique_id, mitre_technique_name,
				event_count, first_event_at, last_event_at, tags, metadata, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9,
				$10, $11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21, $21
			)
			ON CONFLICT (id) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				severity = EXCLUDED.severity,
				status = EXCLUDED.status,
				source = EXCLUDED.source,
				asset_id = EXCLUDED.asset_id,
				asset_ids = EXCLUDED.asset_ids,
				explanation = EXCLUDED.explanation,
				confidence_score = EXCLUDED.confidence_score,
				mitre_tactic_id = EXCLUDED.mitre_tactic_id,
				mitre_tactic_name = EXCLUDED.mitre_tactic_name,
				mitre_technique_id = EXCLUDED.mitre_technique_id,
				mitre_technique_name = EXCLUDED.mitre_technique_name,
				event_count = EXCLUDED.event_count,
				first_event_at = EXCLUDED.first_event_at,
				last_event_at = EXCLUDED.last_event_at,
				tags = EXCLUDED.tags,
				metadata = EXCLUDED.metadata,
				updated_at = EXCLUDED.updated_at
		`,
			alert.id, tenantID, alert.title, alert.description, alert.severity, alert.status, alert.source, alert.assetID,
			alert.assetIDs, explanation, alert.confidence, alert.tacticID, alert.tacticName, alert.techniqueID, alert.techniqueName,
			alert.eventCount, alert.createdAt, alert.createdAt.Add(2*time.Minute), []string{"prompt59", "seeded", "rca"}, mustJSON(alert.metadata), alert.createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert alert %s: %w", alert.id, err)
		}
	}

	timelineRows := []struct {
		id          uuid.UUID
		createdAt   time.Time
		action      string
		description string
		oldValue    *string
		newValue    *string
	}{
		{alertTimelineIDOne, lateralAt.Add(1 * time.Minute), "triage_started", "Analyst started seeded Prompt 59 triage on the lateral movement alert.", nil, nil},
		{alertTimelineIDTwo, lateralAt.Add(3 * time.Minute), "containment_recommended", "Containment guidance recommended isolating the customer API and rotating exposed credentials.", nil, nil},
		{alertTimelineIDThree, lateralAt.Add(5 * time.Minute), "rca_candidate", "This alert was marked as a root-cause analysis candidate for Prompt 59 verification.", nil, nil},
	}
	for _, item := range timelineRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO alert_timeline (
				id, tenant_id, alert_id, action, actor_id, actor_name, old_value, new_value, description, metadata, created_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (id) DO UPDATE SET
				action = EXCLUDED.action,
				actor_id = EXCLUDED.actor_id,
				actor_name = EXCLUDED.actor_name,
				old_value = EXCLUDED.old_value,
				new_value = EXCLUDED.new_value,
				description = EXCLUDED.description,
				metadata = EXCLUDED.metadata,
				created_at = EXCLUDED.created_at
		`,
			item.id, tenantID, lateralAlertID, item.action, actorID, "Admin Dev", item.oldValue, item.newValue, item.description,
			mustJSON(map[string]any{"seed_key": seedKey, "seeded": true}), item.createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert alert timeline %s: %w", item.id, err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ueba_profiles (
			id, tenant_id, entity_type, entity_id, entity_name, entity_email, baseline, observation_count,
			profile_maturity, first_seen_at, last_seen_at, days_active, risk_score, risk_level, risk_factors,
			risk_last_updated, alert_count_7d, alert_count_30d, last_alert_at, status, created_at, updated_at
		) VALUES (
			$1, $2, 'user', $3, 'Admin Dev', 'admin@clario.dev', $4, 42,
			'mature', $5, $6, 30, 78.50, 'high', $7,
			$8, 2, 3, $9, 'active', $10, $10
		)
		ON CONFLICT (tenant_id, entity_type, entity_id) DO UPDATE SET
			entity_name = EXCLUDED.entity_name,
			entity_email = EXCLUDED.entity_email,
			baseline = EXCLUDED.baseline,
			observation_count = EXCLUDED.observation_count,
			profile_maturity = EXCLUDED.profile_maturity,
			first_seen_at = EXCLUDED.first_seen_at,
			last_seen_at = EXCLUDED.last_seen_at,
			days_active = EXCLUDED.days_active,
			risk_score = EXCLUDED.risk_score,
			risk_level = EXCLUDED.risk_level,
			risk_factors = EXCLUDED.risk_factors,
			risk_last_updated = EXCLUDED.risk_last_updated,
			alert_count_7d = EXCLUDED.alert_count_7d,
			alert_count_30d = EXCLUDED.alert_count_30d,
			last_alert_at = EXCLUDED.last_alert_at,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`,
		uebaProfileID, tenantID, actorID.String(),
		mustJSON(map[string]any{"seed_key": seedKey, "usual_login_regions": []string{"lagos"}, "usual_hours": []int{8, 9, 10, 11, 12}}),
		now.AddDate(0, 0, -30), lateralAt, mustJSON([]map[string]any{{"factor": "seeded_rca_chain", "weight": 0.8}}), lateralAt, lateralAt, now,
	)
	if err != nil {
		return fmt.Errorf("insert ueba profile: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO ueba_alerts (
			id, tenant_id, cyber_alert_id, entity_type, entity_id, entity_name, alert_type, severity, confidence,
			risk_score_before, risk_score_after, risk_score_delta, title, description, triggering_signals,
			triggering_event_ids, baseline_comparison, correlated_signal_count, correlation_window_start,
			correlation_window_end, mitre_technique_ids, mitre_tactic, status, created_at, updated_at
		) VALUES (
			$1, $2, $3, 'user', $4, 'Admin Dev', 'possible_lateral_movement', 'high', 0.91,
			41.00, 78.50, 37.50, $5, $6, $7,
			$8, $9, 3, $10, $11, $12, $13, 'new', $14, $14
		)
		ON CONFLICT (id) DO UPDATE SET
			cyber_alert_id = EXCLUDED.cyber_alert_id,
			entity_type = EXCLUDED.entity_type,
			entity_id = EXCLUDED.entity_id,
			entity_name = EXCLUDED.entity_name,
			alert_type = EXCLUDED.alert_type,
			severity = EXCLUDED.severity,
			confidence = EXCLUDED.confidence,
			risk_score_before = EXCLUDED.risk_score_before,
			risk_score_after = EXCLUDED.risk_score_after,
			risk_score_delta = EXCLUDED.risk_score_delta,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			triggering_signals = EXCLUDED.triggering_signals,
			triggering_event_ids = EXCLUDED.triggering_event_ids,
			baseline_comparison = EXCLUDED.baseline_comparison,
			correlated_signal_count = EXCLUDED.correlated_signal_count,
			correlation_window_start = EXCLUDED.correlation_window_start,
			correlation_window_end = EXCLUDED.correlation_window_end,
			mitre_technique_ids = EXCLUDED.mitre_technique_ids,
			mitre_tactic = EXCLUDED.mitre_tactic,
			status = EXCLUDED.status,
			updated_at = EXCLUDED.updated_at
	`,
		uebaAlertID, tenantID, lateralAlertID, actorID.String(),
		"UEBA detected abnormal follow-on access to the seeded lateral movement path",
		"Behavioral analytics confirmed the seeded user context showed abnormal warehouse access immediately after the API compromise.",
		mustJSON([]map[string]any{
			{"name": "abnormal_ip_reuse", "value": "203.0.113.10"},
			{"name": "warehouse_access", "value": "customer-ledger-warehouse"},
			{"name": "credential_abuse_history", "value": true},
		}),
		[]uuid.UUID{lateralAlertID},
		mustJSON(map[string]any{"seed_key": seedKey, "delta": "warehouse access outside baseline"}),
		credentialAt.Add(-5*time.Minute), lateralAt, []string{"T1021"}, "lateral-movement", uebaAt,
	)
	if err != nil {
		return fmt.Errorf("insert ueba alert: %w", err)
	}

	schemaInfo := mustJSON(map[string]any{
		"tables": []map[string]any{
			{
				"name": "customer_records",
				"columns": []map[string]any{
					{"name": "customer_id", "data_type": "uuid"},
					{"name": "email", "data_type": "text"},
					{"name": "phone", "data_type": "text"},
					{"name": "credit_card", "data_type": "text"},
					{"name": "bvn", "data_type": "text"},
				},
			},
		},
	})
	riskFactors := mustJSON([]map[string]any{
		{"factor": "contains_regulated_pii", "description": "Dataset contains customer PII and payment attributes.", "weight": 0.45, "value": 1},
		{"factor": "internet_exposure", "description": "Asset is reachable from the internet or external shares.", "weight": 0.35, "value": 1},
	})
	postureWarehouse := mustJSON([]map[string]any{
		{"control": "encryption_at_rest", "severity": "medium", "description": "Warehouse is encrypted at rest.", "guidance": "Continue key rotation cadence."},
	})
	postureShadow := mustJSON([]map[string]any{
		{"control": "encryption_at_rest", "severity": "high", "description": "Shadow copy is not encrypted at rest.", "guidance": "Encrypt or decommission the copy immediately."},
		{"control": "access_control", "severity": "high", "description": "No role-based access control is configured.", "guidance": "Move behind tenant-approved RBAC controls."},
	})

	type dspmRow struct {
		id                uuid.UUID
		assetID           uuid.UUID
		name              string
		location          string
		classification    string
		piiTypes          []string
		estimatedRows     int64
		encryptedAtRest   bool
		encryptedTransit  bool
		accessControlType string
		networkExposure   string
		riskScore         float64
		postureScore      float64
		postureFindings   []byte
	}
	dspmRows := []dspmRow{
		{
			id:                dspmWarehouseID,
			assetID:           warehouseSourceID,
			name:              "customer-ledger-warehouse",
			location:          "postgresql://warehouse.internal/customer",
			classification:    "restricted",
			piiTypes:          []string{"email", "phone", "credit_card", "bvn"},
			estimatedRows:     240000,
			encryptedAtRest:   true,
			encryptedTransit:  true,
			accessControlType: "rbac",
			networkExposure:   "internal_only",
			riskScore:         72,
			postureScore:      84,
			postureFindings:   postureWarehouse,
		},
		{
			id:                dspmShadowCopyID,
			assetID:           shadowCopySourceID,
			name:              "finance-export-shadow",
			location:          "s3://finance-shadow/customer_exports",
			classification:    "restricted",
			piiTypes:          []string{"email", "phone", "credit_card", "bvn"},
			estimatedRows:     240000,
			encryptedAtRest:   false,
			encryptedTransit:  false,
			accessControlType: "none",
			networkExposure:   "internet_facing",
			riskScore:         97,
			postureScore:      21,
			postureFindings:   postureShadow,
		},
	}
	for _, row := range dspmRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO dspm_data_assets (
				id, tenant_id, name, type, location, classification, asset_id, scan_id, data_classification,
				sensitivity_score, contains_pii, pii_types, pii_column_count, estimated_record_count,
				encrypted_at_rest, encrypted_in_transit, access_control_type, network_exposure,
				backup_configured, audit_logging, risk_score, risk_factors, posture_score, posture_findings,
				consumer_count, producer_count, database_type, schema_info, metadata, last_scanned_at, created_at, updated_at
			) VALUES (
				$1, $2, $3, 'database', $4, $5::data_classification, $6, $7, $5,
				88, true, $8, $9, $10,
				$11, $12, $13, $14,
				true, true, $15, $16, $17, $18,
				2, 1, 'postgresql', $19, $20, $21, $21, $21
			)
			ON CONFLICT (tenant_id, asset_id) DO UPDATE SET
				name = EXCLUDED.name,
				type = EXCLUDED.type,
				location = EXCLUDED.location,
				classification = EXCLUDED.classification,
				scan_id = EXCLUDED.scan_id,
				data_classification = EXCLUDED.data_classification,
				sensitivity_score = EXCLUDED.sensitivity_score,
				contains_pii = EXCLUDED.contains_pii,
				pii_types = EXCLUDED.pii_types,
				pii_column_count = EXCLUDED.pii_column_count,
				estimated_record_count = EXCLUDED.estimated_record_count,
				encrypted_at_rest = EXCLUDED.encrypted_at_rest,
				encrypted_in_transit = EXCLUDED.encrypted_in_transit,
				access_control_type = EXCLUDED.access_control_type,
				network_exposure = EXCLUDED.network_exposure,
				backup_configured = EXCLUDED.backup_configured,
				audit_logging = EXCLUDED.audit_logging,
				risk_score = EXCLUDED.risk_score,
				risk_factors = EXCLUDED.risk_factors,
				posture_score = EXCLUDED.posture_score,
				posture_findings = EXCLUDED.posture_findings,
				consumer_count = EXCLUDED.consumer_count,
				producer_count = EXCLUDED.producer_count,
				database_type = EXCLUDED.database_type,
				schema_info = EXCLUDED.schema_info,
				metadata = EXCLUDED.metadata,
				last_scanned_at = EXCLUDED.last_scanned_at,
				updated_at = EXCLUDED.updated_at
		`,
			row.id, tenantID, row.name, row.location, row.classification, row.assetID, dspmScanID,
			row.piiTypes, len(row.piiTypes), row.estimatedRows, row.encryptedAtRest, row.encryptedTransit,
			row.accessControlType, row.networkExposure, row.riskScore, riskFactors, row.postureScore,
			row.postureFindings, schemaInfo, mustJSON(map[string]any{"seed_key": seedKey, "seeded": true, "location": row.location}), lastScannedAt,
		)
		if err != nil {
			return fmt.Errorf("insert dspm asset %s: %w", row.assetID, err)
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO dspm_scans (
			id, tenant_id, status, assets_scanned, pii_assets_found, high_risk_found, findings_count,
			started_at, completed_at, duration_ms, created_by, created_at
		) VALUES ($1, $2, 'completed', 2, 2, 1, 3, $3, $4, $5, $6, $3)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			assets_scanned = EXCLUDED.assets_scanned,
			pii_assets_found = EXCLUDED.pii_assets_found,
			high_risk_found = EXCLUDED.high_risk_found,
			findings_count = EXCLUDED.findings_count,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			duration_ms = EXCLUDED.duration_ms,
			created_by = EXCLUDED.created_by,
			created_at = EXCLUDED.created_at
	`, dspmScanID, tenantID, scanStartedAt, scanCompletedAt, scanCompletedAt.Sub(scanStartedAt).Milliseconds(), actorID)
	if err != nil {
		return fmt.Errorf("insert dspm scan: %w", err)
	}

	return tx.Commit(ctx)
}

func seedData(ctx context.Context, pool *pgxpool.Pool, tenantID, actorID uuid.UUID, now time.Time) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return err
	}
	if err := deleteDataSeed(ctx, tx); err != nil {
		return err
	}

	schemaMetadata := mustJSON(map[string]any{
		"tables": []map[string]any{
			{
				"name": "customer_records",
				"columns": []map[string]any{
					{"name": "customer_id", "data_type": "uuid"},
					{"name": "email", "data_type": "text"},
					{"name": "phone", "data_type": "text"},
					{"name": "credit_card", "data_type": "text"},
					{"name": "bvn", "data_type": "text"},
				},
			},
		},
	})

	connectionConfigs := map[uuid.UUID]map[string]any{
		crmSourceID: {
			"base_url":        "http://localhost:8080/api/v1",
			"health_url":      "http://localhost:8080/health",
			"data_path":       "/customers",
			"auth_type":       "bearer",
			"auth_config":     map[string]any{"token": "seed-demo-token"},
			"timeout_seconds": 30,
			"rate_limit_rps":  10,
			"pagination_type": "offset",
		},
		warehouseSourceID: {
			"host":     "localhost",
			"port":     5432,
			"database": "data_db",
			"schema":   "public",
			"username": "clario",
			"password": "clario_dev_pass",
			"ssl_mode": "disable",
		},
		martSourceID: {
			"host":     "localhost",
			"port":     5432,
			"database": "data_db",
			"schema":   "public",
			"username": "clario",
			"password": "clario_dev_pass",
			"ssl_mode": "disable",
		},
		shadowCopySourceID: {
			"endpoint":   "localhost:9000",
			"bucket":     "clario360-shadow",
			"prefix":     "finance-export/",
			"access_key": "clario_minio",
			"secret_key": "clario_minio_secret",
			"use_ssl":    false,
		},
	}

	dataSources := []struct {
		id          uuid.UUID
		name        string
		description string
		sourceType  string
		status      string
		tables      int
		rows        int64
		sizeBytes   int64
		createdAt   time.Time
	}{
		{crmSourceID, "Customer API Source", "Public API ingestion source used by the seeded pipeline RCA scenario.", "api", "active", 1, 120000, 94371840, now.Add(-4 * time.Hour)},
		{warehouseSourceID, "Customer Ledger Warehouse", "Primary regulated warehouse used in Prompt 59 RCA and DSPM scenarios.", "postgresql", "active", 1, 240000, 157286400, now.Add(-4 * time.Hour)},
		{martSourceID, "Executive Risk Mart", "Downstream mart fed by the warehouse for executive reporting.", "postgresql", "active", 1, 98000, 73400320, now.Add(-4 * time.Hour)},
		{shadowCopySourceID, "Finance Export Shadow", "Unauthorized externalized shadow copy used to exercise shadow-copy detection.", "s3", "error", 1, 240000, 157286400, now.Add(-3 * time.Hour)},
	}
	for _, source := range dataSources {
		configBytes := mustJSON(connectionConfigs[source.id])
		_, err = tx.Exec(ctx, `
			INSERT INTO data_sources (
				id, tenant_id, name, description, type, connection_config, encryption_key_id, status, last_error,
				schema_metadata, schema_discovered_at, last_synced_at, last_sync_status, last_sync_error, last_sync_duration_ms,
				table_count, total_row_count, total_size_bytes, tags, metadata, created_by, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, 'seed-local', $7, $8,
				$9, $10, $11, 'completed', NULL, 1900,
				$12, $13, $14, $15, $16, $17, $18, $18
			)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				description = EXCLUDED.description,
				type = EXCLUDED.type,
				connection_config = EXCLUDED.connection_config,
				encryption_key_id = EXCLUDED.encryption_key_id,
				status = EXCLUDED.status,
				last_error = EXCLUDED.last_error,
				schema_metadata = EXCLUDED.schema_metadata,
				schema_discovered_at = EXCLUDED.schema_discovered_at,
				last_synced_at = EXCLUDED.last_synced_at,
				last_sync_status = EXCLUDED.last_sync_status,
				last_sync_error = EXCLUDED.last_sync_error,
				last_sync_duration_ms = EXCLUDED.last_sync_duration_ms,
				table_count = EXCLUDED.table_count,
				total_row_count = EXCLUDED.total_row_count,
				total_size_bytes = EXCLUDED.total_size_bytes,
				tags = EXCLUDED.tags,
				metadata = EXCLUDED.metadata,
				updated_at = EXCLUDED.updated_at
		`,
			source.id, tenantID, source.name, source.description, source.sourceType, configBytes, source.status,
			nullIfEmpty("Shadow copy is unmanaged and unapproved.", source.id == shadowCopySourceID),
			schemaMetadata, source.createdAt, now.Add(-25*time.Minute), source.tables, source.rows, source.sizeBytes,
			[]string{"prompt59", "seeded"}, mustJSON(map[string]any{"seed_key": seedKey, "seeded": true}), actorID, source.createdAt,
		)
		if err != nil {
			return fmt.Errorf("insert data source %s: %w", source.id, err)
		}
	}

	upstreamConfig := mustJSON(map[string]any{
		"source_table":         "customer_api_events",
		"target_table":         "customer_records",
		"load_strategy":        "incremental",
		"incremental_field":    "updated_at",
		"fail_on_quality_gate": false,
		"retry_backoff_sec":    30,
		"metadata":             map[string]any{"seed_key": seedKey},
	})
	downstreamConfig := mustJSON(map[string]any{
		"source_table":         "customer_records",
		"target_table":         "risk_customer_snapshot",
		"load_strategy":        "full_replace",
		"fail_on_quality_gate": true,
		"merge_keys":           []string{"customer_id"},
		"metadata":             map[string]any{"seed_key": seedKey},
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO pipelines (
			id, tenant_id, name, description, type, source_id, target_id, schedule, config, status,
			last_run_id, last_run_at, last_run_status, last_run_error, total_runs, successful_runs, failed_runs,
			total_records_processed, avg_duration_ms, tags, created_by, created_at, updated_at
		) VALUES
			($1, $2, $3, $4, 'etl', $5, $6, '*/30 * * * *', $7, 'error', $8, $9, 'failed', $10, 1, 0, 1, 0, $11, $12, $13, $14, $14),
			($15, $2, $16, $17, 'elt', $18, $19, '*/30 * * * *', $20, 'error', $21, $22, 'failed', $23, 1, 0, 1, 0, $24, $25, $13, $26, $26)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			source_id = EXCLUDED.source_id,
			target_id = EXCLUDED.target_id,
			schedule = EXCLUDED.schedule,
			config = EXCLUDED.config,
			status = EXCLUDED.status,
			last_run_id = EXCLUDED.last_run_id,
			last_run_at = EXCLUDED.last_run_at,
			last_run_status = EXCLUDED.last_run_status,
			last_run_error = EXCLUDED.last_run_error,
			total_runs = EXCLUDED.total_runs,
			successful_runs = EXCLUDED.successful_runs,
			failed_runs = EXCLUDED.failed_runs,
			total_records_processed = EXCLUDED.total_records_processed,
			avg_duration_ms = EXCLUDED.avg_duration_ms,
			tags = EXCLUDED.tags,
			updated_at = EXCLUDED.updated_at
	`,
		upstreamPipelineID, tenantID,
		"CRM API to Customer Warehouse", "Ingests customer API events into the regulated warehouse.",
		crmSourceID, warehouseSourceID, upstreamConfig,
		upstreamRunID, now.Add(-50*time.Minute), "source connection timeout: customer-api.internal unreachable", int64(240000),
		[]string{"prompt59", "seeded"}, actorID, now.Add(-4*time.Hour),
		downstreamPipelineID,
		"Warehouse to Executive Risk Mart", "Builds the executive risk mart from the customer warehouse feed.",
		warehouseSourceID, martSourceID, downstreamConfig,
		downstreamRunID, now.Add(-20*time.Minute), "upstream dataset unavailable after dependency failure", int64(180000),
		[]string{"prompt59", "seeded"}, now.Add(-4*time.Hour),
	)
	if err != nil {
		return fmt.Errorf("insert pipelines: %w", err)
	}

	upstreamDuration := int64((5 * time.Minute).Milliseconds())
	downstreamDuration := int64((4 * time.Minute).Milliseconds())
	upstreamErrorDetails := mustJSON(map[string]any{
		"seed_key":   seedKey,
		"category":   "network",
		"source_url": "https://customer-api.internal/v1/customers",
	})
	downstreamErrorDetails := mustJSON(map[string]any{
		"seed_key":          seedKey,
		"category":          "dependency",
		"upstream_pipeline": upstreamPipelineID.String(),
	})
	_, err = tx.Exec(ctx, `
		INSERT INTO pipeline_runs (
			id, tenant_id, pipeline_id, status, current_phase, records_processed, records_failed, metrics,
			created_at, started_at, completed_at, records_extracted, records_transformed, records_loaded,
			records_filtered, records_deduplicated, bytes_read, bytes_written, quality_gate_results,
			quality_gates_passed, quality_gates_failed, quality_gates_warned, extract_started_at, extract_completed_at,
			transform_started_at, transform_completed_at, load_started_at, load_completed_at, duration_ms,
			error_phase, error_message, error_details, triggered_by, triggered_by_user, retry_count
		) VALUES
			($1, $2, $3, 'failed', 'extracting', 0, 1, $4,
			 $5, $6, $7, 0, 0, 0,
			 0, 0, 0, 0, '[]'::jsonb,
			 0, 0, 0, $6, NULL,
			 NULL, NULL, NULL, NULL, $8,
			 'extract', $9, $10, 'schedule', $11, 1),
			($12, $2, $13, 'failed', 'loading', 0, 1, $14,
			 $15, $16, $17, 240000, 240000, 0,
			 0, 0, 157286400, 0, '[]'::jsonb,
			 1, 0, 0, $16, $18,
			 $18, $19, $19, NULL, $20,
			 'load', $21, $22, 'event', $11, 0)
		ON CONFLICT (id) DO UPDATE SET
			pipeline_id = EXCLUDED.pipeline_id,
			status = EXCLUDED.status,
			current_phase = EXCLUDED.current_phase,
			records_processed = EXCLUDED.records_processed,
			records_failed = EXCLUDED.records_failed,
			metrics = EXCLUDED.metrics,
			created_at = EXCLUDED.created_at,
			started_at = EXCLUDED.started_at,
			completed_at = EXCLUDED.completed_at,
			records_extracted = EXCLUDED.records_extracted,
			records_transformed = EXCLUDED.records_transformed,
			records_loaded = EXCLUDED.records_loaded,
			records_filtered = EXCLUDED.records_filtered,
			records_deduplicated = EXCLUDED.records_deduplicated,
			bytes_read = EXCLUDED.bytes_read,
			bytes_written = EXCLUDED.bytes_written,
			quality_gate_results = EXCLUDED.quality_gate_results,
			quality_gates_passed = EXCLUDED.quality_gates_passed,
			quality_gates_failed = EXCLUDED.quality_gates_failed,
			quality_gates_warned = EXCLUDED.quality_gates_warned,
			extract_started_at = EXCLUDED.extract_started_at,
			extract_completed_at = EXCLUDED.extract_completed_at,
			transform_started_at = EXCLUDED.transform_started_at,
			transform_completed_at = EXCLUDED.transform_completed_at,
			load_started_at = EXCLUDED.load_started_at,
			load_completed_at = EXCLUDED.load_completed_at,
			duration_ms = EXCLUDED.duration_ms,
			error_phase = EXCLUDED.error_phase,
			error_message = EXCLUDED.error_message,
			error_details = EXCLUDED.error_details,
			triggered_by = EXCLUDED.triggered_by,
			triggered_by_user = EXCLUDED.triggered_by_user,
			retry_count = EXCLUDED.retry_count
	`,
		upstreamRunID, tenantID, upstreamPipelineID,
		mustJSON(map[string]any{"seed_key": seedKey, "phase": "extracting"}),
		now.Add(-55*time.Minute), now.Add(-55*time.Minute), now.Add(-50*time.Minute), upstreamDuration,
		"source connection timeout: customer-api.internal unreachable", upstreamErrorDetails, actorID,
		downstreamRunID, downstreamPipelineID,
		mustJSON(map[string]any{"seed_key": seedKey, "phase": "loading"}),
		now.Add(-24*time.Minute), now.Add(-24*time.Minute), now.Add(-20*time.Minute),
		now.Add(-22*time.Minute), now.Add(-21*time.Minute), downstreamDuration,
		"upstream dataset unavailable after dependency failure", downstreamErrorDetails,
	)
	if err != nil {
		return fmt.Errorf("insert pipeline runs: %w", err)
	}

	lineageRows := []struct {
		id             uuid.UUID
		sourceType     string
		sourceID       uuid.UUID
		sourceName     string
		targetType     string
		targetID       uuid.UUID
		targetName     string
		relationship   string
		transformation string
		pipelineID     *uuid.UUID
		runID          *uuid.UUID
	}{
		{
			id:             sourceLineageEdgeID,
			sourceType:     "data_source",
			sourceID:       warehouseSourceID,
			sourceName:     "Customer Ledger Warehouse",
			targetType:     "data_source",
			targetID:       martSourceID,
			targetName:     "Executive Risk Mart",
			relationship:   "feeds",
			transformation: "warehouse feed into executive mart",
			pipelineID:     &downstreamPipelineID,
			runID:          &downstreamRunID,
		},
		{
			id:             pipelineDependencyID,
			sourceType:     "pipeline",
			sourceID:       upstreamPipelineID,
			sourceName:     "CRM API to Customer Warehouse",
			targetType:     "pipeline",
			targetID:       downstreamPipelineID,
			targetName:     "Warehouse to Executive Risk Mart",
			relationship:   "depends_on",
			transformation: "downstream mart depends on upstream warehouse refresh",
			pipelineID:     &downstreamPipelineID,
			runID:          &downstreamRunID,
		},
	}
	for _, edge := range lineageRows {
		_, err = tx.Exec(ctx, `
			INSERT INTO data_lineage_edges (
				id, tenant_id, source_type, source_id, source_name, target_type, target_id, target_name,
				relationship, transformation_desc, transformation_type, columns_affected, pipeline_id, pipeline_run_id,
				recorded_by, active, first_seen_at, last_seen_at, metadata, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8,
				$9, $10, 'etl', $11, $12, $13,
				'event', true, $14, $14, $15, $14, $14
			)
			ON CONFLICT (tenant_id, source_type, source_id, target_type, target_id, relationship) DO UPDATE SET
				source_name = EXCLUDED.source_name,
				target_name = EXCLUDED.target_name,
				transformation_desc = EXCLUDED.transformation_desc,
				transformation_type = EXCLUDED.transformation_type,
				columns_affected = EXCLUDED.columns_affected,
				pipeline_id = EXCLUDED.pipeline_id,
				pipeline_run_id = EXCLUDED.pipeline_run_id,
				recorded_by = EXCLUDED.recorded_by,
				active = EXCLUDED.active,
				last_seen_at = EXCLUDED.last_seen_at,
				metadata = EXCLUDED.metadata,
				updated_at = EXCLUDED.updated_at
		`,
			edge.id, tenantID, edge.sourceType, edge.sourceID, edge.sourceName, edge.targetType, edge.targetID, edge.targetName,
			edge.relationship, edge.transformation, []string{"customer_id", "email", "phone"}, edge.pipelineID, edge.runID,
			now.Add(-20*time.Minute), mustJSON(map[string]any{"seed_key": seedKey, "seeded": true}),
		)
		if err != nil {
			return fmt.Errorf("insert lineage edge %s: %w", edge.id, err)
		}
	}

	return tx.Commit(ctx)
}

func deleteCyberSeed(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM ueba_alerts WHERE id = ANY($1)`, []uuid.UUID{uebaAlertID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM ueba_profiles WHERE id = ANY($1)`, []uuid.UUID{uebaProfileID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM alert_timeline WHERE id = ANY($1)`, []uuid.UUID{alertTimelineIDOne, alertTimelineIDTwo, alertTimelineIDThree}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM alerts WHERE id = ANY($1)`, []uuid.UUID{initialAccessAlertID, credentialAlertID, lateralAlertID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM dspm_scans WHERE id = $1`, dspmScanID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM dspm_data_assets WHERE id = ANY($1)`, []uuid.UUID{dspmWarehouseID, dspmShadowCopyID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM asset_relationships WHERE id = ANY($1)`, []uuid.UUID{assetRelOneID, assetRelTwoID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM assets WHERE id = ANY($1)`, []uuid.UUID{crmSourceID, warehouseSourceID, martSourceID, shadowCopySourceID}); err != nil {
		return err
	}
	return nil
}

func deleteDataSeed(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `DELETE FROM data_lineage_edges WHERE id = ANY($1)`, []uuid.UUID{pipelineDependencyID, sourceLineageEdgeID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pipeline_run_logs WHERE run_id = ANY($1)`, []uuid.UUID{upstreamRunID, downstreamRunID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pipeline_runs WHERE id = ANY($1)`, []uuid.UUID{upstreamRunID, downstreamRunID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM pipelines WHERE id = ANY($1)`, []uuid.UUID{upstreamPipelineID, downstreamPipelineID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM data_sources WHERE id = ANY($1)`, []uuid.UUID{crmSourceID, warehouseSourceID, martSourceID, shadowCopySourceID}); err != nil {
		return err
	}
	return nil
}

func setTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, tenantID.String())
	return err
}

func newPool(ctx context.Context, dbURL string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create pool: %v\n", err)
		os.Exit(1)
	}
	if err := pool.Ping(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "ping pool: %v\n", err)
		os.Exit(1)
	}
	return pool
}

func mustParseFlagUUID(flagName, raw string) uuid.UUID {
	id, err := uuid.Parse(strings.TrimSpace(raw))
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s: %v\n", flagName, err)
		os.Exit(1)
	}
	return id
}

func mustUUID(raw string) uuid.UUID {
	id, err := uuid.Parse(raw)
	if err != nil {
		panic(err)
	}
	return id
}

func mustJSON(v any) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func nullIfEmpty(value string, enabled bool) *string {
	if !enabled || strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
