-- name: CreatePasswordCredential :one
INSERT INTO auth.password_credentials (
    id,
    user_id,
    password_hash
) VALUES (
    $1,
    $2,
    $3
)
RETURNING *;

-- name: UpdatePasswordCredentialByUserID :execrows
UPDATE auth.password_credentials
SET password_hash = $2,
    updated_at = NOW()
WHERE user_id = $1;
