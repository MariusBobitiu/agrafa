package types

import (
	"encoding/json"
	"time"
)

const (
	NodeStateOnline  = "online"
	NodeStateOffline = "offline"
)

const (
	NodeTypeManaged = "managed"
	NodeTypeAgent   = "agent"
)

const (
	ExecutionModeManaged = "managed"
	ExecutionModeAgent   = "agent"
)

const (
	ServiceStateHealthy   = "healthy"
	ServiceStateDegraded  = "degraded"
	ServiceStateUnhealthy = "unhealthy"
)

const (
	EventTypeNodeOnline       = "node_online"
	EventTypeNodeOffline      = "node_offline"
	EventTypeServiceDegraded  = "service_degraded"
	EventTypeServiceUnhealthy = "service_unhealthy"
	EventTypeServiceRecovered = "service_recovered"
	EventTypeAlertTriggered   = "alert_triggered"
	EventTypeAlertResolved    = "alert_resolved"
)

const (
	AlertRuleTypeNodeOffline          = "node_offline"
	AlertRuleTypeServiceUnhealthy     = "service_unhealthy"
	AlertRuleTypeCPUAboveThreshold    = "cpu_above_threshold"
	AlertRuleTypeMemoryAboveThreshold = "memory_above_threshold"
	AlertRuleTypeDiskAboveThreshold   = "disk_above_threshold"
)

const (
	AlertStatusActive   = "active"
	AlertStatusResolved = "resolved"
)

const (
	AlertSeverityInfo     = "info"
	AlertSeverityWarning  = "warning"
	AlertSeverityCritical = "critical"
)

const (
	MetricNameCPUUsage    = "cpu_usage"
	MetricNameMemoryUsage = "memory_usage"
	MetricNameDiskUsage   = "disk_usage"
)

const (
	NotificationChannelTypeEmail = "email"
)

const (
	NotificationDeliveryStatusSent   = "sent"
	NotificationDeliveryStatusFailed = "failed"
)

const (
	VerificationTokenTypeEmailVerification = "email_verification"
	VerificationTokenTypePasswordReset     = "password_reset"
)

type HeartbeatInput struct {
	AuthenticatedNodeID int64
	ReportedNodeID      *int64
	ObservedAt          time.Time
	Source              string
	Payload             json.RawMessage
}

type AgentShutdownInput struct {
	AuthenticatedNodeID int64
	ReportedNodeID      *int64
	ObservedAt          time.Time
	Reason              string
	Payload             json.RawMessage
}

type HealthCheckInput struct {
	AuthenticatedNodeID int64
	ServiceID           int64
	ObservedAt          time.Time
	IsSuccess           bool
	StatusCode          *int32
	ResponseTimeMs      *int32
	Message             string
	Payload             json.RawMessage
}

type MetricSampleInput struct {
	MetricName  string
	MetricValue float64
	MetricUnit  string
	ObservedAt  time.Time
	Payload     json.RawMessage
}

type MetricIngestionInput struct {
	AuthenticatedNodeID int64
	ReportedNodeID      *int64
	ServiceID           *int64
	Samples             []MetricSampleInput
}

type CreateAlertRuleInput struct {
	ProjectID      int64
	NodeID         *int64
	ServiceID      *int64
	RuleType       string
	Severity       string
	ThresholdValue *float64
}

type UpdateAlertRuleInput struct {
	ID        int64
	IsEnabled bool
}

type UpdateProjectInput struct {
	Name *string
}

type UpdateNodeInput struct {
	Name       *string
	Identifier *string
}

type UpdateServiceInput struct {
	Name        *string
	CheckType   *string
	CheckTarget *string
}

type CreateServiceInput struct {
	ProjectID     int64
	NodeID        *int64
	ExecutionMode string
	Name          string
	CheckType     string
	CheckTarget   string
}

type CreateNotificationRecipientItemInput struct {
	Target      string
	MinSeverity string
}

type CreateNotificationRecipientsInput struct {
	ProjectID   int64
	ChannelType string
	Recipients  []CreateNotificationRecipientItemInput
}

type UpdateNotificationRecipientInput struct {
	ID        int64
	IsEnabled bool
}

