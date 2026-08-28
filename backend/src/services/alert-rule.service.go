package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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
	ListByProject(ctx context.Context, projectID int64) ([]generated.Node, error)
}

type alertRuleServiceServiceRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
	ListByProject(ctx context.Context, projectID int64) ([]generated.Service, error)
}

type alertRuleServiceEvaluator interface {
	EvaluateNodeRules(ctx context.Context, node generated.Node, occurredAt time.Time) error
	EvaluateServiceRules(ctx context.Context, service generated.Service, occurredAt time.Time) error
	EvaluateMetricRules(ctx context.Context, nodeID int64, metricName string, occurredAt time.Time) error
	ReconcileRule(ctx context.Context, rule generated.AlertRule, occurredAt time.Time) error
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
		targetScope, scopeErr := normalizeAlertRuleTargetScope(input.TargetScope, input.NodeID != nil)
		if scopeErr != nil {
			return types.AlertRuleReadData{}, scopeErr
		}
		if targetScope == types.AlertRuleTargetScopeAll {
			break
		}
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
		targetScope, scopeErr := normalizeAlertRuleTargetScope(input.TargetScope, input.ServiceID != nil)
		if scopeErr != nil {
			return types.AlertRuleReadData{}, scopeErr
		}
		if targetScope == types.AlertRuleTargetScopeAll {
			break
		}
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
		if input.ThresholdValue == nil || *input.ThresholdValue <= 0 {
			return types.AlertRuleReadData{}, types.ErrInvalidThresholdValue
		}
		targetScope, scopeErr := normalizeAlertRuleTargetScope(input.TargetScope, input.NodeID != nil)
		if scopeErr != nil {
			return types.AlertRuleReadData{}, scopeErr
		}

		metricName = sql.NullString{String: metricNameForRuleType(ruleType), Valid: true}
		thresholdValue = sql.NullFloat64{Float64: *input.ThresholdValue, Valid: true}
		if targetScope == types.AlertRuleTargetScopeAll {
			break
		}
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

	if err := s.evaluateCurrentState(ctx, rule); err != nil {
		return types.AlertRuleReadData{}, err
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) evaluateCurrentState(
	ctx context.Context,
	rule generated.AlertRule,
) error {
	if s.evaluator == nil {
		return nil
	}

	switch rule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		nodes, err := s.nodesForRule(ctx, rule)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if err := s.evaluator.EvaluateNodeRules(ctx, node, rule.UpdatedAt); err != nil {
				return fmt.Errorf("evaluate current node state: %w", err)
			}
		}
	case types.AlertRuleTypeServiceUnhealthy:
		services, err := s.servicesForRule(ctx, rule)
		if err != nil {
			return err
		}
		for _, service := range services {
			if err := s.evaluator.EvaluateServiceRules(ctx, service, rule.UpdatedAt); err != nil {
				return fmt.Errorf("evaluate current service state: %w", err)
			}
		}
	default:
		metricName := metricNameForRuleType(rule.RuleType)
		if metricName == "" {
			return nil
		}

		nodes, err := s.nodesForRule(ctx, rule)
		if err != nil {
			return err
		}
		for _, node := range nodes {
			if err := s.evaluator.EvaluateMetricRules(ctx, node.ID, metricName, rule.UpdatedAt); err != nil {
				return fmt.Errorf("evaluate current metric state: %w", err)
			}
		}
	}

	return nil
}

func (s *AlertRuleService) nodesForRule(ctx context.Context, rule generated.AlertRule) ([]generated.Node, error) {
	if !rule.NodeID.Valid {
		nodes, err := s.nodeRepo.ListByProject(ctx, rule.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("list project nodes: %w", err)
		}
		return nodes, nil
	}

	node, err := s.nodeRepo.GetByID(ctx, rule.NodeID.Int64)
	if err != nil {
		return nil, fmt.Errorf("get node: %w", err)
	}
	return []generated.Node{node}, nil
}

func (s *AlertRuleService) servicesForRule(ctx context.Context, rule generated.AlertRule) ([]generated.Service, error) {
	if !rule.ServiceID.Valid {
		services, err := s.serviceRepo.ListByProject(ctx, rule.ProjectID)
		if err != nil {
			return nil, fmt.Errorf("list project services: %w", err)
		}
		return services, nil
	}

	service, err := s.serviceRepo.GetByID(ctx, rule.ServiceID.Int64)
	if err != nil {
		return nil, fmt.Errorf("get service: %w", err)
	}
	return []generated.Service{service}, nil
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

	nextRule, err := s.buildUpdatedAlertRule(ctx, currentRule, input)
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

	if s.evaluator != nil {
		if err := s.evaluator.ReconcileRule(ctx, rule, rule.UpdatedAt); err != nil {
			return types.AlertRuleReadData{}, fmt.Errorf("reconcile alert rule scope: %w", err)
		}
	}

	if rule.IsEnabled {
		if err := s.evaluateCurrentState(ctx, rule); err != nil {
			return types.AlertRuleReadData{}, err
		}
	}

	return mapAlertRule(rule), nil
}

