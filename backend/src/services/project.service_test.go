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

type fakeProjectRepository struct {
	project               generated.Project
	projectsForUser       []generated.ListProjectsForUserRow
	getByIDErr            error
	listForUserErr        error
	updateNameID          int64
	updateNameValue       string
	updateNameErr         error
	createWithOwnerParams generated.CreateProjectParams
	createWithOwnerMember generated.CreateProjectMemberParams
	deleteID              int64
	deleteRows            int64
	deleteErr             error
}

func (r *fakeProjectRepository) Create(_ context.Context, _ generated.CreateProjectParams) (generated.Project, error) {
	return r.project, nil
}

func (r *fakeProjectRepository) CreateWithOwner(_ context.Context, projectParams generated.CreateProjectParams, memberParams generated.CreateProjectMemberParams) (generated.Project, error) {
	r.createWithOwnerParams = projectParams
	r.createWithOwnerMember = memberParams
	return r.project, nil
}

func (r *fakeProjectRepository) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return r.project, r.getByIDErr
}

func (r *fakeProjectRepository) ListForUser(_ context.Context, _ string) ([]generated.ListProjectsForUserRow, error) {
	return r.projectsForUser, r.listForUserErr
}

func (r *fakeProjectRepository) UpdateName(_ context.Context, id int64, name string) (generated.Project, error) {
	r.updateNameID = id
	r.updateNameValue = name
	return r.project, r.updateNameErr
}

func (r *fakeProjectRepository) Delete(_ context.Context, id int64) (int64, error) {
	r.deleteID = id
	return r.deleteRows, r.deleteErr
}

type fakeProjectMembershipRepository struct {
	member generated.ProjectMember
	err    error
}

func (r *fakeProjectMembershipRepository) GetByProjectAndUser(_ context.Context, _ int64, _ string) (generated.ProjectMember, error) {
	return r.member, r.err
}

type fakeProjectOverviewRepository struct {
	stats generated.GetOverviewStatsByProjectRow
	err   error
}

func (r *fakeProjectOverviewRepository) GetStatsByProject(_ context.Context, _ int64) (generated.GetOverviewStatsByProjectRow, error) {
	return r.stats, r.err
}

func TestProjectServiceCreateAssignsOwnerMembership(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectRepository{
		project: generated.Project{ID: 7, Name: "Platform"},
	}

	service := &ProjectService{
		projectRepo:       repo,
		projectMemberRepo: &fakeProjectMembershipRepository{},
		overviewRepo:      &fakeProjectOverviewRepository{},
	}

	project, err := service.Create(context.Background(), "usr_123", "Platform")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if project.ID != 7 {
		t.Fatalf("project.ID = %d, want 7", project.ID)
	}
	if repo.createWithOwnerParams.Name != "Platform" {
		t.Fatalf("project name = %q, want Platform", repo.createWithOwnerParams.Name)
	}
	if repo.createWithOwnerMember.UserID != "usr_123" {
		t.Fatalf("member user_id = %q, want usr_123", repo.createWithOwnerMember.UserID)
	}
	if repo.createWithOwnerMember.Role != ProjectRoleOwner {
		t.Fatalf("member role = %q, want %q", repo.createWithOwnerMember.Role, ProjectRoleOwner)
	}
	if repo.createWithOwnerMember.ID == "" {
		t.Fatal("member id = empty, want generated id")
	}
}

