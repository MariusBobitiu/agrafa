package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type AlertRuleRepository struct {
	queries *generated.Queries
}

func NewAlertRuleRepository(queries *generated.Queries) *AlertRuleRepository {
	return &AlertRuleRepository{queries: queries}
}

func (r *AlertRuleRepository) Create(ctx context.Context, params generated.CreateAlertRuleParams) (generated.AlertRule, error) {
	return r.queries.CreateAlertRule(ctx, params)
}

func (r *AlertRuleRepository) GetByID(ctx context.Context, id int64) (generated.AlertRule, error) {
	return r.queries.GetAlertRuleByID(ctx, id)
}

func (r *AlertRuleRepository) Update(ctx context.Context, params generated.UpdateAlertRuleParams) (generated.AlertRule, error) {
	return r.queries.UpdateAlertRule(ctx, params)
}

func (r *AlertRuleRepository) UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.AlertRule, error) {
	return r.Update(ctx, generated.UpdateAlertRuleParams{
		ID:        id,
		Column10:  true,
		IsEnabled: isEnabled,
	})
}

func (r *AlertRuleRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return r.queries.DeleteAlertRuleByID(ctx, id)
}

func (r *AlertRuleRepository) List(ctx context.Context, projectID *int64) ([]generated.AlertRule, error) {
	params := generated.ListAlertRulesParams{}
	if projectID != nil {
		params.Column1 = true
		params.ProjectID = *projectID
	}

	return r.queries.ListAlertRules(ctx, params)
}

func (r *AlertRuleRepository) ListEnabled(
	ctx context.Context,
	ruleType string,
	nodeID *int64,
	serviceID *int64,
	metricName *string,
) ([]generated.AlertRule, error) {
	params := generated.ListEnabledAlertRulesParams{
		RuleType: ruleType,
	}

	if nodeID != nil {
		params.Column2 = true
		params.NodeID = sql.NullInt64{Int64: *nodeID, Valid: true}
	}

	if serviceID != nil {
		params.Column4 = true
		params.ServiceID = sql.NullInt64{Int64: *serviceID, Valid: true}
	}

	if metricName != nil {
		params.Column6 = true
		params.MetricName = sql.NullString{String: *metricName, Valid: true}
	}

	return r.queries.ListEnabledAlertRules(ctx, params)
}
