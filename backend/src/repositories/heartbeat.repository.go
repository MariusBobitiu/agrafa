package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type HeartbeatRepository struct {
	queries *generated.Queries
}

func NewHeartbeatRepository(queries *generated.Queries) *HeartbeatRepository {
	return &HeartbeatRepository{queries: queries}
}

func (r *HeartbeatRepository) Create(ctx context.Context, params generated.CreateHeartbeatParams) (generated.Heartbeat, error) {
	return r.queries.CreateHeartbeat(ctx, params)
}
