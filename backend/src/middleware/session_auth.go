package middleware

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type authenticatedUserContextKey struct{}

type sessionAuthenticator interface {
	Authenticate(ctx context.Context, rawSessionToken string) (generated.User, time.Time, error)
}

func SessionAuth(authService sessionAuthenticator, sessionService *services.SessionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(services.SessionCookieName)
			if err != nil {
				utils.WriteError(w, http.StatusUnauthorized, types.ErrUnauthenticated.Error())
				return
			}

			user, _, err := authService.Authenticate(r.Context(), cookie.Value)
			if err != nil {
				if errors.Is(err, types.ErrUnauthenticated) {
					http.SetCookie(w, sessionService.ClearCookie())
				}

				if utils.WriteDomainError(w, err) {
					return
				}

				utils.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}

			ctx := context.WithValue(r.Context(), authenticatedUserContextKey{}, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticatedUser(ctx context.Context) (generated.User, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey{}).(generated.User)
	return user, ok
}

func WithAuthenticatedUser(ctx context.Context, user generated.User) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey{}, user)
}
