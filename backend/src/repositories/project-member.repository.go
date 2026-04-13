package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type ProjectMemberRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewProjectMemberRepository(db *sql.DB, queries *generated.Queries) *ProjectMemberRepository {
	return &ProjectMemberRepository{
		db:      db,
		queries: queries,
	}
}

func (r *ProjectMemberRepository) Create(ctx context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectMember, error) {
		return queries.CreateProjectMember(ctx, params)
	})
}

func (r *ProjectMemberRepository) GetByID(ctx context.Context, id string) (generated.ProjectMember, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectMember, error) {
		return queries.GetProjectMemberByID(ctx, id)
	})
}

func (r *ProjectMemberRepository) GetByProjectAndUser(ctx context.Context, projectID int64, userID string) (generated.ProjectMember, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectMember, error) {
		return queries.GetProjectMemberByProjectAndUser(ctx, generated.GetProjectMemberByProjectAndUserParams{
			ProjectID: projectID,
			UserID:    userID,
		})
	})
}

func (r *ProjectMemberRepository) GetForReadByID(ctx context.Context, id string) (generated.GetProjectMemberForReadByIDRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.GetProjectMemberForReadByIDRow, error) {
		return queries.GetProjectMemberForReadByID(ctx, id)
	})
}

func (r *ProjectMemberRepository) ListForRead(ctx context.Context, projectID int64) ([]generated.ListProjectMembersForReadRow, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListProjectMembersForReadRow, error) {
		return queries.ListProjectMembersForRead(ctx, projectID)
	})
}

func (r *ProjectMemberRepository) UpdateRole(ctx context.Context, id string, role string) (generated.ProjectMember, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.ProjectMember, error) {
		return queries.UpdateProjectMemberRole(ctx, generated.UpdateProjectMemberRoleParams{
			ID:   id,
			Role: role,
		})
	})
}

func (r *ProjectMemberRepository) Delete(ctx context.Context, id string) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.DeleteProjectMemberByID(ctx, id)
	})
}

func (r *ProjectMemberRepository) CountOwners(ctx context.Context, projectID int64) (int64, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (int64, error) {
		return queries.CountProjectOwners(ctx, projectID)
	})
}
