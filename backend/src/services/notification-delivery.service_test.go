package services

import (
	"context"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNotificationDeliveryRepo struct {
	rows []generated.NotificationDelivery
}

func (r *fakeNotificationDeliveryRepo) Create(_ context.Context, params generated.CreateNotificationDeliveryParams) (generated.NotificationDelivery, error) {
	row := generated.NotificationDelivery{
		ID:                      int64(len(r.rows) + 1),
		ProjectID:               params.ProjectID,
		NotificationRecipientID: params.NotificationRecipientID,
		AlertInstanceID:         params.AlertInstanceID,
		ChannelType:             params.ChannelType,
		Target:                  params.Target,
		EventType:               params.EventType,
		Status:                  params.Status,
		ErrorMessage:            params.ErrorMessage,
		SentAt:                  params.SentAt,
		CreatedAt:               params.SentAt,
	}
	r.rows = append(r.rows, row)
	return row, nil
}

func (r *fakeNotificationDeliveryRepo) List(_ context.Context, _ *int64, _ *string, _ int32) ([]generated.NotificationDelivery, error) {
	return r.rows, nil
}

func TestNotificationDeliveryServiceRejectsInvalidStatusFilter(t *testing.T) {
	t.Parallel()

	service := &NotificationDeliveryService{
		notificationDeliveryRepo: &fakeNotificationDeliveryRepo{},
	}

	status := "unknown"
	_, err := service.List(context.Background(), nil, &status, 50)
	if err != types.ErrInvalidNotificationDeliveryStatus {
		t.Fatalf("expected ErrInvalidNotificationDeliveryStatus, got %v", err)
	}
}

func TestNotificationDeliveryServiceRecordStoresDelivery(t *testing.T) {
	t.Parallel()

	repo := &fakeNotificationDeliveryRepo{}
	service := &NotificationDeliveryService{
		notificationDeliveryRepo: repo,
	}

	err := service.Record(context.Background(), types.CreateNotificationDeliveryInput{
		ProjectID:   1,
		ChannelType: types.NotificationChannelTypeEmail,
		Target:      "ops@example.com",
		EventType:   types.EventTypeAlertTriggered,
		Status:      types.NotificationDeliveryStatusSent,
		SentAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}

	if len(repo.rows) != 1 {
		t.Fatalf("expected 1 delivery row, got %d", len(repo.rows))
	}

	if repo.rows[0].Status != types.NotificationDeliveryStatusSent {
		t.Fatalf("expected sent status, got %q", repo.rows[0].Status)
	}
}
