package email

import (
	"context"
	"strings"
	"testing"
	"time"
)

type fakeSender struct {
	messages []Message
}

func (s *fakeSender) Send(_ context.Context, message Message) error {
	s.messages = append(s.messages, message)
	return nil
}

func TestSendAlertTriggeredEmailRendersHTMLTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Alerts <alerts@example.com>")

	err := service.SendAlertTriggeredEmail(context.Background(), "ops@example.com", AlertTemplateData{
		ProjectID: 1, ProjectName: "Agrafa", AlertTitle: "⚠ web-01 is offline",
		AlertMessage: "Agrafa stopped receiving heartbeats from this node.", RuleType: "node_offline", RuleLabel: "Node offline",
		Severity: "critical", Status: "active", ResourceType: "node", NodeName: "web-01",
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC), ResourceURL: "https://app.agrafa.test/nodes/5", AlertsURL: "https://app.agrafa.test/alerts",
	})
	if err != nil {
		t.Fatalf("SendAlertTriggeredEmail returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if !strings.Contains(message.HTML, "CRITICAL") || !strings.Contains(message.HTML, `data-title-icon="triggered"`) || !strings.Contains(message.HTML, "web-01 is offline") {
		t.Fatalf("expected rendered HTML to contain heading, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "CRITICAL — web-01 is offline") || !strings.Contains(message.Text, "Open alerts:") {
		t.Fatalf("expected rendered text to contain heading, got %q", message.Text)
	}

	if !strings.Contains(message.HTML, "web-01 is offline") {
		t.Fatalf("expected rendered HTML to contain alert title, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "web-01 is offline") {
		t.Fatalf("expected rendered text to contain alert title, got %q", message.Text)
	}
}

func TestSendAlertResolvedEmailRendersHTMLTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Alerts <alerts@example.com>")
	resolvedAt := time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)

	err := service.SendAlertResolvedEmail(context.Background(), "ops@example.com", AlertTemplateData{
		ProjectID: 1, ProjectName: "Agrafa", AlertTitle: "✓ Landing has recovered",
		AlertMessage: "HTTP health checks are passing again. The service is responding normally.", RuleType: "service_unhealthy", RuleLabel: "Service unhealthy",
		Severity: "critical", Status: "resolved", ResourceType: "service", ServiceName: "Landing", Duration: "5m",
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC), ResolvedAt: &resolvedAt, ResourceURL: "https://app.agrafa.test/services/9",
	})
	if err != nil {
		t.Fatalf("SendAlertResolvedEmail returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if !strings.Contains(message.HTML, "RESOLVED") || !strings.Contains(message.HTML, `data-title-icon="resolved"`) || !strings.Contains(message.HTML, "Landing has recovered") {
		t.Fatalf("expected rendered HTML to contain heading, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "RESOLVED — Landing has recovered") || !strings.Contains(message.Text, "Downtime: 5m") {
		t.Fatalf("expected rendered text to contain heading, got %q", message.Text)
	}

	if !strings.Contains(message.HTML, "Landing has recovered") {
		t.Fatalf("expected rendered HTML to contain alert title, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "Landing has recovered") {
		t.Fatalf("expected rendered text to contain alert title, got %q", message.Text)
	}
}

func TestRendererRenderTextTriggeredTemplate(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	output, err := renderer.RenderText("alert_triggered.txt", AlertTemplateData{
		ProjectID: 1, ProjectName: "Agrafa", AlertTitle: "⚠ web-01 is offline", AlertMessage: "Agrafa stopped receiving heartbeats from this node.",
		RuleType: "node_offline", RuleLabel: "Node offline", Severity: "critical", Status: "active", ResourceType: "node", NodeName: "web-01",
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RenderText returned error: %v", err)
	}

	if !strings.Contains(output, "CRITICAL — web-01 is offline") || !strings.Contains(output, "Rule: Node offline") || !strings.Contains(output, "Project: Agrafa") {
		t.Fatalf("unexpected rendered text output: %q", output)
	}
}

func TestRendererRenderTextResolvedTemplate(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	resolvedAt := time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)
	output, err := renderer.RenderText("alert_resolved.txt", AlertTemplateData{
		ProjectID: 1, ProjectName: "Agrafa", AlertTitle: "✓ Landing has recovered", AlertMessage: "HTTP health checks are passing again.",
		RuleType: "service_unhealthy", RuleLabel: "Service unhealthy", Severity: "critical", Status: "resolved", ResourceType: "service", ServiceName: "Landing", Duration: "5m",
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC), ResolvedAt: &resolvedAt,
	})
	if err != nil {
		t.Fatalf("RenderText returned error: %v", err)
	}

	if !strings.Contains(output, "RESOLVED — Landing has recovered") || !strings.Contains(output, "Recovered at:") || !strings.Contains(output, "Downtime: 5m") {
		t.Fatalf("unexpected rendered text output: %q", output)
	}
}

func TestBuildSecurityFromAddress(t *testing.T) {
	t.Parallel()

	from := BuildSecurityFromAddress("email.agrafa.co")
	if from != "Agrafa Security <security@email.agrafa.co>" {
		t.Fatalf("from = %q", from)
	}
}

func TestSendVerifyEmailRendersSecurityTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Security <security@example.com>")

	err := service.SendVerifyEmail(context.Background(), "alice@example.com", "Alice", "https://app.agrafa.co/verify-email?token=abc123")
	if err != nil {
		t.Fatalf("SendVerifyEmail returned error: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if message.From != "Agrafa Security <security@example.com>" {
		t.Fatalf("from = %q", message.From)
	}
	if !strings.Contains(message.HTML, "Verify your email") || !strings.Contains(message.HTML, "https://app.agrafa.co/verify-email?token=abc123") {
		t.Fatalf("unexpected rendered HTML: %q", message.HTML)
	}
	if !strings.Contains(message.Text, "finish setting up your Agrafa account") {
		t.Fatalf("unexpected rendered text: %q", message.Text)
	}
}

func TestSendPasswordResetEmailRendersSecurityTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Security <security@example.com>")

	err := service.SendPasswordResetEmail(context.Background(), "alice@example.com", "Alice", "https://app.agrafa.co/reset-password?token=abc123")
	if err != nil {
		t.Fatalf("SendPasswordResetEmail returned error: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if message.From != "Agrafa Security <security@example.com>" {
		t.Fatalf("from = %q", message.From)
	}
	if !strings.Contains(message.HTML, "Reset your password") || !strings.Contains(message.HTML, "https://app.agrafa.co/reset-password?token=abc123") {
		t.Fatalf("unexpected rendered HTML: %q", message.HTML)
	}
	if !strings.Contains(message.Text, "Use this link to reset your Agrafa password") {
		t.Fatalf("unexpected rendered text: %q", message.Text)
	}
}

func TestBuildNotificationsFromAddress(t *testing.T) {
	t.Parallel()

	from := BuildNotificationsFromAddress("email.agrafa.co")
	if from != "Agrafa Notifications <notifications@email.agrafa.co>" {
		t.Fatalf("from = %q", from)
	}
}

func TestSendProjectInviteRendersInviteTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Notifications <notifications@example.com>")

	err := service.SendProjectInvite(context.Background(), "alice@example.com", ProjectInviteTemplateData{
		ProjectName: "Agrafa Team",
		Role:        "viewer",
		InviterName: "Alice",
		AcceptURL:   "https://app.agrafa.co/invite?token=abc123",
	})
	if err != nil {
		t.Fatalf("SendProjectInvite returned error: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if message.From != "Agrafa Notifications <notifications@example.com>" {
		t.Fatalf("from = %q", message.From)
	}
	if !strings.Contains(message.HTML, "Accept invitation") || !strings.Contains(message.HTML, "https://app.agrafa.co/invite?token=abc123") {
		t.Fatalf("unexpected rendered HTML: %q", message.HTML)
	}
	if !strings.Contains(message.Text, "Access level: viewer") {
		t.Fatalf("unexpected rendered text: %q", message.Text)
	}
}

func TestSendNotificationRecipientTestEmailRendersTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Notifications <notifications@example.com>")

	err := service.SendNotificationRecipientTestEmail(context.Background(), "ops@example.com", NotificationRecipientTestTemplateData{
		ProjectName: "Agrafa Team",
		ProjectID:   1,
		Recipient:   "ops@example.com",
		SentAt:      time.Date(2026, time.April, 11, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SendNotificationRecipientTestEmail returned error: %v", err)
	}
	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if message.From != "Agrafa Notifications <notifications@example.com>" {
		t.Fatalf("from = %q", message.From)
	}
	if !strings.Contains(message.Subject, "Agrafa Team") {
		t.Fatalf("subject = %q", message.Subject)
	}
	if !strings.Contains(message.HTML, "Alert email delivery is working") || !strings.Contains(message.HTML, "ops@example.com") {
		t.Fatalf("unexpected rendered HTML: %q", message.HTML)
	}
	if !strings.Contains(message.Text, "AGRAFA — NOTIFICATION TEST") || !strings.Contains(message.Text, "Project: Agrafa Team") {
		t.Fatalf("unexpected rendered text: %q", message.Text)
	}
}

func TestSendNotificationRecipientTestEmailWithoutRendererReturnsError(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(nil, sender, "Agrafa Notifications <notifications@example.com>")

	err := service.SendNotificationRecipientTestEmail(context.Background(), "ops@example.com", NotificationRecipientTestTemplateData{
		ProjectName: "Agrafa Team",
		ProjectID:   1,
		Recipient:   "ops@example.com",
		SentAt:      time.Date(2026, time.April, 11, 18, 0, 0, 0, time.UTC),
	})
	if err == nil {
		t.Fatal("SendNotificationRecipientTestEmail() error = nil")
	}
	if !strings.Contains(err.Error(), "email renderer is not configured") {
		t.Fatalf("unexpected error: %v", err)
	}
}
