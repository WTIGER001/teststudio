package authorization

import "slices"

const (
	ReaderRole      = "builtin.teststudio.reader"
	MemberRole      = "builtin.teststudio.member"
	DeveloperRole   = "builtin.teststudio.developer"
	ContributorRole = "builtin.teststudio.contributor"
	OwnerRole       = "builtin.teststudio.owner"
)

func StandardRole(id string) bool {
	return slices.Contains([]string{ReaderRole, MemberRole, DeveloperRole, ContributorRole, OwnerRole}, id)
}

func standardRoles() []RoleDefinition {
	read := []string{"projects.read", "applications.read", "catalog.read", "runs.read", "evidence.read", "evidence.download", "schedules.read", "environments.read", "sources.read", "suites.read", "notifications.read", "roles.read"}
	member := append(slices.Clone(read), "runs.create", "runs.cancel", "schedules.create", "schedules.update", "schedules.archive")
	developer := append(slices.Clone(member), "catalog.create", "catalog.update", "catalog.import", "catalog.export", "catalog.archive", "catalog.restore", "catalog.publish", "sources.manage", "suites.manage")
	contributor := append(slices.Clone(developer), "projects.manage", "applications.manage", "environments.manage", "credentials.read", "credentials.manage", "notifications.manage", "members.read", "members.manage", "groups.read", "groups.manage", "roles.manage", "permissions.read", "permissions.grant")
	contributor = append(contributor, "evidence.retain")
	owner := append(slices.Clone(contributor), "members.owners", "catalog.delete", "evidence.delete", "schedules.delete")
	return []RoleDefinition{
		{ID: ReaderRole, Name: "Reader", Permissions: read},
		{ID: MemberRole, Name: "Member", Permissions: member},
		{ID: DeveloperRole, Name: "Developer", Permissions: developer},
		{ID: ContributorRole, Name: "Contributor", Permissions: contributor},
		{ID: OwnerRole, Name: "Owner", Permissions: owner},
	}
}
