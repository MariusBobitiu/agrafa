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
		ProjectID:    1,
		ProjectName:  "Agrafa",
		AlertTitle:   "Node 5 is offline",
		AlertMessage: "Node 5 is currently offline.",
		RuleType:     "node_offline",
		Status:       "active",
		TriggeredAt:  time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("SendAlertTriggeredEmail returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if !strings.Contains(message.HTML, "Alert triggered") {
		t.Fatalf("expected rendered HTML to contain heading, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "AGRAFA ALERT TRIGGERED") {
		t.Fatalf("expected rendered text to contain heading, got %q", message.Text)
	}

	if !strings.Contains(message.HTML, "Node 5 is offline") {
		t.Fatalf("expected rendered HTML to contain alert title, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "Node 5 is offline") {
		t.Fatalf("expected rendered text to contain alert title, got %q", message.Text)
	}
}

func TestSendAlertResolvedEmailRendersHTMLTemplate(t *testing.T) {
	t.Parallel()

	sender := &fakeSender{}
	service := NewService(NewRenderer(), sender, "Agrafa Alerts <alerts@example.com>")
	resolvedAt := time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)

	err := service.SendAlertResolvedEmail(context.Background(), "ops@example.com", AlertTemplateData{
		ProjectID:    1,
		ProjectName:  "Agrafa",
		AlertTitle:   "Service 9 is unhealthy",
		AlertMessage: "Service 9 is currently unhealthy.",
		RuleType:     "service_unhealthy",
		Status:       "resolved",
		TriggeredAt:  time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		ResolvedAt:   &resolvedAt,
	})
	if err != nil {
		t.Fatalf("SendAlertResolvedEmail returned error: %v", err)
	}

	if len(sender.messages) != 1 {
		t.Fatalf("expected 1 email, got %d", len(sender.messages))
	}

	message := sender.messages[0]
	if !strings.Contains(message.HTML, "Alert resolved") {
		t.Fatalf("expected rendered HTML to contain heading, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "AGRAFA ALERT RESOLVED") {
		t.Fatalf("expected rendered text to contain heading, got %q", message.Text)
	}

	if !strings.Contains(message.HTML, "Service 9 is unhealthy") {
		t.Fatalf("expected rendered HTML to contain alert title, got %q", message.HTML)
	}

	if !strings.Contains(message.Text, "Service 9 is unhealthy") {
		t.Fatalf("expected rendered text to contain alert title, got %q", message.Text)
	}
}

func TestRendererRenderTextTriggeredTemplate(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	output, err := renderer.RenderText("alert_triggered.txt", AlertTemplateData{
		ProjectID:    1,
		ProjectName:  "Agrafa",
		AlertTitle:   "Node 5 is offline",
		AlertMessage: "Node 5 is currently offline.",
		RuleType:     "node_offline",
		Status:       "active",
		TriggeredAt:  time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("RenderText returned error: %v", err)
	}

	if !strings.Contains(output, "AGRAFA ALERT TRIGGERED") || !strings.Contains(output, "Project: Agrafa") {
		t.Fatalf("unexpected rendered text output: %q", output)
	}
}

func TestRendererRenderTextResolvedTemplate(t *testing.T) {
	t.Parallel()

	renderer := NewRenderer()
	resolvedAt := time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC)
	output, err := renderer.RenderText("alert_resolved.txt", AlertTemplateData{
		ProjectID:    1,
		ProjectName:  "Agrafa",
		AlertTitle:   "Service 9 is unhealthy",
		AlertMessage: "Service 9 is currently unhealthy.",
		RuleType:     "service_unhealthy",
		Status:       "resolved",
		TriggeredAt:  time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		ResolvedAt:   &resolvedAt,
	})
	if err != nil {
		t.Fatalf("RenderText returned error: %v", err)
	}

	if !strings.Contains(output, "AGRAFA ALERT RESOLVED") || !strings.Contains(output, "Resolved at:") {
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
	if !strings.Contains(message.Text, "secure your Agrafa account") {
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
	if !strings.Contains(message.Text, "Reset your Agrafa password") {
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
	if !strings.Contains(message.Text, "Role: viewer") {
		t.Fatalf("unexpected rendered text: %q", message.Text)
	}
}
