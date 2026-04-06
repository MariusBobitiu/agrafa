-- name: CreateHealthCheckResult :one
INSERT INTO app.health_check_results (
    service_id,
    observed_at,
    is_success,
    status_code,
    response_time_ms,
    message,
    payload
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;

-- name: GetLatestHealthCheckResultByServiceID :one
SELECT *
FROM app.health_check_results
WHERE service_id = $1
ORDER BY observed_at DESC, id DESC
LIMIT 1;

-- name: ListLatestHealthCheckResults :many
SELECT DISTINCT ON (h.service_id)
    h.*
FROM app.health_check_results AS h
ORDER BY h.service_id, h.observed_at DESC, h.id DESC;

-- name: ListLatestHealthCheckResultsByProject :many
SELECT DISTINCT ON (h.service_id)
    h.*
FROM app.health_check_results AS h
JOIN app.services AS s ON s.id = h.service_id
WHERE s.project_id = $1
ORDER BY h.service_id, h.observed_at DESC, h.id DESC;

-- name: ListLatestHealthCheckResultsForRead :many
SELECT DISTINCT ON (h.service_id)
    h.*
FROM app.health_check_results AS h
JOIN app.services AS s ON s.id = h.service_id
WHERE (NOT sqlc.arg(has_project_id)::boolean OR s.project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR s.node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_status)::boolean OR s.current_state = sqlc.arg(status))
ORDER BY h.service_id, h.observed_at DESC, h.id DESC;
