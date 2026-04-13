DROP POLICY IF EXISTS alert_instances_delete_admin_access ON app.alert_instances;
DROP POLICY IF EXISTS alert_instances_update_admin_access ON app.alert_instances;
DROP POLICY IF EXISTS alert_instances_insert_admin_access ON app.alert_instances;
DROP POLICY IF EXISTS alert_instances_select_member_access ON app.alert_instances;

DROP POLICY IF EXISTS notification_recipients_delete_admin_access ON app.notification_recipients;
DROP POLICY IF EXISTS notification_recipients_update_admin_access ON app.notification_recipients;
DROP POLICY IF EXISTS notification_recipients_insert_admin_access ON app.notification_recipients;
DROP POLICY IF EXISTS notification_recipients_select_member_access ON app.notification_recipients;

DROP POLICY IF EXISTS alert_rules_delete_admin_access ON app.alert_rules;
DROP POLICY IF EXISTS alert_rules_update_admin_access ON app.alert_rules;
DROP POLICY IF EXISTS alert_rules_insert_admin_access ON app.alert_rules;
DROP POLICY IF EXISTS alert_rules_select_member_access ON app.alert_rules;

DROP POLICY IF EXISTS services_delete_admin_access ON app.services;
DROP POLICY IF EXISTS services_update_admin_access ON app.services;
DROP POLICY IF EXISTS services_insert_admin_access ON app.services;
DROP POLICY IF EXISTS services_select_member_access ON app.services;

DROP POLICY IF EXISTS nodes_delete_admin_access ON app.nodes;
DROP POLICY IF EXISTS nodes_update_admin_access ON app.nodes;
DROP POLICY IF EXISTS nodes_insert_admin_access ON app.nodes;
DROP POLICY IF EXISTS nodes_select_member_access ON app.nodes;

DROP POLICY IF EXISTS project_members_delete_owner_access ON app.project_members;
DROP POLICY IF EXISTS project_members_update_owner_access ON app.project_members;
DROP POLICY IF EXISTS project_members_insert_owner_access ON app.project_members;
DROP POLICY IF EXISTS project_members_select_access ON app.project_members;

DROP POLICY IF EXISTS projects_delete_owner_access ON app.projects;
DROP POLICY IF EXISTS projects_update_member_access ON app.projects;
DROP POLICY IF EXISTS projects_insert_authenticated_access ON app.projects;
DROP POLICY IF EXISTS projects_select_member_access ON app.projects;

ALTER TABLE app.alert_instances NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.alert_instances DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.notification_recipients NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.notification_recipients DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.alert_rules NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.alert_rules DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.services NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.services DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.nodes NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.nodes DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.project_members NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.project_members DISABLE ROW LEVEL SECURITY;
ALTER TABLE app.projects NO FORCE ROW LEVEL SECURITY;
ALTER TABLE app.projects DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS app.project_owner_context_matches(BIGINT);
DROP FUNCTION IF EXISTS app.project_write_context_matches(BIGINT);
DROP FUNCTION IF EXISTS app.project_read_context_matches(BIGINT);
DROP FUNCTION IF EXISTS app.internal_rls_bypass();
DROP FUNCTION IF EXISTS app.current_project_role();
DROP FUNCTION IF EXISTS app.current_project_id();
DROP FUNCTION IF EXISTS app.current_user_id();
