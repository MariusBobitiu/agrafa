package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type ServiceRepository struct {
	queries *generated.Queries
}

func NewServiceRepository(queries *generated.Queries) *ServiceRepository {
	return &ServiceRepository{queries: queries}
}

func (r *ServiceRepository) GetByID(ctx context.Context, id int64) (generated.Service, error) {
	return r.queries.GetServiceByID(ctx, id)
}

func (r *ServiceRepository) Create(ctx context.Context, params generated.CreateServiceParams) (generated.Service, error) {
	return r.queries.CreateService(ctx, params)
}

func (r *ServiceRepository) Delete(ctx context.Context, id int64) (int64, error) {
	return r.queries.DeleteServiceByID(ctx, id)
}

func (r *ServiceRepository) CountByNodeID(ctx context.Context, nodeID int64) (int64, error) {
	return r.queries.CountServicesByNodeID(ctx, nodeID)
}

func (r *ServiceRepository) List(ctx context.Context) ([]generated.Service, error) {
	return r.queries.ListServices(ctx)
}

func (r *ServiceRepository) ListByProject(ctx context.Context, projectID int64) ([]generated.Service, error) {
	return r.queries.ListServicesByProject(ctx, projectID)
}

func (r *ServiceRepository) ListAgentConfigChecksByNodeID(ctx context.Context, nodeID int64) ([]generated.ListAgentConfigChecksByNodeIDRow, error) {
	return r.queries.ListAgentConfigChecksByNodeID(ctx, nodeID)
}

func (r *ServiceRepository) ListForRead(ctx context.Context, filters types.ServiceListFilters) ([]generated.Service, error) {
	params := generated.ListServicesForReadParams{}
	if filters.ProjectID != nil {
		params.HasProjectID = true
		params.ProjectID = *filters.ProjectID
	}
	if filters.NodeID != nil {
		params.HasNodeID = true
		params.NodeID = *filters.NodeID
	}
	if filters.Status != nil {
		params.HasStatus = true
		params.Status = *filters.Status
	}

	if filters.Limit != nil {
		return r.queries.ListServicesForReadLimited(ctx, generated.ListServicesForReadLimitedParams{
			HasProjectID: params.HasProjectID,
			ProjectID:    params.ProjectID,
			HasNodeID:    params.HasNodeID,
			NodeID:       params.NodeID,
			HasStatus:    params.HasStatus,
			Status:       params.Status,
			LimitRows:    *filters.Limit,
		})
	}

	return r.queries.ListServicesForRead(ctx, params)
}

func (r *ServiceRepository) UpdateState(ctx context.Context, params generated.UpdateServiceStateParams) (generated.Service, error) {
	return r.queries.UpdateServiceState(ctx, params)
}

func (r *ServiceRepository) UpdateDefinition(ctx context.Context, serviceID int64, name string, checkType string, checkTarget string) (generated.Service, error) {
	return r.queries.UpdateServiceDefinition(ctx, generated.UpdateServiceDefinitionParams{
		ID:          serviceID,
		Name:        name,
		CheckType:   checkType,
		CheckTarget: checkTarget,
	})
}

func (r *ServiceRepository) ListCountsByNode(ctx context.Context, projectID *int64) ([]generated.ListServiceCountsByNodeRow, error) {
	if projectID != nil {
		rows, err := r.queries.ListServiceCountsByNodeByProject(ctx, *projectID)
		if err != nil {
			return nil, err
		}

		items := make([]generated.ListServiceCountsByNodeRow, 0, len(rows))
		for _, row := range rows {
			items = append(items, generated.ListServiceCountsByNodeRow{
				NodeID:       row.NodeID,
				ServiceCount: row.ServiceCount,
			})
		}

		return items, nil
	}

	return r.queries.ListServiceCountsByNode(ctx)
}
