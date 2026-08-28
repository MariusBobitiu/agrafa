package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type AlertInstanceRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

type AlertReadRow struct {
	ID             int64
	AlertRuleID    int64
	ProjectID      int64
	NodeID         sql.NullInt64
	NodeName       sql.NullString
	NodeIdentifier sql.NullString
	ServiceID      sql.NullInt64
	ServiceName    sql.NullString
	RuleType       string
	Severity       string
	Status         string
	TriggeredAt    time.Time
	ResolvedAt     sql.NullTime
	Title          string
	Message        string
	CreatedAt      time.Time
}

func NewAlertInstanceRepository(db *sql.DB, queries *generated.Queries) *AlertInstanceRepository {
	return &AlertInstanceRepository{
		db:      db,
		queries: queries,
	}
}

func (r *AlertInstanceRepository) FindActiveByRuleAndTarget(
	ctx context.Context,
	ruleID int64,
	nodeID sql.NullInt64,
	serviceID sql.NullInt64,
) (generated.AlertInstance, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertInstance, error) {
		return queries.FindActiveAlertInstanceByRuleAndTarget(ctx, generated.FindActiveAlertInstanceByRuleAndTargetParams{
			AlertRuleID: ruleID,
			NodeID:      nodeID,
			ServiceID:   serviceID,
		})
	})
}

func (r *AlertInstanceRepository) ListActiveByRuleID(ctx context.Context, ruleID int64) ([]generated.AlertInstance, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.AlertInstance, error) {
		return queries.ListActiveAlertInstancesByRuleID(ctx, ruleID)
	})
}

func (r *AlertInstanceRepository) Create(ctx context.Context, params generated.CreateAlertInstanceParams) (generated.AlertInstance, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertInstance, error) {
		return queries.CreateAlertInstance(ctx, params)
	})
}

func (r *AlertInstanceRepository) Resolve(ctx context.Context, id int64, resolvedAt time.Time) (generated.AlertInstance, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertInstance, error) {
		return queries.ResolveAlertInstance(ctx, generated.ResolveAlertInstanceParams{
			ID:         id,
			ResolvedAt: sql.NullTime{Time: resolvedAt, Valid: true},
		})
	})
}

