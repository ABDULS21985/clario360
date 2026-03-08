//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
	tc "github.com/testcontainers/testcontainers-go"
	postgresmod "github.com/testcontainers/testcontainers-go/modules/postgres"
	redpandamod "github.com/testcontainers/testcontainers-go/modules/redpanda"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/clario360/platform/internal/auth"
	appconfig "github.com/clario360/platform/internal/config"
	cyberconsumer "github.com/clario360/platform/internal/cyber/consumer"
	cyberdto "github.com/clario360/platform/internal/cyber/dto"
	cybermodel "github.com/clario360/platform/internal/cyber/model"
	cyberrepo "github.com/clario360/platform/internal/cyber/repository"
	cyberservice "github.com/clario360/platform/internal/cyber/service"
	dataconsumer "github.com/clario360/platform/internal/data/consumer"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	notifchannel "github.com/clario360/platform/internal/notification/channel"
	notifconsumer "github.com/clario360/platform/internal/notification/consumer"
	notifdto "github.com/clario360/platform/internal/notification/dto"
	notifmodel "github.com/clario360/platform/internal/notification/model"
	notifrepo "github.com/clario360/platform/internal/notification/repository"
	notifservice "github.com/clario360/platform/internal/notification/service"
	visusapp "github.com/clario360/platform/internal/visus"
	visusconfig "github.com/clario360/platform/internal/visus/config"
	visusmodel "github.com/clario360/platform/internal/visus/model"
	visusrepo "github.com/clario360/platform/internal/visus/repository"
)

var (
	sharedCrossSuiteEnv     *crossSuiteEnv
	sharedCrossSuiteEnvErr  error
	sharedCrossSuiteEnvOnce sync.Once
)

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedCrossSuiteEnv != nil {
		sharedCrossSuiteEnv.Close()
	}
	os.Exit(code)
}

type noopChannel struct {
	name string
}

func (c noopChannel) Name() string {
	return c.name
}

func (c noopChannel) Send(_ context.Context, _ *notifmodel.Notification) *notifchannel.ChannelResult {
	return &notifchannel.ChannelResult{Success: true}
}

type crossSuiteEnv struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger zerolog.Logger

	redpanda tc.Container
	redisTC  tc.Container
	cyberPG  tc.Container
	notifPG  tc.Container
	visusPG  tc.Container

	cyberDB *pgxpool.Pool
	notifDB *pgxpool.Pool
	visusDB *pgxpool.Pool
	rdb     *redis.Client

	kafkaBrokers []string
	producer     *events.Producer
	consumers    []*events.Consumer

	cyberAlerts *cyberrepo.AlertRepository
	notifRepo   *notifrepo.NotificationRepository
	visusApp    *visusapp.Application

	iamServer *httptest.Server
	mu        sync.RWMutex
	roleUsers map[string][]string
	emails    map[string]string
}

func TestBruteForceIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()
	env.seedVisusTenant(t, tenantID)

	for idx := 0; idx < 5; idx++ {
		event, err := events.NewEvent("iam.user.login.failed", "iam-service", tenantID.String(), map[string]any{
			"user_id":       "user-1",
			"email":         "user@example.com",
			"ip_address":    "203.0.113.10",
			"attempt_count": idx + 1,
			"user_agent":    "integration-test",
			"timestamp":     time.Now().UTC(),
		})
		if err != nil {
			t.Fatalf("create login event: %v", err)
		}
		event.ID = fmt.Sprintf("bf-%s-%d", tenantID.String(), idx)
		if err := env.producer.Publish(context.Background(), events.Topics.IAMEvents, event); err != nil {
			t.Fatalf("publish login event: %v", err)
		}
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		alerts, err := env.listCyberAlerts(tenantID)
		if err != nil {
			return false, err
		}
		for _, alert := range alerts {
			if strings.Contains(alert.Title, "Brute Force Attack Detected") {
				return true, nil
			}
		}
		return false, nil
	})
}

func TestPipelineEscalationIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()
	env.seedVisusTenant(t, tenantID)

	for idx := 0; idx < 3; idx++ {
		event, err := events.NewEvent("data.pipeline.run.failed", "data-service", tenantID.String(), map[string]any{
			"id":            uuid.NewString(),
			"pipeline_id":   "pipeline-escalation",
			"pipeline_name": "Nightly Warehouse Sync",
			"tenant_id":     tenantID.String(),
			"status":        "failed",
			"error_message": "warehouse unavailable",
		})
		if err != nil {
			t.Fatalf("create pipeline event: %v", err)
		}
		event.ID = fmt.Sprintf("pipe-fail-%s-%d", tenantID.String(), idx)
		if err := env.producer.Publish(context.Background(), events.Topics.PipelineEvents, event); err != nil {
			t.Fatalf("publish pipeline event: %v", err)
		}
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		alerts, err := env.listVisusAlerts(tenantID)
		if err != nil {
			return false, err
		}
		for _, alert := range alerts {
			if strings.Contains(alert.Title, "Pipeline Reliability Issue") && alert.SourceSuite == "data" {
				return true, nil
			}
		}
		return false, nil
	})
}

func TestContractExpiryChainIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()
	env.seedVisusTenant(t, tenantID)
	env.setRoleUsers(tenantID.String(), "legal-manager", []string{"legal-manager-1"})
	env.setEmail("contract-owner-1", "owner@integration.test")
	env.setEmail("legal-manager-1", "manager@integration.test")

	event, err := events.NewEvent("lex.contract.expiring", "lex-service", tenantID.String(), map[string]any{
		"id":                uuid.NewString(),
		"title":             "Third-Party Services Agreement",
		"party_name":        "Example Vendor",
		"owner_user_id":     "contract-owner-1",
		"days_until_expiry": 7,
		"severity":          "critical",
	})
	if err != nil {
		t.Fatalf("create contract expiry event: %v", err)
	}
	event.ID = "contract-expiry-" + tenantID.String()
	if err := env.producer.Publish(context.Background(), events.Topics.LexEvents, event); err != nil {
		t.Fatalf("publish contract expiry event: %v", err)
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		ownerCount, err := env.notificationCount(tenantID.String(), "contract-owner-1")
		if err != nil {
			return false, err
		}
		managerCount, err := env.notificationCount(tenantID.String(), "legal-manager-1")
		if err != nil {
			return false, err
		}
		alerts, err := env.listVisusAlerts(tenantID)
		if err != nil {
			return false, err
		}
		hasVisusAlert := false
		for _, alert := range alerts {
			if strings.Contains(alert.Title, "Contract Expiring") && alert.SourceSuite == "lex" {
				hasVisusAlert = true
				break
			}
		}
		return ownerCount >= 1 && managerCount >= 1 && hasVisusAlert, nil
	})
}

func TestMalwareChainIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()
	env.seedVisusTenant(t, tenantID)
	env.setRoleUsers(tenantID.String(), "tenant-admin", []string{"tenant-admin-1"})
	env.setEmail("uploader-1", "uploader@integration.test")
	env.setEmail("tenant-admin-1", "admin@integration.test")

	event, err := events.NewEvent("file.scan.infected", "file-service", tenantID.String(), map[string]any{
		"file_id":      "malware-file-1",
		"virus_name":   "EICAR-Test-File",
		"uploaded_by":  "uploader-1",
		"suite":        "lex",
		"content_type": "application/pdf",
	})
	if err != nil {
		t.Fatalf("create malware event: %v", err)
	}
	event.ID = "malware-" + tenantID.String()
	if err := env.producer.Publish(context.Background(), events.Topics.FileEvents, event); err != nil {
		t.Fatalf("publish malware event: %v", err)
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		cyberAlerts, err := env.listCyberAlerts(tenantID)
		if err != nil {
			return false, err
		}
		hasCyberAlert := false
		for _, alert := range cyberAlerts {
			if strings.Contains(alert.Title, "Malware Detected in Uploaded File") {
				hasCyberAlert = true
				break
			}
		}

		uploaderCount, err := env.notificationCount(tenantID.String(), "uploader-1")
		if err != nil {
			return false, err
		}
		adminCount, err := env.notificationCount(tenantID.String(), "tenant-admin-1")
		if err != nil {
			return false, err
		}

		visusAlerts, err := env.listVisusAlerts(tenantID)
		if err != nil {
			return false, err
		}
		hasVisusAlert := false
		for _, alert := range visusAlerts {
			if strings.Contains(alert.Title, "Malware Detected in Uploaded File") && alert.SourceSuite == "file" {
				hasVisusAlert = true
				break
			}
		}
		return hasCyberAlert && uploaderCount >= 1 && adminCount >= 1 && hasVisusAlert, nil
	})
}

func TestQualityKPIUpdateIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()
	env.seedVisusTenant(t, tenantID)

	event, err := events.NewEvent("data.quality.score_changed", "data-service", tenantID.String(), map[string]any{
		"old_score": 92.0,
		"new_score": 84.0,
	})
	if err != nil {
		t.Fatalf("create quality event: %v", err)
	}
	event.ID = "quality-" + tenantID.String()
	if err := env.producer.Publish(context.Background(), events.Topics.QualityEvents, event); err != nil {
		t.Fatalf("publish quality event: %v", err)
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		value, err := env.latestKPIValue(tenantID, "Data Quality Score")
		if err != nil {
			return false, err
		}
		return value == 84.0, nil
	})
}

func TestIdempotencyIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	tenantID := uuid.New()

	event, err := events.NewEvent("file.scan.infected", "file-service", tenantID.String(), map[string]any{
		"file_id":      "dup-file-1",
		"virus_name":   "EICAR-Test-File",
		"uploaded_by":  "dup-user-1",
		"suite":        "data",
		"content_type": "text/plain",
	})
	if err != nil {
		t.Fatalf("create idempotency event: %v", err)
	}
	event.ID = "idempotent-event-" + tenantID.String()
	if err := env.producer.Publish(context.Background(), events.Topics.FileEvents, event); err != nil {
		t.Fatalf("publish first event: %v", err)
	}
	if err := env.producer.Publish(context.Background(), events.Topics.FileEvents, event); err != nil {
		t.Fatalf("publish duplicate event: %v", err)
	}

	waitFor(t, 20*time.Second, func() (bool, error) {
		alerts, err := env.listCyberAlerts(tenantID)
		if err != nil {
			return false, err
		}
		matches := 0
		for _, alert := range alerts {
			if strings.Contains(alert.Title, "dup-file-1") {
				matches++
			}
		}
		return matches == 1, nil
	})
}

func TestDeadLetterIntegration(t *testing.T) {
	env := crossSuiteEnvForTest(t)
	topic := "integration.deadletter.events"
	dlqTopic := topic + ".dlq"
	dlqEvents := make(chan *events.Event, 1)

	dlqConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         env.brokers(),
		GroupID:         "crosssuite-dlq-reader-" + uuid.NewString(),
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		t.Fatalf("create dlq reader: %v", err)
	}
	defer dlqConsumer.Close()
	dlqConsumer.Subscribe(dlqTopic, events.EventHandlerFunc(func(ctx context.Context, event *events.Event) error {
		select {
		case dlqEvents <- event:
		default:
		}
		return nil
	}))
	go func() { _ = dlqConsumer.Start(env.ctx) }()

	failingConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         env.brokers(),
		GroupID:         "crosssuite-dlq-failing-" + uuid.NewString(),
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		t.Fatalf("create failing consumer: %v", err)
	}
	defer failingConsumer.Close()
	failingConsumer.SetDeadLetterProducer(env.producer)
	failingConsumer.Subscribe(topic, events.EventHandlerFunc(func(ctx context.Context, event *events.Event) error {
		return errors.New("forced failure for DLQ integration test")
	}))
	go func() { _ = failingConsumer.Start(env.ctx) }()

	event, err := events.NewEvent("integration.deadletter.failed", "integration-test", uuid.NewString(), map[string]any{"value": "boom"})
	if err != nil {
		t.Fatalf("create dlq source event: %v", err)
	}
	event.ID = "dlq-source-" + uuid.NewString()
	if err := env.producer.Publish(context.Background(), topic, event); err != nil {
		t.Fatalf("publish dlq source event: %v", err)
	}

	select {
	case dlqEvent := <-dlqEvents:
		if dlqEvent == nil {
			t.Fatal("expected DLQ event, got nil")
		}
	case <-time.After(20 * time.Second):
		t.Fatal("timed out waiting for DLQ event")
	}
}

func crossSuiteEnvForTest(t *testing.T) *crossSuiteEnv {
	t.Helper()
	testcontainersSkipIfUnavailable(t)

	sharedCrossSuiteEnvOnce.Do(func() {
		sharedCrossSuiteEnv, sharedCrossSuiteEnvErr = startCrossSuiteEnv()
	})
	if sharedCrossSuiteEnvErr != nil {
		t.Fatalf("start cross-suite env: %v", sharedCrossSuiteEnvErr)
	}
	return sharedCrossSuiteEnv
}

