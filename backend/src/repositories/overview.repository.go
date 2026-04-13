package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type OverviewRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewOverviewRepository(db *sql.DB, queries *generated.Queries) *OverviewRepository {
	return &OverviewRepository{
		db:      db,
		queries: queries,
	}
}

func (r *OverviewRepository) GetStats(ctx context.Context) (generated.GetOverviewStatsRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.GetOverviewStatsRow, error) {
		return queries.GetOverviewStats(ctx)
	})
}

func (r *OverviewRepository) GetStatsByProject(ctx context.Context, projectID int64) (generated.GetOverviewStatsByProjectRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.GetOverviewStatsByProjectRow, error) {
		return queries.GetOverviewStatsByProject(ctx, projectID)
	})
}
