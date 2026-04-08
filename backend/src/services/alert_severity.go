package services

import (
	"github.com/MariusBobitiu/agrafa-backend/src/types"
	"github.com/MariusBobitiu/agrafa-backend/src/utils"
)

func normalizeAlertSeverity(value string) string {
	return utils.NormalizeRequiredString(value)
}

func isSupportedAlertSeverity(severity string) bool {
	switch severity {
	case types.AlertSeverityInfo, types.AlertSeverityWarning, types.AlertSeverityCritical:
		return true
	default:
		return false
	}
}

func alertSeverityRank(severity string) (int, bool) {
	switch severity {
	case types.AlertSeverityInfo:
		return 1, true
	case types.AlertSeverityWarning:
		return 2, true
	case types.AlertSeverityCritical:
		return 3, true
	default:
		return 0, false
	}
}

func shouldNotifyForSeverity(minSeverity string, alertSeverity string) bool {
	minRank, ok := alertSeverityRank(minSeverity)
	if !ok {
		return false
	}

	alertRank, ok := alertSeverityRank(alertSeverity)
	if !ok {
		return false
	}

	return alertRank >= minRank
}
