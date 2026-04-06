package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type NotificationDeliveryRepository struct {
	queries *generated.Queries
}

func NewNotificationDeliveryRepository(queries *generated.Queries) *NotificationDeliveryRepository {
	return &NotificationDeliveryRepository{queries: queries}
}

func (r *NotificationDeliveryRepository) Create(ctx context.Context, params generated.CreateNotificationDeliveryParams) (generated.NotificationDelivery, error) {
	return r.queries.CreateNotificationDelivery(ctx, params)
}

func (r *NotificationDeliveryRepository) List(ctx context.Context, projectID *int64, status *string, limit int32) ([]generated.NotificationDelivery, error) {
	params := generated.ListNotificationDeliveriesParams{
		Limit: limit,
	}

	if projectID != nil {
		params.Column1 = true
		params.ProjectID = *projectID
	}

	if status != nil {
		params.Column3 = true
		params.Status = *status
	}

	return r.queries.ListNotificationDeliveries(ctx, params)
}
