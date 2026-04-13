package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type HealthCheckRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewHealthCheckRepository(db *sql.DB, queries *generated.Queries) *HealthCheckRepository {
	return &HealthCheckRepository{
		db:      db,
		queries: queries,
	}
}

func (r *HealthCheckRepository) Create(ctx context.Context, params generated.CreateHealthCheckResultParams) (generated.HealthCheckResult, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.HealthCheckResult, error) {
		return queries.CreateHealthCheckResult(ctx, params)
	})
}

func (r *HealthCheckRepository) GetLatestByServiceID(ctx context.Context, serviceID int64) (generated.HealthCheckResult, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.HealthCheckResult, error) {
		return queries.GetLatestHealthCheckResultByServiceID(ctx, serviceID)
	})
}

func (r *HealthCheckRepository) ListLatest(ctx context.Context, projectID *int64) ([]generated.HealthCheckResult, error) {
	if projectID != nil {
		return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.HealthCheckResult, error) {
			return queries.ListLatestHealthCheckResultsByProject(ctx, *projectID)
		})
	}

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.HealthCheckResult, error) {
		return queries.ListLatestHealthCheckResults(ctx)
	})
}

func (r *HealthCheckRepository) ListLatestForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.HealthCheckResult, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.HealthCheckResult, error) {
		return queries.ListLatestHealthCheckResultsForRead(ctx, generated.ListLatestHealthCheckResultsForReadParams{
			HasProjectID: filters.ProjectID != nil,
			ProjectID:    derefInt64(filters.ProjectID),
			HasNodeID:    filters.NodeID != nil,
			NodeID:       derefInt64(filters.NodeID),
			HasStatus:    filters.Status != nil,
			Status:       derefString(filters.Status),
		})
	})
}
