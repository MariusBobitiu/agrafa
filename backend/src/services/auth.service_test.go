package services

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	emailpkg "github.com/MariusBobitiu/agrafa-backend/src/email"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAuthStore struct {
	registerUser                generated.User
	registerErr                 error
	markEmailVerifiedUserID      string
	markEmailVerifiedErr         error
	userWithPassword            generated.GetUserWithPasswordByEmailRow
	userWithPasswordErr         error
	userWithPasswordByID        generated.GetUserWithPasswordByIDRow
	userWithPasswordByIDErr     error
	completedOnboardingUser     generated.User
	completedOnboardingErr      error
	completedOnboardingUserID   string
	createdSessions             []generated.CreateSessionParams
	createSessionErr            error
	sessionByToken              generated.Session
	sessionByTokenErr           error
	sessionUser                 generated.GetSessionUserByTokenHashRow
	sessionUserErr              error
	listSessionsUserID          string
	listSessions                []generated.Session
	listSessionsErr             error
	deletedSessionTokenHashes   []string
	deleteSessionByTokenHashErr error
	deleteSessionByIDParams     generated.DeleteSessionByIDAndUserParams
	deleteSessionByIDRows       int64
	deleteSessionByIDErr        error
	deleteOtherSessionsParams   generated.DeleteOtherSessionsByUserParams
	deleteOtherSessionsRows     int64
	deleteOtherSessionsErr      error
	replacedVerificationTokens  []generated.CreateVerificationTokenParams
	replaceVerificationTokenErr error
	verificationToken           generated.VerificationToken
	verificationTokenErr        error
	verificationTokenHash       string
	verificationTokenType       string
	verificationTokenSingleUse  bool
	verificationTokenConsumed   bool
	deletedVerificationTokenIDs []string
	deleteVerificationTokenRows int64
	deleteVerificationTokenErr  error
	confirmedTokenID            string
	confirmedUserID             string
	confirmEmailErr             error
	resetPasswordTokenID        string
	resetPasswordUserID         string
	resetPasswordHash           string
	resetPasswordErr            error
}

func (r *fakeAuthStore) Register(_ context.Context, _ generated.CreateUserParams, _ generated.CreatePasswordCredentialParams, _ generated.CreateSessionParams) (generated.User, error) {
	return r.registerUser, r.registerErr
}

func (r *fakeAuthStore) MarkEmailVerifiedByID(_ context.Context, id string) error {
	r.markEmailVerifiedUserID = id
	return r.markEmailVerifiedErr
}

func (r *fakeAuthStore) GetUserWithPasswordByEmail(_ context.Context, _ string) (generated.GetUserWithPasswordByEmailRow, error) {
	return r.userWithPassword, r.userWithPasswordErr
}

func (r *fakeAuthStore) GetUserWithPasswordByID(_ context.Context, _ string) (generated.GetUserWithPasswordByIDRow, error) {
	return r.userWithPasswordByID, r.userWithPasswordByIDErr
}

func (r *fakeAuthStore) CompleteOnboardingByUserID(_ context.Context, id string) (generated.User, error) {
	r.completedOnboardingUserID = id
	return r.completedOnboardingUser, r.completedOnboardingErr
}

func (r *fakeAuthStore) CreateSession(_ context.Context, params generated.CreateSessionParams) (generated.Session, error) {
	r.createdSessions = append(r.createdSessions, params)
	if r.createSessionErr != nil {
		return generated.Session{}, r.createSessionErr
	}

	return generated.Session{
		ID:        params.ID,
		TokenHash: params.TokenHash,
		UserID:    params.UserID,
		ExpiresAt: params.ExpiresAt,
		IpAddress: params.IpAddress,
		UserAgent: params.UserAgent,
	}, nil
}

func (r *fakeAuthStore) GetSessionUserByTokenHash(_ context.Context, _ string) (generated.GetSessionUserByTokenHashRow, error) {
	return r.sessionUser, r.sessionUserErr
}

