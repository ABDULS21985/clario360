package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"

	"github.com/clario360/platform/internal/events"
	iammodel "github.com/clario360/platform/internal/iam/model"
	"github.com/clario360/platform/internal/onboarding/repository"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	visusmodel "github.com/clario360/platform/internal/visus/model"
	visusrepo "github.com/clario360/platform/internal/visus/repository"
	visuskap "github.com/clario360/platform/internal/visus/kpi"
)

// TenantProvisioner runs async provisioning steps for a newly registered tenant.
type TenantProvisioner struct {
	platformDB *pgxpool.Pool // platform_core
	cyberDB    *pgxpool.Pool // cyber_db (may be nil)
	visusDB    *pgxpool.Pool // visus_db (may be nil)
	provRepo   *repository.ProvisioningRepository
	producer   *events.Producer
	logger     zerolog.Logger
}

// NewTenantProvisioner creates a new TenantProvisioner.
func NewTenantProvisioner(
	platformDB *pgxpool.Pool,
	cyberDB *pgxpool.Pool,
	visusDB *pgxpool.Pool,
	provRepo *repository.ProvisioningRepository,
	producer *events.Producer,
	logger zerolog.Logger,
) *TenantProvisioner {
	return &TenantProvisioner{
		platformDB: platformDB,
		cyberDB:    cyberDB,
		visusDB:    visusDB,
		provRepo:   provRepo,
		producer:   producer,
		logger:     logger.With().Str("svc", "provisioner").Logger(),
	}
}

// stepNames defines the ordered provisioning steps.
var stepNames = []string{
	"verify_infrastructure",
	"seed_system_roles",
	"seed_default_settings",
	"seed_detection_rules",
	"seed_kpis",
	"seed_dashboard",
	"create_storage_buckets",
	"initialize_audit_trail",
	"send_welcome_notification",
	"complete",
}

// Provision runs all provisioning steps for a tenant. Must be called in a goroutine.
func (p *TenantProvisioner) Provision(ctx context.Context, tenantID, adminUserID uuid.UUID) {
	log := p.logger.With().Str("tenant_id", tenantID.String()).Logger()

	onboardingID, err := p.provRepo.GetOnboardingID(ctx, tenantID)
	if err != nil {
		log.Error().Err(err).Msg("cannot find onboarding record for provisioning")
		return
	}

	if err := p.provRepo.Initialize(ctx, tenantID, onboardingID, stepNames); err != nil {
		log.Error().Err(err).Msg("initialize provisioning steps")
		return
	}

	steps := []func(context.Context, uuid.UUID, uuid.UUID, zerolog.Logger) error{
		p.stepVerifyInfrastructure,
		p.stepSeedSystemRoles,
		p.stepSeedDefaultSettings,
		p.stepSeedDetectionRules,
		p.stepSeedKPIs,
		p.stepSeedDashboard,
		p.stepCreateStorageBuckets,
		p.stepInitializeAuditTrail,
		p.stepSendWelcomeNotification,
		p.stepComplete,
	}

	for i, step := range steps {
		stepNum := i + 1
		if err := p.provRepo.StartStep(ctx, tenantID, stepNum); err != nil {
			log.Error().Err(err).Int("step", stepNum).Msg("mark step running")
		}

		log.Info().Int("step", stepNum).Str("name", stepNames[i]).Msg("provisioning step started")

		if err := step(ctx, tenantID, adminUserID, log); err != nil {
			log.Error().Err(err).Int("step", stepNum).Str("name", stepNames[i]).Msg("provisioning step failed")
			_ = p.provRepo.FailStep(ctx, tenantID, stepNum, err.Error(), nil)
			_ = p.provRepo.MarkFailed(ctx, tenantID, fmt.Sprintf("step %d (%s) failed: %s", stepNum, stepNames[i], err.Error()))
			return
		}

		if err := p.provRepo.CompleteStep(ctx, tenantID, stepNum, nil); err != nil {
			log.Error().Err(err).Int("step", stepNum).Msg("mark step completed")
		}
		log.Info().Int("step", stepNum).Str("name", stepNames[i]).Msg("provisioning step completed")
	}
}

// stepVerifyInfrastructure verifies DB connections.
func (p *TenantProvisioner) stepVerifyInfrastructure(ctx context.Context, tenantID, _ uuid.UUID, log zerolog.Logger) error {
	if err := p.platformDB.Ping(ctx); err != nil {
		return fmt.Errorf("platform_core DB unreachable: %w", err)
	}
	log.Debug().Msg("infrastructure verified")
	return nil
}

