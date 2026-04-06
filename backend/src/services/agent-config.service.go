package services

import (
	"context"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

const (
	defaultAgentConfigIntervalSeconds = 30
	defaultAgentConfigTimeoutSeconds  = 10
)

type agentConfigRepository interface {
	ListAgentConfigChecksByNodeID(ctx context.Context, nodeID int64) ([]generated.ListAgentConfigChecksByNodeIDRow, error)
}

type AgentConfigService struct {
	serviceRepo agentConfigRepository
}

func NewAgentConfigService(serviceRepo agentConfigRepository) *AgentConfigService {
	return &AgentConfigService{
		serviceRepo: serviceRepo,
	}
}

func (s *AgentConfigService) GetForNode(ctx context.Context, node generated.Node) (types.AgentConfigData, error) {
	rows, err := s.serviceRepo.ListAgentConfigChecksByNodeID(ctx, node.ID)
	if err != nil {
		return types.AgentConfigData{}, fmt.Errorf("list agent config checks: %w", err)
	}

	healthChecks := make([]types.AgentConfigCheckData, 0, len(rows))
	for _, row := range rows {
		healthChecks = append(healthChecks, types.AgentConfigCheckData{
			ServiceID:       row.ServiceID,
			Name:            row.Name,
			CheckType:       row.CheckType,
			CheckTarget:     row.CheckTarget,
			IntervalSeconds: defaultAgentConfigIntervalSeconds,
			TimeoutSeconds:  defaultAgentConfigTimeoutSeconds,
		})
	}

	return types.AgentConfigData{
		Node: types.AgentConfigNodeData{
			ID:         node.ID,
			Name:       node.Name,
			Identifier: node.Identifier,
		},
		HealthChecks: healthChecks,
	}, nil
}
