package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type authStore interface {
	Register(ctx context.Context, userParams generated.CreateUserParams, passwordParams generated.CreatePasswordCredentialParams, sessionParams generated.CreateSessionParams) (generated.User, error)
	GetUserWithPasswordByEmail(ctx context.Context, email string) (generated.GetUserWithPasswordByEmailRow, error)
	GetUserWithPasswordByID(ctx context.Context, id string) (generated.GetUserWithPasswordByIDRow, error)
	CompleteOnboardingByUserID(ctx context.Context, id string) (generated.User, error)
	CreateSession(ctx context.Context, params generated.CreateSessionParams) (generated.Session, error)
	GetSessionUserByTokenHash(ctx context.Context, tokenHash string) (generated.GetSessionUserByTokenHashRow, error)
	GetSessionByTokenHash(ctx context.Context, tokenHash string) (generated.Session, error)
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	ListSessionsByUserID(ctx context.Context, userID string) ([]generated.Session, error)
	DeleteSessionByIDAndUser(ctx context.Context, id string, userID string) (int64, error)
	DeleteOtherSessionsByUser(ctx context.Context, userID string, currentTokenHash string) (int64, error)
	ReplaceVerificationToken(ctx context.Context, params generated.CreateVerificationTokenParams) (generated.VerificationToken, error)
	GetVerificationTokenByTokenHashAndType(ctx context.Context, tokenHash string, tokenType string) (generated.VerificationToken, error)
	DeleteVerificationTokenByID(ctx context.Context, tokenID string) (int64, error)
	ConfirmEmailVerification(ctx context.Context, tokenID string, userID string) error
	ResetPasswordWithToken(ctx context.Context, tokenID string, userID string, passwordHash string) error
}

type authSecurityEmailSender interface {
	SendVerifyEmail(ctx context.Context, to string, name string, verifyURL string) error
	SendPasswordResetEmail(ctx context.Context, to string, name string, resetURL string) error
}

type AuthService struct {
	authRepo                 authStore
	passwordService          *PasswordService
	sessionService           *SessionService
	verificationTokenService *VerificationTokenService
	securityEmailService     authSecurityEmailSender
	appBaseURL               string
	now                      func() time.Time
}

func NewAuthService(authRepo authStore, passwordService *PasswordService, sessionService *SessionService) *AuthService {
	return &AuthService{
		authRepo:                 authRepo,
		passwordService:          passwordService,
		sessionService:           sessionService,
		verificationTokenService: NewVerificationTokenService(),
		now:                      time.Now,
	}
}

func (s *AuthService) WithSecurityEmail(emailService authSecurityEmailSender, appBaseURL string) *AuthService {
	s.securityEmailService = emailService
	s.appBaseURL = strings.TrimRight(strings.TrimSpace(appBaseURL), "/")
	return s
}

func (s *AuthService) WithVerificationTokenService(tokenService *VerificationTokenService) *AuthService {
	if tokenService != nil {
		s.verificationTokenService = tokenService
	}

	return s
}

func (s *AuthService) Register(ctx context.Context, input types.RegisterInput, actor types.SessionActor) (generated.User, string, time.Time, error) {
	name, email, password, err := s.validateRegisterInput(input)
	if err != nil {
		return generated.User{}, "", time.Time{}, err
	}

	passwordHash, err := s.passwordService.Hash(password)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("hash password: %w", err)
	}

	userID, err := utils.GenerateOpaqueID("usr", 16)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate user id: %w", err)
	}

	passwordCredentialID, err := utils.GenerateOpaqueID("pwd", 16)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate password credential id: %w", err)
	}

	sessionID, err := utils.GenerateOpaqueID("ses", 16)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate session id: %w", err)
	}

	rawSessionToken, err := s.sessionService.GenerateToken()
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := s.sessionService.ExpiresAt(false)
	user, err := s.authRepo.Register(
		ctx,
		generated.CreateUserParams{
			ID:    userID,
			Name:  name,
			Email: email,
		},
		generated.CreatePasswordCredentialParams{
			ID:           passwordCredentialID,
			UserID:       userID,
			PasswordHash: passwordHash,
		},
		generated.CreateSessionParams{
			ID:        sessionID,
			TokenHash: s.sessionService.HashToken(rawSessionToken),
			UserID:    userID,
			ExpiresAt: expiresAt,
			IpAddress: utils.ToNullString(utils.OptionalTrimmed(actor.IPAddress)),
			UserAgent: utils.ToNullString(utils.OptionalTrimmed(actor.UserAgent)),
		},
	)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("register user: %w", err)
	}

	return user, rawSessionToken, expiresAt, nil
}

