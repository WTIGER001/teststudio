export const accessContractVersion = "0.14.0"

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
export type GroupDefinition = { "description"?: string; "id": string; "managedBy"?: string; "name": string; "projectId"?: string }
export type GroupPage = { "cursor"?: string; "items": (GroupRecord)[]; "partial": boolean }
export type GroupRecord = { "data": Group; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type IdentityLinkOperation = { "expiresAt": string; "id": string; "identity"?: IdentityLinkProof; "reason": string; "revision": number; "sourceMethod": string; "sourceProvider"?: IdentityLinkProvider; "state": "authenticate-source" | "prove-target" | "review" | "complete" | "cancelled" | "failed" | "expired"; "targetProvider": IdentityLinkProvider }
export type IdentityLinkOptions = { "canStart": boolean; "providers": (IdentityLinkProvider)[]; "sourceMethod": string }
export type IdentityLinkProof = { "issuer": string; "subject": string }
export type IdentityLinkProvider = { "id": string; "issuer": string; "name": string; "revision": number }
export type IdentityLinkStart = { "confirmLocal"?: boolean; "password"?: string; "providerId": string; "reason": string }
export type LocalCredential = { "canManage": boolean; "configured": boolean; "createdAt"?: string; "disabled": boolean; "localAccountsEnabled": boolean; "mustChangePassword": boolean; "updatedAt"?: string; "userId": string }
export type LocalCredentialRecord = { "data": LocalCredential; "id": string; "revision": number }
export type LocalPasswordDefinition = { "password": string; "reason": string }
export type LogoutResponse = { "providerLogout"?: ProviderLogoutOffer; "providerLogoutUnavailable"?: boolean; "signedOut": boolean }
export type Membership = { "active": boolean; "expiresAt"?: string | null; "groupId": string; "groupName"?: string; "name": string; "projectId"?: string; "providerRevision"?: number; "readOnly": boolean; "source": string; "userId": string }
export type MembershipDefinition = { "expiresAt"?: string | null }
export type MembershipPage = { "cursor"?: string; "items": (MembershipRecord)[]; "partial": boolean }
export type MembershipRecord = { "data": Membership; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type PasswordChange = { "currentPassword": string; "newPassword": string }
export type PasswordChanged = { "changed": boolean }
export type PasswordLogin = { "password": string; "userId": string }
export type PasswordSignedIn = { "mustChangePassword": boolean; "signedIn": boolean }
export type PendingAccess = { "activatedAt"?: string; "activatedUserId"?: string; "approvedBy": string; "assignmentId"?: string; "blockReason"?: "provider-changed" | "role-changed" | "authority-changed" | "project-unavailable"; "canCancel": boolean; "createdAt": string; "email": string; "id": string; "issuer": string; "projectId": string; "providerId": string; "providerName": string; "providerRevision": number; "reason": string; "requireVerifiedEmail": boolean; "roleId": string; "roleName": string; "state": "pending" | "activated" | "cancelled" | "blocked" }
export type PendingAccessInput = { "email": string; "providerId": string; "providerRevision": number; "reason": string; "roleId": string }
export type PendingAccessPage = { "cursor"?: string; "items": (PendingAccessRecord)[]; "partial": boolean }
export type PendingAccessRecord = { "data": PendingAccess; "id": string; "revision": number }
export type PendingProviderChoice = { "autoProvision": boolean; "id": string; "issuer": string; "name": string; "revision": number }
export type PendingProviderChoices = { "items": (PendingProviderChoice)[] }
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
export type Provider = { "autoProvision": boolean; "callbackUrl": string; "canManage": boolean; "clientId": string; "disabled": boolean; "groupClaim"?: string; "groupMappings": ProviderMappings; "id": string; "issuer": string; "logoutCallbackUrl": string; "logoutEndpoint"?: string; "membershipTtlSeconds": number; "name": string; "scopes": (string)[]; "secretRef"?: string }
export type ProviderLogoutOffer = { "expiresAt": string; "providerName": string; "url": string }
export type ProviderMappings = { [key: string]: string }
export type ProviderMutation = { "autoProvision": boolean; "clientId": string; "disabled": boolean; "groupClaim"?: string; "groupMappings": ProviderMappings; "id": string; "issuer": string; "logoutEndpoint"?: string; "membershipTtlSeconds": number; "name": string; "reason": string; "scopes": (string)[]; "secretRef"?: string }
export type ProviderPage = { "canCreate": boolean; "credentialRefs": (string)[]; "cursor"?: string; "items": (ProviderRecord)[]; "partial": boolean }
export type ProviderRecord = { "data": Provider; "id": string; "revision": number; "updatedAt": string }
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
export type SourceCapturePolicy = { "failureRetentionDays"?: number; "kindRetentionDays"?: { [key: string]: number }; "logs": boolean; "maxBytes"?: number; "maxFileBytes"?: number; "retentionDays"?: number; "screenshots": string; "trace": string; "video": string }
export type SourceCatalogArtifact = { "discoveredSelectors"?: { [key: string]: string }; "image"?: string; "provenance"?: SourceOrigin; "sourceProjectId"?: string; "sourceRevision"?: string; "sourceSnapshotId"?: string }
export type SourceCatalogDraft = { "artifact": SourceCatalogArtifact; "capture": SourceCapturePolicy; "id": string; "publishedReleaseId"?: string; "tests": (SourceCatalogTest)[]; "timing": SourceTimingPolicy }
export type SourceCatalogHTTPCheck = { "endpoint": string; "forbiddenText"?: string; "headers"?: { [key: string]: string }; "json"?: { [key: string]: SourceJSONValue }; "maxResponseMs"?: number; "method": string; "minCertificateDays"?: number; "path": string; "requiredText"?: string; "statusCodes": (number)[] }
export type SourceCatalogImportPlan = { "actor": string; "appliedAt"?: string; "artifact"?: SourceCatalogArtifact; "catalogRevision": number; "changes": (SourceCatalogPlannedTest)[]; "createdAt": string; "expiresAt": string; "id": string; "issues": (SourceCatalogIssue)[]; "kind": string; "projectId": string; "sourceSnapshotId"?: string; "suiteId": string; "suiteRevision": number }
export type SourceCatalogIssue = { "field": string; "message": string; "testId"?: string }
export type SourceCatalogPlannedTest = { "action": string; "before"?: SourceCatalogTest; "expectedRevision": number; "test": SourceCatalogTest }
export type SourceCatalogRelease = { "activatedOnPublication": boolean; "actor": string; "catalog": SourceCatalogDraft; "createdAt": string; "draftRevision": number; "id": string; "projectId": string; "suiteId": string }
export type SourceCatalogTest = { "applicationId"?: string; "capture"?: SourceCapturePolicy; "category": string; "check"?: SourceCatalogHTTPCheck; "conditions"?: string; "definitionSourceId"?: string; "dependencies": (string)[] | null; "description": string; "id": string; "owner"?: string; "rationale": string; "requiredBindings": (string)[] | null; "runner": string; "safety": string; "selector"?: string; "source"?: string; "state": string; "tags": (string)[] | null; "timeoutSeconds": number; "timing"?: SourceTimingPolicy; "title": string; "type": string }
export type SourceContent = { "content": string; "digest": string }
export type SourceJSONValue = null | string | number | boolean | (SourceJSONValue)[] | { [key: string]: SourceJSONValue }
export type SourceOrigin = { "commit"?: string; "reference"?: string; "repository"?: string; "snapshotDigest"?: string }
export type SourcePrebuiltInput = { "image": string; "runner": string }
export type SourcePreparationJob = { "actor": string; "cancelRequested": boolean; "cleanupPending": boolean; "createdAt": string; "error"?: string; "executionLocation": string; "finishedAt"?: string; "id": string; "image": string; "leaseUntil"?: string; "localImageId"?: string; "log"?: string; "logTruncated"?: boolean; "phase": string; "projectId": string; "reference"?: SourcePreparationReference; "releaseId"?: string; "runner": string; "runtime"?: string; "startedAt"?: string; "state": string; "suiteId": string; "uploadId"?: string; "worker"?: string }
export type SourcePreparationOptions = { "executionLocation": string; "runners": (string)[] }
export type SourcePreparationPage = { "cursor"?: string; "items": (SourcePreparationRecord)[] }
export type SourcePreparationRecord = { "data": SourcePreparationJob; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SourcePreparationReference = { "attempt": number; "container": string; "id": string; "installationId": string; "profile": string; "projectId": string; "runId": string; "testId": string; "volume": string; "workerId": string }
export type SourcePublishInput = { "activate": boolean }
export type SourceTimingPolicy = { "maxAgeDays": number; "minimumIncreaseMs": number; "minimumSamples": number; "minimumTimeoutMs": number; "mode": string; "slowdownFactor": number; "timeoutFactor": number; "timeoutMarginMs": number; "timeoutMode": string; "window": number }
export type SourceUpload = { "activated": boolean; "actor": string; "artifact"?: SourceCatalogArtifact; "bytes": number; "createdAt": string; "digest": string; "expiresAt": string; "format": "json" | "yaml"; "id": string; "issues": (SourceCatalogIssue)[]; "kind"?: string; "name": string; "planId"?: string; "preparationId"?: string; "projectId": string; "releaseId"?: string; "state": "staged" | "failed" | "published" | "expired" | "cancelled"; "suiteId": string }
export type SourceUploadInput = { "content": string; "format": "json" | "yaml"; "name": string }
export type SourceUploadPage = { "cursor"?: string; "items": (SourceUploadRecord)[] }
export type SourceUploadPreview = { "plan"?: SourceCatalogImportPlan; "upload": SourceUploadRecord }
export type SourceUploadRecord = { "data": SourceUpload; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
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
  listIdentityProviders: { query?: { "limit"?: number; "cursor"?: string } }
  createIdentityProvider: { body: ProviderMutation }
  readIdentityProvider: { "provider": string }
  editIdentityProvider: { "provider": string; revision: number; body: ProviderMutation }
  currentProfile: { query?: { "limit"?: number; "cursor"?: string } }
  explainOwnPermission: { query: { "project"?: string; "object"?: string; "permission": string; "limit"?: number; "cursor"?: string } }
  identityLinkOptions: {  }
  startIdentityLink: { body: IdentityLinkStart }
  readIdentityLink: { "operation": string }
  cancelIdentityLink: { "operation": string; revision: number }
  confirmIdentityLink: { "operation": string; revision: number }
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
  listPendingAccess: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createPendingAccess: { "project": string; body: PendingAccessInput }
  pendingAccessProviders: { "project": string }
  readPendingAccess: { "project": string; "access": string }
  cancelPendingAccess: { "project": string; "access": string; revision: number; body: Reason }
  listProjectGrants: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectGrant: { "project": string; body: GrantDefinition }
  removeProjectGrant: { "project": string; "grant": string; revision: number; body: Reason }
  readProjectGrant: { "project": string; "grant": string }
  updateProjectGrant: { "project": string; "grant": string; revision: number; body: GrantUpdate }
  listProjectAssignments: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectAssignment: { "project": string; body: AssignmentDefinition }
  revokeProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string }
  readPreparationOptions: { "project": string; "suite": string }
  listSourcePreparations: { "project": string; "suite": string; query?: { "limit"?: number; "cursor"?: string } }
  enqueuePrebuiltPreparation: { "project": string; "suite": string; key: string; body: SourcePrebuiltInput }
  readSourcePreparation: { "project": string; "suite": string; "id": string }
  cancelSourcePreparation: { "project": string; "suite": string; "id": string; revision: number }
  listSourceUploads: { "project": string; "suite": string; query?: { "limit"?: number; "cursor"?: string } }
  stageSourceUpload: { "project": string; "suite": string; body: SourceUploadInput }
  readSourceUpload: { "project": string; "suite": string; "id": string }
  readSourceContent: { "project": string; "suite": string; "id": string }
  publishSourceUpload: { "project": string; "suite": string; "id": string; revision: number; body: SourcePublishInput }
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
  signOut: {  }
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
  listIdentityProviders: ProviderPage
  createIdentityProvider: ProviderRecord
  readIdentityProvider: ProviderRecord
  editIdentityProvider: ProviderRecord
  currentProfile: Profile
  explainOwnPermission: PermissionExplanation
  identityLinkOptions: IdentityLinkOptions
  startIdentityLink: IdentityLinkOperation
  readIdentityLink: IdentityLinkOperation
  cancelIdentityLink: IdentityLinkOperation
  confirmIdentityLink: IdentityLinkOperation
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
  listPendingAccess: PendingAccessPage
  createPendingAccess: PendingAccessRecord
  pendingAccessProviders: PendingProviderChoices
  readPendingAccess: PendingAccessRecord
  cancelPendingAccess: PendingAccessRecord
  listProjectGrants: GrantsPage
  createProjectGrant: GrantRecord
  removeProjectGrant: GrantRemoved
  readProjectGrant: GrantRecord
  updateProjectGrant: GrantRecord
  listProjectAssignments: AssignmentPage
  createProjectAssignment: AssignmentRecord
  revokeProjectAssignment: Revoked
  readProjectAssignment: AssignmentRecord
  readPreparationOptions: SourcePreparationOptions
  listSourcePreparations: SourcePreparationPage
  enqueuePrebuiltPreparation: SourcePreparationRecord
  readSourcePreparation: SourcePreparationRecord
  cancelSourcePreparation: SourcePreparationRecord
  listSourceUploads: SourceUploadPage
  stageSourceUpload: SourceUploadPreview
  readSourceUpload: SourceUploadPreview
  readSourceContent: SourceContent
  publishSourceUpload: SourceCatalogRelease
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
  signOut: LogoutResponse
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
  listIdentityProviders: { method: "GET", path: "/api/v1/identity-providers" },
  createIdentityProvider: { method: "POST", path: "/api/v1/identity-providers" },
  readIdentityProvider: { method: "GET", path: "/api/v1/identity-providers/{provider}" },
  editIdentityProvider: { method: "PUT", path: "/api/v1/identity-providers/{provider}" },
  currentProfile: { method: "GET", path: "/api/v1/me" },
  explainOwnPermission: { method: "GET", path: "/api/v1/me/effective-permissions" },
  identityLinkOptions: { method: "GET", path: "/api/v1/me/identity-linking" },
  startIdentityLink: { method: "POST", path: "/api/v1/me/identity-links" },
  readIdentityLink: { method: "GET", path: "/api/v1/me/identity-links/{operation}" },
  cancelIdentityLink: { method: "POST", path: "/api/v1/me/identity-links/{operation}/cancel" },
  confirmIdentityLink: { method: "POST", path: "/api/v1/me/identity-links/{operation}/confirm" },
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
  listPendingAccess: { method: "GET", path: "/api/v1/projects/{project}/pending-access" },
  createPendingAccess: { method: "POST", path: "/api/v1/projects/{project}/pending-access" },
  pendingAccessProviders: { method: "GET", path: "/api/v1/projects/{project}/pending-access/providers" },
  readPendingAccess: { method: "GET", path: "/api/v1/projects/{project}/pending-access/{access}" },
  cancelPendingAccess: { method: "POST", path: "/api/v1/projects/{project}/pending-access/{access}/cancel" },
  listProjectGrants: { method: "GET", path: "/api/v1/projects/{project}/permissions" },
  createProjectGrant: { method: "POST", path: "/api/v1/projects/{project}/permissions" },
  removeProjectGrant: { method: "DELETE", path: "/api/v1/projects/{project}/permissions/{grant}" },
  readProjectGrant: { method: "GET", path: "/api/v1/projects/{project}/permissions/{grant}" },
  updateProjectGrant: { method: "PUT", path: "/api/v1/projects/{project}/permissions/{grant}" },
  listProjectAssignments: { method: "GET", path: "/api/v1/projects/{project}/role-assignments" },
  createProjectAssignment: { method: "POST", path: "/api/v1/projects/{project}/role-assignments" },
  revokeProjectAssignment: { method: "DELETE", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  readProjectAssignment: { method: "GET", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  readPreparationOptions: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparation-options" },
  listSourcePreparations: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations" },
  enqueuePrebuiltPreparation: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations" },
  readSourcePreparation: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}" },
  cancelSourcePreparation: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}/cancel" },
  listSourceUploads: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads" },
  stageSourceUpload: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads" },
  readSourceUpload: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}" },
  readSourceContent: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}/content" },
  publishSourceUpload: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}/publish" },
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
  signOut: { method: "POST", path: "/auth/logout" },
  signInWithPassword: { method: "POST", path: "/auth/password" },
  changeLocalPassword: { method: "POST", path: "/auth/password/change" }
} as const
