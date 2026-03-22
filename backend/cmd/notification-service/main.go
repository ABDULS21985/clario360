package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/sync/errgroup"

	"github.com/clario360/platform/internal/config"
	"github.com/clario360/platform/internal/database"
	"github.com/clario360/platform/internal/events"
	intbot "github.com/clario360/platform/internal/integration/bot"
	intconsumer "github.com/clario360/platform/internal/integration/consumer"
	intencrypt "github.com/clario360/platform/internal/integration/encryption"
	inthandler "github.com/clario360/platform/internal/integration/handler"
	intmodel "github.com/clario360/platform/internal/integration/model"
	intrepo "github.com/clario360/platform/internal/integration/repository"
	intservice "github.com/clario360/platform/internal/integration/service"
	jirasvc "github.com/clario360/platform/internal/integration/service/jira"
	servicenowsvc "github.com/clario360/platform/internal/integration/service/servicenow"
	slacksvc "github.com/clario360/platform/internal/integration/service/slack"
	teamssvc "github.com/clario360/platform/internal/integration/service/teams"
	webhooksvc "github.com/clario360/platform/internal/integration/service/webhook"
	"github.com/clario360/platform/internal/middleware"
	notifchannel "github.com/clario360/platform/internal/notification/channel"
	notifcfg "github.com/clario360/platform/internal/notification/config"
	"github.com/clario360/platform/internal/notification/consumer"
	"github.com/clario360/platform/internal/notification/handler"
	"github.com/clario360/platform/internal/notification/health"
	_ "github.com/clario360/platform/internal/notification/metrics" // registers Prometheus metrics on import
	notifmw "github.com/clario360/platform/internal/notification/middleware"
	notifrepo "github.com/clario360/platform/internal/notification/repository"
	notifservice "github.com/clario360/platform/internal/notification/service"
	"github.com/clario360/platform/internal/notification/websocket"
	"github.com/clario360/platform/internal/observability"
	"github.com/clario360/platform/internal/server"
)

