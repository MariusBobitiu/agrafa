package repositories

import (
	"context"
	"database/sql"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type MetricRepository struct {
	db      *sql.DB
	queries *generated.Queries
}

func NewMetricRepository(db *sql.DB, queries *generated.Queries) *MetricRepository {
	return &MetricRepository{
		db:      db,
		queries: queries,
	}
}

func (r *MetricRepository) Create(ctx context.Context, params generated.CreateMetricSampleParams) (generated.MetricSample, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.MetricSample, error) {
		return queries.CreateMetricSample(ctx, params)
	})
}

func (r *MetricRepository) GetLatestNodeMetricByName(ctx context.Context, nodeID int64, metricName string) (generated.MetricSample, error) {
	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) (generated.MetricSample, error) {
		return queries.GetLatestNodeMetricSampleByName(ctx, generated.GetLatestNodeMetricSampleByNameParams{
			NodeID:     nodeID,
			MetricName: metricName,
		})
	})
}

func (r *MetricRepository) ListLatestNodeMetrics(ctx context.Context, projectID *int64) ([]generated.ListLatestOperationalNodeMetricsRow, error) {
	if projectID != nil {
		rows, err := withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListLatestOperationalNodeMetricsByProjectRow, error) {
			return queries.ListLatestOperationalNodeMetricsByProject(ctx, *projectID)
		})
		if err != nil {
			return nil, err
		}

		items := make([]generated.ListLatestOperationalNodeMetricsRow, 0, len(rows))
		for _, row := range rows {
			items = append(items, generated.ListLatestOperationalNodeMetricsRow{
				NodeID:      row.NodeID,
				MetricName:  row.MetricName,
				MetricValue: row.MetricValue,
				MetricUnit:  row.MetricUnit,
				ObservedAt:  row.ObservedAt,
			})
		}

		return items, nil
	}

	return withRLSQueries(ctx, r.db, r.queries, func(queries *generated.Queries) ([]generated.ListLatestOperationalNodeMetricsRow, error) {
		return queries.ListLatestOperationalNodeMetrics(ctx)
	})
}
