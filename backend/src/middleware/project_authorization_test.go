package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/go-chi/chi/v5"
)

type fakeProjectPermissionAuthorizer struct {
	projectID  int64
	userID     string
	permission string
	role       string
	err        error
}

func (a *fakeProjectPermissionAuthorizer) RequireProjectPermission(_ context.Context, userID string, projectID int64, permission string) (string, error) {
	a.userID = userID
	a.projectID = projectID
	a.permission = permission
	if a.err != nil {
		return "", a.err
	}
	if a.role == "" {
		return services.ProjectRoleViewer, nil
	}
	return a.role, nil
}

func TestRequireProjectPermissionMembersRead(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{err: types.ErrForbidden}
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersRead,
		ProjectIDFromRequiredQueryParam("project_id"),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/project-members?project_id=12", nil)
	request = request.WithContext(context.WithValue(request.Context(), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if authorizer.permission != services.PermissionMembersRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersRead)
	}
	if authorizer.projectID != 12 {
		t.Fatalf("projectID = %d, want 12", authorizer.projectID)
	}
}

func TestRequireProjectPermissionMembersManageFromBody(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{role: services.ProjectRoleOwner}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersManage,
		ProjectIDFromBodyField("project_id"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"project_id":12`) {
			t.Fatalf("request body was not preserved, got %q", string(body))
		}
		if !appdb.HasRLSSessionContext(r.Context()) {
			t.Fatal("expected RLS session context to be attached")
		}
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/project-members", strings.NewReader(`{"project_id":12,"user_id":"usr_2","role":"viewer"}`))
	request = request.WithContext(context.WithValue(request.Context(), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionMembersManage {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersManage)
	}
	if authorizer.projectID != 12 {
		t.Fatalf("projectID = %d, want 12", authorizer.projectID)
	}
}

func TestRequireProjectPermissionProjectInvitationsManageFromBody(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersManage,
		ProjectIDFromBodyField("project_id"),
	)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"project_id":21`) {
			t.Fatalf("request body was not preserved, got %q", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPost, "/v1/project-invitations", strings.NewReader(`{"project_id":21,"email":"teammate@example.com","role":"viewer"}`))
	request = request.WithContext(context.WithValue(request.Context(), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionMembersManage {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersManage)
	}
	if authorizer.projectID != 21 {
		t.Fatalf("projectID = %d, want 21", authorizer.projectID)
	}
}

func TestRequireProjectPermissionMembersManageFromStringResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersManage,
		ProjectIDFromURLParamStringResource("id", func(_ context.Context, id string) (int64, error) {
			if id != "pm_123" {
				t.Fatalf("resolver id = %q, want pm_123", id)
			}
			return 19, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/v1/project-members/pm_123", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "pm_123")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionMembersManage {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersManage)
	}
	if authorizer.projectID != 19 {
		t.Fatalf("projectID = %d, want 19", authorizer.projectID)
	}
}

func TestRequireProjectPermissionMembersManageDeleteFromStringResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{err: types.ErrForbidden}
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersManage,
		ProjectIDFromURLParamStringResource("id", func(_ context.Context, id string) (int64, error) {
			if id != "pm_456" {
				t.Fatalf("resolver id = %q, want pm_456", id)
			}
			return 33, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodDelete, "/v1/project-members/pm_456", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "pm_456")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if authorizer.permission != services.PermissionMembersManage {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersManage)
	}
	if authorizer.projectID != 33 {
		t.Fatalf("projectID = %d, want 33", authorizer.projectID)
	}
}

func TestRequireProjectPermissionProjectReadFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{err: types.ErrForbidden}
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionProjectRead,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 12 {
				t.Fatalf("resolver id = %d, want 12", id)
			}
			return 12, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/projects/12", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "12")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	if authorizer.permission != services.PermissionProjectRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionProjectRead)
	}
	if authorizer.projectID != 12 {
		t.Fatalf("projectID = %d, want 12", authorizer.projectID)
	}
}

func TestRequireProjectPermissionServicesReadFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionServicesRead,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 21 {
				t.Fatalf("resolver id = %d, want 21", id)
			}
			return 8, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/services/21", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "21")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionServicesRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionServicesRead)
	}
	if authorizer.projectID != 8 {
		t.Fatalf("projectID = %d, want 8", authorizer.projectID)
	}
}

func TestRequireProjectPermissionProjectUpdateFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionProjectUpdate,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 17 {
				t.Fatalf("resolver id = %d, want 17", id)
			}
			return 17, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/v1/projects/17", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "17")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionProjectUpdate {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionProjectUpdate)
	}
	if authorizer.projectID != 17 {
		t.Fatalf("projectID = %d, want 17", authorizer.projectID)
	}
}

func TestRequireProjectPermissionProjectMemberReadFromStringResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionMembersRead,
		ProjectIDFromURLParamStringResource("id", func(_ context.Context, id string) (int64, error) {
			if id != "pm_789" {
				t.Fatalf("resolver id = %q, want pm_789", id)
			}
			return 29, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/project-members/pm_789", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "pm_789")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionMembersRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionMembersRead)
	}
	if authorizer.projectID != 29 {
		t.Fatalf("projectID = %d, want 29", authorizer.projectID)
	}
}

func TestRequireProjectPermissionNodesReadFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionNodesRead,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 31 {
				t.Fatalf("resolver id = %d, want 31", id)
			}
			return 12, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/nodes/31", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "31")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionNodesRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionNodesRead)
	}
	if authorizer.projectID != 12 {
		t.Fatalf("projectID = %d, want 12", authorizer.projectID)
	}
}

func TestRequireProjectPermissionNodesWritePatchFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionNodesWrite,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 31 {
				t.Fatalf("resolver id = %d, want 31", id)
			}
			return 12, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/v1/nodes/31", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "31")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionNodesWrite {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionNodesWrite)
	}
}

func TestRequireProjectPermissionServicesWritePatchFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionServicesWrite,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 55 {
				t.Fatalf("resolver id = %d, want 55", id)
			}
			return 44, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodPatch, "/v1/services/55", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "55")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionServicesWrite {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionServicesWrite)
	}
	if authorizer.projectID != 44 {
		t.Fatalf("projectID = %d, want 44", authorizer.projectID)
	}
}

func TestRequireProjectPermissionAlertsReadFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionAlertsRead,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 91 {
				t.Fatalf("resolver id = %d, want 91", id)
			}
			return 23, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/alert-rules/91", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "91")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionAlertsRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionAlertsRead)
	}
	if authorizer.projectID != 23 {
		t.Fatalf("projectID = %d, want 23", authorizer.projectID)
	}
}

func TestRequireProjectPermissionNotificationRecipientsReadFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionNotificationRecipientsRead,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 77 {
				t.Fatalf("resolver id = %d, want 77", id)
			}
			return 15, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	request := httptest.NewRequest(http.MethodGet, "/v1/notification-recipients/77", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "77")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionNotificationRecipientsRead {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionNotificationRecipientsRead)
	}
	if authorizer.projectID != 15 {
		t.Fatalf("projectID = %d, want 15", authorizer.projectID)
	}
}

func TestRequireProjectPermissionNotificationRecipientsWriteDeleteFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionNotificationRecipientsWrite,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 77 {
				t.Fatalf("resolver id = %d, want 77", id)
			}
			return 15, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodDelete, "/v1/notification-recipients/77", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "77")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionNotificationRecipientsWrite {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionNotificationRecipientsWrite)
	}
	if authorizer.projectID != 15 {
		t.Fatalf("projectID = %d, want 15", authorizer.projectID)
	}
}

func TestRequireProjectPermissionAlertsWriteDeleteFromResource(t *testing.T) {
	t.Parallel()

	authorizer := &fakeProjectPermissionAuthorizer{}
	nextCalled := false
	handler := RequireProjectPermission(
		authorizer,
		services.PermissionAlertsWrite,
		ProjectIDFromURLParamResource("id", func(_ context.Context, id int64) (int64, error) {
			if id != 91 {
				t.Fatalf("resolver id = %d, want 91", id)
			}
			return 23, nil
		}),
	)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodDelete, "/v1/alert-rules/91", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "91")
	request = request.WithContext(context.WithValue(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext), authenticatedUserContextKey{}, generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if !nextCalled {
		t.Fatal("expected next handler to be called")
	}
	if authorizer.permission != services.PermissionAlertsWrite {
		t.Fatalf("permission = %q, want %q", authorizer.permission, services.PermissionAlertsWrite)
	}
	if authorizer.projectID != 23 {
		t.Fatalf("projectID = %d, want 23", authorizer.projectID)
	}
}
