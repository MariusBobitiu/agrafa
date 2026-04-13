package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

type alertRuleServiceAlertRuleRepository interface {
	Create(ctx context.Context, params generated.CreateAlertRuleParams) (generated.AlertRule, error)
	GetByID(ctx context.Context, id int64) (generated.AlertRule, error)
	Update(ctx context.Context, params generated.UpdateAlertRuleParams) (generated.AlertRule, error)
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

type alertRuleServiceEvaluator interface {
	EvaluateNodeRules(ctx context.Context, node generated.Node, occurredAt time.Time) error
	EvaluateServiceRules(ctx context.Context, service generated.Service, occurredAt time.Time) error
	EvaluateMetricRules(ctx context.Context, nodeID int64, metricName string, occurredAt time.Time) error
}

type AlertRuleService struct {
	alertRuleRepo alertRuleServiceAlertRuleRepository
	projectRepo   alertRuleServiceProjectRepository
	nodeRepo      alertRuleServiceNodeRepository
	serviceRepo   alertRuleServiceServiceRepository
	evaluator     alertRuleServiceEvaluator
}

func NewAlertRuleService(
	alertRuleRepo *repositories.AlertRuleRepository,
	projectRepo *repositories.ProjectRepository,
	nodeRepo *repositories.NodeRepository,
	serviceRepo *repositories.ServiceRepository,
	evaluator alertRuleServiceEvaluator,
) *AlertRuleService {
	return &AlertRuleService{
		alertRuleRepo: alertRuleRepo,
		projectRepo:   projectRepo,
		nodeRepo:      nodeRepo,
		serviceRepo:   serviceRepo,
		evaluator:     evaluator,
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
		node           generated.Node
		service        generated.Service
		err            error
	)

	switch ruleType {
	case types.AlertRuleTypeNodeOffline:
		if input.NodeID == nil || *input.NodeID <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidNodeID
		}

		node, err = s.nodeRepo.GetByID(ctx, *input.NodeID)
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

		service, err = s.serviceRepo.GetByID(ctx, *input.ServiceID)
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

		node, err = s.nodeRepo.GetByID(ctx, *input.NodeID)
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

	if err := s.evaluateCurrentState(ctx, rule, node, service); err != nil {
		return types.AlertRuleReadData{}, err
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) evaluateCurrentState(
	ctx context.Context,
	rule generated.AlertRule,
	node generated.Node,
	service generated.Service,
) error {
	if s.evaluator == nil {
		return nil
	}

	switch rule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		if err := s.evaluator.EvaluateNodeRules(ctx, node, rule.CreatedAt); err != nil {
			return fmt.Errorf("evaluate current node state: %w", err)
		}
	case types.AlertRuleTypeServiceUnhealthy:
		if err := s.evaluator.EvaluateServiceRules(ctx, service, rule.CreatedAt); err != nil {
			return fmt.Errorf("evaluate current service state: %w", err)
		}
	default:
		metricName := metricNameForRuleType(rule.RuleType)
		if metricName == "" {
			return nil
		}

		if err := s.evaluator.EvaluateMetricRules(ctx, node.ID, metricName, rule.CreatedAt); err != nil {
			return fmt.Errorf("evaluate current metric state: %w", err)
		}
	}

	return nil
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

func (s *AlertRuleService) Update(ctx context.Context, input types.UpdateAlertRuleInput) (types.AlertRuleReadData, error) {
	if input.ID <= 0 {
		return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
	}

	currentRule, err := s.alertRuleRepo.GetByID(ctx, input.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
		}

		return types.AlertRuleReadData{}, fmt.Errorf("get alert rule: %w", err)
	}

	nextRule, node, service, err := s.buildUpdatedAlertRule(ctx, currentRule, input)
	if err != nil {
		return types.AlertRuleReadData{}, err
	}

	rule, err := s.alertRuleRepo.Update(ctx, nextRule)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.AlertRuleReadData{}, types.ErrAlertRuleNotFound
		}

		return types.AlertRuleReadData{}, fmt.Errorf("update alert rule: %w", err)
	}

	if rule.IsEnabled {
		if err := s.evaluateCurrentState(ctx, rule, node, service); err != nil {
			return types.AlertRuleReadData{}, err
		}
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) buildUpdatedAlertRule(
	ctx context.Context,
	currentRule generated.AlertRule,
	input types.UpdateAlertRuleInput,
) (generated.UpdateAlertRuleParams, generated.Node, generated.Service, error) {
	params := generated.UpdateAlertRuleParams{ID: currentRule.ID}
	updatedNodeID := currentRule.NodeID
	updatedServiceID := currentRule.ServiceID
	node := generated.Node{}
	service := generated.Service{}

	if input.Severity != nil {
		severity := normalizeAlertSeverity(*input.Severity)
		if severity == "" {
			return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrMissingAlertSeverity
		}

		if !isSupportedAlertSeverity(severity) {
			return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrInvalidAlertSeverity
		}

		params.Column6 = true
		params.Severity = severity
	}

	if input.IsEnabled != nil {
		params.Column10 = true
		params.IsEnabled = *input.IsEnabled
	}

	switch currentRule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		if input.NodeID != nil {
			if *input.NodeID <= 0 {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrInvalidNodeID
			}

			var err error
			node, err = s.nodeRepo.GetByID(ctx, *input.NodeID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrNodeNotFound
				}

				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get node: %w", err)
			}

			if node.ProjectID != currentRule.ProjectID {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrNodeProjectMismatch
			}

			updatedNodeID = sql.NullInt64{Int64: node.ID, Valid: true}
			params.Column2 = true
			params.NodeID = updatedNodeID
		} else if currentRule.NodeID.Valid {
			var err error
			node, err = s.nodeRepo.GetByID(ctx, currentRule.NodeID.Int64)
			if err != nil {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get node: %w", err)
			}
		}
	case types.AlertRuleTypeServiceUnhealthy:
		if input.ServiceID != nil {
			if *input.ServiceID <= 0 {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrInvalidServiceID
			}

			var err error
			service, err = s.serviceRepo.GetByID(ctx, *input.ServiceID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrServiceNotFound
				}

				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get service: %w", err)
			}

			if service.ProjectID != currentRule.ProjectID {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrServiceProjectMismatch
			}

			updatedServiceID = sql.NullInt64{Int64: service.ID, Valid: true}
			params.Column4 = true
			params.ServiceID = updatedServiceID
		} else if currentRule.ServiceID.Valid {
			var err error
			service, err = s.serviceRepo.GetByID(ctx, currentRule.ServiceID.Int64)
			if err != nil {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get service: %w", err)
			}
		}
	default:
		if input.NodeID != nil {
			if *input.NodeID <= 0 {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrInvalidNodeID
			}

			var err error
			node, err = s.nodeRepo.GetByID(ctx, *input.NodeID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrNodeNotFound
				}

				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get node: %w", err)
			}

			if node.ProjectID != currentRule.ProjectID {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrNodeProjectMismatch
			}

			updatedNodeID = sql.NullInt64{Int64: node.ID, Valid: true}
			params.Column2 = true
			params.NodeID = updatedNodeID
		} else if currentRule.NodeID.Valid {
			var err error
			node, err = s.nodeRepo.GetByID(ctx, currentRule.NodeID.Int64)
			if err != nil {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, fmt.Errorf("get node: %w", err)
			}
		}

		if input.ThresholdValue != nil {
			if *input.ThresholdValue <= 0 {
				return generated.UpdateAlertRuleParams{}, generated.Node{}, generated.Service{}, types.ErrInvalidThresholdValue
			}

			params.Column8 = true
			params.ThresholdValue = sql.NullFloat64{Float64: *input.ThresholdValue, Valid: true}
		}
	}

	_ = updatedNodeID
	_ = updatedServiceID

	return params, node, service, nil
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
