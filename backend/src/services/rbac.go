package services

const (
	ProjectRoleOwner  = "owner"
	ProjectRoleAdmin  = "admin"
	ProjectRoleViewer = "viewer"
)

const (
	PermissionProjectRead                 = "project.read"
	PermissionProjectUpdate               = "project.update"
	PermissionProjectDelete               = "project.delete"
	PermissionMembersRead                 = "members.read"
	PermissionMembersManage               = "members.manage"
	PermissionNodesRead                   = "nodes.read"
	PermissionNodesWrite                  = "nodes.write"
	PermissionServicesRead                = "services.read"
	PermissionServicesWrite               = "services.write"
	PermissionAlertsRead                  = "alerts.read"
	PermissionAlertsWrite                 = "alerts.write"
	PermissionNotificationRecipientsRead  = "notification_recipients.read"
	PermissionNotificationRecipientsWrite = "notification_recipients.write"
	PermissionAgentTokensWrite            = "agent_tokens.write"
	PermissionEventsRead                  = "events.read"
)

var allProjectPermissions = []string{
	PermissionProjectRead,
	PermissionProjectUpdate,
	PermissionProjectDelete,
	PermissionMembersRead,
	PermissionMembersManage,
	PermissionNodesRead,
	PermissionNodesWrite,
	PermissionServicesRead,
	PermissionServicesWrite,
	PermissionAlertsRead,
	PermissionAlertsWrite,
	PermissionNotificationRecipientsRead,
	PermissionNotificationRecipientsWrite,
	PermissionAgentTokensWrite,
	PermissionEventsRead,
}

var rolePermissions = map[string]map[string]struct{}{
	ProjectRoleOwner: permissionSet(allProjectPermissions...),
	ProjectRoleAdmin: permissionSet(
		PermissionProjectRead,
		PermissionProjectUpdate,
		PermissionMembersRead,
		PermissionNodesRead,
		PermissionNodesWrite,
		PermissionServicesRead,
		PermissionServicesWrite,
		PermissionAlertsRead,
		PermissionAlertsWrite,
		PermissionNotificationRecipientsRead,
		PermissionNotificationRecipientsWrite,
		PermissionAgentTokensWrite,
		PermissionEventsRead,
	),
	ProjectRoleViewer: permissionSet(
		PermissionProjectRead,
		PermissionNodesRead,
		PermissionServicesRead,
		PermissionAlertsRead,
		PermissionNotificationRecipientsRead,
		PermissionEventsRead,
	),
}

func ProjectPermissions() []string {
	permissions := make([]string, 0, len(allProjectPermissions))
	permissions = append(permissions, allProjectPermissions...)
	return permissions
}

func RolePermissions(role string) []string {
	permissionSetForRole, ok := rolePermissions[role]
	if !ok {
		return nil
	}

	permissions := make([]string, 0, len(allProjectPermissions))
	for _, permission := range allProjectPermissions {
		if _, ok := permissionSetForRole[permission]; ok {
			permissions = append(permissions, permission)
		}
	}

	return permissions
}

func RoleHasPermission(role, permission string) bool {
	permissionSetForRole, ok := rolePermissions[role]
	if !ok {
		return false
	}

	_, ok = permissionSetForRole[permission]
	return ok
}

func IsValidProjectRole(role string) bool {
	_, ok := rolePermissions[role]
	return ok
}

func permissionSet(permissions ...string) map[string]struct{} {
	set := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		set[permission] = struct{}{}
	}

	return set
}
