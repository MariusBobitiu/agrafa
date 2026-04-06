-- name: CreateVerificationToken :one
INSERT INTO auth.verification_tokens (
    id,
    user_id,
    identifier,
    token_hash,
    type,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetVerificationTokenByTokenHashAndType :one
SELECT *
FROM auth.verification_tokens
WHERE token_hash = $1
  AND type = $2
LIMIT 1;

-- name: DeleteVerificationTokenByID :execrows
DELETE FROM auth.verification_tokens
WHERE id = $1;

-- name: DeleteVerificationTokensByIdentifierAndType :execrows
DELETE FROM auth.verification_tokens
WHERE identifier = $1
  AND type = $2;
