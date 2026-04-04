package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	"golang.org/x/crypto/bcrypt"

	"github.com/clario360/platform/internal/acta"
	actarepo "github.com/clario360/platform/internal/acta/repository"
	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/lex"
	lexconfig "github.com/clario360/platform/internal/lex/config"
	notifrepo "github.com/clario360/platform/internal/notification/repository"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/visus"
	visusconfig "github.com/clario360/platform/internal/visus/config"
	workflowrepo "github.com/clario360/platform/internal/workflow/repository"
)

//go:embed sql/*.sql
var seedSQLFS embed.FS

var allDatabases = []string{
	"platform_core",
	"cyber_db",
	"data_db",
	"acta_db",
	"lex_db",
	"visus_db",
	"audit_db",
	"notification_db",
}

var (
	mainTenantID          = uuid.MustParse("aaaaaaaa-0000-0000-0000-000000000001")
	mainAdminUserID       = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000001")
	securityManagerUserID = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000002")
	dataStewardUserID     = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000003")
	legalManagerUserID    = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000004")
	boardSecretaryUserID  = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000005")
	executiveUserID       = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000006")
	auditorUserID         = uuid.MustParse("bbbbbbbb-0000-0000-0000-000000000007")
	actaTenantID          = uuid.MustParse("11111111-1111-1111-1111-111111111111")
	actaAdminUserID       = uuid.MustParse("11111111-1111-1111-1111-111111111001")
	lexTenantID           = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	lexSystemUserID       = uuid.MustParse("22222222-2222-2222-2222-222222222201")
	seedNamespace         = uuid.MustParse("9bdfe4f5-7d16-4dc3-8387-6b7a79352a10")
)

type scaleProfile struct {
	Name                      string
	PlatformNotificationCount int
	WorkflowDefinitionCount   int
	WorkflowTemplateCount     int
	WorkflowInstanceCount     int
	WorkflowTaskCount         int
	AIModelCount              int
	AIPredictionLogCount      int
	NotificationCount         int
	NotificationDeliveryCount int
	IntegrationDeliveryCount  int
	AuditLogCount             int
	FileCount                 int
	FileAccessLogCount        int
	DataSourceCount           int
	DataModelCount            int
	PipelineCount             int
	PipelineRunCount          int
	PipelineRunLogCount       int
	QualityRuleCount          int
	QualityResultCount        int
	ContradictionCount        int
	DarkDataAssetCount        int
	SavedQueryCount           int
	AnalyticsAuditLogCount    int
	AssetCount                int
	VulnerabilityCount        int
	ThreatCount               int
	ThreatIndicatorCount      int
	DetectionRuleCount        int
	AlertCount                int
	SecurityEventCount        int
	CTEMAssessmentCount       int
	CTEMFindingCount          int
	RemediationActionCount    int
	DSPMAssetCount            int
	DSPMAccessMappingCount    int
	DSPMIdentityProfileCount  int
	DSPMAccessAuditCount      int
	DSPMRemediationCount      int
	UEBAProfileCount          int
	UEBAAccessEventCount      int
	UEBAAlertCount            int
	VCISOConversationCount    int
	VCISOMessageCount         int
	VCISOLLMAuditCount        int
	VCISOFeatureSnapshotCount int
	VCISOPredictionCount      int
}

type seedTemplateData struct {
	SeedKey               string
	SeedNamespace         string
	MainTenantID          string
	MainAdminUserID       string
	SecurityManagerUserID string
	DataStewardUserID     string
	LegalManagerUserID    string
	BoardSecretaryUserID  string
	ExecutiveUserID       string
	AuditorUserID         string
	ActaTenantID          string
	ActaAdminUserID       string
	LexTenantID           string
	LexSystemUserID       string
	Scale                 scaleProfile
}

type demoTenant struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	Domain           string
	SubscriptionTier string
	Settings         map[string]any
}

type demoUser struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Email     string
	FirstName string
	LastName  string
}

type demoRole struct {
	ID          uuid.UUID
	Name        string
	Slug        string
	Description string
	Permissions []string
}

type roleAssignment struct {
	UserID   uuid.UUID
	RoleSlug string
}

type systemSeeder struct {
	cfg           *config.Config
	logger        zerolog.Logger
	scale         scaleProfile
	templateData  seedTemplateData
	passwordHash  string
	migrationsDir string
	pools         map[string]*pgxpool.Pool
}

