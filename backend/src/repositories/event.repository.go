package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type EventRepository struct {
	queries *generated.Queries
}

func NewEventRepository(queries *generated.Queries) *EventRepository {
	return &EventRepository{queries: queries}
}

func (r *EventRepository) Create(ctx context.Context, params generated.CreateEventParams) (generated.Event, error) {
	return r.queries.CreateEvent(ctx, params)
}

func (r *EventRepository) List(ctx context.Context, limit int32) ([]generated.Event, error) {
	return r.queries.ListEvents(ctx, limit)
}

func (r *EventRepository) ListByProject(ctx context.Context, projectID int64, limit int32) ([]generated.Event, error) {
	return r.queries.ListEventsByProject(ctx, generated.ListEventsByProjectParams{
		ProjectID: projectID,
		Limit:     limit,
	})
}

func (r *EventRepository) ListRecentAlertEvents(ctx context.Context, limit int32, projectID *int64) ([]generated.Event, error) {
	if projectID != nil {
		return r.queries.ListRecentAlertEventsByProject(ctx, generated.ListRecentAlertEventsByProjectParams{
			ProjectID: *projectID,
			Limit:     limit,
		})
	}

	return r.queries.ListRecentAlertEvents(ctx, limit)
}
