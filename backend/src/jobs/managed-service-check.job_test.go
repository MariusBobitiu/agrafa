package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeManagedServiceLister struct {
	services []generated.Service
	err      error
}

func (l *fakeManagedServiceLister) ListServices(_ context.Context, _ *int64) ([]generated.Service, error) {
	if l.err != nil {
		return nil, l.err
	}

	return append([]generated.Service(nil), l.services...), nil
}

type fakeManagedNodeLister struct {
	nodes []generated.Node
	err   error
}

func (l *fakeManagedNodeLister) ListNodes(_ context.Context, _ *int64) ([]generated.Node, error) {
	if l.err != nil {
		return nil, l.err
	}

	return append([]generated.Node(nil), l.nodes...), nil
}

type fakeManagedHeartbeatIngester struct {
	inputs []types.HeartbeatInput
}

func (i *fakeManagedHeartbeatIngester) Ingest(_ context.Context, input types.HeartbeatInput) (generated.Node, error) {
	i.inputs = append(i.inputs, input)
	return generated.Node{ID: input.AuthenticatedNodeID}, nil
}

type fakeManagedHealthIngester struct {
	inputs []types.HealthCheckInput
}

func (i *fakeManagedHealthIngester) Ingest(_ context.Context, input types.HealthCheckInput) (generated.Service, error) {
	i.inputs = append(i.inputs, input)
	return generated.Service{ID: input.ServiceID}, nil
}

func TestManagedServiceCheckJobRunOnceChecksManagedServicesOnly(t *testing.T) {
	serviceLister := &fakeManagedServiceLister{
		services: []generated.Service{
			{ID: 11, NodeID: 101, CheckType: "http", CheckTarget: "http://example.com"},
			{ID: 12, NodeID: 102, CheckType: "http", CheckTarget: "http://example.org"},
			{ID: 13, NodeID: 101, CheckType: "tcp", CheckTarget: "db.internal:5432"},
		},
	}
	nodeLister := &fakeManagedNodeLister{
		nodes: []generated.Node{
			{ID: 101, NodeType: types.NodeTypeManaged},
			{ID: 102, NodeType: types.NodeTypeAgent},
		},
	}
	heartbeatIngester := &fakeManagedHeartbeatIngester{}
	healthIngester := &fakeManagedHealthIngester{}

	job := &ManagedServiceCheckJob{
		services:   serviceLister,
		nodes:      nodeLister,
		heartbeats: heartbeatIngester,
		health:     healthIngester,
		interval:   time.Minute,
		timeout:    10 * time.Second,
	}
	job.checkRunner = func(_ context.Context, checkType string, _ string) managedCheckResult {
		return managedCheckResult{
			observedAt: time.Date(2026, time.April, 6, 12, 30, 0, 0, time.UTC),
			isSuccess:  checkType == "http",
			message:    "checked",
			payload:    managedCheckPayload(checkType),
		}
	}

	job.runOnce(context.Background())

	if len(heartbeatIngester.inputs) != 1 {
		t.Fatalf("expected 1 managed heartbeat, got %d", len(heartbeatIngester.inputs))
	}

	if heartbeatIngester.inputs[0].AuthenticatedNodeID != 101 {
		t.Fatalf("expected heartbeat for managed node 101, got %d", heartbeatIngester.inputs[0].AuthenticatedNodeID)
	}

	if len(healthIngester.inputs) != 2 {
		t.Fatalf("expected 2 managed health ingestions, got %d", len(healthIngester.inputs))
	}

	if healthIngester.inputs[0].ServiceID != 11 || healthIngester.inputs[1].ServiceID != 13 {
		t.Fatalf("expected managed services 11 and 13, got %d and %d", healthIngester.inputs[0].ServiceID, healthIngester.inputs[1].ServiceID)
	}

	if healthIngester.inputs[0].AuthenticatedNodeID != 101 || healthIngester.inputs[1].AuthenticatedNodeID != 101 {
		t.Fatalf("expected health ingestions to authenticate as managed node 101: %#v", healthIngester.inputs)
	}
}
