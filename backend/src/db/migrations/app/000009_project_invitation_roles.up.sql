ALTER TABLE app.project_invitations
    DROP CONSTRAINT IF EXISTS project_invitations_role_check;

ALTER TABLE app.project_invitations
    ADD CONSTRAINT project_invitations_role_check
    CHECK (role IN ('admin', 'viewer'));
