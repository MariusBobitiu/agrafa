package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeProjectMemberRepository struct {
	createParams           generated.CreateProjectMemberParams
	createResult           generated.ProjectMember
	createErr              error
	memberByID             generated.ProjectMember
	memberByIDErr          error
	memberByProjectAndUser generated.ProjectMember
	memberByProjectUserErr error
	memberForReadByID      generated.GetProjectMemberForReadByIDRow
	memberForReadErr       error
	listRows               []generated.ListProjectMembersForReadRow
	listErr                error
	updateRoleID           string
	updateRoleValue        string
	updateRoleErr          error
	deleteID               string
	deleteRows             int64
	deleteErr              error
	ownerCount             int64
	ownerCountErr          error
}

func (r *fakeProjectMemberRepository) Create(_ context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error) {
	r.createParams = params
	return r.createResult, r.createErr
}

func (r *fakeProjectMemberRepository) GetByID(_ context.Context, _ string) (generated.ProjectMember, error) {
	return r.memberByID, r.memberByIDErr
}

func (r *fakeProjectMemberRepository) GetByProjectAndUser(_ context.Context, _ int64, _ string) (generated.ProjectMember, error) {
	return r.memberByProjectAndUser, r.memberByProjectUserErr
}

func (r *fakeProjectMemberRepository) GetForReadByID(_ context.Context, _ string) (generated.GetProjectMemberForReadByIDRow, error) {
	return r.memberForReadByID, r.memberForReadErr
}

func (r *fakeProjectMemberRepository) ListForRead(_ context.Context, _ int64) ([]generated.ListProjectMembersForReadRow, error) {
	return r.listRows, r.listErr
}

func (r *fakeProjectMemberRepository) UpdateRole(_ context.Context, id string, role string) (generated.ProjectMember, error) {
	r.updateRoleID = id
	r.updateRoleValue = role
	if r.updateRoleErr != nil {
		return generated.ProjectMember{}, r.updateRoleErr
	}

	return generated.ProjectMember{
		ID:        id,
		ProjectID: r.memberByID.ProjectID,
		UserID:    r.memberByID.UserID,
		Role:      role,
	}, nil
}

func (r *fakeProjectMemberRepository) Delete(_ context.Context, id string) (int64, error) {
	r.deleteID = id
	return r.deleteRows, r.deleteErr
}

func (r *fakeProjectMemberRepository) CountOwners(_ context.Context, _ int64) (int64, error) {
	return r.ownerCount, r.ownerCountErr
}

type fakeProjectMemberProjectRepo struct {
	project generated.Project
	err     error
}

func (r *fakeProjectMemberProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return r.project, r.err
}

type fakeProjectMemberUserRepo struct {
	user generated.User
	err  error
}

func (r *fakeProjectMemberUserRepo) GetByID(_ context.Context, _ string) (generated.User, error) {
	return r.user, r.err
}

func TestProjectMemberServiceCreateRejectsDuplicateMembership(t *testing.T) {
	t.Parallel()

	service := &ProjectMemberService{
		projectMemberRepo: &fakeProjectMemberRepository{
			memberByProjectAndUser: generated.ProjectMember{ID: "pm_existing"},
		},
		projectRepo: &fakeProjectMemberProjectRepo{
			project: generated.Project{ID: 1},
		},
		userRepo: &fakeProjectMemberUserRepo{
			user: generated.User{ID: "usr_2"},
		},
	}

	_, err := service.Create(context.Background(), types.CreateProjectMemberInput{
		ProjectID: 1,
		UserID:    "usr_2",
		Role:      "viewer",
	})
	if !errors.Is(err, types.ErrProjectMemberAlreadyExists) {
		t.Fatalf("Create() error = %v, want ErrProjectMemberAlreadyExists", err)
	}
}

func TestProjectMemberServiceCreateRejectsInvalidRole(t *testing.T) {
	t.Parallel()

	service := &ProjectMemberService{
		projectMemberRepo: &fakeProjectMemberRepository{},
		projectRepo:       &fakeProjectMemberProjectRepo{},
		userRepo:          &fakeProjectMemberUserRepo{},
	}

	_, err := service.Create(context.Background(), types.CreateProjectMemberInput{
		ProjectID: 1,
		UserID:    "usr_2",
		Role:      "operator",
	})
	if !errors.Is(err, types.ErrInvalidProjectMemberRole) {
		t.Fatalf("Create() error = %v, want ErrInvalidProjectMemberRole", err)
	}
}