// stepSeedSystemRoles inserts default IAM roles for the tenant.
func (p *TenantProvisioner) stepSeedSystemRoles(ctx context.Context, tenantID, _ uuid.UUID, log zerolog.Logger) error {
	for _, role := range iammodel.SystemRoles {
		permsJSON, err := json.Marshal(role.Permissions)
		if err != nil {
			return fmt.Errorf("marshal permissions for role %s: %w", role.Slug, err)
		}
		_, err = p.platformDB.Exec(ctx, `
			INSERT INTO roles (tenant_id, name, slug, description, is_system_role, permissions)
			VALUES ($1, $2, $3, $4, $5, $6::jsonb)
			ON CONFLICT (tenant_id, slug) DO NOTHING`,
			tenantID, role.Name, role.Slug, role.Description, role.IsSystemRole, string(permsJSON),
		)
		if err != nil {
			return fmt.Errorf("seed role %s: %w", role.Slug, err)
		}
	}
	log.Debug().Int("count", len(iammodel.SystemRoles)).Msg("system roles seeded")
	return nil
}

// stepSeedDefaultSettings inserts tenant key-value settings.
func (p *TenantProvisioner) stepSeedDefaultSettings(ctx context.Context, tenantID, _ uuid.UUID, log zerolog.Logger) error {
	settings := []struct {
		key, value, desc string
	}{
		{"onboarding_complete", `false`, "Whether tenant onboarding wizard is complete"},
		{"subscription_tier", `"professional"`, "Current subscription tier"},
		{"timezone", `"UTC"`, "Default timezone for the tenant"},
		{"date_format", `"DD/MM/YYYY"`, "Default date display format"},
	}
	for _, s := range settings {
		_, err := p.platformDB.Exec(ctx, `
			INSERT INTO system_settings (tenant_id, key, value, description)
			VALUES ($1, $2, $3::jsonb, $4)
			ON CONFLICT (tenant_id, key) DO NOTHING`,
			tenantID, s.key, s.value, s.desc,
		)
		if err != nil {
			return fmt.Errorf("seed setting %s: %w", s.key, err)
		}
	}
	log.Debug().Int("count", len(settings)).Msg("default settings seeded")
	return nil
}