func (r *fakeAuthStore) GetSessionByTokenHash(_ context.Context, _ string) (generated.Session, error) {
	return r.sessionByToken, r.sessionByTokenErr
}

func (r *fakeAuthStore) DeleteSessionByTokenHash(_ context.Context, tokenHash string) error {
	r.deletedSessionTokenHashes = append(r.deletedSessionTokenHashes, tokenHash)
	return r.deleteSessionByTokenHashErr
}

func (r *fakeAuthStore) ListSessionsByUserID(_ context.Context, userID string) ([]generated.Session, error) {
	r.listSessionsUserID = userID
	return r.listSessions, r.listSessionsErr
}

func (r *fakeAuthStore) DeleteSessionByIDAndUser(_ context.Context, id string, userID string) (int64, error) {
	r.deleteSessionByIDParams = generated.DeleteSessionByIDAndUserParams{
		ID:     id,
		UserID: userID,
	}
	return r.deleteSessionByIDRows, r.deleteSessionByIDErr
}

func (r *fakeAuthStore) DeleteOtherSessionsByUser(_ context.Context, userID string, currentTokenHash string) (int64, error) {
	r.deleteOtherSessionsParams = generated.DeleteOtherSessionsByUserParams{
		UserID:    userID,
		TokenHash: currentTokenHash,
	}
	return r.deleteOtherSessionsRows, r.deleteOtherSessionsErr
}

func (r *fakeAuthStore) ReplaceVerificationToken(_ context.Context, params generated.CreateVerificationTokenParams) (generated.VerificationToken, error) {
	r.replacedVerificationTokens = append(r.replacedVerificationTokens, params)
	if r.replaceVerificationTokenErr != nil {
		return generated.VerificationToken{}, r.replaceVerificationTokenErr
	}

	return generated.VerificationToken{
		ID:         params.ID,
		UserID:     params.UserID,
		Identifier: params.Identifier,
		TokenHash:  params.TokenHash,
		Type:       params.Type,
		ExpiresAt:  params.ExpiresAt,
	}, nil
}

func (r *fakeAuthStore) GetVerificationTokenByTokenHashAndType(_ context.Context, tokenHash string, tokenType string) (generated.VerificationToken, error) {
	r.verificationTokenHash = tokenHash
	r.verificationTokenType = tokenType
	if r.verificationTokenSingleUse && r.verificationTokenConsumed {
		return generated.VerificationToken{}, sql.ErrNoRows
	}
	if r.verificationTokenErr != nil {
		return generated.VerificationToken{}, r.verificationTokenErr
	}

	return r.verificationToken, nil
}

func (r *fakeAuthStore) DeleteVerificationTokenByID(_ context.Context, tokenID string) (int64, error) {
	r.deletedVerificationTokenIDs = append(r.deletedVerificationTokenIDs, tokenID)
	if r.deleteVerificationTokenErr != nil {
		return 0, r.deleteVerificationTokenErr
	}
	if r.deleteVerificationTokenRows == 0 {
		return 1, nil
	}

	return r.deleteVerificationTokenRows, nil
}

func (r *fakeAuthStore) ConfirmEmailVerification(_ context.Context, tokenID string, userID string) error {
	r.confirmedTokenID = tokenID
	r.confirmedUserID = userID
	if r.confirmEmailErr != nil {
		return r.confirmEmailErr
	}
	if r.verificationTokenSingleUse && r.verificationTokenConsumed {
		return sql.ErrNoRows
	}

	r.verificationTokenConsumed = true
	return nil
}

func (r *fakeAuthStore) ResetPasswordWithToken(_ context.Context, tokenID string, userID string, passwordHash string) error {
	r.resetPasswordTokenID = tokenID
	r.resetPasswordUserID = userID
	r.resetPasswordHash = passwordHash
	if r.resetPasswordErr != nil {
		return r.resetPasswordErr
	}
	if r.verificationTokenSingleUse && r.verificationTokenConsumed {
		return sql.ErrNoRows
	}

	r.verificationTokenConsumed = true
	return nil
}

