package services

import (
	"bytes"
	"database/sql"
	"encoding/json"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/MariusBobitiu/agrafa-backend/src/types"
)

func mapEvents(rows []generated.Event) []types.EventReadData {
	items := make([]types.EventReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapEvent(row))
	}

	return items
}

func mapEvent(row generated.Event) types.EventReadData {
	return types.EventReadData{
		ID:         row.ID,
		ProjectID:  row.ProjectID,
		NodeID:     nullInt64Ptr(row.NodeID),
		ServiceID:  nullInt64Ptr(row.ServiceID),
		EventType:  row.EventType,
		Severity:   row.Severity,
		Title:      row.Title,
		Details:    normalizeJSONValue(rawJSONValue(row.Details)),
		OccurredAt: row.OccurredAt,
		CreatedAt:  row.CreatedAt,
	}
}

func MapNodeResponse(row generated.Node) types.NodeResponseData {
	return types.NodeResponseData{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		Name:            row.Name,
		Identifier:      row.Identifier,
		CurrentState:    row.CurrentState,
		LastHeartbeatAt: nullTimePtr(row.LastHeartbeatAt),
		Metadata:        row.Metadata,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

func MapServiceResponse(row generated.Service) types.ServiceResponseData {
	return types.ServiceResponseData{
		ID:                   row.ID,
		ProjectID:            row.ProjectID,
		NodeID:               row.NodeID,
		Name:                 row.Name,
		CheckType:            row.CheckType,
		CheckTarget:          row.CheckTarget,
		CurrentState:         row.CurrentState,
		ConsecutiveFailures:  row.ConsecutiveFailures,
		ConsecutiveSuccesses: row.ConsecutiveSuccesses,
		LastCheckAt:          nullTimePtr(row.LastCheckAt),
		CreatedAt:            row.CreatedAt,
		UpdatedAt:            row.UpdatedAt,
	}
}

func MapUserResponse(row generated.User) types.UserData {
	return types.UserData{
		ID:                  row.ID,
		Name:                row.Name,
		Email:               row.Email,
		EmailVerified:       row.EmailVerified,
		Image:               nullStringPtr(row.Image),
		OnboardingCompleted: row.OnboardingCompleted,
		TwoFactorEnabled:    row.TwoFactorEnabled,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}
}

func mapAuthSessions(rows []generated.Session, currentTokenHash string) []types.AuthUserSessionData {
	items := make([]types.AuthUserSessionData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAuthSession(row, currentTokenHash))
	}

	return items
}

func mapAuthSession(row generated.Session, currentTokenHash string) types.AuthUserSessionData {
	return types.AuthUserSessionData{
		ID:        row.ID,
		ExpiresAt: row.ExpiresAt,
		IPAddress: nullStringPtr(row.IpAddress),
		UserAgent: nullStringPtr(row.UserAgent),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		IsCurrent: row.TokenHash == currentTokenHash,
	}
}

func mapProjectMembers(rows []generated.ListProjectMembersForReadRow) []types.ProjectMemberReadData {
	items := make([]types.ProjectMemberReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapProjectMemberListRow(row))
	}

	return items
}

func mapProjectMemberListRow(row generated.ListProjectMembersForReadRow) types.ProjectMemberReadData {
	return types.ProjectMemberReadData{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Role:      row.Role,
		CreatedAt: row.CreatedAt,
		User: types.ProjectMemberUserSummary{
			Name:  row.Name,
			Email: row.Email,
			Image: nullStringPtr(row.Image),
		},
	}
}

func mapProjectMember(row generated.GetProjectMemberForReadByIDRow) types.ProjectMemberReadData {
	return types.ProjectMemberReadData{
		ID:        row.ID,
		ProjectID: row.ProjectID,
		UserID:    row.UserID,
		Role:      row.Role,
		CreatedAt: row.CreatedAt,
		User: types.ProjectMemberUserSummary{
			Name:  row.Name,
			Email: row.Email,
			Image: nullStringPtr(row.Image),
		},
	}
}

func mapProjectInvitations(rows []generated.ProjectInvitation) []types.ProjectInvitationReadData {
	items := make([]types.ProjectInvitationReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapProjectInvitation(row))
	}

	return items
}

func mapProjectInvitation(row generated.ProjectInvitation) types.ProjectInvitationReadData {
	return types.ProjectInvitationReadData{
		ID:              row.ID,
		ProjectID:       row.ProjectID,
		Email:           row.Email,
		Role:            row.Role,
		InvitedByUserID: row.InvitedByUserID,
		ExpiresAt:       row.ExpiresAt,
		AcceptedAt:      nullTimePtr(row.AcceptedAt),
		CreatedAt:       row.CreatedAt,
	}
}

