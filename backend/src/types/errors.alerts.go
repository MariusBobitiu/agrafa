package types

import "errors"

var (
	ErrInvalidAlertRuleType              = errors.New("rule_type is required")
	ErrUnsupportedAlertRuleType          = errors.New("invalid rule_type")
	ErrMissingAlertSeverity              = errors.New("severity is required")
	ErrInvalidAlertSeverity              = errors.New("severity must be info, warning, or critical")
	ErrInvalidThresholdValue             = errors.New("threshold_value must be greater than 0")
	ErrInvalidAlertStatus                = errors.New("status must be active or resolved")
	ErrAlertRuleNotFound                 = errors.New("alert rule not found")
	ErrInvalidNotificationChannelType    = errors.New("channel_type must be email")
	ErrInvalidNotificationTarget         = errors.New("target must be a valid email address")
	ErrEmptyNotificationRecipients       = errors.New("at least one recipient is required")
	ErrInvalidNotificationMinSeverity    = errors.New("min_severity must be info, warning, or critical")
	ErrNotificationRecipientNotFound     = errors.New("notification recipient not found")
	ErrInvalidNotificationDeliveryStatus = errors.New("status must be sent or failed")
)