func TestProjectServiceListForUserReturnsOnlyMemberProjects(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectRepository{
		projectsForUser: []generated.ListProjectsForUserRow{
			{
				ID:        1,
				Slug:      "alpha",
				Name:      "Alpha",
				CreatedAt: time.Date(2026, time.April, 5, 10, 0, 0, 0, time.UTC),
				Role:      ProjectRoleOwner,
			},
			{
				ID:        2,
				Slug:      "beta",
				Name:      "Beta",
				CreatedAt: time.Date(2026, time.April, 5, 11, 0, 0, 0, time.UTC),
				Role:      ProjectRoleViewer,
			},
		},
	}

	service := &ProjectService{
		projectRepo:       repo,
		projectMemberRepo: &fakeProjectMembershipRepository{},
		overviewRepo:      &fakeProjectOverviewRepository{},
	}

	projects, err := service.ListForUser(context.Background(), "usr_1")
	if err != nil {
		t.Fatalf("ListForUser() error = %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("len(projects) = %d, want 2", len(projects))
	}
	if projects[0].CurrentUserRole != ProjectRoleOwner {
		t.Fatalf("projects[0].CurrentUserRole = %q, want %q", projects[0].CurrentUserRole, ProjectRoleOwner)
	}
	if projects[1].ID != 2 {
		t.Fatalf("projects[1].ID = %d, want 2", projects[1].ID)
	}
}

func TestProjectServiceGetReturnsProjectDetails(t *testing.T) {
	t.Parallel()

	service := &ProjectService{
		projectRepo: &fakeProjectRepository{
			project: generated.Project{
				ID:        7,
				Slug:      "platform",
				Name:      "Platform",
				CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
			},
		},
		projectMemberRepo: &fakeProjectMembershipRepository{
			member: generated.ProjectMember{Role: ProjectRoleAdmin},
		},
		overviewRepo: &fakeProjectOverviewRepository{
			stats: generated.GetOverviewStatsByProjectRow{
				TotalNodes:    3,
				TotalServices: 5,
				ActiveAlerts:  2,
			},
		},
	}

	project, err := service.Get(context.Background(), "usr_1", 7)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if project.ID != 7 || project.Name != "Platform" {
		t.Fatalf("unexpected project detail: %#v", project)
	}
	if project.CurrentUserRole != ProjectRoleAdmin {
		t.Fatalf("CurrentUserRole = %q, want %q", project.CurrentUserRole, ProjectRoleAdmin)
	}
	if project.NodeCount != 3 || project.ServiceCount != 5 || project.ActiveAlertCount != 2 {
		t.Fatalf("unexpected counts: %#v", project)
	}
}

func TestProjectServiceGetMissingProjectReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &ProjectService{
		projectRepo:       &fakeProjectRepository{getByIDErr: sql.ErrNoRows},
		projectMemberRepo: &fakeProjectMembershipRepository{},
		overviewRepo:      &fakeProjectOverviewRepository{},
	}

	_, err := service.Get(context.Background(), "usr_1", 9)
	if !errors.Is(err, types.ErrProjectNotFound) {
		t.Fatalf("Get() error = %v, want ErrProjectNotFound", err)
	}
}

func TestProjectServiceDeleteMissingProjectReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &ProjectService{
		projectRepo:       &fakeProjectRepository{deleteRows: 0},
		projectMemberRepo: &fakeProjectMembershipRepository{},
		overviewRepo:      &fakeProjectOverviewRepository{},
	}

	err := service.Delete(context.Background(), 12)
	if !errors.Is(err, types.ErrProjectNotFound) {
		t.Fatalf("Delete() error = %v, want ErrProjectNotFound", err)
	}
}

func TestProjectServiceUpdateRenamesProject(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectRepository{
		project: generated.Project{
			ID:        7,
			Slug:      "platform",
			Name:      "Renamed Platform",
			CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
		},
	}

	service := &ProjectService{
		projectRepo: repo,
		projectMemberRepo: &fakeProjectMembershipRepository{
			member: generated.ProjectMember{Role: ProjectRoleOwner},
		},
		overviewRepo: &fakeProjectOverviewRepository{},
	}

	newName := "Renamed Platform"
	project, err := service.Update(context.Background(), "usr_1", 7, types.UpdateProjectInput{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updateNameID != 7 {
		t.Fatalf("updateNameID = %d, want 7", repo.updateNameID)
	}
	if repo.updateNameValue != newName {
		t.Fatalf("updateNameValue = %q, want %q", repo.updateNameValue, newName)
	}
	if project.Name != newName {
		t.Fatalf("project.Name = %q, want %q", project.Name, newName)
	}
}

func TestProjectServiceUpdateRejectsEmptyUpdate(t *testing.T) {
	t.Parallel()

	service := &ProjectService{
		projectRepo:       &fakeProjectRepository{},
		projectMemberRepo: &fakeProjectMembershipRepository{},
		overviewRepo:      &fakeProjectOverviewRepository{},
	}

	_, err := service.Update(context.Background(), "usr_1", 7, types.UpdateProjectInput{})
	if !errors.Is(err, types.ErrNoFieldsToUpdate) {
		t.Fatalf("Update() error = %v, want ErrNoFieldsToUpdate", err)
	}
}
