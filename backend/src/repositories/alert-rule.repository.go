package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type AlertRuleRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewAlertRuleRepository(db *sql.DB, queries *generated.Queries) *AlertRuleRepository {
	return &AlertRuleRepository{
		db:      db,
		queries: queries,
	}
}

func (r *AlertRuleRepository) Create(ctx context.Context, params generated.CreateAlertRuleParams) (generated.AlertRule, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertRule, error) {
		return queries.CreateAlertRule(ctx, params)
	})
}

func (r *AlertRuleRepository) GetByID(ctx context.Context, id int64) (generated.AlertRule, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertRule, error) {
		return queries.GetAlertRuleByID(ctx, id)
	})
}

func (r *AlertRuleRepository) Update(ctx context.Context, params generated.UpdateAlertRuleParams) (generated.AlertRule, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.AlertRule, error) {
		return queries.UpdateAlertRule(ctx, params)
	})
}

func (r *AlertRuleRepository) UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.AlertRule, error) {
	return r.Update(ctx, generated.UpdateAlertRuleParams{
		ID:           id,
		SetIsEnabled: true,
		IsEnabled:    isEnabled,
	})
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.DeleteAlertRuleByID(ctx, id)
	})
}

func (r *AlertRuleRepository) List(ctx context.Context, projectID *int64) ([]generated.AlertRule, error) {
	params := generated.ListAlertRulesParams{}
	if projectID != nil {
		params.Column1 = true
		params.ProjectID = *projectID
	}

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.AlertRule, error) {
		return queries.ListAlertRules(ctx, params)
	})
}

func (r *AlertRuleRepository) ListEnabled(
	ctx context.Context,
	projectID int64,
	ruleType string,
	nodeID *int64,
	serviceID *int64,
	metricName *string,
) ([]generated.AlertRule, error) {
	params := generated.ListEnabledAlertRulesParams{
		ProjectID: projectID,
		RuleType:  ruleType,
	}

	if nodeID != nil {
		params.HasNodeID = true
		params.NodeID = sql.NullInt64{Int64: *nodeID, Valid: true}
	}

	if serviceID != nil {
		params.HasServiceID = true
		params.ServiceID = sql.NullInt64{Int64: *serviceID, Valid: true}
	}

	if metricName != nil {
		params.HasMetricName = true
		params.MetricName = sql.NullString{String: *metricName, Valid: true}
	}

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.AlertRule, error) {
		return queries.ListEnabledAlertRules(ctx, params)
	})
}
