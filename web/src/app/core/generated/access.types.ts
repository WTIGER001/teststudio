export const accessContractVersion = "0.8.0"

export type Assignment = { "bindingValues": AssignmentBindings; "canRevoke": boolean; "managed": boolean; "principalActive": boolean; "principalId": string; "principalKind": "user" | "group"; "principalName": string; "roleDisabled": boolean; "roleId": string; "roleName": string }
export type AssignmentBindings = { "team"?: string }
export type AssignmentDefinition = { "principalId": string; "principalKind": "user" | "group"; "reason": string; "roleId": string }
export type AssignmentPage = { "cursor"?: string; "items": (AssignmentRecord)[]; "partial": boolean }
export type AssignmentRecord = { "data": Assignment; "id": string; "projectId"?: string; "revision": number; "updatedAt"?: string }
export type ExternalIdentity = { "issuer": string; "subject": string; "userId": string }
export type ExternalIdentityPage = { "cursor"?: string; "items": (ExternalIdentityRecord)[]; "partial": boolean }
export type ExternalIdentityRecord = { "data": ExternalIdentity; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type GrantDefinition = { "effect": "allow" | "deny"; "expiresAt"?: string | null; "object"?: string; "permission": string; "principalId": string; "principalKind": "user" | "group"; "reason": string }
export type GrantRecord = { "data": GrantView; "id": string; "projectId": string; "revision": number; "updatedAt": string }
export type GrantRemoved = { "removed": boolean }
export type GrantUpdate = { "expiresAt"?: string | null; "reason": string }
export type GrantView = { "canRemove": boolean; "canUpdate": boolean; "effect": "allow" | "deny"; "expired": boolean; "expiresAt"?: string | null; "object"?: string; "permission": string; "principalActive": boolean; "principalId": string; "principalKind": "user" | "group"; "principalName": string; "projectId": string }
export type GrantsPage = { "canCreate": boolean; "cursor"?: string; "items": (GrantRecord)[]; "partial": boolean }
export type Group = { "archived": boolean; "description": string; "id": string; "managedBy"?: string; "name": string; "projectId"?: string }
export type GroupDefinition = { "description"?: string; "id": string; "name": string; "projectId"?: string }
export type GroupPage = { "cursor"?: string; "items": (GroupRecord)[]; "partial": boolean }
export type GroupRecord = { "data": Group; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type LocalCredential = { "canManage": boolean; "configured": boolean; "createdAt"?: string; "disabled": boolean; "localAccountsEnabled": boolean; "mustChangePassword": boolean; "updatedAt"?: string; "userId": string }
export type LocalCredentialRecord = { "data": LocalCredential; "id": string; "revision": number }
export type LocalPasswordDefinition = { "password": string; "reason": string }
export type Membership = { "active": boolean; "expiresAt"?: string | null; "groupId": string; "groupName"?: string; "name": string; "projectId"?: string; "readOnly": boolean; "source": string; "userId": string }
export type MembershipDefinition = { "expiresAt"?: string | null }
export type MembershipPage = { "cursor"?: string; "items": (MembershipRecord)[]; "partial": boolean }
export type MembershipRecord = { "data": Membership; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type PasswordChange = { "currentPassword": string; "newPassword": string }
export type PasswordChanged = { "changed": boolean }
export type PasswordLogin = { "password": string; "userId": string }
export type PasswordSignedIn = { "mustChangePassword": boolean; "signedIn": boolean }
export type PermissionCatalog = { "items": (PermissionChoice)[] }
export type PermissionChoice = { "action": string; "area": string; "canGrant": boolean; "description": string; "id": string; "installationOnly": boolean }
export type PermissionComparison = { "after": boolean; "before": boolean; "permission": string }
export type PermissionDefinition = { "action": string; "area": string; "description": string; "id": string; "installationOnly": boolean }
export type PermissionExplanation = { "allowed": boolean; "canInspectOthers": boolean; "checkedAt": string; "cursor"?: string; "items": (PermissionSource)[]; "object": string; "partial": boolean; "permission": string; "permissions": (PermissionDefinition)[]; "projectId": string; "reason": "allowed" | "explicit-deny" | "no-matching-allow" | "user-disabled"; "userDisabled": boolean; "userId": string; "userName": string }
export type PermissionSource = { "assignedRoleId"?: string; "assignedRoleName"?: string; "assignmentProject"?: string; "effect": "allow" | "deny"; "expiresAt"?: string; "grantId"?: string; "id": string; "kind": "direct" | "role"; "objectScope"?: string; "principalId"?: string; "principalKind"?: "user" | "group"; "principalName"?: string; "projectId"?: string; "restricted": boolean; "restrictedFields"?: (string)[]; "roleId"?: string; "roleName"?: string; "scopeKnown": boolean; "status": "matching" | "expired"; "teamScope"?: string }
export type PredefinedRole = { "canManage": boolean; "disabled": boolean; "id": string; "name": string; "permissions": (string)[]; "scope": "project" | "installation" }
export type PredefinedRolePage = { "items": (PredefinedRoleRecord)[] }
export type PredefinedRoleRecord = { "data": PredefinedRole; "id": string; "revision": number; "updatedAt"?: string }
export type PredefinedRoleUpdate = { "disabled": boolean; "reason": string }
export type Problem = { "message": string; "status"?: number }
export type Profile = { "identities": ExternalIdentityPage; "localAccount": boolean; "user": User }
export type Reason = { "reason": string }
export type Revoked = { "revoked": boolean }
export type Role = { "builtIn": boolean; "delegatable": boolean; "disabled": boolean; "id": string; "name": string; "permissions": (string)[]; "scope": "project" | "installation" }
export type RoleChoicePage = { "cursor"?: string; "items": (Role)[]; "partial": boolean }
export type RolePage = { "items": (Role)[] }
export type RolePolicy = { "description": string; "disabled": boolean; "id": string; "inherits": (string)[]; "name": string; "projectId": string; "rules": (RoleRule)[] }
export type RolePolicyDetail = { "canManage": boolean; "data": RolePolicy; "delegatable": boolean; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type RolePolicyMutation = { "description"?: string; "disabled"?: boolean; "id": string; "inherits"?: (string)[]; "name": string; "projectId": string; "reason": string; "rules"?: (RoleRule)[] }
export type RolePolicyPage = { "canCreate": boolean; "cursor"?: string; "items": (RolePolicyRecord)[]; "partial": boolean }
export type RolePolicyRecord = { "data": RolePolicy; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type RolePreview = { "decisions": (PermissionComparison)[]; "object": string; "projectId": string; "userDisabled": boolean; "userId": string }
export type RolePreviewInput = { "object"?: string; "policy": RolePolicyMutation; "revision": number; "userId": string }
export type RoleRule = { "effect": "allow" | "deny"; "expiresAt"?: string | null; "object"?: string; "permission": string }
export type Saved = { "saved": boolean }
export type Session = { "authenticationUntil": string; "createdAt": string; "current": boolean; "expiresAt": string; "id": string; "lastSeen": string; "provider": string }
export type SessionPage = { "cursor"?: string; "items": (SessionRecord)[]; "partial": boolean }
export type SessionRecord = { "data": Session; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type User = { "createdAt": string; "disabled": boolean; "email"?: string; "emailVerified": boolean; "id": string; "lastSignIn"?: string; "name": string; "sessionEpoch": number }
export type UserDefinition = { "email"?: string; "id": string; "name": string }
export type UserPage = { "cursor"?: string; "items": (UserRecord)[] }
export type UserRecord = { "data": User; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }

export interface AccessRequests {
  listGroups: { query?: { "project"?: string; "archived"?: boolean; "limit"?: number; "cursor"?: string } }
  createGroup: { body: GroupDefinition }
  readGroup: { "group": string }
  editGroup: { "group": string; revision: number; body: GroupDefinition }
  archiveGroup: { "group": string; revision: number; body: Reason }
  listGroupMembers: { "group": string; query?: { "limit"?: number; "cursor"?: string } }
  removeLocalMembership: { "user": string; "group": string; revision: number; body: Reason }
  readGroupMember: { "group": string; "user": string; query?: { "source"?: string } }
  saveLocalMembership: { "user": string; "group": string; revision: number; body: MembershipDefinition }
  restoreGroup: { "group": string; revision: number; body: Reason }
  currentProfile: { query?: { "limit"?: number; "cursor"?: string } }
  explainOwnPermission: { query: { "project"?: string; "object"?: string; "permission": string; "limit"?: number; "cursor"?: string } }
  readOwnLocalCredential: {  }
  permissionCatalog: { query?: { "project"?: string; "object"?: string } }
  listInstallationGrants: { query?: { "limit"?: number; "cursor"?: string } }
  createInstallationGrant: { body: GrantDefinition }
  removeInstallationGrant: { "grant": string; revision: number; body: Reason }
  readInstallationGrant: { "grant": string }
  updateInstallationGrant: { "grant": string; revision: number; body: GrantUpdate }
  listPredefinedRoles: {  }
  readPredefinedRole: { "role": string }
  setPredefinedRoleState: { "role": string; revision: number; body: PredefinedRoleUpdate }
  listProjectGrants: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectGrant: { "project": string; body: GrantDefinition }
  removeProjectGrant: { "project": string; "grant": string; revision: number; body: Reason }
  readProjectGrant: { "project": string; "grant": string }
  updateProjectGrant: { "project": string; "grant": string; revision: number; body: GrantUpdate }
  listProjectAssignments: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectAssignment: { "project": string; body: AssignmentDefinition }
  revokeProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string }
  listInstallationAssignments: { query?: { "limit"?: number; "cursor"?: string } }
  createInstallationAssignment: { body: AssignmentDefinition }
  revokeInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string }
  listRoles: { query?: { "project"?: string } }
  createRolePolicy: { body: RolePolicyMutation }
  customRoleChoices: { query?: { "project"?: string; "limit"?: number; "cursor"?: string } }
  listRolePolicies: { query?: { "project"?: string; "limit"?: number; "cursor"?: string } }
  previewRolePolicy: { body: RolePreviewInput }
  readRolePolicy: { "role": string }
  editRolePolicy: { "role": string; revision: number; body: RolePolicyMutation }
  listUsers: { query?: { "limit"?: number; "cursor"?: string } }
  createUser: { body: UserDefinition }
  readUser: { "user": string }
  editUser: { "user": string; revision: number; body: UserDefinition }
  disableUser: { "user": string; revision: number; body: Reason }
  explainUserPermission: { "user": string; query: { "project"?: string; "object"?: string; "permission": string; "limit"?: number; "cursor"?: string } }
  listUserIdentities: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  readLocalCredential: { "user": string }
  setLocalPassword: { "user": string; revision: number; body: LocalPasswordDefinition }
  disableLocalCredential: { "user": string; revision: number; body: Reason }
  enableLocalCredential: { "user": string; revision: number; body: Reason }
  listUserMemberships: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  restoreUser: { "user": string; revision: number; body: Reason }
  listUserSessions: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  revokeUserSession: { "user": string; "session": string; revision: number; body: Reason }
  signInWithPassword: { body: PasswordLogin }
  changeLocalPassword: { body: PasswordChange }
}

export interface AccessResponses {
  listGroups: GroupPage
  createGroup: GroupRecord
  readGroup: GroupRecord
  editGroup: GroupRecord
  archiveGroup: GroupRecord
  listGroupMembers: MembershipPage
  removeLocalMembership: Saved
  readGroupMember: MembershipRecord
  saveLocalMembership: Saved
  restoreGroup: GroupRecord
  currentProfile: Profile
  explainOwnPermission: PermissionExplanation
  readOwnLocalCredential: LocalCredentialRecord
  permissionCatalog: PermissionCatalog
  listInstallationGrants: GrantsPage
  createInstallationGrant: GrantRecord
  removeInstallationGrant: GrantRemoved
  readInstallationGrant: GrantRecord
  updateInstallationGrant: GrantRecord
  listPredefinedRoles: PredefinedRolePage
  readPredefinedRole: PredefinedRoleRecord
  setPredefinedRoleState: PredefinedRoleRecord
  listProjectGrants: GrantsPage
  createProjectGrant: GrantRecord
  removeProjectGrant: GrantRemoved
  readProjectGrant: GrantRecord
  updateProjectGrant: GrantRecord
  listProjectAssignments: AssignmentPage
  createProjectAssignment: AssignmentRecord
  revokeProjectAssignment: Revoked
  readProjectAssignment: AssignmentRecord
  listInstallationAssignments: AssignmentPage
  createInstallationAssignment: AssignmentRecord
  revokeInstallationAssignment: Revoked
  readInstallationAssignment: AssignmentRecord
  listRoles: RolePage
  createRolePolicy: RolePolicyRecord
  customRoleChoices: RoleChoicePage
  listRolePolicies: RolePolicyPage
  previewRolePolicy: RolePreview
  readRolePolicy: RolePolicyDetail
  editRolePolicy: RolePolicyRecord
  listUsers: UserPage
  createUser: UserRecord
  readUser: UserRecord
  editUser: UserRecord
  disableUser: UserRecord
  explainUserPermission: PermissionExplanation
  listUserIdentities: ExternalIdentityPage
  readLocalCredential: LocalCredentialRecord
  setLocalPassword: LocalCredentialRecord
  disableLocalCredential: LocalCredentialRecord
  enableLocalCredential: LocalCredentialRecord
  listUserMemberships: MembershipPage
  restoreUser: UserRecord
  listUserSessions: SessionPage
  revokeUserSession: Revoked
  signInWithPassword: PasswordSignedIn
  changeLocalPassword: PasswordChanged
}

export const accessOperations = {
  listGroups: { method: "GET", path: "/api/v1/groups" },
  createGroup: { method: "POST", path: "/api/v1/groups" },
  readGroup: { method: "GET", path: "/api/v1/groups/{group}" },
  editGroup: { method: "PUT", path: "/api/v1/groups/{group}" },
  archiveGroup: { method: "POST", path: "/api/v1/groups/{group}/archive" },
  listGroupMembers: { method: "GET", path: "/api/v1/groups/{group}/members" },
  removeLocalMembership: { method: "DELETE", path: "/api/v1/groups/{group}/members/{user}" },
  readGroupMember: { method: "GET", path: "/api/v1/groups/{group}/members/{user}" },
  saveLocalMembership: { method: "PUT", path: "/api/v1/groups/{group}/members/{user}" },
  restoreGroup: { method: "POST", path: "/api/v1/groups/{group}/restore" },
  currentProfile: { method: "GET", path: "/api/v1/me" },
  explainOwnPermission: { method: "GET", path: "/api/v1/me/effective-permissions" },
  readOwnLocalCredential: { method: "GET", path: "/api/v1/me/local-credential" },
  permissionCatalog: { method: "GET", path: "/api/v1/permissions" },
  listInstallationGrants: { method: "GET", path: "/api/v1/permissions/grants" },
  createInstallationGrant: { method: "POST", path: "/api/v1/permissions/grants" },
  removeInstallationGrant: { method: "DELETE", path: "/api/v1/permissions/grants/{grant}" },
  readInstallationGrant: { method: "GET", path: "/api/v1/permissions/grants/{grant}" },
  updateInstallationGrant: { method: "PUT", path: "/api/v1/permissions/grants/{grant}" },
  listPredefinedRoles: { method: "GET", path: "/api/v1/predefined-roles" },
  readPredefinedRole: { method: "GET", path: "/api/v1/predefined-roles/{role}" },
  setPredefinedRoleState: { method: "PUT", path: "/api/v1/predefined-roles/{role}" },
  listProjectGrants: { method: "GET", path: "/api/v1/projects/{project}/permissions" },
  createProjectGrant: { method: "POST", path: "/api/v1/projects/{project}/permissions" },
  removeProjectGrant: { method: "DELETE", path: "/api/v1/projects/{project}/permissions/{grant}" },
  readProjectGrant: { method: "GET", path: "/api/v1/projects/{project}/permissions/{grant}" },
  updateProjectGrant: { method: "PUT", path: "/api/v1/projects/{project}/permissions/{grant}" },
  listProjectAssignments: { method: "GET", path: "/api/v1/projects/{project}/role-assignments" },
  createProjectAssignment: { method: "POST", path: "/api/v1/projects/{project}/role-assignments" },
  revokeProjectAssignment: { method: "DELETE", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  readProjectAssignment: { method: "GET", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  listInstallationAssignments: { method: "GET", path: "/api/v1/role-assignments" },
  createInstallationAssignment: { method: "POST", path: "/api/v1/role-assignments" },
  revokeInstallationAssignment: { method: "DELETE", path: "/api/v1/role-assignments/{kind}/{principal}/{role}" },
  readInstallationAssignment: { method: "GET", path: "/api/v1/role-assignments/{kind}/{principal}/{role}" },
  listRoles: { method: "GET", path: "/api/v1/roles" },
  createRolePolicy: { method: "POST", path: "/api/v1/roles" },
  customRoleChoices: { method: "GET", path: "/api/v1/roles/choices" },
  listRolePolicies: { method: "GET", path: "/api/v1/roles/custom" },
  previewRolePolicy: { method: "POST", path: "/api/v1/roles/preview" },
  readRolePolicy: { method: "GET", path: "/api/v1/roles/{role}" },
  editRolePolicy: { method: "PUT", path: "/api/v1/roles/{role}" },
  listUsers: { method: "GET", path: "/api/v1/users" },
  createUser: { method: "POST", path: "/api/v1/users" },
  readUser: { method: "GET", path: "/api/v1/users/{user}" },
  editUser: { method: "PUT", path: "/api/v1/users/{user}" },
  disableUser: { method: "POST", path: "/api/v1/users/{user}/disable" },
  explainUserPermission: { method: "GET", path: "/api/v1/users/{user}/effective-permissions" },
  listUserIdentities: { method: "GET", path: "/api/v1/users/{user}/identities" },
  readLocalCredential: { method: "GET", path: "/api/v1/users/{user}/local-credential" },
  setLocalPassword: { method: "PUT", path: "/api/v1/users/{user}/local-credential" },
  disableLocalCredential: { method: "POST", path: "/api/v1/users/{user}/local-credential/disable" },
  enableLocalCredential: { method: "POST", path: "/api/v1/users/{user}/local-credential/enable" },
  listUserMemberships: { method: "GET", path: "/api/v1/users/{user}/memberships" },
  restoreUser: { method: "POST", path: "/api/v1/users/{user}/restore" },
  listUserSessions: { method: "GET", path: "/api/v1/users/{user}/sessions" },
  revokeUserSession: { method: "DELETE", path: "/api/v1/users/{user}/sessions/{session}" },
  signInWithPassword: { method: "POST", path: "/auth/password" },
  changeLocalPassword: { method: "POST", path: "/auth/password/change" }
} as const
