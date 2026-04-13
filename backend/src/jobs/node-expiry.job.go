package jobs

import (
	"context"
	"log"
	"time"

	appdb "github.com/MariusBobitiu/agrafa-backend/src/db"
	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
)

var timeNow = func() time.Time {
	return time.Now().UTC()
}

type staleNodeRepository interface {
	ListStaleOnline(ctx context.Context, cutoff time.Time) ([]generated.Node, error)
}

type staleNodeStateService interface {
	MarkOfflineIfStale(ctx context.Context, node generated.Node, cutoff time.Time) (generated.Node, bool, error)
}

type NodeExpiryJob struct {
	nodeRepo     staleNodeRepository
	nodeState    staleNodeStateService
	heartbeatTTL time.Duration
	interval     time.Duration
}

func NewNodeExpiryJob(
	nodeRepo *repositories.NodeRepository,
	nodeState *services.NodeStateService,
	heartbeatTTL time.Duration,
	interval time.Duration,
) *NodeExpiryJob {
	return &NodeExpiryJob{
		nodeRepo:     nodeRepo,
		nodeState:    nodeState,
		heartbeatTTL: heartbeatTTL,
		interval:     interval,
	}
}

func (j *NodeExpiryJob) Start(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *NodeExpiryJob) runOnce(ctx context.Context) {
	ctx = appdb.WithInternalRLSBypass(ctx)
	cutoff := timeNow().Add(-j.heartbeatTTL)

	nodes, err := j.nodeRepo.ListStaleOnline(ctx, cutoff)
	if err != nil {
		log.Printf("node expiry query failed: %v", err)
		return
	}

	for _, node := range nodes {
		if _, _, err := j.nodeState.MarkOfflineIfStale(ctx, node, cutoff); err != nil {
			log.Printf("node expiry update failed for node %d: %v", node.ID, err)
		}
	}
}
