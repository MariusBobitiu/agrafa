package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type agentAuthNodeRepository interface {
	GetByAgentTokenHash(ctx context.Context, hash string) (generated.Node, error)
}

type AgentAuthService struct {
	nodeRepo agentAuthNodeRepository
}

func NewAgentAuthService(nodeRepo agentAuthNodeRepository) *AgentAuthService {
	return &AgentAuthService{nodeRepo: nodeRepo}
}

func (s *AgentAuthService) AuthenticateToken(ctx context.Context, token string) (generated.Node, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return generated.Node{}, types.ErrMissingAgentToken
	}

	node, err := s.nodeRepo.GetByAgentTokenHash(ctx, utils.HashAgentToken(token))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Node{}, types.ErrInvalidAgentToken
		}

		return generated.Node{}, fmt.Errorf("get node by agent token: %w", err)
	}

	return node, nil
}
