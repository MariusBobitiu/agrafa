ALTER TABLE app.alert_rules
    ADD COLUMN severity TEXT;

UPDATE app.alert_rules
SET severity = CASE
    WHEN rule_type IN ('node_offline', 'service_unhealthy') THEN 'critical'
    ELSE 'warning'
END
WHERE severity IS NULL;

ALTER TABLE app.alert_rules
    ALTER COLUMN severity SET NOT NULL,
    ADD CONSTRAINT alert_rules_severity_check
        CHECK (severity IN ('info', 'warning', 'critical'));

ALTER TABLE app.notification_recipients
    ADD COLUMN min_severity TEXT;

UPDATE app.notification_recipients
SET min_severity = 'info'
WHERE min_severity IS NULL;

ALTER TABLE app.notification_recipients
    ALTER COLUMN min_severity SET NOT NULL,
    ADD CONSTRAINT notification_recipients_min_severity_check
        CHECK (min_severity IN ('info', 'warning', 'critical'));
