package auth

import "testing"

func TestHasPermission_NormalizesHyphenatedRoleSlugs(t *testing.T) {
	if !HasPermission([]string{"tenant-admin"}, PermUserWrite) {
		t.Fatal("expected tenant-admin slug to grant tenant_admin permissions")
	}
	if !HasPermission([]string{"super-admin"}, PermAdminAll) {
		t.Fatal("expected super-admin slug to grant super_admin permissions")
	}
}
