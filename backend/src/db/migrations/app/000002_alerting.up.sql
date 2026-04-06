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

CREATE UNIQUE INDEX idx_alert_instances_rule_active
    ON app.alert_instances(alert_rule_id)
    WHERE status = 'active';

CREATE INDEX idx_alert_rules_project_id ON app.alert_rules(project_id);
CREATE INDEX idx_alert_rules_node_id ON app.alert_rules(node_id);
CREATE INDEX idx_alert_rules_service_id ON app.alert_rules(service_id);
CREATE INDEX idx_alert_rules_enabled_type ON app.alert_rules(is_enabled, rule_type);
CREATE INDEX idx_alert_instances_project_status_triggered_at
    ON app.alert_instances(project_id, status, triggered_at DESC, id DESC);
CREATE INDEX idx_alert_instances_triggered_at ON app.alert_instances(triggered_at DESC, id DESC);
