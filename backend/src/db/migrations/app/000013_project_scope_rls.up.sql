CREATE OR REPLACE FUNCTION app.current_user_id()
RETURNS TEXT
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('app.current_user_id', true), '');
$$;

CREATE OR REPLACE FUNCTION app.current_project_id()
RETURNS BIGINT
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('app.current_project_id', true), '')::bigint;
$$;

CREATE OR REPLACE FUNCTION app.current_project_role()
RETURNS TEXT
LANGUAGE sql
STABLE
AS $$
    SELECT NULLIF(current_setting('app.current_project_role', true), '');
$$;

CREATE OR REPLACE FUNCTION app.internal_rls_bypass()
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT COALESCE(current_setting('app.internal_bypass_rls', true), '') = 'on';
$$;

CREATE OR REPLACE FUNCTION app.project_read_context_matches(target_project_id BIGINT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT app.internal_rls_bypass() OR (
        app.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM app.project_members AS pm
            WHERE pm.project_id = target_project_id
              AND pm.user_id = app.current_user_id()
              AND pm.role IN ('owner', 'admin', 'viewer')
        )
    );
$$;

CREATE OR REPLACE FUNCTION app.project_write_context_matches(target_project_id BIGINT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT app.internal_rls_bypass() OR (
        app.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM app.project_members AS pm
            WHERE pm.project_id = target_project_id
              AND pm.user_id = app.current_user_id()
              AND pm.role IN ('owner', 'admin')
        )
    );
$$;

CREATE OR REPLACE FUNCTION app.project_owner_context_matches(target_project_id BIGINT)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT app.internal_rls_bypass() OR (
        app.current_user_id() IS NOT NULL
        AND EXISTS (
            SELECT 1
            FROM app.project_members AS pm
            WHERE pm.project_id = target_project_id
              AND pm.user_id = app.current_user_id()
              AND pm.role = 'owner'
        )
    );
$$;

ALTER TABLE app.projects ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.projects FORCE ROW LEVEL SECURITY;
ALTER TABLE app.project_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_members FORCE ROW LEVEL SECURITY;
ALTER TABLE app.nodes ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.nodes FORCE ROW LEVEL SECURITY;
ALTER TABLE app.services ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.services FORCE ROW LEVEL SECURITY;
ALTER TABLE app.alert_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.alert_rules FORCE ROW LEVEL SECURITY;
ALTER TABLE app.notification_recipients ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.notification_recipients FORCE ROW LEVEL SECURITY;
ALTER TABLE app.alert_instances ENABLE ROW LEVEL SECURITY;
ALTER TABLE app.alert_instances FORCE ROW LEVEL SECURITY;

CREATE POLICY projects_select_member_access
ON app.projects
FOR SELECT
USING (
    app.internal_rls_bypass()
    OR EXISTS (
        SELECT 1
        FROM app.project_members AS pm
        WHERE pm.project_id = app.projects.id
          AND pm.user_id = app.current_user_id()
    )
);

CREATE POLICY projects_insert_authenticated_access
ON app.projects
FOR INSERT
WITH CHECK (
    app.internal_rls_bypass()
    OR app.current_user_id() IS NOT NULL
);

CREATE POLICY projects_update_member_access
ON app.projects
FOR UPDATE
USING (app.project_write_context_matches(id))
WITH CHECK (app.project_write_context_matches(id));

CREATE POLICY projects_delete_owner_access
ON app.projects
FOR DELETE
USING (app.project_owner_context_matches(id));

CREATE POLICY project_members_select_access
ON app.project_members
FOR SELECT
USING (
    app.internal_rls_bypass()
    OR user_id = app.current_user_id()
    OR app.project_read_context_matches(project_id)
);

CREATE POLICY project_members_insert_owner_access
ON app.project_members
FOR INSERT
WITH CHECK (app.project_owner_context_matches(project_id));

CREATE POLICY project_members_update_owner_access
ON app.project_members
FOR UPDATE
USING (app.project_owner_context_matches(project_id))
WITH CHECK (app.project_owner_context_matches(project_id));

CREATE POLICY project_members_delete_owner_access
ON app.project_members
FOR DELETE
USING (app.project_owner_context_matches(project_id));

CREATE POLICY nodes_select_member_access
ON app.nodes
FOR SELECT
USING (app.project_read_context_matches(project_id));

CREATE POLICY nodes_insert_admin_access
ON app.nodes
FOR INSERT
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY nodes_update_admin_access
ON app.nodes
FOR UPDATE
USING (app.project_write_context_matches(project_id))
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY nodes_delete_admin_access
ON app.nodes
FOR DELETE
USING (app.project_write_context_matches(project_id));

CREATE POLICY services_select_member_access
ON app.services
FOR SELECT
USING (app.project_read_context_matches(project_id));

CREATE POLICY services_insert_admin_access
ON app.services
FOR INSERT
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY services_update_admin_access
ON app.services
FOR UPDATE
USING (app.project_write_context_matches(project_id))
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY services_delete_admin_access
ON app.services
FOR DELETE
USING (app.project_write_context_matches(project_id));

CREATE POLICY alert_rules_select_member_access
ON app.alert_rules
FOR SELECT
USING (app.project_read_context_matches(project_id));

CREATE POLICY alert_rules_insert_admin_access
ON app.alert_rules
FOR INSERT
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY alert_rules_update_admin_access
ON app.alert_rules
FOR UPDATE
USING (app.project_write_context_matches(project_id))
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY alert_rules_delete_admin_access
ON app.alert_rules
FOR DELETE
USING (app.project_write_context_matches(project_id));

CREATE POLICY notification_recipients_select_member_access
ON app.notification_recipients
FOR SELECT
USING (app.project_read_context_matches(project_id));

CREATE POLICY notification_recipients_insert_admin_access
ON app.notification_recipients
FOR INSERT
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY notification_recipients_update_admin_access
ON app.notification_recipients
FOR UPDATE
USING (app.project_write_context_matches(project_id))
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY notification_recipients_delete_admin_access
ON app.notification_recipients
FOR DELETE
USING (app.project_write_context_matches(project_id));

CREATE POLICY alert_instances_select_member_access
ON app.alert_instances
FOR SELECT
USING (app.project_read_context_matches(project_id));

CREATE POLICY alert_instances_insert_admin_access
ON app.alert_instances
FOR INSERT
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY alert_instances_update_admin_access
ON app.alert_instances
FOR UPDATE
USING (app.project_write_context_matches(project_id))
WITH CHECK (app.project_write_context_matches(project_id));

CREATE POLICY alert_instances_delete_admin_access
ON app.alert_instances
FOR DELETE
USING (app.project_write_context_matches(project_id));