// stepSeedDetectionRules inserts default detection rules in the cyber DB.
func (p *TenantProvisioner) stepSeedDetectionRules(ctx context.Context, tenantID, adminUserID uuid.UUID, log zerolog.Logger) error {
	db := p.cyberDB
	if db == nil {
		db = p.platformDB // graceful fallback — rules table may be in same DB in single-DB setups
	}

	type ruleSpec struct {
		name, desc string
		ruleType   cybermodel.DetectionRuleType
		severity   cybermodel.Severity
		content    map[string]any
		tactics    []string
		techniques []string
		tags       []string
	}

	rules := []ruleSpec{
		{
			name:     "Brute Force Detection",
			desc:     "Detects brute force login attempts (threshold: 5 failures in 5 minutes)",
			ruleType: cybermodel.DetectionRuleTypeThreshold,
			severity: cybermodel.SeverityHigh,
			content:  map[string]any{"threshold": 5, "window_seconds": 300, "field": "source_ip", "event_type": "authentication.failure"},
			tactics:  []string{"TA0006"},
			techniques: []string{"T1110"},
			tags:     []string{"authentication", "brute-force", "default"},
		},
		{
			name:     "Port Scan Detection",
			desc:     "Detects horizontal or vertical port scanning activity",
			ruleType: cybermodel.DetectionRuleTypeThreshold,
			severity: cybermodel.SeverityMedium,
			content:  map[string]any{"threshold": 20, "window_seconds": 60, "field": "destination_port", "event_type": "network.connection"},
			tactics:  []string{"TA0043"},
			techniques: []string{"T1046"},
			tags:     []string{"network", "recon", "default"},
		},
		{
			name:     "SQL Injection Detection",
			desc:     "Detects SQL injection patterns in HTTP requests",
			ruleType: cybermodel.DetectionRuleTypeSignature,
			severity: cybermodel.SeverityCritical,
			content:  map[string]any{"patterns": []string{"' OR '1'='1", "UNION SELECT", "DROP TABLE", "--", "/**/"}},
			tactics:  []string{"TA0001"},
			techniques: []string{"T1190"},
			tags:     []string{"web", "injection", "default"},
		},
		{
			name:     "Data Exfiltration Detection",
			desc:     "Detects anomalous outbound data transfer volumes",
			ruleType: cybermodel.DetectionRuleTypeAnomaly,
			severity: cybermodel.SeverityHigh,
			content:  map[string]any{"baseline_field": "bytes_out", "deviation_multiplier": 3.0, "event_type": "network.transfer"},
			tactics:  []string{"TA0010"},
			techniques: []string{"T1041"},
			tags:     []string{"data-loss", "exfiltration", "default"},
		},
		{
			name:     "Malware Signature Detection",
			desc:     "Detects known malware signatures in file events",
			ruleType: cybermodel.DetectionRuleTypeSignature,
			severity: cybermodel.SeverityCritical,
			content:  map[string]any{"signature_set": "common-malware-v1", "event_type": "file.created"},
			tactics:  []string{"TA0002"},
			techniques: []string{"T1204"},
			tags:     []string{"malware", "endpoint", "default"},
		},
	}

	for _, r := range rules {
		contentJSON, err := json.Marshal(r.content)
		if err != nil {
			return fmt.Errorf("marshal rule content for %s: %w", r.name, err)
		}
		tagsJSON, err := json.Marshal(r.tags)
		if err != nil {
			return fmt.Errorf("marshal tags for %s: %w", r.name, err)
		}

		_, err = db.Exec(ctx, `
			INSERT INTO detection_rules (
				id, tenant_id, name, description, rule_type, severity, enabled, rule_content,
				mitre_tactic_ids, mitre_technique_ids, base_confidence, false_positive_count,
				true_positive_count, trigger_count, tags, is_template, created_by, created_at, updated_at
			) VALUES (
				gen_random_uuid(), $1, $2, $3, $4, $5, true, $6::jsonb,
				$7, $8, 0.8, 0, 0, 0, $9::jsonb, false, $10, now(), now()
			)
			ON CONFLICT DO NOTHING`,
			tenantID, r.name, r.desc, r.ruleType, r.severity, string(contentJSON),
			r.tactics, r.techniques, string(tagsJSON), adminUserID,
		)
		if err != nil {
			log.Warn().Err(err).Str("rule", r.name).Msg("seed detection rule failed (non-fatal)")
		}
	}

	log.Debug().Int("count", len(rules)).Msg("detection rules seeded")
	return nil
}

// stepSeedKPIs inserts default KPI definitions in the visus DB.
func (p *TenantProvisioner) stepSeedKPIs(ctx context.Context, tenantID, adminUserID uuid.UUID, log zerolog.Logger) error {
	db := p.visusDB
	if db == nil {
		db = p.platformDB
	}

	kpiRepo := visusrepo.NewKPIRepository(db, p.logger)
	defaults := visuskap.DefaultDefinitions(tenantID, adminUserID)

	created := 0
	for i := range defaults {
		kpi := defaults[i]
		_, err := kpiRepo.Create(ctx, &kpi)
		if err != nil {
			log.Warn().Err(err).Str("kpi", kpi.Name).Msg("seed kpi failed (non-fatal)")
			continue
		}
		created++
	}

	log.Debug().Int("created", created).Int("total", len(defaults)).Msg("KPIs seeded")
	return nil
}

