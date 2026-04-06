-- name: CreateUser :one
INSERT INTO auth.users (
    id,
    name,
    email,
    email_verified,
    image,
    onboarding_completed,
    two_factor_enabled
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

-- name: GetUserByID :one
SELECT *
FROM auth.users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT *
FROM auth.users
WHERE email = $1
LIMIT 1;

-- name: MarkUserEmailVerifiedByID :execrows
UPDATE auth.users
SET email_verified = TRUE,
    updated_at = NOW()
WHERE id = $1;

-- name: CompleteUserOnboardingByID :one
UPDATE auth.users
SET onboarding_completed = TRUE,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetUserWithPasswordByEmail :one
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
    pc.password_hash
FROM auth.users AS u
JOIN auth.password_credentials AS pc ON pc.user_id = u.id
WHERE u.email = $1
LIMIT 1;

-- name: GetUserWithPasswordByID :one
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
    pc.password_hash
FROM auth.users AS u
JOIN auth.password_credentials AS pc ON pc.user_id = u.id
WHERE u.id = $1
LIMIT 1;
