package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeStaleNodeRepository struct {
	nodes      []generated.Node
	err        error
	lastCutoff time.Time
}

func (r *fakeStaleNodeRepository) ListStaleOnline(_ context.Context, cutoff time.Time) ([]generated.Node, error) {
	r.lastCutoff = cutoff
	if r.err != nil {
		return nil, r.err
	}

	return append([]generated.Node(nil), r.nodes...), nil
}

type fakeStaleNodeStateService struct {
	calls []staleNodeStateCall
}

type staleNodeStateCall struct {
	node   generated.Node
	cutoff time.Time
}

func (s *fakeStaleNodeStateService) MarkOfflineIfStale(_ context.Context, node generated.Node, cutoff time.Time) (generated.Node, bool, error) {
	s.calls = append(s.calls, staleNodeStateCall{
		node:   node,
		cutoff: cutoff,
	})

	if node.CurrentState == types.NodeStateOnline {
		node.CurrentState = types.NodeStateOffline
		return node, true, nil
	}

	return node, false, nil
}

func TestNodeExpiryJobRunOnceProcessesReturnedNodesWithSharedCutoff(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	heartbeatTTL := 30 * time.Second

	staleRepo := &fakeStaleNodeRepository{
		nodes: []generated.Node{
			{ID: 1, CurrentState: types.NodeStateOnline},
			{ID: 2, CurrentState: types.NodeStateOffline},
			{ID: 3, CurrentState: types.NodeStateOnline},
		},
	}
	nodeState := &fakeStaleNodeStateService{}

	job := &NodeExpiryJob{
		nodeRepo:     staleRepo,
		nodeState:    nodeState,
		heartbeatTTL: heartbeatTTL,
		interval:     time.Minute,
	}

	withNow(t, now, func() {
		job.runOnce(context.Background())
	})

	expectedCutoff := now.Add(-heartbeatTTL)
	if !staleRepo.lastCutoff.Equal(expectedCutoff) {
		t.Fatalf("expected cutoff %s, got %s", expectedCutoff, staleRepo.lastCutoff)
	}

	if len(nodeState.calls) != len(staleRepo.nodes) {
		t.Fatalf("expected %d processed nodes, got %d", len(staleRepo.nodes), len(nodeState.calls))
	}

	for index, call := range nodeState.calls {
		if !call.cutoff.Equal(expectedCutoff) {
			t.Fatalf("call %d expected cutoff %s, got %s", index, expectedCutoff, call.cutoff)
		}

		if call.node.ID != staleRepo.nodes[index].ID {
			t.Fatalf("call %d expected node id %d, got %d", index, staleRepo.nodes[index].ID, call.node.ID)
		}
	}
}

func withNow(t *testing.T, now time.Time, fn func()) {
	t.Helper()

	previousNow := timeNow
	timeNow = func() time.Time { return now }
	defer func() {
		timeNow = previousNow
	}()

	fn()
}
