-- name: CreateNotificationDelivery :one
INSERT INTO app.notification_deliveries (
    project_id,
    notification_recipient_id,
    alert_instance_id,
    channel_type,
    target,
    event_type,
    status,
    error_message,
    sent_at
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

-- name: ListNotificationDeliveries :many
SELECT *
FROM app.notification_deliveries
WHERE (NOT $1::boolean OR project_id = $2)
  AND (NOT $3::boolean OR status = $4)
ORDER BY sent_at DESC, id DESC
LIMIT $5;
