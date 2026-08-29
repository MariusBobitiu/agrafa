ALTER TABLE app.alert_instances
    ADD COLUMN closed_at TIMESTAMPTZ,
    ADD COLUMN closure_reason TEXT;

-- Existing resolved rows remain untouched: their historical resolution cause is unknowable.

ALTER TABLE app.alert_instances
    DROP CONSTRAINT alert_instances_status_check,
    DROP CONSTRAINT alert_instances_check,
    ADD CONSTRAINT alert_instances_status_check
        CHECK (status IN ('active', 'resolved', 'closed')),
    ADD CONSTRAINT alert_instances_lifecycle_check CHECK (
        (status = 'active' AND resolved_at IS NULL AND closed_at IS NULL AND closure_reason IS NULL)
        OR (status = 'resolved' AND resolved_at IS NOT NULL AND closed_at IS NULL AND closure_reason IS NULL)
        OR (
            status = 'closed'
            AND resolved_at IS NULL
            AND closed_at IS NOT NULL
            AND closure_reason IN ('rule_disabled', 'rule_scope_changed', 'target_hidden')
        )
    );
