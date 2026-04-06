package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeProjectInvitationRepository struct {
	createCalls          []generated.CreateProjectInvitationParams
	createParams         generated.CreateProjectInvitationParams
	createResult         generated.ProjectInvitation
	createErr            error
	activeInvitations    map[string]generated.ProjectInvitation
	activeInvitationErr  error
	invitationByID       generated.ProjectInvitation
	invitationByIDErr    error
	listResult           []generated.ProjectInvitation
	listErr              error
	invitationByToken    generated.GetProjectInvitationByTokenHashRow
	invitationByTokenErr error
	markAcceptedID       string
	markAcceptedAt       time.Time
	markAcceptedErr      error
	deleteID             string
	deleteRows           int64
	deleteErr            error
}

func (r *fakeProjectInvitationRepository) CreateReplacingPending(_ context.Context, params generated.CreateProjectInvitationParams) (generated.ProjectInvitation, error) {
	r.createCalls = append(r.createCalls, params)
	r.createParams = params
	if r.createErr != nil {
		return generated.ProjectInvitation{}, r.createErr
	}
	if r.createResult.ID != "" {
		return r.createResult, nil
	}

	return generated.ProjectInvitation{
		ID:              params.ID,
		ProjectID:       params.ProjectID,
		Email:           params.Email,
		Role:            params.Role,
		TokenHash:       params.TokenHash,
		InvitedByUserID: params.InvitedByUserID,
		ExpiresAt:       params.ExpiresAt,
		CreatedAt:       time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC),
	}, nil
}

func (r *fakeProjectInvitationRepository) GetByID(_ context.Context, _ string) (generated.ProjectInvitation, error) {
	return r.invitationByID, r.invitationByIDErr
}

func (r *fakeProjectInvitationRepository) GetActiveByProjectAndEmail(_ context.Context, projectID int64, email string, _ time.Time) (generated.ProjectInvitation, error) {
	if r.activeInvitationErr != nil {
		return generated.ProjectInvitation{}, r.activeInvitationErr
	}

	if invitation, ok := r.activeInvitations[fmt.Sprintf("%d|%s", projectID, email)]; ok {
		return invitation, nil
	}

	return generated.ProjectInvitation{}, sql.ErrNoRows
}

func (r *fakeProjectInvitationRepository) ListByProjectID(_ context.Context, _ int64) ([]generated.ProjectInvitation, error) {
	return r.listResult, r.listErr
}

func (r *fakeProjectInvitationRepository) GetByTokenHash(_ context.Context, _ string) (generated.GetProjectInvitationByTokenHashRow, error) {
	return r.invitationByToken, r.invitationByTokenErr
}

func (r *fakeProjectInvitationRepository) MarkAccepted(_ context.Context, id string, acceptedAt time.Time) (generated.ProjectInvitation, error) {
	r.markAcceptedID = id
	r.markAcceptedAt = acceptedAt
	if r.markAcceptedErr != nil {
		return generated.ProjectInvitation{}, r.markAcceptedErr
	}

	r.invitationByToken.AcceptedAt = sql.NullTime{Time: acceptedAt, Valid: true}
	return generated.ProjectInvitation{
		ID:              r.invitationByToken.ID,
		ProjectID:       r.invitationByToken.ProjectID,
		Email:           r.invitationByToken.Email,
		Role:            r.invitationByToken.Role,
		TokenHash:       r.invitationByToken.TokenHash,
		InvitedByUserID: r.invitationByToken.InvitedByUserID,
		ExpiresAt:       r.invitationByToken.ExpiresAt,
		AcceptedAt:      r.invitationByToken.AcceptedAt,
		CreatedAt:       r.invitationByToken.CreatedAt,
		UpdatedAt:       acceptedAt,
	}, nil
}

func (r *fakeProjectInvitationRepository) Delete(_ context.Context, id string) (int64, error) {
	r.deleteID = id
	return r.deleteRows, r.deleteErr
}

type fakeProjectInvitationProjectRepo struct {
	project generated.Project
	err     error
}

func (r *fakeProjectInvitationProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return r.project, r.err
}

type fakeProjectInvitationMemberRepo struct {
	memberByProjectAndUser generated.ProjectMember
	memberByProjectUserErr error
	createParams           generated.CreateProjectMemberParams
	createErr              error
}

func (r *fakeProjectInvitationMemberRepo) Create(_ context.Context, params generated.CreateProjectMemberParams) (generated.ProjectMember, error) {
	r.createParams = params
	if r.createErr != nil {
		return generated.ProjectMember{}, r.createErr
	}

	return generated.ProjectMember{
		ID:        params.ID,
		ProjectID: params.ProjectID,
		UserID:    params.UserID,
		Role:      params.Role,
	}, nil
}

