ALTER TABLE app.health_check_results
    ADD COLUMN node_id BIGINT REFERENCES app.nodes(id) ON DELETE CASCADE,
    ADD COLUMN check_type TEXT,
    ADD COLUMN source TEXT;

SELECT set_config('app.internal_bypass_rls', 'on', false);

UPDATE app.health_check_results AS h
SET node_id = s.node_id,
    check_type = s.check_type,
    source = COALESCE(NULLIF(h.payload ->> 'runner', ''), 'agent')
FROM app.services AS s
WHERE s.id = h.service_id;

SELECT set_config('app.internal_bypass_rls', 'off', false);

ALTER TABLE app.health_check_results
    ALTER COLUMN node_id SET NOT NULL,
    ALTER COLUMN check_type SET NOT NULL,
    ALTER COLUMN source SET NOT NULL;

DROP INDEX app.idx_health_checks_service_observed_at;

CREATE INDEX idx_health_checks_service_observed_at
    ON app.health_check_results(service_id, observed_at DESC, id DESC);

ALTER TABLE app.health_check_results ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.health_check_results FORCE ROW LEVEL SECURITY;

CREATE POLICY health_check_results_select_member_access
ON app.health_check_results
FOR SELECT
USING (
    EXISTS (
        SELECT 1
        FROM app.services AS s
        WHERE s.id = app.health_check_results.service_id
          AND app.project_read_context_matches(s.project_id)
    )
);

CREATE POLICY health_check_results_insert_access
ON app.health_check_results
FOR INSERT
WITH CHECK (
    app.internal_rls_bypass()
    OR EXISTS (
        SELECT 1
        FROM app.services AS s
        WHERE s.id = app.health_check_results.service_id
          AND app.project_write_context_matches(s.project_id)
    )
);

COMMENT ON TABLE app.health_check_results IS
    'Append-only service check observations. Retries are distinct observations until an external observation identity is introduced.';
