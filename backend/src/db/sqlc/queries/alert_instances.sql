-- name: FindActiveAlertInstanceByRuleAndTarget :one
SELECT *
FROM app.alert_instances
WHERE alert_rule_id = sqlc.arg(alert_rule_id)
  AND node_id IS NOT DISTINCT FROM sqlc.narg(node_id)::bigint
  AND service_id IS NOT DISTINCT FROM sqlc.narg(service_id)::bigint
  AND status = 'active'
LIMIT 1;

-- name: ListActiveAlertInstancesByRuleID :many
SELECT *
FROM app.alert_instances
WHERE alert_rule_id = $1
  AND status = 'active'
ORDER BY id;

-- name: CreateAlertInstance :one
INSERT INTO app.alert_instances (
    alert_rule_id,
    project_id,
    node_id,
    service_id,
    status,
    triggered_at,
    resolved_at,
    closed_at,
    closure_reason,
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
    $9,
    $10,
    $11
)
RETURNING *;

-- name: ResolveAlertInstance :one
UPDATE app.alert_instances
SET status = 'resolved',
    resolved_at = $2
WHERE id = $1
  AND status = 'active'
RETURNING *;

-- name: CloseAlertInstance :one
UPDATE app.alert_instances
SET status = 'closed',
    closed_at = $2,
    closure_reason = $3
WHERE id = $1
  AND status = 'active'
RETURNING *;

-- name: ListAlertInstances :many
SELECT
    ai.id,
    ai.alert_rule_id,
    ai.project_id,
    ai.node_id,
    n.name AS node_name,
    n.identifier AS node_identifier,
    ai.service_id,
    s.name AS service_name,
    ar.rule_type,
    ar.severity,
    ai.status,
    ai.triggered_at,
    ai.resolved_at,
    ai.closed_at,
    ai.closure_reason,
    ai.title,
    ai.message,
    ai.created_at
FROM app.alert_instances AS ai
JOIN app.alert_rules AS ar ON ar.id = ai.alert_rule_id
LEFT JOIN app.nodes AS n ON n.id = ai.node_id
LEFT JOIN app.services AS s ON s.id = ai.service_id
WHERE (NOT sqlc.arg(has_project_id)::boolean OR ai.project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_service_id)::boolean OR ai.service_id = sqlc.arg(service_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR ai.node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_rule_type)::boolean OR ar.rule_type = sqlc.arg(rule_type))
  AND (NOT sqlc.arg(has_severity)::boolean OR ar.severity = sqlc.arg(severity))
  AND (
      NOT sqlc.arg(has_category)::boolean
      OR (sqlc.arg(category)::text = 'node' AND ar.rule_type = 'node_offline')
      OR (sqlc.arg(category)::text = 'service' AND ar.rule_type = 'service_unhealthy')
      OR (
          sqlc.arg(category)::text = 'metric'
          AND ar.rule_type IN ('cpu_above_threshold', 'memory_above_threshold', 'disk_above_threshold')
      )
  )
ORDER BY ai.triggered_at DESC, ai.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListActiveAlertInstancesForRead :many
SELECT
    ai.id,
    ai.alert_rule_id,
    ai.project_id,
    ai.node_id,
    n.name AS node_name,
    n.identifier AS node_identifier,
    ai.service_id,
    s.name AS service_name,
    ar.rule_type,
    ar.severity,
    ai.status,
    ai.triggered_at,
    ai.resolved_at,
    ai.closed_at,
    ai.closure_reason,
    ai.title,
    ai.message,
    ai.created_at