type fakeAuthSecurityEmailSender struct {
	verifyTo   string
	verifyName string
	verifyURL  string
	resetTo    string
	resetName  string
	resetURL   string
}

func (s *fakeAuthSecurityEmailSender) SendVerifyEmail(_ context.Context, to string, name string, verifyURL string) error {
	s.verifyTo = to
	s.verifyName = name
	s.verifyURL = verifyURL
	return nil
}

func (s *fakeAuthSecurityEmailSender) SendPasswordResetEmail(_ context.Context, to string, name string, resetURL string) error {
	s.resetTo = to
	s.resetName = name
	s.resetURL = resetURL
	return nil
}

type fakeAuthSecurityEmailProvider struct {
	service authSecurityEmailSender
	err     error
}

func (p *fakeAuthSecurityEmailProvider) Security(_ context.Context) (*emailpkg.Service, error) {
	if p.err != nil {
		return nil, p.err
	}
	if p.service == nil {
		return nil, nil
	}

	service, _ := p.service.(*emailpkg.Service)
	return service, nil
}

func TestAuthServiceRegisterAutoVerifiesWhenSecurityEmailUnavailable(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		registerUser: generated.User{
			ID:            "usr_1",
			Name:          "Alice",
			Email:         "alice@example.com",
			EmailVerified: false,
		},
	}

	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)).
		WithSecurityEmailProvider(&fakeAuthSecurityEmailProvider{}, "http://localhost:5173")

	user, token, expiresAt, err := service.Register(context.Background(), types.RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "supersecret",
	}, types.SessionActor{IPAddress: "127.0.0.1", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if token == "" {
		t.Fatal("token = empty, want non-empty")
	}
	if expiresAt.IsZero() {
		t.Fatal("expiresAt = zero, want non-zero")
	}
	if store.markEmailVerifiedUserID != "usr_1" {
		t.Fatalf("markEmailVerifiedUserID = %q, want usr_1", store.markEmailVerifiedUserID)
	}
	if !user.EmailVerified {
		t.Fatal("user.EmailVerified = false, want true")
	}
}

func TestAuthServiceSendVerifyEmailNoopsWhenProviderReturnsTypedNilService(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{}

	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)).
		WithSecurityEmailProvider(&fakeAuthSecurityEmailProvider{}, "http://localhost:5173")

	err := service.SendVerifyEmail(context.Background(), generated.User{
		ID:            "usr_1",
		Name:          "Alice",
		Email:         "alice@example.com",
		EmailVerified: false,
	})
	if err != nil {
		t.Fatalf("SendVerifyEmail() error = %v", err)
	}
}

