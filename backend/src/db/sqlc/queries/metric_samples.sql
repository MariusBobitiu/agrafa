-- name: CreateMetricSample :one
INSERT INTO app.metric_samples (
    node_id,
    service_id,
    metric_name,
    metric_value,
    metric_unit,
    observed_at,
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

-- name: GetLatestNodeMetricSampleByName :one
SELECT *
FROM app.metric_samples
WHERE node_id = $1
  AND metric_name = $2
ORDER BY observed_at DESC, id DESC
LIMIT 1;

-- name: ListLatestOperationalNodeMetrics :many
SELECT DISTINCT ON (ms.node_id, ms.metric_name)
    ms.node_id,
    ms.metric_name,
    ms.metric_value,
    ms.metric_unit,
    ms.observed_at
FROM app.metric_samples AS ms
WHERE ms.metric_name IN ('cpu_usage', 'memory_usage', 'disk_usage')
ORDER BY ms.node_id, ms.metric_name, ms.observed_at DESC, ms.id DESC;

-- name: ListLatestOperationalNodeMetricsByProject :many
SELECT DISTINCT ON (ms.node_id, ms.metric_name)
    ms.node_id,
    ms.metric_name,
    ms.metric_value,
    ms.metric_unit,
    ms.observed_at
FROM app.metric_samples AS ms
JOIN app.nodes AS n ON n.id = ms.node_id
WHERE n.project_id = $1
  AND ms.metric_name IN ('cpu_usage', 'memory_usage', 'disk_usage')
ORDER BY ms.node_id, ms.metric_name, ms.observed_at DESC, ms.id DESC;
