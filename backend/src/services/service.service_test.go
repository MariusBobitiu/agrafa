package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeServiceServiceRepo struct {
	service         generated.Service
	createParams    generated.CreateServiceParams
	getErr          error
	updateID        int64
	updateName      string
	updateCheckType string
	updateTarget    string
	updateErr       error
	deleteRows      int64
	deleteErr       error
}

func (r *fakeServiceServiceRepo) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return r.service, r.getErr
}

func (r *fakeServiceServiceRepo) Create(_ context.Context, params generated.CreateServiceParams) (generated.Service, error) {
	r.createParams = params
	return r.service, nil
}

func (r *fakeServiceServiceRepo) UpdateDefinition(_ context.Context, serviceID int64, name string, checkType string, checkTarget string) (generated.Service, error) {
	r.updateID = serviceID
	r.updateName = name
	r.updateCheckType = checkType
	r.updateTarget = checkTarget
	if r.updateErr != nil {
		return generated.Service{}, r.updateErr
	}

	r.service.ID = serviceID
	r.service.Name = name
	r.service.CheckType = checkType
	r.service.CheckTarget = checkTarget
	return r.service, nil
}

func (r *fakeServiceServiceRepo) Delete(_ context.Context, _ int64) (int64, error) {
	return r.deleteRows, r.deleteErr
}

type fakeServiceServiceProjectRepo struct{}

func (r *fakeServiceServiceProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return generated.Project{ID: 1}, nil
}

type fakeServiceServiceNodeRepo struct {
	node                 generated.Node
	err                  error
	managedNode          generated.Node
	ensureManagedCalls   int
	ensureManagedProject int64
	ensureManagedName    string
	ensureManagedIdent   string
}

func (r *fakeServiceServiceNodeRepo) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return r.node, r.err
}

func (r *fakeServiceServiceNodeRepo) EnsureManagedByProject(_ context.Context, projectID int64, name string, identifier string) (generated.Node, error) {
	r.ensureManagedCalls++
	r.ensureManagedProject = projectID
	r.ensureManagedName = name
	r.ensureManagedIdent = identifier
	return r.managedNode, r.err
}

type fakeManagedServiceBootstrapChecker struct {
	checked []generated.Service
	err     error
}

func (c *fakeManagedServiceBootstrapChecker) CheckNow(_ context.Context, service generated.Service) error {
	c.checked = append(c.checked, service)
	return c.err
}

func TestServiceServiceCreateManagedAutoCreatesManagedNode(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceServiceRepo{
		service: generated.Service{ID: 9, NodeID: 88},
	}
	nodeRepo := &fakeServiceServiceNodeRepo{
		managedNode: generated.Node{ID: 88, ProjectID: 3, NodeType: types.NodeTypeManaged, IsVisible: false},
	}
	service := &ServiceService{
		serviceRepo: repo,
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    nodeRepo,
	}

	created, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		ExecutionMode: types.ExecutionModeManaged,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.ID != 9 {
		t.Fatalf("created.ID = %d, want 9", created.ID)
	}
	if nodeRepo.ensureManagedCalls != 1 {
		t.Fatalf("ensureManagedCalls = %d, want 1", nodeRepo.ensureManagedCalls)
	}
	if repo.createParams.NodeID != 88 {
		t.Fatalf("createParams.NodeID = %d, want 88", repo.createParams.NodeID)
	}
	if nodeRepo.ensureManagedProject != 3 {
		t.Fatalf("ensureManagedProject = %d, want 3", nodeRepo.ensureManagedProject)
	}
	if nodeRepo.ensureManagedIdent != "agrafa-managed-3" {
		t.Fatalf("managed identifier = %q, want agrafa-managed-3", nodeRepo.ensureManagedIdent)
	}
}

func TestServiceServiceCreateManagedTriggersImmediateCheck(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceServiceRepo{
		service: generated.Service{ID: 9, NodeID: 88, CheckType: "http", CheckTarget: "https://example.com/health"},
	}
	nodeRepo := &fakeServiceServiceNodeRepo{
		managedNode: generated.Node{ID: 88, ProjectID: 3, NodeType: types.NodeTypeManaged, IsVisible: false},
	}
	checker := &fakeManagedServiceBootstrapChecker{}
	service := (&ServiceService{
		serviceRepo: repo,
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    nodeRepo,
	}).WithManagedChecker(checker)

	created, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		ExecutionMode: types.ExecutionModeManaged,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(checker.checked) != 1 {
		t.Fatalf("checked len = %d, want 1", len(checker.checked))
	}
	if checker.checked[0].ID != created.ID {
		t.Fatalf("checked service id = %d, want %d", checker.checked[0].ID, created.ID)
	}
}

func TestServiceServiceCreateAgentDoesNotTriggerImmediateManagedCheck(t *testing.T) {
	t.Parallel()

	nodeID := int64(11)
	repo := &fakeServiceServiceRepo{
		service: generated.Service{ID: 9, NodeID: nodeID, CheckType: "http", CheckTarget: "https://example.com/health"},
	}
	checker := &fakeManagedServiceBootstrapChecker{}
	service := (&ServiceService{
		serviceRepo: repo,
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo: &fakeServiceServiceNodeRepo{
			node: generated.Node{ID: nodeID, ProjectID: 3, NodeType: types.NodeTypeAgent},
		},
	}).WithManagedChecker(checker)

	if _, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		NodeID:        &nodeID,
		ExecutionMode: types.ExecutionModeAgent,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if len(checker.checked) != 0 {
		t.Fatalf("checked len = %d, want 0", len(checker.checked))
	}
}

