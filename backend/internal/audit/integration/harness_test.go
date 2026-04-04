//go:build integration

package integration

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	tc "github.com/testcontainers/testcontainers-go"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/clario360/platform/internal/audit/repository"
	"github.com/clario360/platform/internal/audit/service"
	"github.com/clario360/platform/internal/database"
)

var (
	sharedEnvOnce sync.Once
	sharedEnv     *auditIntegrationEnv
	sharedEnvErr  error
)

type auditIntegrationEnv struct {
	logger   zerolog.Logger
	postgres *postgresmod.PostgresContainer
	db       *pgxpool.Pool
}

type auditHarness struct {
	env          *auditIntegrationEnv
	repo         *repository.AuditRepository
	partitionMgr *repository.PartitionManager
	querySvc     *service.QueryService
	integritySvc *service.IntegrityService
	tenantID     string
}

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnv != nil {
		if err := sharedEnv.Close(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "audit integration cleanup failed: %v\n", err)
			if code == 0 {
				code = 1
			}
		}
	}
	os.Exit(code)
}

func newAuditHarness(t *testing.T, tenantID string) *auditHarness {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)

	env := mustSharedEnv(t)
	logger := env.logger

	repo := repository.NewAuditRepository(env.db, logger)
	partMgr := repository.NewPartitionManager(env.db, logger)
	masking := service.NewMaskingService()
	querySvc := service.NewQueryService(repo, masking, logger)
	integritySvc := service.NewIntegrityService(repo, logger)

	// Ensure partitions exist for current month
	_, err := partMgr.EnsurePartitions(context.Background())
	if err != nil {
		t.Fatalf("ensure partitions: %v", err)
	}

	h := &auditHarness{
		env:          env,
		repo:         repo,
		partitionMgr: partMgr,
		querySvc:     querySvc,
		integritySvc: integritySvc,
		tenantID:     tenantID,
	}
	t.Cleanup(func() {
		h.cleanupTenant(t)
	})
	return h
}

func mustSharedEnv(t *testing.T) *auditIntegrationEnv {
	t.Helper()
	sharedEnvOnce.Do(func() {
		sharedEnv, sharedEnvErr = startSharedEnv()
	})
	if sharedEnvErr != nil {
		t.Fatalf("start audit integration environment: %v", sharedEnvErr)
	}
	return sharedEnv
}

func startSharedEnv() (*auditIntegrationEnv, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	logger := zerolog.New(io.Discard)

	postgresContainer, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase("audit_service"),
		postgresmod.WithUsername("audit"),
		postgresmod.WithPassword("audit"),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dbURL := postgresContainer.MustConnectionString(ctx, "sslmode=disable")
	if err := database.RunMigrations(dbURL, auditMigrationsPath()); err != nil {
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("run audit migrations: %w", err)
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("open audit postgres pool: %w", err)
	}
	if err := db.Ping(ctx); err != nil {
		db.Close()
		_ = postgresContainer.Terminate(context.Background())
		return nil, fmt.Errorf("ping audit postgres: %w", err)
	}

	return &auditIntegrationEnv{
		logger:   logger,
		postgres: postgresContainer,
		db:       db,
	}, nil
}

func (e *auditIntegrationEnv) Close() error {
	closeCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if e.db != nil {
		e.db.Close()
	}
	if e.postgres != nil {
		if err := e.postgres.Terminate(closeCtx); err != nil {
			return err
		}
	}
	return nil
}

func (h *auditHarness) cleanupTenant(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Delete from all known partitions for this tenant.
	// The immutability trigger prevents DELETE, so we must disable it temporarily.
	_, _ = h.env.db.Exec(ctx, `ALTER TABLE audit_logs DISABLE TRIGGER audit_immutability_guard`)
	_, _ = h.env.db.Exec(ctx, `DELETE FROM audit_chain_state WHERE tenant_id = $1`, h.tenantID)
	_, _ = h.env.db.Exec(ctx, `DELETE FROM audit_logs WHERE tenant_id = $1`, h.tenantID)
	_, _ = h.env.db.Exec(ctx, `ALTER TABLE audit_logs ENABLE TRIGGER audit_immutability_guard`)
}

func auditMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "migrations", "audit_db")
}
