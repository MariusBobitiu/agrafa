-- name: FindActiveAlertInstanceByRuleID :one
SELECT *
FROM app.alert_instances
WHERE alert_rule_id = $1
  AND status = 'active'
LIMIT 1;

-- name: CreateAlertInstance :one
INSERT INTO app.alert_instances (
    alert_rule_id,
    project_id,
    node_id,
    service_id,
    status,
    triggered_at,
    resolved_at,
    title,
    message
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9
)
RETURNING *;

-- name: ResolveAlertInstance :one
UPDATE app.alert_instances
SET status = 'resolved',
    resolved_at = $2
WHERE id = $1
  AND status = 'active'
RETURNING *;

-- name: ListAlertInstances :many
SELECT *
FROM app.alert_instances
WHERE (NOT $1::boolean OR project_id = $2)
  AND (NOT $3::boolean OR status = $4)
ORDER BY triggered_at DESC, id DESC
LIMIT $5;

-- name: ListActiveAlertCountsByNode :many
SELECT
    node_id,
    COUNT(*)::bigint AS active_alert_count
FROM app.alert_instances
WHERE status = 'active'
  AND node_id IS NOT NULL
GROUP BY node_id
ORDER BY node_id;

-- name: ListActiveAlertCountsByNodeByProject :many
SELECT
    node_id,
    COUNT(*)::bigint AS active_alert_count
FROM app.alert_instances
WHERE project_id = $1
  AND status = 'active'
  AND node_id IS NOT NULL
GROUP BY node_id
ORDER BY node_id;

-- name: ListActiveAlertCountsByService :many
SELECT
    service_id,
    COUNT(*)::bigint AS active_alert_count
FROM app.alert_instances
WHERE status = 'active'
  AND service_id IS NOT NULL
GROUP BY service_id
ORDER BY service_id;

-- name: CountActiveAlertInstancesByServiceID :one
SELECT COUNT(*)::bigint
FROM app.alert_instances
WHERE service_id = $1
  AND status = 'active';

-- name: CountActiveAlertInstancesByNodeID :one
SELECT COUNT(*)::bigint
FROM app.alert_instances
WHERE node_id = $1
  AND status = 'active';

-- name: ListActiveAlertCountsByServiceByProject :many
SELECT
    service_id,
    COUNT(*)::bigint AS active_alert_count
FROM app.alert_instances
WHERE project_id = $1
  AND status = 'active'
  AND service_id IS NOT NULL
GROUP BY service_id
ORDER BY service_id;

-- name: ListActiveAlertCountsByServiceForRead :many
SELECT
    a.service_id,
    COUNT(*)::bigint AS active_alert_count
FROM app.alert_instances AS a
JOIN app.services AS s ON s.id = a.service_id
WHERE a.status = 'active'
  AND a.service_id IS NOT NULL
  AND (NOT sqlc.arg(has_project_id)::boolean OR s.project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR s.node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_status)::boolean OR s.current_state = sqlc.arg(status))
GROUP BY a.service_id
ORDER BY a.service_id;