func TestAuthServiceRegisterKeepsVerificationPendingWhenSecurityEmailConfigured(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		registerUser: generated.User{
			ID:            "usr_1",
			Name:          "Alice",
			Email:         "alice@example.com",
			EmailVerified: false,
		},
	}

	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)).
		WithSecurityEmail(&fakeAuthSecurityEmailSender{}, "http://localhost:5173")

	user, _, _, err := service.Register(context.Background(), types.RegisterInput{
		Name:     "Alice",
		Email:    "alice@example.com",
		Password: "supersecret",
	}, types.SessionActor{IPAddress: "127.0.0.1", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if store.markEmailVerifiedUserID != "" {
		t.Fatalf("markEmailVerifiedUserID = %q, want empty", store.markEmailVerifiedUserID)
	}
	if user.EmailVerified {
		t.Fatal("user.EmailVerified = true, want false")
	}
}

func TestAuthServiceLoginCreatesSessionWithDefaultTTL(t *testing.T) {
	t.Parallel()

	passwordService := NewPasswordService()
	passwordHash, err := passwordService.Hash("supersecret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	store := &fakeAuthStore{
		userWithPassword: generated.GetUserWithPasswordByEmailRow{
			ID:           "usr_1",
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: passwordHash,
		},
	}

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	sessionService.now = func() time.Time {
		return time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	}

	service := NewAuthService(store, passwordService, sessionService)

	user, token, expiresAt, err := service.Login(context.Background(), types.LoginInput{
		Email:    "alice@example.com",
		Password: "supersecret",
	}, types.SessionActor{IPAddress: "127.0.0.1", UserAgent: "test-agent"})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user.Email != "alice@example.com" {
		t.Fatalf("user.Email = %q", user.Email)
	}
	if token == "" {
		t.Fatal("token = empty, want non-empty")
	}
	if !expiresAt.Equal(time.Date(2026, time.April, 12, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expiresAt = %s", expiresAt)
	}
	if len(store.createdSessions) != 1 {
		t.Fatalf("createdSessions = %d, want 1", len(store.createdSessions))
	}
	if !store.createdSessions[0].ExpiresAt.Equal(expiresAt) {
		t.Fatalf("session expiresAt = %s, want %s", store.createdSessions[0].ExpiresAt, expiresAt)
	}
}

func TestAuthServiceLoginCreatesSessionWithRememberMeTTL(t *testing.T) {
	t.Parallel()

	passwordService := NewPasswordService()
	passwordHash, err := passwordService.Hash("supersecret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	store := &fakeAuthStore{
		userWithPassword: generated.GetUserWithPasswordByEmailRow{
			ID:           "usr_1",
			Name:         "Alice",
			Email:        "alice@example.com",
			PasswordHash: passwordHash,
		},
	}

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	sessionService.now = func() time.Time {
		return time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	}

	service := NewAuthService(store, passwordService, sessionService)

	_, _, expiresAt, err := service.Login(context.Background(), types.LoginInput{
		Email:      "alice@example.com",
		Password:   "supersecret",
		RememberMe: true,
	}, types.SessionActor{})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if !expiresAt.Equal(time.Date(2026, time.May, 5, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("expiresAt = %s", expiresAt)
	}
}

func TestAuthServiceLoginRejectsInvalidPassword(t *testing.T) {
	t.Parallel()

	passwordService := NewPasswordService()
	passwordHash, err := passwordService.Hash("supersecret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	service := NewAuthService(&fakeAuthStore{
		userWithPassword: generated.GetUserWithPasswordByEmailRow{
			ID:           "usr_1",
			Email:        "alice@example.com",
			PasswordHash: passwordHash,
		},
	}, passwordService, NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))

	_, _, _, err = service.Login(context.Background(), types.LoginInput{
		Email:    "alice@example.com",
		Password: "wrong-password",
	}, types.SessionActor{})
	if !errors.Is(err, types.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthServiceAuthenticateRejectsExpiredSession(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		sessionUser: generated.GetSessionUserByTokenHashRow{
			ID:        "usr_1",
			Email:     "alice@example.com",
			ExpiresAt: time.Date(2026, time.April, 5, 11, 0, 0, 0, time.UTC),
		},
	}

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	service := NewAuthService(store, NewPasswordService(), sessionService)
	service.now = func() time.Time {
		return time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	}

	_, _, err := service.Authenticate(context.Background(), "raw-token")
	if !errors.Is(err, types.ErrUnauthenticated) {
		t.Fatalf("Authenticate() error = %v, want ErrUnauthenticated", err)
	}
	if len(store.deletedSessionTokenHashes) != 1 {
		t.Fatalf("deletedSessionTokenHashes = %d, want 1", len(store.deletedSessionTokenHashes))
	}
}

func TestAuthServiceLogoutInvalidatesSession(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{}
	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	service := NewAuthService(store, NewPasswordService(), sessionService)

	if err := service.Logout(context.Background(), "raw-token"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}

	if len(store.deletedSessionTokenHashes) != 1 {
		t.Fatalf("deletedSessionTokenHashes = %d, want 1", len(store.deletedSessionTokenHashes))
	}
}

func TestAuthServiceListSessionsMarksCurrentAndScopesByUser(t *testing.T) {
	t.Parallel()

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	currentHash := sessionService.HashToken("current-token")
	store := &fakeAuthStore{
		listSessions: []generated.Session{
			{
				ID:        "ses_current",
				UserID:    "usr_1",
				TokenHash: currentHash,
				ExpiresAt: time.Date(2026, time.April, 12, 12, 0, 0, 0, time.UTC),
				CreatedAt: time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.April, 5, 12, 30, 0, 0, time.UTC),
			},
			{
				ID:        "ses_other",
				UserID:    "usr_1",
				TokenHash: sessionService.HashToken("other-token"),
				ExpiresAt: time.Date(2026, time.April, 10, 12, 0, 0, 0, time.UTC),
				CreatedAt: time.Date(2026, time.April, 4, 12, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, time.April, 4, 13, 0, 0, 0, time.UTC),
			},
		},
	}
	service := NewAuthService(store, NewPasswordService(), sessionService)

	sessions, err := service.ListSessions(context.Background(), "usr_1", "current-token")
	if err != nil {
		t.Fatalf("ListSessions() error = %v", err)
	}
	if store.listSessionsUserID != "usr_1" {
		t.Fatalf("listSessionsUserID = %q, want usr_1", store.listSessionsUserID)
	}
	if len(sessions) != 2 {
		t.Fatalf("len(sessions) = %d, want 2", len(sessions))
	}
	if !sessions[0].IsCurrent {
		t.Fatal("sessions[0].IsCurrent = false, want true")
	}
	if sessions[1].IsCurrent {
		t.Fatal("sessions[1].IsCurrent = true, want false")
	}
}

func TestAuthServiceDeleteSessionRejectsOtherUsersSession(t *testing.T) {
	t.Parallel()

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	store := &fakeAuthStore{
		sessionByToken: generated.Session{
			ID:        "ses_current",
			UserID:    "usr_1",
			TokenHash: sessionService.HashToken("current-token"),
		},
		deleteSessionByIDRows: 0,
	}
	service := NewAuthService(store, NewPasswordService(), sessionService)

	isCurrent, err := service.DeleteSession(context.Background(), "usr_1", "ses_other_user", "current-token")
	if !errors.Is(err, types.ErrSessionNotFound) {
		t.Fatalf("DeleteSession() error = %v, want ErrSessionNotFound", err)
	}
	if isCurrent {
		t.Fatal("isCurrent = true, want false")
	}
	if store.deleteSessionByIDParams.UserID != "usr_1" {
		t.Fatalf("deleteSessionByIDParams.UserID = %q, want usr_1", store.deleteSessionByIDParams.UserID)
	}
}

func TestAuthServiceDeleteSessionReturnsCurrentWhenDeletingCurrentSession(t *testing.T) {
	t.Parallel()

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	store := &fakeAuthStore{
		sessionByToken: generated.Session{
			ID:        "ses_current",
			UserID:    "usr_1",
			TokenHash: sessionService.HashToken("current-token"),
		},
		deleteSessionByIDRows: 1,
	}
	service := NewAuthService(store, NewPasswordService(), sessionService)

	isCurrent, err := service.DeleteSession(context.Background(), "usr_1", "ses_current", "current-token")
	if err != nil {
		t.Fatalf("DeleteSession() error = %v", err)
	}
	if !isCurrent {
		t.Fatal("isCurrent = false, want true")
	}
}

func TestAuthServiceLogoutAllDeletesOtherSessionsOnly(t *testing.T) {
	t.Parallel()

	sessionService := NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)
	store := &fakeAuthStore{}
	service := NewAuthService(store, NewPasswordService(), sessionService)

	if err := service.LogoutAll(context.Background(), "usr_1", "current-token"); err != nil {
		t.Fatalf("LogoutAll() error = %v", err)
	}

	if store.deleteOtherSessionsParams.UserID != "usr_1" {
		t.Fatalf("deleteOtherSessionsParams.UserID = %q, want usr_1", store.deleteOtherSessionsParams.UserID)
	}
	if store.deleteOtherSessionsParams.TokenHash != sessionService.HashToken("current-token") {
		t.Fatalf("deleteOtherSessionsParams.TokenHash = %q, want current token hash", store.deleteOtherSessionsParams.TokenHash)
	}
}