func main() {
	scaleName := flag.String("scale", "large", "Seed volume preset: small|large|massive")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "loading config: %v\n", err)
		os.Exit(1)
	}

	scale, err := parseScale(*scaleName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid scale: %v\n", err)
		os.Exit(1)
	}

	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"system-seeder",
	)

	passwordHashBytes, err := bcrypt.GenerateFromPassword([]byte("DemoPass123!"), 12)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to build demo password hash")
	}

	seeder := &systemSeeder{
		cfg:           cfg,
		logger:        logger,
		scale:         scale,
		passwordHash:  string(passwordHashBytes),
		migrationsDir: findMigrationsPath(),
		templateData: seedTemplateData{
			SeedKey:               "system_demo_v1",
			SeedNamespace:         seedNamespace.String(),
			MainTenantID:          mainTenantID.String(),
			MainAdminUserID:       mainAdminUserID.String(),
			SecurityManagerUserID: securityManagerUserID.String(),
			DataStewardUserID:     dataStewardUserID.String(),
			LegalManagerUserID:    legalManagerUserID.String(),
			BoardSecretaryUserID:  boardSecretaryUserID.String(),
			ExecutiveUserID:       executiveUserID.String(),
			AuditorUserID:         auditorUserID.String(),
			ActaTenantID:          actaTenantID.String(),
			ActaAdminUserID:       actaAdminUserID.String(),
			LexTenantID:           lexTenantID.String(),
			LexSystemUserID:       lexSystemUserID.String(),
			Scale:                 scale,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	if err := seeder.run(ctx); err != nil {
		logger.Fatal().Err(err).Msg("system seeding failed")
	}
}

func parseScale(name string) (scaleProfile, error) {
	switch name {
	case "small":
		return scaleProfile{
			Name:                      "small",
			PlatformNotificationCount: 250,
			WorkflowDefinitionCount:   6,
			WorkflowTemplateCount:     6,
			WorkflowInstanceCount:     400,
			WorkflowTaskCount:         1200,
			AIModelCount:              10,
			AIPredictionLogCount:      2500,
			NotificationCount:         2500,
			NotificationDeliveryCount: 2500,
			IntegrationDeliveryCount:  2000,
			AuditLogCount:             4000,
			FileCount:                 120,
			FileAccessLogCount:        800,
			DataSourceCount:           12,
			DataModelCount:            24,
			PipelineCount:             12,
			PipelineRunCount:          240,
			PipelineRunLogCount:       4000,
			QualityRuleCount:          60,
			QualityResultCount:        2400,
			ContradictionCount:        120,
			DarkDataAssetCount:        180,
			SavedQueryCount:           12,
			AnalyticsAuditLogCount:    3000,
			AssetCount:                400,
			VulnerabilityCount:        1200,
			ThreatCount:               80,
			ThreatIndicatorCount:      400,
			DetectionRuleCount:        16,
			AlertCount:                600,
			SecurityEventCount:        5000,
			CTEMAssessmentCount:       12,
			CTEMFindingCount:          400,
			RemediationActionCount:    160,
			DSPMAssetCount:            220,
			DSPMAccessMappingCount:    500,
			DSPMIdentityProfileCount:  80,
			DSPMAccessAuditCount:      4000,
			DSPMRemediationCount:      100,
			UEBAProfileCount:          60,
			UEBAAccessEventCount:      5000,
			UEBAAlertCount:            140,
			VCISOConversationCount:    25,
			VCISOMessageCount:         240,
			VCISOLLMAuditCount:        80,
			VCISOFeatureSnapshotCount: 300,
			VCISOPredictionCount:      120,
		}, nil
	case "large":
		return scaleProfile{
			Name:                      "large",
			PlatformNotificationCount: 1500,
			WorkflowDefinitionCount:   10,
			WorkflowTemplateCount:     10,
			WorkflowInstanceCount:     5000,
			WorkflowTaskCount:         18000,
			AIModelCount:              10,
			AIPredictionLogCount:      50000,
			NotificationCount:         50000,
			NotificationDeliveryCount: 50000,
			IntegrationDeliveryCount:  40000,
			AuditLogCount:             75000,
			FileCount:                 500,
			FileAccessLogCount:        6000,
			DataSourceCount:           18,
			DataModelCount:            40,
			PipelineCount:             24,
			PipelineRunCount:          1600,
			PipelineRunLogCount:       25000,
			QualityRuleCount:          120,
			QualityResultCount:        10000,
			ContradictionCount:        350,
			DarkDataAssetCount:        700,
			SavedQueryCount:           24,
			AnalyticsAuditLogCount:    25000,
			AssetCount:                1600,
			VulnerabilityCount:        8000,
			ThreatCount:               220,
			ThreatIndicatorCount:      1200,
			DetectionRuleCount:        28,
			AlertCount:                3500,
			SecurityEventCount:        60000,
			CTEMAssessmentCount:       24,
			CTEMFindingCount:          2200,
			RemediationActionCount:    650,
			DSPMAssetCount:            1400,
			DSPMAccessMappingCount:    4000,
			DSPMIdentityProfileCount:  320,
			DSPMAccessAuditCount:      50000,
			DSPMRemediationCount:      320,
			UEBAProfileCount:          220,
			UEBAAccessEventCount:      60000,
			UEBAAlertCount:            800,
			VCISOConversationCount:    80,
			VCISOMessageCount:         1200,
			VCISOLLMAuditCount:        400,
			VCISOFeatureSnapshotCount: 800,
			VCISOPredictionCount:      400,
		}, nil
	case "massive":
		return scaleProfile{
			Name:                      "massive",
			PlatformNotificationCount: 5000,
			WorkflowDefinitionCount:   12,
			WorkflowTemplateCount:     12,
			WorkflowInstanceCount:     15000,
			WorkflowTaskCount:         50000,
			AIModelCount:              10,
			AIPredictionLogCount:      150000,
			NotificationCount:         150000,
			NotificationDeliveryCount: 150000,
			IntegrationDeliveryCount:  100000,
			AuditLogCount:             200000,
			FileCount:                 1200,
			FileAccessLogCount:        15000,
			DataSourceCount:           24,
			DataModelCount:            60,
			PipelineCount:             36,
			PipelineRunCount:          5000,
			PipelineRunLogCount:       50000,
			QualityRuleCount:          180,
			QualityResultCount:        25000,
			ContradictionCount:        800,
			DarkDataAssetCount:        1800,
			SavedQueryCount:           40,
			AnalyticsAuditLogCount:    50000,
			AssetCount:                3000,
			VulnerabilityCount:        15000,
			ThreatCount:               420,
			ThreatIndicatorCount:      2400,
			DetectionRuleCount:        36,
			AlertCount:                8000,
			SecurityEventCount:        150000,
			CTEMAssessmentCount:       48,
			CTEMFindingCount:          4500,
			RemediationActionCount:    1400,
			DSPMAssetCount:            2500,
			DSPMAccessMappingCount:    8000,
			DSPMIdentityProfileCount:  600,
			DSPMAccessAuditCount:      100000,
			DSPMRemediationCount:      700,
			UEBAProfileCount:          500,
			UEBAAccessEventCount:      150000,
			UEBAAlertCount:            1600,
			VCISOConversationCount:    150,
			VCISOMessageCount:         3000,
			VCISOLLMAuditCount:        1200,
			VCISOFeatureSnapshotCount: 1600,
			VCISOPredictionCount:      900,
		}, nil
	default:
		return scaleProfile{}, fmt.Errorf("unsupported preset %q", name)
	}
}

func (s *systemSeeder) run(ctx context.Context) error {
	s.logger.Info().
		Str("scale", s.scale.Name).
		Str("migrations_dir", s.migrationsDir).
		Msg("starting system seeder")

	if err := s.ensureDatabases(ctx); err != nil {
		return err
	}
	if err := s.runMigrations(ctx); err != nil {
		return err
	}
	if err := s.openPools(ctx); err != nil {
		return err
	}
	defer s.closePools()

	if err := s.runSchemaPatches(ctx); err != nil {
		return err
	}
	if err := s.seedPlatformBase(ctx); err != nil {
		return err
	}
	if err := s.seedSuiteDatasets(ctx); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["platform_core"], mainTenantID, "platform_core.sql"); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["platform_core"], mainTenantID, "file_storage.sql"); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["notification_db"], mainTenantID, "notification.sql"); err != nil {
		return err
	}
	if err := s.ensureAuditPartition(ctx); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["audit_db"], mainTenantID, "audit.sql"); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["data_db"], mainTenantID, "data.sql"); err != nil {
		return err
	}
	if err := s.ensureCyberPartitions(ctx); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["cyber_db"], mainTenantID, "cyber.sql"); err != nil {
		return err
	}
	if err := s.execTenantSQL(ctx, s.pools["cyber_db"], mainTenantID, "vciso_governance.sql"); err != nil {
		return err
	}
	if err := s.seedActaComplianceChecks(ctx); err != nil {
		return err
	}
	if err := s.seedLexExpiryNotifications(ctx); err != nil {
		return err
	}
	if err := s.reportCounts(ctx); err != nil {
		return err
	}

	s.logger.Info().Str("scale", s.scale.Name).Msg("system seeding completed")
	return nil
}

