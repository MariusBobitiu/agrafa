package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeHeartbeatRepo struct {
	createCalls int
}

func (r *fakeHeartbeatRepo) Create(_ context.Context, _ generated.CreateHeartbeatParams) (generated.Heartbeat, error) {
	r.createCalls++
	return generated.Heartbeat{}, nil
}

type fakeHeartbeatNodeStateService struct {
	markCalls int
}

func (s *fakeHeartbeatNodeStateService) MarkOnlineFromHeartbeat(_ context.Context, _ int64, _ time.Time) (generated.Node, error) {
	s.markCalls++
	return generated.Node{ID: 1}, nil
}

func TestHeartbeatIngestRejectsMismatchedReportedNodeID(t *testing.T) {
	t.Parallel()

	heartbeatRepo := &fakeHeartbeatRepo{}
	nodeStateService := &fakeHeartbeatNodeStateService{}
	service := NewHeartbeatService(heartbeatRepo, nodeStateService)
	reportedNodeID := int64(99)

	_, err := service.Ingest(context.Background(), types.HeartbeatInput{
		AuthenticatedNodeID: 1,
		ReportedNodeID:      &reportedNodeID,
		ObservedAt:          time.Now().UTC(),
	})
	if !errors.Is(err, types.ErrAgentNodeMismatch) {
		t.Fatalf("expected ErrAgentNodeMismatch, got %v", err)
	}

	if heartbeatRepo.createCalls != 0 {
		t.Fatalf("expected no heartbeat write, got %d", heartbeatRepo.createCalls)
	}

	if nodeStateService.markCalls != 0 {
		t.Fatalf("expected no node state update, got %d", nodeStateService.markCalls)
	}
}
