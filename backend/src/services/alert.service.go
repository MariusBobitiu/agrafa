package services

import (
	"context"
	"sort"

	"github.com/MariusBobitiu/agrafa-backend/src/repositories"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

const (
	DefaultAlertHistoryLimit int32 = 50
	MaxAlertHistoryLimit     int32 = 100
)

type alertReadRepository interface {
	ListForRead(ctx context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error)
	ListActiveForRead(ctx context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error)
	ListResolvedForRead(ctx context.Context, filters types.AlertListFilters) ([]repositories.AlertReadRow, error)
}

type AlertService struct {
	alertInstanceRepo alertReadRepository
}

func NewAlertService(alertInstanceRepo alertReadRepository) *AlertService {
	return &AlertService{alertInstanceRepo: alertInstanceRepo}
}

func (s *AlertService) List(ctx context.Context, filters types.AlertListFilters) (types.AlertPageData, error) {
	if err := validateAlertListFilters(filters); err != nil {
		return types.AlertPageData{}, err
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = DefaultAlertHistoryLimit
	}
	if limit > MaxAlertHistoryLimit {
		limit = MaxAlertHistoryLimit
	}
	filters.Limit = limit

	var (
		rows []repositories.AlertReadRow
		err  error
	)
	switch {
	case filters.Status != nil && *filters.Status == types.AlertStatusActive:
		// Active alerts deliberately have no history page limit. The one-active-instance-per-rule
		// invariant keeps this read naturally bounded by the project's alert rules.
		rows, err = s.alertInstanceRepo.ListActiveForRead(ctx, filters)
	case filters.Status != nil && *filters.Status == types.AlertStatusResolved:
		filters.Limit = limit + 1
		rows, err = s.alertInstanceRepo.ListResolvedForRead(ctx, filters)
	default:
		// Preserve the bounded combined read for existing API consumers. The Alerts page uses
		// the dedicated active and resolved paths above.
		rows, err = s.alertInstanceRepo.ListForRead(ctx, filters)
	}
	if err != nil {
		return types.AlertPageData{}, err
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TriggeredAt.Equal(rows[j].TriggeredAt) {
			return rows[i].ID > rows[j].ID
		}
		return rows[i].TriggeredAt.After(rows[j].TriggeredAt)
	})

	hasMore := filters.Status != nil && *filters.Status == types.AlertStatusResolved && len(rows) > int(limit)
	if hasMore {
		rows = rows[:limit]
	}

	page := types.AlertPageData{Alerts: mapAlertReadRows(rows)}
	if hasMore && len(rows) > 0 {
		last := rows[len(rows)-1]
		page.NextCursor = &types.AlertCursor{TriggeredAt: last.TriggeredAt, ID: last.ID}
	}
	return page, nil
}

func validateAlertListFilters(filters types.AlertListFilters) error {
	if filters.Status != nil && *filters.Status != types.AlertStatusActive && *filters.Status != types.AlertStatusResolved {
		return types.ErrInvalidAlertStatus
	}
	if filters.Before != nil && (filters.Status == nil || *filters.Status != types.AlertStatusResolved) {
		return types.ErrInvalidAlertStatus
	}
	if filters.RuleType != nil && !isSupportedAlertRuleType(*filters.RuleType) {
		return types.ErrUnsupportedAlertRuleType
	}
	if filters.Severity != nil && !isSupportedAlertSeverity(*filters.Severity) {
		return types.ErrInvalidAlertSeverity
	}
	if filters.Category != nil {
		switch *filters.Category {
		case types.AlertCategoryNode, types.AlertCategoryService, types.AlertCategoryMetric:
		default:
			return types.ErrInvalidAlertCategory
		}
	}
	return nil
}

func mapAlertReadRows(rows []repositories.AlertReadRow) []types.AlertReadData {
	items := make([]types.AlertReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.AlertReadData{
			ID:             row.ID,
			AlertRuleID:    row.AlertRuleID,
			ProjectID:      row.ProjectID,
			RuleType:       row.RuleType,
			Severity:       row.Severity,
			NodeID:         nullInt64Ptr(row.NodeID),
			NodeName:       nullStringPtr(row.NodeName),
			NodeIdentifier: nullStringPtr(row.NodeIdentifier),
			ServiceID:      nullInt64Ptr(row.ServiceID),
			ServiceName:    nullStringPtr(row.ServiceName),
			Status:         row.Status,
			TriggeredAt:    row.TriggeredAt,
			ResolvedAt:     nullTimePtr(row.ResolvedAt),
			Title:          row.Title,
			Message:        row.Message,
			CreatedAt:      row.CreatedAt,
		})
	}
	return items
}