func startCrossSuiteEnv() (*crossSuiteEnv, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()

	env := &crossSuiteEnv{
		logger:    logger,
		roleUsers: make(map[string][]string),
		emails:    make(map[string]string),
	}

	redisTC, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			Cmd:          []string{"redis-server", "--save", "", "--appendonly", "no", "--bind", "0.0.0.0"},
			WaitingFor:   wait.ForLog("Ready to accept connections"),
		},
		Started: true,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start redis: %w", err)
	}
	env.redisTC = redisTC

	redpandaTC, err := redpandamod.Run(ctx, "docker.redpanda.com/redpandadata/redpanda:v24.1.8", redpandamod.WithAutoCreateTopics())
	if err != nil {
		cancel()
		return nil, fmt.Errorf("start redpanda: %w", err)
	}
	env.redpanda = redpandaTC

	redisHost, err := redisTC.Host(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("redis host: %w", err)
	}
	redisPort, err := redisTC.MappedPort(ctx, "6379/tcp")
	if err != nil {
		cancel()
		return nil, fmt.Errorf("redis port: %w", err)
	}
	env.rdb = redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", redisHost, redisPort.Port())})
	if err := env.rdb.Ping(ctx).Err(); err != nil {
		cancel()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	broker, err := redpandaTC.KafkaSeedBroker(ctx)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("resolve kafka broker: %w", err)
	}

	env.cyberPG, env.cyberDB = startSuiteDB(ctx, cyberMigrationsPath(), "cyber_integration", "cyber", "cyber")
	env.notifPG, env.notifDB = startSuiteDB(ctx, "", "notification_integration", "notification", "notification")
	env.visusPG, env.visusDB = startSuiteDB(ctx, visusMigrationsPath(), "visus_integration", "visus", "visus")

	if err := notifrepo.RunMigration(ctx, env.notifDB); err != nil {
		cancel()
		return nil, fmt.Errorf("run notification migrations: %w", err)
	}

	env.ctx, env.cancel = context.WithCancel(context.Background())

	env.producer, err = events.NewProducer(appconfig.KafkaConfig{
		Brokers: []string{broker},
		GroupID: "crosssuite-producer",
	}, env.logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create producer: %w", err)
	}
	env.kafkaBrokers = []string{broker}

	jwtMgr, err := auth.NewJWTManager(appconfig.AuthConfig{
		JWTIssuer:       "crosssuite-integration",
		AccessTokenTTL:  time.Hour,
		RefreshTokenTTL: 24 * time.Hour,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create jwt manager: %w", err)
	}

	env.iamServer = httptest.NewServer(http.HandlerFunc(env.handleIAM))

	visusCfg := visusconfig.Default()
	visusCfg.SuiteCyberURL = "http://127.0.0.1:1"
	visusCfg.SuiteDataURL = "http://127.0.0.1:1"
	visusCfg.SuiteActaURL = "http://127.0.0.1:1"
	visusCfg.SuiteLexURL = "http://127.0.0.1:1"
	visusCfg.SuiteTimeout = 200 * time.Millisecond
	visusCfg.SuiteMaxRetries = 1
	visusCfg.CircuitThreshold = 1
	visusCfg.CircuitReset = time.Second
	visusCfg.ServiceTokenTTL = time.Minute
	env.visusApp, err = visusapp.NewApplication(visusapp.Dependencies{
		DB:         env.visusDB,
		Redis:      env.rdb,
		Publisher:  env.producer,
		Logger:     env.logger,
		Registerer: prometheus.NewRegistry(),
		Config:     visusCfg,
		JWTManager: jwtMgr,
	})
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create visus app: %w", err)
	}

	cyberAlertRepo := cyberrepo.NewAlertRepository(env.cyberDB, env.logger)
	env.cyberAlerts = cyberAlertRepo
	cyberCommentRepo := cyberrepo.NewCommentRepository(env.cyberDB, env.logger)
	cyberAlertSvc := cyberservice.NewAlertService(cyberAlertRepo, cyberCommentRepo, env.cyberDB, env.producer, env.logger)

	cyberGuard := events.NewIdempotencyGuard(env.rdb, 24*time.Hour)
	cyberConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         []string{broker},
		GroupID:         "crosssuite-cyber-consumer",
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create cyber consumer: %w", err)
	}
	cyberConsumer.Subscribe(events.Topics.IAMEvents, cyberconsumer.NewIAMEventConsumer(cyberAlertSvc, env.rdb, cyberGuard, env.producer, env.logger, nil))
	cyberConsumer.Subscribe(events.Topics.FileEvents, cyberconsumer.NewFileEventConsumer(cyberAlertSvc, cyberGuard, env.producer, env.logger, nil))
	env.consumers = append(env.consumers, cyberConsumer)
	go func() { _ = cyberConsumer.Start(env.ctx) }()

	visusConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         []string{broker},
		GroupID:         "crosssuite-visus-consumer",
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create visus consumer: %w", err)
	}
	env.visusApp.Consumer.Register(visusConsumer)
	env.consumers = append(env.consumers, visusConsumer)
	go func() { _ = visusConsumer.Start(env.ctx) }()

	failureTrackerConsumer, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         []string{broker},
		GroupID:         "crosssuite-data-consumer",
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create failure tracker consumer: %w", err)
	}
	failureTrackerConsumer.Subscribe(events.Topics.PipelineEvents, dataconsumer.NewFailureTracker(env.rdb, events.NewIdempotencyGuard(env.rdb, 24*time.Hour), env.producer, env.logger, nil))
	env.consumers = append(env.consumers, failureTrackerConsumer)
	go func() { _ = failureTrackerConsumer.Start(env.ctx) }()

	env.notifRepo = notifrepo.NewNotificationRepository(env.notifDB, env.logger)
	prefRepo := notifrepo.NewPreferenceRepository(env.notifDB, env.logger)
	deliveryRepo := notifrepo.NewDeliveryRepository(env.notifDB, env.logger)
	tmplSvc := notifservice.NewTemplateService(env.logger)
	prefSvc := notifservice.NewPreferenceService(prefRepo, env.rdb, env.logger)
	dispatcher := notifservice.NewDispatcherService(map[string]notifchannel.Channel{
		"email":  noopChannel{name: "email"},
		"in_app": noopChannel{name: "in_app"},
		"push":   noopChannel{name: "push"},
	}, deliveryRepo, env.logger)
	notifSvc := notifservice.NewNotificationService(env.notifRepo, prefSvc, dispatcher, tmplSvc, env.producer, env.rdb, env.logger)
	notifConsumerClient, err := events.NewConsumer(appconfig.KafkaConfig{
		Brokers:         []string{broker},
		GroupID:         "crosssuite-notification-consumer",
		AutoOffsetReset: "earliest",
	}, env.logger)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("create notification consumer: %w", err)
	}
	notifResolver := notifconsumer.NewRecipientResolver(env.iamServer.URL, "", "", "", env.logger)
	notifConsumer := notifconsumer.NewNotificationConsumer(notifConsumerClient, notifSvc, notifResolver, events.NewIdempotencyGuard(env.rdb, 24*time.Hour), nil, env.logger)
	env.consumers = append(env.consumers, notifConsumerClient)
	go func() { _ = notifConsumer.Start(env.ctx) }()

	time.Sleep(2 * time.Second)
	return env, nil
}