type CreateNotificationDeliveryInput struct {
	ProjectID               int64
	NotificationRecipientID *int64
	AlertInstanceID         *int64
	ChannelType             string
	Target                  string
	EventType               string
	Status                  string
	ErrorMessage            *string
	SentAt                  time.Time
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email      string
	Password   string
	RememberMe bool
}

type SessionActor struct {
	IPAddress string
	UserAgent string
}

type CreateProjectMemberInput struct {
	ProjectID int64
	UserID    string
	Role      string
}

type UpdateProjectMemberInput struct {
	ID   string
	Role string
}

type CreateProjectInvitationInput struct {
	ProjectID       int64
	Email           string
	Role            string
	InvitedByUserID string
	InvitedByName   string
}

type ProjectInvitationCreateResultData struct {
	Email        string                     `json:"email"`
	Role         string                     `json:"role"`
	Status       string                     `json:"status"`
	Invitation   *ProjectInvitationReadData `json:"invitation,omitempty"`
	ErrorCode    *string                    `json:"error_code,omitempty"`
	ErrorMessage *string                    `json:"error_message,omitempty"`
}

type ProjectInvitationCreateBatchData struct {
	ProjectID int64                               `json:"project_id"`
	Results   []ProjectInvitationCreateResultData `json:"results"`
}

type AuthSessionResult struct {
	User      UserData  `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type InstanceSettingReadData struct {
	Key             string `json:"key"`
	Group           string `json:"group"`
	Label           string `json:"label"`
	Description     string `json:"description"`
	Type            string `json:"type"`
	Value           any    `json:"value,omitempty"`
	IsSensitive     bool   `json:"is_sensitive"`
	IsEncrypted     bool   `json:"is_encrypted"`
	IsEnvOverridden bool   `json:"is_env_overridden"`
	IsEditable      bool   `json:"is_editable"`
	IsConfigured    *bool  `json:"is_configured,omitempty"`
}

type AuthUserSessionData struct {
	ID        string    `json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
	IPAddress *string   `json:"ip_address"`
	UserAgent *string   `json:"user_agent"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	IsCurrent bool      `json:"is_current"`
}

type UserData struct {
	ID                  string    `json:"id"`
	Name                string    `json:"name"`
	Email               string    `json:"email"`
	EmailVerified       bool      `json:"email_verified"`
	Image               *string   `json:"image"`
	OnboardingCompleted bool      `json:"onboarding_completed"`
	TwoFactorEnabled    bool      `json:"two_factor_enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type ProjectMemberUserSummary struct {
	Name  string  `json:"name"`
	Email string  `json:"email"`
	Image *string `json:"image"`
}

type ProjectSummaryData struct {
	ID              int64     `json:"id"`
	Slug            string    `json:"slug"`
	Name            string    `json:"name"`
	CreatedAt       time.Time `json:"created_at"`
	CurrentUserRole string    `json:"current_user_role"`
}

type ProjectDetailData struct {
	ID               int64     `json:"id"`
	Slug             string    `json:"slug"`
	Name             string    `json:"name"`
	CreatedAt        time.Time `json:"created_at"`
	CurrentUserRole  string    `json:"current_user_role"`
	NodeCount        int64     `json:"node_count"`
	ServiceCount     int64     `json:"service_count"`
	ActiveAlertCount int64     `json:"active_alert_count"`
}

type ProjectMemberReadData struct {
	ID        string                   `json:"id"`
	ProjectID int64                    `json:"project_id"`
	UserID    string                   `json:"user_id"`
	Role      string                   `json:"role"`
	CreatedAt time.Time                `json:"created_at"`
	User      ProjectMemberUserSummary `json:"user"`
}

