package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type serviceReadRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Service, error)
	ListForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.Service, error)
}

type serviceReadNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
	List(ctx context.Context) ([]generated.Node, error)
	ListByProject(ctx context.Context, projectID int64) ([]generated.Node, error)
}

type serviceReadHealthCheckRepository interface {
	GetLatestByServiceID(ctx context.Context, serviceID int64) (generated.HealthCheckResult, error)
	ListLatestForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.HealthCheckResult, error)
}

type serviceReadAlertInstanceRepository interface {
	CountActiveByServiceID(ctx context.Context, serviceID int64) (int64, error)
	ListActiveDetailsByServiceID(ctx context.Context, serviceID int64) ([]generated.ListActiveAlertDetailsByServiceIDRow, error)
	ListActiveCountsByServiceForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.ListActiveAlertCountsByServiceRow, error)
}

type ServiceReadService struct {
	serviceRepo       serviceReadRepository
	nodeRepo          serviceReadNodeRepository
	healthCheckRepo   serviceReadHealthCheckRepository
	alertInstanceRepo serviceReadAlertInstanceRepository
}

func NewServiceReadService(
	serviceRepo serviceReadRepository,
	nodeRepo serviceReadNodeRepository,
	healthCheckRepo serviceReadHealthCheckRepository,
	alertInstanceRepo serviceReadAlertInstanceRepository,
) *ServiceReadService {
	return &ServiceReadService{
		serviceRepo:       serviceRepo,
		nodeRepo:          nodeRepo,
		healthCheckRepo:   healthCheckRepo,
		alertInstanceRepo: alertInstanceRepo,
	}
}

func (s *ServiceReadService) List(ctx context.Context, filters types.ServiceListFilters) ([]types.ServiceReadData, error) {
	services, err := s.serviceRepo.ListForRead(ctx, filters)
	if err != nil {
		return nil, err
	}

	nodeExecutionModes, err := s.listNodeExecutionModes(ctx, filters.ProjectID)
	if err != nil {
		return nil, err
	}

	latestHealthChecks, err := s.healthCheckRepo.ListLatestForRead(ctx, filters)
	if err != nil {
		return nil, err
	}

	activeAlertCounts, err := s.alertInstanceRepo.ListActiveCountsByServiceForRead(ctx, filters)
	if err != nil {
		return nil, err
	}

	latestHealthByService := mapLatestHealthChecksByService(latestHealthChecks)
	alertCountsByService := mapServiceAlertCounts(activeAlertCounts)

	items := make([]types.ServiceReadData, 0, len(services))
	for _, service := range services {
		executionMode, ok := nodeExecutionModes[service.NodeID]
		if !ok {
			return nil, fmt.Errorf("missing node execution mode for node %d", service.NodeID)
		}

		items = append(items, types.ServiceReadData{
			ID:                  service.ID,
			ProjectID:           service.ProjectID,
			NodeID:              service.NodeID,
			ExecutionMode:       executionMode,
			Name:                service.Name,
			CheckType:           service.CheckType,
			CheckTarget:         service.CheckTarget,
			Status:              service.CurrentState,
			LastCheckedAt:       nullTimePtr(service.LastCheckAt),
			ConsecutiveFailures: service.ConsecutiveFailures,
			ActiveAlertCount:    alertCountsByService[service.ID],
			LatestHealthCheck:   latestHealthByService[service.ID],
		})
	}

	return items, nil
}

