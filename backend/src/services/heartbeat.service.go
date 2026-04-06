package services

import (
	"context"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type heartbeatNodeStateService interface {
	MarkOnlineFromHeartbeat(ctx context.Context, nodeID int64, observedAt time.Time) (generated.Node, error)
}

type HeartbeatService struct {
	heartbeatRepo    heartbeatRepository
	nodeStateService heartbeatNodeStateService
}

type heartbeatRepository interface {
	Create(ctx context.Context, params generated.CreateHeartbeatParams) (generated.Heartbeat, error)
}

func NewHeartbeatService(heartbeatRepo heartbeatRepository, nodeStateService heartbeatNodeStateService) *HeartbeatService {
	return &HeartbeatService{
		heartbeatRepo:    heartbeatRepo,
		nodeStateService: nodeStateService,
	}
}

func (s *HeartbeatService) Ingest(ctx context.Context, input types.HeartbeatInput) (generated.Node, error) {
	if input.ReportedNodeID != nil && *input.ReportedNodeID != input.AuthenticatedNodeID {
		return generated.Node{}, types.ErrAgentNodeMismatch
	}

	if _, err := s.heartbeatRepo.Create(ctx, generated.CreateHeartbeatParams{
		NodeID:     input.AuthenticatedNodeID,
		ObservedAt: input.ObservedAt,
		Source:     input.Source,
		Payload:    utils.NormalizeJSON(input.Payload),
	}); err != nil {
		return generated.Node{}, fmt.Errorf("create heartbeat: %w", err)
	}

	return s.nodeStateService.MarkOnlineFromHeartbeat(ctx, input.AuthenticatedNodeID, input.ObservedAt)
}