func TestProjectMemberServiceUpdateRoleRejectsLastOwnerDemotion(t *testing.T) {
	t.Parallel()

	service := &ProjectMemberService{
		projectMemberRepo: &fakeProjectMemberRepository{
			memberByID: generated.ProjectMember{
				ID:        "pm_owner",
				ProjectID: 1,
				UserID:    "usr_owner",
				Role:      ProjectRoleOwner,
			},
			ownerCount: 1,
		},
		projectRepo: &fakeProjectMemberProjectRepo{},
		userRepo:    &fakeProjectMemberUserRepo{},
	}

	_, err := service.UpdateRole(context.Background(), types.UpdateProjectMemberInput{
		ID:   "pm_owner",
		Role: "admin",
	})
	if !errors.Is(err, types.ErrCannotRemoveLastProjectOwner) {
		t.Fatalf("UpdateRole() error = %v, want ErrCannotRemoveLastProjectOwner", err)
	}
}

func TestProjectMemberServiceUpdateRoleSucceeds(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectMemberRepository{
		memberByID: generated.ProjectMember{
			ID:        "pm_member",
			ProjectID: 1,
			UserID:    "usr_member",
			Role:      ProjectRoleViewer,
		},
		memberForReadByID: generated.GetProjectMemberForReadByIDRow{
			ID:        "pm_member",
			ProjectID: 1,
			UserID:    "usr_member",
			Role:      ProjectRoleAdmin,
			CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
			Name:      "Alice",
			Email:     "alice@example.com",
		},
	}

	service := &ProjectMemberService{
		projectMemberRepo: repo,
		projectRepo:       &fakeProjectMemberProjectRepo{},
		userRepo:          &fakeProjectMemberUserRepo{},
	}

	member, err := service.UpdateRole(context.Background(), types.UpdateProjectMemberInput{
		ID:   "pm_member",
		Role: "admin",
	})
	if err != nil {
		t.Fatalf("UpdateRole() error = %v", err)
	}
	if repo.updateRoleID != "pm_member" {
		t.Fatalf("updateRoleID = %q, want pm_member", repo.updateRoleID)
	}
	if repo.updateRoleValue != ProjectRoleAdmin {
		t.Fatalf("updateRoleValue = %q, want %q", repo.updateRoleValue, ProjectRoleAdmin)
	}
	if member.Role != ProjectRoleAdmin {
		t.Fatalf("member.Role = %q, want %q", member.Role, ProjectRoleAdmin)
	}
	if member.User.Email != "alice@example.com" {
		t.Fatalf("member.User.Email = %q, want alice@example.com", member.User.Email)
	}
}

