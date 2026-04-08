ALTER TABLE app.notification_recipients
    DROP CONSTRAINT IF EXISTS notification_recipients_min_severity_check,
    DROP COLUMN IF EXISTS min_severity;

ALTER TABLE app.alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_severity_check,
    DROP COLUMN IF EXISTS severity;
