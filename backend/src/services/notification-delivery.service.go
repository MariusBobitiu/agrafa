package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type notificationDeliveryRepository interface {
	Create(ctx context.Context, params generated.CreateNotificationDeliveryParams) (generated.NotificationDelivery, error)
	List(ctx context.Context, projectID *int64, status *string, limit int32) ([]generated.NotificationDelivery, error)
}

type NotificationDeliveryService struct {
	notificationDeliveryRepo notificationDeliveryRepository
}

func NewNotificationDeliveryService(notificationDeliveryRepo *repositories.NotificationDeliveryRepository) *NotificationDeliveryService {
	return &NotificationDeliveryService{notificationDeliveryRepo: notificationDeliveryRepo}
}

func (s *NotificationDeliveryService) Record(ctx context.Context, input types.CreateNotificationDeliveryInput) error {
	if input.SentAt.IsZero() {
		input.SentAt = time.Now().UTC()
	}

	_, err := s.notificationDeliveryRepo.Create(ctx, generated.CreateNotificationDeliveryParams{
		ProjectID:               input.ProjectID,
		NotificationRecipientID: toNullInt64(input.NotificationRecipientID),
		AlertInstanceID:         toNullInt64(input.AlertInstanceID),
		ChannelType:             input.ChannelType,
		Target:                  input.Target,
		EventType:               input.EventType,
		Status:                  input.Status,
		ErrorMessage:            toNullString(input.ErrorMessage),
		SentAt:                  input.SentAt,
	})
	if err != nil {
		return fmt.Errorf("create notification delivery: %w", err)
	}

	return nil
}

func (s *NotificationDeliveryService) List(ctx context.Context, projectID *int64, status *string, limit int32) ([]types.NotificationDeliveryReadData, error) {
	if limit <= 0 {
		limit = 50
	}

	if status != nil && *status != types.NotificationDeliveryStatusSent && *status != types.NotificationDeliveryStatusFailed {
		return nil, types.ErrInvalidNotificationDeliveryStatus
	}

	rows, err := s.notificationDeliveryRepo.List(ctx, projectID, status, limit)
	if err != nil {
		return nil, fmt.Errorf("list notification deliveries: %w", err)
	}

	return mapNotificationDeliveries(rows), nil
}

func toNullInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: *value, Valid: true}
}

func toNullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}

	return sql.NullString{String: *value, Valid: true}
}
