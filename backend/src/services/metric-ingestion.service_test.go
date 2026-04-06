package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeMetricRepo struct {
	createCalls int
}

func (r *fakeMetricRepo) Create(_ context.Context, _ generated.CreateMetricSampleParams) (generated.MetricSample, error) {
	r.createCalls++
	return generated.MetricSample{}, nil
}

type fakeMetricServiceRepo struct {
	service generated.Service
}

func (r *fakeMetricServiceRepo) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return r.service, nil
}

func TestMetricIngestRejectsMismatchedReportedNodeID(t *testing.T) {
	t.Parallel()

	metricRepo := &fakeMetricRepo{}
	service := NewMetricIngestionService(metricRepo, &fakeMetricServiceRepo{}, nil)
	reportedNodeID := int64(2)

	err := service.Ingest(context.Background(), types.MetricIngestionInput{
		AuthenticatedNodeID: 1,
		ReportedNodeID:      &reportedNodeID,
		Samples: []types.MetricSampleInput{
			{
				MetricName:  "cpu_usage",
				MetricValue: 42,
				ObservedAt:  time.Now().UTC(),
			},
		},
	})
	if !errors.Is(err, types.ErrAgentNodeMismatch) {
		t.Fatalf("expected ErrAgentNodeMismatch, got %v", err)
	}

	if metricRepo.createCalls != 0 {
		t.Fatalf("expected no metric writes, got %d", metricRepo.createCalls)
	}
}