func (s *systemSeeder) ensureDatabases(ctx context.Context) error {
	adminDSN := s.buildDSN("postgres")
	adminPool, err := pgxpool.New(ctx, adminDSN)
	if err != nil {
		return fmt.Errorf("open postgres admin pool: %w", err)
	}
	defer adminPool.Close()

	for _, dbName := range allDatabases {
		var exists bool
		if err := adminPool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName).Scan(&exists); err != nil {
			return fmt.Errorf("check database %s: %w", dbName, err)
		}
		if exists {
			continue
		}
		if _, err := adminPool.Exec(ctx, fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)); err != nil {
			return fmt.Errorf("create database %s: %w", dbName, err)
		}
		s.logger.Info().Str("database", dbName).Msg("created database")
	}
	return nil
}

func (s *systemSeeder) runMigrations(ctx context.Context) error {
	for _, dbName := range allDatabases {
		migrationsPath := filepath.Join(s.migrationsDir, dbName)
		dsn := s.buildDSN(dbName)
		if err := s.normalizeMigrationState(ctx, dbName, dsn, migrationsPath); err != nil {
			return err
		}
		s.logger.Info().Str("database", dbName).Str("path", migrationsPath).Msg("running migrations")
		if err := database.RunMigrations(dsn, migrationsPath); err != nil {
			if exists, probeErr := s.sentinelTableExists(ctx, dsn, migrationSentinelTable(dbName)); probeErr == nil && exists {
				s.logger.Warn().
					Err(err).
					Str("database", dbName).
					Str("sentinel_table", migrationSentinelTable(dbName)).
					Msg("migration failed but sentinel table exists; normalizing migration state and continuing")
				if normalizeErr := s.forceMigrationClean(ctx, dsn, migrationsPath); normalizeErr != nil {
					return fmt.Errorf("normalize migration state for %s after failure: %w", dbName, normalizeErr)
				}
			} else {
				return fmt.Errorf("run migrations for %s: %w", dbName, err)
			}
		}

		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("open pool for extension setup on %s: %w", dbName, err)
		}
		if _, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"; CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`); err != nil {
			pool.Close()
			return fmt.Errorf("ensure extensions for %s: %w", dbName, err)
		}
		pool.Close()
	}
	return nil
}

func (s *systemSeeder) normalizeMigrationState(ctx context.Context, dbName, dsn, migrationsPath string) error {
	exists, err := s.schemaMigrationsExists(ctx, dsn)
	if err != nil || !exists {
		return err
	}

	var version int
	var dirty bool
	if err := s.readSchemaMigrationState(ctx, dsn, &version, &dirty); err != nil {
		return err
	}
	if !dirty {
		return nil
	}

	sentinel := migrationSentinelTable(dbName)
	ok, err := s.sentinelTableExists(ctx, dsn, sentinel)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := s.forceMigrationClean(ctx, dsn, migrationsPath); err != nil {
		return err
	}

	s.logger.Warn().
		Str("database", dbName).
		Int("dirty_version", version).
		Str("sentinel_table", sentinel).
		Msg("cleared dirty migration state because schema objects already exist")
	return nil
}

func (s *systemSeeder) schemaMigrationsExists(ctx context.Context, dsn string) (bool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return false, fmt.Errorf("open pool for migration state: %w", err)
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('public.schema_migrations') IS NOT NULL`).Scan(&exists); err != nil {
		return false, fmt.Errorf("check schema_migrations existence: %w", err)
	}
	return exists, nil
}

func (s *systemSeeder) readSchemaMigrationState(ctx context.Context, dsn string, version *int, dirty *bool) error {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("open pool for migration state read: %w", err)
	}
	defer pool.Close()

	if err := pool.QueryRow(ctx, `SELECT version, dirty FROM schema_migrations LIMIT 1`).Scan(version, dirty); err != nil {
		return fmt.Errorf("read schema_migrations state: %w", err)
	}
	return nil
}

func (s *systemSeeder) sentinelTableExists(ctx context.Context, dsn, tableName string) (bool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return false, fmt.Errorf("open pool for sentinel probe: %w", err)
	}
	defer pool.Close()

	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass($1) IS NOT NULL`, "public."+tableName).Scan(&exists); err != nil {
		return false, fmt.Errorf("probe sentinel table %s: %w", tableName, err)
	}
	return exists, nil
}

func (s *systemSeeder) forceMigrationClean(ctx context.Context, dsn, migrationsPath string) error {
	latest, err := latestMigrationVersion(migrationsPath)
	if err != nil {
		return err
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("open pool for migration normalization: %w", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `UPDATE schema_migrations SET version = $1, dirty = false`, latest); err != nil {
		return fmt.Errorf("mark schema_migrations clean: %w", err)
	}
	return nil
}

func latestMigrationVersion(migrationsPath string) (int, error) {
	entries, err := os.ReadDir(migrationsPath)
	if err != nil {
		return 0, fmt.Errorf("read migrations directory %s: %w", migrationsPath, err)
	}

	latest := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		parts := strings.SplitN(name, "_", 2)
		if len(parts) == 0 {
			continue
		}
		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}
		if version > latest {
			latest = version
		}
	}
	if latest == 0 {
		return 0, fmt.Errorf("no migration versions found in %s", migrationsPath)
	}
	return latest, nil
}

func migrationSentinelTable(dbName string) string {
	switch dbName {
	case "platform_core":
		return "tenants"
	case "cyber_db":
		return "assets"
	case "data_db":
		return "data_sources"
	case "acta_db":
		return "committees"
	case "lex_db":
		return "contracts"
	case "visus_db":
		return "visus_dashboards"
	case "audit_db":
		return "audit_logs"
	case "notification_db":
		return "notifications"
	default:
		return "schema_migrations"
	}
}

