package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
	"github.com/rs/zerolog"

	aigovseeder "github.com/clario360/platform/internal/aigovernance/seeder"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	onboardingmodel "github.com/clario360/platform/internal/onboarding/model"
	"github.com/clario360/platform/internal/onboarding/service/seeder"
	"github.com/clario360/platform/internal/workflow/repository"
	"github.com/clario360/platform/pkg/storage"
)

type TenantProvisioner struct {
	platformPool           *pgxpool.Pool
	dbPools                map[string]*pgxpool.Pool
	dbDSNs                 map[string]string
	migrationsBasePath     string
	onboardingRepo         provisioningOnboardingRepository
	provisioningRepo       provisioningStatusRepository
	roleSeeder             *seeder.RoleSeeder
	settingsSeeder         *seeder.SettingsSeeder
	detectionRuleSeeder    *seeder.DetectionRuleSeeder
	kpiSeeder              *seeder.KPISeeder
	dashboardSeeder        *seeder.DashboardSeeder
	complianceRuleSeeder   *seeder.ComplianceRuleSeeder
	workflowTemplateSeeder *seeder.WorkflowTemplateSeeder
	modelSeeder            *aigovseeder.ModelSeeder
	storage                *storage.MinIOStorage
	emailSender            EmailSender
	producer               *events.Producer
	logger                 zerolog.Logger
	metrics                *Metrics
	pipeline               []provisioningPipelineStep
}

type provisioningPipelineStep struct {
	Name string
	Run  func(context.Context) error
}

func NewTenantProvisioner(
	platformPool *pgxpool.Pool,
	dbPools map[string]*pgxpool.Pool,
	dbDSNs map[string]string,
	migrationsBasePath string,
	onboardingRepo provisioningOnboardingRepository,
	provisioningRepo provisioningStatusRepository,
	storageClient *storage.MinIOStorage,
	emailSender EmailSender,
	producer *events.Producer,
	logger zerolog.Logger,
	metrics *Metrics,
) *TenantProvisioner {
	return &TenantProvisioner{
		platformPool:       platformPool,
		dbPools:            dbPools,
		dbDSNs:             dbDSNs,
		migrationsBasePath: migrationsBasePath,
		onboardingRepo:     onboardingRepo,
		provisioningRepo:   provisioningRepo,
		roleSeeder:         seeder.NewRoleSeeder(platformPool),
		settingsSeeder:     seeder.NewSettingsSeeder(platformPool),
		detectionRuleSeeder: func() *seeder.DetectionRuleSeeder {
			if dbPools["cyber_db"] == nil {
				return nil
			}
			return seeder.NewDetectionRuleSeeder(dbPools["cyber_db"])
		}(),
		kpiSeeder: func() *seeder.KPISeeder {
			if dbPools["visus_db"] == nil {
				return nil
			}
			return seeder.NewKPISeeder(dbPools["visus_db"])
		}(),
		dashboardSeeder: func() *seeder.DashboardSeeder {
			if dbPools["visus_db"] == nil {
				return nil
			}
			return seeder.NewDashboardSeeder(dbPools["visus_db"], logger)
		}(),
		complianceRuleSeeder: func() *seeder.ComplianceRuleSeeder {
			if dbPools["lex_db"] == nil {
				return nil
			}
			return seeder.NewComplianceRuleSeeder(dbPools["lex_db"])
		}(),
		workflowTemplateSeeder: seeder.NewWorkflowTemplateSeeder(repository.NewDefinitionRepository(platformPool), logger),
		modelSeeder:            aigovseeder.NewModelSeeder(platformPool, logger),
		storage:                storageClient,
		emailSender:            emailSender,
		producer:               producer,
		logger:                 logger.With().Str("service", "tenant_provisioner").Logger(),
		metrics:                metrics,
	}
}

