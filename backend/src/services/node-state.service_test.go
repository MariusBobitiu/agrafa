package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNodeStateRepo struct {
	nodes               map[int64]generated.Node
	touchHeartbeatCalls []touchHeartbeatCall
	updateStateCalls    []updateStateCall
}

type touchHeartbeatCall struct {
	nodeID     int64
	observedAt time.Time
}

type updateStateCall struct {
	nodeID int64
	state  string
}

func (r *fakeNodeStateRepo) GetByID(_ context.Context, id int64) (generated.Node, error) {
	return r.nodes[id], nil
}

func (r *fakeNodeStateRepo) List(_ context.Context) ([]generated.Node, error) {
	return nil, nil
}

func (r *fakeNodeStateRepo) ListByProject(_ context.Context, _ int64) ([]generated.Node, error) {
	return nil, nil
}

func (r *fakeNodeStateRepo) TouchHeartbeat(_ context.Context, nodeID int64, observedAt time.Time) (generated.Node, error) {
	r.touchHeartbeatCalls = append(r.touchHeartbeatCalls, touchHeartbeatCall{
		nodeID:     nodeID,
		observedAt: observedAt,
	})

	node := r.nodes[nodeID]
	node.LastHeartbeatAt = sql.NullTime{Time: observedAt, Valid: true}
	r.nodes[nodeID] = node

	return node, nil
}

func (r *fakeNodeStateRepo) UpdateState(_ context.Context, nodeID int64, state string) (generated.Node, error) {
	r.updateStateCalls = append(r.updateStateCalls, updateStateCall{
		nodeID: nodeID,
		state:  state,
	})

	node := r.nodes[nodeID]
	node.CurrentState = state
	r.nodes[nodeID] = node

	return node, nil
}

type fakeNodeEventRecorder struct {
	calls []nodeEventRecord
}

type nodeEventRecord struct {
	node       generated.Node
	newState   string
	occurredAt time.Time
}

func (r *fakeNodeEventRecorder) CreateNodeStateChange(_ context.Context, node generated.Node, newState string, occurredAt time.Time) error {
	r.calls = append(r.calls, nodeEventRecord{
		node:       node,
		newState:   newState,
		occurredAt: occurredAt,
	})

	return nil
}

func TestEvaluateNodeOnlineTransition(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		currentState  string
		expectedState string
		transitioned  bool
	}{
		{
			name:          "already online + fresh heartbeat => stays online with no transition",
			currentState:  types.NodeStateOnline,
			expectedState: types.NodeStateOnline,
			transitioned:  false,
		},
		{
			name:          "fresh heartbeat => online",
			currentState:  types.NodeStateOffline,
			expectedState: types.NodeStateOnline,
			transitioned:  true,
		},
		{
			name:          "offline + fresh heartbeat => online transition",
			currentState:  types.NodeStateOffline,
			expectedState: types.NodeStateOnline,
			transitioned:  true,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			nextState, transitioned := evaluateNodeOnlineTransition(testCase.currentState)

			if nextState != testCase.expectedState {
				t.Fatalf("expected state %q, got %q", testCase.expectedState, nextState)
			}

			if transitioned != testCase.transitioned {
				t.Fatalf("expected transitioned=%t, got %t", testCase.transitioned, transitioned)
			}
		})
	}
}

func TestEvaluateNodeOfflineTransition(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.January, 1, 12, 0, 0, 0, time.UTC)
	staleHeartbeat := sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true}
	freshHeartbeat := sql.NullTime{Time: now.Add(-10 * time.Second), Valid: true}

	testCases := []struct {
		name               string
		currentState       string
		lastHeartbeatAt    sql.NullTime
		cutoff             time.Time
		expectedState      string
		expectedTransition bool
	}{
		{
			name:               "stale heartbeat => offline",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    staleHeartbeat,
			cutoff:             now.Add(-30 * time.Second),
			expectedState:      types.NodeStateOffline,
			expectedTransition: true,
		},
		{
			name:               "already offline + stale heartbeat => stays offline with no transition",
			currentState:       types.NodeStateOffline,
			lastHeartbeatAt:    staleHeartbeat,
			cutoff:             now.Add(-30 * time.Second),
			expectedState:      types.NodeStateOffline,
			expectedTransition: false,
		},
		{
			name:               "online + stale heartbeat => offline transition",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    staleHeartbeat,
			cutoff:             now.Add(-30 * time.Second),
			expectedState:      types.NodeStateOffline,
			expectedTransition: true,
		},
		{
			name:               "fresh heartbeat keeps online node online with no transition",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    freshHeartbeat,
			cutoff:             now.Add(-30 * time.Second),
			expectedState:      types.NodeStateOnline,
			expectedTransition: false,
		},
		{
			name:               "online with missing heartbeat timestamp stays online",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    sql.NullTime{},
			cutoff:             now.Add(-30 * time.Second),
			expectedState:      types.NodeStateOnline,
			expectedTransition: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			nextState, transitioned := evaluateNodeOfflineTransition(testCase.currentState, testCase.lastHeartbeatAt, testCase.cutoff)

			if nextState != testCase.expectedState {
				t.Fatalf("expected state %q, got %q", testCase.expectedState, nextState)
			}

			if transitioned != testCase.expectedTransition {
				t.Fatalf("expected transitioned=%t, got %t", testCase.expectedTransition, transitioned)
			}
		})
	}
}

