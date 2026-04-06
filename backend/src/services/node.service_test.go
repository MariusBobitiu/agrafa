package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeNodeServiceNodeRepo struct {
	node       generated.Node
	getErr     error
	updateID   int64
	updateName string
	updateSlug string
	updateErr  error
	deleteID   int64
	deleteRows int64
	deleteErr  error
}

func (r *fakeNodeServiceNodeRepo) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return r.node, r.getErr
}

func (r *fakeNodeServiceNodeRepo) Create(_ context.Context, _ generated.CreateNodeParams) (generated.Node, error) {
	return r.node, nil
}

func (r *fakeNodeServiceNodeRepo) UpdateAgentTokenHash(_ context.Context, _ int64, _ string) (generated.Node, error) {
	return r.node, nil
}

func (r *fakeNodeServiceNodeRepo) UpdateIdentity(_ context.Context, nodeID int64, name string, identifier string) (generated.Node, error) {
	r.updateID = nodeID
	r.updateName = name
	r.updateSlug = identifier
	if r.updateErr != nil {
		return generated.Node{}, r.updateErr
	}

	r.node.ID = nodeID
	r.node.Name = name
	r.node.Identifier = identifier
	return r.node, nil
}

func (r *fakeNodeServiceNodeRepo) Delete(_ context.Context, nodeID int64) (int64, error) {
	r.deleteID = nodeID
	return r.deleteRows, r.deleteErr
}

type fakeNodeServiceProjectRepo struct{}

func (r *fakeNodeServiceProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return generated.Project{ID: 1}, nil
}

type fakeNodeServiceServiceRepo struct {
	count int64
	err   error
}

func (r *fakeNodeServiceServiceRepo) CountByNodeID(_ context.Context, _ int64) (int64, error) {
	return r.count, r.err
}

func TestNodeServiceDeleteRejectsWhenServicesExist(t *testing.T) {
	t.Parallel()

	service := &NodeService{
		nodeRepo:    &fakeNodeServiceNodeRepo{node: generated.Node{ID: 5}},
		projectRepo: &fakeNodeServiceProjectRepo{},
		serviceRepo: &fakeNodeServiceServiceRepo{count: 2},
	}

	err := service.Delete(context.Background(), 5)
	if !errors.Is(err, types.ErrNodeHasServices) {
		t.Fatalf("Delete() error = %v, want ErrNodeHasServices", err)
	}
}

func TestNodeServiceDeleteMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &NodeService{
		nodeRepo:    &fakeNodeServiceNodeRepo{getErr: sql.ErrNoRows},
		projectRepo: &fakeNodeServiceProjectRepo{},
		serviceRepo: &fakeNodeServiceServiceRepo{},
	}

	err := service.Delete(context.Background(), 5)
	if !errors.Is(err, types.ErrNodeNotFound) {
		t.Fatalf("Delete() error = %v, want ErrNodeNotFound", err)
	}
}

func TestNodeServiceUpdateAllowedFieldsOnly(t *testing.T) {
	t.Parallel()

	repo := &fakeNodeServiceNodeRepo{
		node: generated.Node{
			ID:         5,
			Name:       "node-a",
			Identifier: "node-a",
		},
	}
	service := &NodeService{
		nodeRepo:    repo,
		projectRepo: &fakeNodeServiceProjectRepo{},
		serviceRepo: &fakeNodeServiceServiceRepo{},
	}

	name := "Node A Prime"
	identifier := "Node A Prime"
	node, err := service.Update(context.Background(), 5, types.UpdateNodeInput{
		Name:       &name,
		Identifier: &identifier,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updateID != 5 {
		t.Fatalf("updateID = %d, want 5", repo.updateID)
	}
	if repo.updateName != name {
		t.Fatalf("updateName = %q, want %q", repo.updateName, name)
	}
	if repo.updateSlug != "node-a-prime" {
		t.Fatalf("updateSlug = %q, want node-a-prime", repo.updateSlug)
	}
	if node.Identifier != "node-a-prime" {
		t.Fatalf("node.Identifier = %q, want node-a-prime", node.Identifier)
	}
}

func TestNodeServiceUpdateRejectsEmptyUpdate(t *testing.T) {
	t.Parallel()

	service := &NodeService{
		nodeRepo:    &fakeNodeServiceNodeRepo{node: generated.Node{ID: 5}},
		projectRepo: &fakeNodeServiceProjectRepo{},
		serviceRepo: &fakeNodeServiceServiceRepo{},
	}

	_, err := service.Update(context.Background(), 5, types.UpdateNodeInput{})
	if !errors.Is(err, types.ErrNoFieldsToUpdate) {
		t.Fatalf("Update() error = %v, want ErrNoFieldsToUpdate", err)
	}
}
