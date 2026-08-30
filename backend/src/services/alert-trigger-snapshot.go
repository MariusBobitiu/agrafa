package services

import (
	"encoding/json"

	"github.com/MariusBobitiu/agrafa-backend/src/db/sqlc/generated"
	"github.com/sqlc-dev/pqtype"
)

type alertTriggerSnapshot struct {
	MetricName     string   `json:"metric_name,omitempty"`
	MetricValue    *float64 `json:"metric_value,omitempty"`
	ThresholdValue *float64 `json:"threshold_value,omitempty"`
}

func buildMetricAlertTriggerSnapshot(rule generated.AlertRule, metricValue *float64) (pqtype.NullRawMessage, error) {
	if metricValue == nil || !rule.MetricName.Valid || !rule.ThresholdValue.Valid {
		return pqtype.NullRawMessage{}, nil
	}

	value := *metricValue
	threshold := rule.ThresholdValue.Float64
	snapshot, err := json.Marshal(alertTriggerSnapshot{
		MetricName:     rule.MetricName.String,
		MetricValue:    &value,
		ThresholdValue: &threshold,
	})
	if err != nil {
		return pqtype.NullRawMessage{}, err
	}

	return pqtype.NullRawMessage{RawMessage: snapshot, Valid: true}, nil
}

func parseMetricAlertTriggerSnapshot(value pqtype.NullRawMessage) (alertTriggerSnapshot, bool) {
	if !value.Valid || len(value.RawMessage) == 0 {
		return alertTriggerSnapshot{}, false
	}

	var snapshot alertTriggerSnapshot
	if err := json.Unmarshal(value.RawMessage, &snapshot); err != nil || snapshot.MetricName == "" || snapshot.MetricValue == nil || snapshot.ThresholdValue == nil {
		return alertTriggerSnapshot{}, false
	}

	return snapshot, true
}