func (s *systemSeeder) openPools(ctx context.Context) error {
	s.pools = make(map[string]*pgxpool.Pool, len(allDatabases))
	for _, dbName := range allDatabases {
		pool, err := pgxpool.New(ctx, s.buildDSN(dbName))
		if err != nil {
			return fmt.Errorf("open pool %s: %w", dbName, err)
		}
		if err := pool.Ping(ctx); err != nil {
			pool.Close()
			return fmt.Errorf("ping pool %s: %w", dbName, err)
		}
		s.pools[dbName] = pool
	}
	return nil
}

func (s *systemSeeder) closePools() {
	for _, pool := range s.pools {
		pool.Close()
	}
}

func (s *systemSeeder) runSchemaPatches(ctx context.Context) error {
	if err := workflowrepo.RunMigration(ctx, s.pools["platform_core"]); err != nil {
		return fmt.Errorf("workflow migration platform_core: %w", err)
	}
	if err := workflowrepo.RunMigration(ctx, s.pools["acta_db"]); err != nil {
		return fmt.Errorf("workflow migration acta_db: %w", err)
	}
	if err := workflowrepo.RunMigration(ctx, s.pools["lex_db"]); err != nil {
		return fmt.Errorf("workflow migration lex_db: %w", err)
	}
	if err := notifrepo.RunMigration(ctx, s.pools["notification_db"]); err != nil {
		return fmt.Errorf("notification schema migration: %w", err)
	}
	return nil
}

