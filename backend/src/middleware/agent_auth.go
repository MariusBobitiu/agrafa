package middleware

import (
	"context"
	"errors"
	"net/http"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

const AgentTokenHeader = "X-Agent-Token"

type authenticatedNodeContextKey struct{}

func AgentAuth(agentAuthService *services.AgentAuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := appdb.WithInternalRLSBypass(r.Context())

			node, err := agentAuthService.AuthenticateToken(ctx, r.Header.Get(AgentTokenHeader))
			if err != nil {
				switch {
				case errors.Is(err, types.ErrMissingAgentToken), errors.Is(err, types.ErrInvalidAgentToken):
					utils.WriteError(w, http.StatusUnauthorized, err.Error())
				default:
					utils.WriteError(w, http.StatusInternalServerError, err.Error())
				}

				return
			}

			ctx = context.WithValue(ctx, authenticatedNodeContextKey{}, node)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func AuthenticatedNode(ctx context.Context) (generated.Node, bool) {
	node, ok := ctx.Value(authenticatedNodeContextKey{}).(generated.Node)
	return node, ok
}
