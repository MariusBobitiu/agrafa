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
)

type nodeReadNodeRepository interface {
	GetByID(ctx context.Context, id int64) (generated.Node, error)
	ListVisible(ctx context.Context) ([]generated.Node, error)
	ListVisibleByProject(ctx context.Context, projectID int64) ([]generated.Node, error)
}

type nodeReadMetricRepository interface {
	GetLatestNodeMetricByName(ctx context.Context, nodeID int64, metricName string) (generated.MetricSample, error)
	ListLatestNodeMetrics(ctx context.Context, projectID *int64) ([]generated.ListLatestOperationalNodeMetricsRow, error)
}

type nodeReadAlertInstanceRepository interface {
	CountActiveByNodeID(ctx context.Context, nodeID int64) (int64, error)
	ListActiveCountsByNode(ctx context.Context, projectID *int64) ([]generated.ListActiveAlertCountsByNodeRow, error)
}

type nodeReadServiceRepository interface {
	CountByNodeID(ctx context.Context, nodeID int64) (int64, error)
	ListCountsByNode(ctx context.Context, projectID *int64) ([]generated.ListServiceCountsByNodeRow, error)
}

type NodeReadService struct {
	nodeRepo          nodeReadNodeRepository
	metricRepo        nodeReadMetricRepository
	alertInstanceRepo nodeReadAlertInstanceRepository
	serviceRepo       nodeReadServiceRepository
}

func NewNodeReadService(
	nodeRepo *repositories.NodeRepository,
	metricRepo *repositories.MetricRepository,
	alertInstanceRepo *repositories.AlertInstanceRepository,
	serviceRepo *repositories.ServiceRepository,
) *NodeReadService {
	return &NodeReadService{
		nodeRepo:          nodeRepo,
		metricRepo:        metricRepo,
		alertInstanceRepo: alertInstanceRepo,
		serviceRepo:       serviceRepo,
	}
}

func (s *NodeReadService) List(ctx context.Context, projectID *int64) ([]types.NodeReadData, error) {
	var (
		nodes []generated.Node
		err   error
	)

	if projectID != nil {
		nodes, err = s.nodeRepo.ListVisibleByProject(ctx, *projectID)
	} else {
		nodes, err = s.nodeRepo.ListVisible(ctx)
	}
	if err != nil {
		return nil, err
	}

	latestMetrics, err := s.metricRepo.ListLatestNodeMetrics(ctx, projectID)
	if err != nil {
		return nil, err
	}

	activeAlertCounts, err := s.alertInstanceRepo.ListActiveCountsByNode(ctx, projectID)
	if err != nil {
		return nil, err
	}

	serviceCounts, err := s.serviceRepo.ListCountsByNode(ctx, projectID)
	if err != nil {
		return nil, err
	}

	metricsByNode := mapLatestMetricsByNode(latestMetrics)
	alertCountsByNode := mapNodeAlertCounts(activeAlertCounts)
	serviceCountsByNode := mapServiceCountsByNode(serviceCounts)

	items := make([]types.NodeReadData, 0, len(nodes))
	for _, node := range nodes {
		metricSet := metricsByNode[node.ID]

		items = append(items, types.NodeReadData{
			ID:               node.ID,
			ProjectID:        node.ProjectID,
			Name:             node.Name,
			Identifier:       node.Identifier,
			CurrentState:     node.CurrentState,
			LastSeenAt:       nullTimePtr(node.LastHeartbeatAt),
			Metadata:         node.Metadata,
			LatestCPU:        metricSet[types.MetricNameCPUUsage],
			LatestMemory:     metricSet[types.MetricNameMemoryUsage],
			LatestDisk:       metricSet[types.MetricNameDiskUsage],
			ActiveAlertCount: alertCountsByNode[node.ID],
			ServiceCount:     serviceCountsByNode[node.ID],
			CreatedAt:        node.CreatedAt,
			UpdatedAt:        node.UpdatedAt,
		})
	}

	return items, nil
}

