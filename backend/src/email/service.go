package email

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AlertTemplateData struct {
	ProjectID   int64
	ProjectName string

	AlertTitle   string
	AlertMessage string
	RuleType     string
	RuleLabel    string
	Severity     string
	Status       string

	ResourceType string
	ResourceID   *int64
	ResourceName string

	NodeName       string
	NodeIdentifier string
	NodeState      string
	LastSeenAt     *time.Time

	ServiceName      string
	ServiceCheckType string
	ServiceTarget    string
	ServiceState     string

	StatusCode     *int
	ResponseTimeMs *int64
	FailureReason  string

	MetricName     string
	MetricLabel    string
	MetricValue    *float64
	ThresholdValue *float64

	TriggeredAt time.Time
	ResolvedAt  *time.Time
	Duration    string

	ResourceURL      string
	AlertsURL        string
	NotificationsURL string
}

type VerifyEmailTemplateData struct {
	Name      string
	VerifyURL string
}

type ResetPasswordTemplateData struct {
	Name     string
	ResetURL string
}

type ProjectInviteTemplateData struct {
	ProjectName string
	Role        string
	InviterName string
	AcceptURL   string
}

type NotificationRecipientTestTemplateData struct {
	ProjectName string
	ProjectID   int64
	Recipient   string
	SentAt      time.Time
}

type Service struct {
	renderer *Renderer
	sender   Sender
	from     string
}

type alertEmailDefinition struct {
	subject      func(AlertTemplateData) string
	htmlTemplate string
	textTemplate string
}

func NewService(renderer *Renderer, sender Sender, from string) *Service {
	return &Service{
		renderer: renderer,
		sender:   sender,
		from:     strings.TrimSpace(from),
	}
}

func BuildAlertsFromAddress(domain, override string) string {
	override = strings.TrimSpace(override)
	if override != "" {
		return override
	}

	return fmt.Sprintf("Agrafa Alerts <alerts@%s>", strings.TrimSpace(domain))
}

func BuildSecurityFromAddress(domain string) string {
	return fmt.Sprintf("Agrafa Security <security@%s>", strings.TrimSpace(domain))
}

func BuildNotificationsFromAddress(domain string) string {
	return fmt.Sprintf("Agrafa Notifications <notifications@%s>", strings.TrimSpace(domain))
}

func (s *Service) SendAlertTriggeredEmail(ctx context.Context, to string, data AlertTemplateData) error {
	return s.sendAlert(ctx, to, alertTriggeredEmailDefinition(), data)
}

func (s *Service) SendAlertResolvedEmail(ctx context.Context, to string, data AlertTemplateData) error {
	return s.sendAlert(ctx, to, alertResolvedEmailDefinition(), data)
}

func (s *Service) SendVerifyEmail(ctx context.Context, to string, name string, verifyURL string) error {
	data := VerifyEmailTemplateData{Name: name, VerifyURL: verifyURL}
	htmlBody, err := s.renderHTML("verify_email.html", data)
	if err != nil {
		return err
	}

	textBody, err := s.renderText("verify_email.txt", data)
	if err != nil {
		return err
	}

	return s.sendMessage(ctx, Message{
		From:    s.from,
		To:      []string{strings.TrimSpace(to)},
		Subject: "[Agrafa] Verify your email",
		HTML:    htmlBody,
		Text:    textBody,
	})
}

func (s *Service) SendPasswordResetEmail(ctx context.Context, to string, name string, resetURL string) error {
	data := ResetPasswordTemplateData{Name: name, ResetURL: resetURL}
	htmlBody, err := s.renderHTML("reset_password.html", data)
	if err != nil {
		return err
	}

	textBody, err := s.renderText("reset_password.txt", data)
	if err != nil {
		return err
	}

	return s.sendMessage(ctx, Message{
		From:    s.from,
		To:      []string{strings.TrimSpace(to)},
		Subject: "[Agrafa] Reset your password",
		HTML:    htmlBody,
		Text:    textBody,
	})
}

func (s *Service) SendProjectInvite(ctx context.Context, to string, data ProjectInviteTemplateData) error {
	htmlBody, err := s.renderHTML("project_invite.html", data)
	if err != nil {
		return err
	}

	textBody, err := s.renderText("project_invite.txt", data)
	if err != nil {
		return err
	}

	subject := "[Agrafa] You're invited to join"
	if strings.TrimSpace(data.ProjectName) != "" {
		subject = "[Agrafa] You're invited to join " + data.ProjectName
	}

	return s.sendMessage(ctx, Message{
		From:    s.from,
		To:      []string{strings.TrimSpace(to)},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	})
}