func (r *AlertInstanceRepository) ListForRead(ctx context.Context, filters types.AlertListFilters) ([]AlertReadRow, error) {
	params := generated.ListAlertInstancesParams{
		HasProjectID: filters.ProjectID != nil,
		ProjectID:    derefInt64(filters.ProjectID),
		HasServiceID: filters.ServiceID != nil,
		ServiceID:    nullableInt64(filters.ServiceID),
		HasNodeID:    filters.NodeID != nil,
		NodeID:       nullableInt64(filters.NodeID),
		HasRuleType:  filters.RuleType != nil,
		RuleType:     derefString(filters.RuleType),
		HasSeverity:  filters.Severity != nil,
		Severity:     derefString(filters.Severity),
		HasCategory:  filters.Category != nil,
		Category:     derefString(filters.Category),
		LimitRows:    filters.Limit,
	}

	rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListAlertInstancesRow, error) {
		return queries.ListAlertInstances(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	items := make([]AlertReadRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, AlertReadRow{
			ID: row.ID, AlertRuleID: row.AlertRuleID, ProjectID: row.ProjectID,
			NodeID: row.NodeID, NodeName: row.NodeName, NodeIdentifier: row.NodeIdentifier,
			ServiceID: row.ServiceID, ServiceName: row.ServiceName,
			RuleType: row.RuleType, Severity: row.Severity, Status: row.Status,
			TriggeredAt: row.TriggeredAt, ResolvedAt: row.ResolvedAt,
			Title: row.Title, Message: row.Message, CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}

func (r *AlertInstanceRepository) ListActiveForRead(ctx context.Context, filters types.AlertListFilters) ([]AlertReadRow, error) {
	params := generated.ListActiveAlertInstancesForReadParams{
		HasProjectID: filters.ProjectID != nil,
		ProjectID:    derefInt64(filters.ProjectID),
		HasServiceID: filters.ServiceID != nil,
		ServiceID:    nullableInt64(filters.ServiceID),
		HasNodeID:    filters.NodeID != nil,
		NodeID:       nullableInt64(filters.NodeID),
		HasRuleType:  filters.RuleType != nil,
		RuleType:     derefString(filters.RuleType),
		HasSeverity:  filters.Severity != nil,
		Severity:     derefString(filters.Severity),
		HasCategory:  filters.Category != nil,
		Category:     derefString(filters.Category),
	}

	rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertInstancesForReadRow, error) {
		return queries.ListActiveAlertInstancesForRead(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	items := make([]AlertReadRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, AlertReadRow{
			ID: row.ID, AlertRuleID: row.AlertRuleID, ProjectID: row.ProjectID,
			NodeID: row.NodeID, NodeName: row.NodeName, NodeIdentifier: row.NodeIdentifier,
			ServiceID: row.ServiceID, ServiceName: row.ServiceName,
			RuleType: row.RuleType, Severity: row.Severity, Status: row.Status,
			TriggeredAt: row.TriggeredAt, ResolvedAt: row.ResolvedAt,
			Title: row.Title, Message: row.Message, CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}

func (r *AlertInstanceRepository) ListResolvedForRead(ctx context.Context, filters types.AlertListFilters) ([]AlertReadRow, error) {
	params := generated.ListResolvedAlertInstancesForReadParams{
		HasProjectID: filters.ProjectID != nil,
		ProjectID:    derefInt64(filters.ProjectID),
		HasServiceID: filters.ServiceID != nil,
		ServiceID:    nullableInt64(filters.ServiceID),
		HasNodeID:    filters.NodeID != nil,
		NodeID:       nullableInt64(filters.NodeID),
		HasRuleType:  filters.RuleType != nil,
		RuleType:     derefString(filters.RuleType),
		HasSeverity:  filters.Severity != nil,
		Severity:     derefString(filters.Severity),
		HasCategory:  filters.Category != nil,
		Category:     derefString(filters.Category),
		HasBefore:    filters.Before != nil,
		LimitRows:    filters.Limit,
	}
	if filters.Before != nil {
		params.BeforeTriggeredAt = filters.Before.TriggeredAt.UTC()
		params.BeforeID = filters.Before.ID
	}

	rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListResolvedAlertInstancesForReadRow, error) {
		return queries.ListResolvedAlertInstancesForRead(ctx, params)
	})
	if err != nil {
		return nil, err
	}

	items := make([]AlertReadRow, 0, len(rows))
	for _, row := range rows {
		items = append(items, AlertReadRow{
			ID: row.ID, AlertRuleID: row.AlertRuleID, ProjectID: row.ProjectID,
			NodeID: row.NodeID, NodeName: row.NodeName, NodeIdentifier: row.NodeIdentifier,
			ServiceID: row.ServiceID, ServiceName: row.ServiceName,
			RuleType: row.RuleType, Severity: row.Severity, Status: row.Status,
			TriggeredAt: row.TriggeredAt, ResolvedAt: row.ResolvedAt,
			Title: row.Title, Message: row.Message, CreatedAt: row.CreatedAt,
		})
	}
	return items, nil
}

func nullableInt64(value *int64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *value, Valid: true}
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

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.AlertInstance, error) {
		return queries.ListAlertInstancesByNodeAndStatus(ctx, params)
	})
}

func (r *AlertInstanceRepository) ListActiveDetailsByServiceID(ctx context.Context, serviceID int64) ([]generated.ListActiveAlertDetailsByServiceIDRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertDetailsByServiceIDRow, error) {
		return queries.ListActiveAlertDetailsByServiceID(ctx, sql.NullInt64{Int64: serviceID, Valid: true})
	})
}

func (r *AlertInstanceRepository) ListActiveCountsByNode(ctx context.Context, projectID *int64) ([]generated.ListActiveAlertCountsByNodeRow, error) {
	if projectID != nil {
		rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertCountsByNodeByProjectRow, error) {
			return queries.ListActiveAlertCountsByNodeByProject(ctx, *projectID)
		})
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

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertCountsByNodeRow, error) {
		return queries.ListActiveAlertCountsByNode(ctx)
	})
}

func (r *AlertInstanceRepository) ListActiveCountsByService(ctx context.Context, projectID *int64) ([]generated.ListActiveAlertCountsByServiceRow, error) {
	if projectID != nil {
		rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertCountsByServiceByProjectRow, error) {
			return queries.ListActiveAlertCountsByServiceByProject(ctx, *projectID)
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

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertCountsByServiceRow, error) {
		return queries.ListActiveAlertCountsByService(ctx)
	})
}

func (r *AlertInstanceRepository) CountActiveByServiceID(ctx context.Context, serviceID int64) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.CountActiveAlertInstancesByServiceID(ctx, sql.NullInt64{Int64: serviceID, Valid: true})
	})
}

func (r *AlertInstanceRepository) CountActiveByNodeID(ctx context.Context, nodeID int64) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.CountActiveAlertInstancesByNodeID(ctx, sql.NullInt64{Int64: nodeID, Valid: true})
	})
}

func (r *AlertInstanceRepository) ListActiveCountsByServiceForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.ListActiveAlertCountsByServiceRow, error) {
	rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListActiveAlertCountsByServiceForReadRow, error) {
		return queries.ListActiveAlertCountsByServiceForRead(ctx, generated.ListActiveAlertCountsByServiceForReadParams{
			HasProjectID: filters.ProjectID != nil,
			ProjectID:    derefInt64(filters.ProjectID),
			HasNodeID:    filters.NodeID != nil,
			NodeID:       derefInt64(filters.NodeID),
			HasStatus:    filters.Status != nil,
			Status:       derefString(filters.Status),
		})
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
