package controllers

import (
	"context"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	authmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type authService interface {
	Register(ctx context.Context, input types.RegisterInput, actor types.SessionActor) (generated.User, string, time.Time, error)
	Login(ctx context.Context, input types.LoginInput, actor types.SessionActor) (generated.User, string, time.Time, error)
	Logout(ctx context.Context, rawSessionToken string) error
	CompleteOnboarding(ctx context.Context, userID string) (generated.User, error)
	SendVerifyEmail(ctx context.Context, user generated.User) error
	ConfirmVerifyEmail(ctx context.Context, rawToken string) error
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, rawToken string, password string) error
	VerifyPassword(ctx context.Context, userID string, password string) error
	ListSessions(ctx context.Context, userID string, rawSessionToken string) ([]types.AuthUserSessionData, error)
	LogoutAll(ctx context.Context, userID string, rawSessionToken string) error
	DeleteSession(ctx context.Context, userID string, sessionID string, rawSessionToken string) (bool, error)
	Authenticate(ctx context.Context, rawSessionToken string) (generated.User, time.Time, error)
}

type sessionCookieService interface {
	BuildCookie(token string, expiresAt time.Time) *http.Cookie
	ClearCookie() *http.Cookie
}

type AuthController struct {
	authService    authService
	sessionService sessionCookieService
}

func NewAuthController(authService authService, sessionService sessionCookieService) *AuthController {
	return &AuthController{
		authService:    authService,
		sessionService: sessionService,
	}
}

// Register creates a new user and session.
//
// @Summary      Register
// @Description  Creates a user, password credential, and session cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthRegisterRequest  true  "Registration payload"
// @Success      201      {object}  types.AuthSessionResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      409      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/register [post]
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var request types.AuthRegisterRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid register payload")
		return
	}

	user, sessionToken, expiresAt, err := c.authService.Register(r.Context(), types.RegisterInput{
		Name:     request.Name,
		Email:    request.Email,
		Password: request.Password,
	}, sessionActorFromRequest(r))
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.SetCookie(w, c.sessionService.BuildCookie(sessionToken, expiresAt))
	utils.WriteJSON(w, http.StatusCreated, types.AuthSessionResponse{
		User:      mapUserDocument(services.MapUserResponse(user)),
		ExpiresAt: expiresAt,
	})
}

// Login creates a new session for an existing user.
//
// @Summary      Login
// @Description  Verifies email/password and sets a session cookie.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthLoginRequest  true  "Login payload"
// @Success      200      {object}  types.AuthSessionResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var request types.AuthLoginRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid login payload")
		return
	}

	user, sessionToken, expiresAt, err := c.authService.Login(r.Context(), types.LoginInput{
		Email:      request.Email,
		Password:   request.Password,
		RememberMe: request.RememberMe,
	}, sessionActorFromRequest(r))
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.SetCookie(w, c.sessionService.BuildCookie(sessionToken, expiresAt))
	utils.WriteJSON(w, http.StatusOK, types.AuthSessionResponse{
		User:      mapUserDocument(services.MapUserResponse(user)),
		ExpiresAt: expiresAt,
	})
}

// Logout removes the current session.
//
// @Summary      Logout
// @Description  Deletes the current session and clears the session cookie.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthLogoutResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/logout [post]
func (c *AuthController) Logout(w http.ResponseWriter, r *http.Request) {
	if err := c.authService.Logout(r.Context(), sessionTokenFromRequest(r)); err != nil {
		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	http.SetCookie(w, c.sessionService.ClearCookie())
	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// CompleteOnboarding marks onboarding as complete for the authenticated user.
//
// @Summary      Complete onboarding
// @Description  Marks the authenticated user onboarding flow as complete and returns the updated user payload.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthMeResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/onboarding/complete [post]
func (c *AuthController) CompleteOnboarding(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	updatedUser, err := c.authService.CompleteOnboarding(r.Context(), user.ID)
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthMeResponse{
		User: mapUserDocument(services.MapUserResponse(updatedUser)),
	})
}

// SendVerifyEmail sends an email verification message to the authenticated user.
//
// @Summary      Send email verification
// @Description  Sends an email verification link for the authenticated user unless the email is already verified.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthLogoutResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/verify-email/send [post]
func (c *AuthController) SendVerifyEmail(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	if err := c.authService.SendVerifyEmail(r.Context(), user); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// ConfirmVerifyEmail validates an email verification token and marks the user email as verified.
//
// @Summary      Confirm email verification
// @Description  Validates an email verification token, marks the user as verified, and invalidates the token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthVerifyEmailConfirmRequest  true  "Email verification token"
// @Success      200      {object}  types.AuthLogoutResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/verify-email/confirm [post]
func (c *AuthController) ConfirmVerifyEmail(w http.ResponseWriter, r *http.Request) {
	var request types.AuthVerifyEmailConfirmRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid verify email payload")
		return
	}

	if err := c.authService.ConfirmVerifyEmail(r.Context(), request.Token); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// ForgotPassword sends a password reset email when the account exists and always returns a generic success response.
//
// @Summary      Forgot password
// @Description  Accepts an email address and, when an account exists, sends a password reset email without revealing account existence.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthForgotPasswordRequest  true  "Password reset request"
// @Success      200      {object}  types.AuthLogoutResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/forgot-password [post]
func (c *AuthController) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var request types.AuthForgotPasswordRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid forgot password payload")
		return
	}

	if err := c.authService.ForgotPassword(r.Context(), request.Email); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// ResetPassword validates a password reset token, updates the password, and revokes all sessions.
//
// @Summary      Reset password
// @Description  Validates a password reset token, updates the password hash, invalidates the token, and revokes all sessions for the user.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthResetPasswordRequest  true  "Password reset payload"
// @Success      200      {object}  types.AuthLogoutResponse
// @Failure      400      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/reset-password [post]
func (c *AuthController) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var request types.AuthResetPasswordRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid reset password payload")
		return
	}

	if err := c.authService.ResetPassword(r.Context(), request.Token, request.Password); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// VerifyPassword verifies the current password for the authenticated session without mutating session state.
