-- name: GetInstanceSettingByKey :one
SELECT *
FROM agrafa_meta.instance_settings
WHERE key = $1
LIMIT 1;

-- name: ListInstanceSettings :many
SELECT *
FROM agrafa_meta.instance_settings
ORDER BY key ASC;

-- name: UpsertInstanceSetting :one
INSERT INTO agrafa_meta.instance_settings (
    key,
    value,
    description,
    is_sensitive,
    is_encrypted
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
ON CONFLICT (key) DO UPDATE
SET value = EXCLUDED.value,
    description = EXCLUDED.description,
    is_sensitive = EXCLUDED.is_sensitive,
    is_encrypted = EXCLUDED.is_encrypted,
    updated_at = NOW()
RETURNING *;

-- name: DeleteInstanceSettingByKey :execrows
DELETE FROM agrafa_meta.instance_settings
WHERE key = $1;
