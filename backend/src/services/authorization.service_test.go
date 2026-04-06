package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAuthorizationProjectMemberRepo struct {
	member     generated.ProjectMember
	memberByID generated.ProjectMember
	err        error
	errByID    error
}

func (r *fakeAuthorizationProjectMemberRepo) GetByProjectAndUser(_ context.Context, _ int64, _ string) (generated.ProjectMember, error) {
	return r.member, r.err
}

func (r *fakeAuthorizationProjectMemberRepo) GetByID(_ context.Context, _ string) (generated.ProjectMember, error) {
	return r.memberByID, r.errByID
}

type fakeAuthorizationNodeRepo struct {
	node generated.Node
	err  error
}

func (r *fakeAuthorizationNodeRepo) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return r.node, r.err
}

type fakeAuthorizationProjectRepo struct {
	project generated.Project
	err     error
}

func (r *fakeAuthorizationProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return r.project, r.err
}

type fakeAuthorizationServiceRepo struct {
	service generated.Service
	err     error
}

func (r *fakeAuthorizationServiceRepo) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return r.service, r.err
}

type fakeAuthorizationAlertRuleRepo struct {
	rule generated.AlertRule
	err  error
}

func (r *fakeAuthorizationAlertRuleRepo) GetByID(_ context.Context, _ int64) (generated.AlertRule, error) {
	return r.rule, r.err
}

type fakeAuthorizationNotificationRecipientRepo struct {
	recipient generated.NotificationRecipient
	err       error
}

func (r *fakeAuthorizationNotificationRecipientRepo) GetByID(_ context.Context, _ int64) (generated.NotificationRecipient, error) {
	return r.recipient, r.err
}

func TestAuthorizationServiceRequireProjectPermission(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		memberRepo *fakeAuthorizationProjectMemberRepo
		permission string
		wantErr    error
	}{
		{
			name: "missing membership is forbidden",
			memberRepo: &fakeAuthorizationProjectMemberRepo{
				err: sql.ErrNoRows,
			},
			permission: PermissionProjectRead,
			wantErr:    types.ErrForbidden,
		},
		{
			name: "viewer write is forbidden",
			memberRepo: &fakeAuthorizationProjectMemberRepo{
				member: generated.ProjectMember{Role: ProjectRoleViewer},
			},
			permission: PermissionNodesWrite,
			wantErr:    types.ErrForbidden,
		},
		{
			name: "admin operational write is allowed",
			memberRepo: &fakeAuthorizationProjectMemberRepo{
				member: generated.ProjectMember{Role: ProjectRoleAdmin},
			},
			permission: PermissionAlertsWrite,
		},
		{
			name: "admin project delete is forbidden",
			memberRepo: &fakeAuthorizationProjectMemberRepo{
				member: generated.ProjectMember{Role: ProjectRoleAdmin},
			},
			permission: PermissionProjectDelete,
			wantErr:    types.ErrForbidden,
		},
		{
			name: "owner project delete is allowed",
			memberRepo: &fakeAuthorizationProjectMemberRepo{
				member: generated.ProjectMember{Role: ProjectRoleOwner},
			},
			permission: PermissionProjectDelete,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			service := &AuthorizationService{
				projectMemberRepo:         testCase.memberRepo,
				projectRepo:               &fakeAuthorizationProjectRepo{},
				nodeRepo:                  &fakeAuthorizationNodeRepo{},
				serviceRepo:               &fakeAuthorizationServiceRepo{},
				alertRuleRepo:             &fakeAuthorizationAlertRuleRepo{},
				notificationRecipientRepo: &fakeAuthorizationNotificationRecipientRepo{},
			}

			err := service.RequireProjectPermission(context.Background(), "usr_1", 42, testCase.permission)
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("RequireProjectPermission() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}

func TestAuthorizationServiceProjectResourceLookups(t *testing.T) {
	t.Parallel()

	service := &AuthorizationService{
		projectMemberRepo: &fakeAuthorizationProjectMemberRepo{
			memberByID: generated.ProjectMember{ProjectID: 44},
		},
		projectRepo: &fakeAuthorizationProjectRepo{
			project: generated.Project{ID: 55},
		},
		nodeRepo: &fakeAuthorizationNodeRepo{
			node: generated.Node{ProjectID: 11},
		},
		serviceRepo: &fakeAuthorizationServiceRepo{
			service: generated.Service{ProjectID: 66},
		},
		alertRuleRepo: &fakeAuthorizationAlertRuleRepo{
			rule: generated.AlertRule{ProjectID: 22},
		},
		notificationRecipientRepo: &fakeAuthorizationNotificationRecipientRepo{
			recipient: generated.NotificationRecipient{ProjectID: 33},
		},
	}

	nodeProjectID, err := service.ProjectIDForNode(context.Background(), 1)
	if err != nil {
		t.Fatalf("ProjectIDForNode() error = %v", err)
	}
	if nodeProjectID != 11 {
		t.Fatalf("ProjectIDForNode() = %d, want 11", nodeProjectID)
	}

	projectID, err := service.ProjectIDForProject(context.Background(), 55)
	if err != nil {
		t.Fatalf("ProjectIDForProject() error = %v", err)
	}
	if projectID != 55 {
		t.Fatalf("ProjectIDForProject() = %d, want 55", projectID)
	}

	serviceProjectID, err := service.ProjectIDForService(context.Background(), 6)
	if err != nil {
		t.Fatalf("ProjectIDForService() error = %v", err)
	}
	if serviceProjectID != 66 {
		t.Fatalf("ProjectIDForService() = %d, want 66", serviceProjectID)
	}

	alertRuleProjectID, err := service.ProjectIDForAlertRule(context.Background(), 2)
	if err != nil {
		t.Fatalf("ProjectIDForAlertRule() error = %v", err)
	}
	if alertRuleProjectID != 22 {
		t.Fatalf("ProjectIDForAlertRule() = %d, want 22", alertRuleProjectID)
	}

	recipientProjectID, err := service.ProjectIDForNotificationRecipient(context.Background(), 3)
	if err != nil {
		t.Fatalf("ProjectIDForNotificationRecipient() error = %v", err)
	}
	if recipientProjectID != 33 {
		t.Fatalf("ProjectIDForNotificationRecipient() = %d, want 33", recipientProjectID)
	}

	projectMemberProjectID, err := service.ProjectIDForProjectMember(context.Background(), "pm_1")
	if err != nil {
		t.Fatalf("ProjectIDForProjectMember() error = %v", err)
	}
	if projectMemberProjectID != 44 {
		t.Fatalf("ProjectIDForProjectMember() = %d, want 44", projectMemberProjectID)
	}
}
