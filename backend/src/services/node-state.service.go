package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type nodeStateRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
	List(ctx context.Context) ([]generated.Node, error)
	ListByProject(ctx context.Context, projectID int64) ([]generated.Node, error)
	TouchHeartbeat(ctx context.Context, nodeID int64, observedAt time.Time) (generated.Node, error)
	UpdateState(ctx context.Context, nodeID int64, state string) (generated.Node, error)
}

type nodeStateEventService interface {
	CreateNodeStateChange(ctx context.Context, node generated.Node, newState string, occurredAt time.Time, extraDetails map[string]any) error
}

type nodeAlertEvaluator interface {
	EvaluateNodeRules(ctx context.Context, node generated.Node, occurredAt time.Time) error
}

type NodeStateService struct {
	nodeRepo       nodeStateRepository
	eventService   nodeStateEventService
	alertEvaluator nodeAlertEvaluator
}

func NewNodeStateService(nodeRepo *repositories.NodeRepository, eventService *EventService, alertEvaluator nodeAlertEvaluator) *NodeStateService {
	return &NodeStateService{
		nodeRepo:       nodeRepo,
		eventService:   eventService,
		alertEvaluator: alertEvaluator,
	}
}

func (s *NodeStateService) ListNodes(ctx context.Context, projectID *int64) ([]generated.Node, error) {
	if projectID != nil {
		return s.nodeRepo.ListByProject(ctx, *projectID)
	}

	return s.nodeRepo.List(ctx)
}

func (s *NodeStateService) MarkOnlineFromHeartbeat(ctx context.Context, nodeID int64, observedAt time.Time) (generated.Node, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return generated.Node{}, fmt.Errorf("get node: %w", err)
	}

	updatedNode, err := s.nodeRepo.TouchHeartbeat(ctx, nodeID, observedAt)
	if err != nil {
		return generated.Node{}, fmt.Errorf("touch heartbeat: %w", err)
	}

	_, transitioned := evaluateNodeOnlineTransition(node.CurrentState)
	if !transitioned {
		if err := s.evaluateNodeAlerts(ctx, updatedNode, observedAt); err != nil {
			return generated.Node{}, err
		}
		return updatedNode, nil
	}

	return s.transitionNodeState(ctx, nodeID, types.NodeStateOnline, observedAt, nil)
}

func (s *NodeStateService) MarkOfflineIfStale(ctx context.Context, node generated.Node, cutoff time.Time) (generated.Node, bool, error) {
	nextState, transitioned := evaluateNodeOfflineTransition(node.CurrentState, node.LastHeartbeatAt, cutoff)
	if !transitioned {
		return node, false, nil
	}

	updatedNode, err := s.transitionNodeState(ctx, node.ID, nextState, cutoff, nil)
	if err != nil {
		return generated.Node{}, false, err
	}

	return updatedNode, true, nil
}

func (s *NodeStateService) MarkOfflineFromShutdown(
	ctx context.Context,
	nodeID int64,
	occurredAt time.Time,
	reason string,
	payload json.RawMessage,
) (generated.Node, bool, error) {
	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		return generated.Node{}, false, fmt.Errorf("get node: %w", err)
	}

	if node.CurrentState != types.NodeStateOnline {
		return node, false, nil
	}

	extraDetails := map[string]any{
		"offline_reason": "agent_shutdown",
	}
	if reason != "" {
		extraDetails["shutdown_reason"] = reason
	}
	if len(payload) > 0 {
		var payloadValue any
		if err := json.Unmarshal(payload, &payloadValue); err != nil {
			return generated.Node{}, false, fmt.Errorf("decode shutdown payload: %w", err)
		}
		extraDetails["shutdown_payload"] = payloadValue
	}

	updatedNode, err := s.transitionNodeState(ctx, nodeID, types.NodeStateOffline, occurredAt, extraDetails)
	if err != nil {
		return generated.Node{}, false, err
	}

	return updatedNode, true, nil
}

func (s *NodeStateService) transitionNodeState(
	ctx context.Context,
	nodeID int64,
	nextState string,
	occurredAt time.Time,
	extraDetails map[string]any,
) (generated.Node, error) {
	updatedNode, err := s.nodeRepo.UpdateState(ctx, nodeID, nextState)
	if err != nil {
		return generated.Node{}, fmt.Errorf("update node state: %w", err)
	}

	if err := s.evaluateNodeAlerts(ctx, updatedNode, occurredAt); err != nil {
		return generated.Node{}, err
	}

	if s.eventService != nil {
		if err := s.eventService.CreateNodeStateChange(ctx, updatedNode, nextState, occurredAt, extraDetails); err != nil {
			return generated.Node{}, err
		}
	}

	return updatedNode, nil
}

func (s *NodeStateService) evaluateNodeAlerts(ctx context.Context, node generated.Node, occurredAt time.Time) error {
	if s.alertEvaluator == nil {
		return nil
	}

	return s.alertEvaluator.EvaluateNodeRules(ctx, node, occurredAt)
}

func evaluateNodeOnlineTransition(currentState string) (string, bool) {
	if currentState == types.NodeStateOnline {
		return types.NodeStateOnline, false
	}

	return types.NodeStateOnline, true
}

func evaluateNodeOfflineTransition(currentState string, lastHeartbeatAt sql.NullTime, cutoff time.Time) (string, bool) {
	if currentState != types.NodeStateOnline {
		return currentState, false
	}

	if !lastHeartbeatAt.Valid || !lastHeartbeatAt.Time.Before(cutoff) {
		return currentState, false
	}

	return types.NodeStateOffline, true
}
