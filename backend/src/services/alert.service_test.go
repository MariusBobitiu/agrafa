package services

import (
	"context"
	"database/sql"
	"sort"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type alertReadRepositoryStub struct {
	combinedRows []repositories.AlertReadRow
	activeRows   []repositories.AlertReadRow
	resolvedRows []repositories.AlertReadRow
	combinedCall int
	activeCall   int
	resolvedCall int
	lastFilters  types.AlertListFilters
}

func (r *alertReadRepositoryStub) ListForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.combinedCall++
	r.lastFilters = filters
	return r.combinedRows, nil
}

func (r *alertReadRepositoryStub) ListActiveForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.activeCall++
	r.lastFilters = filters
	return r.activeRows, nil
}

func (r *alertReadRepositoryStub) ListResolvedForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.resolvedCall++
	r.lastFilters = filters
	rows := append([]repositories.AlertReadRow(nil), r.resolvedRows...)
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TriggeredAt.Equal(rows[j].TriggeredAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].TriggeredAt.After(rows[j].TriggeredAt)
	})
	if filters.Before != nil {
		filtered := rows[:0]
		for _, row := range rows {
			if row.TriggeredAt.Before(filters.Before.TriggeredAt) ||
				(row.TriggeredAt.Equal(filters.Before.TriggeredAt) && row.ID < filters.Before.ID) {
				filtered = append(filtered, row)
			}
		}
		rows = filtered
	}
	if len(rows) > int(filters.Limit) {
		rows = rows[:filters.Limit]
	}
	return rows, nil
}

