package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeHealthCheckRepo struct {
	createCalls int
	params      []generated.CreateHealthCheckResultParams
	operations  *[]string
}

func (r *fakeHealthCheckRepo) Create(_ context.Context, params generated.CreateHealthCheckResultParams) (generated.HealthCheckResult, error) {
	r.createCalls++
	r.params = append(r.params, params)
	if r.operations != nil {
		*r.operations = append(*r.operations, "history")
	}
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
	serviceID  int64
	observedAt time.Time
	isSuccess  bool
	operations *[]string
}

func (s *fakeHealthStateService) ApplyHealthResult(_ context.Context, serviceID int64, observedAt time.Time, isSuccess bool) (generated.Service, error) {
	s.applyCalls++
	s.serviceID = serviceID
	s.observedAt = observedAt
	s.isSuccess = isSuccess
	if s.operations != nil {
		*s.operations = append(*s.operations, "state")
	}
	return generated.Service{ID: serviceID, CurrentState: types.ServiceStateHealthy}, nil
}

func TestHealthIngestStoresHTTPAndTCPObservationsBeforeUpdatingState(t *testing.T) {
	t.Parallel()

	statusOK := int32(200)
	statusUnavailable := int32(503)
	responseTime := int32(87)
	observedAt := time.Date(2026, time.August, 23, 12, 30, 0, 0, time.UTC)

	testCases := []struct {
		name       string
		checkType  string
		storedType string
		isSuccess  bool
		statusCode *int32
		message    string
	}{
		{name: "successful HTTP", checkType: " HTTP ", storedType: "http", isSuccess: true, statusCode: &statusOK, message: "200 OK"},
		{name: "failed HTTP", checkType: "http", isSuccess: false, statusCode: &statusUnavailable, message: "503 Service Unavailable"},
		{name: "successful TCP", checkType: "tcp", isSuccess: true, message: "tcp connection succeeded"},
		{name: "failed TCP", checkType: "tcp", isSuccess: false, message: "connection refused"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			operations := []string{}
			healthCheckRepo := &fakeHealthCheckRepo{operations: &operations}
			serviceRepo := &fakeHealthServiceRepo{service: generated.Service{
				ID:        11,
				NodeID:    99,
				CheckType: testCase.checkType,
			}}
			stateService := &fakeHealthStateService{operations: &operations}
			service := NewHealthIngestionService(healthCheckRepo, serviceRepo, stateService)

			inputStatusCode := testCase.statusCode
			if testCase.checkType == "tcp" {
				inputStatusCode = &statusOK
			}

			updated, err := service.Ingest(context.Background(), types.HealthCheckInput{
				AuthenticatedNodeID: 99,
				ServiceID:           11,
				Source:              types.HealthCheckSourceManaged,
				ObservedAt:          observedAt,
				IsSuccess:           testCase.isSuccess,
				StatusCode:          inputStatusCode,
				ResponseTimeMs:      &responseTime,
				Message:             testCase.message,
			})
			if err != nil {
				t.Fatalf("Ingest() error = %v", err)
			}
			if updated.ID != 11 || updated.CurrentState != types.ServiceStateHealthy {
				t.Fatalf("unexpected updated state: %#v", updated)
			}
			if len(healthCheckRepo.params) != 1 {
				t.Fatalf("history writes = %d, want 1", len(healthCheckRepo.params))
			}

			stored := healthCheckRepo.params[0]
			expectedCheckType := testCase.storedType
			if expectedCheckType == "" {
				expectedCheckType = testCase.checkType
			}
			if stored.ServiceID != 11 || stored.NodeID != 99 || stored.CheckType != expectedCheckType {
				t.Fatalf("unexpected stored identity: %#v", stored)
			}
			if stored.Source != types.HealthCheckSourceManaged || !stored.ObservedAt.Equal(observedAt) {
				t.Fatalf("unexpected stored source/time: %#v", stored)
			}
			if stored.IsSuccess != testCase.isSuccess || stored.Message != testCase.message {
				t.Fatalf("unexpected stored outcome: %#v", stored)
			}
			if !stored.ResponseTimeMs.Valid || stored.ResponseTimeMs.Int32 != responseTime {
				t.Fatalf("response_time_ms = %#v, want %d", stored.ResponseTimeMs, responseTime)
			}
			if testCase.statusCode == nil {
				if stored.StatusCode != (sql.NullInt32{}) {
					t.Fatalf("TCP status_code = %#v, want null", stored.StatusCode)
				}
			} else if !stored.StatusCode.Valid || stored.StatusCode.Int32 != *testCase.statusCode {
				t.Fatalf("status_code = %#v, want %d", stored.StatusCode, *testCase.statusCode)
			}
			if len(operations) != 2 || operations[0] != "history" || operations[1] != "state" {
				t.Fatalf("operation order = %#v, want history then state", operations)
			}
			if stateService.applyCalls != 1 || stateService.serviceID != 11 || stateService.isSuccess != testCase.isSuccess || !stateService.observedAt.Equal(observedAt) {
				t.Fatalf("unexpected state update: %#v", stateService)
			}
		})
	}
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
