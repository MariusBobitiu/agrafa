package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type AlertInstanceRepository struct {
	queries *generated.Queries
}

func NewAlertInstanceRepository(queries *generated.Queries) *AlertInstanceRepository {
	return &AlertInstanceRepository{queries: queries}
}

func (r *AlertInstanceRepository) FindActiveByRuleID(ctx context.Context, ruleID int64) (generated.AlertInstance, error) {
	return r.queries.FindActiveAlertInstanceByRuleID(ctx, ruleID)
}

func (r *AlertInstanceRepository) Create(ctx context.Context, params generated.CreateAlertInstanceParams) (generated.AlertInstance, error) {
	return r.queries.CreateAlertInstance(ctx, params)
}

func (r *AlertInstanceRepository) Resolve(ctx context.Context, id int64, resolvedAt time.Time) (generated.AlertInstance, error) {
	return r.queries.ResolveAlertInstance(ctx, generated.ResolveAlertInstanceParams{
		ID:         id,
		ResolvedAt: sql.NullTime{Time: resolvedAt, Valid: true},
	})
}

func (r *AlertInstanceRepository) List(ctx context.Context, projectID *int64, status *string, limit int32) ([]generated.AlertInstance, error) {
	params := generated.ListAlertInstancesParams{
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

	return r.queries.ListAlertInstances(ctx, params)
}

func (r *AlertInstanceRepository) ListByNodeID(ctx context.Context, nodeID int64, status *string, limit int32) ([]generated.AlertInstance, error) {
	params := generated.ListAlertInstancesByNodeAndStatusParams{
		NodeID: sql.NullInt64{Int64: nodeID, Valid: true},
		Limit:  limit,
	}

	if status != nil {
		params.Column2 = true
		params.Status = *status
	}

	return r.queries.ListAlertInstancesByNodeAndStatus(ctx, params)
}

func (r *AlertInstanceRepository) ListActiveDetailsByServiceID(ctx context.Context, serviceID int64) ([]generated.ListActiveAlertDetailsByServiceIDRow, error) {
	return r.queries.ListActiveAlertDetailsByServiceID(ctx, sql.NullInt64{Int64: serviceID, Valid: true})
}

func (r *AlertInstanceRepository) ListActiveCountsByNode(ctx context.Context, projectID *int64) ([]generated.ListActiveAlertCountsByNodeRow, error) {
	if projectID != nil {
		rows, err := r.queries.ListActiveAlertCountsByNodeByProject(ctx, *projectID)
		if err != nil {
			return nil, err
		}

		items := make([]generated.ListActiveAlertCountsByNodeRow, 0, len(rows))
		for _, row := range rows {
			items = append(items, generated.ListActiveAlertCountsByNodeRow{
				NodeID:           row.NodeID,
				ActiveAlertCount: row.ActiveAlertCount,
			})
		}

		return items, nil
	}

	return r.queries.ListActiveAlertCountsByNode(ctx)
}

func (r *AlertInstanceRepository) ListActiveCountsByService(ctx context.Context, projectID *int64) ([]generated.ListActiveAlertCountsByServiceRow, error) {
	if projectID != nil {
		rows, err := r.queries.ListActiveAlertCountsByServiceByProject(ctx, *projectID)
		if err != nil {
			return nil, err
		}

		items := make([]generated.ListActiveAlertCountsByServiceRow, 0, len(rows))
		for _, row := range rows {
			items = append(items, generated.ListActiveAlertCountsByServiceRow{
				ServiceID:        row.ServiceID,
				ActiveAlertCount: row.ActiveAlertCount,
			})
		}

		return items, nil
	}

	return r.queries.ListActiveAlertCountsByService(ctx)
}

func (r *AlertInstanceRepository) CountActiveByServiceID(ctx context.Context, serviceID int64) (int64, error) {
	return r.queries.CountActiveAlertInstancesByServiceID(ctx, sql.NullInt64{Int64: serviceID, Valid: true})
}

func (r *AlertInstanceRepository) CountActiveByNodeID(ctx context.Context, nodeID int64) (int64, error) {
	return r.queries.CountActiveAlertInstancesByNodeID(ctx, sql.NullInt64{Int64: nodeID, Valid: true})
}

func (r *AlertInstanceRepository) ListActiveCountsByServiceForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.ListActiveAlertCountsByServiceRow, error) {
	rows, err := r.queries.ListActiveAlertCountsByServiceForRead(ctx, generated.ListActiveAlertCountsByServiceForReadParams{
		HasProjectID: filters.ProjectID != nil,
		ProjectID:    derefInt64(filters.ProjectID),
		HasNodeID:    filters.NodeID != nil,
		NodeID:       derefInt64(filters.NodeID),
		HasStatus:    filters.Status != nil,
		Status:       derefString(filters.Status),
	})
	if err != nil {
		return nil, err
	}

	items := make([]generated.ListActiveAlertCountsByServiceRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, generated.ListActiveAlertCountsByServiceRow{
			ServiceID:        row.ServiceID,
			ActiveAlertCount: row.ActiveAlertCount,
		})
	}

	return items, nil
}
