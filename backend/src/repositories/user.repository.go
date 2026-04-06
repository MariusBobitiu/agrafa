package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type UserRepository struct {
	queries *generated.Queries
}

func NewUserRepository(queries *generated.Queries) *UserRepository {
	return &UserRepository{queries: queries}
}

func (r *UserRepository) GetByID(ctx context.Context, id string) (generated.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *UserRepository) MarkEmailVerifiedByID(ctx context.Context, id string) error {
	rowsUpdated, err := r.queries.MarkUserEmailVerifiedByID(ctx, id)
	if err != nil {
		return err
	}
	if rowsUpdated == 0 {
		return sql.ErrNoRows
	}

	return nil
}
