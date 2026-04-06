CREATE TABLE app.notification_recipients (
    id BIGSERIAL PRIMARY KEY,
    project_id BIGINT NOT NULL REFERENCES app.projects(id) ON DELETE CASCADE,
    channel_type TEXT NOT NULL CHECK (channel_type IN ('email')),
    target TEXT NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (project_id, channel_type, target)
);

CREATE INDEX idx_notification_recipients_project_id
    ON app.notification_recipients(project_id);
CREATE INDEX idx_notification_recipients_project_channel_enabled
    ON app.notification_recipients(project_id, channel_type, is_enabled);
