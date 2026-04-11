package repositories

import (
	"context"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
)

type InstanceSettingRepository struct {
	queries *generated.Queries
}

func NewInstanceSettingRepository(queries *generated.Queries) *InstanceSettingRepository {
	return &InstanceSettingRepository{queries: queries}
}

func (r *InstanceSettingRepository) GetByKey(ctx context.Context, key string) (generated.InstanceSetting, error) {
	return r.queries.GetInstanceSettingByKey(ctx, key)
}

func (r *InstanceSettingRepository) List(ctx context.Context) ([]generated.InstanceSetting, error) {
	return r.queries.ListInstanceSettings(ctx)
}

func (r *InstanceSettingRepository) Upsert(ctx context.Context, params generated.UpsertInstanceSettingParams) (generated.InstanceSetting, error) {
	return r.queries.UpsertInstanceSetting(ctx, params)
}

func (r *InstanceSettingRepository) DeleteByKey(ctx context.Context, key string) (int64, error) {
	return r.queries.DeleteInstanceSettingByKey(ctx, key)
}