func (s *AuthService) Login(ctx context.Context, input types.LoginInput, actor types.SessionActor) (generated.User, string, time.Time, error) {
	email, password, err := s.validateLoginInput(input)
	if err != nil {
		return generated.User{}, "", time.Time{}, err
	}

	row, err := s.authRepo.GetUserWithPasswordByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.User{}, "", time.Time{}, types.ErrInvalidCredentials
		}

		return generated.User{}, "", time.Time{}, fmt.Errorf("get user by email: %w", err)
	}

	isValid, err := s.passwordService.Verify(password, row.PasswordHash)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("verify password: %w", err)
	}
	if !isValid {
		return generated.User{}, "", time.Time{}, types.ErrInvalidCredentials
	}

	sessionID, err := utils.GenerateOpaqueID("ses", 16)
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate session id: %w", err)
	}

	rawSessionToken, err := s.sessionService.GenerateToken()
	if err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := s.sessionService.ExpiresAt(input.RememberMe)
	if _, err := s.authRepo.CreateSession(ctx, generated.CreateSessionParams{
		ID:        sessionID,
		TokenHash: s.sessionService.HashToken(rawSessionToken),
		UserID:    row.ID,
		ExpiresAt: expiresAt,
		IpAddress: utils.ToNullString(utils.OptionalTrimmed(actor.IPAddress)),
		UserAgent: utils.ToNullString(utils.OptionalTrimmed(actor.UserAgent)),
	}); err != nil {
		return generated.User{}, "", time.Time{}, fmt.Errorf("create session: %w", err)
	}

	return authUserRowToUser(row), rawSessionToken, expiresAt, nil
}

func (s *AuthService) Logout(ctx context.Context, rawSessionToken string) error {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return nil
	}

	if err := s.authRepo.DeleteSessionByTokenHash(ctx, s.sessionService.HashToken(rawSessionToken)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (s *AuthService) ListSessions(ctx context.Context, userID string, rawSessionToken string) ([]types.AuthUserSessionData, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, types.ErrUnauthenticated
	}

	currentTokenHash, err := s.currentTokenHash(rawSessionToken)
	if err != nil {
		return nil, err
	}

	rows, err := s.authRepo.ListSessionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	return mapAuthSessions(rows, currentTokenHash), nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID string, rawSessionToken string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return types.ErrUnauthenticated
	}

	currentTokenHash, err := s.currentTokenHash(rawSessionToken)
	if err != nil {
		return err
	}

	if _, err := s.authRepo.DeleteOtherSessionsByUser(ctx, userID, currentTokenHash); err != nil {
		return fmt.Errorf("delete other sessions: %w", err)
	}

	return nil
}

func (s *AuthService) DeleteSession(ctx context.Context, userID string, sessionID string, rawSessionToken string) (bool, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return false, types.ErrUnauthenticated
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, types.ErrSessionNotFound
	}

	currentTokenHash, err := s.currentTokenHash(rawSessionToken)
	if err != nil {
		return false, err
	}

	currentSession, err := s.authRepo.GetSessionByTokenHash(ctx, currentTokenHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, types.ErrUnauthenticated
		}

		return false, fmt.Errorf("get current session: %w", err)
	}

	rowsDeleted, err := s.authRepo.DeleteSessionByIDAndUser(ctx, sessionID, userID)
	if err != nil {
		return false, fmt.Errorf("delete session: %w", err)
	}
	if rowsDeleted == 0 {
		return false, types.ErrSessionNotFound
	}

	return currentSession.ID == sessionID, nil
}

