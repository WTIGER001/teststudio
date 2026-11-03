package authorization

import (
	"fmt"
	"strings"
)

type PermissionDefinition struct {
	ID               string `json:"id"`
	Area             string `json:"area"`
	Action           string `json:"action"`
	Description      string `json:"description"`
	InstallationOnly bool   `json:"installationOnly"`
}

func InstallationPermission(name string) bool {
	for _, prefix := range []string{"teststudio.users.", "teststudio.providers.", "teststudio.workers.", "teststudio.system."} {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return name == "teststudio.projects.create"
}

func PermissionDefinitions() []PermissionDefinition {
	areas := map[string]string{"catalog": "test catalog entries", "runs": "test runs", "evidence": "run evidence", "schedules": "run schedules", "environments": "test environments", "sources": "registered test sources", "suites": "test suites", "credentials": "credential binding references", "notifications": "completion notifications", "projects": "project settings", "members": "scoped role assignments", "roles": "custom role policies", "permissions": "access permissions", "users": "installation user profiles", "groups": "owned groups and local membership", "providers": "sign-in providers and mappings", "workers": "installation workers", "system": "installation settings"}
	verbs := map[string]string{"read": "Read %s.", "create": "Create %s.", "update": "Update %s.", "import": "Import %s.", "export": "Export %s.", "publish": "Publish and activate %s.", "archive": "Archive %s while retaining history.", "restore": "Restore archived %s.", "delete": "Permanently delete %s.", "cancel": "Cancel %s.", "manage": "Manage %s.", "download": "Download %s.", "retain": "Keep %s beyond ordinary retention.", "grant": "Delegate %s.", "override-capture": "Override %s capture defaults."}
	result := []PermissionDefinition{}
	for _, id := range PermissionNames() {
		parts := strings.Split(id, ".")
		area, action := parts[1], parts[2]
		description := "Start tests in protected environments."
		if action != "execute-protected" {
			description = fmt.Sprintf(verbs[action], areas[area])
		}
		result = append(result, PermissionDefinition{ID: id, Area: area, Action: action, Description: description, InstallationOnly: InstallationPermission(id)})
	}
	return result
}
