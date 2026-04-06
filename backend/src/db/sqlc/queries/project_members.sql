-- name: CreateProjectMember :one
INSERT INTO app.project_members (
    id,
    project_id,
    user_id,
    role
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;

-- name: GetProjectMemberByProjectAndUser :one
SELECT *
FROM app.project_members
WHERE project_id = $1
  AND user_id = $2
LIMIT 1;

-- name: GetProjectMemberByID :one
SELECT *
FROM app.project_members
WHERE id = $1
LIMIT 1;

-- name: GetProjectMemberForReadByID :one
SELECT
    pm.id,
    pm.project_id,
    pm.user_id,
    pm.role,
    pm.created_at,
    u.name,
    u.email,
    u.image
FROM app.project_members AS pm
JOIN auth.users AS u ON u.id = pm.user_id
WHERE pm.id = $1
LIMIT 1;

-- name: ListProjectMembersForRead :many
SELECT
    pm.id,
    pm.project_id,
    pm.user_id,
    pm.role,
    pm.created_at,
    u.name,
    u.email,
    u.image
FROM app.project_members AS pm
JOIN auth.users AS u ON u.id = pm.user_id
WHERE pm.project_id = $1
ORDER BY pm.created_at ASC, pm.id ASC;

-- name: UpdateProjectMemberRole :one
UPDATE app.project_members
SET role = $2
WHERE id = $1
RETURNING *;

-- name: DeleteProjectMemberByID :execrows
DELETE FROM app.project_members
WHERE id = $1;

-- name: CountProjectOwners :one
SELECT COUNT(*)::bigint
FROM app.project_members
WHERE project_id = $1
  AND role = 'owner';
