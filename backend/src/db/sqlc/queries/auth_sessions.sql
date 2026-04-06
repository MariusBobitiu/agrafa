-- name: CreateSession :one
INSERT INTO auth.sessions (
    id,
    token_hash,
    user_id,
    expires_at,
    ip_address,
    user_agent
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: DeleteSessionByTokenHash :exec
DELETE FROM auth.sessions
WHERE token_hash = $1;

-- name: GetSessionByTokenHash :one
SELECT *
FROM auth.sessions
WHERE token_hash = $1
LIMIT 1;

-- name: GetSessionUserByTokenHash :one
SELECT
    u.id,
    u.name,
    u.email,
    u.email_verified,
    u.image,
    u.onboarding_completed,
    u.two_factor_enabled,
    u.created_at,
    u.updated_at,
    s.expires_at
FROM auth.sessions AS s
JOIN auth.users AS u ON u.id = s.user_id
WHERE s.token_hash = $1
LIMIT 1;

-- name: ListSessionsByUserID :many
SELECT *
FROM auth.sessions
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: DeleteSessionByIDAndUser :execrows
DELETE FROM auth.sessions
WHERE id = $1
  AND user_id = $2;

-- name: DeleteOtherSessionsByUser :execrows
DELETE FROM auth.sessions
WHERE user_id = $1
  AND token_hash <> $2;

-- name: DeleteSessionsByUserID :execrows
DELETE FROM auth.sessions
WHERE user_id = $1;
