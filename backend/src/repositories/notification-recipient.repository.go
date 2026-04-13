package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type NotificationRecipientRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewNotificationRecipientRepository(db *sql.DB, queries *generated.Queries) *NotificationRecipientRepository {
	return &NotificationRecipientRepository{
		db:      db,
		queries: queries,
	}
}

func (r *NotificationRecipientRepository) Create(ctx context.Context, params generated.CreateNotificationRecipientParams) (generated.NotificationRecipient, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.NotificationRecipient, error) {
		return queries.CreateNotificationRecipient(ctx, params)
	})
}

func (r *NotificationRecipientRepository) GetByID(ctx context.Context, id int64) (generated.NotificationRecipient, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.NotificationRecipient, error) {
		return queries.GetNotificationRecipientByID(ctx, id)
	})
}

func (r *NotificationRecipientRepository) List(ctx context.Context, projectID *int64) ([]generated.NotificationRecipient, error) {
	params := generated.ListNotificationRecipientsParams{}

	if projectID != nil {
		params.Column1 = true
		params.ProjectID = *projectID
	}

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.NotificationRecipient, error) {
		return queries.ListNotificationRecipients(ctx, params)
	})
}

func (r *NotificationRecipientRepository) ListByProjectAndChannel(ctx context.Context, projectID int64, channelType string) ([]generated.NotificationRecipient, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.NotificationRecipient, error) {
		return queries.ListNotificationRecipientsByProjectAndChannel(ctx, generated.ListNotificationRecipientsByProjectAndChannelParams{
			ProjectID:   projectID,
			ChannelType: channelType,
		})
	})
}

func (r *NotificationRecipientRepository) UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.NotificationRecipient, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.NotificationRecipient, error) {
		return queries.UpdateNotificationRecipientEnabled(ctx, generated.UpdateNotificationRecipientEnabledParams{
			ID:        id,
			IsEnabled: isEnabled,
		})
	})
}

func (r *NotificationRecipientRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.DeleteNotificationRecipientByID(ctx, id)
	})
}