func TestAlertServiceActiveReadIsIndependentOfHistoryLimit(t *testing.T) {
	t.Parallel()

	triggeredAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	repo := &alertReadRepositoryStub{activeRows: []repositories.AlertReadRow{
		alertReadRow(3, triggeredAt),
		alertReadRow(2, triggeredAt.Add(-time.Minute)),
		alertReadRow(1, triggeredAt.Add(-2*time.Minute)),
	}}
	repo.activeRows[0].RuleType = types.AlertRuleTypeServiceUnhealthy
	repo.activeRows[0].Severity = types.AlertSeverityInfo
	repo.activeRows[0].ServiceID = sql.NullInt64{Int64: 41, Valid: true}
	repo.activeRows[0].ServiceName = sql.NullString{String: "Payments API", Valid: true}
	repo.activeRows[0].Title = "Service is critically unhealthy"
	status := types.AlertStatusActive

	page, err := NewAlertService(repo).List(context.Background(), types.AlertListFilters{
		ProjectID: int64TestPointer(7),
		Status:    &status,
		Limit:     1,
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(page.Alerts) != 3 {
		t.Fatalf("active alerts = %d, want all 3 despite limit=1", len(page.Alerts))
	}
	if repo.activeCall != 1 || repo.resolvedCall != 0 || repo.combinedCall != 0 {
		t.Fatalf("repository calls active/resolved/combined = %d/%d/%d", repo.activeCall, repo.resolvedCall, repo.combinedCall)
	}
	alert := page.Alerts[0]
	if alert.RuleType != types.AlertRuleTypeServiceUnhealthy || alert.Severity != types.AlertSeverityInfo {
		t.Fatalf("authoritative rule fields = %q/%q", alert.RuleType, alert.Severity)
	}
	if alert.ServiceName == nil || *alert.ServiceName != "Payments API" {
		t.Fatalf("service identity = %#v", alert.ServiceName)
	}
}

func TestAlertServiceResolvedHistoryCursorHasNoGapsOrDuplicates(t *testing.T) {
	t.Parallel()

	newest := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	repo := &alertReadRepositoryStub{resolvedRows: []repositories.AlertReadRow{
		alertReadRow(1, newest.Add(-time.Minute)),
		alertReadRow(5, newest),
		alertReadRow(3, newest),
		alertReadRow(4, newest),
		alertReadRow(2, newest.Add(-time.Minute)),
	}}
	status := types.AlertStatusResolved
	filters := types.AlertListFilters{Status: &status, Limit: 2}
	var ids []int64

	for {
		page, err := NewAlertService(repo).List(context.Background(), filters)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		for _, alert := range page.Alerts {
			ids = append(ids, alert.ID)
		}
		if page.NextCursor == nil {
			break
		}
		filters.Before = page.NextCursor
	}

	want := []int64{5, 4, 3, 2, 1}
	if len(ids) != len(want) {
		t.Fatalf("ids = %v, want %v", ids, want)
	}
	seen := map[int64]bool{}
	for i, id := range ids {
		if seen[id] || id != want[i] {
			t.Fatalf("ids = %v, want newest-first without gaps or duplicates: %v", ids, want)
		}
		seen[id] = true
	}
}

func TestAlertServicePassesResolvedHistoryFilters(t *testing.T) {
	t.Parallel()

	status := types.AlertStatusResolved
	ruleType := types.AlertRuleTypeCPUAboveThreshold
	severity := types.AlertSeverityWarning
	category := types.AlertCategoryMetric
	filters := types.AlertListFilters{
		ProjectID: int64TestPointer(9),
		Status:    &status,
		ServiceID: int64TestPointer(11),
		NodeID:    int64TestPointer(12),
		RuleType:  &ruleType,
		Severity:  &severity,
		Category:  &category,
		Limit:     25,
	}
	repo := &alertReadRepositoryStub{}

	if _, err := NewAlertService(repo).List(context.Background(), filters); err != nil {
		t.Fatalf("List() error = %v", err)
	}
	got := repo.lastFilters
	if got.ProjectID == nil || *got.ProjectID != 9 || got.ServiceID == nil || *got.ServiceID != 11 || got.NodeID == nil || *got.NodeID != 12 {
		t.Fatalf("identity filters = %#v", got)
	}
	if got.RuleType == nil || *got.RuleType != ruleType || got.Severity == nil || *got.Severity != severity || got.Category == nil || *got.Category != category {
		t.Fatalf("enum filters = %#v", got)
	}
	if got.Limit != 26 {
		t.Fatalf("repository limit = %d, want limit+1 (26)", got.Limit)
	}
}

func TestMapAlertReadRowsAllowsMissingOptionalEntityJoins(t *testing.T) {
	t.Parallel()

	row := alertReadRow(8, time.Now().UTC())
	row.NodeID = sql.NullInt64{Int64: 71, Valid: true}
	row.ServiceID = sql.NullInt64{Int64: 72, Valid: true}
	row.NodeName = sql.NullString{}
	row.NodeIdentifier = sql.NullString{}
	row.ServiceName = sql.NullString{}

	alert := mapAlertReadRows([]repositories.AlertReadRow{row})[0]
	if alert.NodeID == nil || *alert.NodeID != 71 || alert.ServiceID == nil || *alert.ServiceID != 72 {
		t.Fatalf("fallback IDs were not preserved: %#v", alert)
	}
	if alert.NodeName != nil || alert.NodeIdentifier != nil || alert.ServiceName != nil {
		t.Fatalf("missing joins should map to nullable names: %#v", alert)
	}
}

func TestMapAlertReadRowsPreservesAdministrativeClosure(t *testing.T) {
	t.Parallel()

	closedAt := time.Date(2026, time.August, 29, 14, 32, 0, 0, time.UTC)
	row := alertReadRow(9, closedAt.Add(-10*time.Minute))
	row.Status = types.AlertStatusClosed
	row.ResolvedAt = sql.NullTime{}
	row.ClosedAt = sql.NullTime{Time: closedAt, Valid: true}
	row.ClosureReason = sql.NullString{String: types.AlertClosureReasonRuleDisabled, Valid: true}

	alert := mapAlertReadRows([]repositories.AlertReadRow{row})[0]
	if alert.Status != types.AlertStatusClosed || alert.ResolvedAt != nil || alert.ClosedAt == nil || !alert.ClosedAt.Equal(closedAt) {
		t.Fatalf("closed alert lifecycle = %#v", alert)
	}
	if alert.ClosureReason == nil || *alert.ClosureReason != types.AlertClosureReasonRuleDisabled {
		t.Fatalf("closure reason = %#v, want rule_disabled", alert.ClosureReason)
	}
}

func alertReadRow(id int64, triggeredAt time.Time) repositories.AlertReadRow {
	return repositories.AlertReadRow{
		ID: id, AlertRuleID: id + 100, ProjectID: 7,
		RuleType: types.AlertRuleTypeNodeOffline, Severity: types.AlertSeverityCritical,
		Status: types.AlertStatusResolved, TriggeredAt: triggeredAt,
		ResolvedAt: sql.NullTime{Time: triggeredAt.Add(time.Minute), Valid: true},
		Title:      "Alert", Message: "Alert message", CreatedAt: triggeredAt,
	}
}

func int64TestPointer(value int64) *int64 {
	return &value
}
