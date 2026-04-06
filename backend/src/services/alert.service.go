package services

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type AlertService struct {
	alertInstanceRepo *repositories.AlertInstanceRepository
}

func NewAlertService(alertInstanceRepo *repositories.AlertInstanceRepository) *AlertService {
	return &AlertService{alertInstanceRepo: alertInstanceRepo}
}

func (s *AlertService) List(ctx context.Context, projectID *int64, status *string, limit int32) ([]types.AlertReadData, error) {
	if limit <= 0 {
		limit = 50
	}

	if status != nil && *status != types.AlertStatusActive && *status != types.AlertStatusResolved {
		return nil, types.ErrInvalidAlertStatus
	}

	alerts, err := s.alertInstanceRepo.List(ctx, projectID, status, limit)
	if err != nil {
		return nil, err
	}

	return mapAlerts(alerts), nil
}