func (r *fakeProjectInvitationMemberRepo) GetByProjectAndUser(_ context.Context, _ int64, _ string) (generated.ProjectMember, error) {
	return r.memberByProjectAndUser, r.memberByProjectUserErr
}

type fakeProjectInvitationUserRepo struct {
	markVerifiedUserID string
	markVerifiedErr    error
}

func (r *fakeProjectInvitationUserRepo) MarkEmailVerifiedByID(_ context.Context, id string) error {
	r.markVerifiedUserID = id
	return r.markVerifiedErr
}

type fakeProjectInviteEmailSender struct {
	to   string
	data emailpkg.ProjectInviteTemplateData
}

func (s *fakeProjectInviteEmailSender) SendProjectInvite(_ context.Context, to string, data emailpkg.ProjectInviteTemplateData) error {
	s.to = to
	s.data = data
	return nil
}

func newTestProjectInvitationService(
	invitationRepo projectInvitationRepository,
	projectRepo projectInvitationProjectRepository,
	memberRepo projectInvitationMemberRepository,
	userRepo projectInvitationUserRepository,
) *ProjectInvitationService {
	return &ProjectInvitationService{
		invitationRepo: invitationRepo,
		projectRepo:    projectRepo,
		memberRepo:     memberRepo,
		userRepo:       userRepo,
		tokenService:   NewVerificationTokenService(),
		now:            time.Now,
	}
}

