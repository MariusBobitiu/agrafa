-- name: CreateAlertRule :one
INSERT INTO app.alert_rules (
    project_id,
    node_id,
    service_id,
    rule_type,
    severity,
    metric_name,
    threshold_value,
    is_enabled
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

-- name: GetAlertRuleByID :one
SELECT *
FROM app.alert_rules
WHERE id = $1
LIMIT 1;

-- name: ListAlertRules :many
SELECT *
FROM app.alert_rules
WHERE (NOT $1::boolean OR project_id = $2)
ORDER BY id DESC;

-- name: ListEnabledAlertRules :many
SELECT *
FROM app.alert_rules
WHERE is_enabled = TRUE
  AND project_id = sqlc.arg(project_id)
  AND rule_type = sqlc.arg(rule_type)
  AND (
      NOT sqlc.arg(has_node_id)::boolean
      OR node_id = sqlc.narg(node_id)::bigint
      OR node_id IS NULL
  )
  AND (
      NOT sqlc.arg(has_service_id)::boolean
      OR service_id = sqlc.narg(service_id)::bigint
      OR service_id IS NULL
  )
  AND (
      NOT sqlc.arg(has_metric_name)::boolean
      OR metric_name = sqlc.narg(metric_name)::text
  )
ORDER BY id DESC;

-- name: UpdateAlertRule :one
UPDATE app.alert_rules
SET node_id = CASE WHEN sqlc.arg(set_node_id)::boolean THEN sqlc.narg(node_id)::bigint ELSE node_id END,
    service_id = CASE WHEN sqlc.arg(set_service_id)::boolean THEN sqlc.narg(service_id)::bigint ELSE service_id END,
    severity = CASE WHEN sqlc.arg(set_severity)::boolean THEN sqlc.arg(severity) ELSE severity END,
    threshold_value = CASE WHEN sqlc.arg(set_threshold_value)::boolean THEN sqlc.narg(threshold_value)::double precision ELSE threshold_value END,
    is_enabled = CASE WHEN sqlc.arg(set_is_enabled)::boolean THEN sqlc.arg(is_enabled) ELSE is_enabled END,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteAlertRuleByID :execrows
DELETE FROM app.alert_rules
WHERE id = $1;