func TestAuthServiceLoginMissingUserReturnsInvalidCredentials(t *testing.T) {
	t.Parallel()

	service := NewAuthService(&fakeAuthStore{
		userWithPasswordErr: sql.ErrNoRows,
	}, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))

	_, _, _, err := service.Login(context.Background(), types.LoginInput{
		Email:    "alice@example.com",
		Password: "supersecret",
	}, types.SessionActor{})
	if !errors.Is(err, types.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthServiceSendVerifyEmailStoresHashedTokenAndSendsEmail(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{}
	emailSender := &fakeAuthSecurityEmailSender{}
	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)).
		WithSecurityEmail(emailSender, "https://app.agrafa.co")
	service.verificationTokenService.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC)
	}

	err := service.SendVerifyEmail(context.Background(), generated.User{
		ID:    "usr_1",
		Name:  "Alice",
		Email: "alice@example.com",
	})
	if err != nil {
		t.Fatalf("SendVerifyEmail() error = %v", err)
	}
	if len(store.replacedVerificationTokens) != 1 {
		t.Fatalf("replacedVerificationTokens = %d, want 1", len(store.replacedVerificationTokens))
	}

	tokenParams := store.replacedVerificationTokens[0]
	if tokenParams.Type != types.VerificationTokenTypeEmailVerification {
		t.Fatalf("token type = %q", tokenParams.Type)
	}
	if !tokenParams.ExpiresAt.Equal(time.Date(2026, time.April, 7, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("expiresAt = %s", tokenParams.ExpiresAt)
	}
	if tokenParams.Identifier != "alice@example.com" {
		t.Fatalf("identifier = %q", tokenParams.Identifier)
	}
	if emailSender.verifyTo != "alice@example.com" {
		t.Fatalf("verifyTo = %q", emailSender.verifyTo)
	}
	if emailSender.verifyName != "Alice" {
		t.Fatalf("verifyName = %q", emailSender.verifyName)
	}

	verifyURL, err := url.Parse(emailSender.verifyURL)
	if err != nil {
		t.Fatalf("Parse(verifyURL) error = %v", err)
	}
	rawToken := verifyURL.Query().Get("token")
	if rawToken == "" {
		t.Fatal("verify email raw token = empty")
	}
	if service.verificationTokenService.HashToken(rawToken) != tokenParams.TokenHash {
		t.Fatal("stored token hash does not match emailed raw token")
	}
}

func TestAuthServiceConfirmVerifyEmailRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		verificationToken: generated.VerificationToken{
			ID:        "vtk_1",
			UserID:    sql.NullString{String: "usr_1", Valid: true},
			ExpiresAt: time.Date(2026, time.April, 6, 8, 0, 0, 0, time.UTC),
		},
	}
	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC)
	}

	err := service.ConfirmVerifyEmail(context.Background(), "expired-token")
	if !errors.Is(err, types.ErrInvalidVerificationToken) {
		t.Fatalf("ConfirmVerifyEmail() error = %v, want ErrInvalidVerificationToken", err)
	}
	if len(store.deletedVerificationTokenIDs) != 1 || store.deletedVerificationTokenIDs[0] != "vtk_1" {
		t.Fatalf("deletedVerificationTokenIDs = %#v", store.deletedVerificationTokenIDs)
	}
}

func TestAuthServiceForgotPasswordReturnsGenericSuccessForMissingUser(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{userWithPasswordErr: sql.ErrNoRows}
	emailSender := &fakeAuthSecurityEmailSender{}
	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false)).
		WithSecurityEmail(emailSender, "https://app.agrafa.co")

	if err := service.ForgotPassword(context.Background(), "missing@example.com"); err != nil {
		t.Fatalf("ForgotPassword() error = %v", err)
	}
	if len(store.replacedVerificationTokens) != 0 {
		t.Fatalf("replacedVerificationTokens = %d, want 0", len(store.replacedVerificationTokens))
	}
	if emailSender.resetURL != "" {
		t.Fatalf("resetURL = %q, want empty", emailSender.resetURL)
	}
}

