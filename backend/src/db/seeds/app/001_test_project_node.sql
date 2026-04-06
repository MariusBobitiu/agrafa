BEGIN;

INSERT INTO app.projects (
    id,
    slug,
    name
) VALUES (
    1,
    'test-project',
    'Test Project'
)
ON CONFLICT (slug) DO UPDATE
SET name = EXCLUDED.name;

INSERT INTO app.nodes (
    id,
    project_id,
    name,
    identifier,
    current_state,
    metadata
) VALUES (
    1,
    1,
    'test-node',
    'test-node-01',
    'offline',
    '{"environment":"local","seeded":true}'::jsonb
)
ON CONFLICT (project_id, identifier) DO UPDATE
SET name = EXCLUDED.name,
    current_state = EXCLUDED.current_state,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();

INSERT INTO app.services (
    id,
    project_id,
    node_id,
    name,
    check_type,
    check_target,
    current_state,
    consecutive_failures,
    consecutive_successes
) VALUES (
    1,
    1,
    1,
    'test-api',
    'http',
    'http://localhost:8080/v1/health',
    'healthy',
    0,
    0
)
ON CONFLICT (project_id, node_id, name) DO UPDATE
SET check_type = EXCLUDED.check_type,
    check_target = EXCLUDED.check_target,
    current_state = EXCLUDED.current_state,
    consecutive_failures = EXCLUDED.consecutive_failures,
    consecutive_successes = EXCLUDED.consecutive_successes,
    updated_at = NOW();

SELECT setval('app.projects_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM app.projects), 1), true);
SELECT setval('app.nodes_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM app.nodes), 1), true);
SELECT setval('app.services_id_seq', GREATEST((SELECT COALESCE(MAX(id), 1) FROM app.services), 1), true);

COMMIT;
