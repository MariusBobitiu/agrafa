package types

import "time"

type ErrorResponse struct {
	Error string `json:"error" example:"invalid request"`
}

type NullableTimeResponse struct {
	Time  time.Time `json:"Time" swaggertype:"string" format:"date-time"`
	Valid bool      `json:"Valid" example:"true"`
}

type NullableInt64Response struct {
	Int64 int64 `json:"Int64" example:"1"`
	Valid bool  `json:"Valid" example:"true"`
}

type NullableInt32Response struct {
	Int32 int32 `json:"Int32" example:"200"`
	Valid bool  `json:"Valid" example:"true"`
}

type NullableStringResponse struct {
	String string `json:"String" example:"cpu_usage"`
	Valid  bool   `json:"Valid" example:"true"`
}

type NullableFloat64Response struct {
	Float64 float64 `json:"Float64" example:"80"`
	Valid   bool    `json:"Valid" example:"true"`
}

type MetricValueDocument struct {
	Value      float64   `json:"value" example:"84.2"`
	Unit       string    `json:"unit" example:"percent"`
	ObservedAt time.Time `json:"observed_at" swaggertype:"string" format:"date-time"`
}

type HealthCheckSummaryDocument struct {
	ObservedAt     time.Time `json:"observed_at" swaggertype:"string" format:"date-time"`
	IsSuccess      bool      `json:"is_success" example:"true"`
	StatusCode     *int32    `json:"status_code" example:"200"`
	ResponseTimeMs *int32    `json:"response_time_ms" example:"42"`
	Message        string    `json:"message" example:"ok"`
}

type AgentConfigNodeDocument struct {
	ID         int64  `json:"id" example:"12"`
	Name       string `json:"name" example:"web-01"`
	Identifier string `json:"identifier" example:"web-01"`
}

type AgentConfigCheckDocument struct {
	ServiceID       int64  `json:"service_id" example:"101"`
	Name            string `json:"name" example:"internal-api"`
	CheckType       string `json:"check_type" example:"http"`
	CheckTarget     string `json:"check_target" example:"http://internal-api.local/health"`
	IntervalSeconds int    `json:"interval_seconds" example:"30"`
	TimeoutSeconds  int    `json:"timeout_seconds" example:"10"`
}

type AgentConfigResponseDocument struct {
	Node         AgentConfigNodeDocument    `json:"node"`
	HealthChecks []AgentConfigCheckDocument `json:"health_checks"`
}

type ProjectDocument struct {
	ID        int64     `json:"id" example:"1"`
	Slug      string    `json:"slug" example:"my-project"`
	Name      string    `json:"name" example:"My Project"`
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
}

type ProjectSummaryDocument struct {
	ID              int64     `json:"id" example:"1"`
	Slug            string    `json:"slug" example:"my-project"`
	Name            string    `json:"name" example:"My Project"`
	CreatedAt       time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	CurrentUserRole string    `json:"current_user_role" example:"owner"`
}

type ProjectDetailDocument struct {
	ID               int64     `json:"id" example:"1"`
	Slug             string    `json:"slug" example:"my-project"`
	Name             string    `json:"name" example:"My Project"`
	CreatedAt        time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	CurrentUserRole  string    `json:"current_user_role" example:"owner"`
	NodeCount        int64     `json:"node_count" example:"3"`
	ServiceCount     int64     `json:"service_count" example:"7"`
	ActiveAlertCount int64     `json:"active_alert_count" example:"1"`
}

