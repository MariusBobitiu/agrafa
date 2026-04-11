package email

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type AlertTemplateData struct {
	ProjectID    int64
	ProjectName  string
	AlertTitle   string
	AlertMessage string
	RuleType     string
	Status       string
	NodeID       *int64
	ServiceID    *int64
	TriggeredAt  time.Time
	ResolvedAt   *time.Time
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
			return "[Agrafa] Alert triggered: " + data.AlertTitle
		},
		htmlTemplate: "alert_triggered.html",
		textTemplate: "alert_triggered.txt",
	}
}

func alertResolvedEmailDefinition() alertEmailDefinition {
	return alertEmailDefinition{
		subject: func(data AlertTemplateData) string {
			return "[Agrafa] Alert resolved: " + data.AlertTitle
		},
		htmlTemplate: "alert_resolved.html",
		textTemplate: "alert_resolved.txt",
	}
}