func mapProjectSummaries(rows []generated.ListProjectsForUserRow) []types.ProjectSummaryData {
	items := make([]types.ProjectSummaryData, 0, len(rows))
	for _, row := range rows {
		items = append(items, types.ProjectSummaryData{
			ID:              row.ID,
			Slug:            row.Slug,
			Name:            row.Name,
			CreatedAt:       row.CreatedAt,
			CurrentUserRole: row.Role,
		})
	}

	return items
}

func mapAlerts(rows []generated.AlertInstance) []types.AlertReadData {
	items := make([]types.AlertReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAlert(row))
	}

	return items
}

func mapAlert(row generated.AlertInstance) types.AlertReadData {
	return types.AlertReadData{
		ID:          row.ID,
		AlertRuleID: row.AlertRuleID,
		ProjectID:   row.ProjectID,
		NodeID:      nullInt64Ptr(row.NodeID),
		ServiceID:   nullInt64Ptr(row.ServiceID),
		Status:      row.Status,
		TriggeredAt: row.TriggeredAt,
		ResolvedAt:  nullTimePtr(row.ResolvedAt),
		Title:       row.Title,
		Message:     row.Message,
		CreatedAt:   row.CreatedAt,
	}
}

func mapAlertRules(rows []generated.AlertRule) []types.AlertRuleReadData {
	items := make([]types.AlertRuleReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapAlertRule(row))
	}

	return items
}

func mapAlertRule(row generated.AlertRule) types.AlertRuleReadData {
	return types.AlertRuleReadData{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		NodeID:         nullInt64Ptr(row.NodeID),
		ServiceID:      nullInt64Ptr(row.ServiceID),
		RuleType:       row.RuleType,
		MetricName:     nullStringPtr(row.MetricName),
		ThresholdValue: nullFloat64Ptr(row.ThresholdValue),
		IsEnabled:      row.IsEnabled,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
	}
}

func mapNotificationRecipients(rows []generated.NotificationRecipient) []types.NotificationRecipientReadData {
	items := make([]types.NotificationRecipientReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapNotificationRecipient(row))
	}

	return items
}

func mapNotificationRecipient(row generated.NotificationRecipient) types.NotificationRecipientReadData {
	return types.NotificationRecipientReadData{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		ChannelType: row.ChannelType,
		Target:      row.Target,
		IsEnabled:   row.IsEnabled,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func mapNotificationDeliveries(rows []generated.NotificationDelivery) []types.NotificationDeliveryReadData {
	items := make([]types.NotificationDeliveryReadData, 0, len(rows))
	for _, row := range rows {
		items = append(items, mapNotificationDelivery(row))
	}

	return items
}

func mapNotificationDelivery(row generated.NotificationDelivery) types.NotificationDeliveryReadData {
	return types.NotificationDeliveryReadData{
		ID:                      row.ID,
		ProjectID:               row.ProjectID,
		NotificationRecipientID: nullInt64Ptr(row.NotificationRecipientID),
		AlertInstanceID:         nullInt64Ptr(row.AlertInstanceID),
		ChannelType:             row.ChannelType,
		Target:                  row.Target,
		EventType:               row.EventType,
		Status:                  row.Status,
		ErrorMessage:            nullStringPtr(row.ErrorMessage),
		SentAt:                  row.SentAt,
		CreatedAt:               row.CreatedAt,
	}
}

func nullInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}

	number := value.Int64
	return &number
}

func nullFloat64Ptr(value sql.NullFloat64) *float64 {
	if !value.Valid {
		return nil
	}

	number := value.Float64
	return &number
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}

	text := value.String
	return &text
}

func rawJSONValue(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil
	}

	return value
}

func normalizeJSONValue(value any) any {
	switch typed := value.(type) {
	case []any:
		items := make([]any, len(typed))
		for index, item := range typed {
			items[index] = normalizeJSONValue(item)
		}

		return items
	case map[string]any:
		if normalized, ok := unwrapNullableJSONValue(typed); ok {
			return normalized
		}

		items := make(map[string]any, len(typed))
		for key, item := range typed {
			items[key] = normalizeJSONValue(item)
		}

		return items
	default:
		return value
	}
}

func unwrapNullableJSONValue(value map[string]any) (any, bool) {
	valid, ok := value["Valid"].(bool)
	if !ok || len(value) != 2 {
		return nil, false
	}

	for _, key := range []string{"Int64", "Int32", "Float64", "String", "Time"} {
		nullableValue, exists := value[key]
		if !exists {
			continue
		}

		if !valid {
			return nil, true
		}

		return nullableValue, true
	}

	return nil, false
}
