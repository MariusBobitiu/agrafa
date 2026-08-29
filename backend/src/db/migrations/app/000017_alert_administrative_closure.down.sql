ALTER TABLE app.alert_instances
    DROP CONSTRAINT alert_instances_lifecycle_check,
    DROP CONSTRAINT alert_instances_status_check;

UPDATE app.alert_instances
SET status = 'resolved',
    resolved_at = closed_at
WHERE status = 'closed';

ALTER TABLE app.alert_instances
    DROP COLUMN closure_reason,
    DROP COLUMN closed_at,
    ADD CONSTRAINT alert_instances_status_check CHECK (status IN ('active', 'resolved')),
    ADD CONSTRAINT alert_instances_check CHECK (
        (status = 'active' AND resolved_at IS NULL)
        OR (status = 'resolved' AND resolved_at IS NOT NULL)
    );
