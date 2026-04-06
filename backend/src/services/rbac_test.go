package services

import "testing"

func TestRoleHasPermission(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		role       string
		permission string
		want       bool
	}{
		{
			name:       "owner has destructive project permission",
			role:       ProjectRoleOwner,
			permission: PermissionProjectDelete,
			want:       true,
		},
		{
			name:       "owner can manage members",
			role:       ProjectRoleOwner,
			permission: PermissionMembersManage,
			want:       true,
		},
		{
			name:       "admin has operational write permission",
			role:       ProjectRoleAdmin,
			permission: PermissionServicesWrite,
			want:       true,
		},
		{
			name:       "admin cannot delete project",
			role:       ProjectRoleAdmin,
			permission: PermissionProjectDelete,
			want:       false,
		},
		{
			name:       "viewer has read access",
			role:       ProjectRoleViewer,
			permission: PermissionEventsRead,
			want:       true,
		},
		{
			name:       "viewer cannot write nodes",
			role:       ProjectRoleViewer,
			permission: PermissionNodesWrite,
			want:       false,
		},
		{
			name:       "unknown role has no permissions",
			role:       "ghost",
			permission: PermissionProjectRead,
			want:       false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			if got := RoleHasPermission(testCase.role, testCase.permission); got != testCase.want {
				t.Fatalf("RoleHasPermission(%q, %q) = %t, want %t", testCase.role, testCase.permission, got, testCase.want)
			}
		})
	}
}
