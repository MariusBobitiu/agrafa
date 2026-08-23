DROP POLICY IF EXISTS health_check_results_insert_access ON app.health_check_results;
DROP POLICY IF EXISTS health_check_results_select_member_access ON app.health_check_results;

ALTER TABLE app.health_check_results NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.health_check_results DISABLE ROW LEVEL SECURITY;

DROP INDEX app.idx_health_checks_service_observed_at;

CREATE INDEX idx_health_checks_service_observed_at
    ON app.health_check_results(service_id, observed_at DESC);

ALTER TABLE app.health_check_results
    DROP COLUMN source,
    DROP COLUMN check_type,
    DROP COLUMN node_id;

COMMENT ON TABLE app.health_check_results IS NULL;
