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

type nodeServiceNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
	Create(ctx context.Context, params generated.CreateNodeParams) (generated.Node, error)
	UpdateIdentity(ctx context.Context, nodeID int64, name string, identifier string) (generated.Node, error)
	UpdateAgentTokenHash(ctx context.Context, nodeID int64, hash string) (generated.Node, error)
	Delete(ctx context.Context, nodeID int64) (int64, error)
}

type nodeServiceProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type nodeServiceServiceRepository interface {
	CountByNodeID(ctx context.Context, nodeID int64) (int64, error)
}

type NodeService struct {
	nodeRepo    nodeServiceNodeRepository
	projectRepo nodeServiceProjectRepository
	serviceRepo nodeServiceServiceRepository
}

func NewNodeService(
	nodeRepo *repositories.NodeRepository,
	projectRepo *repositories.ProjectRepository,
	serviceRepo *repositories.ServiceRepository,
) *NodeService {
	return &NodeService{
		nodeRepo:    nodeRepo,
		projectRepo: projectRepo,
		serviceRepo: serviceRepo,
	}
}

func (s *NodeService) Create(ctx context.Context, projectID int64, name string) (generated.Node, error) {
	if projectID <= 0 {
		return generated.Node{}, types.ErrInvalidProjectID
	}

	name = utils.NormalizeRequiredString(name)
	if name == "" {
		return generated.Node{}, types.ErrInvalidName
	}

	identifier := utils.BuildSlug(name)
	if identifier == "" {
		return generated.Node{}, types.ErrInvalidName
	}

	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Node{}, types.ErrProjectNotFound
		}

		return generated.Node{}, fmt.Errorf("get project: %w", err)
	}

	node, err := s.nodeRepo.Create(ctx, generated.CreateNodeParams{
		ProjectID:  projectID,
		Name:       name,
		Identifier: identifier,
	})
	if err != nil {
		return generated.Node{}, fmt.Errorf("create node: %w", err)
	}

	return node, nil
}

func (s *NodeService) RegenerateAgentToken(ctx context.Context, nodeID int64) (string, error) {
	if nodeID <= 0 {
		return "", types.ErrInvalidNodeID
	}

	if _, err := s.nodeRepo.GetByID(ctx, nodeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", types.ErrNodeNotFound
		}

		return "", fmt.Errorf("get node: %w", err)
	}

	token, err := utils.GenerateAgentToken()
	if err != nil {
		return "", fmt.Errorf("generate agent token: %w", err)
	}

	if _, err := s.nodeRepo.UpdateAgentTokenHash(ctx, nodeID, utils.HashAgentToken(token)); err != nil {
		return "", fmt.Errorf("update node agent token: %w", err)
	}

	return token, nil
}

func (s *NodeService) Delete(ctx context.Context, nodeID int64) error {
	if nodeID <= 0 {
		return types.ErrInvalidNodeID
	}

	if _, err := s.nodeRepo.GetByID(ctx, nodeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrNodeNotFound
		}

		return fmt.Errorf("get node: %w", err)
	}

	serviceCount, err := s.serviceRepo.CountByNodeID(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("count node services: %w", err)
	}
	if serviceCount > 0 {
		return types.ErrNodeHasServices
	}

	rowsDeleted, err := s.nodeRepo.Delete(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("delete node: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrNodeNotFound
	}

	return nil
}

func (s *NodeService) Update(ctx context.Context, nodeID int64, input types.UpdateNodeInput) (generated.Node, error) {
	if nodeID <= 0 {
		return generated.Node{}, types.ErrInvalidNodeID
	}

	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Node{}, types.ErrNodeNotFound
		}

		return generated.Node{}, fmt.Errorf("get node: %w", err)
	}

	name := node.Name
	identifier := node.Identifier
	hasChanges := false

	if input.Name != nil {
		hasChanges = true
		name = utils.NormalizeRequiredString(*input.Name)
		if name == "" {
			return generated.Node{}, types.ErrInvalidName
		}
	}

	if input.Identifier != nil {
		hasChanges = true
		identifier = utils.BuildSlug(*input.Identifier)
		if identifier == "" {
			return generated.Node{}, types.ErrInvalidIdentifier
		}
	}

	if !hasChanges {
		return generated.Node{}, types.ErrNoFieldsToUpdate
	}

	updatedNode, err := s.nodeRepo.UpdateIdentity(ctx, nodeID, name, identifier)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Node{}, types.ErrNodeNotFound
		}

		return generated.Node{}, fmt.Errorf("update node: %w", err)
	}

	return updatedNode, nil
}
