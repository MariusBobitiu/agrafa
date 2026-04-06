package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type ServiceStateService struct {
	serviceRepo    *repositories.ServiceRepository
	eventService   *EventService
	alertEvaluator serviceAlertEvaluator
}

type serviceAlertEvaluator interface {
	EvaluateServiceRules(ctx context.Context, service generated.Service, occurredAt time.Time) error
}

func NewServiceStateService(
	serviceRepo *repositories.ServiceRepository,
	eventService *EventService,
	alertEvaluator serviceAlertEvaluator,
) *ServiceStateService {
	return &ServiceStateService{
		serviceRepo:    serviceRepo,
		eventService:   eventService,
		alertEvaluator: alertEvaluator,
	}
}

func (s *ServiceStateService) ListServices(ctx context.Context, projectID *int64) ([]generated.Service, error) {
	if projectID != nil {
		return s.serviceRepo.ListByProject(ctx, *projectID)
	}

	return s.serviceRepo.List(ctx)
}

func (s *ServiceStateService) ApplyHealthResult(ctx context.Context, serviceID int64, observedAt time.Time, isSuccess bool) (generated.Service, error) {
	currentService, err := s.serviceRepo.GetByID(ctx, serviceID)
	if err != nil {
		return generated.Service{}, fmt.Errorf("get service: %w", err)
	}

	nextState, failures, successes := calculateServiceState(currentService.CurrentState, int(currentService.ConsecutiveFailures), int(currentService.ConsecutiveSuccesses), isSuccess)

	updatedService, err := s.serviceRepo.UpdateState(ctx, generated.UpdateServiceStateParams{
		ID:                   serviceID,
		CurrentState:         nextState,
		ConsecutiveFailures:  int32(failures),
		ConsecutiveSuccesses: int32(successes),
		LastCheckAt:          sql.NullTime{Time: observedAt, Valid: true},
	})
	if err != nil {
		return generated.Service{}, fmt.Errorf("update service state: %w", err)
	}

	if s.alertEvaluator != nil {
		if err := s.alertEvaluator.EvaluateServiceRules(ctx, updatedService, observedAt); err != nil {
			return generated.Service{}, err
		}
	}

	if nextState != currentService.CurrentState {
		if err := s.eventService.CreateServiceStateChange(ctx, updatedService, nextState, observedAt); err != nil {
			return generated.Service{}, err
		}
	}

	return updatedService, nil
}

func calculateServiceState(currentState string, failures, successes int, isSuccess bool) (string, int, int) {
	if isSuccess {
		failures = 0

		switch currentState {
		case types.ServiceStateUnhealthy, types.ServiceStateDegraded:
			successes++
			if successes >= 2 {
				return types.ServiceStateHealthy, failures, 0
			}

			return types.ServiceStateDegraded, failures, successes
		default:
			return types.ServiceStateHealthy, failures, 0
		}
	}

	failures++
	successes = 0

	if failures >= 3 {
		return types.ServiceStateUnhealthy, failures, successes
	}

	return types.ServiceStateDegraded, failures, successes
}
