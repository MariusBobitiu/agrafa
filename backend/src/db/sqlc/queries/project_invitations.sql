-- name: CreateProjectInvitation :one
INSERT INTO app.project_invitations (
    id,
    project_id,
    email,
    role,
    token_hash,
    invited_by_user_id,
    expires_at
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

-- name: DeletePendingProjectInvitationsByProjectAndEmail :execrows
DELETE FROM app.project_invitations
WHERE project_id = $1
  AND email = $2
  AND accepted_at IS NULL;

-- name: GetProjectInvitationByID :one
SELECT *
FROM app.project_invitations
WHERE id = $1
LIMIT 1;

-- name: GetActiveProjectInvitationByProjectAndEmail :one
SELECT *
FROM app.project_invitations
WHERE project_id = $1
  AND email = $2
  AND accepted_at IS NULL
  AND expires_at > $3
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListProjectInvitationsByProjectID :many
SELECT *
FROM app.project_invitations
WHERE project_id = $1
ORDER BY created_at DESC, id DESC;

-- name: GetProjectInvitationByTokenHash :one
SELECT
    pi.id,
    pi.project_id,
    pi.email,
    pi.role,
    pi.token_hash,
    pi.invited_by_user_id,
    pi.expires_at,
    pi.accepted_at,
    pi.created_at,
    pi.updated_at,
    p.name AS project_name
FROM app.project_invitations AS pi
JOIN app.projects AS p ON p.id = pi.project_id
WHERE pi.token_hash = $1
LIMIT 1;

-- name: MarkProjectInvitationAccepted :one
UPDATE app.project_invitations
SET accepted_at = $2,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProjectInvitationByID :execrows
DELETE FROM app.project_invitations
WHERE id = $1;
