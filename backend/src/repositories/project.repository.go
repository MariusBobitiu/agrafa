package repositories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type ProjectRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewProjectRepository(db *sql.DB, queries *generated.Queries) *ProjectRepository {
	return &ProjectRepository{
		db:      db,
		queries: queries,
	}
}

func (r *ProjectRepository) Create(ctx context.Context, params generated.CreateProjectParams) (generated.Project, error) {
	return r.queries.CreateProject(ctx, params)
}

func (r *ProjectRepository) GetByID(ctx context.Context, id int64) (generated.Project, error) {
	return r.queries.GetProjectByID(ctx, id)
}

func (r *ProjectRepository) ListForUser(ctx context.Context, userID string) ([]generated.ListProjectsForUserRow, error) {
	return r.queries.ListProjectsForUser(ctx, userID)
}

func (r *ProjectRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return r.queries.DeleteProjectByID(ctx, id)
}

func (r *ProjectRepository) UpdateName(ctx context.Context, id int64, name string) (generated.Project, error) {
	return r.queries.UpdateProjectName(ctx, generated.UpdateProjectNameParams{
		ID:   id,
		Name: name,
	})
}

func (r *ProjectRepository) CreateWithOwner(
	ctx context.Context,
	projectParams generated.CreateProjectParams,
	projectMemberParams generated.CreateProjectMemberParams,
) (generated.Project, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return generated.Project{}, fmt.Errorf("begin tx: %w", err)
	}

	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	queries := r.queries.WithTx(tx)

	project, err := queries.CreateProject(ctx, projectParams)
	if err != nil {
		return generated.Project{}, err
	}

	projectMemberParams.ProjectID = project.ID
	if _, err := queries.CreateProjectMember(ctx, projectMemberParams); err != nil {
		return generated.Project{}, err
	}

	if err := tx.Commit(); err != nil {
		return generated.Project{}, fmt.Errorf("commit tx: %w", err)
	}

	tx = nil
	return project, nil
}