func (s *systemSeeder) seedPlatformBase(ctx context.Context) error {
	platformPool := s.pools["platform_core"]
	tenants := []demoTenant{
		{
			ID:               mainTenantID,
			Name:             "Apex Bank Holdings",
			Slug:             "apex-bank-holdings",
			Domain:           "demo.apexbank.clario.local",
			SubscriptionTier: "enterprise",
			Settings: map[string]any{
				"timezone": "Africa/Lagos",
				"industry": "financial",
				"seeded":   true,
			},
		},
		{
			ID:               actaTenantID,
			Name:             "Apex Governance Board",
			Slug:             "apex-governance-board",
			Domain:           "acta.apex.clario.local",
			SubscriptionTier: "enterprise",
			Settings:         map[string]any{"suite": "acta", "seeded": true},
		},
		{
			ID:               lexTenantID,
			Name:             "Apex Legal Operations",
			Slug:             "apex-legal-operations",
			Domain:           "lex.apex.clario.local",
			SubscriptionTier: "enterprise",
			Settings:         map[string]any{"suite": "lex", "seeded": true},
		},
	}

	for _, tenant := range tenants {
		settingsJSON, err := json.Marshal(tenant.Settings)
		if err != nil {
			return fmt.Errorf("marshal tenant settings: %w", err)
		}
		if _, err := platformPool.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, domain, settings, status, subscription_tier)
			VALUES ($1, $2, $3, $4, $5::jsonb, 'active', $6)
			ON CONFLICT (id) DO UPDATE SET
				name = EXCLUDED.name,
				slug = EXCLUDED.slug,
				domain = EXCLUDED.domain,
				settings = EXCLUDED.settings,
				status = 'active',
				subscription_tier = EXCLUDED.subscription_tier,
				updated_at = now()`,
			tenant.ID,
			tenant.Name,
			tenant.Slug,
			tenant.Domain,
			settingsJSON,
			tenant.SubscriptionTier,
		); err != nil {
			return fmt.Errorf("upsert tenant %s: %w", tenant.Slug, err)
		}
	}

	mainUsers := []demoUser{
		{ID: mainAdminUserID, TenantID: mainTenantID, Email: "admin@apexbank.demo", FirstName: "Ada", LastName: "Okafor"},
		{ID: securityManagerUserID, TenantID: mainTenantID, Email: "security@apexbank.demo", FirstName: "Musa", LastName: "Adebayo"},
		{ID: dataStewardUserID, TenantID: mainTenantID, Email: "data@apexbank.demo", FirstName: "Ifeoma", LastName: "Nwosu"},
		{ID: legalManagerUserID, TenantID: mainTenantID, Email: "legal@apexbank.demo", FirstName: "Lara", LastName: "Bamidele"},
		{ID: boardSecretaryUserID, TenantID: mainTenantID, Email: "board@apexbank.demo", FirstName: "Tade", LastName: "Akinola"},
		{ID: executiveUserID, TenantID: mainTenantID, Email: "executive@apexbank.demo", FirstName: "Chika", LastName: "Nwachukwu"},
		{ID: auditorUserID, TenantID: mainTenantID, Email: "audit@apexbank.demo", FirstName: "Emeka", LastName: "Daniels"},
	}
	mainAssignments := []roleAssignment{
		{UserID: mainAdminUserID, RoleSlug: "tenant-admin"},
		{UserID: securityManagerUserID, RoleSlug: "security-manager"},
		{UserID: dataStewardUserID, RoleSlug: "data-steward"},
		{UserID: legalManagerUserID, RoleSlug: "legal-manager"},
		{UserID: boardSecretaryUserID, RoleSlug: "board-secretary"},
		{UserID: executiveUserID, RoleSlug: "executive"},
		{UserID: auditorUserID, RoleSlug: "auditor"},
	}
	if err := s.seedTenantUsersAndRoles(ctx, platformPool, mainTenantID, mainUsers, mainAssignments, true); err != nil {
		return err
	}

	actaUsers := []demoUser{
		{ID: actaAdminUserID, TenantID: actaTenantID, Email: "board-admin@acta.demo", FirstName: "Nadia", LastName: "Rahman"},
	}
	if err := s.seedTenantUsersAndRoles(ctx, platformPool, actaTenantID, actaUsers, []roleAssignment{{UserID: actaAdminUserID, RoleSlug: "tenant-admin"}}, false); err != nil {
		return err
	}

	lexUsers := []demoUser{
		{ID: lexSystemUserID, TenantID: lexTenantID, Email: "system@lex.demo", FirstName: "Legal", LastName: "System"},
	}
	if err := s.seedTenantUsersAndRoles(ctx, platformPool, lexTenantID, lexUsers, []roleAssignment{{UserID: lexSystemUserID, RoleSlug: "tenant-admin"}}, false); err != nil {
		return err
	}

	return nil
}

func (s *systemSeeder) seedTenantUsersAndRoles(
	ctx context.Context,
	pool *pgxpool.Pool,
	tenantID uuid.UUID,
	users []demoUser,
	assignments []roleAssignment,
	withOnboarding bool,
) error {
	return database.RunWithTenant(ctx, pool, tenantID, func(tx pgx.Tx) error {
		for _, role := range defaultRoles(tenantID) {
			permsJSON, err := json.Marshal(role.Permissions)
			if err != nil {
				return fmt.Errorf("marshal permissions for %s: %w", role.Slug, err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO roles (id, tenant_id, name, slug, description, is_system_role, permissions)
				VALUES ($1, $2, $3, $4, $5, true, $6::jsonb)
				ON CONFLICT (tenant_id, slug) DO UPDATE SET
					name = EXCLUDED.name,
					description = EXCLUDED.description,
					is_system_role = true,
					permissions = EXCLUDED.permissions,
					updated_at = now()`,
				role.ID, tenantID, role.Name, role.Slug, role.Description, permsJSON,
			); err != nil {
				return fmt.Errorf("upsert role %s: %w", role.Slug, err)
			}
		}

		for idx, user := range users {
			var createdBy any
			if idx > 0 {
				createdBy = users[0].ID
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO users (
					id, tenant_id, email, password_hash, first_name, last_name,
					status, mfa_enabled, last_login_at, created_by, updated_by
				) VALUES (
					$1, $2, $3, $4, $5, $6,
					'active', false, now() - interval '2 hours', $7, $7
				)
				ON CONFLICT (id) DO UPDATE SET
					email = EXCLUDED.email,
					password_hash = EXCLUDED.password_hash,
					first_name = EXCLUDED.first_name,
					last_name = EXCLUDED.last_name,
					status = 'active',
					last_login_at = EXCLUDED.last_login_at,
					updated_at = now(),
					deleted_at = NULL`,
				user.ID, tenantID, user.Email, s.passwordHash, user.FirstName, user.LastName, createdBy,
			); err != nil {
				return fmt.Errorf("upsert user %s: %w", user.Email, err)
			}
		}

		for _, assignment := range assignments {
			if _, err := tx.Exec(ctx, `
				INSERT INTO user_roles (user_id, role_id, tenant_id, assigned_by)
				SELECT $1, id, $2, $3
				FROM roles
				WHERE tenant_id = $2 AND slug = $4
				ON CONFLICT (user_id, role_id) DO UPDATE SET assigned_by = EXCLUDED.assigned_by`,
				assignment.UserID, tenantID, users[0].ID, assignment.RoleSlug,
			); err != nil {
				return fmt.Errorf("assign role %s: %w", assignment.RoleSlug, err)
			}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO system_settings (id, tenant_id, key, value, description, updated_by)
			VALUES
				($1, $2, 'general.timezone', '{"value":"Africa/Lagos"}', 'Seeded demo timezone', $3),
				($4, $2, 'notification.email_enabled', '{"value":true}', 'Seeded email notification default', $3),
				($5, $2, 'data.pii_detection_enabled', '{"value":true}', 'Seeded PII detection default', $3),
				($6, $2, 'cyber.auto_remediation_enabled', '{"value":false}', 'Seeded auto-remediation default', $3)
			ON CONFLICT (tenant_id, key) DO UPDATE SET
				value = EXCLUDED.value,
				description = EXCLUDED.description,
				updated_by = EXCLUDED.updated_by,
				updated_at = now()`,
			uuid.NewSHA1(tenantID, []byte("setting:general.timezone")),
			tenantID,
			users[0].ID,
			uuid.NewSHA1(tenantID, []byte("setting:notification.email_enabled")),
			uuid.NewSHA1(tenantID, []byte("setting:data.pii_detection_enabled")),
			uuid.NewSHA1(tenantID, []byte("setting:cyber.auto_remediation_enabled")),
		); err != nil {
			return fmt.Errorf("upsert system settings: %w", err)
		}

		if withOnboarding {
			if err := s.seedMainTenantOnboarding(ctx, tx, users[0]); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *systemSeeder) seedMainTenantOnboarding(ctx context.Context, tx pgx.Tx, admin demoUser) error {
	onboardingID := uuid.NewSHA1(mainTenantID, []byte("onboarding"))
	if _, err := tx.Exec(ctx, `
		INSERT INTO tenant_onboarding (
			id, tenant_id, admin_user_id, admin_email, email_verified, email_verified_at,
			current_step, steps_completed, wizard_completed, wizard_completed_at,
			org_name, org_industry, org_country, org_city, org_size,
			primary_color, accent_color, active_suites, provisioning_status,
			provisioning_started_at, provisioning_completed_at, referral_source
		) VALUES (
			$1, $2, $3, $4, true, now() - interval '10 days',
			6, '{1,2,3,4,5,6}', true, now() - interval '8 days',
			'Apex Bank Holdings', 'financial', 'NG', 'Lagos', '1000+',
			'#0b5fff', '#18a957', '{cyber,data,acta,lex,visus}', 'completed',
			now() - interval '10 days', now() - interval '8 days', 'demo_seed'
		)
		ON CONFLICT (tenant_id) DO UPDATE SET
			admin_user_id = EXCLUDED.admin_user_id,
			admin_email = EXCLUDED.admin_email,
			email_verified = EXCLUDED.email_verified,
			email_verified_at = EXCLUDED.email_verified_at,
			current_step = EXCLUDED.current_step,
			steps_completed = EXCLUDED.steps_completed,
			wizard_completed = EXCLUDED.wizard_completed,
			wizard_completed_at = EXCLUDED.wizard_completed_at,
			org_name = EXCLUDED.org_name,
			org_industry = EXCLUDED.org_industry,
			org_country = EXCLUDED.org_country,
			org_city = EXCLUDED.org_city,
			org_size = EXCLUDED.org_size,
			primary_color = EXCLUDED.primary_color,
			accent_color = EXCLUDED.accent_color,
			active_suites = EXCLUDED.active_suites,
			provisioning_status = EXCLUDED.provisioning_status,
			provisioning_started_at = EXCLUDED.provisioning_started_at,
			provisioning_completed_at = EXCLUDED.provisioning_completed_at,
			referral_source = EXCLUDED.referral_source,
			updated_at = now()`,
		onboardingID, mainTenantID, admin.ID, admin.Email,
	); err != nil {
		return fmt.Errorf("upsert tenant onboarding: %w", err)
	}

	stepNames := []string{
		"Create tenant",
		"Verify admin email",
		"Provision core platform",
		"Seed roles and defaults",
		"Enable suites",
		"Finalize setup",
	}
	for idx, name := range stepNames {
		if _, err := tx.Exec(ctx, `
			INSERT INTO provisioning_steps (
				id, tenant_id, onboarding_id, step_number, step_name, status,
				started_at, completed_at, duration_ms, retry_count, idempotency_key, metadata
			) VALUES (
				$1, $2, $3, $4, $5, 'completed',
				now() - make_interval(days => $6),
				now() - make_interval(days => $7),
				$8, 0, $9, '{"seeded":true}'::jsonb
			)
			ON CONFLICT (onboarding_id, step_number) DO UPDATE SET
				step_name = EXCLUDED.step_name,
				status = EXCLUDED.status,
				started_at = EXCLUDED.started_at,
				completed_at = EXCLUDED.completed_at,
				duration_ms = EXCLUDED.duration_ms,
				idempotency_key = EXCLUDED.idempotency_key,
				metadata = EXCLUDED.metadata`,
			uuid.NewSHA1(mainTenantID, []byte(fmt.Sprintf("provisioning-step-%d", idx+1))),
			mainTenantID,
			onboardingID,
			idx+1,
			name,
			10-idx,
			9-idx,
			(int64(idx)+1)*45000,
			fmt.Sprintf("seed-step-%d", idx+1),
		); err != nil {
			return fmt.Errorf("upsert provisioning step %d: %w", idx+1, err)
		}
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO email_verifications (
			id, email, otp_hash, purpose, verified, attempts, expires_at, verified_at, ip_address, user_agent
		) VALUES (
			$1, $2, $3, 'registration', true, 1, now() + interval '30 days', now() - interval '10 days',
			'127.0.0.1', 'Clario Demo Seeder'
		)
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email,
			otp_hash = EXCLUDED.otp_hash,
			verified = EXCLUDED.verified,
			attempts = EXCLUDED.attempts,
			expires_at = EXCLUDED.expires_at,
			verified_at = EXCLUDED.verified_at`,
		uuid.NewSHA1(mainTenantID, []byte("email-verification")),
		admin.Email,
		sha256Hex("000000"),
	); err != nil {
		return fmt.Errorf("upsert email verification: %w", err)
	}

	type inviteSeed struct {
		id       uuid.UUID
		email    string
		roleSlug string
	}
	invites := []inviteSeed{
		{id: uuid.NewSHA1(mainTenantID, []byte("invite-security")), email: "analyst@apexbank.demo", roleSlug: "security-manager"},
		{id: uuid.NewSHA1(mainTenantID, []byte("invite-data")), email: "analyst.data@apexbank.demo", roleSlug: "data-steward"},
	}
	for _, invite := range invites {
		if _, err := tx.Exec(ctx, `
			INSERT INTO invitations (
				id, tenant_id, email, role_slug, token_hash, token_prefix, status,
				invited_by, invited_by_name, expires_at, message
			) VALUES (
				$1, $2, $3, $4, $5, $6, 'pending',
				$7, 'Ada Okafor', now() + interval '21 days', 'Seeded demonstration invitation'
			)
			ON CONFLICT (id) DO UPDATE SET
				email = EXCLUDED.email,
				role_slug = EXCLUDED.role_slug,
				token_hash = EXCLUDED.token_hash,
				token_prefix = EXCLUDED.token_prefix,
				status = EXCLUDED.status,
				expires_at = EXCLUDED.expires_at,
				message = EXCLUDED.message,
				updated_at = now()`,
			invite.id,
			mainTenantID,
			invite.email,
			invite.roleSlug,
			sha256Hex(invite.email+":invite"),
			"inv_demo",
			admin.ID,
		); err != nil {
			return fmt.Errorf("upsert invitation %s: %w", invite.email, err)
		}
	}

	return nil
}

func (s *systemSeeder) seedSuiteDatasets(ctx context.Context) error {
	actaStore := actarepo.NewStore(s.pools["acta_db"], s.logger)
	if _, err := acta.SeedDemoData(ctx, actaStore, s.logger); err != nil {
		return fmt.Errorf("seed acta demo data: %w", err)
	}

	lexCfg := lexconfig.Default()
	lexApp, err := lex.NewApplication(lex.Dependencies{
		DB:               s.pools["lex_db"],
		Logger:           s.logger,
		WorkflowDefRepo:  workflowrepo.NewDefinitionRepository(s.pools["lex_db"]),
		WorkflowInstRepo: workflowrepo.NewInstanceRepository(s.pools["lex_db"]),
		WorkflowTaskRepo: workflowrepo.NewTaskRepository(s.pools["lex_db"]),
		Config:           lexCfg,
		OrgJurisdiction:  lexCfg.OrgJurisdiction,
	})
	if err != nil {
		return fmt.Errorf("create lex application: %w", err)
	}
	if _, err := lex.SeedDemoData(ctx, lexApp, s.logger); err != nil {
		return fmt.Errorf("seed lex demo data: %w", err)
	}

	visusCfg := visusconfig.Default()
	visusCfg.DemoTenantID = mainTenantID.String()
	visusCfg.DemoUserID = mainAdminUserID.String()
	visusApp, err := visus.NewApplication(visus.Dependencies{
		DB:     s.pools["visus_db"],
		Logger: s.logger,
		Config: visusCfg,
	})
	if err != nil {
		return fmt.Errorf("create visus application: %w", err)
	}
	if _, err := visus.SeedDemoData(ctx, visusApp, visusCfg, s.logger); err != nil {
		return fmt.Errorf("seed visus demo data: %w", err)
	}

	return nil
}

func (s *systemSeeder) seedActaComplianceChecks(ctx context.Context) error {
	rows, err := s.pools["acta_db"].Query(ctx, `
		SELECT id, name
		FROM committees
		WHERE tenant_id = $1
		ORDER BY created_at ASC
		LIMIT 8`, actaTenantID)
	if err != nil {
		return fmt.Errorf("list acta committees: %w", err)
	}
	defer rows.Close()

	idx := 0
	for rows.Next() {
		var committeeID uuid.UUID
		var committeeName string
		if err := rows.Scan(&committeeID, &committeeName); err != nil {
			return fmt.Errorf("scan acta committee: %w", err)
		}
		checkID := uuid.NewSHA1(actaTenantID, []byte("compliance:"+committeeID.String()))
		if _, err := s.pools["acta_db"].Exec(ctx, `
			INSERT INTO compliance_checks (
				id, tenant_id, committee_id, check_type, check_name, status, severity,
				description, finding, recommendation, evidence, period_start, period_end, checked_at, checked_by
			) VALUES (
				$1, $2, $3, 'meeting_frequency', $4, $5, $6,
				$7, $8, $9, $10::jsonb, current_date - interval '90 days', current_date,
				now() - make_interval(days => $11), 'system'
			)
			ON CONFLICT (id) DO NOTHING`,
			checkID,
			actaTenantID,
			committeeID,
			fmt.Sprintf("%s quarterly governance review", committeeName),
			[]string{"warning", "compliant", "non_compliant"}[idx%3],
			[]string{"medium", "low", "high"}[idx%3],
			fmt.Sprintf("Seeded compliance review for %s", committeeName),
			fmt.Sprintf("%s has seeded governance findings for demonstrations.", committeeName),
			"Review committee cadence and board pack timeliness.",
			`{"seeded":true,"module":"acta"}`,
			idx+1,
		); err != nil {
			return fmt.Errorf("insert acta compliance check: %w", err)
		}
		idx++
	}
	return rows.Err()
}

func (s *systemSeeder) seedLexExpiryNotifications(ctx context.Context) error {
	rows, err := s.pools["lex_db"].Query(ctx, `
		SELECT id
		FROM contracts
		WHERE tenant_id = $1
		  AND expiry_date IS NOT NULL
		ORDER BY expiry_date ASC
		LIMIT 12`, lexTenantID)
	if err != nil {
		return fmt.Errorf("list lex contracts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var contractID uuid.UUID
		if err := rows.Scan(&contractID); err != nil {
			return fmt.Errorf("scan lex contract: %w", err)
		}
		for _, horizon := range []int{30, 60, 90} {
			id := uuid.NewSHA1(lexTenantID, []byte(fmt.Sprintf("expiry:%s:%d", contractID, horizon)))
			if _, err := s.pools["lex_db"].Exec(ctx, `
				INSERT INTO expiry_notifications (id, tenant_id, contract_id, horizon_days, sent_at)
				VALUES ($1, $2, $3, $4, now() - interval '1 day')
				ON CONFLICT (contract_id, horizon_days) DO NOTHING`,
				id, lexTenantID, contractID, horizon,
			); err != nil {
				return fmt.Errorf("insert expiry notification: %w", err)
			}
		}
	}
	return rows.Err()
}

func (s *systemSeeder) execTenantSQL(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fileName string) error {
	rendered, err := s.renderSQL(fileName)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(rendered)) == 0 {
		return nil
	}
	s.logger.Info().Str("file", fileName).Str("tenant_id", tenantID.String()).Msg("applying seed sql")
	return database.RunWithTenant(ctx, pool, tenantID, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, string(rendered)); err != nil {
			return fmt.Errorf("exec %s: %w", fileName, err)
		}
		return nil
	})
}

func (s *systemSeeder) renderSQL(fileName string) ([]byte, error) {
	raw, err := seedSQLFS.ReadFile("sql/" + fileName)
	if err != nil {
		return nil, fmt.Errorf("read embedded sql %s: %w", fileName, err)
	}
	tmpl, err := template.New(fileName).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("parse sql template %s: %w", fileName, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s.templateData); err != nil {
		return nil, fmt.Errorf("render sql template %s: %w", fileName, err)
	}
	return buf.Bytes(), nil
}

func (s *systemSeeder) ensureAuditPartition(ctx context.Context) error {
	_, err := s.pools["audit_db"].Exec(ctx, `
		DO $$
		DECLARE
			start_month DATE := date_trunc('month', CURRENT_DATE)::date;
			end_month   DATE := (start_month + INTERVAL '1 month')::date;
			partition_name TEXT := format('audit_logs_%s', to_char(start_month, 'YYYY_MM'));
		BEGIN
			EXECUTE format(
				'CREATE TABLE IF NOT EXISTS %I PARTITION OF audit_logs FOR VALUES FROM (%L) TO (%L)',
				partition_name, start_month, end_month
			);
		END $$;`)
	if err != nil {
		return fmt.Errorf("ensure audit partition: %w", err)
	}
	return nil
}

func (s *systemSeeder) ensureCyberPartitions(ctx context.Context) error {
	_, err := s.pools["cyber_db"].Exec(ctx, `
		DO $$
		DECLARE
			start_month DATE := date_trunc('month', CURRENT_DATE)::date;
			partition_date DATE;
			partition_name TEXT;
		BEGIN
			IF to_regprocedure('create_security_events_partition(date)') IS NOT NULL THEN
				PERFORM create_security_events_partition(start_month);
				PERFORM create_security_events_partition((start_month + INTERVAL '1 month')::date);
			END IF;

			FOR i IN 0..1 LOOP
				partition_date := (start_month + make_interval(months => i))::date;

				partition_name := 'ueba_access_events_' || to_char(partition_date, 'YYYY_MM');
				EXECUTE format(
					'CREATE TABLE IF NOT EXISTS %I PARTITION OF ueba_access_events FOR VALUES FROM (%L) TO (%L)',
					partition_name,
					date_trunc('month', partition_date),
					date_trunc('month', partition_date + interval '1 month')
				);

				partition_name := 'dspm_access_audit_' || to_char(partition_date, 'YYYY_MM');
				EXECUTE format(
					'CREATE TABLE IF NOT EXISTS %I PARTITION OF dspm_access_audit FOR VALUES FROM (%L) TO (%L)',
					partition_name,
					date_trunc('month', partition_date),
					date_trunc('month', partition_date + interval '1 month')
				);
			END LOOP;
		END $$;`)
	if err != nil {
		return fmt.Errorf("ensure cyber partitions: %w", err)
	}
	return nil
}

func (s *systemSeeder) reportCounts(ctx context.Context) error {
	type countQuery struct {
		db    string
		label string
		sql   string
		args  []any
	}

	queries := []countQuery{
		{db: "platform_core", label: "platform.tenants", sql: `SELECT COUNT(*) FROM tenants WHERE id IN ($1, $2, $3)`, args: []any{mainTenantID, actaTenantID, lexTenantID}},
		{db: "platform_core", label: "platform.users", sql: `SELECT COUNT(*) FROM users WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "platform_core", label: "platform.ai_prediction_logs", sql: `SELECT COUNT(*) FROM ai_prediction_logs WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "platform_core", label: "platform.workflow_tasks", sql: `SELECT COUNT(*) FROM workflow_tasks WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "platform_core", label: "platform.files", sql: `SELECT COUNT(*) FROM files WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "notification_db", label: "notification.notifications", sql: `SELECT COUNT(*) FROM notifications WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "notification_db", label: "notification.integration_deliveries", sql: `SELECT COUNT(*) FROM integration_deliveries WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "audit_db", label: "audit.audit_logs", sql: `SELECT COUNT(*) FROM audit_logs WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.data_sources", sql: `SELECT COUNT(*) FROM data_sources WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.quality_results", sql: `SELECT COUNT(*) FROM quality_results WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.contradictions", sql: `SELECT COUNT(*) FROM contradictions WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.dark_data_assets", sql: `SELECT COUNT(*) FROM dark_data_assets WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.pipeline_run_logs", sql: `SELECT COUNT(*) FROM pipeline_run_logs WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "data_db", label: "data.analytics_audit_log", sql: `SELECT COUNT(*) FROM analytics_audit_log WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.assets", sql: `SELECT COUNT(*) FROM assets WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vulnerabilities", sql: `SELECT COUNT(*) FROM vulnerabilities WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.alerts", sql: `SELECT COUNT(*) FROM alerts WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.security_events", sql: `SELECT COUNT(*) FROM security_events WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.ctem_findings", sql: `SELECT COUNT(*) FROM ctem_findings WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.remediation_actions", sql: `SELECT COUNT(*) FROM remediation_actions WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.dspm_data_assets", sql: `SELECT COUNT(*) FROM dspm_data_assets WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.ueba_access_events", sql: `SELECT COUNT(*) FROM ueba_access_events WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.ueba_alerts", sql: `SELECT COUNT(*) FROM ueba_alerts WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.dspm_access_audit", sql: `SELECT COUNT(*) FROM dspm_access_audit WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_risks", sql: `SELECT COUNT(*) FROM vciso_risks WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_policies", sql: `SELECT COUNT(*) FROM vciso_policies WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_vendors", sql: `SELECT COUNT(*) FROM vciso_vendors WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_evidence", sql: `SELECT COUNT(*) FROM vciso_evidence WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_integrations", sql: `SELECT COUNT(*) FROM vciso_integrations WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_approvals", sql: `SELECT COUNT(*) FROM vciso_approvals WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_benchmarks", sql: `SELECT COUNT(*) FROM vciso_benchmarks WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_control_dependencies", sql: `SELECT COUNT(*) FROM vciso_control_dependencies WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_messages", sql: `SELECT COUNT(*) FROM vciso_messages WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "cyber_db", label: "cyber.vciso_predictions", sql: `SELECT COUNT(*) FROM vciso_predictions WHERE tenant_id = $1`, args: []any{mainTenantID}},
		{db: "acta_db", label: "acta.committees", sql: `SELECT COUNT(*) FROM committees WHERE tenant_id = $1`, args: []any{actaTenantID}},
		{db: "acta_db", label: "acta.compliance_checks", sql: `SELECT COUNT(*) FROM compliance_checks WHERE tenant_id = $1`, args: []any{actaTenantID}},
		{db: "lex_db", label: "lex.contracts", sql: `SELECT COUNT(*) FROM contracts WHERE tenant_id = $1`, args: []any{lexTenantID}},
		{db: "lex_db", label: "lex.expiry_notifications", sql: `SELECT COUNT(*) FROM expiry_notifications WHERE tenant_id = $1`, args: []any{lexTenantID}},
		{db: "visus_db", label: "visus.dashboards", sql: `SELECT COUNT(*) FROM visus_dashboards WHERE tenant_id = $1`, args: []any{mainTenantID}},
	}

	for _, item := range queries {
		var count int64
		if err := s.pools[item.db].QueryRow(ctx, item.sql, item.args...).Scan(&count); err != nil {
			return fmt.Errorf("count %s: %w", item.label, err)
		}
		s.logger.Info().Str("metric", item.label).Int64("count", count).Msg("seed verification")
	}
	return nil
}

func (s *systemSeeder) buildDSN(dbName string) string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		s.cfg.Database.User,
		s.cfg.Database.Password,
		s.cfg.Database.Host,
		s.cfg.Database.Port,
		dbName,
		s.cfg.Database.SSLMode,
	)
}

