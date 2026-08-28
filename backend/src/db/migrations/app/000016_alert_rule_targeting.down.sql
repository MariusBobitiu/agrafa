DROP INDEX IF EXISTS app.idx_alert_instances_rule_service_active;
DROP INDEX IF EXISTS app.idx_alert_instances_rule_node_active;

-- Global rules cannot be represented by the previous schema. Removing them is
-- the only deterministic rollback; their alert instances cascade with them.
DELETE FROM app.alert_rules
WHERE node_id IS NULL
  AND service_id IS NULL;

WITH ranked_active AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY alert_rule_id
               ORDER BY triggered_at DESC, id DESC
           ) AS position
    FROM app.alert_instances
    WHERE status = 'active'
)
UPDATE app.alert_instances AS alert
SET status = 'resolved',
    resolved_at = NOW()
FROM ranked_active
WHERE alert.id = ranked_active.id
  AND ranked_active.position > 1;

CREATE UNIQUE INDEX idx_alert_instances_rule_active
    ON app.alert_instances(alert_rule_id)
    WHERE status = 'active';

ALTER TABLE app.alert_instances
    DROP CONSTRAINT IF EXISTS alert_instances_concrete_target_check;

ALTER TABLE app.alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_target_check,
    ADD CONSTRAINT alert_rules_check CHECK (
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
    );
