-- name: CreateEvent :one
INSERT INTO app.events (
    project_id,
    node_id,
    service_id,
    event_type,
    severity,
    title,
    details,
    occurred_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: ListEvents :many
SELECT *
FROM app.events
ORDER BY occurred_at DESC, id DESC
LIMIT $1;

-- name: ListEventsByProject :many
SELECT *
FROM app.events
WHERE project_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT $2;

-- name: ListRecentAlertEvents :many
SELECT *
FROM app.events
WHERE event_type IN ('alert_triggered', 'alert_resolved')
ORDER BY occurred_at DESC, id DESC
LIMIT $1;

-- name: ListRecentAlertEventsByProject :many
SELECT *
FROM app.events
WHERE project_id = $1
  AND event_type IN ('alert_triggered', 'alert_resolved')
ORDER BY occurred_at DESC, id DESC
LIMIT $2;