func TestMarkOnlineFromHeartbeat(t *testing.T) {
	t.Parallel()

	observedAt := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)

	testCases := []struct {
		name               string
		currentState       string
		expectedState      string
		expectedTransition bool
	}{
		{
			name:               "fresh heartbeat updates offline node to online",
			currentState:       types.NodeStateOffline,
			expectedState:      types.NodeStateOnline,
			expectedTransition: true,
		},
		{
			name:               "already online node gets heartbeat without transition",
			currentState:       types.NodeStateOnline,
			expectedState:      types.NodeStateOnline,
			expectedTransition: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeNodeStateRepo{
				nodes: map[int64]generated.Node{
					1: {
						ID:           1,
						ProjectID:    1,
						Name:         "node-1",
						Identifier:   "node-1",
						CurrentState: testCase.currentState,
					},
				},
			}
			events := &fakeNodeEventRecorder{}
			service := &NodeStateService{
				nodeRepo:     repo,
				eventService: events,
			}

			node, err := service.MarkOnlineFromHeartbeat(context.Background(), 1, observedAt)
			if err != nil {
				t.Fatalf("MarkOnlineFromHeartbeat returned error: %v", err)
			}

			if node.CurrentState != testCase.expectedState {
				t.Fatalf("expected state %q, got %q", testCase.expectedState, node.CurrentState)
			}

			if !node.LastHeartbeatAt.Valid || !node.LastHeartbeatAt.Time.Equal(observedAt) {
				t.Fatalf("expected last heartbeat %s, got %+v", observedAt, node.LastHeartbeatAt)
			}

			if len(repo.touchHeartbeatCalls) != 1 {
				t.Fatalf("expected 1 heartbeat touch, got %d", len(repo.touchHeartbeatCalls))
			}

			expectedUpdates := 0
			expectedEvents := 0
			if testCase.expectedTransition {
				expectedUpdates = 1
				expectedEvents = 1
			}

			if len(repo.updateStateCalls) != expectedUpdates {
				t.Fatalf("expected %d state updates, got %d", expectedUpdates, len(repo.updateStateCalls))
			}

			if len(events.calls) != expectedEvents {
				t.Fatalf("expected %d node events, got %d", expectedEvents, len(events.calls))
			}

			if testCase.expectedTransition {
				if events.calls[0].newState != types.NodeStateOnline {
					t.Fatalf("expected online event, got %q", events.calls[0].newState)
				}

				if !events.calls[0].occurredAt.Equal(observedAt) {
					t.Fatalf("expected event time %s, got %s", observedAt, events.calls[0].occurredAt)
				}
			}
		})
	}
}

func TestMarkOfflineIfStale(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC)
	cutoff := now.Add(-30 * time.Second)

	testCases := []struct {
		name               string
		currentState       string
		lastHeartbeatAt    sql.NullTime
		expectedState      string
		expectedTransition bool
	}{
		{
			name:               "stale heartbeat marks online node offline",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true},
			expectedState:      types.NodeStateOffline,
			expectedTransition: true,
		},
		{
			name:               "already offline node with stale heartbeat stays offline",
			currentState:       types.NodeStateOffline,
			lastHeartbeatAt:    sql.NullTime{Time: now.Add(-2 * time.Minute), Valid: true},
			expectedState:      types.NodeStateOffline,
			expectedTransition: false,
		},
		{
			name:               "fresh node is ignored",
			currentState:       types.NodeStateOnline,
			lastHeartbeatAt:    sql.NullTime{Time: now.Add(-10 * time.Second), Valid: true},
			expectedState:      types.NodeStateOnline,
			expectedTransition: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeNodeStateRepo{
				nodes: map[int64]generated.Node{
					1: {
						ID:              1,
						ProjectID:       1,
						Name:            "node-1",
						Identifier:      "node-1",
						CurrentState:    testCase.currentState,
						LastHeartbeatAt: testCase.lastHeartbeatAt,
					},
				},
			}
			events := &fakeNodeEventRecorder{}
			service := &NodeStateService{
				nodeRepo:     repo,
				eventService: events,
			}

			node, transitioned, err := service.MarkOfflineIfStale(context.Background(), repo.nodes[1], cutoff)
			if err != nil {
				t.Fatalf("MarkOfflineIfStale returned error: %v", err)
			}

			if node.CurrentState != testCase.expectedState {
				t.Fatalf("expected state %q, got %q", testCase.expectedState, node.CurrentState)
			}

			if transitioned != testCase.expectedTransition {
				t.Fatalf("expected transitioned=%t, got %t", testCase.expectedTransition, transitioned)
			}

			expectedUpdates := 0
			expectedEvents := 0
			if testCase.expectedTransition {
				expectedUpdates = 1
				expectedEvents = 1
			}

			if len(repo.updateStateCalls) != expectedUpdates {
				t.Fatalf("expected %d state updates, got %d", expectedUpdates, len(repo.updateStateCalls))
			}

			if len(events.calls) != expectedEvents {
				t.Fatalf("expected %d node events, got %d", expectedEvents, len(events.calls))
			}

			if testCase.expectedTransition {
				if events.calls[0].newState != types.NodeStateOffline {
					t.Fatalf("expected offline event, got %q", events.calls[0].newState)
				}

				if !events.calls[0].occurredAt.Equal(cutoff) {
					t.Fatalf("expected event time %s, got %s", cutoff, events.calls[0].occurredAt)
				}
			}
		})
	}
}