func (p *TenantProvisioner) Provision(ctx context.Context, tenantID uuid.UUID) error {
	onboardingRow, err := p.onboardingRepo.GetOnboardingByTenantID(ctx, tenantID)
	if err != nil {
		return fmt.Errorf("load onboarding for provisioning: %w", err)
	}
	startedAt := time.Now()
	if p.metrics != nil && p.metrics.provisioningTotal != nil {
		p.metrics.provisioningTotal.WithLabelValues("started").Inc()
	}
	publishOnboardingEvent(ctx, p.producer,
		"com.clario360.onboarding.provisioning.started",
		tenantID,
		&onboardingRow.AdminUserID,
		map[string]any{"tenant_id": tenantID.String()},
		p.logger,
	)

	steps, err := p.pipelineSteps(ctx, tenantID, onboardingRow, startedAt)
	if err != nil {
		return err
	}
	stepNames := make([]string, 0, len(steps))
	for idx := range steps {
		stepNames = append(stepNames, steps[idx].Name)
	}
	if err := p.provisioningRepo.Initialize(ctx, tenantID, onboardingRow.ID, stepNames); err != nil {
		return fmt.Errorf("initialize provisioning status: %w", err)
	}

	currentSteps, err := p.provisioningRepo.ListSteps(ctx, tenantID)
	if err != nil {
		return err
	}
	statusByNumber := map[int]onboardingmodel.ProvisioningStepStatus{}
	for _, step := range currentSteps {
		statusByNumber[step.StepNumber] = step.Status
	}

	for idx, step := range steps {
		stepNumber := idx + 1
		stepName := step.Name
		if statusByNumber[stepNumber] == onboardingmodel.ProvisioningStepCompleted {
			continue
		}
		stepStarted := time.Now()
		if err := p.provisioningRepo.StartStep(ctx, tenantID, stepNumber); err != nil {
			return err
		}
		if err := step.Run(ctx); err != nil {
			message := err.Error()
			_ = p.provisioningRepo.FailStep(ctx, tenantID, stepNumber, message, map[string]any{"step_name": stepName})
			_ = p.provisioningRepo.MarkFailed(ctx, tenantID, message)
			publishOnboardingEvent(ctx, p.producer,
				"com.clario360.onboarding.provisioning.step_failed",
				tenantID,
				&onboardingRow.AdminUserID,
				map[string]any{
					"tenant_id":   tenantID.String(),
					"step_number": stepNumber,
					"step_name":   stepName,
					"error":       message,
				},
				p.logger,
			)
			if p.metrics != nil {
				if p.metrics.provisioningTotal != nil {
					p.metrics.provisioningTotal.WithLabelValues("failed").Inc()
				}
				if p.metrics.provisioningStepDuration != nil {
					p.metrics.provisioningStepDuration.WithLabelValues(stepName).Observe(time.Since(stepStarted).Seconds())
				}
			}
			return fmt.Errorf("step %d (%s) failed: %w", stepNumber, stepName, err)
		}
		_ = p.provisioningRepo.CompleteStep(ctx, tenantID, stepNumber, map[string]any{"step_name": stepName})
		publishOnboardingEvent(ctx, p.producer,
			"com.clario360.onboarding.provisioning.step_completed",
			tenantID,
			&onboardingRow.AdminUserID,
			map[string]any{
				"tenant_id":   tenantID.String(),
				"step_number": stepNumber,
				"step_name":   stepName,
			},
			p.logger,
		)
		if p.metrics != nil && p.metrics.provisioningStepDuration != nil {
			p.metrics.provisioningStepDuration.WithLabelValues(stepName).Observe(time.Since(stepStarted).Seconds())
		}
	}

	if err := p.provisioningRepo.SetTenantStatus(ctx, tenantID, "active"); err != nil {
		return err
	}
	if err := p.provisioningRepo.MarkCompleted(ctx, tenantID); err != nil {
		return err
	}
	if p.metrics != nil {
		if p.metrics.provisioningTotal != nil {
			p.metrics.provisioningTotal.WithLabelValues("completed").Inc()
		}
		if p.metrics.provisioningDuration != nil {
			p.metrics.provisioningDuration.WithLabelValues("completed").Observe(time.Since(startedAt).Seconds())
		}
		if p.metrics.timeToActive != nil {
			p.metrics.timeToActive.WithLabelValues(tenantID.String()).Observe(time.Since(onboardingRow.CreatedAt).Seconds())
		}
	}
	publishOnboardingEvent(ctx, p.producer,
		"com.clario360.onboarding.provisioning.completed",
		tenantID,
		&onboardingRow.AdminUserID,
		map[string]any{
			"tenant_id":   tenantID.String(),
			"duration_ms": time.Since(startedAt).Milliseconds(),
			"steps_count": len(steps),
		},
		p.logger,
	)
	return nil
}