FROM app.alert_instances AS ai
JOIN app.alert_rules AS ar ON ar.id = ai.alert_rule_id
LEFT JOIN app.nodes AS n ON n.id = ai.node_id
LEFT JOIN app.services AS s ON s.id = ai.service_id
WHERE ai.status = 'active'
  AND (NOT sqlc.arg(has_project_id)::boolean OR ai.project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_service_id)::boolean OR ai.service_id = sqlc.arg(service_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR ai.node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_rule_type)::boolean OR ar.rule_type = sqlc.arg(rule_type))
  AND (NOT sqlc.arg(has_severity)::boolean OR ar.severity = sqlc.arg(severity))
  AND (
      NOT sqlc.arg(has_category)::boolean
      OR (sqlc.arg(category)::text = 'node' AND ar.rule_type = 'node_offline')
      OR (sqlc.arg(category)::text = 'service' AND ar.rule_type = 'service_unhealthy')
      OR (
          sqlc.arg(category)::text = 'metric'
          AND ar.rule_type IN ('cpu_above_threshold', 'memory_above_threshold', 'disk_above_threshold')
      )
  )
ORDER BY ai.triggered_at DESC, ai.id DESC;

-- name: ListResolvedAlertInstancesForRead :many
SELECT
    ai.id,
    ai.alert_rule_id,
    ai.project_id,
    ai.node_id,
    n.name AS node_name,
    n.identifier AS node_identifier,
    ai.service_id,
    s.name AS service_name,
    ar.rule_type,
    ar.severity,
    ai.status,
    ai.triggered_at,
    ai.resolved_at,
    ai.closed_at,
    ai.closure_reason,
    ai.title,
    ai.message,
    ai.created_at
FROM app.alert_instances AS ai
JOIN app.alert_rules AS ar ON ar.id = ai.alert_rule_id
LEFT JOIN app.nodes AS n ON n.id = ai.node_id
LEFT JOIN app.services AS s ON s.id = ai.service_id
WHERE ai.status IN ('resolved', 'closed')
  AND (NOT sqlc.arg(has_project_id)::boolean OR ai.project_id = sqlc.arg(project_id))
  AND (NOT sqlc.arg(has_service_id)::boolean OR ai.service_id = sqlc.arg(service_id))
  AND (NOT sqlc.arg(has_node_id)::boolean OR ai.node_id = sqlc.arg(node_id))
  AND (NOT sqlc.arg(has_rule_type)::boolean OR ar.rule_type = sqlc.arg(rule_type))
  AND (NOT sqlc.arg(has_severity)::boolean OR ar.severity = sqlc.arg(severity))
  AND (
      NOT sqlc.arg(has_category)::boolean
      OR (sqlc.arg(category)::text = 'node' AND ar.rule_type = 'node_offline')
      OR (sqlc.arg(category)::text = 'service' AND ar.rule_type = 'service_unhealthy')
      OR (
          sqlc.arg(category)::text = 'metric'
          AND ar.rule_type IN ('cpu_above_threshold', 'memory_above_threshold', 'disk_above_threshold')
      )
  )
  AND (
      NOT sqlc.arg(has_before)::boolean
      OR ai.triggered_at < sqlc.arg(before_triggered_at)::timestamptz
      OR (
          ai.triggered_at = sqlc.arg(before_triggered_at)::timestamptz
          AND ai.id < sqlc.arg(before_id)::bigint
      )
  )
ORDER BY ai.triggered_at DESC, ai.id DESC
LIMIT sqlc.arg(limit_rows);

-- name: ListAlertInstancesByNodeAndStatus :many
SELECT *
FROM app.alert_instances
WHERE node_id = $1
  AND (NOT $2::boolean OR status = $3)
ORDER BY triggered_at DESC, id DESC
LIMIT $4;

-- name: ListActiveAlertDetailsByServiceID :many
SELECT
    ai.id,
    ai.alert_rule_id AS rule_id,
    ar.rule_type,
    ar.severity,
    ai.title,
    ai.status,
    ai.triggered_at
FROM app.alert_instances AS ai
JOIN app.alert_rules AS ar ON ar.id = ai.alert_rule_id
WHERE ai.service_id = $1
  AND ai.status = 'active'
ORDER BY ai.triggered_at DESC, ai.id DESC;

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
