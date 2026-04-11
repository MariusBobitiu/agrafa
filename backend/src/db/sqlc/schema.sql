CREATE SCHEMA IF NOT EXISTS app;
CREATE SCHEMA IF NOT EXISTS auth;
CREATE SCHEMA IF NOT EXISTS agrafa_meta;

CREATE TABLE auth.users (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    image TEXT,
    onboarding_completed BOOLEAN NOT NULL DEFAULT FALSE,
    two_factor_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth.password_credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id)
);

CREATE TABLE auth.sessions (
    id TEXT PRIMARY KEY,
    token_hash TEXT NOT NULL UNIQUE,
    user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auth.verification_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES auth.users(id) ON DELETE CASCADE,
    identifier TEXT NOT NULL,
    token_hash TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.projects (
    id BIGSERIAL PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.nodes (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    identifier TEXT NOT NULL,
    agent_token_hash TEXT,
    node_type TEXT NOT NULL DEFAULT 'agent' CHECK (node_type IN ('managed', 'agent')),
    is_visible BOOLEAN NOT NULL DEFAULT TRUE,
    current_state TEXT NOT NULL DEFAULT 'offline' CHECK (current_state IN ('online', 'offline')),
    last_heartbeat_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, identifier)
);

CREATE TABLE app.services (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    node_id BIGINT NOT NULL REFERENCES app.nodes(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    check_type TEXT NOT NULL,
    check_target TEXT NOT NULL,
    current_state TEXT NOT NULL DEFAULT 'healthy' CHECK (current_state IN ('healthy', 'degraded', 'unhealthy')),
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    consecutive_successes INTEGER NOT NULL DEFAULT 0,
    last_check_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, node_id, name)
);

CREATE TABLE app.heartbeats (
    id BIGSERIAL PRIMARY KEY,
    node_id BIGINT NOT NULL REFERENCES app.nodes(id) ON DELETE CASCADE,
    observed_at TIMESTAMPTZ NOT NULL,
    source TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.health_check_results (
    id BIGSERIAL PRIMARY KEY,
    service_id BIGINT NOT NULL REFERENCES app.services(id) ON DELETE CASCADE,
    observed_at TIMESTAMPTZ NOT NULL,
    is_success BOOLEAN NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    message TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.metric_samples (
    id BIGSERIAL PRIMARY KEY,
    node_id BIGINT NOT NULL REFERENCES app.nodes(id) ON DELETE CASCADE,
    service_id BIGINT REFERENCES app.services(id) ON DELETE CASCADE,
    metric_name TEXT NOT NULL,
    metric_value DOUBLE PRECISION NOT NULL,
    metric_unit TEXT NOT NULL DEFAULT '',
    observed_at TIMESTAMPTZ NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.events (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    node_id BIGINT REFERENCES app.nodes(id) ON DELETE CASCADE,
    service_id BIGINT REFERENCES app.services(id) ON DELETE CASCADE,
    event_type TEXT NOT NULL,
    severity TEXT NOT NULL,
    title TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.alert_rules (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    node_id BIGINT REFERENCES app.nodes(id) ON DELETE CASCADE,
    service_id BIGINT REFERENCES app.services(id) ON DELETE CASCADE,
    rule_type TEXT NOT NULL CHECK (
        rule_type IN (
            'node_offline',
            'service_unhealthy',
            'cpu_above_threshold',
            'memory_above_threshold',
            'disk_above_threshold'
        )
    ),
    severity TEXT NOT NULL CHECK (
        severity IN ('info', 'warning', 'critical')
    ),
    metric_name TEXT,
    threshold_value DOUBLE PRECISION,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (
            rule_type = 'node_offline'
            AND node_id IS NOT NULL
            AND service_id IS NULL
            AND metric_name IS NULL
            AND threshold_value IS NULL
        )
        OR (
            rule_type = 'service_unhealthy'
            AND node_id IS NULL
            AND service_id IS NOT NULL
            AND metric_name IS NULL
            AND threshold_value IS NULL
        )
        OR (
            rule_type = 'cpu_above_threshold'
            AND node_id IS NOT NULL
            AND service_id IS NULL
            AND metric_name = 'cpu_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
        OR (
            rule_type = 'memory_above_threshold'
            AND node_id IS NOT NULL
            AND service_id IS NULL
            AND metric_name = 'memory_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
        OR (
            rule_type = 'disk_above_threshold'
            AND node_id IS NOT NULL
            AND service_id IS NULL
            AND metric_name = 'disk_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
    )
);

CREATE TABLE app.alert_instances (
    id BIGSERIAL PRIMARY KEY,
    alert_rule_id BIGINT NOT NULL REFERENCES app.alert_rules(id) ON DELETE CASCADE,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    node_id BIGINT REFERENCES app.nodes(id) ON DELETE CASCADE,
    service_id BIGINT REFERENCES app.services(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK (status IN ('active', 'resolved')),
    triggered_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    title TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (status = 'active' AND resolved_at IS NULL)
        OR (status = 'resolved' AND resolved_at IS NOT NULL)
    )
);

CREATE TABLE app.notification_recipients (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('email')),
    target TEXT NOT NULL,
    min_severity TEXT NOT NULL CHECK (
        min_severity IN ('info', 'warning', 'critical')
    ),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, channel_type, target)
);

CREATE TABLE app.notification_deliveries (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    notification_recipient_id BIGINT REFERENCES app.notification_recipients(id) ON DELETE SET NULL,
    alert_instance_id BIGINT REFERENCES app.alert_instances(id) ON DELETE SET NULL,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('email')),
    target TEXT NOT NULL,
    event_type TEXT NOT NULL CHECK (event_type IN ('alert_triggered', 'alert_resolved')),
    status TEXT NOT NULL CHECK (status IN ('sent', 'failed')),
    error_message TEXT,
    sent_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.project_invitations (
    id TEXT PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    email TEXT NOT NULL,
    role TEXT NOT NULL CHECK (role IN ('admin', 'viewer')),
    token_hash TEXT NOT NULL UNIQUE,
    invited_by_user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE app.project_members (
    id TEXT PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    role TEXT NOT NULL CHECK (role IN ('owner', 'admin', 'viewer')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, user_id)
);

CREATE TABLE agrafa_meta.instance_settings (
    id BIGSERIAL PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    value TEXT,
    description TEXT,
    is_sensitive BOOLEAN NOT NULL DEFAULT FALSE,
    is_encrypted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_users_email ON auth.users(email);
CREATE INDEX idx_auth_sessions_user_id ON auth.sessions(user_id);
CREATE INDEX idx_auth_sessions_expires_at ON auth.sessions(expires_at);
CREATE INDEX idx_auth_verification_tokens_lookup
    ON auth.verification_tokens(identifier, type, expires_at DESC);
CREATE INDEX idx_nodes_project_state ON app.nodes(project_id, current_state);
CREATE INDEX idx_nodes_last_heartbeat_at ON app.nodes(last_heartbeat_at);
CREATE UNIQUE INDEX idx_nodes_one_managed_per_project
    ON app.nodes(project_id)
    WHERE node_type = 'managed';
CREATE UNIQUE INDEX idx_nodes_agent_token_hash
    ON app.nodes(agent_token_hash)
    WHERE agent_token_hash IS NOT NULL;
CREATE INDEX idx_services_node_state ON app.services(node_id, current_state);
CREATE INDEX idx_heartbeats_node_observed_at ON app.heartbeats(node_id, observed_at DESC);
CREATE INDEX idx_health_checks_service_observed_at ON app.health_check_results(service_id, observed_at DESC);
CREATE INDEX idx_metric_samples_node_observed_at ON app.metric_samples(node_id, observed_at DESC);
CREATE INDEX idx_events_occurred_at ON app.events(occurred_at DESC);
CREATE INDEX idx_alert_rules_project_id ON app.alert_rules(project_id);
CREATE INDEX idx_alert_rules_node_id ON app.alert_rules(node_id);
CREATE INDEX idx_alert_rules_service_id ON app.alert_rules(service_id);
CREATE INDEX idx_alert_rules_enabled_type ON app.alert_rules(is_enabled, rule_type);
CREATE UNIQUE INDEX idx_alert_instances_rule_active
    ON app.alert_instances(alert_rule_id)
    WHERE status = 'active';
CREATE INDEX idx_alert_instances_project_status_triggered_at
    ON app.alert_instances(project_id, status, triggered_at DESC, id DESC);
CREATE INDEX idx_alert_instances_triggered_at ON app.alert_instances(triggered_at DESC, id DESC);
CREATE INDEX idx_notification_recipients_project_id ON app.notification_recipients(project_id);
CREATE INDEX idx_notification_recipients_project_channel_enabled
    ON app.notification_recipients(project_id, channel_type, is_enabled);
CREATE INDEX idx_notification_deliveries_project_sent_at
    ON app.notification_deliveries(project_id, sent_at DESC, id DESC);
CREATE INDEX idx_notification_deliveries_status_sent_at
    ON app.notification_deliveries(status, sent_at DESC, id DESC);
CREATE INDEX idx_project_invitations_project_id ON app.project_invitations(project_id);
CREATE INDEX idx_project_invitations_email ON app.project_invitations(email);
CREATE INDEX idx_project_invitations_expires_at ON app.project_invitations(expires_at);
CREATE INDEX idx_app_project_members_project_id ON app.project_members(project_id);
CREATE INDEX idx_app_project_members_user_id ON app.project_members(user_id);