func (s *AlertRuleService) buildUpdatedAlertRule(
	ctx context.Context,
	currentRule generated.AlertRule,
	input types.UpdateAlertRuleInput,
) (generated.UpdateAlertRuleParams, error) {
	params := generated.UpdateAlertRuleParams{ID: currentRule.ID}

	if input.Severity != nil {
		severity := normalizeAlertSeverity(*input.Severity)
		if severity == "" {
			return generated.UpdateAlertRuleParams{}, types.ErrMissingAlertSeverity
		}

		if !isSupportedAlertSeverity(severity) {
			return generated.UpdateAlertRuleParams{}, types.ErrInvalidAlertSeverity
		}

		params.SetSeverity = true
		params.Severity = severity
	}

	if input.IsEnabled != nil {
		params.SetIsEnabled = true
		params.IsEnabled = *input.IsEnabled
	}

	switch currentRule.RuleType {
	case types.AlertRuleTypeNodeOffline:
		nodeID, changed, err := s.updatedNodeTarget(ctx, currentRule, input)
		if err != nil {
			return generated.UpdateAlertRuleParams{}, err
		}
		if changed {
			params.SetNodeID = true
			params.NodeID = nodeID
		}
	case types.AlertRuleTypeServiceUnhealthy:
		serviceID, changed, err := s.updatedServiceTarget(ctx, currentRule, input)
		if err != nil {
			return generated.UpdateAlertRuleParams{}, err
		}
		if changed {
			params.SetServiceID = true
			params.ServiceID = serviceID
		}
	default:
		nodeID, changed, err := s.updatedNodeTarget(ctx, currentRule, input)
		if err != nil {
			return generated.UpdateAlertRuleParams{}, err
		}
		if changed {
			params.SetNodeID = true
			params.NodeID = nodeID
		}

		if input.ThresholdValue != nil {
			if *input.ThresholdValue <= 0 {
				return generated.UpdateAlertRuleParams{}, types.ErrInvalidThresholdValue
			}

			params.SetThresholdValue = true
			params.ThresholdValue = sql.NullFloat64{Float64: *input.ThresholdValue, Valid: true}
		}
	}

	return params, nil
}

func (s *AlertRuleService) updatedNodeTarget(
	ctx context.Context,
	currentRule generated.AlertRule,
	input types.UpdateAlertRuleInput,
) (sql.NullInt64, bool, error) {
	if input.TargetScope == nil && input.NodeID == nil {
		return currentRule.NodeID, false, nil
	}

	if input.TargetScope != nil {
		scope := strings.ToLower(strings.TrimSpace(*input.TargetScope))
		if scope != types.AlertRuleTargetScopeAll && scope != types.AlertRuleTargetScopeSpecific {
			return sql.NullInt64{}, false, types.ErrInvalidAlertRuleTargetScope
		}
		if scope == types.AlertRuleTargetScopeAll {
			if input.NodeID != nil {
				return sql.NullInt64{}, false, types.ErrInvalidAlertRuleTargetScope
			}
			return sql.NullInt64{}, true, nil
		}
	}

	nodeID := input.NodeID
	if nodeID == nil && currentRule.NodeID.Valid {
		nodeID = &currentRule.NodeID.Int64
	}
	if nodeID == nil || *nodeID <= 0 {
		return sql.NullInt64{}, false, types.ErrInvalidNodeID
	}

	node, err := s.nodeRepo.GetByID(ctx, *nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.NullInt64{}, false, types.ErrNodeNotFound
		}
		return sql.NullInt64{}, false, fmt.Errorf("get node: %w", err)
	}
	if node.ProjectID != currentRule.ProjectID {
		return sql.NullInt64{}, false, types.ErrNodeProjectMismatch
	}

	return sql.NullInt64{Int64: node.ID, Valid: true}, true, nil
}

func (s *AlertRuleService) updatedServiceTarget(
	ctx context.Context,
	currentRule generated.AlertRule,
	input types.UpdateAlertRuleInput,
) (sql.NullInt64, bool, error) {
	if input.TargetScope == nil && input.ServiceID == nil {
		return currentRule.ServiceID, false, nil
	}

	if input.TargetScope != nil {
		scope := strings.ToLower(strings.TrimSpace(*input.TargetScope))
		if scope != types.AlertRuleTargetScopeAll && scope != types.AlertRuleTargetScopeSpecific {
			return sql.NullInt64{}, false, types.ErrInvalidAlertRuleTargetScope
		}
		if scope == types.AlertRuleTargetScopeAll {
			if input.ServiceID != nil {
				return sql.NullInt64{}, false, types.ErrInvalidAlertRuleTargetScope
			}
			return sql.NullInt64{}, true, nil
		}
	}

	serviceID := input.ServiceID
	if serviceID == nil && currentRule.ServiceID.Valid {
		serviceID = &currentRule.ServiceID.Int64
	}
	if serviceID == nil || *serviceID <= 0 {
		return sql.NullInt64{}, false, types.ErrInvalidServiceID
	}

	service, err := s.serviceRepo.GetByID(ctx, *serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sql.NullInt64{}, false, types.ErrServiceNotFound
		}
		return sql.NullInt64{}, false, fmt.Errorf("get service: %w", err)
	}
	if service.ProjectID != currentRule.ProjectID {
		return sql.NullInt64{}, false, types.ErrServiceProjectMismatch
	}

	return sql.NullInt64{Int64: service.ID, Valid: true}, true, nil
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

func normalizeAlertRuleTargetScope(value string, hasTarget bool) (string, error) {
	scope := strings.ToLower(strings.TrimSpace(value))
	if scope == "" {
		if hasTarget {
			return types.AlertRuleTargetScopeSpecific, nil
		}
		return types.AlertRuleTargetScopeAll, nil
	}
	if scope != types.AlertRuleTargetScopeAll && scope != types.AlertRuleTargetScopeSpecific {
		return "", types.ErrInvalidAlertRuleTargetScope
	}
	if scope == types.AlertRuleTargetScopeAll && hasTarget {
		return "", types.ErrInvalidAlertRuleTargetScope
	}
	return scope, nil
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
