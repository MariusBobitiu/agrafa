package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type HealthIngestionService struct {
	healthCheckRepo     healthCheckRepository
	serviceRepo         healthIngestionServiceRepository
	serviceStateService healthServiceStateService
}

type healthCheckRepository interface {
	Create(ctx context.Context, params generated.CreateHealthCheckResultParams) (generated.HealthCheckResult, error)
}

type healthIngestionServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
}

type healthServiceStateService interface {
	ApplyHealthResult(ctx context.Context, serviceID int64, observedAt time.Time, isSuccess bool) (generated.Service, error)
}

func NewHealthIngestionService(
	healthCheckRepo healthCheckRepository,
	serviceRepo healthIngestionServiceRepository,
	serviceStateService healthServiceStateService,
) *HealthIngestionService {
	return &HealthIngestionService{
		healthCheckRepo:     healthCheckRepo,
		serviceRepo:         serviceRepo,
		serviceStateService: serviceStateService,
	}
}

func (s *HealthIngestionService) Ingest(ctx context.Context, input types.HealthCheckInput) (generated.Service, error) {
	service, err := s.serviceRepo.GetByID(ctx, input.ServiceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return generated.Service{}, types.ErrServiceNotFound
		}

		return generated.Service{}, fmt.Errorf("get service: %w", err)
	}

	if service.NodeID != input.AuthenticatedNodeID {
		return generated.Service{}, types.ErrServiceNodeMismatch
	}

	if _, err := s.healthCheckRepo.Create(ctx, generated.CreateHealthCheckResultParams{
		ServiceID:      input.ServiceID,
		ObservedAt:     input.ObservedAt,
		IsSuccess:      input.IsSuccess,
		StatusCode:     utils.ToNullInt32(input.StatusCode),
		ResponseTimeMs: utils.ToNullInt32(input.ResponseTimeMs),
		Message:        input.Message,
		Payload:        utils.NormalizeJSON(input.Payload),
	}); err != nil {
		return generated.Service{}, fmt.Errorf("create health check result: %w", err)
	}

	return s.serviceStateService.ApplyHealthResult(ctx, input.ServiceID, input.ObservedAt, input.IsSuccess)
}
