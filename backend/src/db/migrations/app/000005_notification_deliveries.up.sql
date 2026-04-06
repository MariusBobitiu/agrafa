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

CREATE INDEX idx_notification_deliveries_project_sent_at
    ON app.notification_deliveries(project_id, sent_at DESC, id DESC);
CREATE INDEX idx_notification_deliveries_status_sent_at
    ON app.notification_deliveries(status, sent_at DESC, id DESC);
