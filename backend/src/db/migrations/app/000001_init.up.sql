CREATE SCHEMA IF NOT EXISTS app;
CREATE SCHEMA IF NOT EXISTS auth;

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

CREATE INDEX idx_nodes_project_state ON app.nodes(project_id, current_state);
CREATE INDEX idx_nodes_last_heartbeat_at ON app.nodes(last_heartbeat_at);
CREATE INDEX idx_services_node_state ON app.services(node_id, current_state);
CREATE INDEX idx_heartbeats_node_observed_at ON app.heartbeats(node_id, observed_at DESC);
CREATE INDEX idx_health_checks_service_observed_at ON app.health_check_results(service_id, observed_at DESC);
CREATE INDEX idx_metric_samples_node_observed_at ON app.metric_samples(node_id, observed_at DESC);
CREATE INDEX idx_events_occurred_at ON app.events(occurred_at DESC);