func (p *TenantProvisioner) pipelineSteps(ctx context.Context, tenantID uuid.UUID, onboardingRow *onboardingmodel.OnboardingStatus, startedAt time.Time) ([]provisioningPipelineStep, error) {
	if len(p.pipeline) > 0 {
		return p.pipeline, nil
	}

	name, slug, _, _, err := p.onboardingRepo.GetTenantIdentity(ctx, tenantID)
	if err != nil {
		return nil, err
	}

	return []provisioningPipelineStep{
		{Name: "Verify Database Connectivity", Run: p.verifyDatabaseConnectivity},
		{Name: "Verify Migrations", Run: p.verifyMigrations},
		{Name: "Seed System Roles", Run: func(stepCtx context.Context) error { return p.roleSeeder.Seed(stepCtx, tenantID) }},
		{Name: "Seed Default Settings", Run: func(stepCtx context.Context) error {
			return p.settingsSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Seed Detection Rules", Run: func(stepCtx context.Context) error {
			if p.detectionRuleSeeder == nil {
				return nil
			}
			return p.detectionRuleSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Seed Default KPIs", Run: func(stepCtx context.Context) error {
			if p.kpiSeeder == nil {
				return nil
			}
			return p.kpiSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Seed Default Dashboard", Run: func(stepCtx context.Context) error {
			if p.dashboardSeeder == nil {
				return nil
			}
			return p.dashboardSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Seed Compliance Rules", Run: func(stepCtx context.Context) error {
			if p.complianceRuleSeeder == nil {
				return nil
			}
			return p.complianceRuleSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Seed AI Governance Models", Run: func(stepCtx context.Context) error {
			if p.modelSeeder == nil {
				return nil
			}
			return p.modelSeeder.Seed(stepCtx, tenantID, onboardingRow.AdminUserID)
		}},
		{Name: "Create Storage Buckets", Run: func(stepCtx context.Context) error { return p.createStorageBuckets(stepCtx, slug) }},
		{Name: "Initialize Audit Trail", Run: func(stepCtx context.Context) error {
			return p.initializeAuditTrail(stepCtx, onboardingRow, name, slug, startedAt)
		}},
	}, nil
}

func (p *TenantProvisioner) verifyDatabaseConnectivity(ctx context.Context) error {
	for name := range p.dbDSNs {
		var pool *pgxpool.Pool
		if name == "platform_core" {
			pool = p.platformPool
		} else {
			pool = p.dbPools[name]
		}
		if pool == nil {
			return fmt.Errorf("%s connection pool is not configured", name)
		}
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("%s ping: %w", name, err)
		}
	}
	return nil
}

func (p *TenantProvisioner) verifyMigrations(ctx context.Context) error {
	for dbName, dsn := range p.dbDSNs {
		migrationsDir := filepath.Join(p.migrationsBasePath, dbName)
		expectedVersion, err := latestMigrationVersion(migrationsDir)
		if err != nil {
			return fmt.Errorf("%s migration discovery: %w", dbName, err)
		}
		version, dirty, err := database.MigrationVersion(dsn, migrationsDir)
		if err != nil {
			return fmt.Errorf("%s migration version: %w", dbName, err)
		}
		if dirty {
			return fmt.Errorf("%s migrations are dirty", dbName)
		}
		if expectedVersion != 0 && uint(expectedVersion) != version {
			return fmt.Errorf("%s migrations mismatch: expected %d, got %d", dbName, expectedVersion, version)
		}
	}
	return nil
}

