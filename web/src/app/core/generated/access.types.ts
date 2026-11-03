export const accessContractVersion = "0.3.0"

export type Assignment = { "bindingValues": AssignmentBindings; "canRevoke": boolean; "managed": boolean; "principalActive": boolean; "principalId": string; "principalKind": "user" | "group"; "principalName": string; "roleDisabled": boolean; "roleId": string; "roleName": string }
export type AssignmentBindings = { "team"?: string }
export type AssignmentDefinition = { "principalId": string; "principalKind": "user" | "group"; "reason": string; "roleId": string }
export type AssignmentPage = { "cursor"?: string; "items": (AssignmentRecord)[]; "partial": boolean }
export type AssignmentRecord = { "data": Assignment; "id": string; "projectId"?: string; "revision": number; "updatedAt"?: string }
export type ExternalIdentity = { "issuer": string; "subject": string; "userId": string }
export type ExternalIdentityPage = { "cursor"?: string; "items": (ExternalIdentityRecord)[]; "partial": boolean }
export type ExternalIdentityRecord = { "data": ExternalIdentity; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
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
export type Problem = { "message": string; "status"?: number }
export type Profile = { "identities": ExternalIdentityPage; "localAccount": boolean; "user": User }
export type Reason = { "reason": string }
export type Revoked = { "revoked": boolean }
export type Role = { "builtIn": boolean; "delegatable": boolean; "disabled": boolean; "id": string; "name": string; "permissions": (string)[]; "scope": "project" | "installation" }
export type RolePage = { "items": (Role)[] }
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
  readOwnLocalCredential: {  }
  listProjectAssignments: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectAssignment: { "project": string; body: AssignmentDefinition }
  revokeProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string }
  listInstallationAssignments: { query?: { "limit"?: number; "cursor"?: string } }
  createInstallationAssignment: { body: AssignmentDefinition }
  revokeInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string }
  listRoles: { query?: { "project"?: string } }
  listUsers: { query?: { "limit"?: number; "cursor"?: string } }
  createUser: { body: UserDefinition }
  readUser: { "user": string }
  editUser: { "user": string; revision: number; body: UserDefinition }
  disableUser: { "user": string; revision: number; body: Reason }
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
  readOwnLocalCredential: LocalCredentialRecord
  listProjectAssignments: AssignmentPage
  createProjectAssignment: AssignmentRecord
  revokeProjectAssignment: Revoked
  readProjectAssignment: AssignmentRecord
  listInstallationAssignments: AssignmentPage
  createInstallationAssignment: AssignmentRecord
  revokeInstallationAssignment: Revoked
  readInstallationAssignment: AssignmentRecord
  listRoles: RolePage
  listUsers: UserPage
  createUser: UserRecord
  readUser: UserRecord
  editUser: UserRecord
  disableUser: UserRecord
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
  readOwnLocalCredential: { method: "GET", path: "/api/v1/me/local-credential" },
  listProjectAssignments: { method: "GET", path: "/api/v1/projects/{project}/role-assignments" },
  createProjectAssignment: { method: "POST", path: "/api/v1/projects/{project}/role-assignments" },
  revokeProjectAssignment: { method: "DELETE", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  readProjectAssignment: { method: "GET", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  listInstallationAssignments: { method: "GET", path: "/api/v1/role-assignments" },
  createInstallationAssignment: { method: "POST", path: "/api/v1/role-assignments" },
  revokeInstallationAssignment: { method: "DELETE", path: "/api/v1/role-assignments/{kind}/{principal}/{role}" },
  readInstallationAssignment: { method: "GET", path: "/api/v1/role-assignments/{kind}/{principal}/{role}" },
  listRoles: { method: "GET", path: "/api/v1/roles" },
  listUsers: { method: "GET", path: "/api/v1/users" },
  createUser: { method: "POST", path: "/api/v1/users" },
  readUser: { method: "GET", path: "/api/v1/users/{user}" },
  editUser: { method: "PUT", path: "/api/v1/users/{user}" },
  disableUser: { method: "POST", path: "/api/v1/users/{user}/disable" },
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
