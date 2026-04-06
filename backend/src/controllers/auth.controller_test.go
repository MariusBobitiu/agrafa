package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	authmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/go-chi/chi/v5"
)

type fakeAuthControllerService struct {
	registerUser      generated.User
	registerToken     string
	registerExpiry    time.Time
	registerErr       error
	loginUser         generated.User
	loginToken        string
	loginExpiry       time.Time
	loginErr          error
	sessions          []types.AuthUserSessionData
	sessionsErr       error
	logoutAllErr      error
	deleteCurrent     bool
	deleteErr         error
	completeUser      generated.User
	completeErr       error
	sendVerifyErr     error
	confirmVerifyErr  error
	forgotPasswordErr error
	resetPasswordErr  error
	verifyPasswordErr error
	authenticateErr   error
}

func (s *fakeAuthControllerService) Register(_ context.Context, _ types.RegisterInput, _ types.SessionActor) (generated.User, string, time.Time, error) {
	return s.registerUser, s.registerToken, s.registerExpiry, s.registerErr
}

func (s *fakeAuthControllerService) Login(_ context.Context, _ types.LoginInput, _ types.SessionActor) (generated.User, string, time.Time, error) {
	return s.loginUser, s.loginToken, s.loginExpiry, s.loginErr
}

func (s *fakeAuthControllerService) Logout(_ context.Context, _ string) error {
	return nil
}

func (s *fakeAuthControllerService) CompleteOnboarding(_ context.Context, _ string) (generated.User, error) {
	return s.completeUser, s.completeErr
}

func (s *fakeAuthControllerService) SendVerifyEmail(_ context.Context, _ generated.User) error {
	return s.sendVerifyErr
}

func (s *fakeAuthControllerService) ConfirmVerifyEmail(_ context.Context, _ string) error {
	return s.confirmVerifyErr
}

func (s *fakeAuthControllerService) ForgotPassword(_ context.Context, _ string) error {
	return s.forgotPasswordErr
}

func (s *fakeAuthControllerService) ResetPassword(_ context.Context, _ string, _ string) error {
	return s.resetPasswordErr
}

func (s *fakeAuthControllerService) VerifyPassword(_ context.Context, _ string, _ string) error {
	return s.verifyPasswordErr
}

func (s *fakeAuthControllerService) ListSessions(_ context.Context, _ string, _ string) ([]types.AuthUserSessionData, error) {
	return s.sessions, s.sessionsErr
}

func (s *fakeAuthControllerService) LogoutAll(_ context.Context, _ string, _ string) error {
	return s.logoutAllErr
}

func (s *fakeAuthControllerService) DeleteSession(_ context.Context, _ string, _ string, _ string) (bool, error) {
	return s.deleteCurrent, s.deleteErr
}

func (s *fakeAuthControllerService) Authenticate(_ context.Context, _ string) (generated.User, time.Time, error) {
	return generated.User{}, time.Time{}, s.authenticateErr
}

func TestAuthControllerMeRequiresValidSession(t *testing.T) {
	t.Parallel()

	controller := NewAuthController(
		&fakeAuthControllerService{authenticateErr: types.ErrUnauthenticated},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	request.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: "stale-token"})
	recorder := httptest.NewRecorder()

	controller.Me(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "authentication required") {
		t.Fatalf("body = %s", recorder.Body.String())
	}
	if cookie := recorder.Result().Cookies()[0]; cookie.Name != services.SessionCookieName || cookie.MaxAge != -1 {
		t.Fatalf("clear cookie not set correctly: %+v", cookie)
	}
}

func TestAuthControllerLoginSetsSessionCookie(t *testing.T) {
	t.Parallel()

	expiresAt := time.Date(2026, time.April, 12, 12, 0, 0, 0, time.UTC)
	controller := NewAuthController(
		&fakeAuthControllerService{
			loginUser: generated.User{
				ID:    "usr_1",
				Name:  "Alice",
				Email: "alice@example.com",
			},
			loginToken:  "raw-session-token",
			loginExpiry: expiresAt,
		},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/login", strings.NewReader(`{"email":"alice@example.com","password":"supersecret","remember_me":true}`))
	recorder := httptest.NewRecorder()

	controller.Login(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != services.SessionCookieName {
		t.Fatalf("cookie name = %q", cookies[0].Name)
	}
	if cookies[0].Value != "raw-session-token" {
		t.Fatalf("cookie value = %q", cookies[0].Value)
	}
	if !cookies[0].Expires.Equal(expiresAt) {
		t.Fatalf("cookie expires = %s, want %s", cookies[0].Expires, expiresAt)
	}
}

func TestAuthControllerLogoutAllKeepsCurrentCookie(t *testing.T) {
	t.Parallel()

	controller := NewAuthController(
		&fakeAuthControllerService{},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/logout-all", nil)
	request.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: "current-token"})
	request = request.WithContext(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	controller.LogoutAll(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if len(recorder.Result().Cookies()) != 0 {
		t.Fatalf("cookies = %d, want 0", len(recorder.Result().Cookies()))
	}
}

func TestAuthControllerDeleteCurrentSessionClearsCookie(t *testing.T) {
	t.Parallel()

	controller := NewAuthController(
		&fakeAuthControllerService{deleteCurrent: true},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodDelete, "/v1/auth/sessions/ses_current", nil)
	request.AddCookie(&http.Cookie{Name: services.SessionCookieName, Value: "current-token"})
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("id", "ses_current")
	request = request.WithContext(context.WithValue(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1"}), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	controller.DeleteSession(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	cookies := recorder.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	if cookies[0].Name != services.SessionCookieName || cookies[0].MaxAge != -1 {
		t.Fatalf("clear cookie not set correctly: %+v", cookies[0])
	}
}

func TestAuthControllerForgotPasswordReturnsGenericSuccess(t *testing.T) {
	t.Parallel()

	controller := NewAuthController(
		&fakeAuthControllerService{},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/forgot-password", strings.NewReader(`{"email":"missing@example.com"}`))
	recorder := httptest.NewRecorder()

	controller.ForgotPassword(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"status":"ok"`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}

func TestAuthControllerCompleteOnboardingReturnsUpdatedUser(t *testing.T) {
	t.Parallel()

	controller := NewAuthController(
		&fakeAuthControllerService{
			completeUser: generated.User{
				ID:                  "usr_1",
				Name:                "Alice",
				Email:               "alice@example.com",
				OnboardingCompleted: true,
			},
		},
		services.NewSessionService(7*24*time.Hour, 30*24*time.Hour, false),
	)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/onboarding/complete", nil)
	request = request.WithContext(authmiddleware.WithAuthenticatedUser(request.Context(), generated.User{ID: "usr_1"}))
	recorder := httptest.NewRecorder()

	controller.CompleteOnboarding(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"onboarding_completed":true`) {
		t.Fatalf("body = %s", recorder.Body.String())
	}
}
