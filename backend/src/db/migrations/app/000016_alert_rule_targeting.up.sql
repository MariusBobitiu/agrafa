DO $migration$
DECLARE
    legacy_target_constraint RECORD;
BEGIN
    FOR legacy_target_constraint IN
        SELECT constraint_row.conname
        FROM pg_constraint AS constraint_row
        WHERE constraint_row.conrelid = 'app.alert_rules'::regclass
          AND constraint_row.contype = 'c'
          AND constraint_row.conname <> 'alert_rules_target_check'
          AND LOWER(pg_get_constraintdef(constraint_row.oid)) LIKE '%rule_type = ''node_offline''%'
          AND LOWER(pg_get_constraintdef(constraint_row.oid)) LIKE '%rule_type = ''service_unhealthy''%'
          AND LOWER(pg_get_constraintdef(constraint_row.oid)) LIKE '%node_id is not null%'
          AND LOWER(pg_get_constraintdef(constraint_row.oid)) LIKE '%service_id is not null%'
    LOOP
        EXECUTE format(
            'ALTER TABLE app.alert_rules DROP CONSTRAINT %I',
            legacy_target_constraint.conname
        );
    END LOOP;
END
$migration$;

ALTER TABLE app.alert_rules
    DROP CONSTRAINT IF EXISTS alert_rules_target_check,
    ADD CONSTRAINT alert_rules_target_check CHECK (
        (
            rule_type = 'node_offline'
            AND service_id IS NULL
            AND metric_name IS NULL
            AND threshold_value IS NULL
        )
        OR (
            rule_type = 'service_unhealthy'
            AND node_id IS NULL
            AND metric_name IS NULL
            AND threshold_value IS NULL
        )
        OR (
            rule_type = 'cpu_above_threshold'
            AND service_id IS NULL
            AND metric_name = 'cpu_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
        OR (
            rule_type = 'memory_above_threshold'
            AND service_id IS NULL
            AND metric_name = 'memory_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
        OR (
            rule_type = 'disk_above_threshold'
            AND service_id IS NULL
            AND metric_name = 'disk_usage'
            AND threshold_value IS NOT NULL
            AND threshold_value > 0
        )
    );

ALTER TABLE app.alert_instances
    ADD CONSTRAINT alert_instances_concrete_target_check
        CHECK (num_nonnulls(node_id, service_id) = 1);

DROP INDEX IF EXISTS app.idx_alert_instances_rule_active;

CREATE UNIQUE INDEX idx_alert_instances_rule_node_active
    ON app.alert_instances(alert_rule_id, node_id)
    WHERE status = 'active' AND node_id IS NOT NULL;

CREATE UNIQUE INDEX idx_alert_instances_rule_service_active
    ON app.alert_instances(alert_rule_id, service_id)
    WHERE status = 'active' AND service_id IS NOT NULL;
