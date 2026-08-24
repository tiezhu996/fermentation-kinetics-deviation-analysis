package constants

import "testing"

func TestAnalystCannotConfirm(t *testing.T) {
	if HasPermission(RoleDataAnalyst, PermissionAnalysisConfirm) {
		t.Fatal("data analyst must not hold the analysis confirm permission")
	}
}
