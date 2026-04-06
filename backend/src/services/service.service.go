package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type serviceServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
	Create(ctx context.Context, params generated.CreateServiceParams) (generated.Service, error)
	UpdateDefinition(ctx context.Context, serviceID int64, name string, checkType string, checkTarget string) (generated.Service, error)
	Delete(ctx context.Context, id int64) (int64, error)
}

type serviceServiceProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type serviceServiceNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
	EnsureManagedByProject(ctx context.Context, projectID int64, name string, identifier string) (generated.Node, error)
}

type ServiceService struct {
	serviceRepo serviceServiceRepository
	projectRepo serviceServiceProjectRepository
	nodeRepo    serviceServiceNodeRepository
}

func NewServiceService(
	serviceRepo *repositories.ServiceRepository,
	projectRepo *repositories.ProjectRepository,
	nodeRepo *repositories.NodeRepository,
) *ServiceService {
	return &ServiceService{
		serviceRepo: serviceRepo,
		projectRepo: projectRepo,
		nodeRepo:    nodeRepo,
	}
}

func (s *ServiceService) Create(
	ctx context.Context,
	input types.CreateServiceInput,
) (generated.Service, error) {
	if input.ProjectID <= 0 {
		return generated.Service{}, types.ErrInvalidProjectID
	}

	name := utils.NormalizeRequiredString(input.Name)
	if name == "" {
		return generated.Service{}, types.ErrInvalidName
	}

	checkType := utils.NormalizeRequiredString(input.CheckType)
	if checkType == "" {
		return generated.Service{}, types.ErrInvalidCheckType
	}

	checkTarget := utils.NormalizeRequiredString(input.CheckTarget)
	if checkTarget == "" {
		return generated.Service{}, types.ErrInvalidCheckTarget
	}

	if _, err := s.projectRepo.GetByID(ctx, input.ProjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Service{}, types.ErrProjectNotFound
		}

		return generated.Service{}, fmt.Errorf("get project: %w", err)
	}

	node, err := s.resolveNodeForExecutionMode(ctx, input)
	if err != nil {
		return generated.Service{}, err
	}

	service, err := s.serviceRepo.Create(ctx, generated.CreateServiceParams{
		ProjectID:   input.ProjectID,
		NodeID:      node.ID,
		Name:        name,
		CheckType:   checkType,
		CheckTarget: checkTarget,
	})
	if err != nil {
		return generated.Service{}, fmt.Errorf("create service: %w", err)
	}

	return service, nil
}

func (s *ServiceService) resolveNodeForExecutionMode(ctx context.Context, input types.CreateServiceInput) (generated.Node, error) {
	switch utils.NormalizeRequiredString(input.ExecutionMode) {
	case types.ExecutionModeManaged:
		if input.NodeID != nil {
			return generated.Node{}, types.ErrManagedExecutionDisallowsNodeID
		}

		return s.ensureManagedNode(ctx, input.ProjectID)
	case types.ExecutionModeAgent:
		if input.NodeID == nil || *input.NodeID <= 0 {
			return generated.Node{}, types.ErrAgentExecutionRequiresNodeID
		}

		node, err := s.nodeRepo.GetByID(ctx, *input.NodeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return generated.Node{}, types.ErrNodeNotFound
			}

			return generated.Node{}, fmt.Errorf("get node: %w", err)
		}

		if node.ProjectID != input.ProjectID {
			return generated.Node{}, types.ErrNodeProjectMismatch
		}

		if node.NodeType != types.NodeTypeAgent {
			return generated.Node{}, types.ErrNodeMustBeAgent
		}

		return node, nil
	default:
		return generated.Node{}, types.ErrInvalidExecutionMode
	}
}

func (s *ServiceService) ensureManagedNode(ctx context.Context, projectID int64) (generated.Node, error) {
	node, err := s.nodeRepo.EnsureManagedByProject(ctx, projectID, managedNodeName(), managedNodeIdentifier(projectID))
	if err != nil {
		return generated.Node{}, fmt.Errorf("ensure managed node: %w", err)
	}

	return node, nil
}

func managedNodeName() string {
	return "Agrafa Managed"
}

func managedNodeIdentifier(projectID int64) string {
	return fmt.Sprintf("agrafa-managed-%d", projectID)
}

func (s *ServiceService) Delete(ctx context.Context, serviceID int64) error {
	if serviceID <= 0 {
		return types.ErrInvalidServiceID
	}

	rowsDeleted, err := s.serviceRepo.Delete(ctx, serviceID)
	if err != nil {
		return fmt.Errorf("delete service: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrServiceNotFound
	}

	return nil
}

func (s *ServiceService) Update(ctx context.Context, serviceID int64, input types.UpdateServiceInput) (generated.Service, error) {
	if serviceID <= 0 {
		return generated.Service{}, types.ErrInvalidServiceID
	}

	service, err := s.serviceRepo.GetByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Service{}, types.ErrServiceNotFound
		}

		return generated.Service{}, fmt.Errorf("get service: %w", err)
	}

	name := service.Name
	checkType := service.CheckType
	checkTarget := service.CheckTarget
	hasChanges := false

	if input.Name != nil {
		hasChanges = true
		name = utils.NormalizeRequiredString(*input.Name)
		if name == "" {
			return generated.Service{}, types.ErrInvalidName
		}
	}

	if input.CheckType != nil {
		hasChanges = true
		checkType = utils.NormalizeRequiredString(*input.CheckType)
		if checkType == "" {
			return generated.Service{}, types.ErrInvalidCheckType
		}
	}

	if input.CheckTarget != nil {
		hasChanges = true
		checkTarget = utils.NormalizeRequiredString(*input.CheckTarget)
		if checkTarget == "" {
			return generated.Service{}, types.ErrInvalidCheckTarget
		}
	}

	if !hasChanges {
		return generated.Service{}, types.ErrNoFieldsToUpdate
	}

	updatedService, err := s.serviceRepo.UpdateDefinition(ctx, serviceID, name, checkType, checkTarget)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Service{}, types.ErrServiceNotFound
		}

		return generated.Service{}, fmt.Errorf("update service: %w", err)
	}

	return updatedService, nil
}
