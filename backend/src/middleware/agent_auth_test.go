package middleware

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type fakeAgentAuthNodeRepo struct {
	nodesByHash map[string]generated.Node
}

func (r *fakeAgentAuthNodeRepo) GetByAgentTokenHash(_ context.Context, hash string) (generated.Node, error) {
	node, ok := r.nodesByHash[hash]
	if !ok {
		return generated.Node{}, sql.ErrNoRows
	}

	return node, nil
}

func TestAgentAuthMiddleware(t *testing.T) {
	t.Parallel()

	validToken := "test-agent-token"
	authService := services.NewAgentAuthService(&fakeAgentAuthNodeRepo{
		nodesByHash: map[string]generated.Node{
			utils.HashAgentToken(validToken): {
				ID:        42,
				Name:      "node-42",
				ProjectID: 7,
			},
		},
	})

	handler := AgentAuth(authService)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		node, ok := AuthenticatedNode(r.Context())
		if !ok {
			t.Fatal("expected authenticated node in context")
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"node_id": node.ID,
		})
	}))

	testCases := []struct {
		name           string
		token          string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "missing token is rejected",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"X-Agent-Token header is required"}`,
		},
		{
			name:           "invalid token is rejected",
			token:          "wrong-token",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   `{"error":"invalid agent token"}`,
		},
		{
			name:           "valid token authenticates node",
			token:          validToken,
			expectedStatus: http.StatusOK,
			expectedBody:   `{"node_id":42}`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			request := httptest.NewRequest(http.MethodPost, "/v1/agent/heartbeat", nil)
			if testCase.token != "" {
				request.Header.Set(AgentTokenHeader, testCase.token)
			}

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)

			if recorder.Code != testCase.expectedStatus {
				t.Fatalf("expected status %d, got %d", testCase.expectedStatus, recorder.Code)
			}

			if body := recorder.Body.String(); body != testCase.expectedBody+"\n" {
				t.Fatalf("expected body %q, got %q", testCase.expectedBody, body)
			}
		})
	}
}