func main() {
	// 1. Load platform config.
	cfg, err := config.Load()
	if err != nil {
		panic("loading config: " + err.Error())
	}

	// Load notification-specific config.
	notifCfg := notifcfg.LoadFromEnv()
	if err := notifCfg.Validate(); err != nil {
		panic("invalid notification config: " + err.Error())
	}
	cfg.Server.Port = notifCfg.HTTPPort

	// 2. Initialize logger.
	logger := observability.NewLogger(
		cfg.Observability.LogLevel,
		cfg.Observability.LogFormat,
		"notification-service",
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 3. Initialize tracer.
	shutdownTracer, err := observability.InitTracer(ctx, "notification-service", cfg.Observability.OTLPEndpoint)
	if err != nil {
		logger.Warn().Err(err).Msg("failed to initialize tracer")
	} else {
		defer shutdownTracer(ctx)
	}

	// 4. Connect PostgreSQL.
	db, err := database.NewPostgresPool(ctx, cfg.Database, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// 5. Run schema migration.
	if err := notifrepo.RunMigration(ctx, db); err != nil {
		logger.Fatal().Err(err).Msg("failed to run notification schema migration")
	}
	logger.Info().Msg("notification schema migration completed")

	// 6. Connect Redis.
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer rdb.Close()

	if err := rdb.Ping(ctx).Err(); err != nil {
		logger.Warn().Err(err).Msg("redis connection failed — continuing with degraded functionality")
	}

	// 6. Create HTTP server with middleware stack.
	srv, err := server.New(cfg, db, rdb, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	// 7. Initialize Kafka producer.
	var producer *events.Producer
	kafkaProducer, err := events.NewProducer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka producer unavailable — notification events will not be published")
	} else {
		producer = kafkaProducer
		defer producer.Close()
	}

	// 8. Initialize WebSocket hub.
	hub := websocket.NewHub(notifCfg.WSMaxConnectionsPerUser, logger)

	// 9. Initialize repositories.
	notifRepo := notifrepo.NewNotificationRepository(db, logger)
	prefRepo := notifrepo.NewPreferenceRepository(db, logger)
	deliveryRepo := notifrepo.NewDeliveryRepository(db, logger)
	webhookRepo := notifrepo.NewWebhookRepository(db, logger)
	integrationRepo := intrepo.NewIntegrationRepository(db, logger)
	integrationDeliveryRepo := intrepo.NewDeliveryRepository(db, logger)
	ticketLinkRepo := intrepo.NewTicketLinkRepository(db, logger)

	// 10. Initialize services.
	tmplSvc := notifservice.NewTemplateService(logger)
	prefSvc := notifservice.NewPreferenceService(prefRepo, rdb, logger)

	// Initialize channels.
	websocketChannel := notifchannel.NewWebSocketChannel(hub, logger)
	channels := map[string]notifchannel.Channel{
		"in_app":    notifchannel.NewInAppChannel(hub, logger),
		"websocket": websocketChannel,
		"push":      websocketChannel,
		"email": notifchannel.NewEmailChannel(notifchannel.EmailConfig{
			Provider:       notifCfg.EmailProvider,
			SMTPHost:       notifCfg.SMTPHost,
			SMTPPort:       notifCfg.SMTPPort,
			SMTPUser:       notifCfg.SMTPUsername,
			SMTPPass:       notifCfg.SMTPPassword,
			SMTPFrom:       notifCfg.SMTPFrom,
			TLSEnabled:     notifCfg.SMTPTLSEnabled,
			SendGridAPIKey: notifCfg.SendGridAPIKey,
			SendGridFrom:   notifCfg.SendGridFrom,
		}, tmplSvc, logger),
		"webhook": notifchannel.NewWebhookChannel(
			webhookRepo,
			time.Duration(notifCfg.WebhookTimeoutSec)*time.Second,
			notifCfg.WebhookHMACSecret,
			notifCfg.Environment,
			logger,
		),
	}

	dispatcher := notifservice.NewDispatcherService(channels, deliveryRepo, logger)
	notifSvc := notifservice.NewNotificationService(notifRepo, prefSvc, dispatcher, tmplSvc, producer, rdb, logger)
	digestSvc := notifservice.NewDigestService(notifRepo, prefRepo, tmplSvc, dispatcher, notifCfg.DigestDailyUTCHour, notifCfg.DigestWeeklyDay, logger)
	guard := events.NewIdempotencyGuard(rdb, 24*time.Hour)
	crossSuiteMetrics := events.NewCrossSuiteMetrics(prometheus.DefaultRegisterer)
	dlqTracker := events.NewDLQTracker(rdb)

	encryptor, err := intencrypt.NewConfigEncryptor(cfg.Encryption.Key, "notification-service")
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize integration config encryptor")
	}
	clarioAPI := intservice.NewClarioAPIClient(notifCfg.GatewayURL, notifCfg.IAMServiceURL, srv.JWTManager, logger)
	slackClient := slacksvc.NewClient(15*time.Second, notifCfg.PublicAppURL)
	teamsClient := teamssvc.NewClient(15*time.Second, notifCfg.PublicAppURL)
	jiraClient := jirasvc.NewClient(20 * time.Second)
	snClient := servicenowsvc.NewClient(25 * time.Second)
	webhookClient := webhooksvc.NewClient(15 * time.Second)
	slackMapper := slacksvc.NewUserMapper(slackClient, rdb, logger)
	jiraTicketSvc := jirasvc.NewTicketService(jiraClient, clarioAPI, ticketLinkRepo, notifCfg.PublicAppURL)
	snIncidentSvc := servicenowsvc.NewIncidentService(snClient, clarioAPI, ticketLinkRepo, notifCfg.PublicAppURL)
	integrationSvc := intservice.NewIntegrationService(
		integrationRepo,
		integrationDeliveryRepo,
		ticketLinkRepo,
		encryptor,
		producer,
		slackClient,
		teamsClient,
		jiraTicketSvc,
		snIncidentSvc,
		webhookClient,
		logger,
	)
	integrationDeliverySvc := intservice.NewDeliveryService(
		integrationDeliveryRepo,
		integrationRepo,
		encryptor,
		rdb,
		slackClient,
		teamsClient,
		jiraTicketSvc,
		snIncidentSvc,
		webhookClient,
		logger,
	)
	integrationWorker := intservice.NewDeliveryWorker(integrationDeliverySvc, integrationDeliveryRepo, logger)
	botRouter := intbot.NewRouter(clarioAPI, logger)
	gatewayURL := strings.TrimRight(notifCfg.GatewayURL, "/")
	providerStatuses := []inthandler.ProviderStatus{
		{
			Type:             intmodel.IntegrationTypeSlack,
			Name:             "Slack",
			Description:      "Workspace install plus slash commands, interactions, and outbound alert delivery.",
			SetupMode:        "oauth",
			Configured:       strings.TrimSpace(notifCfg.SlackClientID) != "" && strings.TrimSpace(notifCfg.SlackClientSecret) != "" && strings.TrimSpace(notifCfg.SlackSigningSecret) != "",
			OAuthEnabled:     true,
			OAuthStartURL:    gatewayURL + "/api/v1/integrations/slack/oauth/start",
			MissingConfig:    missingProviderConfig(map[string]string{"NOTIF_SLACK_CLIENT_ID": notifCfg.SlackClientID, "NOTIF_SLACK_CLIENT_SECRET": notifCfg.SlackClientSecret, "NOTIF_SLACK_SIGNING_SECRET": notifCfg.SlackSigningSecret}),
			SupportsInbound:  true,
			SupportsOutbound: true,
		},
		{
			Type:             intmodel.IntegrationTypeTeams,
			Name:             "Microsoft Teams",
			Description:      "Manual Bot Framework setup for outbound cards and inbound bot commands.",
			SetupMode:        "manual",
			Configured:       true,
			OAuthEnabled:     false,
			SupportsInbound:  true,
			SupportsOutbound: true,
		},
		{
			Type:             intmodel.IntegrationTypeJira,
			Name:             "Jira Cloud",
			Description:      "OAuth install for outbound ticket creation and inbound status synchronization.",
			SetupMode:        "oauth",
			Configured:       strings.TrimSpace(notifCfg.AtlassianClientID) != "" && strings.TrimSpace(notifCfg.AtlassianClientSecret) != "",
			OAuthEnabled:     true,
			OAuthStartURL:    gatewayURL + "/api/v1/integrations/jira/oauth/start",
			MissingConfig:    missingProviderConfig(map[string]string{"NOTIF_ATLASSIAN_CLIENT_ID": notifCfg.AtlassianClientID, "NOTIF_ATLASSIAN_CLIENT_SECRET": notifCfg.AtlassianClientSecret}),
			SupportsInbound:  true,
			SupportsOutbound: true,
		},
		{
			Type:             intmodel.IntegrationTypeServiceNow,
			Name:             "ServiceNow",
			Description:      "Manual incident integration with bidirectional webhook synchronization.",
			SetupMode:        "manual",
			Configured:       true,
			OAuthEnabled:     false,
			SupportsInbound:  true,
			SupportsOutbound: true,
		},
		{
			Type:             intmodel.IntegrationTypeWebhook,
			Name:             "Generic Webhook",
			Description:      "Signed outbound webhook delivery to arbitrary HTTP endpoints.",
			SetupMode:        "manual",
			Configured:       true,
			OAuthEnabled:     false,
			SupportsInbound:  false,
			SupportsOutbound: true,
		},
	}

	// 11. Initialize handlers.
	notifHandler := handler.NewNotificationHandler(notifSvc, notifRepo, logger)
	prefHandler := handler.NewPreferenceHandler(prefSvc, webhookRepo, deliveryRepo, logger)
	wsHandler := handler.NewWebSocketHandler(hub, srv.JWTManager, notifRepo, notifCfg, logger)
	adminHandler := handler.NewAdminHandler(notifSvc, deliveryRepo, dispatcher, logger)
	integrationHandler := inthandler.NewIntegrationHandler(integrationSvc, providerStatuses, logger)
	slackHandler := inthandler.NewSlackHandler(
		integrationSvc,
		clarioAPI,
		slackClient,
		slackMapper,
		botRouter,
		producer,
		rdb,
		slacksvc.OAuthConfig{
			ClientID:     notifCfg.SlackClientID,
			ClientSecret: notifCfg.SlackClientSecret,
			RedirectURI:  strings.TrimRight(notifCfg.GatewayURL, "/") + "/api/v1/integrations/slack/oauth/callback",
			Scopes:       notifCfg.SlackScopes,
		},
		notifCfg.SlackSigningSecret,
		notifCfg.PublicAppURL,
		time.Duration(notifCfg.IntegrationStateTTLMin)*time.Minute,
		logger,
	)
	teamsHandler := inthandler.NewTeamsHandler(integrationSvc, clarioAPI, teamsClient, botRouter, producer, logger)
	jiraHandler := inthandler.NewJiraHandler(
		integrationSvc,
		jiraTicketSvc,
		producer,
		rdb,
		jirasvc.OAuthConfig{
			ClientID:     notifCfg.AtlassianClientID,
			ClientSecret: notifCfg.AtlassianClientSecret,
			RedirectURI:  strings.TrimRight(notifCfg.GatewayURL, "/") + "/api/v1/integrations/jira/oauth/callback",
			Scopes:       notifCfg.AtlassianScopes,
		},
		notifCfg.PublicAppURL,
		time.Duration(notifCfg.IntegrationStateTTLMin)*time.Minute,
		logger,
	)
	serviceNowHandler := inthandler.NewServiceNowHandler(integrationSvc, snIncidentSvc, producer, logger)
	webhookHandler := inthandler.NewWebhookHandler(logger)

	// 12. Initialize health checker.
	smtpAddr := ""
	if notifCfg.EmailProvider == "smtp" && notifCfg.SMTPHost != "" {
		smtpAddr = fmt.Sprintf("%s:%d", notifCfg.SMTPHost, notifCfg.SMTPPort)
	}
	healthChecker := health.NewChecker(db, rdb, cfg.Kafka.Brokers, smtpAddr, logger)

	// Override health and metrics endpoints.
	srv.Router.Get("/healthz", health.LivenessHandler())
	srv.Router.Get("/readyz", healthChecker.ReadinessHandler())
	srv.Router.Handle("/metrics", promhttp.Handler())
	srv.Router.Get("/api/v1/admin/dlq/count", events.DLQCountHandler("notification-service", dlqTracker, logger))

	// 13. WebSocket endpoint (authenticated via query param, not middleware).
	srv.Router.Get("/ws/v1/notifications", wsHandler.HandleWebSocket)

	// 14. Register API routes.
	srv.Router.Route("/api/v1/notifications", func(r chi.Router) {
		r.Use(middleware.Auth(srv.JWTManager))
		r.Use(notifmw.TenantGuard)
		r.Use(notifmw.RateLimiter(rdb, notifCfg.RateLimitPerMinute, logger))

		// Notification endpoints.
		r.Get("/", notifHandler.ListNotifications)
		r.Get("/counts", notifHandler.GetCounts)
		r.Get("/unread-count", notifHandler.UnreadCount)
		r.Get("/read-all", notifHandler.MarkAllRead) // PUT mapped to GET for simplicity; see below
		r.Put("/read-all", notifHandler.MarkAllRead)
		r.Post("/bulk", notifHandler.BulkDeleteNotifications)
		r.Get("/{id}", notifHandler.GetNotification)
		r.Put("/{id}/read", notifHandler.MarkRead)
		r.Delete("/{id}", notifHandler.DeleteNotification)

		// Preference endpoints.
		r.Get("/preferences", prefHandler.GetPreferences)
		r.Put("/preferences", prefHandler.UpdatePreferences)

		// Webhook endpoints.
		r.Get("/webhooks", prefHandler.ListWebhooks)
		r.Post("/webhooks", prefHandler.CreateWebhook)
		r.Get("/webhooks/{id}", prefHandler.GetWebhook)
		r.Put("/webhooks/{id}", prefHandler.UpdateWebhook)
		r.Delete("/webhooks/{id}", prefHandler.DeleteWebhook)
		r.Post("/webhooks/{id}/test", prefHandler.TestWebhook)
		r.Post("/webhooks/{id}/rotate", prefHandler.RotateWebhookSecret)
		r.Get("/webhooks/{id}/deliveries", prefHandler.ListWebhookDeliveries)
		r.Post("/webhooks/{id}/deliveries/{deliveryId}/retry", prefHandler.RetryWebhookDelivery)

		// Admin endpoints.
		r.Post("/test", adminHandler.SendTestNotification)
		r.Get("/delivery-stats", adminHandler.GetDeliveryStats)
		r.Post("/retry-failed", adminHandler.RetryFailed)
	})
	inthandler.RegisterRoutes(srv.Router, inthandler.RouteDependencies{
		JWTManager:         srv.JWTManager,
		Redis:              rdb,
		RateLimitPerMinute: notifCfg.RateLimitPerMinute,
		Integration:        integrationHandler,
		Slack:              slackHandler,
		Teams:              teamsHandler,
		Jira:               jiraHandler,
		ServiceNow:         serviceNowHandler,
		Webhook:            webhookHandler,
		Logger:             logger,
	})

	// 15. Initialize Kafka consumer.
	var notifConsumer *consumer.NotificationConsumer
	kafkaConsumer, err := events.NewConsumer(cfg.Kafka, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("kafka consumer unavailable — notification event ingestion disabled")
	} else {
		kafkaConsumer.SetDeadLetterProducer(producer)
		kafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
		kafkaConsumer.SetDLQTracker(dlqTracker, "notification-service")
		recipientResolver := consumer.NewRecipientResolver(
			notifCfg.IAMServiceURL,
			notifCfg.DataServiceURL,
			notifCfg.ActaServiceURL,
			notifCfg.CyberServiceURL,
			logger,
		)
		notifConsumer = consumer.NewNotificationConsumer(kafkaConsumer, notifSvc, recipientResolver, guard, crossSuiteMetrics, logger)
	}
	var integrationConsumer *intconsumer.IntegrationConsumer
	integrationKafkaConsumer, err := events.NewConsumerWithConfig(events.ConsumerConfig{
		Brokers:             cfg.Kafka.Brokers,
		GroupID:             "integration-delivery-consumer",
		AutoOffsetReset:     cfg.Kafka.AutoOffsetReset,
		WorkersPerPartition: 1,
	}, logger)
	if err != nil {
		logger.Warn().Err(err).Msg("integration kafka consumer unavailable — external delivery ingestion disabled")
	} else {
		integrationKafkaConsumer.SetDeadLetterProducer(producer)
		integrationKafkaConsumer.SetCrossSuiteMetrics(crossSuiteMetrics)
		integrationKafkaConsumer.SetDLQTracker(dlqTracker, "notification-service")
		integrationConsumer = intconsumer.NewIntegrationConsumer(integrationKafkaConsumer, integrationRepo, integrationDeliverySvc, rdb, time.Minute, logger)
	}

	// 16. Start all components via errgroup.
	g, gCtx := errgroup.WithContext(ctx)

	// WebSocket hub.
	g.Go(func() error {
		return hub.Run(gCtx)
	})

	// HTTP server.
	g.Go(func() error {
		logger.Info().Int("port", cfg.Server.Port).Msg("notification-service HTTP server starting")
		if err := srv.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	// Kafka consumer.
	if notifConsumer != nil {
		g.Go(func() error {
			return notifConsumer.Start(gCtx)
		})
	}
	if integrationConsumer != nil {
		g.Go(func() error {
			return integrationConsumer.Start(gCtx)
		})
	}

	// Digest scheduler.
	if notifCfg.DigestEnabled {
		g.Go(func() error {
			return digestSvc.RunScheduler(gCtx)
		})
	}
	g.Go(func() error {
		return integrationWorker.Run(gCtx)
	})

	// 17. Wait for shutdown signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	case <-gCtx.Done():
		logger.Info().Msg("context cancelled")
	}

	// 18. Graceful shutdown sequence.
	cancel()

	// Shutdown HTTP server first (stops accepting new connections).
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := srv.HTTPServer.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
	}

	// Stop Kafka consumer.
	if notifConsumer != nil {
		if err := notifConsumer.Stop(); err != nil {
			logger.Error().Err(err).Msg("kafka consumer shutdown error")
		}
	}
	if integrationConsumer != nil {
		if err := integrationConsumer.Stop(); err != nil {
			logger.Error().Err(err).Msg("integration kafka consumer shutdown error")
		}
	}

	// Hub shutdown happens via context cancellation (gCtx.Done in hub.Run).

	if err := g.Wait(); err != nil {
		logger.Error().Err(err).Msg("errgroup finished with error")
	}

	logger.Info().Msg("notification-service stopped")
}

func missingProviderConfig(values map[string]string) []string {
	missing := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}
