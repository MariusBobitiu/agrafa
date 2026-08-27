package controllers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/services"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

type readAlertRepositoryStub struct {
	rows        []repositories.AlertReadRow
	lastFilters types.AlertListFilters
	calls       int
}

func (r *readAlertRepositoryStub) ListForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.calls++
	r.lastFilters = filters
	return r.rows, nil
}

func (r *readAlertRepositoryStub) ListActiveForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.calls++
	r.lastFilters = filters
	return r.rows, nil
}

func (r *readAlertRepositoryStub) ListResolvedForRead(_ context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error) {
	r.calls++
	r.lastFilters = filters
	return r.rows, nil
}

func TestReadControllerListAlertsReturnsResolvedCursorAndPresentationData(t *testing.T) {
	t.Parallel()

	triggeredAt := time.Date(2026, time.August, 23, 12, 0, 0, 0, time.UTC)
	rows := make([]repositories.AlertReadRow, 0, 3)
	for id := int64(3); id >= 1; id-- {
		rows = append(rows, repositories.AlertReadRow{
			ID: id, AlertRuleID: 10 + id, ProjectID: 7,
			ServiceID:   sql.NullInt64{Int64: 44, Valid: true},
			ServiceName: sql.NullString{String: "Public API", Valid: true},
			RuleType:    types.AlertRuleTypeServiceUnhealthy, Severity: types.AlertSeverityCritical,
			Status: types.AlertStatusResolved, TriggeredAt: triggeredAt.Add(time.Duration(id) * time.Second),
			ResolvedAt: sql.NullTime{Time: triggeredAt.Add(time.Minute), Valid: true},
			Title:      "Service unhealthy", Message: "HTTP 503", CreatedAt: triggeredAt,
		})
	}
	repo := &readAlertRepositoryStub{rows: rows}
	alertService := services.NewAlertService(repo)
	controller := NewReadController(nil, nil, nil, nil, alertService, nil)
	request := httptest.NewRequest(
		http.MethodGet,
		"/v1/alerts?project_id=7&status=resolved&service_id=44&rule_type=service_unhealthy&severity=critical&category=service&limit=2",
		nil,
	)
	recorder := httptest.NewRecorder()

	controller.ListAlerts(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", recorder.Code, recorder.Body.String())
	}
	if repo.lastFilters.Limit != 3 || repo.lastFilters.ServiceID == nil || *repo.lastFilters.ServiceID != 44 {
		t.Fatalf("repository filters = %#v", repo.lastFilters)
	}
	var response struct {
		Alerts     []types.AlertReadData `json:"alerts"`
		Pagination struct {
			Limit      int32   `json:"limit"`
			HasMore    bool    `json:"has_more"`
			NextCursor *string `json:"next_cursor"`
		} `json:"pagination"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Alerts) != 2 || response.Alerts[0].ServiceName == nil || *response.Alerts[0].ServiceName != "Public API" {
		t.Fatalf("alerts = %#v", response.Alerts)
	}
	if response.Pagination.Limit != 2 || !response.Pagination.HasMore || response.Pagination.NextCursor == nil {
		t.Fatalf("pagination = %#v", response.Pagination)
	}
	cursor, err := decodeAlertCursor(*response.Pagination.NextCursor)
	if err != nil || cursor.ID != response.Alerts[1].ID || !cursor.TriggeredAt.Equal(response.Alerts[1].TriggeredAt) {
		t.Fatalf("decoded cursor = %#v, err = %v", cursor, err)
	}
}

func TestReadControllerListAlertsRejectsInvalidEnumFilters(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"status=unknown",
		"rule_type=message_guessed",
		"severity=urgent",
		"category=incident",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			repo := &readAlertRepositoryStub{}
			controller := NewReadController(nil, nil, nil, nil, services.NewAlertService(repo), nil)
			request := httptest.NewRequest(http.MethodGet, "/v1/alerts?"+query, nil)
			recorder := httptest.NewRecorder()

			controller.ListAlerts(recorder, request)

			if recorder.Code != http.StatusBadRequest || repo.calls != 0 {
				t.Fatalf("status/calls = %d/%d, want 400/0: %s", recorder.Code, repo.calls, recorder.Body.String())
			}
		})
	}
}