func (s *AuthService) Authenticate(ctx context.Context, rawSessionToken string) (generated.User, time.Time, error) {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return generated.User{}, time.Time{}, types.ErrUnauthenticated
	}

	row, err := s.authRepo.GetSessionUserByTokenHash(ctx, s.sessionService.HashToken(rawSessionToken))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.User{}, time.Time{}, types.ErrUnauthenticated
		}

		return generated.User{}, time.Time{}, fmt.Errorf("get session: %w", err)
	}

	if !row.ExpiresAt.After(s.now().UTC()) {
		_ = s.authRepo.DeleteSessionByTokenHash(ctx, s.sessionService.HashToken(rawSessionToken))
		return generated.User{}, time.Time{}, types.ErrUnauthenticated
	}

	return authSessionRowToUser(row), row.ExpiresAt, nil
}

func (s *AuthService) CompleteOnboarding(ctx context.Context, userID string) (generated.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return generated.User{}, types.ErrUnauthenticated
	}

	user, err := s.authRepo.CompleteOnboardingByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.User{}, types.ErrUserNotFound
		}

		return generated.User{}, fmt.Errorf("complete onboarding: %w", err)
	}

	return user, nil
}

func (s *AuthService) SendVerifyEmail(ctx context.Context, user generated.User) error {
	if strings.TrimSpace(user.ID) == "" {
		return types.ErrUnauthenticated
	}
	if user.EmailVerified {
		return nil
	}
	if s.securityEmailService == nil {
		return nil
	}

	rawToken, err := s.issueVerificationToken(ctx, user, types.VerificationTokenTypeEmailVerification)
	if err != nil {
		return err
	}

	verifyURL := s.buildTokenURL("/verify-email", rawToken)
	if err := s.securityEmailService.SendVerifyEmail(ctx, user.Email, user.Name, verifyURL); err != nil {
		return fmt.Errorf("send verify email: %w", err)
	}
	return nil
}

func (s *AuthService) ConfirmVerifyEmail(ctx context.Context, rawToken string) error {
	token, err := s.getValidVerificationToken(ctx, rawToken, types.VerificationTokenTypeEmailVerification)
	if err != nil {
		return err
	}
	if !token.UserID.Valid || strings.TrimSpace(token.UserID.String) == "" {
		return types.ErrInvalidVerificationToken
	}

	if err := s.authRepo.ConfirmEmailVerification(ctx, token.ID, token.UserID.String); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrInvalidVerificationToken
		}

		return fmt.Errorf("confirm email verification: %w", err)
	}

	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, email string) error {
	normalizedEmail, err := utils.NormalizeEmail(email)
	if err != nil {
		return nil
	}

	user, err := s.authRepo.GetUserWithPasswordByEmail(ctx, normalizedEmail)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("get user by email: %w", err)
	}
	if s.securityEmailService == nil {
		return nil
	}

	rawToken, err := s.issueVerificationToken(ctx, authUserRowToUser(user), types.VerificationTokenTypePasswordReset)
	if err != nil {
		return err
	}

	resetURL := s.buildTokenURL("/reset-password", rawToken)
	if err := s.securityEmailService.SendPasswordResetEmail(ctx, user.Email, user.Name, resetURL); err != nil {
		return fmt.Errorf("send password reset email: %w", err)
	}

	return nil
}

func (s *AuthService) ResetPassword(ctx context.Context, rawToken string, password string) error {
	token, err := s.getValidVerificationToken(ctx, rawToken, types.VerificationTokenTypePasswordReset)
	if err != nil {
		return err
	}
	if !token.UserID.Valid || strings.TrimSpace(token.UserID.String) == "" {
		return types.ErrInvalidVerificationToken
	}

	password = strings.TrimSpace(password)
	if len(password) < s.passwordService.MinimumLength() {
		return types.ErrInvalidPassword
	}

	passwordHash, err := s.passwordService.Hash(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	if err := s.authRepo.ResetPasswordWithToken(ctx, token.ID, token.UserID.String, passwordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrInvalidVerificationToken
		}

		return fmt.Errorf("reset password: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyPassword(ctx context.Context, userID string, password string) error {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return types.ErrUnauthenticated
	}

	password = strings.TrimSpace(password)
	if password == "" {
		return types.ErrInvalidCredentials
	}

	row, err := s.authRepo.GetUserWithPasswordByID(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrInvalidCredentials
		}

		return fmt.Errorf("get user by id: %w", err)
	}

	isValid, err := s.passwordService.Verify(password, row.PasswordHash)
	if err != nil {
		return fmt.Errorf("verify password: %w", err)
	}
	if !isValid {
		return types.ErrInvalidCredentials
	}

	return nil
}

