-- name: CreateHealthCheckResult :one
INSERT INTO app.health_check_results (
    service_id,
    node_id,
    check_type,
    source,
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
    $7,
    $8,
    $9,
    $10
)
RETURNING *;

-- name: GetLatestHealthCheckResultByServiceID :one
SELECT *
FROM app.health_check_results
WHERE service_id = $1
ORDER BY observed_at DESC, id DESC
LIMIT 1;

-- name: ListHealthCheckResultsByServiceIDAfterID :many
SELECT *
FROM app.health_check_results
WHERE service_id = $1
  AND id > sqlc.arg(after_id)
ORDER BY id ASC;

-- name: ListLatestHealthCheckResults :many
SELECT DISTINCT ON (h.service_id)
    h.*
FROM app.health_check_results AS h
ORDER BY h.service_id, h.observed_at DESC, h.id DESC;

-- name: ListHealthCheckHistoryByServiceID :many
SELECT *
FROM app.health_check_results
WHERE service_id = sqlc.arg(service_id)
  AND (
      NOT sqlc.arg(has_range)::boolean
      OR observed_at BETWEEN sqlc.arg(from_observed_at)::timestamptz
                         AND sqlc.arg(to_observed_at)::timestamptz
  )
  AND (
      NOT sqlc.arg(has_before)::boolean
      OR observed_at < sqlc.arg(before_observed_at)::timestamptz
      OR (
          observed_at = sqlc.arg(before_observed_at)::timestamptz
          AND id < sqlc.arg(before_id)::bigint
      )
  )
ORDER BY observed_at DESC, id DESC
LIMIT sqlc.arg(limit_rows);

-- name: SummarizeHealthCheckHistoryByServiceID :one
SELECT
    COUNT(*)::bigint AS total_checks,
    COUNT(*) FILTER (WHERE is_success)::bigint AS successful_checks,
    COUNT(response_time_ms) FILTER (WHERE is_success)::bigint AS measured_latency_checks,
    COALESCE(SUM(response_time_ms) FILTER (WHERE is_success), 0)::bigint AS total_latency_ms,
    COALESCE(EXTRACT(EPOCH FROM MAX(observed_at)) * 1000000000, 0)::bigint AS last_checked_unix_nano
FROM app.health_check_results
WHERE service_id = sqlc.arg(service_id)
  AND observed_at BETWEEN sqlc.arg(from_observed_at)::timestamptz
                      AND sqlc.arg(to_observed_at)::timestamptz;

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
