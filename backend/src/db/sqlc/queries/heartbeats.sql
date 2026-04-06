-- name: CreateHeartbeat :one
INSERT INTO app.heartbeats (
    node_id,
    observed_at,
    source,
    payload
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;
