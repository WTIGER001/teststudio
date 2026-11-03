export const accessContractVersion = "0.1.0"

export type ExternalIdentity = { "issuer": string; "subject": string; "userId": string }
export type ExternalIdentityPage = { "cursor"?: string; "items": (ExternalIdentityRecord)[]; "partial": boolean }
export type ExternalIdentityRecord = { "data": ExternalIdentity; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type Group = { "archived": boolean; "description": string; "id": string; "managedBy"?: string; "name": string; "projectId"?: string }
export type GroupDefinition = { "description"?: string; "id": string; "name": string; "projectId"?: string }
export type GroupPage = { "cursor"?: string; "items": (GroupRecord)[]; "partial": boolean }
export type GroupRecord = { "data": Group; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type Membership = { "active": boolean; "expiresAt"?: string | null; "groupId": string; "groupName"?: string; "name": string; "projectId"?: string; "readOnly": boolean; "source": string; "userId": string }
export type MembershipDefinition = { "expiresAt"?: string | null }
export type MembershipPage = { "cursor"?: string; "items": (MembershipRecord)[]; "partial": boolean }
export type MembershipRecord = { "data": Membership; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type Problem = { "message": string; "status"?: number }
export type Profile = { "identities": ExternalIdentityPage; "localAccount": boolean; "user": User }
export type Reason = { "reason": string }
export type Revoked = { "revoked": boolean }
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
  listUsers: { query?: { "limit"?: number; "cursor"?: string } }
  createUser: { body: UserDefinition }
  readUser: { "user": string }
  editUser: { "user": string; revision: number; body: UserDefinition }
  disableUser: { "user": string; revision: number; body: Reason }
  listUserIdentities: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  listUserMemberships: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  restoreUser: { "user": string; revision: number; body: Reason }
  listUserSessions: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  revokeUserSession: { "user": string; "session": string; revision: number; body: Reason }
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
  listUsers: UserPage
  createUser: UserRecord
  readUser: UserRecord
  editUser: UserRecord
  disableUser: UserRecord
  listUserIdentities: ExternalIdentityPage
  listUserMemberships: MembershipPage
  restoreUser: UserRecord
  listUserSessions: SessionPage
  revokeUserSession: Revoked
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
  listUsers: { method: "GET", path: "/api/v1/users" },
  createUser: { method: "POST", path: "/api/v1/users" },
  readUser: { method: "GET", path: "/api/v1/users/{user}" },
  editUser: { method: "PUT", path: "/api/v1/users/{user}" },
  disableUser: { method: "POST", path: "/api/v1/users/{user}/disable" },
  listUserIdentities: { method: "GET", path: "/api/v1/users/{user}/identities" },
  listUserMemberships: { method: "GET", path: "/api/v1/users/{user}/memberships" },
  restoreUser: { method: "POST", path: "/api/v1/users/{user}/restore" },
  listUserSessions: { method: "GET", path: "/api/v1/users/{user}/sessions" },
  revokeUserSession: { method: "DELETE", path: "/api/v1/users/{user}/sessions/{session}" }
} as const