func TestAuthServiceResetPasswordUpdatesHashAndBlocksTokenReuse(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		verificationToken: generated.VerificationToken{
			ID:        "vtk_1",
			UserID:    sql.NullString{String: "usr_1", Valid: true},
			ExpiresAt: time.Date(2026, time.April, 6, 10, 0, 0, 0, time.UTC),
		},
		verificationTokenSingleUse: true,
	}
	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))
	service.now = func() time.Time {
		return time.Date(2026, time.April, 6, 9, 0, 0, 0, time.UTC)
	}

	if err := service.ResetPassword(context.Background(), "reset-token", "new-supersecret"); err != nil {
		t.Fatalf("ResetPassword() first call error = %v", err)
	}
	if store.resetPasswordTokenID != "vtk_1" {
		t.Fatalf("resetPasswordTokenID = %q", store.resetPasswordTokenID)
	}
	if store.resetPasswordUserID != "usr_1" {
		t.Fatalf("resetPasswordUserID = %q", store.resetPasswordUserID)
	}
	if store.resetPasswordHash == "" || strings.Contains(store.resetPasswordHash, "new-supersecret") {
		t.Fatalf("resetPasswordHash = %q, want argon2id hash", store.resetPasswordHash)
	}
	valid, err := service.passwordService.Verify("new-supersecret", store.resetPasswordHash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !valid {
		t.Fatal("resetPasswordHash does not verify the new password")
	}

	err = service.ResetPassword(context.Background(), "reset-token", "another-supersecret")
	if !errors.Is(err, types.ErrInvalidVerificationToken) {
		t.Fatalf("ResetPassword() second call error = %v, want ErrInvalidVerificationToken", err)
	}
}

func TestAuthServiceVerifyPasswordAcceptsCorrectPassword(t *testing.T) {
	t.Parallel()

	passwordService := NewPasswordService()
	passwordHash, err := passwordService.Hash("supersecret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	store := &fakeAuthStore{
		userWithPasswordByID: generated.GetUserWithPasswordByIDRow{
			ID:           "usr_1",
			Email:        "alice@example.com",
			PasswordHash: passwordHash,
		},
	}
	service := NewAuthService(store, passwordService, NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))

	if err := service.VerifyPassword(context.Background(), "usr_1", "supersecret"); err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
}

func TestAuthServiceVerifyPasswordRejectsIncorrectPassword(t *testing.T) {
	t.Parallel()

	passwordService := NewPasswordService()
	passwordHash, err := passwordService.Hash("supersecret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	store := &fakeAuthStore{
		userWithPasswordByID: generated.GetUserWithPasswordByIDRow{
			ID:           "usr_1",
			Email:        "alice@example.com",
			PasswordHash: passwordHash,
		},
	}
	service := NewAuthService(store, passwordService, NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))

	err = service.VerifyPassword(context.Background(), "usr_1", "wrong-password")
	if !errors.Is(err, types.ErrInvalidCredentials) {
		t.Fatalf("VerifyPassword() error = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthServiceCompleteOnboardingUpdatesFlag(t *testing.T) {
	t.Parallel()

	store := &fakeAuthStore{
		completedOnboardingUser: generated.User{
			ID:                  "usr_1",
			Email:               "alice@example.com",
			OnboardingCompleted: true,
		},
	}
	service := NewAuthService(store, NewPasswordService(), NewSessionService(7*24*time.Hour, 30*24*time.Hour, false))

	user, err := service.CompleteOnboarding(context.Background(), "usr_1")
	if err != nil {
		t.Fatalf("CompleteOnboarding() error = %v", err)
	}
	if store.completedOnboardingUserID != "usr_1" {
		t.Fatalf("completedOnboardingUserID = %q, want usr_1", store.completedOnboardingUserID)
	}
	if !user.OnboardingCompleted {
		t.Fatal("user.OnboardingCompleted = false, want true")
	}
}
