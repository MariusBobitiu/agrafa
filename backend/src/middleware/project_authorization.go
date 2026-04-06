package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
	"github.com/go-chi/chi/v5"
)

type projectPermissionAuthorizer interface {
	RequireProjectPermission(ctx context.Context, userID string, projectID int64, permission string) error
}

type ProjectIDResolver func(ctx context.Context, r *http.Request) (int64, error)

type badRequestError struct {
	message string
}

func (e badRequestError) Error() string {
	return e.message
}

func RequireProjectPermission(
	authorizer projectPermissionAuthorizer,
	permission string,
	resolveProjectID ProjectIDResolver,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := AuthenticatedUser(r.Context())
			if !ok {
				utils.WriteError(w, http.StatusInternalServerError, "authenticated user missing from context")
				return
			}

			projectID, err := resolveProjectID(r.Context(), r)
			if err != nil {
				if utils.WriteDomainError(w, err) {
					return
				}

				var requestErr badRequestError
				if errors.As(err, &requestErr) {
					utils.WriteError(w, http.StatusBadRequest, err.Error())
					return
				}

				utils.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}

			if err := authorizer.RequireProjectPermission(r.Context(), user.ID, projectID, permission); err != nil {
				if utils.WriteDomainError(w, err) {
					return
				}

				utils.WriteError(w, http.StatusInternalServerError, err.Error())
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func ProjectIDFromRequiredQueryParam(param string) ProjectIDResolver {
	return func(_ context.Context, r *http.Request) (int64, error) {
		rawValue := r.URL.Query().Get(param)
		if rawValue == "" {
			return 0, badRequestError{message: fmt.Sprintf("%s is required", param)}
		}

		projectID, err := strconv.ParseInt(rawValue, 10, 64)
		if err != nil || projectID <= 0 {
			return 0, badRequestError{message: fmt.Sprintf("%s must be a positive integer", param)}
		}

		return projectID, nil
	}
}

func ProjectIDFromBodyField(field string) ProjectIDResolver {
	return func(_ context.Context, r *http.Request) (int64, error) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return 0, fmt.Errorf("read request body: %w", err)
		}

		r.Body = io.NopCloser(bytes.NewReader(body))

		var payload map[string]json.RawMessage
		decoder := json.NewDecoder(bytes.NewReader(body))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&payload); err != nil {
			return 0, badRequestError{message: "invalid request payload"}
		}

		rawValue, ok := payload[field]
		if !ok {
			return 0, types.ErrInvalidProjectID
		}

		var projectID int64
		if err := json.Unmarshal(rawValue, &projectID); err != nil || projectID <= 0 {
			return 0, types.ErrInvalidProjectID
		}

		return projectID, nil
	}
}

func ProjectIDFromURLParamResource(
	param string,
	lookup func(ctx context.Context, id int64) (int64, error),
) ProjectIDResolver {
	return func(ctx context.Context, r *http.Request) (int64, error) {
		rawID := chi.URLParam(r, param)
		resourceID, err := strconv.ParseInt(rawID, 10, 64)
		if err != nil || resourceID <= 0 {
			return 0, badRequestError{message: fmt.Sprintf("%s must be a positive integer", param)}
		}

		return lookup(ctx, resourceID)
	}
}

func ProjectIDFromURLParamStringResource(
	param string,
	lookup func(ctx context.Context, id string) (int64, error),
) ProjectIDResolver {
	return func(ctx context.Context, r *http.Request) (int64, error) {
		resourceID := strings.TrimSpace(chi.URLParam(r, param))
		if resourceID == "" {
			return 0, badRequestError{message: fmt.Sprintf("%s is required", param)}
		}

		return lookup(ctx, resourceID)
	}
}
