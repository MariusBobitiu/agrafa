package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	agentmiddleware "github.com/MariusBobitiu/agrafa-backend/src/middleware"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type fakeAgentControllerHeartbeatService struct{}

func (s *fakeAgentControllerHeartbeatService) Ingest(context.Context, types.HeartbeatInput) (generated.Node, error) {
	return generated.Node{}, nil
}

type fakeAgentControllerNodeStateService struct {
	node         generated.Node
	transitioned bool
	calls        []fakeAgentControllerShutdownCall
	err          error
}

type fakeAgentControllerShutdownCall struct {
	nodeID     int64
	occurredAt time.Time
	reason     string
	payload    json.RawMessage
}

func (s *fakeAgentControllerNodeStateService) MarkOfflineFromShutdown(_ context.Context, nodeID int64, occurredAt time.Time, reason string, payload json.RawMessage) (generated.Node, bool, error) {
	s.calls = append(s.calls, fakeAgentControllerShutdownCall{
		nodeID:     nodeID,
		occurredAt: occurredAt,
		reason:     reason,
		payload:    payload,
	})
	return s.node, s.transitioned, s.err
}

type fakeAgentControllerHealthService struct{}

func (s *fakeAgentControllerHealthService) Ingest(context.Context, types.HealthCheckInput) (generated.Service, error) {
	return generated.Service{}, nil
}

type fakeAgentControllerMetricService struct{}

func (s *fakeAgentControllerMetricService) Ingest(context.Context, types.MetricIngestionInput) error {
	return nil
}

type fakeAgentControllerConfigService struct {
	config types.AgentConfigData
	err    error
}

func (s *fakeAgentControllerConfigService) GetForNode(context.Context, generated.Node) (types.AgentConfigData, error) {
	return s.config, s.err
}

type fakeAgentConfigAuthNodeRepo struct {
	nodesByHash map[string]generated.Node
}

func (r *fakeAgentConfigAuthNodeRepo) GetByAgentTokenHash(_ context.Context, hash string) (generated.Node, error) {
	node, ok := r.nodesByHash[hash]
	if !ok {
		return generated.Node{}, sql.ErrNoRows
	}

	return node, nil
}

func TestAgentControllerGetConfigReturnsAssignedChecks(t *testing.T) {
	t.Parallel()

	validToken := "agent-token"
	authService := services.NewAgentAuthService(&fakeAgentConfigAuthNodeRepo{
		nodesByHash: map[string]generated.Node{
			utils.HashAgentToken(validToken): {
				ID:         12,
				Name:       "web-01",
				Identifier: "web-01",
				ProjectID:  7,
			},
		},
	})

	controller := NewAgentController(
		&fakeAgentControllerHeartbeatService{},
		&fakeAgentControllerNodeStateService{},
		&fakeAgentControllerHealthService{},
		&fakeAgentControllerMetricService{},
		&fakeAgentControllerConfigService{
			config: types.AgentConfigData{
				Node: types.AgentConfigNodeData{
					ID:         12,
					Name:       "web-01",
					Identifier: "web-01",
				},
				HealthChecks: []types.AgentConfigCheckData{
					{
						ServiceID:       101,
						Name:            "internal-api",
						CheckType:       "http",
						CheckTarget:     "http://internal-api.local/health",
						IntervalSeconds: 30,
						TimeoutSeconds:  10,
					},
				},
			},
		},
	)

	handler := agentmiddleware.AgentAuth(authService)(http.HandlerFunc(controller.GetConfig))
	request := httptest.NewRequest(http.MethodGet, "/v1/agent/config", nil)
	request.Header.Set(agentmiddleware.AgentTokenHeader, validToken)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	body := recorder.Body.String()
	for _, expected := range []string{
		`"node":{"id":12,"name":"web-01","identifier":"web-01"}`,
		`"health_checks":[{"service_id":101,"name":"internal-api","check_type":"http","check_target":"http://internal-api.local/health","interval_seconds":30,"timeout_seconds":10}]`,
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected body to contain %q, got %s", expected, body)
		}
	}
}

func TestAgentControllerGetConfigRejectsInvalidToken(t *testing.T) {
	t.Parallel()

	authService := services.NewAgentAuthService(&fakeAgentConfigAuthNodeRepo{nodesByHash: map[string]generated.Node{}})
	controller := NewAgentController(
		&fakeAgentControllerHeartbeatService{},
		&fakeAgentControllerNodeStateService{},
		&fakeAgentControllerHealthService{},
		&fakeAgentControllerMetricService{},
		&fakeAgentControllerConfigService{},
	)

	handler := agentmiddleware.AgentAuth(authService)(http.HandlerFunc(controller.GetConfig))
	request := httptest.NewRequest(http.MethodGet, "/v1/agent/config", nil)
	request.Header.Set(agentmiddleware.AgentTokenHeader, "wrong-token")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, `{"error":"invalid agent token"}`) {
		t.Fatalf("unexpected body: %s", body)
	}
}

func TestAgentControllerIngestShutdownMarksNodeOfflineWithReason(t *testing.T) {
	t.Parallel()

	validToken := "agent-token"
	authenticatedNode := generated.Node{
		ID:         12,
		Name:       "web-01",
		Identifier: "web-01",
		ProjectID:  7,
	}

	authService := services.NewAgentAuthService(&fakeAgentConfigAuthNodeRepo{
		nodesByHash: map[string]generated.Node{
			utils.HashAgentToken(validToken): authenticatedNode,
		},
	})

	nodeState := &fakeAgentControllerNodeStateService{
		node: generated.Node{
			ID:           12,
			ProjectID:    7,
			Name:         "web-01",
			Identifier:   "web-01",
			CurrentState: types.NodeStateOffline,
		},
		transitioned: true,
	}

	controller := NewAgentController(
		&fakeAgentControllerHeartbeatService{},
		nodeState,
		&fakeAgentControllerHealthService{},
		&fakeAgentControllerMetricService{},
		&fakeAgentControllerConfigService{},
	)

	handler := agentmiddleware.AgentAuth(authService)(http.HandlerFunc(controller.IngestShutdown))
	request := httptest.NewRequest(http.MethodPost, "/v1/agent/shutdown", strings.NewReader(`{"node_id":12,"reason":"user_closed","payload":{"signal":"SIGINT"}}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(agentmiddleware.AgentTokenHeader, validToken)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}

	if len(nodeState.calls) != 1 {
		t.Fatalf("expected 1 shutdown call, got %d", len(nodeState.calls))
	}

	if nodeState.calls[0].nodeID != authenticatedNode.ID {
		t.Fatalf("nodeID = %d, want %d", nodeState.calls[0].nodeID, authenticatedNode.ID)
	}

	if nodeState.calls[0].reason != "user_closed" {
		t.Fatalf("reason = %q, want user_closed", nodeState.calls[0].reason)
	}

	if !strings.Contains(string(nodeState.calls[0].payload), `"signal":"SIGINT"`) {
		t.Fatalf("unexpected payload: %s", nodeState.calls[0].payload)
	}
}
