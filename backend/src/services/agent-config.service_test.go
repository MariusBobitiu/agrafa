package services

import (
	"context"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type fakeAgentConfigRepository struct {
	nodeID int64
	rows   []generated.ListAgentConfigChecksByNodeIDRow
	err    error
}

func (r *fakeAgentConfigRepository) ListAgentConfigChecksByNodeID(_ context.Context, nodeID int64) ([]generated.ListAgentConfigChecksByNodeIDRow, error) {
	r.nodeID = nodeID
	return r.rows, r.err
}

func TestAgentConfigServiceGetForNodeMapsCleanResponse(t *testing.T) {
	t.Parallel()

	repo := &fakeAgentConfigRepository{
		rows: []generated.ListAgentConfigChecksByNodeIDRow{
			{
				ServiceID:   101,
				Name:        "internal-api",
				CheckType:   "http",
				CheckTarget: "http://internal-api.local/health",
			},
		},
	}

	service := NewAgentConfigService(repo)
	config, err := service.GetForNode(context.Background(), generated.Node{
		ID:         12,
		Name:       "web-01",
		Identifier: "web-01",
	})
	if err != nil {
		t.Fatalf("GetForNode() error = %v", err)
	}

	if repo.nodeID != 12 {
		t.Fatalf("repo.nodeID = %d, want 12", repo.nodeID)
	}
	if config.Node.ID != 12 || config.Node.Name != "web-01" || config.Node.Identifier != "web-01" {
		t.Fatalf("unexpected node summary: %#v", config.Node)
	}
	if len(config.HealthChecks) != 1 {
		t.Fatalf("len(config.HealthChecks) = %d, want 1", len(config.HealthChecks))
	}

	check := config.HealthChecks[0]
	if check.ServiceID != 101 || check.Name != "internal-api" {
		t.Fatalf("unexpected check identity: %#v", check)
	}
	if check.CheckType != "http" || check.CheckTarget != "http://internal-api.local/health" {
		t.Fatalf("unexpected check definition: %#v", check)
	}
	if check.IntervalSeconds != defaultAgentConfigIntervalSeconds {
		t.Fatalf("check.IntervalSeconds = %d, want %d", check.IntervalSeconds, defaultAgentConfigIntervalSeconds)
	}
	if check.TimeoutSeconds != defaultAgentConfigTimeoutSeconds {
		t.Fatalf("check.TimeoutSeconds = %d, want %d", check.TimeoutSeconds, defaultAgentConfigTimeoutSeconds)
	}
}
