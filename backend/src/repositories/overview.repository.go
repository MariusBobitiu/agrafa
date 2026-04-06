package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type OverviewRepository struct {
	queries *generated.Queries
}

func NewOverviewRepository(queries *generated.Queries) *OverviewRepository {
	return &OverviewRepository{queries: queries}
}

func (r *OverviewRepository) GetStats(ctx context.Context) (generated.GetOverviewStatsRow, error) {
	return r.queries.GetOverviewStats(ctx)
}

func (r *OverviewRepository) GetStatsByProject(ctx context.Context, projectID int64) (generated.GetOverviewStatsByProjectRow, error) {
	return r.queries.GetOverviewStatsByProject(ctx, projectID)
}