type UserDocument struct {
	ID                  string    `json:"id" example:"usr_0123456789abcdef"`
	Name                string    `json:"name" example:"Alice Example"`
	Email               string    `json:"email" example:"alice@example.com"`
	EmailVerified       bool      `json:"email_verified" example:"false"`
	Image               *string   `json:"image,omitempty" example:"https://example.com/avatar.png"`
	OnboardingCompleted bool      `json:"onboarding_completed" example:"false"`
	TwoFactorEnabled    bool      `json:"two_factor_enabled" example:"false"`
	CreatedAt           time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt           time.Time `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type AuthUserSessionDocument struct {
	ID        string    `json:"id" example:"ses_0123456789abcdef"`
	ExpiresAt time.Time `json:"expires_at" swaggertype:"string" format:"date-time"`
	IPAddress *string   `json:"ip_address,omitempty" example:"203.0.113.10"`
	UserAgent *string   `json:"user_agent,omitempty" example:"Mozilla/5.0"`
	CreatedAt time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt time.Time `json:"updated_at" swaggertype:"string" format:"date-time"`
	IsCurrent bool      `json:"is_current" example:"true"`
}

type ProjectMemberUserSummaryDocument struct {
	Name  string  `json:"name" example:"Alice Example"`
	Email string  `json:"email" example:"alice@example.com"`
	Image *string `json:"image,omitempty" example:"https://example.com/avatar.png"`
}

type ProjectMemberDocument struct {
	ID        string                           `json:"id" example:"pm_0123456789abcdef"`
	ProjectID int64                            `json:"project_id" example:"1"`
	UserID    string                           `json:"user_id" example:"usr_0123456789abcdef"`
	Role      string                           `json:"role" example:"viewer"`
	CreatedAt time.Time                        `json:"created_at" swaggertype:"string" format:"date-time"`
	User      ProjectMemberUserSummaryDocument `json:"user"`
}

type ProjectInvitationDocument struct {
	ID              string     `json:"id" example:"pinv_0123456789abcdef"`
	ProjectID       int64      `json:"project_id" example:"1"`
	Email           string     `json:"email" example:"teammate@example.com"`
	Role            string     `json:"role" example:"viewer"`
	InvitedByUserID string     `json:"invited_by_user_id" example:"usr_0123456789abcdef"`
	ExpiresAt       time.Time  `json:"expires_at" swaggertype:"string" format:"date-time"`
	AcceptedAt      *time.Time `json:"accepted_at" swaggertype:"string" format:"date-time"`
	CreatedAt       time.Time  `json:"created_at" swaggertype:"string" format:"date-time"`
}

type ProjectInvitationLookupDocument struct {
	ID          string    `json:"id" example:"pinv_0123456789abcdef"`
	ProjectID   int64     `json:"project_id" example:"1"`
	ProjectName string    `json:"project_name" example:"Agrafa Team"`
	Email       string    `json:"email" example:"teammate@example.com"`
	Role        string    `json:"role" example:"viewer"`
	ExpiresAt   time.Time `json:"expires_at" swaggertype:"string" format:"date-time"`
}

type NodeDocument struct {
	ID               int64                `json:"id" example:"1"`
	ProjectID        int64                `json:"project_id" example:"1"`
	Name             string               `json:"name" example:"hetzner-01"`
	Identifier       string               `json:"identifier" example:"hetzner-01"`
	CurrentState     string               `json:"current_state" example:"offline"`
	LastSeenAt       *time.Time           `json:"last_seen_at" swaggertype:"string" format:"date-time"`
	Metadata         map[string]any       `json:"metadata" swaggertype:"object"`
	LatestCPU        *MetricValueDocument `json:"latest_cpu,omitempty"`
	LatestMemory     *MetricValueDocument `json:"latest_memory,omitempty"`
	LatestDisk       *MetricValueDocument `json:"latest_disk,omitempty"`
	ActiveAlertCount int64                `json:"active_alert_count" example:"1"`
	ActiveAlerts     []AlertDocument      `json:"active_alerts,omitempty"`
	ServiceCount     int64                `json:"service_count" example:"2"`
	CreatedAt        time.Time            `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt        time.Time            `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type ServiceDocument struct {
	ID                  int64                       `json:"id" example:"1"`
	ProjectID           int64                       `json:"project_id" example:"1"`
	NodeID              int64                       `json:"node_id" example:"1"`
	ExecutionMode       string                      `json:"execution_mode" example:"agent"`
	Name                string                      `json:"name" example:"planzi-api"`
	CheckType           string                      `json:"check_type" example:"http"`
	CheckTarget         string                      `json:"check_target" example:"https://api.planzi.app/health"`
	Status              string                      `json:"status" example:"healthy"`
	LastCheckedAt       *time.Time                  `json:"last_checked_at" swaggertype:"string" format:"date-time"`
	ConsecutiveFailures int32                       `json:"consecutive_failures" example:"0"`
	ActiveAlertCount    int64                       `json:"active_alert_count" example:"1"`
	LatestHealthCheck   *HealthCheckSummaryDocument `json:"latest_health_check"`
}

type NodeResponseDocument struct {
	ID              int64          `json:"id" example:"1"`
	ProjectID       int64          `json:"project_id" example:"1"`
	Name            string         `json:"name" example:"hetzner-01"`
	Identifier      string         `json:"identifier" example:"hetzner-01"`
	CurrentState    string         `json:"current_state" example:"online"`
	LastHeartbeatAt *time.Time     `json:"last_heartbeat_at" swaggertype:"string" format:"date-time"`
	Metadata        map[string]any `json:"metadata" swaggertype:"object"`
	CreatedAt       time.Time      `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt       time.Time      `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type NodeAgentTokenResponseDocument struct {
	NodeID     int64  `json:"node_id" example:"1"`
	AgentToken string `json:"agent_token" example:"eHh4YWFhYmJiY2NjZGRkZWVlZmZmZ2dn"`
}

type ServiceResponseDocument struct {
	ID                   int64      `json:"id" example:"1"`
	ProjectID            int64      `json:"project_id" example:"1"`
	NodeID               int64      `json:"node_id" example:"1"`
	Name                 string     `json:"name" example:"planzi-api"`
	CheckType            string     `json:"check_type" example:"http"`
	CheckTarget          string     `json:"check_target" example:"https://api.planzi.app/health"`
	CurrentState         string     `json:"current_state" example:"healthy"`
	ConsecutiveFailures  int32      `json:"consecutive_failures" example:"0"`
	ConsecutiveSuccesses int32      `json:"consecutive_successes" example:"0"`
	LastCheckAt          *time.Time `json:"last_check_at" swaggertype:"string" format:"date-time"`
	CreatedAt            time.Time  `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt            time.Time  `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type EventDocument struct {
	ID         int64          `json:"id" example:"1"`
	ProjectID  int64          `json:"project_id" example:"1"`
	NodeID     *int64         `json:"node_id" example:"1"`
	ServiceID  *int64         `json:"service_id" example:"1"`
	EventType  string         `json:"event_type" example:"node_online"`
	Severity   string         `json:"severity" example:"info"`
	Title      string         `json:"title" example:"Node hetzner-01 is online"`
	Details    map[string]any `json:"details" swaggertype:"object"`
	OccurredAt time.Time      `json:"occurred_at" swaggertype:"string" format:"date-time"`
	CreatedAt  time.Time      `json:"created_at" swaggertype:"string" format:"date-time"`
}

type AlertRuleDocument struct {
	ID             int64     `json:"id" example:"1"`
	ProjectID      int64     `json:"project_id" example:"1"`
	NodeID         *int64    `json:"node_id" example:"1"`
	ServiceID      *int64    `json:"service_id" example:"1"`
	RuleType       string    `json:"rule_type" example:"node_offline"`
	MetricName     *string   `json:"metric_name" example:"cpu_usage"`
	ThresholdValue *float64  `json:"threshold_value" example:"80"`
	IsEnabled      bool      `json:"is_enabled" example:"true"`
	CreatedAt      time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt      time.Time `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type AlertDocument struct {
	ID          int64      `json:"id" example:"1"`
	AlertRuleID int64      `json:"alert_rule_id" example:"1"`
	ProjectID   int64      `json:"project_id" example:"1"`
	NodeID      *int64     `json:"node_id" example:"1"`
	ServiceID   *int64     `json:"service_id" example:"1"`
	Status      string     `json:"status" example:"active"`
	TriggeredAt time.Time  `json:"triggered_at" swaggertype:"string" format:"date-time"`
	ResolvedAt  *time.Time `json:"resolved_at" swaggertype:"string" format:"date-time"`
	Title       string     `json:"title" example:"Node 1 is offline"`
	Message     string     `json:"message" example:"Node 1 is currently offline."`
	CreatedAt   time.Time  `json:"created_at" swaggertype:"string" format:"date-time"`
}

type NotificationRecipientDocument struct {
	ID          int64     `json:"id" example:"1"`
	ProjectID   int64     `json:"project_id" example:"1"`
	ChannelType string    `json:"channel_type" example:"email"`
	Target      string    `json:"target" example:"ops@example.com"`
	IsEnabled   bool      `json:"is_enabled" example:"true"`
	CreatedAt   time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
	UpdatedAt   time.Time `json:"updated_at" swaggertype:"string" format:"date-time"`
}

type NotificationDeliveryDocument struct {
	ID                      int64     `json:"id" example:"1"`
	ProjectID               int64     `json:"project_id" example:"1"`
	NotificationRecipientID *int64    `json:"notification_recipient_id" example:"1"`
	AlertInstanceID         *int64    `json:"alert_instance_id" example:"1"`
	ChannelType             string    `json:"channel_type" example:"email"`
	Target                  string    `json:"target" example:"ops@example.com"`
	EventType               string    `json:"event_type" example:"alert_triggered"`
	Status                  string    `json:"status" example:"sent"`
	ErrorMessage            *string   `json:"error_message" example:"resend returned status 500"`
	SentAt                  time.Time `json:"sent_at" swaggertype:"string" format:"date-time"`
	CreatedAt               time.Time `json:"created_at" swaggertype:"string" format:"date-time"`
}

type NodeSummaryDocument struct {
	ID               int64                `json:"id" example:"1"`
	ProjectID        int64                `json:"project_id" example:"1"`
	Name             string               `json:"name" example:"hetzner-01"`
	Identifier       string               `json:"identifier" example:"hetzner-01"`
	CurrentState     string               `json:"current_state" example:"online"`
	LastSeenAt       *time.Time           `json:"last_seen_at" swaggertype:"string" format:"date-time"`
	LatestCPU        *MetricValueDocument `json:"latest_cpu,omitempty"`
	LatestMemory     *MetricValueDocument `json:"latest_memory,omitempty"`
	LatestDisk       *MetricValueDocument `json:"latest_disk,omitempty"`
	ActiveAlertCount int64                `json:"active_alert_count" example:"0"`
	ServiceCount     int64                `json:"service_count" example:"2"`
}

type HeartbeatRequest struct {
	NodeID     *int64         `json:"node_id,omitempty" example:"1"`
	ObservedAt *time.Time     `json:"observed_at,omitempty" swaggertype:"string" format:"date-time"`
	Source     string         `json:"source" example:"agent"`
	Payload    map[string]any `json:"payload" swaggertype:"object"`
}

type AgentShutdownRequest struct {
	NodeID     *int64         `json:"node_id,omitempty" example:"1"`
	ObservedAt *time.Time     `json:"observed_at,omitempty" swaggertype:"string" format:"date-time"`
	Reason     string         `json:"reason" example:"user_closed"`
	Payload    map[string]any `json:"payload" swaggertype:"object"`
}

type HealthRequest struct {
	ServiceID      int64          `json:"service_id" example:"1"`
	ObservedAt     *time.Time     `json:"observed_at,omitempty" swaggertype:"string" format:"date-time"`
	IsSuccess      *bool          `json:"is_success" example:"true"`
	StatusCode     *int32         `json:"status_code,omitempty" example:"200"`
	ResponseTimeMs *int32         `json:"response_time_ms,omitempty" example:"42"`
	Message        string         `json:"message" example:"ok"`
	Payload        map[string]any `json:"payload" swaggertype:"object"`
}

type MetricSampleRequest struct {
	MetricName  string         `json:"metric_name" example:"cpu_usage"`
	Value       *float64       `json:"value,omitempty" example:"42.5"`
	MetricValue *float64       `json:"metric_value,omitempty" example:"42.5"`
	MetricUnit  string         `json:"metric_unit" example:"percent"`
	ObservedAt  time.Time      `json:"observed_at,omitempty" swaggertype:"string" format:"date-time"`
	Payload     map[string]any `json:"payload" swaggertype:"object"`
}

type MetricsRequest struct {
	NodeID    *int64                `json:"node_id,omitempty" example:"1"`
	ServiceID *int64                `json:"service_id,omitempty" example:"1"`
	Samples   []MetricSampleRequest `json:"samples"`
}

type ProjectCreateRequest struct {
	Name string `json:"name" example:"My Project"`
}

type ProjectUpdateRequest struct {
	Name *string `json:"name,omitempty" example:"My Renamed Project"`
}

type AuthRegisterRequest struct {
	Name     string `json:"name" example:"Alice Example"`
	Email    string `json:"email" example:"alice@example.com"`
	Password string `json:"password" example:"supersecretpassword"`
}

type AuthLoginRequest struct {
	Email      string `json:"email" example:"alice@example.com"`
	Password   string `json:"password" example:"supersecretpassword"`
	RememberMe bool   `json:"remember_me" example:"true"`
}

type AuthVerifyEmailConfirmRequest struct {
	Token string `json:"token" example:"raw-verification-token"`
}

type AuthForgotPasswordRequest struct {
	Email string `json:"email" example:"alice@example.com"`
}

type AuthResetPasswordRequest struct {
	Token    string `json:"token" example:"raw-reset-token"`
	Password string `json:"password" example:"new-supersecretpassword"`
}

type AuthVerifyPasswordRequest struct {
	Password string `json:"password" example:"current-supersecretpassword"`
}

type ProjectInvitationCreateRequest struct {
	ProjectID   int64                                `json:"project_id" example:"1"`
	Email       string                               `json:"email,omitempty" example:"teammate@example.com"`
	Role        string                               `json:"role,omitempty" example:"viewer"`
	Invitations []ProjectInvitationCreateItemRequest `json:"invitations,omitempty"`
}

type ProjectInvitationCreateItemRequest struct {
	Email string `json:"email" example:"teammate@example.com"`
	Role  string `json:"role" example:"viewer"`
}

type ProjectInvitationCreateResultDocument struct {
	Email        string                     `json:"email" example:"teammate@example.com"`
	Role         string                     `json:"role" example:"viewer"`
	Status       string                     `json:"status" example:"created"`
	Invitation   *ProjectInvitationDocument `json:"invitation,omitempty"`
	ErrorCode    *string                    `json:"error_code,omitempty" example:"already_invited"`
	ErrorMessage *string                    `json:"error_message,omitempty" example:"An active invitation already exists for this email."`
}

type ProjectInvitationAcceptRequest struct {
	Token string `json:"token" example:"raw-project-invitation-token"`
}

type NodeCreateRequest struct {
	ProjectID int64  `json:"project_id" example:"1"`
	Name      string `json:"name" example:"hetzner-01"`
}

type NodeUpdateRequest struct {
	Name       *string `json:"name,omitempty" example:"hetzner-02"`
	Identifier *string `json:"identifier,omitempty" example:"hetzner-02"`
}

type ServiceCreateRequest struct {
	ProjectID     int64  `json:"project_id" example:"1"`
	ExecutionMode string `json:"execution_mode" example:"managed"`
	NodeID        *int64 `json:"node_id,omitempty" example:"1"`
	Name          string `json:"name" example:"planzi-api"`
	CheckType     string `json:"check_type" example:"http"`
	CheckTarget   string `json:"check_target" example:"https://api.planzi.app/health"`
}

type ServiceUpdateRequest struct {
	Name        *string `json:"name,omitempty" example:"planzi-api-v2"`
	CheckType   *string `json:"check_type,omitempty" example:"http"`
	CheckTarget *string `json:"check_target,omitempty" example:"https://api.planzi.app/status"`
}

type AlertRuleCreateRequest struct {
	ProjectID      int64    `json:"project_id" example:"1"`
	NodeID         *int64   `json:"node_id,omitempty" example:"1"`
	ServiceID      *int64   `json:"service_id,omitempty" example:"1"`
	RuleType       string   `json:"rule_type" example:"cpu_above_threshold"`
	ThresholdValue *float64 `json:"threshold_value,omitempty" example:"80"`
}

type AlertRuleUpdateRequest struct {
	IsEnabled *bool `json:"is_enabled" example:"false"`
}

type NotificationRecipientCreateRequest struct {
	ProjectID   int64  `json:"project_id" example:"1"`
	ChannelType string `json:"channel_type" example:"email"`
	Target      string `json:"target" example:"ops@example.com"`
}

type NotificationRecipientUpdateRequest struct {
	IsEnabled *bool `json:"is_enabled" example:"false"`
}

type ProjectMemberCreateRequest struct {
	ProjectID int64  `json:"project_id" example:"1"`
	UserID    string `json:"user_id" example:"usr_0123456789abcdef"`
	Role      string `json:"role" example:"viewer"`
}

type ProjectMemberUpdateRequest struct {
	Role string `json:"role" example:"admin"`
}

type HeartbeatResponse struct {
	Status string               `json:"status" example:"ok"`
	Node   NodeResponseDocument `json:"node"`
}

type HealthResponse struct {
	Status  string                  `json:"status" example:"ok"`
	Service ServiceResponseDocument `json:"service"`
}

type MetricsResponse struct {
	Status string `json:"status" example:"ok"`
}

type ProjectResponse struct {
	Project ProjectDocument `json:"project"`
}

type ProjectsResponse struct {
	Projects []ProjectSummaryDocument `json:"projects"`
}

type ProjectDetailResponse struct {
	Project ProjectDetailDocument `json:"project"`
}

type AuthSessionResponse struct {
	User      UserDocument `json:"user"`
	ExpiresAt time.Time    `json:"expires_at" swaggertype:"string" format:"date-time"`
}

type AuthMeResponse struct {
	User UserDocument `json:"user"`
}

type AuthLogoutResponse struct {
	Status string `json:"status" example:"ok"`
}

type AuthSessionsResponse struct {
	Sessions []AuthUserSessionDocument `json:"sessions"`
}

type NodeResponse struct {
	Node NodeResponseDocument `json:"node"`
}

type NodeDetailResponse struct {
	Node NodeDocument `json:"node"`
}

type NodeAgentTokenResponse struct {
	NodeID     int64  `json:"node_id" example:"1"`
	AgentToken string `json:"agent_token" example:"eHh4YWFhYmJiY2NjZGRkZWVlZmZmZ2dn"`
}

type ServiceResponse struct {
	Service ServiceResponseDocument `json:"service"`
}

type ServiceDetailResponse struct {
	Service ServiceDocument `json:"service"`
}

type AlertRuleResponse struct {
	AlertRule AlertRuleDocument `json:"alert_rule"`
}

type NotificationRecipientResponse struct {
	NotificationRecipient NotificationRecipientDocument `json:"notification_recipient"`
}

type ProjectMemberResponse struct {
	ProjectMember ProjectMemberDocument `json:"project_member"`
}

type ProjectInvitationResponse struct {
	ProjectInvitation ProjectInvitationDocument `json:"project_invitation"`
}

type ProjectInvitationCreateResponse struct {
	ProjectID int64                                   `json:"project_id" example:"1"`
	Results   []ProjectInvitationCreateResultDocument `json:"results"`
}

type ProjectInvitationLookupResponse struct {
	ProjectInvitation ProjectInvitationLookupDocument `json:"project_invitation"`
}

type NodesResponse struct {
	Nodes []NodeDocument `json:"nodes"`
}

type ServicesResponse struct {
	Services []ServiceDocument `json:"services"`
}

type EventsResponse struct {
	Events []EventDocument `json:"events"`
}

type AlertRulesResponse struct {
	AlertRules []AlertRuleDocument `json:"alert_rules"`
}

type AlertsResponse struct {
	Alerts []AlertDocument `json:"alerts"`
}

type NotificationRecipientsResponse struct {
	NotificationRecipients []NotificationRecipientDocument `json:"notification_recipients"`
}

type NotificationDeliveriesResponse struct {
	NotificationDeliveries []NotificationDeliveryDocument `json:"notification_deliveries"`
}

type ProjectMembersResponse struct {
	ProjectMembers []ProjectMemberDocument `json:"project_members"`
}

type ProjectInvitationsResponse struct {
	ProjectInvitations []ProjectInvitationDocument `json:"project_invitations"`
}

type ProjectInvitationAcceptResponse struct {
	Status        string `json:"status" example:"ok"`
	AlreadyMember bool   `json:"already_member" example:"false"`
}

type OverviewResponse struct {
	TotalProjects     int64                 `json:"total_projects" example:"1"`
	TotalNodes        int64                 `json:"total_nodes" example:"1"`
	NodesOnline       int64                 `json:"nodes_online" example:"1"`
	NodesOffline      int64                 `json:"nodes_offline" example:"0"`
	TotalServices     int64                 `json:"total_services" example:"1"`
	ServicesHealthy   int64                 `json:"services_healthy" example:"1"`
	ServicesDegraded  int64                 `json:"services_degraded" example:"0"`
	ServicesUnhealthy int64                 `json:"services_unhealthy" example:"0"`
	ActiveAlerts      int64                 `json:"active_alerts" example:"1"`
	ResolvedAlerts    int64                 `json:"resolved_alerts" example:"3"`
	RecentEvents      []EventDocument       `json:"recent_events"`
	RecentAlertEvents []EventDocument       `json:"recent_alert_events"`
	NodeSummaries     []NodeSummaryDocument `json:"node_summaries"`
}
