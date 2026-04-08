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

type alertRuleServiceAlertRuleRepository interface {
	Create(ctx context.Context, params generated.CreateAlertRuleParams) (generated.AlertRule, error)
	GetByID(ctx context.Context, id int64) (generated.AlertRule, error)
	UpdateEnabled(ctx context.Context, id int64, isEnabled bool) (generated.AlertRule, error)
	List(ctx context.Context, projectID *int64) ([]generated.AlertRule, error)
	Delete(ctx context.Context, id int64) (int64, error)
}

type alertRuleServiceProjectRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Project, error)
}

type alertRuleServiceNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
}

type alertRuleServiceServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
}

type AlertRuleService struct {
	alertRuleRepo alertRuleServiceAlertRuleRepository
	projectRepo   alertRuleServiceProjectRepository
	nodeRepo      alertRuleServiceNodeRepository
	serviceRepo   alertRuleServiceServiceRepository
}

func NewAlertRuleService(
	alertRuleRepo *repositories.AlertRuleRepository,
	projectRepo *repositories.ProjectRepository,
	nodeRepo *repositories.NodeRepository,
	serviceRepo *repositories.ServiceRepository,
) *AlertRuleService {
	return &AlertRuleService{
		alertRuleRepo: alertRuleRepo,
		projectRepo:   projectRepo,
		nodeRepo:      nodeRepo,
		serviceRepo:   serviceRepo,
	}
}

func (s *AlertRuleService) Create(ctx context.Context, input types.CreateAlertRuleInput) (types.AlertRuleReadData, error) {
	if input.ProjectID <= 0 {
		return types.AlertRuleReadData{}, types.ErrInvalidProjectID
	}

	ruleType := utils.NormalizeRequiredString(input.RuleType)
	if ruleType == "" {
		return types.AlertRuleReadData{}, types.ErrInvalidAlertRuleType
	}

	if !isSupportedAlertRuleType(ruleType) {
		return types.AlertRuleReadData{}, types.ErrUnsupportedAlertRuleType
	}

	severity := normalizeAlertSeverity(input.Severity)
	if severity == "" {
		return types.AlertRuleReadData{}, types.ErrMissingAlertSeverity
	}

	if !isSupportedAlertSeverity(severity) {
		return types.AlertRuleReadData{}, types.ErrInvalidAlertSeverity
	}

	if _, err := s.projectRepo.GetByID(ctx, input.ProjectID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.AlertRuleReadData{}, types.ErrProjectNotFound
		}

		return types.AlertRuleReadData{}, fmt.Errorf("get project: %w", err)
	}

	var (
		nodeID         sql.NullInt64
		serviceID      sql.NullInt64
		metricName     sql.NullString
		thresholdValue sql.NullFloat64
	)

	switch ruleType {
	case types.AlertRuleTypeNodeOffline:
		if input.NodeID == nil || *input.NodeID <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidNodeID
		}

		node, err := s.nodeRepo.GetByID(ctx, *input.NodeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return types.AlertRuleReadData{}, types.ErrNodeNotFound
			}

			return types.AlertRuleReadData{}, fmt.Errorf("get node: %w", err)
		}

		if node.ProjectID != input.ProjectID {
			return types.AlertRuleReadData{}, types.ErrNodeProjectMismatch
		}

		nodeID = sql.NullInt64{Int64: node.ID, Valid: true}
	case types.AlertRuleTypeServiceUnhealthy:
		if input.ServiceID == nil || *input.ServiceID <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidServiceID
		}

		service, err := s.serviceRepo.GetByID(ctx, *input.ServiceID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return types.AlertRuleReadData{}, types.ErrServiceNotFound
			}

			return types.AlertRuleReadData{}, fmt.Errorf("get service: %w", err)
		}

		if service.ProjectID != input.ProjectID {
			return types.AlertRuleReadData{}, types.ErrServiceProjectMismatch
		}

		serviceID = sql.NullInt64{Int64: service.ID, Valid: true}
	default:
		if input.NodeID == nil || *input.NodeID <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidNodeID
		}

		if input.ThresholdValue == nil || *input.ThresholdValue <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidThresholdValue
		}

		node, err := s.nodeRepo.GetByID(ctx, *input.NodeID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return types.AlertRuleReadData{}, types.ErrNodeNotFound
			}

			return types.AlertRuleReadData{}, fmt.Errorf("get node: %w", err)
		}

		if node.ProjectID != input.ProjectID {
			return types.AlertRuleReadData{}, types.ErrNodeProjectMismatch
		}

		metricName = sql.NullString{String: metricNameForRuleType(ruleType), Valid: true}
		nodeID = sql.NullInt64{Int64: node.ID, Valid: true}
		thresholdValue = sql.NullFloat64{Float64: *input.ThresholdValue, Valid: true}
	}

	rule, err := s.alertRuleRepo.Create(ctx, generated.CreateAlertRuleParams{
		ProjectID:      input.ProjectID,
		NodeID:         nodeID,
		ServiceID:      serviceID,
		RuleType:       ruleType,
		Severity:       severity,
		MetricName:     metricName,
		ThresholdValue: thresholdValue,
		IsEnabled:      true,
	})
	if err != nil {
		return types.AlertRuleReadData{}, fmt.Errorf("create alert rule: %w", err)
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) List(ctx context.Context, projectID *int64) ([]types.AlertRuleReadData, error) {
	rules, err := s.alertRuleRepo.List(ctx, projectID)
	if err != nil {
		return nil, err
	}

	return mapAlertRules(rules), nil
}

func (s *AlertRuleService) GetByID(ctx context.Context, alertRuleID int64) (types.AlertRuleReadData, error) {
	if alertRuleID <= 0 {
		return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
	}

	rule, err := s.alertRuleRepo.GetByID(ctx, alertRuleID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
		}

		return types.AlertRuleReadData{}, fmt.Errorf("get alert rule: %w", err)
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) SetEnabled(ctx context.Context, input types.UpdateAlertRuleInput) (types.AlertRuleReadData, error) {
	rule, err := s.alertRuleRepo.UpdateEnabled(ctx, input.ID, input.IsEnabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
		}

		return types.AlertRuleReadData{}, fmt.Errorf("update alert rule enabled state: %w", err)
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) Delete(ctx context.Context, alertRuleID int64) error {
	if alertRuleID <= 0 {
		return types.ErrAlertRuleNotFound
	}

	rowsDeleted, err := s.alertRuleRepo.Delete(ctx, alertRuleID)
	if err != nil {
		return fmt.Errorf("delete alert rule: %w", err)
	}
	if rowsDeleted == 0 {
		return types.ErrAlertRuleNotFound
	}

	return nil
}

func isSupportedAlertRuleType(ruleType string) bool {
	switch ruleType {
	case types.AlertRuleTypeNodeOffline,
		types.AlertRuleTypeServiceUnhealthy,
		types.AlertRuleTypeCPUAboveThreshold,
		types.AlertRuleTypeMemoryAboveThreshold,
		types.AlertRuleTypeDiskAboveThreshold:
		return true
	default:
		return false
	}
}

func metricNameForRuleType(ruleType string) string {
	switch ruleType {
	case types.AlertRuleTypeCPUAboveThreshold:
		return types.MetricNameCPUUsage
	case types.AlertRuleTypeMemoryAboveThreshold:
		return types.MetricNameMemoryUsage
	case types.AlertRuleTypeDiskAboveThreshold:
		return types.MetricNameDiskUsage
	default:
		return ""
	}
}
