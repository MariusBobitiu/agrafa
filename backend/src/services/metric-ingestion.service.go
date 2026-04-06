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

type metricAlertEvaluator interface {
	EvaluateMetricRules(ctx context.Context, nodeID int64, metricName string, occurredAt time.Time) error
}

type MetricIngestionService struct {
	metricRepo     metricIngestionMetricRepository
	serviceRepo    metricIngestionServiceRepository
	alertEvaluator metricAlertEvaluator
}

type metricIngestionMetricRepository interface {
	Create(ctx context.Context, params generated.CreateMetricSampleParams) (generated.MetricSample, error)
}

type metricIngestionServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
}

func NewMetricIngestionService(
	metricRepo metricIngestionMetricRepository,
	serviceRepo metricIngestionServiceRepository,
	alertEvaluator metricAlertEvaluator,
) *MetricIngestionService {
	return &MetricIngestionService{
		metricRepo:     metricRepo,
		serviceRepo:    serviceRepo,
		alertEvaluator: alertEvaluator,
	}
}

func (s *MetricIngestionService) Ingest(ctx context.Context, input types.MetricIngestionInput) error {
	if input.ReportedNodeID != nil && *input.ReportedNodeID != input.AuthenticatedNodeID {
		return types.ErrAgentNodeMismatch
	}

	if input.ServiceID != nil {
		service, err := s.serviceRepo.GetByID(ctx, *input.ServiceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return types.ErrServiceNotFound
			}

			return fmt.Errorf("get service: %w", err)
		}

		if service.NodeID != input.AuthenticatedNodeID {
			return types.ErrServiceNodeMismatch
		}
	}

	for _, sample := range input.Samples {
		if _, err := s.metricRepo.Create(ctx, generated.CreateMetricSampleParams{
			NodeID:      input.AuthenticatedNodeID,
			ServiceID:   utils.ToNullInt64(input.ServiceID),
			MetricName:  sample.MetricName,
			MetricValue: sample.MetricValue,
			MetricUnit:  sample.MetricUnit,
			ObservedAt:  sample.ObservedAt,
			Payload:     utils.NormalizeJSON(sample.Payload),
		}); err != nil {
			return fmt.Errorf("create metric sample: %w", err)
		}

		if s.alertEvaluator != nil {
			if err := s.alertEvaluator.EvaluateMetricRules(ctx, input.AuthenticatedNodeID, sample.MetricName, sample.ObservedAt); err != nil {
				return err
			}
		}
	}

	return nil
}
