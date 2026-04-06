package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	authmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeProjectInvitationControllerService struct {
	createManyInputs []types.CreateProjectInvitationInput
	createManyResult types.ProjectInvitationCreateBatchData
	createManyErr    error
}

func (s *fakeProjectInvitationControllerService) Create(_ context.Context, _ types.CreateProjectInvitationInput) (types.ProjectInvitationReadData, error) {
	return types.ProjectInvitationReadData{}, nil
}

func (s *fakeProjectInvitationControllerService) CreateMany(_ context.Context, inputs []types.CreateProjectInvitationInput) (types.ProjectInvitationCreateBatchData, error) {
	s.createManyInputs = inputs
	return s.createManyResult, s.createManyErr
}

func (s *fakeProjectInvitationControllerService) List(_ context.Context, _ int64) ([]types.ProjectInvitationReadData, error) {
	return nil, nil
}

func (s *fakeProjectInvitationControllerService) GetByToken(_ context.Context, _ string) (types.ProjectInvitationLookupData, error) {
	return types.ProjectInvitationLookupData{}, nil
}

func (s *fakeProjectInvitationControllerService) Accept(_ context.Context, _ string, _ generated.User) (bool, error) {
	return false, nil
}

func (s *fakeProjectInvitationControllerService) Delete(_ context.Context, _ string) error {
	return nil
}

func TestProjectInvitationControllerAcceptRequiresAuthenticatedUser(t *testing.T) {
	t.Parallel()

	controller := NewProjectInvitationController(&fakeProjectInvitationControllerService{})
	request := httptest.NewRequest(http.MethodPost, "/v1/project-invitations/accept", nil)
	recorder := httptest.NewRecorder()

	controller.Accept(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
}

func TestProjectInvitationControllerCreateAcceptsBatchPayload(t *testing.T) {
	t.Parallel()

	service := &fakeProjectInvitationControllerService{
		createManyResult: types.ProjectInvitationCreateBatchData{
			ProjectID: 1,
			Results: []types.ProjectInvitationCreateResultData{
				{
					Email:      "one@example.com",
					Role:       "viewer",
					Status:     "created",
					Invitation: &types.ProjectInvitationReadData{ID: "pinv_1", ProjectID: 1, Email: "one@example.com", Role: "viewer"},
				},
				{
					Email:      "two@example.com",
					Role:       "admin",
					Status:     "created",
					Invitation: &types.ProjectInvitationReadData{ID: "pinv_2", ProjectID: 1, Email: "two@example.com", Role: "admin"},
				},
			},
		},
	}
	controller := NewProjectInvitationController(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/project-invitations", strings.NewReader(`{"project_id":1,"invitations":[{"email":"one@example.com","role":"viewer"},{"email":"two@example.com","role":"admin"}]}`))
	request = request.WithContext(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1", Name: "Alice"}))
	recorder := httptest.NewRecorder()

	controller.Create(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(service.createManyInputs) != 2 {
		t.Fatalf("createManyInputs = %d, want 2", len(service.createManyInputs))
	}
	if service.createManyInputs[0].ProjectID != 1 || service.createManyInputs[0].InvitedByUserID != "usr_1" {
		t.Fatalf("first input = %#v", service.createManyInputs[0])
	}
	if !strings.Contains(recorder.Body.String(), `"results"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestProjectInvitationControllerCreateAcceptsSinglePayload(t *testing.T) {
	t.Parallel()

	service := &fakeProjectInvitationControllerService{
		createManyResult: types.ProjectInvitationCreateBatchData{
			ProjectID: 1,
			Results: []types.ProjectInvitationCreateResultData{
				{
					Email:      "one@example.com",
					Role:       "viewer",
					Status:     "created",
					Invitation: &types.ProjectInvitationReadData{ID: "pinv_1", ProjectID: 1, Email: "one@example.com", Role: "viewer"},
				},
			},
		},
	}
	controller := NewProjectInvitationController(service)
	request := httptest.NewRequest(http.MethodPost, "/v1/project-invitations", strings.NewReader(`{"project_id":1,"email":"one@example.com","role":"viewer"}`))
	request = request.WithContext(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1", Name: "Alice"}))
	recorder := httptest.NewRecorder()

	controller.Create(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(service.createManyInputs) != 1 {
		t.Fatalf("createManyInputs = %d, want 1", len(service.createManyInputs))
	}
	if !strings.Contains(recorder.Body.String(), `"status":"created"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestProjectInvitationControllerCreateRejectsMalformedPayload(t *testing.T) {
	t.Parallel()

	controller := NewProjectInvitationController(&fakeProjectInvitationControllerService{})
	request := httptest.NewRequest(http.MethodPost, "/v1/project-invitations", strings.NewReader(`{"project_id":1}`))
	request = request.WithContext(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1", Name: "Alice"}))
	recorder := httptest.NewRecorder()

	controller.Create(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}