func (s *AuthService) currentTokenHash(rawSessionToken string) (string, error) {
	rawSessionToken = strings.TrimSpace(rawSessionToken)
	if rawSessionToken == "" {
		return "", types.ErrUnauthenticated
	}

	return s.sessionService.HashToken(rawSessionToken), nil
}

func (s *AuthService) issueVerificationToken(ctx context.Context, user generated.User, tokenType string) (string, error) {
	tokenID, err := utils.GenerateOpaqueID("vtk", 16)
	if err != nil {
		return "", fmt.Errorf("generate verification token id: %w", err)
	}

	rawToken, err := s.verificationTokenService.GenerateToken()
	if err != nil {
		return "", fmt.Errorf("generate verification token: %w", err)
	}

	if _, err := s.authRepo.ReplaceVerificationToken(ctx, generated.CreateVerificationTokenParams{
		ID:         tokenID,
		UserID:     utils.ToNullString(&user.ID),
		Identifier: user.Email,
		TokenHash:  s.verificationTokenService.HashToken(rawToken),
		Type:       tokenType,
		ExpiresAt:  s.verificationTokenService.ExpiresAt(tokenType),
	}); err != nil {
		return "", fmt.Errorf("store verification token: %w", err)
	}

	return rawToken, nil
}

func (s *AuthService) getValidVerificationToken(ctx context.Context, rawToken string, tokenType string) (generated.VerificationToken, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return generated.VerificationToken{}, types.ErrInvalidVerificationToken
	}

	token, err := s.authRepo.GetVerificationTokenByTokenHashAndType(ctx, s.verificationTokenService.HashToken(rawToken), tokenType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.VerificationToken{}, types.ErrInvalidVerificationToken
		}

		return generated.VerificationToken{}, fmt.Errorf("get verification token: %w", err)
	}

	if !token.ExpiresAt.After(s.now().UTC()) {
		_, _ = s.authRepo.DeleteVerificationTokenByID(ctx, token.ID)
		return generated.VerificationToken{}, types.ErrInvalidVerificationToken
	}

	return token, nil
}

func (s *AuthService) buildTokenURL(path string, rawToken string) string {
	baseURL := s.appBaseURL
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}

	return baseURL + path + "?token=" + url.QueryEscape(rawToken)
}

func (s *AuthService) validateRegisterInput(input types.RegisterInput) (string, string, string, error) {
	name := utils.NormalizeRequiredString(input.Name)
	if name == "" || utils.BuildSlug(name) == "" {
		return "", "", "", types.ErrInvalidName
	}

	email, err := utils.NormalizeEmail(input.Email)
	if err != nil {
		return "", "", "", err
	}

	password := strings.TrimSpace(input.Password)
	if len(password) < s.passwordService.MinimumLength() {
		return "", "", "", types.ErrInvalidPassword
	}

	return name, email, password, nil
}

func (s *AuthService) validateLoginInput(input types.LoginInput) (string, string, error) {
	email, err := utils.NormalizeEmail(input.Email)
	if err != nil {
		return "", "", err
	}

	password := strings.TrimSpace(input.Password)
	if password == "" {
		return "", "", types.ErrInvalidPassword
	}

	return email, password, nil
}

func authUserRowToUser(row generated.GetUserWithPasswordByEmailRow) generated.User {
	return generated.User{
		ID:                  row.ID,
		Name:                row.Name,
		Email:               row.Email,
		EmailVerified:       row.EmailVerified,
		Image:               row.Image,
		OnboardingCompleted: row.OnboardingCompleted,
		TwoFactorEnabled:    row.TwoFactorEnabled,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func authSessionRowToUser(row generated.GetSessionUserByTokenHashRow) generated.User {
	return generated.User{
		ID:                  row.ID,
		Name:                row.Name,
		Email:               row.Email,
		EmailVerified:       row.EmailVerified,
		Image:               row.Image,
		OnboardingCompleted: row.OnboardingCompleted,
		TwoFactorEnabled:    row.TwoFactorEnabled,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}