func (e *crossSuiteEnv) Close() {
	if e.cancel != nil {
		e.cancel()
	}
	closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	for _, consumer := range e.consumers {
		_ = consumer.Close()
	}
	if e.producer != nil {
		_ = e.producer.Close()
	}
	if e.iamServer != nil {
		e.iamServer.Close()
	}
	if e.rdb != nil {
		_ = e.rdb.Close()
	}
	if e.cyberDB != nil {
		e.cyberDB.Close()
	}
	if e.notifDB != nil {
		e.notifDB.Close()
	}
	if e.visusDB != nil {
		e.visusDB.Close()
	}
	if e.redpanda != nil {
		_ = e.redpanda.Terminate(closeCtx)
	}
	if e.redisTC != nil {
		_ = e.redisTC.Terminate(closeCtx)
	}
	if e.cyberPG != nil {
		_ = e.cyberPG.Terminate(closeCtx)
	}
	if e.notifPG != nil {
		_ = e.notifPG.Terminate(closeCtx)
	}
	if e.visusPG != nil {
		_ = e.visusPG.Terminate(closeCtx)
	}
}

func (e *crossSuiteEnv) brokers() []string {
	return append([]string(nil), e.kafkaBrokers...)
}

func (e *crossSuiteEnv) seedVisusTenant(t *testing.T, tenantID uuid.UUID) {
	t.Helper()
	cfg := visusconfig.Default()
	cfg.DemoTenantID = tenantID.String()
	cfg.DemoUserID = uuid.NewString()
	if _, err := visusapp.SeedDemoData(context.Background(), e.visusApp, cfg, e.logger); err != nil {
		t.Fatalf("seed visus tenant %s: %v", tenantID, err)
	}
}

