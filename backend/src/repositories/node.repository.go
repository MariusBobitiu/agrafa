package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type NodeRepository struct {
	queries *generated.Queries
}

func NewNodeRepository(queries *generated.Queries) *NodeRepository {
	return &NodeRepository{queries: queries}
}

func (r *NodeRepository) GetByID(ctx context.Context, id int64) (generated.Node, error) {
	return r.queries.GetNodeByID(ctx, id)
}

func (r *NodeRepository) GetByAgentTokenHash(ctx context.Context, hash string) (generated.Node, error) {
	return r.queries.GetNodeByAgentTokenHash(ctx, sql.NullString{String: hash, Valid: hash != ""})
}

func (r *NodeRepository) Create(ctx context.Context, params generated.CreateNodeParams) (generated.Node, error) {
	return r.queries.CreateNode(ctx, params)
}

func (r *NodeRepository) EnsureManagedByProject(ctx context.Context, projectID int64, name string, identifier string) (generated.Node, error) {
	return r.queries.EnsureManagedNodeByProject(ctx, generated.EnsureManagedNodeByProjectParams{
		ProjectID:  projectID,
		Name:       name,
		Identifier: identifier,
	})
}

func (r *NodeRepository) List(ctx context.Context) ([]generated.Node, error) {
	return r.queries.ListNodes(ctx)
}

func (r *NodeRepository) ListVisible(ctx context.Context) ([]generated.Node, error) {
	return r.queries.ListVisibleNodes(ctx)
}

func (r *NodeRepository) ListByProject(ctx context.Context, projectID int64) ([]generated.Node, error) {
	return r.queries.ListNodesByProject(ctx, projectID)
}

func (r *NodeRepository) ListVisibleByProject(ctx context.Context, projectID int64) ([]generated.Node, error) {
	return r.queries.ListVisibleNodesByProject(ctx, projectID)
}

func (r *NodeRepository) TouchHeartbeat(ctx context.Context, nodeID int64, observedAt time.Time) (generated.Node, error) {
	return r.queries.UpdateNodeHeartbeat(ctx, generated.UpdateNodeHeartbeatParams{
		ID:              nodeID,
		LastHeartbeatAt: sql.NullTime{Time: observedAt, Valid: true},
	})
}

func (r *NodeRepository) UpdateState(ctx context.Context, nodeID int64, state string) (generated.Node, error) {
	return r.queries.UpdateNodeState(ctx, generated.UpdateNodeStateParams{
		ID:           nodeID,
		CurrentState: state,
	})
}

func (r *NodeRepository) UpdateAgentTokenHash(ctx context.Context, nodeID int64, hash string) (generated.Node, error) {
	return r.queries.UpdateNodeAgentTokenHash(ctx, generated.UpdateNodeAgentTokenHashParams{
		ID:             nodeID,
		AgentTokenHash: sql.NullString{String: hash, Valid: hash != ""},
	})
}

func (r *NodeRepository) UpdateIdentity(ctx context.Context, nodeID int64, name string, identifier string) (generated.Node, error) {
	return r.queries.UpdateNodeIdentity(ctx, generated.UpdateNodeIdentityParams{
		ID:         nodeID,
		Name:       name,
		Identifier: identifier,
	})
}

func (r *NodeRepository) Delete(ctx context.Context, nodeID int64) (int64, error) {
	return r.queries.DeleteNodeByID(ctx, nodeID)
}

func (r *NodeRepository) ListStaleOnline(ctx context.Context, cutoff time.Time) ([]generated.Node, error) {
	return r.queries.ListStaleOnlineNodes(ctx, sql.NullTime{Time: cutoff, Valid: true})
}