func (s *ServiceReadService) GetByID(ctx context.Context, serviceID int64) (types.ServiceDetailData, error) {
	if serviceID <= 0 {
		return types.ServiceDetailData{}, types.ErrInvalidServiceID
	}

	service, err := s.serviceRepo.GetByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ServiceDetailData{}, types.ErrServiceNotFound
		}

		return types.ServiceDetailData{}, fmt.Errorf("get service: %w", err)
	}

	node, err := s.nodeRepo.GetByID(ctx, service.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ServiceDetailData{}, types.ErrNodeNotFound
		}

		return types.ServiceDetailData{}, fmt.Errorf("get service node: %w", err)
	}

	activeAlertCount, err := s.alertInstanceRepo.CountActiveByServiceID(ctx, serviceID)
	if err != nil {
		return types.ServiceDetailData{}, fmt.Errorf("count active service alerts: %w", err)
	}

	activeAlerts, err := s.alertInstanceRepo.ListActiveDetailsByServiceID(ctx, serviceID)
	if err != nil {
		return types.ServiceDetailData{}, fmt.Errorf("list active service alerts: %w", err)
	}

	var latestHealthCheck *types.HealthCheckSummaryData
	healthCheck, err := s.healthCheckRepo.GetLatestByServiceID(ctx, serviceID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return types.ServiceDetailData{}, fmt.Errorf("get latest health check: %w", err)
		}
	} else {
		latestHealthCheck = &types.HealthCheckSummaryData{
			ObservedAt:     healthCheck.ObservedAt,
			IsSuccess:      healthCheck.IsSuccess,
			StatusCode:     nullInt32Ptr(healthCheck.StatusCode),
			ResponseTimeMs: nullInt32Ptr(healthCheck.ResponseTimeMs),
			Message:        healthCheck.Message,
		}
	}

	return types.ServiceDetailData{
		ID:                  service.ID,
		ProjectID:           service.ProjectID,
		NodeID:              service.NodeID,
		ExecutionMode:       executionModeFromNodeType(node.NodeType),
		Name:                service.Name,
		CheckType:           service.CheckType,
		CheckTarget:         service.CheckTarget,
		Status:              service.CurrentState,
		LastCheckedAt:       nullTimePtr(service.LastCheckAt),
		ConsecutiveFailures: service.ConsecutiveFailures,
		ActiveAlertCount:    activeAlertCount,
		ActiveAlerts:        mapServiceActiveAlerts(activeAlerts),
		LatestHealthCheck:   latestHealthCheck,
	}, nil
}

func (s *ServiceReadService) listNodeExecutionModes(ctx context.Context, projectID *int64) (map[int64]string, error) {
	var (
		nodes []generated.Node
		err   error
	)

	if projectID != nil {
		nodes, err = s.nodeRepo.ListByProject(ctx, *projectID)
	} else {
		nodes, err = s.nodeRepo.List(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("list nodes for execution modes: %w", err)
	}

	modes := make(map[int64]string, len(nodes))
	for _, node := range nodes {
		modes[node.ID] = executionModeFromNodeType(node.NodeType)
	}

	return modes, nil
}

func executionModeFromNodeType(nodeType string) string {
	if nodeType == types.NodeTypeManaged {
		return types.ExecutionModeManaged
	}

	return types.ExecutionModeAgent
}

func mapLatestHealthChecksByService(rows []generated.HealthCheckResult) map[int64]*types.HealthCheckSummaryData {
	result := make(map[int64]*types.HealthCheckSummaryData, len(rows))
	for _, row := range rows {
		result[row.ServiceID] = &types.HealthCheckSummaryData{
			ObservedAt:     row.ObservedAt,
			IsSuccess:      row.IsSuccess,
			StatusCode:     nullInt32Ptr(row.StatusCode),
			ResponseTimeMs: nullInt32Ptr(row.ResponseTimeMs),
			Message:        row.Message,
		}
	}

	return result
}

func mapServiceAlertCounts(rows []generated.ListActiveAlertCountsByServiceRow) map[int64]int64 {
	result := make(map[int64]int64, len(rows))
	for _, row := range rows {
		if row.ServiceID.Valid {
			result[row.ServiceID.Int64] = row.ActiveAlertCount
		}
	}

	return result
}

func mapServiceActiveAlerts(rows []generated.ListActiveAlertDetailsByServiceIDRow) []types.ServiceActiveAlertData {
	items := make([]types.ServiceActiveAlertData, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.ServiceActiveAlertData{
			ID:          row.ID,
			RuleID:      row.RuleID,
			RuleType:    row.RuleType,
			Severity:    row.Severity,
			Title:       row.Title,
			Status:      row.Status,
			TriggeredAt: row.TriggeredAt,
		})
	}

	return items
}

func nullInt32Ptr(value sql.NullInt32) *int32 {
	if !value.Valid {
		return nil
	}

	number := value.Int32
	return &number
}