func findMigrationsPath() string {
	candidates := []string{
		"migrations",
		"backend/migrations",
		"../migrations",
		filepath.Join("..", "..", "migrations"),
	}
	for _, candidate := range candidates {
		abs, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			return abs
		}
	}
	return "migrations"
}

func defaultRoles(tenantID uuid.UUID) []demoRole {
	return []demoRole{
		{ID: uuid.NewSHA1(tenantID, []byte("role:tenant-admin")), Name: "Tenant Admin", Slug: "tenant-admin", Description: "Full tenant administration access.", Permissions: []string{"tenant:*", "users:*", "roles:*", "apikeys:*", "cyber:*", "data:*", "acta:*", "lex:*", "visus:*", "reports:*", "files:*", "workflows:*"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:security-manager")), Name: "Security Manager", Slug: "security-manager", Description: "Security operations management.", Permissions: []string{"cyber:*", "alerts:*", "remediation:*", "visus:read"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:data-steward")), Name: "Data Steward", Slug: "data-steward", Description: "Data governance and quality management.", Permissions: []string{"data:read", "data:write", "quality:*", "lineage:*"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:legal-manager")), Name: "Legal Manager", Slug: "legal-manager", Description: "Legal operations management.", Permissions: []string{"lex:*", "reports:read"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:board-secretary")), Name: "Board Secretary", Slug: "board-secretary", Description: "Board governance and meeting administration.", Permissions: []string{"acta:*", "files:*"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:executive")), Name: "Executive", Slug: "executive", Description: "Executive cross-suite visibility.", Permissions: []string{"visus:*", "reports:read", "acta:read", "lex:read", "cyber:read", "data:read"}},
		{ID: uuid.NewSHA1(tenantID, []byte("role:auditor")), Name: "Auditor", Slug: "auditor", Description: "Read-only audit and oversight access.", Permissions: []string{"*:read"}},
	}
}

func sha256Hex(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