func TestProjectMemberServiceDeleteNonOwnerSucceeds(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectMemberRepository{
		memberByID: generated.ProjectMember{
			ID:        "pm_viewer",
			ProjectID: 1,
			UserID:    "usr_viewer",
			Role:      ProjectRoleViewer,
		},
		deleteRows: 1,
	}

	service := &ProjectMemberService{
		projectMemberRepo: repo,
		projectRepo:       &fakeProjectMemberProjectRepo{},
		userRepo:          &fakeProjectMemberUserRepo{},
	}

	if err := service.Delete(context.Background(), "pm_viewer"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.deleteID != "pm_viewer" {
		t.Fatalf("deleteID = %q, want pm_viewer", repo.deleteID)
	}
}

func TestProjectMemberServiceDeleteOwnerSucceedsWhenAnotherOwnerRemains(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectMemberRepository{
		memberByID: generated.ProjectMember{
			ID:        "pm_owner",
			ProjectID: 1,
			UserID:    "usr_owner",
			Role:      ProjectRoleOwner,
		},
		ownerCount: 2,
		deleteRows: 1,
	}

	service := &ProjectMemberService{
		projectMemberRepo: repo,
		projectRepo:       &fakeProjectMemberProjectRepo{},
		userRepo:          &fakeProjectMemberUserRepo{},
	}

	if err := service.Delete(context.Background(), "pm_owner"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if repo.deleteID != "pm_owner" {
		t.Fatalf("deleteID = %q, want pm_owner", repo.deleteID)
	}
}

func TestProjectMemberServiceDeleteRejectsLastOwnerRemoval(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectMemberRepository{
		memberByID: generated.ProjectMember{
			ID:        "pm_owner",
			ProjectID: 1,
			UserID:    "usr_owner",
			Role:      ProjectRoleOwner,
		},
		ownerCount: 1,
	}

	service := &ProjectMemberService{
		projectMemberRepo: repo,
		projectRepo:       &fakeProjectMemberProjectRepo{},
		userRepo:          &fakeProjectMemberUserRepo{},
	}

	err := service.Delete(context.Background(), "pm_owner")
	if !errors.Is(err, types.ErrCannotRemoveLastProjectOwner) {
		t.Fatalf("Delete() error = %v, want ErrCannotRemoveLastProjectOwner", err)
	}
	if repo.deleteID != "" {
		t.Fatalf("deleteID = %q, want empty", repo.deleteID)
	}
}

func TestProjectMemberServiceDeleteMissingMembershipReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &ProjectMemberService{
		projectMemberRepo: &fakeProjectMemberRepository{
			memberByIDErr: sql.ErrNoRows,
		},
		projectRepo: &fakeProjectMemberProjectRepo{},
		userRepo:    &fakeProjectMemberUserRepo{},
	}

	err := service.Delete(context.Background(), "pm_missing")
	if !errors.Is(err, types.ErrProjectMemberNotFound) {
		t.Fatalf("Delete() error = %v, want ErrProjectMemberNotFound", err)
	}
}

func TestProjectMemberServiceCreateSucceeds(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectMemberRepository{
		memberByProjectUserErr: sql.ErrNoRows,
		createResult: generated.ProjectMember{
			ID:        "pm_new",
			ProjectID: 1,
			UserID:    "usr_2",
			Role:      ProjectRoleViewer,
		},
		memberForReadByID: generated.GetProjectMemberForReadByIDRow{
			ID:        "pm_new",
			ProjectID: 1,
			UserID:    "usr_2",
			Role:      ProjectRoleViewer,
			CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
			Name:      "Bob",
			Email:     "bob@example.com",
		},
	}

	service := &ProjectMemberService{
		projectMemberRepo: repo,
		projectRepo: &fakeProjectMemberProjectRepo{
			project: generated.Project{ID: 1},
		},
		userRepo: &fakeProjectMemberUserRepo{
			user: generated.User{ID: "usr_2"},
		},
	}

	member, err := service.Create(context.Background(), types.CreateProjectMemberInput{
		ProjectID: 1,
		UserID:    "usr_2",
		Role:      "viewer",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.createParams.ProjectID != 1 {
		t.Fatalf("create project_id = %d, want 1", repo.createParams.ProjectID)
	}
	if repo.createParams.UserID != "usr_2" {
		t.Fatalf("create user_id = %q, want usr_2", repo.createParams.UserID)
	}
	if repo.createParams.Role != ProjectRoleViewer {
		t.Fatalf("create role = %q, want %q", repo.createParams.Role, ProjectRoleViewer)
	}
	if repo.createParams.ID == "" {
		t.Fatal("create id = empty, want generated id")
	}
	if member.User.Name != "Bob" {
		t.Fatalf("member.User.Name = %q, want Bob", member.User.Name)
	}
}

func TestProjectMemberServiceGetByIDReturnsUserSummary(t *testing.T) {
	t.Parallel()

	service := &ProjectMemberService{
		projectMemberRepo: &fakeProjectMemberRepository{
			memberForReadByID: generated.GetProjectMemberForReadByIDRow{
				ID:        "pm_1",
				ProjectID: 1,
				UserID:    "usr_1",
				Role:      ProjectRoleViewer,
				CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
				Name:      "Alice",
				Email:     "alice@example.com",
			},
		},
		projectRepo: &fakeProjectMemberProjectRepo{},
		userRepo:    &fakeProjectMemberUserRepo{},
	}

	member, err := service.GetByID(context.Background(), "pm_1")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if member.ID != "pm_1" || member.User.Email != "alice@example.com" {
		t.Fatalf("unexpected project member: %#v", member)
	}
}