//
// @Summary      Verify password
// @Description  Verifies the current password for the authenticated user without creating a new session or changing state.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request  body      types.AuthVerifyPasswordRequest  true  "Password verification payload"
// @Success      200      {object}  types.AuthLogoutResponse
// @Failure      401      {object}  types.ErrorResponse
// @Failure      500      {object}  types.ErrorResponse
// @Router       /auth/verify-password [post]
func (c *AuthController) VerifyPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	var request types.AuthVerifyPasswordRequest
	if err := utils.DecodeJSON(r, &request); err != nil {
		utils.WriteError(w, http.StatusBadRequest, "invalid verify password payload")
		return
	}

	if err := c.authService.VerifyPassword(r.Context(), user.ID, request.Password); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// Me returns the authenticated user for the current session.
//
// @Summary      Me
// @Description  Returns the currently authenticated user when the session cookie is valid.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthMeResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/me [get]
func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	user, _, err := c.authService.Authenticate(r.Context(), sessionTokenFromRequest(r))
	if err != nil {
		if err == types.ErrUnauthenticated {
			http.SetCookie(w, c.sessionService.ClearCookie())
		}

		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthMeResponse{
		User: mapUserDocument(services.MapUserResponse(user)),
	})
}

// ListSessions lists the authenticated user's sessions.
//
// @Summary      List auth sessions
// @Description  Returns all sessions for the authenticated user and marks the current session.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthSessionsResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/sessions [get]
func (c *AuthController) ListSessions(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	sessions, err := c.authService.ListSessions(r.Context(), user.ID, sessionTokenFromRequest(r))
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthSessionsResponse{
		Sessions: mapAuthSessionDocuments(sessions),
	})
}

// LogoutAll removes all other sessions for the authenticated user.
//
// @Summary      Logout all other sessions
// @Description  Deletes all other sessions for the authenticated user and keeps the current session active.
// @Tags         auth
// @Produce      json
// @Success      200  {object}  types.AuthLogoutResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/logout-all [post]
func (c *AuthController) LogoutAll(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	if err := c.authService.LogoutAll(r.Context(), user.ID, sessionTokenFromRequest(r)); err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	utils.WriteJSON(w, http.StatusOK, types.AuthLogoutResponse{Status: "ok"})
}

// DeleteSession removes a specific session for the authenticated user.
//
// @Summary      Delete auth session
// @Description  Deletes a specific session belonging to the authenticated user. If the current session is deleted, the session cookie is cleared.
// @Tags         auth
// @Produce      json
// @Param        id  path      string  true  "Session ID"
// @Success      204
// @Failure      400  {object}  types.ErrorResponse
// @Failure      401  {object}  types.ErrorResponse
// @Failure      404  {object}  types.ErrorResponse
// @Failure      500  {object}  types.ErrorResponse
// @Router       /auth/sessions/{id} [delete]
func (c *AuthController) DeleteSession(w http.ResponseWriter, r *http.Request) {
	user, ok := authmiddleware.AuthenticatedUser(r.Context())
	if !ok {
		utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
		return
	}

	sessionID := strings.TrimSpace(chi.URLParam(r, "id"))
	if sessionID == "" {
		utils.WriteError(w, http.StatusBadRequest, "id is required")
		return
	}

	isCurrent, err := c.authService.DeleteSession(r.Context(), user.ID, sessionID, sessionTokenFromRequest(r))
	if err != nil {
		if utils.WriteDomainError(w, err) {
			return
		}

		utils.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if isCurrent {
		http.SetCookie(w, c.sessionService.ClearCookie())
	}

	w.WriteHeader(http.StatusNoContent)
}

func sessionActorFromRequest(r *http.Request) types.SessionActor {
	return types.SessionActor{
		IPAddress: requestIPAddress(r),
		UserAgent: r.UserAgent(),
	}
}

func sessionTokenFromRequest(r *http.Request) string {
	cookie, err := r.Cookie(services.SessionCookieName)
	if err != nil {
		return ""
	}

	return cookie.Value
}

func requestIPAddress(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}

	return r.RemoteAddr
}

func mapUserDocument(data types.UserData) types.UserDocument {
	return types.UserDocument{
		ID:                  data.ID,
		Name:                data.Name,
		Email:               data.Email,
		EmailVerified:       data.EmailVerified,
		Image:               data.Image,
		OnboardingCompleted: data.OnboardingCompleted,
		TwoFactorEnabled:    data.TwoFactorEnabled,
		CreatedAt:           data.CreatedAt,
		UpdatedAt:           data.UpdatedAt,
	}
}

func mapAuthSessionDocuments(items []types.AuthUserSessionData) []types.AuthUserSessionDocument {
	documents := make([]types.AuthUserSessionDocument, 0, len(items))
	for _, item := range items {
		documents = append(documents, types.AuthUserSessionDocument{
			ID:        item.ID,
			ExpiresAt: item.ExpiresAt,
			IPAddress: item.IPAddress,
			UserAgent: item.UserAgent,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
			IsCurrent: item.IsCurrent,
		})
	}

	return documents
}
