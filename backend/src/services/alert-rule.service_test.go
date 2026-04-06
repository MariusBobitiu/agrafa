package services

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type fakeAlertRuleServiceAlertRuleRepo struct {
	rule       generated.AlertRule
	getErr     error
	deleteRows int64
	deleteErr  error
}

func (r *fakeAlertRuleServiceAlertRuleRepo) Create(_ context.Context, _ generated.CreateAlertRuleParams) (generated.AlertRule, error) {
	return generated.AlertRule{}, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) GetByID(_ context.Context, _ int64) (generated.AlertRule, error) {
	return r.rule, r.getErr
}

func (r *fakeAlertRuleServiceAlertRuleRepo) UpdateEnabled(_ context.Context, _ int64, _ bool) (generated.AlertRule, error) {
	return generated.AlertRule{}, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) List(_ context.Context, _ *int64) ([]generated.AlertRule, error) {
	return nil, nil
}

func (r *fakeAlertRuleServiceAlertRuleRepo) Delete(_ context.Context, _ int64) (int64, error) {
	return r.deleteRows, r.deleteErr
}

type fakeAlertRuleServiceProjectRepo struct{}

func (r *fakeAlertRuleServiceProjectRepo) GetByID(_ context.Context, _ int64) (generated.Project, error) {
	return generated.Project{}, nil
}

type fakeAlertRuleServiceNodeRepo struct{}

func (r *fakeAlertRuleServiceNodeRepo) GetByID(_ context.Context, _ int64) (generated.Node, error) {
	return generated.Node{}, nil
}

type fakeAlertRuleServiceServiceRepo struct{}

func (r *fakeAlertRuleServiceServiceRepo) GetByID(_ context.Context, _ int64) (generated.Service, error) {
	return generated.Service{}, nil
}

func TestAlertRuleServiceDeleteSucceeds(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{deleteRows: 1},
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:      &fakeAlertRuleServiceNodeRepo{},
		serviceRepo:   &fakeAlertRuleServiceServiceRepo{},
	}

	if err := service.Delete(context.Background(), 8); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestAlertRuleServiceDeleteMissingReturnsNotFound(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{deleteRows: 0},
		projectRepo:   &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:      &fakeAlertRuleServiceNodeRepo{},
		serviceRepo:   &fakeAlertRuleServiceServiceRepo{},
	}

	err := service.Delete(context.Background(), 8)
	if !errors.Is(err, types.ErrAlertRuleNotFound) {
		t.Fatalf("Delete() error = %v, want ErrAlertRuleNotFound", err)
	}
}

func TestAlertRuleServiceGetByIDMapsNullableFields(t *testing.T) {
	t.Parallel()

	service := &AlertRuleService{
		alertRuleRepo: &fakeAlertRuleServiceAlertRuleRepo{
			rule: generated.AlertRule{
				ID:             4,
				ProjectID:      1,
				NodeID:         sql.NullInt64{},
				ServiceID:      sql.NullInt64{Int64: 9, Valid: true},
				RuleType:       types.AlertRuleTypeServiceUnhealthy,
				ThresholdValue: sql.NullFloat64{},
				IsEnabled:      true,
				CreatedAt:      time.Date(2026, time.April, 5, 12, 0, 0, 0, time.UTC),
				UpdatedAt:      time.Date(2026, time.April, 5, 12, 1, 0, 0, time.UTC),
			},
		},
		projectRepo: &fakeAlertRuleServiceProjectRepo{},
		nodeRepo:    &fakeAlertRuleServiceNodeRepo{},
		serviceRepo: &fakeAlertRuleServiceServiceRepo{},
	}

	rule, err := service.GetByID(context.Background(), 4)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if rule.NodeID != nil {
		t.Fatalf("rule.NodeID = %#v, want nil", rule.NodeID)
	}
	if rule.ServiceID == nil || *rule.ServiceID != 9 {
		t.Fatalf("rule.ServiceID = %#v, want 9", rule.ServiceID)
	}
	if rule.ThresholdValue != nil {
		t.Fatalf("rule.ThresholdValue = %#v, want nil", rule.ThresholdValue)
	}
}
