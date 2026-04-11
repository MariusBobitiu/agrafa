package services

import (
	"context"
	"testing"
	"time"

	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
)

type fakeRuntimeSender struct {
	messages []emailpkg.Message
}

func (s *fakeRuntimeSender) Send(_ context.Context, message emailpkg.Message) error {
	s.messages = append(s.messages, message)
	return nil
}

type fakeEmailRuntimeConfigResolver struct {
	configs []EmailRuntimeConfig
	err     error
	index   int
}

func (r *fakeEmailRuntimeConfigResolver) ResolveEmailConfig(_ context.Context) (EmailRuntimeConfig, error) {
	if r.err != nil {
		return EmailRuntimeConfig{}, r.err
	}
	if len(r.configs) == 0 {
		return EmailRuntimeConfig{}, nil
	}

	current := r.configs[r.index]
	if r.index < len(r.configs)-1 {
		r.index++
	}

	return current, nil
}

func TestRuntimeEmailProviderNotificationsRebuildsServiceFromLatestConfig(t *testing.T) {
	t.Parallel()

	resolver := &fakeEmailRuntimeConfigResolver{
		configs: []EmailRuntimeConfig{
			{
				IsAvailable:       true,
				Provider:          "resend",
				ResendAPIKey:      "re_first",
				NotificationsFrom: "Agrafa Notifications <notifications@first.example.com>",
			},
			{
				IsAvailable:       true,
				Provider:          "resend",
				ResendAPIKey:      "re_second",
				NotificationsFrom: "Agrafa Notifications <notifications@second.example.com>",
			},
		},
	}

	senders := make([]*fakeRuntimeSender, 0, 2)
	provider := NewRuntimeEmailProvider(resolver)
	provider.senderFactory = func(config EmailRuntimeConfig) (emailpkg.Sender, error) {
		sender := &fakeRuntimeSender{}
		senders = append(senders, sender)
		return sender, nil
	}

	firstService, err := provider.Notifications(context.Background())
	if err != nil {
		t.Fatalf("Notifications() error = %v", err)
	}
	secondService, err := provider.Notifications(context.Background())
	if err != nil {
		t.Fatalf("Notifications() second error = %v", err)
	}

	firstErr := firstService.SendNotificationRecipientTestEmail(context.Background(), "ops@example.com", emailpkg.NotificationRecipientTestTemplateData{
		ProjectName: "First Project",
		ProjectID:   1,
		Recipient:   "ops@example.com",
		SentAt:      time.Date(2026, time.April, 11, 18, 0, 0, 0, time.UTC),
	})
	if firstErr != nil {
		t.Fatalf("first send error = %v", firstErr)
	}

	secondErr := secondService.SendNotificationRecipientTestEmail(context.Background(), "ops@example.com", emailpkg.NotificationRecipientTestTemplateData{
		ProjectName: "Second Project",
		ProjectID:   1,
		Recipient:   "ops@example.com",
		SentAt:      time.Date(2026, time.April, 11, 18, 5, 0, 0, time.UTC),
	})
	if secondErr != nil {
		t.Fatalf("second send error = %v", secondErr)
	}

	if len(senders) != 2 {
		t.Fatalf("len(senders) = %d, want 2", len(senders))
	}
	if got := senders[0].messages[0].From; got != "Agrafa Notifications <notifications@first.example.com>" {
		t.Fatalf("first from = %q", got)
	}
	if got := senders[1].messages[0].From; got != "Agrafa Notifications <notifications@second.example.com>" {
		t.Fatalf("second from = %q", got)
	}
}

func TestRuntimeEmailProviderNotificationsReturnsNilWhenUnavailable(t *testing.T) {
	t.Parallel()

	provider := NewRuntimeEmailProvider(&fakeEmailRuntimeConfigResolver{
		configs: []EmailRuntimeConfig{
			{
				IsAvailable:       false,
				Provider:          "resend",
				NotificationsFrom: "Agrafa Notifications <notifications@example.com>",
			},
		},
	})

	emailService, err := provider.Notifications(context.Background())
	if err != nil {
		t.Fatalf("Notifications() error = %v", err)
	}
	if emailService != nil {
		t.Fatal("Notifications() expected nil service when email is unavailable")
	}
}
