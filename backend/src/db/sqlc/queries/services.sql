-- name: GetServiceByID :one
SELECT *
FROM app.services
WHERE id = $1
LIMIT 1;

-- name: CountServicesByNodeID :one
SELECT COUNT(*)::bigint
FROM app.services
WHERE node_id = $1;

-- name: CreateService :one
INSERT INTO app.services (
    project_id,
    node_id,
    name,
    check_type,
    check_target,
    current_state,
    consecutive_failures,
    consecutive_successes
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    'healthy',
    0,
    0
)
RETURNING *;

-- name: ListServices :many
SELECT *
FROM app.services
ORDER BY id;

-- name: ListServicesByProject :many
SELECT *
FROM app.services
WHERE project_id = $1
ORDER BY id;

-- name: ListAgentConfigChecksByNodeID :many
SELECT
    s.id AS service_id,
    s.name,
    s.check_type,
    s.check_target
FROM app.services AS s
JOIN app.nodes AS n ON n.id = s.node_id
WHERE s.node_id = $1
  AND n.node_type = 'agent'
ORDER BY s.id;

-- name: ListServicesForRead :many
SELECT *
FROM app.services
WHERE (NOT sqlc.arg(has_project_id)::boolean OR project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_status)::boolean OR current_state = sqlc.arg(status))
ORDER BY created_at DESC, id DESC;

-- name: ListServicesForReadLimited :many
SELECT *
FROM app.services
WHERE (NOT sqlc.arg(has_project_id)::boolean OR project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_status)::boolean OR current_state = sqlc.arg(status))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(limit_rows);

-- name: UpdateServiceState :one
UPDATE app.services
SET current_state = $2,
    consecutive_failures = $3,
    consecutive_successes = $4,
    last_check_at = $5,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateServiceDefinition :one
UPDATE app.services
SET name = $2,
    check_type = $3,
    check_target = $4,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteServiceByID :execrows
DELETE FROM app.services
WHERE id = $1;

-- name: ListServiceCountsByNode :many
SELECT
    node_id,
    COUNT(*)::bigint AS service_count
FROM app.services
GROUP BY node_id
ORDER BY node_id;

-- name: ListServiceCountsByNodeByProject :many
SELECT
    node_id,
    COUNT(*)::bigint AS service_count
FROM app.services
WHERE project_id = $1
GROUP BY node_id
ORDER BY node_id;