func (e *crossSuiteEnv) setRoleUsers(tenantID, role string, userIDs []string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.roleUsers[tenantID+":"+role] = append([]string(nil), userIDs...)
}

func (e *crossSuiteEnv) setEmail(userID, email string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.emails[userID] = email
}

func (e *crossSuiteEnv) handleIAM(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/internal/users/by-role":
		tenantID := r.URL.Query().Get("tenant_id")
		role := r.URL.Query().Get("role")
		e.mu.RLock()
		userIDs := append([]string(nil), e.roleUsers[tenantID+":"+role]...)
		e.mu.RUnlock()
		writeJSON(w, http.StatusOK, map[string][]string{"user_ids": userIDs})
	case strings.HasPrefix(r.URL.Path, "/api/v1/internal/users/") && strings.HasSuffix(r.URL.Path, "/email"):
		userID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/v1/internal/users/"), "/email")
		e.mu.RLock()
		email := e.emails[userID]
		e.mu.RUnlock()
		if email == "" {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"email": email})
	default:
		http.NotFound(w, r)
	}
}

func (e *crossSuiteEnv) listCyberAlerts(tenantID uuid.UUID) ([]*cybermodel.Alert, error) {
	params := &cyberdto.AlertListParams{Page: 1, PerPage: 100}
	params.SetDefaults()
	items, _, err := e.cyberAlerts.List(context.Background(), tenantID, params)
	return items, err
}

func (e *crossSuiteEnv) listVisusAlerts(tenantID uuid.UUID) ([]visusmodel.ExecutiveAlert, error) {
	items, _, err := e.visusApp.Store.Alerts.List(context.Background(), tenantID, visusrepo.AlertListFilters{}, 1, 100)
	return items, err
}

func (e *crossSuiteEnv) notificationCount(tenantID, userID string) (int, error) {
	items, _, err := e.notifRepo.Query(context.Background(), &notifdto.QueryParams{
		TenantID: tenantID,
		UserID:   userID,
		Page:     1,
		PerPage:  100,
		Sort:     "created_at",
		Order:    "desc",
	})
	if err != nil {
		return 0, err
	}
	return len(items), nil
}

func (e *crossSuiteEnv) latestKPIValue(tenantID uuid.UUID, name string) (float64, error) {
	defs, _, err := e.visusApp.Store.KPIs.List(context.Background(), tenantID, 1, 100)
	if err != nil {
		return 0, err
	}
	for _, def := range defs {
		if def.Name != name {
			continue
		}
		snapshot, err := e.visusApp.Store.KPISnapshots.LatestByKPI(context.Background(), tenantID, def.ID)
		if err != nil {
			return 0, err
		}
		return snapshot.Value, nil
	}
	return 0, fmt.Errorf("kpi %q not found", name)
}

func waitFor(t *testing.T, timeout time.Duration, check func() (bool, error)) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		ok, err := check()
		if ok {
			return
		}
		if err != nil {
			lastErr = err
		}
		time.Sleep(300 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("condition not met before timeout: %v", lastErr)
	}
	t.Fatal("condition not met before timeout")
}

func startSuiteDB(ctx context.Context, migrationsPath, databaseName, username, password string) (tc.Container, *pgxpool.Pool) {
	container, err := postgresmod.Run(ctx, "postgres:16-alpine",
		postgresmod.WithDatabase(databaseName),
		postgresmod.WithUsername(username),
		postgresmod.WithPassword(password),
		postgresmod.BasicWaitStrategies(),
	)
	if err != nil {
		panic(fmt.Sprintf("start postgres %s: %v", databaseName, err))
	}

	dbURL := container.MustConnectionString(ctx, "sslmode=disable")
	if migrationsPath != "" {
		if err := database.RunMigrations(dbURL, migrationsPath); err != nil {
			panic(fmt.Sprintf("run migrations %s: %v", migrationsPath, err))
		}
	}

	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		panic(fmt.Sprintf("open postgres pool %s: %v", databaseName, err))
	}
	if err := db.Ping(ctx); err != nil {
		panic(fmt.Sprintf("ping postgres %s: %v", databaseName, err))
	}
	return container, db
}

func cyberMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "cyber_db")
}

func visusMigrationsPath() string {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(currentFile), "..", "..", "migrations", "visus_db")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func testcontainersSkipIfUnavailable(t *testing.T) {
	t.Helper()
	tc.SkipIfProviderIsNotHealthy(t)
}