func TestServiceServiceCreateManagedReusesExistingManagedNode(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceServiceRepo{
		service: generated.Service{ID: 9, NodeID: 44},
	}
	nodeRepo := &fakeServiceServiceNodeRepo{
		managedNode: generated.Node{ID: 44, ProjectID: 7, NodeType: types.NodeTypeManaged, IsVisible: false},
	}
	service := &ServiceService{
		serviceRepo: repo,
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    nodeRepo,
	}

	if _, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     7,
		ExecutionMode: types.ExecutionModeManaged,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	}); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if nodeRepo.ensureManagedCalls != 1 {
		t.Fatalf("ensureManagedCalls = %d, want 1", nodeRepo.ensureManagedCalls)
	}
	if repo.createParams.NodeID != 44 {
		t.Fatalf("createParams.NodeID = %d, want 44", repo.createParams.NodeID)
	}
}

func TestServiceServiceCreateAgentRequiresNodeID(t *testing.T) {
	t.Parallel()

	service := &ServiceService{
		serviceRepo: &fakeServiceServiceRepo{},
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    &fakeServiceServiceNodeRepo{},
	}

	_, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		ExecutionMode: types.ExecutionModeAgent,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	})
	if !errors.Is(err, types.ErrAgentExecutionRequiresNodeID) {
		t.Fatalf("Create() error = %v, want ErrAgentExecutionRequiresNodeID", err)
	}
}

func TestServiceServiceCreateAgentRejectsNodeFromAnotherProject(t *testing.T) {
	t.Parallel()

	nodeID := int64(11)
	service := &ServiceService{
		serviceRepo: &fakeServiceServiceRepo{},
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo: &fakeServiceServiceNodeRepo{
			node: generated.Node{ID: nodeID, ProjectID: 99, NodeType: types.NodeTypeAgent},
		},
	}

	_, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		NodeID:        &nodeID,
		ExecutionMode: types.ExecutionModeAgent,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	})
	if !errors.Is(err, types.ErrNodeProjectMismatch) {
		t.Fatalf("Create() error = %v, want ErrNodeProjectMismatch", err)
	}
}

func TestServiceServiceCreateAgentRejectsManagedNode(t *testing.T) {
	t.Parallel()

	nodeID := int64(11)
	service := &ServiceService{
		serviceRepo: &fakeServiceServiceRepo{},
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo: &fakeServiceServiceNodeRepo{
			node: generated.Node{ID: nodeID, ProjectID: 3, NodeType: types.NodeTypeManaged},
		},
	}

	_, err := service.Create(context.Background(), types.CreateServiceInput{
		ProjectID:     3,
		NodeID:        &nodeID,
		ExecutionMode: types.ExecutionModeAgent,
		Name:          "api",
		CheckType:     "http",
		CheckTarget:   "https://example.com/health",
	})
	if !errors.Is(err, types.ErrNodeMustBeAgent) {
		t.Fatalf("Create() error = %v, want ErrNodeMustBeAgent", err)
	}
}

func TestServiceServiceUpdateAllowedFieldsOnly(t *testing.T) {
	t.Parallel()

	repo := &fakeServiceServiceRepo{
		service: generated.Service{
			ID:          9,
			Name:        "api",
			CheckType:   "http",
			CheckTarget: "https://old.example.com/health",
		},
	}
	service := &ServiceService{
		serviceRepo: repo,
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    &fakeServiceServiceNodeRepo{},
	}

	name := "api-v2"
	checkType := "tcp"
	checkTarget := "api.internal:9000"
	updated, err := service.Update(context.Background(), 9, types.UpdateServiceInput{
		Name:        &name,
		CheckType:   &checkType,
		CheckTarget: &checkTarget,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if repo.updateID != 9 {
		t.Fatalf("updateID = %d, want 9", repo.updateID)
	}
	if repo.updateName != name || repo.updateCheckType != checkType || repo.updateTarget != checkTarget {
		t.Fatalf("unexpected update values: %#v", repo)
	}
	if updated.CheckTarget != checkTarget {
		t.Fatalf("updated.CheckTarget = %q, want %q", updated.CheckTarget, checkTarget)
	}
}

func TestServiceServiceUpdateRejectsEmptyUpdate(t *testing.T) {
	t.Parallel()

	service := &ServiceService{
		serviceRepo: &fakeServiceServiceRepo{
			service: generated.Service{ID: 9},
		},
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    &fakeServiceServiceNodeRepo{},
	}

	_, err := service.Update(context.Background(), 9, types.UpdateServiceInput{})
	if !errors.Is(err, types.ErrNoFieldsToUpdate) {
		t.Fatalf("Update() error = %v, want ErrNoFieldsToUpdate", err)
	}
}

func TestServiceServiceUpdateMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &ServiceService{
		serviceRepo: &fakeServiceServiceRepo{getErr: sql.ErrNoRows},
		projectRepo: &fakeServiceServiceProjectRepo{},
		nodeRepo:    &fakeServiceServiceNodeRepo{},
	}

	name := "api"
	_, err := service.Update(context.Background(), 9, types.UpdateServiceInput{Name: &name})
	if !errors.Is(err, types.ErrServiceNotFound) {
		t.Fatalf("Update() error = %v, want ErrServiceNotFound", err)
	}
}