type ProjectInvitationReadData struct {
	ID              string     `json:"id"`
	ProjectID       int64      `json:"project_id"`
	Email           string     `json:"email"`
	Role            string     `json:"role"`
	InvitedByUserID string     `json:"invited_by_user_id"`
	ExpiresAt       time.Time  `json:"expires_at"`
	AcceptedAt      *time.Time `json:"accepted_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type ProjectInvitationLookupData struct {
	ID          string    `json:"id"`
	ProjectID   int64     `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Email       string    `json:"email"`
	Role        string    `json:"role"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type MetricValueData struct {
	Value      float64   `json:"value"`
	Unit       string    `json:"unit"`
	ObservedAt time.Time `json:"observed_at"`
}

type HealthCheckSummaryData struct {
	ObservedAt     time.Time `json:"observed_at"`
	IsSuccess      bool      `json:"is_success"`
	StatusCode     *int32    `json:"status_code"`
	ResponseTimeMs *int32    `json:"response_time_ms"`
	Message        string    `json:"message"`
}

type AgentConfigNodeData struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type AgentConfigCheckData struct {
	ServiceID       int64  `json:"service_id"`
	Name            string `json:"name"`
	CheckType       string `json:"check_type"`
	CheckTarget     string `json:"check_target"`
	IntervalSeconds int    `json:"interval_seconds"`
	TimeoutSeconds  int    `json:"timeout_seconds"`
}

type AgentConfigData struct {
	Node         AgentConfigNodeData    `json:"node"`
	HealthChecks []AgentConfigCheckData `json:"health_checks"`
}

type NodeReadData struct {
	ID               int64            `json:"id"`
	ProjectID        int64            `json:"project_id"`
	Name             string           `json:"name"`
	Identifier       string           `json:"identifier"`
	CurrentState     string           `json:"current_state"`
	LastSeenAt       *time.Time       `json:"last_seen_at"`
	Metadata         json.RawMessage  `json:"metadata"`
	LatestCPU        *MetricValueData `json:"latest_cpu,omitempty"`
	LatestMemory     *MetricValueData `json:"latest_memory,omitempty"`
	LatestDisk       *MetricValueData `json:"latest_disk,omitempty"`
	ActiveAlertCount int64            `json:"active_alert_count"`
	ActiveAlerts     []AlertReadData  `json:"active_alerts,omitempty"`
	ServiceCount     int64            `json:"service_count"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
}

type ServiceReadData struct {
	ID                  int64                   `json:"id"`
	ProjectID           int64                   `json:"project_id"`
	NodeID              int64                   `json:"node_id"`
	ExecutionMode       string                  `json:"execution_mode"`
	Name                string                  `json:"name"`
	CheckType           string                  `json:"check_type"`
	CheckTarget         string                  `json:"check_target"`
	Status              string                  `json:"status"`
	LastCheckedAt       *time.Time              `json:"last_checked_at"`
	ConsecutiveFailures int32                   `json:"consecutive_failures"`
	ActiveAlertCount    int64                   `json:"active_alert_count"`
	LatestHealthCheck   *HealthCheckSummaryData `json:"latest_health_check"`
}

type ServiceActiveAlertData struct {
	ID          int64     `json:"id"`
	RuleID      int64     `json:"rule_id"`
	RuleType    string    `json:"rule_type"`
	Severity    string    `json:"severity"`
	Title       string    `json:"title"`
	Status      string    `json:"status"`
	TriggeredAt time.Time `json:"triggered_at"`
}

type ServiceDetailData struct {
	ID                  int64                    `json:"id"`
	ProjectID           int64                    `json:"project_id"`
	NodeID              int64                    `json:"node_id"`
	ExecutionMode       string                   `json:"execution_mode"`
	Name                string                   `json:"name"`
	CheckType           string                   `json:"check_type"`
	CheckTarget         string                   `json:"check_target"`
	Status              string                   `json:"status"`
	LastCheckedAt       *time.Time               `json:"last_checked_at"`
	ConsecutiveFailures int32                    `json:"consecutive_failures"`
	ActiveAlertCount    int64                    `json:"active_alert_count"`
	ActiveAlerts        []ServiceActiveAlertData `json:"active_alerts"`
	LatestHealthCheck   *HealthCheckSummaryData  `json:"latest_health_check"`
}

type ServiceListFilters struct {
	ProjectID *int64
	NodeID    *int64
	Status    *string
	Limit     *int32
}

