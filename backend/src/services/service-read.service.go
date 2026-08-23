package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"time"

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
	ListByServiceIDAfterID(ctx context.Context, serviceID int64, afterID int64) ([]generated.HealthCheckResult, error)
	ListLatestForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.HealthCheckResult, error)
	ListHistoryByServiceID(ctx context.Context, serviceID int64, filters types.ServiceHistoryFilters) ([]generated.HealthCheckResult, error)
	SummarizeHistoryByServiceID(ctx context.Context, serviceID int64, historyRange types.ServiceHistoryRange) (generated.SummarizeHealthCheckHistoryByServiceIDRow, error)
}

func (s *ServiceReadService) SummarizeHistory(ctx context.Context, serviceID int64, historyRange types.ServiceHistoryRange) (types.ServiceHistorySummaryData, error) {
	if serviceID <= 0 {
		return types.ServiceHistorySummaryData{}, types.ErrInvalidServiceID
	}

	row, err := s.healthCheckRepo.SummarizeHistoryByServiceID(ctx, serviceID, historyRange)
	if err != nil {
		return types.ServiceHistorySummaryData{}, fmt.Errorf("summarize service history: %w", err)
	}

	result := types.ServiceHistorySummaryData{
		From:             historyRange.From.UTC(),
		To:               historyRange.To.UTC(),
		TotalChecks:      row.TotalChecks,
		SuccessfulChecks: row.SuccessfulChecks,
	}
	if row.TotalChecks > 0 {
		uptime := 100 * float64(row.SuccessfulChecks) / float64(row.TotalChecks)
		result.UptimePercent = &uptime
		lastCheckedAt := time.Unix(0, row.LastCheckedUnixNano).UTC()
		result.LastCheckedAt = &lastCheckedAt
	}
	if row.MeasuredLatencyChecks > 0 {
		averageLatency := float64(row.TotalLatencyMs) / float64(row.MeasuredLatencyChecks)
		result.AverageLatencyMs = &averageLatency
	}

	return result, nil
}

func (s *ServiceReadService) ListStreamObservations(ctx context.Context, serviceID int64, afterID int64) ([]types.ServiceHistoryEntryData, error) {
	if serviceID <= 0 {
		return nil, types.ErrInvalidServiceID
	}

	rows, err := s.healthCheckRepo.ListByServiceIDAfterID(ctx, serviceID, afterID)
	if err != nil {
		return nil, fmt.Errorf("list streamed service observations: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	observations := make([]types.ServiceHistoryEntryData, 0, len(rows))
	for _, row := range rows {
		observations = append(observations, mapServiceHistoryEntry(row))
	}

	return observations, nil
}

const (
	DefaultServiceHistoryLimit int32 = 100
	MaxServiceHistoryLimit     int32 = 500
)

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
			CreatedAt:           service.CreatedAt,
			UpdatedAt:           service.UpdatedAt,
		})
	}

	return items, nil
}

func (s *ServiceReadService) GetByID(ctx context.Context, serviceID int64) (types.ServiceDetailData, error) {
	snapshot, err := s.GetStreamSnapshot(ctx, serviceID)
	if err != nil {
		return types.ServiceDetailData{}, err
	}

	return snapshot.Service, nil
}

func (s *ServiceReadService) GetStreamSnapshot(ctx context.Context, serviceID int64) (types.ServiceStreamData, error) {
	if serviceID <= 0 {
		return types.ServiceStreamData{}, types.ErrInvalidServiceID
	}

	service, err := s.serviceRepo.GetByID(ctx, serviceID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ServiceStreamData{}, types.ErrServiceNotFound
		}

		return types.ServiceStreamData{}, fmt.Errorf("get service: %w", err)
	}

	node, err := s.nodeRepo.GetByID(ctx, service.NodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ServiceStreamData{}, types.ErrNodeNotFound
		}

		return types.ServiceStreamData{}, fmt.Errorf("get service node: %w", err)
	}

	activeAlertCount, err := s.alertInstanceRepo.CountActiveByServiceID(ctx, serviceID)
	if err != nil {
		return types.ServiceStreamData{}, fmt.Errorf("count active service alerts: %w", err)
	}

	activeAlerts, err := s.alertInstanceRepo.ListActiveDetailsByServiceID(ctx, serviceID)
	if err != nil {
		return types.ServiceStreamData{}, fmt.Errorf("list active service alerts: %w", err)
	}

	var latestHealthCheck *types.HealthCheckSummaryData
	var latestObservation *types.ServiceHistoryEntryData
	healthCheck, err := s.healthCheckRepo.GetLatestByServiceID(ctx, serviceID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return types.ServiceStreamData{}, fmt.Errorf("get latest health check: %w", err)
		}
	} else {
		observation := mapServiceHistoryEntry(healthCheck)
		latestObservation = &observation
		latestHealthCheck = &types.HealthCheckSummaryData{
			ObservedAt:     observation.ObservedAt,
			IsSuccess:      observation.IsSuccess,
			StatusCode:     observation.StatusCode,
			ResponseTimeMs: observation.ResponseTimeMs,
			Message:        observation.Message,
		}
	}

	return types.ServiceStreamData{
		Service: types.ServiceDetailData{
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
			CreatedAt:           service.CreatedAt,
			UpdatedAt:           service.UpdatedAt,
		},
		Observation: latestObservation,
	}, nil
}

func (s *ServiceReadService) ListHistory(ctx context.Context, serviceID int64, filters types.ServiceHistoryFilters) (types.ServiceHistoryPageData, error) {
	if serviceID <= 0 {
		return types.ServiceHistoryPageData{}, types.ErrInvalidServiceID
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = DefaultServiceHistoryLimit
	}
	if limit > MaxServiceHistoryLimit {
		limit = MaxServiceHistoryLimit
	}

	queryFilters := filters
	queryFilters.Limit = limit + 1
	rows, err := s.healthCheckRepo.ListHistoryByServiceID(ctx, serviceID, queryFilters)
	if err != nil {
		return types.ServiceHistoryPageData{}, fmt.Errorf("list service history: %w", err)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].ObservedAt.Equal(rows[j].ObservedAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].ObservedAt.After(rows[j].ObservedAt)
	})

	hasMore := len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	entries := make([]types.ServiceHistoryEntryData, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, mapServiceHistoryEntry(row))
	}

	page := types.ServiceHistoryPageData{Entries: entries}
	if hasMore && len(entries) > 0 {
		last := entries[len(entries)-1]
		page.NextCursor = &types.ServiceHistoryCursor{ObservedAt: last.ObservedAt, ID: last.ID}
	}

	return page, nil
}

func mapServiceHistoryEntry(row generated.HealthCheckResult) types.ServiceHistoryEntryData {
	return types.ServiceHistoryEntryData{
		ID:             row.ID,
		ServiceID:      row.ServiceID,
		NodeID:         row.NodeID,
		CheckType:      row.CheckType,
		Source:         row.Source,
		ObservedAt:     row.ObservedAt,
		IsSuccess:      row.IsSuccess,
		StatusCode:     nullInt32Ptr(row.StatusCode),
		ResponseTimeMs: nullInt32Ptr(row.ResponseTimeMs),
		Message:        row.Message,
		Metadata:       normalizeJSONValue(rawJSONValue(row.Payload)),
	}
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
