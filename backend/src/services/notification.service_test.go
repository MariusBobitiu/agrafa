package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationDispatchRepo struct {
	recipients []generated.NotificationRecipient
}

func (r *fakeNotificationDispatchRepo) ListByProjectAndChannel(_ context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error) {
	items := make([]generated.NotificationRecipient, 0, len(r.recipients))
	for _, recipient := range r.recipients {
		if recipient.ProjectID == projectID && recipient.ChannelType == channelType {
			items = append(items, recipient)
		}
	}

	return items, nil
}

type fakeNotificationProjectLookupRepo struct {
	projects map[int64]generated.Project
}

func (r *fakeNotificationProjectLookupRepo) GetByID(_ context.Context, id int64) (generated.Project, error) {
	project, ok := r.projects[id]
	if !ok {
		return generated.Project{}, sql.ErrNoRows
	}

	return project, nil
}

type fakeAlertEmailService struct {
	triggeredRecipients []string
	resolvedRecipients  []string
	failFor             map[string]error
}

type fakeNotificationDeliveryRecorder struct {
	records []types.CreateNotificationDeliveryInput
}

func (r *fakeNotificationDeliveryRecorder) Record(_ context.Context, input types.CreateNotificationDeliveryInput) error {
	r.records = append(r.records, input)
	return nil
}

func (s *fakeAlertEmailService) SendAlertTriggeredEmail(_ context.Context, to string, _ emailpkg.AlertTemplateData) error {
	s.triggeredRecipients = append(s.triggeredRecipients, to)
	if err, ok := s.failFor[to]; ok {
		return err
	}

	return nil
}

func (s *fakeAlertEmailService) SendAlertResolvedEmail(_ context.Context, to string, _ emailpkg.AlertTemplateData) error {
	s.resolvedRecipients = append(s.resolvedRecipients, to)
	if err, ok := s.failFor[to]; ok {
		return err
	}

	return nil
}

func TestNotificationServiceSendsOnlyToEnabledRecipients(t *testing.T) {
	t.Parallel()

	emailService := &fakeAlertEmailService{}
	deliveryRecorder := &fakeNotificationDeliveryRecorder{}
	service := &NotificationService{
		notificationRecipientRepo: &fakeNotificationDispatchRepo{
			recipients: []generated.NotificationRecipient{
				{ID: 1, ProjectID: 1, ChannelType: types.NotificationChannelTypeEmail, Target: "enabled@example.com", IsEnabled: true},
				{ID: 2, ProjectID: 1, ChannelType: types.NotificationChannelTypeEmail, Target: "disabled@example.com", IsEnabled: false},
			},
		},
		projectRepo: &fakeNotificationProjectLookupRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
		notificationDeliverySvc: deliveryRecorder,
		emailService:            emailService,
	}

	err := service.NotifyAlertTriggered(context.Background(), generated.AlertRule{
		ID:        1,
		ProjectID: 1,
		RuleType:  types.AlertRuleTypeNodeOffline,
	}, generated.AlertInstance{
		ID:          10,
		ProjectID:   1,
		NodeID:      sql.NullInt64{Int64: 5, Valid: true},
		Status:      types.AlertStatusActive,
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		Title:       "Node 5 is offline",
		Message:     "Node 5 is currently offline.",
	})
	if err != nil {
		t.Fatalf("NotifyAlertTriggered returned error: %v", err)
	}

	if len(emailService.triggeredRecipients) != 1 {
		t.Fatalf("expected 1 triggered email attempt, got %d", len(emailService.triggeredRecipients))
	}

	if emailService.triggeredRecipients[0] != "enabled@example.com" {
		t.Fatalf("unexpected recipient %q", emailService.triggeredRecipients[0])
	}

	if len(deliveryRecorder.records) != 1 {
		t.Fatalf("expected 1 delivery record, got %d", len(deliveryRecorder.records))
	}

	if deliveryRecorder.records[0].Status != types.NotificationDeliveryStatusSent {
		t.Fatalf("expected sent delivery status, got %q", deliveryRecorder.records[0].Status)
	}
}

func TestNotificationServiceRecordsFailedDeliveryAfterEmailFailure(t *testing.T) {
	t.Parallel()

	emailService := &fakeAlertEmailService{
		failFor: map[string]error{
			"fail@example.com": errors.New("smtp-ish failure"),
		},
	}
	deliveryRecorder := &fakeNotificationDeliveryRecorder{}
	service := &NotificationService{
		notificationRecipientRepo: &fakeNotificationDispatchRepo{
			recipients: []generated.NotificationRecipient{
				{ID: 1, ProjectID: 1, ChannelType: types.NotificationChannelTypeEmail, Target: "fail@example.com", IsEnabled: true},
				{ID: 2, ProjectID: 1, ChannelType: types.NotificationChannelTypeEmail, Target: "ok@example.com", IsEnabled: true},
			},
		},
		projectRepo: &fakeNotificationProjectLookupRepo{
			projects: map[int64]generated.Project{
				1: {ID: 1, Name: "Agrafa"},
			},
		},
		notificationDeliverySvc: deliveryRecorder,
		emailService:            emailService,
	}

	err := service.NotifyAlertResolved(context.Background(), generated.AlertRule{
		ID:        1,
		ProjectID: 1,
		RuleType:  types.AlertRuleTypeServiceUnhealthy,
	}, generated.AlertInstance{
		ID:          10,
		ProjectID:   1,
		ServiceID:   sql.NullInt64{Int64: 9, Valid: true},
		Status:      types.AlertStatusResolved,
		TriggeredAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		ResolvedAt:  sql.NullTime{Time: time.Date(2026, time.April, 5, 12, 5, 0, 0, time.UTC), Valid: true},
		Title:       "Service 9 is unhealthy",
		Message:     "Service 9 is currently unhealthy.",
	})
	if err != nil {
		t.Fatalf("NotifyAlertResolved returned error: %v", err)
	}

	if len(emailService.resolvedRecipients) != 2 {
		t.Fatalf("expected 2 resolved email attempts, got %d", len(emailService.resolvedRecipients))
	}

	if len(deliveryRecorder.records) != 2 {
		t.Fatalf("expected 2 delivery records, got %d", len(deliveryRecorder.records))
	}

	if deliveryRecorder.records[0].Status != types.NotificationDeliveryStatusFailed {
		t.Fatalf("expected first delivery to be failed, got %q", deliveryRecorder.records[0].Status)
	}

	if deliveryRecorder.records[0].ErrorMessage == nil || *deliveryRecorder.records[0].ErrorMessage == "" {
		t.Fatal("expected failed delivery to include error message")
	}

	if deliveryRecorder.records[1].Status != types.NotificationDeliveryStatusSent {
		t.Fatalf("expected second delivery to be sent, got %q", deliveryRecorder.records[1].Status)
	}
}