type NodeResponseData struct {
	ID              int64           `json:"id"`
	ProjectID       int64           `json:"project_id"`
	Name            string          `json:"name"`
	Identifier      string          `json:"identifier"`
	CurrentState    string          `json:"current_state"`
	LastHeartbeatAt *time.Time      `json:"last_heartbeat_at"`
	Metadata        json.RawMessage `json:"metadata"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type ServiceResponseData struct {
	ID                   int64      `json:"id"`
	ProjectID            int64      `json:"project_id"`
	NodeID               int64      `json:"node_id"`
	Name                 string     `json:"name"`
	CheckType            string     `json:"check_type"`
	CheckTarget          string     `json:"check_target"`
	CurrentState         string     `json:"current_state"`
	ConsecutiveFailures  int32      `json:"consecutive_failures"`
	ConsecutiveSuccesses int32      `json:"consecutive_successes"`
	LastCheckAt          *time.Time `json:"last_check_at"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type EventReadData struct {
	ID         int64     `json:"id"`
	ProjectID  int64     `json:"project_id"`
	NodeID     *int64    `json:"node_id"`
	ServiceID  *int64    `json:"service_id"`
	EventType  string    `json:"event_type"`
	Severity   string    `json:"severity"`
	Title      string    `json:"title"`
	Details    any       `json:"details"`
	OccurredAt time.Time `json:"occurred_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type AlertReadData struct {
	ID          int64      `json:"id"`
	AlertRuleID int64      `json:"alert_rule_id"`
	ProjectID   int64      `json:"project_id"`
	NodeID      *int64     `json:"node_id"`
	ServiceID   *int64     `json:"service_id"`
	Status      string     `json:"status"`
	TriggeredAt time.Time  `json:"triggered_at"`
	ResolvedAt  *time.Time `json:"resolved_at"`
	Title       string     `json:"title"`
	Message     string     `json:"message"`
	CreatedAt   time.Time  `json:"created_at"`
}

type AlertRuleReadData struct {
	ID             int64     `json:"id"`
	ProjectID      int64     `json:"project_id"`
	NodeID         *int64    `json:"node_id"`
	ServiceID      *int64    `json:"service_id"`
	RuleType       string    `json:"rule_type"`
	Severity       string    `json:"severity"`
	MetricName     *string   `json:"metric_name"`
	ThresholdValue *float64  `json:"threshold_value"`
	IsEnabled      bool      `json:"is_enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type NotificationRecipientReadData struct {
	ID          int64     `json:"id"`
	ProjectID   int64     `json:"project_id"`
	ChannelType string    `json:"channel_type"`
	Target      string    `json:"target"`
	MinSeverity string    `json:"min_severity"`
	IsEnabled   bool      `json:"is_enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type NotificationDeliveryReadData struct {
	ID                      int64     `json:"id"`
	ProjectID               int64     `json:"project_id"`
	NotificationRecipientID *int64    `json:"notification_recipient_id"`
	AlertInstanceID         *int64    `json:"alert_instance_id"`
	ChannelType             string    `json:"channel_type"`
	Target                  string    `json:"target"`
	EventType               string    `json:"event_type"`
	Status                  string    `json:"status"`
	ErrorMessage            *string   `json:"error_message"`
	SentAt                  time.Time `json:"sent_at"`
	CreatedAt               time.Time `json:"created_at"`
}

type NodeSummaryData struct {
	ID               int64            `json:"id"`
	ProjectID        int64            `json:"project_id"`
	Name             string           `json:"name"`
	Identifier       string           `json:"identifier"`
	CurrentState     string           `json:"current_state"`
	LastSeenAt       *time.Time       `json:"last_seen_at"`
	LatestCPU        *MetricValueData `json:"latest_cpu,omitempty"`
	LatestMemory     *MetricValueData `json:"latest_memory,omitempty"`
	LatestDisk       *MetricValueData `json:"latest_disk,omitempty"`
	ActiveAlertCount int64            `json:"active_alert_count"`
	ServiceCount     int64            `json:"service_count"`
}

type OverviewData struct {
	TotalProjects     int64             `json:"total_projects"`
	TotalNodes        int64             `json:"total_nodes"`
	NodesOnline       int64             `json:"nodes_online"`
	NodesOffline      int64             `json:"nodes_offline"`
	TotalServices     int64             `json:"total_services"`
	ServicesHealthy   int64             `json:"services_healthy"`
	ServicesDegraded  int64             `json:"services_degraded"`
	ServicesUnhealthy int64             `json:"services_unhealthy"`
	ActiveAlerts      int64             `json:"active_alerts"`
	ResolvedAlerts    int64             `json:"resolved_alerts"`
	RecentEvents      []EventReadData   `json:"recent_events"`
	RecentAlertEvents []EventReadData   `json:"recent_alert_events"`
	NodeSummaries     []NodeSummaryData `json:"node_summaries"`
}