func (p *TenantProvisioner) createStorageBuckets(ctx context.Context, slug string) error {
	if p.storage == nil {
		return nil
	}
	buckets := []string{
		"clario360-" + slug + "-cyber",
		"clario360-" + slug + "-data",
		"clario360-" + slug + "-acta",
		"clario360-" + slug + "-lex",
		"clario360-" + slug + "-visus",
		"clario360-" + slug + "-platform",
	}
	for _, bucket := range buckets {
		if err := p.storage.EnsureBucket(ctx, bucket); err != nil {
			return err
		}
		config := lifecycle.NewConfiguration()
		config.Rules = []lifecycle.Rule{
			{
				ID:     "noncurrent-delete-after-90-days",
				Status: "Enabled",
				NoncurrentVersionExpiration: lifecycle.NoncurrentVersionExpiration{
					NoncurrentDays: lifecycle.ExpirationDays(90),
				},
			},
		}
		if err := p.storage.Client().SetBucketLifecycle(ctx, bucket, config); err != nil {
			return fmt.Errorf("set lifecycle on %s: %w", bucket, err)
		}
	}
	return nil
}

func (p *TenantProvisioner) initializeAuditTrail(ctx context.Context, onboardingRow *onboardingmodel.OnboardingStatus, tenantName, slug string, startedAt time.Time) error {
	if err := repository.RunMigration(ctx, p.platformPool); err != nil {
		return fmt.Errorf("ensure workflow schema: %w", err)
	}
	if err := p.workflowTemplateSeeder.Seed(ctx, onboardingRow.TenantID.String(), onboardingRow.AdminUserID.String()); err != nil {
		return fmt.Errorf("seed workflow templates: %w", err)
	}

	var exists bool
	if err := p.platformPool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM audit_logs
			WHERE tenant_id = $1
			  AND action = 'tenant.provisioned'
			  AND resource_type = 'tenant'
			  AND resource_id = $1
		)`,
		onboardingRow.TenantID,
	).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		if _, err := p.platformPool.Exec(ctx, `
			INSERT INTO audit_logs (
				tenant_id, user_id, service, action, resource_type, resource_id, metadata
			) VALUES ($1, NULL, 'iam-service', 'tenant.provisioned', 'tenant', $1, $2::jsonb)`,
			onboardingRow.TenantID,
			marshalJSON(map[string]any{
				"tenant_id":                onboardingRow.TenantID.String(),
				"admin_email":              onboardingRow.AdminEmail,
				"suites_activated":         onboardingRow.ActiveSuites,
				"provisioning_duration_ms": time.Since(startedAt).Milliseconds(),
				"tenant_slug":              slug,
				"tenant_name":              tenantName,
			}),
		); err != nil {
			return err
		}
	}
	if _, err := p.platformPool.Exec(ctx, `
		INSERT INTO notifications (tenant_id, user_id, type, title, body, data)
		SELECT $1, $2, 'success', 'Welcome to Clario 360', 'Your platform is ready.',
		       $3::jsonb
		WHERE NOT EXISTS (
			SELECT 1
			FROM notifications
			WHERE tenant_id = $1
			  AND user_id = $2
			  AND title = 'Welcome to Clario 360'
		)`,
		onboardingRow.TenantID,
		onboardingRow.AdminUserID,
		marshalJSON(map[string]any{"path": "/dashboard", "tenant_name": tenantName}),
	); err != nil {
		return err
	}
	_ = p.emailSender.SendWelcomeEmail(ctx, onboardingRow.AdminEmail, tenantName, "Administrator")
	publishOnboardingEvent(ctx, p.producer,
		"com.clario360.platform.tenant.provisioned",
		onboardingRow.TenantID,
		&onboardingRow.AdminUserID,
		map[string]any{"tenant_id": onboardingRow.TenantID.String()},
		p.logger,
	)
	return nil
}

func latestMigrationVersion(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}
	versions := make([]int, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".up.sql") {
			continue
		}
		prefix := strings.SplitN(name, "_", 2)[0]
		value, convErr := strconv.Atoi(prefix)
		if convErr != nil {
			return 0, fmt.Errorf("parse migration version %q: %w", name, convErr)
		}
		versions = append(versions, value)
	}
	sort.Ints(versions)
	if len(versions) == 0 {
		return 0, nil
	}
	return versions[len(versions)-1], nil
}