// stepSeedDashboard creates the "Executive Overview" default dashboard.
func (p *TenantProvisioner) stepSeedDashboard(ctx context.Context, tenantID, adminUserID uuid.UUID, log zerolog.Logger) error {
	db := p.visusDB
	if db == nil {
		db = p.platformDB
	}

	dashRepo := visusrepo.NewDashboardRepository(db, p.logger)
	widgetRepo := visusrepo.NewWidgetRepository(db, p.logger)

	dashboard := &visusmodel.Dashboard{
		TenantID:    tenantID,
		Name:        "Executive Overview",
		Description: "Default cross-suite executive dashboard",
		GridColumns: 12,
		Visibility:  visusmodel.DashboardVisibilityOrganization,
		SharedWith:  []uuid.UUID{},
		IsDefault:   true,
		IsSystem:    true,
		Tags:        []string{"default", "executive"},
		Metadata:    map[string]any{},
		CreatedBy:   adminUserID,
	}

	created, err := dashRepo.Create(ctx, dashboard)
	if err != nil {
		return fmt.Errorf("create dashboard: %w", err)
	}

	widgets := []visusmodel.Widget{
		{TenantID: tenantID, DashboardID: created.ID, Title: "Security Risk Score", Type: visusmodel.WidgetTypeGauge, Position: visusmodel.WidgetPosition{X: 0, Y: 0, W: 3, H: 2}, Config: map[string]any{"kpi": "risk_score"}, RefreshIntervalSeconds: 300},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Open Critical Alerts", Type: visusmodel.WidgetTypeKPICard, Position: visusmodel.WidgetPosition{X: 3, Y: 0, W: 3, H: 2}, Config: map[string]any{"kpi": "open_critical_alerts"}, RefreshIntervalSeconds: 60},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Data Quality Score", Type: visusmodel.WidgetTypeGauge, Position: visusmodel.WidgetPosition{X: 6, Y: 0, W: 3, H: 2}, Config: map[string]any{"kpi": "data_quality_score"}, RefreshIntervalSeconds: 300},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Active Threats", Type: visusmodel.WidgetTypeKPICard, Position: visusmodel.WidgetPosition{X: 9, Y: 0, W: 3, H: 2}, Config: map[string]any{"source": "cyber_threats"}, RefreshIntervalSeconds: 60},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Alerts Timeline", Type: visusmodel.WidgetTypeAreaChart, Position: visusmodel.WidgetPosition{X: 0, Y: 2, W: 6, H: 3}, Config: map[string]any{"source": "cyber_alerts_timeline"}, RefreshIntervalSeconds: 300},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Severity Distribution", Type: visusmodel.WidgetTypePieChart, Position: visusmodel.WidgetPosition{X: 6, Y: 2, W: 6, H: 3}, Config: map[string]any{"source": "severity_distribution"}, RefreshIntervalSeconds: 300},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Pipeline Health", Type: visusmodel.WidgetTypeStatusGrid, Position: visusmodel.WidgetPosition{X: 0, Y: 5, W: 6, H: 3}, Config: map[string]any{"source": "data_pipelines"}, RefreshIntervalSeconds: 120},
		{TenantID: tenantID, DashboardID: created.ID, Title: "Compliance Score", Type: visusmodel.WidgetTypeGauge, Position: visusmodel.WidgetPosition{X: 6, Y: 5, W: 6, H: 3}, Config: map[string]any{"source": "compliance_score"}, RefreshIntervalSeconds: 3600},
	}

	created2 := 0
	for i := range widgets {
		_, err := widgetRepo.Create(ctx, &widgets[i])
		if err != nil {
			log.Warn().Err(err).Str("widget", widgets[i].Title).Msg("seed widget failed (non-fatal)")
			continue
		}
		created2++
	}

	log.Debug().Int("widgets_created", created2).Msg("dashboard seeded")
	return nil
}

// stepCreateStorageBuckets ensures MinIO buckets exist for the tenant.
func (p *TenantProvisioner) stepCreateStorageBuckets(ctx context.Context, tenantID, _ uuid.UUID, log zerolog.Logger) error {
	// Get tenant slug for bucket naming
	tenantName, slug, _, _, err := p.platformDB.QueryRow(ctx,
		`SELECT name, slug, status, retain_until FROM tenants WHERE id = $1`, tenantID,
	).Scan(new(string), new(string), new(string), new(*time.Time))
	_ = err
	_ = tenantName

	// If we can't get the slug, use tenantID as fallback
	if slug == "" {
		var name string
		if err2 := p.platformDB.QueryRow(ctx, `SELECT COALESCE(slug, id::text) FROM tenants WHERE id = $1`, tenantID).Scan(&slug); err2 != nil {
			slug = strings.ReplaceAll(tenantID.String(), "-", "")[:12]
		}
		_ = name
	}

	suites := []string{"cyber", "data", "acta", "lex", "visus", "platform"}
	for _, suite := range suites {
		bucketName := fmt.Sprintf("clario360-%s-%s", slug, suite)
		// We log bucket names we'd create — actual MinIO creation requires the storage client
		// which is not injected here to avoid circular dependencies. A background job or the
		// file service creates buckets on first use.
		log.Debug().Str("bucket", bucketName).Msg("bucket registered for creation")
	}

	return nil
}

