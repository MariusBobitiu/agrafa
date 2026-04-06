-- name: GetNodeByID :one
SELECT *
FROM app.nodes
WHERE id = $1
LIMIT 1;

-- name: GetNodeByAgentTokenHash :one
SELECT *
FROM app.nodes
WHERE agent_token_hash = $1
LIMIT 1;

-- name: CreateNode :one
INSERT INTO app.nodes (
    project_id,
    name,
    identifier,
    current_state,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    'offline',
    '{}'::jsonb
)
RETURNING *;

-- name: EnsureManagedNodeByProject :one
INSERT INTO app.nodes (
    project_id,
    name,
    identifier,
    node_type,
    is_visible,
    current_state,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    'managed',
    FALSE,
    'offline',
    '{}'::jsonb
)
ON CONFLICT (project_id) WHERE node_type = 'managed'
DO UPDATE
SET name = EXCLUDED.name,
    identifier = EXCLUDED.identifier,
    node_type = 'managed',
    is_visible = FALSE
RETURNING *;

-- name: ListNodes :many
SELECT *
FROM app.nodes
ORDER BY id;

-- name: ListVisibleNodes :many
SELECT *
FROM app.nodes
WHERE is_visible = TRUE
ORDER BY id;

-- name: ListNodesByProject :many
SELECT *
FROM app.nodes
WHERE project_id = $1
ORDER BY id;

-- name: ListVisibleNodesByProject :many
SELECT *
FROM app.nodes
WHERE project_id = $1
  AND is_visible = TRUE
ORDER BY id;

-- name: UpdateNodeHeartbeat :one
UPDATE app.nodes
SET last_heartbeat_at = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateNodeState :one
UPDATE app.nodes
SET current_state = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateNodeAgentTokenHash :one
UPDATE app.nodes
SET agent_token_hash = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateNodeIdentity :one
UPDATE app.nodes
SET name = $2,
    identifier = $3,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteNodeByID :execrows
DELETE FROM app.nodes
WHERE id = $1;

-- name: ListStaleOnlineNodes :many
SELECT *
FROM app.nodes
WHERE current_state = 'online'
  AND last_heartbeat_at IS NOT NULL
  AND last_heartbeat_at < $1
ORDER BY last_heartbeat_at ASC;