func TestProjectInvitationServiceCreateStoresHashedTokenAndSendsInvite(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{}
	emailSender := &fakeProjectInviteEmailSender{}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{}).WithEmail(emailSender, "https://app.agrafa.co")
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC)
	}

	invitation, err := service.Create(context.Background(), types.CreateProjectInvitationInput{
		ProjectID:       1,
		Email:           "TEAMMATE@EXAMPLE.COM",
		Role:            "viewer",
		InvitedByUserID: "usr_1",
		InvitedByName:   "Alice",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if repo.createParams.Email != "teammate@example.com" {
		t.Fatalf("create email = %q", repo.createParams.Email)
	}
	if repo.createParams.Role != ProjectRoleViewer {
		t.Fatalf("create role = %q", repo.createParams.Role)
	}
	if !repo.createParams.ExpiresAt.Equal(time.Date(2026, time.April, 13, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("expiresAt = %s", repo.createParams.ExpiresAt)
	}
	if invitation.Email != "teammate@example.com" {
		t.Fatalf("invitation email = %q", invitation.Email)
	}

	acceptURL, err := url.Parse(emailSender.data.AcceptURL)
	if err != nil {
		t.Fatalf("Parse(AcceptURL) error = %v", err)
	}
	rawToken := acceptURL.Query().Get("token")
	if rawToken == "" {
		t.Fatal("raw invite token = empty")
	}
	if service.tokenService.HashToken(rawToken) != repo.createParams.TokenHash {
		t.Fatal("stored invite hash does not match emailed raw token")
	}
}

func TestProjectInvitationServiceCreateRejectsOwnerRole(t *testing.T) {
	t.Parallel()

	service := newTestProjectInvitationService(&fakeProjectInvitationRepository{}, &fakeProjectInvitationProjectRepo{}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	_, err := service.Create(context.Background(), types.CreateProjectInvitationInput{
		ProjectID:       1,
		Email:           "teammate@example.com",
		Role:            "owner",
		InvitedByUserID: "usr_1",
	})
	if !errors.Is(err, types.ErrInvalidProjectInvitationRole) {
		t.Fatalf("Create() error = %v, want ErrInvalidProjectInvitationRole", err)
	}
}

func TestProjectInvitationServiceCreateManyReturnsOneCreatedResultForSingleInvite(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	response, err := service.CreateMany(context.Background(), []types.CreateProjectInvitationInput{
		{ProjectID: 1, Email: "one@example.com", Role: "viewer", InvitedByUserID: "usr_1"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error = %v", err)
	}
	if response.ProjectID != 1 {
		t.Fatalf("projectID = %d, want 1", response.ProjectID)
	}
	if len(response.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(response.Results))
	}
	if response.Results[0].Status != "created" || response.Results[0].Invitation == nil {
		t.Fatalf("result = %#v", response.Results[0])
	}
	if len(repo.createCalls) != 1 {
		t.Fatalf("createCalls = %d, want 1", len(repo.createCalls))
	}
}

func TestProjectInvitationServiceCreateManyReturnsMultipleResultsAndAllowsPartialSuccess(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	response, err := service.CreateMany(context.Background(), []types.CreateProjectInvitationInput{
		{ProjectID: 1, Email: "one@example.com", Role: "viewer", InvitedByUserID: "usr_1"},
		{ProjectID: 1, Email: "bad-email", Role: "admin", InvitedByUserID: "usr_1"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error = %v", err)
	}
	if len(response.Results) != 2 {
		t.Fatalf("results = %d, want 2", len(response.Results))
	}
	if response.Results[0].Status != "created" || response.Results[0].Invitation == nil {
		t.Fatalf("first result = %#v", response.Results[0])
	}
	if response.Results[1].Status != "failed" || response.Results[1].ErrorCode == nil || *response.Results[1].ErrorCode != "invalid_email" {
		t.Fatalf("second result = %#v", response.Results[1])
	}
	if len(repo.createCalls) != 1 {
		t.Fatalf("createCalls = %d, want 1", len(repo.createCalls))
	}
}

func TestProjectInvitationServiceCreateManyRejectsEmptyBatch(t *testing.T) {
	t.Parallel()

	service := newTestProjectInvitationService(&fakeProjectInvitationRepository{}, &fakeProjectInvitationProjectRepo{}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	_, err := service.CreateMany(context.Background(), nil)
	if !errors.Is(err, types.ErrEmptyProjectInvitations) {
		t.Fatalf("CreateMany() error = %v, want ErrEmptyProjectInvitations", err)
	}
}

func TestProjectInvitationServiceCreateManyHandlesDuplicateEmailsInRequest(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	response, err := service.CreateMany(context.Background(), []types.CreateProjectInvitationInput{
		{ProjectID: 1, Email: "one@example.com", Role: "viewer", InvitedByUserID: "usr_1"},
		{ProjectID: 1, Email: "ONE@example.com", Role: "admin", InvitedByUserID: "usr_1"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error = %v", err)
	}
	if response.Results[0].Status != "created" {
		t.Fatalf("first result = %#v", response.Results[0])
	}
	if response.Results[1].Status != "failed" || response.Results[1].ErrorCode == nil || *response.Results[1].ErrorCode != "duplicate_in_request" {
		t.Fatalf("second result = %#v", response.Results[1])
	}
	if len(repo.createCalls) != 1 {
		t.Fatalf("createCalls = %d, want 1", len(repo.createCalls))
	}
}

func TestProjectInvitationServiceCreateManyReturnsAlreadyInvitedFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{
		activeInvitations: map[string]generated.ProjectInvitation{
			"1|one@example.com": {ID: "pinv_existing", ProjectID: 1, Email: "one@example.com", Role: ProjectRoleViewer},
		},
	}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	response, err := service.CreateMany(context.Background(), []types.CreateProjectInvitationInput{
		{ProjectID: 1, Email: "one@example.com", Role: "viewer", InvitedByUserID: "usr_1"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error = %v", err)
	}
	if response.Results[0].Status != "failed" || response.Results[0].ErrorCode == nil || *response.Results[0].ErrorCode != "already_invited" {
		t.Fatalf("result = %#v", response.Results[0])
	}
	if len(repo.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(repo.createCalls))
	}
}

func TestProjectInvitationServiceCreateManyRejectsOwnerRolePerItem(t *testing.T) {
	t.Parallel()

	repo := &fakeProjectInvitationRepository{}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{
		project: generated.Project{ID: 1, Name: "Agrafa Team"},
	}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	response, err := service.CreateMany(context.Background(), []types.CreateProjectInvitationInput{
		{ProjectID: 1, Email: "one@example.com", Role: "owner", InvitedByUserID: "usr_1"},
	})
	if err != nil {
		t.Fatalf("CreateMany() error = %v", err)
	}
	if response.Results[0].Status != "failed" || response.Results[0].ErrorCode == nil || *response.Results[0].ErrorCode != "invalid_role" {
		t.Fatalf("result = %#v", response.Results[0])
	}
	if len(repo.createCalls) != 0 {
		t.Fatalf("createCalls = %d, want 0", len(repo.createCalls))
	}
}

func TestProjectInvitationServiceGetByTokenRejectsExpiredInvitation(t *testing.T) {
	t.Parallel()

	service := newTestProjectInvitationService(&fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleViewer,
			ExpiresAt: time.Date(2026, time.April, 6, 8, 0, 0, 0, time.UTC),
		},
	}, &fakeProjectInvitationProjectRepo{}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC)
	}

	_, err := service.GetByToken(context.Background(), "expired-token")
	if !errors.Is(err, types.ErrInvalidProjectInvitation) {
		t.Fatalf("GetByToken() error = %v, want ErrInvalidProjectInvitation", err)
	}
}

func TestProjectInvitationServiceGetByTokenRejectsAcceptedInvitation(t *testing.T) {
	t.Parallel()

	service := newTestProjectInvitationService(&fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleViewer,
			ExpiresAt: time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
			AcceptedAt: sql.NullTime{
				Time:  time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC),
				Valid: true,
			},
		},
	}, &fakeProjectInvitationProjectRepo{}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	_, err := service.GetByToken(context.Background(), "accepted-token")
	if !errors.Is(err, types.ErrInvalidProjectInvitation) {
		t.Fatalf("GetByToken() error = %v, want ErrInvalidProjectInvitation", err)
	}
}

func TestProjectInvitationServiceAcceptRequiresMatchingEmail(t *testing.T) {
	t.Parallel()

	service := newTestProjectInvitationService(&fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleViewer,
			ExpiresAt: time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
		},
	}, &fakeProjectInvitationProjectRepo{}, &fakeProjectInvitationMemberRepo{}, &fakeProjectInvitationUserRepo{})

	_, err := service.Accept(context.Background(), "invite-token", generated.User{
		ID:    "usr_1",
		Email: "other@example.com",
	})
	if !errors.Is(err, types.ErrProjectInvitationEmailMismatch) {
		t.Fatalf("Accept() error = %v, want ErrProjectInvitationEmailMismatch", err)
	}
}

func TestProjectInvitationServiceAcceptCreatesMembership(t *testing.T) {
	t.Parallel()

	memberRepo := &fakeProjectInvitationMemberRepo{memberByProjectUserErr: sql.ErrNoRows}
	userRepo := &fakeProjectInvitationUserRepo{}
	repo := &fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleAdmin,
			ExpiresAt: time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
		},
	}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{}, memberRepo, userRepo)
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 30, 0, 0, time.UTC)
	}

	alreadyMember, err := service.Accept(context.Background(), "invite-token", generated.User{
		ID:            "usr_1",
		Email:         "teammate@example.com",
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if alreadyMember {
		t.Fatal("alreadyMember = true, want false")
	}
	if memberRepo.createParams.UserID != "usr_1" || memberRepo.createParams.ProjectID != 1 || memberRepo.createParams.Role != ProjectRoleAdmin {
		t.Fatalf("createParams = %#v", memberRepo.createParams)
	}
	if repo.markAcceptedID != "pinv_1" {
		t.Fatalf("markAcceptedID = %q", repo.markAcceptedID)
	}
	if userRepo.markVerifiedUserID != "usr_1" {
		t.Fatalf("markVerifiedUserID = %q, want usr_1", userRepo.markVerifiedUserID)
	}
}

func TestProjectInvitationServiceAcceptHandlesDuplicateMembershipCleanlyAndBlocksReuse(t *testing.T) {
	t.Parallel()

	memberRepo := &fakeProjectInvitationMemberRepo{
		memberByProjectAndUser: generated.ProjectMember{ID: "pm_existing"},
	}
	userRepo := &fakeProjectInvitationUserRepo{}
	repo := &fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleViewer,
			ExpiresAt: time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
		},
	}
	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{}, memberRepo, userRepo)
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 30, 0, 0, time.UTC)
	}

	alreadyMember, err := service.Accept(context.Background(), "invite-token", generated.User{
		ID:            "usr_1",
		Email:         "teammate@example.com",
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("Accept() first call error = %v", err)
	}
	if !alreadyMember {
		t.Fatal("alreadyMember = false, want true")
	}
	if memberRepo.createParams.ID != "" {
		t.Fatalf("createParams = %#v, want zero value", memberRepo.createParams)
	}
	if userRepo.markVerifiedUserID != "usr_1" {
		t.Fatalf("markVerifiedUserID = %q, want usr_1", userRepo.markVerifiedUserID)
	}

	_, err = service.Accept(context.Background(), "invite-token", generated.User{
		ID:            "usr_1",
		Email:         "teammate@example.com",
		EmailVerified: true,
	})
	if !errors.Is(err, types.ErrInvalidProjectInvitation) {
		t.Fatalf("Accept() second call error = %v, want ErrInvalidProjectInvitation", err)
	}
}

func TestProjectInvitationServiceAcceptSkipsVerificationWhenAlreadyVerified(t *testing.T) {
	t.Parallel()

	memberRepo := &fakeProjectInvitationMemberRepo{memberByProjectUserErr: sql.ErrNoRows}
	userRepo := &fakeProjectInvitationUserRepo{}
	repo := &fakeProjectInvitationRepository{
		invitationByToken: generated.GetProjectInvitationByTokenHashRow{
			ID:        "pinv_1",
			ProjectID: 1,
			Email:     "teammate@example.com",
			Role:      ProjectRoleViewer,
			ExpiresAt: time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC),
		},
	}

	service := newTestProjectInvitationService(repo, &fakeProjectInvitationProjectRepo{}, memberRepo, userRepo)
	_, err := service.Accept(context.Background(), "invite-token", generated.User{
		ID:            "usr_1",
		Email:         "teammate@example.com",
		EmailVerified: true,
	})
	if err != nil {
		t.Fatalf("Accept() error = %v", err)
	}
	if userRepo.markVerifiedUserID != "" {
		t.Fatalf("markVerifiedUserID = %q, want empty", userRepo.markVerifiedUserID)
	}
}