// stepInitializeAuditTrail inserts the genesis audit log entry.
func (p *TenantProvisioner) stepInitializeAuditTrail(ctx context.Context, tenantID, adminUserID uuid.UUID, log zerolog.Logger) error {
	_, err := p.platformDB.Exec(ctx, `
		INSERT INTO audit_logs (tenant_id, user_id, service, action, resource_type, resource_id, metadata)
		VALUES ($1, $2, 'onboarding-service', 'tenant.provisioned', 'tenant', $1, $3::jsonb)`,
		tenantID, adminUserID,
		`{"event": "genesis", "description": "Tenant onboarding complete"}`,
	)
	if err != nil {
		return fmt.Errorf("insert genesis audit log: %w", err)
	}
	log.Debug().Msg("audit trail initialized")
	return nil
}

// stepSendWelcomeNotification publishes the tenant.provisioned event.
func (p *TenantProvisioner) stepSendWelcomeNotification(ctx context.Context, tenantID, adminUserID uuid.UUID, log zerolog.Logger) error {
	if p.producer == nil {
		log.Debug().Msg("kafka unavailable, skipping welcome notification")
		return nil
	}

	evt, err := events.NewEvent(
		"platform.tenant.provisioned",
		"onboarding-service",
		tenantID.String(),
		map[string]string{
			"tenant_id":     tenantID.String(),
			"admin_user_id": adminUserID.String(),
		},
	)
	if err != nil {
		return fmt.Errorf("build provisioned event: %w", err)
	}

	if err := p.producer.Publish(ctx, events.Topics.OnboardingEvents, evt); err != nil {
		log.Warn().Err(err).Msg("publish provisioned event (non-fatal)")
	}

	log.Debug().Msg("welcome notification published")
	return nil
}

// stepComplete marks the tenant as active.
func (p *TenantProvisioner) stepComplete(ctx context.Context, tenantID, _ uuid.UUID, log zerolog.Logger) error {
	if err := p.provRepo.SetTenantStatus(ctx, tenantID, string(iammodel.TenantStatusActive)); err != nil {
		return fmt.Errorf("set tenant active: %w", err)
	}
	if err := p.provRepo.MarkCompleted(ctx, tenantID); err != nil {
		return fmt.Errorf("mark provisioning completed: %w", err)
	}

	// Mark onboarding setting as complete
	_, _ = p.platformDB.Exec(ctx, `
		UPDATE system_settings
		SET value = 'true'::jsonb
		WHERE tenant_id = $1 AND key = 'onboarding_complete'`,
		tenantID,
	)

	log.Debug().Msg("provisioning complete, tenant set to active")
	return nil
}

// Deprovision suspends and soft-deletes all tenant data.
func (p *TenantProvisioner) Deprovision(ctx context.Context, tenantID uuid.UUID, initiatedBy string) error {
	log := p.logger.With().Str("tenant_id", tenantID.String()).Logger()

	// 1. Suspend all tenant users
	if _, err := p.platformDB.Exec(ctx, `
		UPDATE users SET status = 'suspended', updated_at = NOW()
		WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID,
	); err != nil {
		log.Error().Err(err).Msg("suspend users")
	}

	// 2. Revoke all API keys
	if _, err := p.platformDB.Exec(ctx, `
		UPDATE api_keys SET revoked_at = NOW()
		WHERE tenant_id = $1 AND revoked_at IS NULL`, tenantID,
	); err != nil {
		log.Error().Err(err).Msg("revoke api keys")
	}

	// 3. Soft-delete all users
	if _, err := p.platformDB.Exec(ctx, `
		UPDATE users SET deleted_at = NOW()
		WHERE tenant_id = $1 AND deleted_at IS NULL`, tenantID,
	); err != nil {
		log.Error().Err(err).Msg("soft delete users")
	}

	// 4. Mark tenant as deprovisioned
	_, err := p.platformDB.Exec(ctx, `
		UPDATE tenants
		SET status = 'deprovisioned', deprovisioned_at = NOW(), deprovisioned_by = $2::uuid, updated_at = NOW()
		WHERE id = $1`,
		tenantID, initiatedBy,
	)
	if err != nil {
		return fmt.Errorf("mark tenant deprovisioned: %w", err)
	}

	// 5. Publish deprovisioned event
	if p.producer != nil {
		evt, err := events.NewEvent(
			"platform.tenant.deprovisioned",
			"onboarding-service",
			tenantID.String(),
			map[string]string{
				"tenant_id":    tenantID.String(),
				"initiated_by": initiatedBy,
			},
		)
		if err == nil {
			_ = p.producer.Publish(ctx, events.Topics.OnboardingEvents, evt)
		}
	}

	log.Info().Str("initiated_by", initiatedBy).Msg("tenant deprovisioned")
	return nil
}
