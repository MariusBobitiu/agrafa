package repositories

import (
	"context"
	"database/sql"
	"fmt"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

func withRLSQueries[T any](
	ctx context.Context,
	db *sql.DB,
	queries *generated.Queries,
	fn func(*generated.Queries) (T, error),
) (T, error) {
	if !appdb.HasRLSSessionContext(ctx) {
		return fn(queries)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	if err := appdb.ApplyRLSSessionContext(ctx, tx); err != nil {
		var zero T
		return zero, err
	}

	value, err := fn(queries.WithTx(tx))
	if err != nil {
		return value, err
	}

	if err := tx.Commit(); err != nil {
		var zero T
		return zero, fmt.Errorf("commit tx: %w", err)
	}

	return value, nil
}