func (s *Service) SendNotificationRecipientTestEmail(ctx context.Context, to string, data NotificationRecipientTestTemplateData) error {
	htmlBody, err := s.renderHTML("notification_recipient_test.html", data)
	if err != nil {
		return err
	}

	textBody, err := s.renderText("notification_recipient_test.txt", data)
	if err != nil {
		return err
	}

	subject := "[Agrafa] Test notification email"
	if strings.TrimSpace(data.ProjectName) != "" {
		subject = "[Agrafa] Test notification email for " + data.ProjectName
	}

	return s.sendMessage(ctx, Message{
		From:    s.from,
		To:      []string{strings.TrimSpace(to)},
		Subject: subject,
		HTML:    htmlBody,
		Text:    textBody,
	})
}

func (s *Service) sendAlert(ctx context.Context, to string, definition alertEmailDefinition, data AlertTemplateData) error {
	if s == nil || s.renderer == nil || s.sender == nil {
		return nil
	}

	htmlBody, err := s.renderHTML(definition.htmlTemplate, data)
	if err != nil {
		return err
	}

	textBody, err := s.renderText(definition.textTemplate, data)
	if err != nil {
		return err
	}

	return s.sendMessage(ctx, Message{
		From:    s.from,
		To:      []string{strings.TrimSpace(to)},
		Subject: definition.subject(data),
		HTML:    htmlBody,
		Text:    textBody,
	})
}

func (s *Service) sendMessage(ctx context.Context, message Message) error {
	if s == nil || s.renderer == nil || s.sender == nil {
		return nil
	}

	return s.sender.Send(ctx, message)
}

func (s *Service) renderHTML(templateName string, data any) (string, error) {
	if s == nil || s.renderer == nil {
		return "", fmt.Errorf("email renderer is not configured")
	}

	return s.renderer.RenderHTML(templateName, data)
}

func (s *Service) renderText(templateName string, data any) (string, error) {
	if s == nil || s.renderer == nil {
		return "", fmt.Errorf("email renderer is not configured")
	}

	return s.renderer.RenderText(templateName, data)
}

func alertTriggeredEmailDefinition() alertEmailDefinition {
	return alertEmailDefinition{
		subject: func(data AlertTemplateData) string {
			return alertTriggeredSubject(data)
		},
		htmlTemplate: "alert_triggered.html",
		textTemplate: "alert_triggered.txt",
	}
}

func alertResolvedEmailDefinition() alertEmailDefinition {
	return alertEmailDefinition{
		subject: func(data AlertTemplateData) string {
			return alertResolvedSubject(data)
		},
		htmlTemplate: "alert_resolved.html",
		textTemplate: "alert_resolved.txt",
	}
}

func alertTriggeredSubject(data AlertTemplateData) string {
	severity := titleCase(data.Severity)
	if severity == "" {
		severity = "Alert"
	}

	var description string
	switch data.RuleType {
	case "service_unhealthy":
		description = fallback(data.ServiceName, "Service") + " is unhealthy"
	case "node_offline":
		description = fallback(data.NodeName, "Node") + " is offline"
	case "cpu_above_threshold", "memory_above_threshold", "disk_above_threshold":
		description = fallback(data.MetricLabel, "Metric") + " above " + formatPercent(data.ThresholdValue) + " on " + fallback(data.NodeName, "node")
	default:
		description = strings.TrimPrefix(strings.TrimSpace(data.AlertTitle), "⚠ ")
	}

	return subjectWithProject("["+severity+"] "+description, data.ProjectName)
}

func alertResolvedSubject(data AlertTemplateData) string {
	var description string
	switch data.RuleType {
	case "service_unhealthy":
		description = fallback(data.ServiceName, "Service") + " recovered"
	case "node_offline":
		description = fallback(data.NodeName, "Node") + " is back online"
	case "cpu_above_threshold", "memory_above_threshold", "disk_above_threshold":
		description = fallback(data.MetricLabel, "Metric") + " back within threshold on " + fallback(data.NodeName, "node")
	default:
		description = strings.TrimPrefix(strings.TrimSpace(data.AlertTitle), "✓ ")
	}

	return subjectWithProject("[Resolved] "+description, data.ProjectName)
}

func subjectWithProject(subject, projectName string) string {
	if projectName = strings.TrimSpace(projectName); projectName != "" {
		return subject + " — " + projectName
	}
	return subject
}

func fallback(value, fallbackValue string) string {
	if value = strings.TrimSpace(value); value != "" {
		return value
	}
	return fallbackValue
}

func titleCase(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + strings.ToLower(value[1:])
}

func formatPercent(value *float64) string {
	if value == nil {
		return "threshold"
	}
	return fmt.Sprintf("%g%%", *value)
}
