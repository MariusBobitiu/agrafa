// @title Agrafa API
// @version 1.0
// @description Agrafa backend API for agent ingestion, inventory creation, and read-side observability queries.
// @BasePath /v1
// @schemes http https
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/config"
	"github.com/MariusBobitiu/agrafa-backend/src/controllers"
	"github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/jobs"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/routes"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
)

const startupLogDelay = 100 * time.Millisecond

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	log.Printf(
		"backend startup\n  port: %s\n  database: %s\n  node_heartbeat_ttl: %s\n  node_expiry_check_interval: %s\n  managed_service_check_interval: %s\n  managed_service_check_timeout: %s",
		cfg.Port,
		formatDatabaseLogTarget(cfg.PostgresURI),
		cfg.NodeHeartbeatTTL,
		cfg.NodeExpiryCheckInterval,
		cfg.ManagedCheckInterval,
		cfg.ManagedCheckTimeout,
	)
	time.Sleep(startupLogDelay)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbConn, err := db.OpenPostgres(ctx, cfg.PostgresURI)
	if err != nil {
		log.Fatalf("open postgres: %v", err)
	}
	defer dbConn.Close()

	log.Print("database connected successfully...")
	time.Sleep(startupLogDelay)

	queries := generated.New(dbConn)

	projectRepo := repositories.NewProjectRepository(dbConn, queries)
	projectMemberRepo := repositories.NewProjectMemberRepository(queries)
	projectInvitationRepo := repositories.NewProjectInvitationRepository(dbConn, queries)
	userRepo := repositories.NewUserRepository(queries)
	authRepo := repositories.NewAuthRepository(dbConn, queries)
	nodeRepo := repositories.NewNodeRepository(queries)
	serviceRepo := repositories.NewServiceRepository(queries)
	heartbeatRepo := repositories.NewHeartbeatRepository(queries)
	healthCheckRepo := repositories.NewHealthCheckRepository(queries)
	metricRepo := repositories.NewMetricRepository(queries)
	eventRepo := repositories.NewEventRepository(queries)
	overviewRepo := repositories.NewOverviewRepository(queries)
	alertRuleRepo := repositories.NewAlertRuleRepository(queries)
	alertInstanceRepo := repositories.NewAlertInstanceRepository(queries)
	notificationRecipientRepo := repositories.NewNotificationRecipientRepository(queries)
	notificationDeliveryRepo := repositories.NewNotificationDeliveryRepository(queries)

	eventService := services.NewEventService(eventRepo)
	passwordService := services.NewPasswordService()
	sessionService := services.NewSessionService(cfg.SessionTTL, cfg.SessionRememberTTL, cfg.SessionCookieSecure)
	authService := services.NewAuthService(authRepo, passwordService, sessionService)
	authorizationService := services.NewAuthorizationService(projectMemberRepo, projectRepo, nodeRepo, serviceRepo, alertRuleRepo, notificationRecipientRepo, projectInvitationRepo)
	projectService := services.NewProjectService(projectRepo, projectMemberRepo, overviewRepo)
	projectMemberService := services.NewProjectMemberService(projectMemberRepo, projectRepo, userRepo)
	projectInvitationService := services.NewProjectInvitationService(projectInvitationRepo, projectRepo, projectMemberRepo, userRepo)
	nodeService := services.NewNodeService(nodeRepo, projectRepo, serviceRepo)
	inventoryService := services.NewServiceService(serviceRepo, projectRepo, nodeRepo)
	alertRuleService := services.NewAlertRuleService(alertRuleRepo, projectRepo, nodeRepo, serviceRepo)
	alertService := services.NewAlertService(alertInstanceRepo)
	notificationRecipientService := services.NewNotificationRecipientService(notificationRecipientRepo, projectRepo)
	notificationDeliveryService := services.NewNotificationDeliveryService(notificationDeliveryRepo)

	var emailService *emailpkg.Service
	var authEmailService *emailpkg.Service
	var inviteEmailService *emailpkg.Service
	if cfg.ResendAPIKey != "" && (cfg.AlertsFromEmail != "" || cfg.ResendEmailDomain != "") {
		emailRenderer := emailpkg.NewRenderer()
		emailSender := emailpkg.NewResendSender(cfg.ResendAPIKey)
		emailService = emailpkg.NewService(
			emailRenderer,
			emailSender,
			emailpkg.BuildAlertsFromAddress(cfg.ResendEmailDomain, cfg.AlertsFromEmail),
		)
		if cfg.ResendEmailDomain != "" {
			authEmailService = emailpkg.NewService(
				emailRenderer,
				emailSender,
				emailpkg.BuildSecurityFromAddress(cfg.ResendEmailDomain),
			)
			inviteEmailService = emailpkg.NewService(
				emailRenderer,
				emailSender,
				emailpkg.BuildNotificationsFromAddress(cfg.ResendEmailDomain),
			)
		}
		log.Print("email notifications configured via Resend")
	} else {
		log.Print("email notifications disabled: missing RESEND_API_KEY and sender address configuration")
	}

	authService.WithSecurityEmail(authEmailService, cfg.AppBaseURL)
	projectInvitationService.WithEmail(inviteEmailService, cfg.AppBaseURL)

	notificationService := services.NewNotificationService(notificationRecipientRepo, projectRepo, notificationDeliveryService, emailService)
	alertEvaluatorService := services.NewAlertEvaluatorService(alertRuleRepo, alertInstanceRepo, metricRepo, eventService, notificationService)
	agentAuthService := services.NewAgentAuthService(nodeRepo)
	nodeStateService := services.NewNodeStateService(nodeRepo, eventService, alertEvaluatorService)
	serviceStateService := services.NewServiceStateService(serviceRepo, eventService, alertEvaluatorService)
	nodeReadService := services.NewNodeReadService(nodeRepo, metricRepo, alertInstanceRepo, serviceRepo)
	serviceReadService := services.NewServiceReadService(serviceRepo, nodeRepo, healthCheckRepo, alertInstanceRepo)
	agentConfigService := services.NewAgentConfigService(serviceRepo)
	heartbeatService := services.NewHeartbeatService(heartbeatRepo, nodeStateService)
	healthIngestionService := services.NewHealthIngestionService(healthCheckRepo, serviceRepo, serviceStateService)
	metricIngestionService := services.NewMetricIngestionService(metricRepo, serviceRepo, alertEvaluatorService)
	overviewService := services.NewOverviewService(overviewRepo, eventService, nodeReadService)

	authController := controllers.NewAuthController(authService, sessionService)
	agentController := controllers.NewAgentController(heartbeatService, nodeStateService, healthIngestionService, metricIngestionService, agentConfigService)
	readController := controllers.NewReadController(nodeReadService, serviceReadService, eventService, alertRuleService, alertService, overviewService)
	projectController := controllers.NewProjectController(projectService)
	projectMemberController := controllers.NewProjectMemberController(projectMemberService)
	projectInvitationController := controllers.NewProjectInvitationController(projectInvitationService)
	nodeController := controllers.NewNodeController(nodeService, nodeReadService)
	serviceController := controllers.NewServiceController(inventoryService, serviceReadService)
	alertRuleController := controllers.NewAlertRuleController(alertRuleService)
	notificationRecipientController := controllers.NewNotificationRecipientController(notificationRecipientService)
	notificationDeliveryController := controllers.NewNotificationDeliveryController(notificationDeliveryService)
	docsController := controllers.NewDocsController()

	router := routes.NewRouter(authController, agentController, readController, projectController, projectMemberController, projectInvitationController, nodeController, serviceController, alertRuleController, notificationRecipientController, notificationDeliveryController, docsController, authService, authorizationService, sessionService, agentAuthService)

	expiryJob := jobs.NewNodeExpiryJob(nodeRepo, nodeStateService, cfg.NodeHeartbeatTTL, cfg.NodeExpiryCheckInterval)
	managedServiceCheckJob := jobs.NewManagedServiceCheckJob(serviceStateService, nodeStateService, heartbeatService, healthIngestionService, cfg.ManagedCheckInterval, cfg.ManagedCheckTimeout)
	go expiryJob.Start(ctx)
	go managedServiceCheckJob.Start(ctx)

	log.Printf(
		"all systems running\n  agent_auth: enabled\n  routes: ready\n  read_api: ready\n  docs: ready\n  background_jobs: node_expiry, managed_service_checks",
	)
	time.Sleep(startupLogDelay)

	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		// Extra log line for better separation in logs between startup logs and subsequent logs
		log.Printf(
			"Agrafa is now running\n  listen_addr: %s\n  docs: http://localhost:%s/docs\n  openapi: http://localhost:%s/openapi/swagger.json\n",
			server.Addr,
			cfg.Port,
			cfg.Port,
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	<-ctx.Done()

	log.Printf("shutdown requested\n  signal: received\n  action: stopping http server\n")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown failed\n  error: %v\n", err)
		return
	}

	log.Printf("shutdown complete\n  status: stopped cleanly\n")
}

func formatDatabaseLogTarget(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "redacted"
	}

	scheme := parsed.Scheme
	if scheme == "" {
		scheme = "postgres"
	}

	host := parsed.Hostname()
	if host == "" {
		host = "unknown"
	}

	port := parsed.Port()
	if port != "" {
		host += ":" + port
	}

	databaseName := parsed.Path
	if len(databaseName) > 0 && databaseName[0] == '/' {
		databaseName = databaseName[1:]
	}
	if databaseName == "" {
		databaseName = "unknown"
	}

	target := scheme + "://<credentials>@" + host + "/" + databaseName

	if sslmode := parsed.Query().Get("sslmode"); sslmode != "" {
		target += " sslmode=" + sslmode
	}

	return target
}
