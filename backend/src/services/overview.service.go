package services

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type OverviewService struct {
	overviewRepo *repositories.OverviewRepository
	eventService *EventService
	nodeRead     *NodeReadService
}

func NewOverviewService(
	overviewRepo *repositories.OverviewRepository,
	eventService *EventService,
	nodeRead *NodeReadService,
) *OverviewService {
	return &OverviewService{
		overviewRepo: overviewRepo,
		eventService: eventService,
		nodeRead:     nodeRead,
	}
}

func (s *OverviewService) Get(ctx context.Context, projectID *int64) (types.OverviewData, error) {
	stats, err := s.getStats(ctx, projectID)
	if err != nil {
		return types.OverviewData{}, err
	}

	recentEvents, err := s.eventService.ListEvents(ctx, 10, projectID)
	if err != nil {
		return types.OverviewData{}, err
	}

	recentAlertEvents, err := s.eventService.ListAlertEvents(ctx, 10, projectID)
	if err != nil {
		return types.OverviewData{}, err
	}

	nodes, err := s.nodeRead.List(ctx, projectID)
	if err != nil {
		return types.OverviewData{}, err
	}

	return types.OverviewData{
		TotalProjects:     stats.TotalProjects,
		TotalNodes:        stats.TotalNodes,
		NodesOnline:       stats.NodesOnline,
		NodesOffline:      stats.NodesOffline,
		TotalServices:     stats.TotalServices,
		ServicesHealthy:   stats.ServicesHealthy,
		ServicesDegraded:  stats.ServicesDegraded,
		ServicesUnhealthy: stats.ServicesUnhealthy,
		ActiveAlerts:      stats.ActiveAlerts,
		ResolvedAlerts:    stats.ResolvedAlerts,
		RecentEvents:      recentEvents,
		RecentAlertEvents: recentAlertEvents,
		NodeSummaries:     mapNodeSummaries(nodes),
	}, nil
}

func (s *OverviewService) getStats(ctx context.Context, projectID *int64) (types.OverviewData, error) {
	if projectID != nil {
		stats, err := s.overviewRepo.GetStatsByProject(ctx, *projectID)
		if err != nil {
			return types.OverviewData{}, err
		}

		return types.OverviewData{
			TotalProjects:     stats.TotalProjects,
			TotalNodes:        stats.TotalNodes,
			NodesOnline:       stats.NodesOnline,
			NodesOffline:      stats.NodesOffline,
			TotalServices:     stats.TotalServices,
			ServicesHealthy:   stats.ServicesHealthy,
			ServicesDegraded:  stats.ServicesDegraded,
			ServicesUnhealthy: stats.ServicesUnhealthy,
			ActiveAlerts:      stats.ActiveAlerts,
			ResolvedAlerts:    stats.ResolvedAlerts,
		}, nil
	}

	stats, err := s.overviewRepo.GetStats(ctx)
	if err != nil {
		return types.OverviewData{}, err
	}

	return types.OverviewData{
		TotalProjects:     stats.TotalProjects,
		TotalNodes:        stats.TotalNodes,
		NodesOnline:       stats.NodesOnline,
		NodesOffline:      stats.NodesOffline,
		TotalServices:     stats.TotalServices,
		ServicesHealthy:   stats.ServicesHealthy,
		ServicesDegraded:  stats.ServicesDegraded,
		ServicesUnhealthy: stats.ServicesUnhealthy,
		ActiveAlerts:      stats.ActiveAlerts,
		ResolvedAlerts:    stats.ResolvedAlerts,
	}, nil
}

func mapNodeSummaries(nodes []types.NodeReadData) []types.NodeSummaryData {
	items := make([]types.NodeSummaryData, 0, len(nodes))
	for _, node := range nodes {
		items = append(items, types.NodeSummaryData{
			ID:               node.ID,
			ProjectID:        node.ProjectID,
			Name:             node.Name,
			Identifier:       node.Identifier,
			CurrentState:     node.CurrentState,
			LastSeenAt:       node.LastSeenAt,
			LatestCPU:        node.LatestCPU,
			LatestMemory:     node.LatestMemory,
			LatestDisk:       node.LatestDisk,
			ActiveAlertCount: node.ActiveAlertCount,
			ServiceCount:     node.ServiceCount,
		})
	}

	return items
}
