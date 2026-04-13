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
  AND rule_type = $1
  AND (NOT $2::boolean OR node_id = $3)
  AND (NOT $4::boolean OR service_id = $5)
  AND (NOT $6::boolean OR metric_name = $7)
ORDER BY id DESC;

-- name: UpdateAlertRule :one
UPDATE app.alert_rules
SET node_id = CASE WHEN $2::boolean THEN $3 ELSE node_id END,
    service_id = CASE WHEN $4::boolean THEN $5 ELSE service_id END,
    severity = CASE WHEN $6::boolean THEN $7 ELSE severity END,
    threshold_value = CASE WHEN $8::boolean THEN $9 ELSE threshold_value END,
    is_enabled = CASE WHEN $10::boolean THEN $11 ELSE is_enabled END,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteAlertRuleByID :execrows
DELETE FROM app.alert_rules
WHERE id = $1;
