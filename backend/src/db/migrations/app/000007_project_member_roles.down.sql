ALTER TABLE app.project_members
DROP CONSTRAINT IF EXISTS project_members_role_check;

UPDATE app.project_members
SET role = 'member'
WHERE role = 'viewer';

ALTER TABLE app.project_members
ADD CONSTRAINT project_members_role_check
CHECK (role IN ('owner', 'admin', 'member'));
