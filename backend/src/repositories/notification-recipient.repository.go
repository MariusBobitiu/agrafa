package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type NotificationRecipientRepository struct {
	queries *generated.Queries
}

func NewNotificationRecipientRepository(queries *generated.Queries) *NotificationRecipientRepository {
	return &NotificationRecipientRepository{queries: queries}
}

func (r *NotificationRecipientRepository) Create(ctx context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error) {
	return r.queries.CreateNotificationRecipient(ctx, params)
}

func (r *NotificationRecipientRepository) GetByID(ctx context.Context, id int64) (generated.NotificationRecipient, error) {
	return r.queries.GetNotificationRecipientByID(ctx, id)
}

func (r *NotificationRecipientRepository) List(ctx context.Context, projectID *int64) ([]generated.NotificationRecipient, error) {
	params := generated.ListNotificationRecipientsParams{}

	if projectID != nil {
		params.Column1 = true
		params.ProjectID = *projectID
	}

	return r.queries.ListNotificationRecipients(ctx, params)
}

func (r *NotificationRecipientRepository) ListByProjectAndChannel(ctx context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error) {
	return r.queries.ListNotificationRecipientsByProjectAndChannel(ctx, generated.ListNotificationRecipientsByProjectAndChannelParams{
		ProjectID:   projectID,
		ChannelType: channelType,
	})
}

func (r *NotificationRecipientRepository) UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.NotificationRecipient, error) {
	return r.queries.UpdateNotificationRecipientEnabled(ctx, generated.UpdateNotificationRecipientEnabledParams{
		ID:        id,
		IsEnabled: isEnabled,
	})
}

func (r *NotificationRecipientRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return r.queries.DeleteNotificationRecipientByID(ctx, id)
}