func (s *NodeReadService) GetByID(ctx context.Context, nodeID int64) (types.NodeReadData, error) {
	if nodeID <= 0 {
		return types.NodeReadData{}, types.ErrInvalidNodeID
	}

	node, err := s.nodeRepo.GetByID(ctx, nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.NodeReadData{}, types.ErrNodeNotFound
		}

		return types.NodeReadData{}, fmt.Errorf("get node: %w", err)
	}

	activeAlertCount, err := s.alertInstanceRepo.CountActiveByNodeID(ctx, nodeID)
	if err != nil {
		return types.NodeReadData{}, fmt.Errorf("count active node alerts: %w", err)
	}

	serviceCount, err := s.serviceRepo.CountByNodeID(ctx, nodeID)
	if err != nil {
		return types.NodeReadData{}, fmt.Errorf("count node services: %w", err)
	}

	latestCPU, err := s.getLatestMetric(ctx, nodeID, types.MetricNameCPUUsage)
	if err != nil {
		return types.NodeReadData{}, err
	}

	latestMemory, err := s.getLatestMetric(ctx, nodeID, types.MetricNameMemoryUsage)
	if err != nil {
		return types.NodeReadData{}, err
	}

	latestDisk, err := s.getLatestMetric(ctx, nodeID, types.MetricNameDiskUsage)
	if err != nil {
		return types.NodeReadData{}, err
	}

	return types.NodeReadData{
		ID:               node.ID,
		ProjectID:        node.ProjectID,
		Name:             node.Name,
		Identifier:       node.Identifier,
		CurrentState:     node.CurrentState,
		LastSeenAt:       nullTimePtr(node.LastHeartbeatAt),
		Metadata:         node.Metadata,
		LatestCPU:        latestCPU,
		LatestMemory:     latestMemory,
		LatestDisk:       latestDisk,
		ActiveAlertCount: activeAlertCount,
		ServiceCount:     serviceCount,
		CreatedAt:        node.CreatedAt,
		UpdatedAt:        node.UpdatedAt,
	}, nil
}

func (s *NodeReadService) getLatestMetric(ctx context.Context, nodeID int64, metricName string) (*types.MetricValueData, error) {
	metric, err := s.metricRepo.GetLatestNodeMetricByName(ctx, nodeID, metricName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, fmt.Errorf("get latest %s metric: %w", metricName, err)
	}

	return &types.MetricValueData{
		Value:      metric.MetricValue,
		Unit:       metric.MetricUnit,
		ObservedAt: metric.ObservedAt,
	}, nil
}

func mapLatestMetricsByNode(rows []generated.ListLatestOperationalNodeMetricsRow) map[int64]map[string]*types.MetricValueData {
	result := make(map[int64]map[string]*types.MetricValueData, len(rows))

	for _, row := range rows {
		if _, ok := result[row.NodeID]; !ok {
			result[row.NodeID] = map[string]*types.MetricValueData{}
		}

		result[row.NodeID][row.MetricName] = &types.MetricValueData{
			Value:      row.MetricValue,
			Unit:       row.MetricUnit,
			ObservedAt: row.ObservedAt,
		}
	}

	return result
}

func mapNodeAlertCounts(rows []generated.ListActiveAlertCountsByNodeRow) map[int64]int64 {
	result := make(map[int64]int64, len(rows))
	for _, row := range rows {
		if row.NodeID.Valid {
			result[row.NodeID.Int64] = row.ActiveAlertCount
		}
	}

	return result
}

func mapServiceCountsByNode(rows []generated.ListServiceCountsByNodeRow) map[int64]int64 {
	result := make(map[int64]int64, len(rows))
	for _, row := range rows {
		result[row.NodeID] = row.ServiceCount
	}

	return result
}

func nullTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}

	timestamp := value.Time
	return &timestamp
}
