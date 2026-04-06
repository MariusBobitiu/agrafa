-- name: CreateProject :one
INSERT INTO app.projects (
    slug,
    name
) VALUES (
    $1,
    $2
)
RETURNING *;

-- name: GetProjectByID :one
SELECT *
FROM app.projects
WHERE id = $1
LIMIT 1;

-- name: ListProjectsForUser :many
SELECT
    p.id,
    p.slug,
    p.name,
    p.created_at,
    pm.role
FROM app.project_members AS pm
JOIN app.projects AS p ON p.id = pm.project_id
WHERE pm.user_id = $1
ORDER BY p.created_at DESC, p.id DESC;

-- name: DeleteProjectByID :execrows
DELETE FROM app.projects
WHERE id = $1;

-- name: UpdateProjectName :one
UPDATE app.projects
SET name = $2
WHERE id = $1
RETURNING *;
