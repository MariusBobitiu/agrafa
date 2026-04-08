-- name: CreateNotificationRecipient :one
INSERT INTO app.notification_recipients (
    project_id,
    channel_type,
    target,
    min_severity,
    is_enabled
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
ON CONFLICT (project_id, channel_type, target) DO UPDATE
SET min_severity = EXCLUDED.min_severity,
    is_enabled = TRUE,
    updated_at = NOW()
RETURNING *;

-- name: GetNotificationRecipientByID :one
SELECT *
FROM app.notification_recipients
WHERE id = $1
LIMIT 1;

-- name: ListNotificationRecipients :many
SELECT *
FROM app.notification_recipients
WHERE (NOT $1::boolean OR project_id = $2)
ORDER BY id DESC;

-- name: ListNotificationRecipientsByProjectAndChannel :many
SELECT *
FROM app.notification_recipients
WHERE project_id = $1
  AND channel_type = $2
ORDER BY id DESC;

-- name: UpdateNotificationRecipientEnabled :one
UPDATE app.notification_recipients
SET is_enabled = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteNotificationRecipientByID :execrows
DELETE FROM app.notification_recipients
WHERE id = $1;
