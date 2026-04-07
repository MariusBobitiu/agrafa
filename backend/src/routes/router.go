package routes

import (
	"net/http"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/controllers"
	agentmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	authController *controllers.AuthController,
	agentController *controllers.AgentController,
	readController *controllers.ReadController,
	projectController *controllers.ProjectController,
	projectMemberController *controllers.ProjectMemberController,
	projectInvitationController *controllers.ProjectInvitationController,
	nodeController *controllers.NodeController,
	serviceController *controllers.ServiceController,
	alertRuleController *controllers.AlertRuleController,
	notificationRecipientController *controllers.NotificationRecipientController,
	notificationDeliveryController *controllers.NotificationDeliveryController,
	docsController *controllers.DocsController,
	authService *services.AuthService,
	authorizationService *services.AuthorizationService,
	sessionService *services.SessionService,
	agentAuthService *services.AgentAuthService,
) http.Handler {
	router := chi.NewRouter()

	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Logger)
	router.Use(chimiddleware.Recoverer)
	router.Use(chimiddleware.Timeout(30 * time.Second))

	router.Get("/docs", docsController.Scalar)
	router.Handle("/openapi/*", http.StripPrefix("/openapi/", http.FileServer(http.Dir("docs"))))

	router.Route("/v1", func(r chi.Router) {
		r.Get("/health", controllers.Health)

		r.Route("/auth", func(authRouter chi.Router) {
			authRouter.Post("/register", authController.Register)
			authRouter.Post("/login", authController.Login)
			authRouter.Post("/logout", authController.Logout)
			authRouter.Post("/verify-email/confirm", authController.ConfirmVerifyEmail)
			authRouter.Post("/forgot-password", authController.ForgotPassword)
			authRouter.Post("/reset-password", authController.ResetPassword)
			authRouter.Get("/me", authController.Me)
		})

		r.Get("/project-invitations/by-token", projectInvitationController.GetByToken)

		r.Route("/agent", func(agentRouter chi.Router) {
			agentRouter.Use(agentmiddleware.AgentAuth(agentAuthService))
			agentRouter.Get("/config", agentController.GetConfig)
			agentRouter.Post("/heartbeat", agentController.IngestHeartbeat)
			agentRouter.Post("/shutdown", agentController.IngestShutdown)
			agentRouter.Post("/health", agentController.IngestHealth)
			agentRouter.Post("/metrics", agentController.IngestMetrics)
		})

		r.Group(func(protected chi.Router) {
			protected.Use(agentmiddleware.SessionAuth(authService, sessionService))

			protected.Get("/auth/sessions", authController.ListSessions)
			protected.Post("/auth/logout-all", authController.LogoutAll)
			protected.Post("/auth/onboarding/complete", authController.CompleteOnboarding)
			protected.Post("/auth/verify-email/send", authController.SendVerifyEmail)
			protected.Post("/auth/verify-password", authController.VerifyPassword)
			protected.Delete("/auth/sessions/{id}", authController.DeleteSession)
			protected.Get("/projects", projectController.List)
			protected.Post("/projects", projectController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionProjectRead, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForProject)),
			).Get("/projects/{id}", projectController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionProjectUpdate, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForProject)),
			).Patch("/projects/{id}", projectController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionProjectDelete, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForProject)),
			).Delete("/projects/{id}", projectController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/project-members", projectMemberController.List)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersRead, agentmiddleware.ProjectIDFromURLParamStringResource("id", authorizationService.ProjectIDForProjectMember)),
			).Get("/project-members/{id}", projectMemberController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersManage, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/project-members", projectMemberController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersManage, agentmiddleware.ProjectIDFromURLParamStringResource("id", authorizationService.ProjectIDForProjectMember)),
			).Patch("/project-members/{id}", projectMemberController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersManage, agentmiddleware.ProjectIDFromURLParamStringResource("id", authorizationService.ProjectIDForProjectMember)),
			).Delete("/project-members/{id}", projectMemberController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersManage, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/project-invitations", projectInvitationController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/project-invitations", projectInvitationController.List)
			protected.Post("/project-invitations/accept", projectInvitationController.Accept)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionMembersManage, agentmiddleware.ProjectIDFromURLParamStringResource("id", authorizationService.ProjectIDForProjectInvitation)),
			).Delete("/project-invitations/{id}", projectInvitationController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNodesWrite, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/nodes", nodeController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNodesRead, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNode)),
			).Get("/nodes/{id}", nodeController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNodesWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNode)),
			).Patch("/nodes/{id}", nodeController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNodesWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNode)),
			).Delete("/nodes/{id}", nodeController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsRead, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForAlertRule)),
			).Get("/alert-rules/{id}", alertRuleController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAgentTokensWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNode)),
			).Post("/nodes/{id}/regenerate-agent-token", nodeController.RegenerateAgentToken)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionServicesWrite, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/services", serviceController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionServicesRead, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForService)),
			).Get("/services/{id}", serviceController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionServicesWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForService)),
			).Patch("/services/{id}", serviceController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionServicesWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForService)),
			).Delete("/services/{id}", serviceController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsWrite, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/alert-rules", alertRuleController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForAlertRule)),
			).Patch("/alert-rules/{id}", alertRuleController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForAlertRule)),
			).Delete("/alert-rules/{id}", alertRuleController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNotificationRecipientsWrite, agentmiddleware.ProjectIDFromBodyField("project_id")),
			).Post("/notification-recipients", notificationRecipientController.Create)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNotificationRecipientsRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/notification-recipients", notificationRecipientController.List)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNotificationRecipientsRead, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNotificationRecipient)),
			).Get("/notification-recipients/{id}", notificationRecipientController.Get)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNotificationRecipientsWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNotificationRecipient)),
			).Patch("/notification-recipients/{id}", notificationRecipientController.Update)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNotificationRecipientsWrite, agentmiddleware.ProjectIDFromURLParamResource("id", authorizationService.ProjectIDForNotificationRecipient)),
			).Delete("/notification-recipients/{id}", notificationRecipientController.Delete)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/notification-deliveries", notificationDeliveryController.List)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionNodesRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/nodes", readController.ListNodes)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionServicesRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/services", readController.ListServices)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/alert-rules", readController.ListAlertRules)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionAlertsRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/alerts", readController.ListAlerts)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionEventsRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/events", readController.ListEvents)
			protected.With(
				withProjectPermission(authorizationService, services.PermissionProjectRead, agentmiddleware.ProjectIDFromRequiredQueryParam("project_id")),
			).Get("/overview", readController.GetOverview)
		})
	})

	return router
}

func withProjectPermission(
	authorizationService *services.AuthorizationService,
	permission string,
	resolver agentmiddleware.ProjectIDResolver,
) func(http.Handler) http.Handler {
	return agentmiddleware.RequireProjectPermission(authorizationService, permission, resolver)
}
