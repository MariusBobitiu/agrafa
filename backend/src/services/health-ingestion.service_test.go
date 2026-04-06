package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeHealthCheckRepo struct {
	createCalls int
}

func (r *fakeHealthCheckRepo) Create(_ context.Context, _ generated.CreateHealthCheckResultParams) (generated.HealthCheckResult, error) {
	r.createCalls++
	return generated.HealthCheckResult{}, nil
}

type fakeHealthServiceRepo struct {
	service generated.Service
}

func (r *fakeHealthServiceRepo) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return r.service, nil
}

type fakeHealthStateService struct {
	applyCalls int
}

func (s *fakeHealthStateService) ApplyHealthResult(_ context.Context, _ int64, _ time.Time, _ bool) (generated.Service, error) {
	s.applyCalls++
	return generated.Service{}, nil
}

func TestHealthIngestRejectsServiceOnAnotherNode(t *testing.T) {
	t.Parallel()

	healthCheckRepo := &fakeHealthCheckRepo{}
	serviceRepo := &fakeHealthServiceRepo{
		service: generated.Service{
			ID:     11,
			NodeID: 99,
		},
	}
	stateService := &fakeHealthStateService{}
	service := NewHealthIngestionService(healthCheckRepo, serviceRepo, stateService)

	_, err := service.Ingest(context.Background(), types.HealthCheckInput{
		AuthenticatedNodeID: 1,
		ServiceID:           11,
		ObservedAt:          time.Now().UTC(),
		IsSuccess:           true,
	})
	if !errors.Is(err, types.ErrServiceNodeMismatch) {
		t.Fatalf("expected ErrServiceNodeMismatch, got %v", err)
	}

	if healthCheckRepo.createCalls != 0 {
		t.Fatalf("expected no health check writes, got %d", healthCheckRepo.createCalls)
	}

	if stateService.applyCalls != 0 {
		t.Fatalf("expected no service state updates, got %d", stateService.applyCalls)
	}
}
