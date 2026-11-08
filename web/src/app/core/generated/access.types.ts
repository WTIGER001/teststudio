export const accessContractVersion = "0.39.0"

export type APIAuthentication = { "binding": string; "header"?: string; "scheme": "bearer" | "basic" | "header" }
export type APIRequestPolicy = { "authentication"?: APIAuthentication; "maxRequests": number; "maxResponseBytes": number; "redactFields"?: (string)[]; "rules": (APIRequestRule)[] }
export type APIRequestRule = { "maxBodyBytes"?: number; "method": "GET" | "HEAD" | "OPTIONS" | "POST" | "PUT" | "PATCH" | "DELETE"; "pathPrefix": string; "readOnly"?: boolean }
export type Application = { "description": string; "id": string; "name": string }
export type ApplicationRecord = { "data": Application; "id": string; "projectId": string; "revision": number; "updatedAt": string }
export type Assignment = { "bindingValues": AssignmentBindings; "canRevoke": boolean; "managed": boolean; "principalActive": boolean; "principalId": string; "principalKind": "user" | "group"; "principalName": string; "roleDisabled": boolean; "roleId": string; "roleName": string }
export type AssignmentBindings = { "application"?: string; "team"?: string }
export type AssignmentDefinition = { "principalId": string; "principalKind": "user" | "group"; "reason": string; "roleId": string }
export type AssignmentPage = { "cursor"?: string; "items": (AssignmentRecord)[]; "partial": boolean }
export type AssignmentRecord = { "applicationId"?: string; "data": Assignment; "id": string; "projectId"?: string; "revision": number; "updatedAt"?: string }
export type BrowserRecording = { "actor": string; "code"?: string; "createdAt": string; "deadlineAt": string; "error"?: string; "id": string; "mode"?: "companion"; "projectId": string; "state": string; "suiteId": string; "testId": string; "title": string; "url": string }
export type BrowserRecordingInput = { "id": string; "testId": string; "title": string; "url": string }
export type BrowserRecordingOptions = { "durationLimitSeconds": number; "enabled": boolean; "mode": string }
export type BrowserRecordingPage = { "cursor"?: string; "items": (BrowserRecordingRecord)[] }
export type BrowserRecordingPairing = { "expiresAt": string; "id": string; "token": string }
export type BrowserRecordingRecord = { "data": BrowserRecording; "id": string; "revision": number; "updatedAt": string }
export type CLIConfirmation = { "approve": boolean; "clientName": string; "code": string }
export type CLIExchange = { "session"?: CLIGrant; "state": string }
export type CLIGrant = { "cookie": string; "csrfToken": string; "expiresAt": string }
export type CLIOptions = { "lifetimeSeconds": number; "pollSeconds": number; "serverTime": string }
export type CLIProof = { "clientName": string; "requestId": string; "verifier": string }
export type CLIView = { "clientName": string; "expiresAt": string; "id": string; "state": string }
export type Capabilities = { [key: string]: boolean }
export type CatalogBulkMetadata = { "category"?: string | null; "conditions"?: string | null; "description"?: string | null; "owner"?: string | null; "rationale"?: string | null; "safety"?: string | null; "tags"?: (string)[] | null; "title"?: string | null }
export type CatalogBulkRequest = { "action": "metadata" | "disable" | "enable" | "archive" | "restore"; "metadata"?: CatalogBulkMetadata; "tests": (CatalogBulkTarget)[] }
export type CatalogBulkTarget = { "id": string; "revision": number }
export type CatalogCategoryNode = { "children": (CatalogCategoryNode)[]; "count": number; "label": string; "path": string }
export type CatalogCounts = { "enabled": number; "safe": number; "suites": number; "total": number }
export type CatalogDocument = { "exportedAt"?: string; "originProjectId"?: string; "originSuiteId"?: string; "schemaVersion": "teststudio.catalog/v1"; "tests": (SourceCatalogTest)[] }
export type CatalogDraftPage = { "items": (CatalogDraftRecord)[] }
export type CatalogDraftRecord = { "data": SourceCatalogDraft; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type CatalogDuplicateInput = { "id": string }
export type CatalogListEntry = { "data": SourceCatalogTest; "id": string; "projectId": string; "revision": number; "suiteId": string; "suiteName": string; "updatedAt": string }
export type CatalogListOptions = { "application": (string)[]; "owner": (string)[]; "runner": (string)[]; "safety": (string)[]; "source": (string)[]; "tag": (string)[]; "type": (string)[] }
export type CatalogListPage = { "categories": (CatalogCategoryNode)[]; "counts": CatalogCounts; "cursor"?: string; "items": (CatalogListEntry)[]; "matched": number; "options": CatalogListOptions; "total": number }
export type CatalogPlanRecord = { "data": SourceCatalogImportPlan; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type CatalogReleasePage = { "items": (CatalogReleaseRecord)[] }
export type CatalogReleaseRecord = { "data": SourceCatalogRelease; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type CatalogRevisionPage = { "items": (CatalogRevisionRecord)[] }
export type CatalogRevisionRecord = { "data": CatalogTestRevision; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type CatalogTestRecord = { "data": SourceCatalogTest; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type CatalogTestRevision = { "actor": string; "at": string; "test": SourceCatalogTest }
export type CatalogValidation = { "issues": (SourceCatalogIssue)[]; "valid": boolean }
export type EditorFile = { "content": string; "executable"?: boolean; "path": string }
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
export type MemberCandidate = { "disabled": boolean; "id": string; "name": string }
export type Membership = { "active": boolean; "expiresAt"?: string | null; "groupId": string; "groupName"?: string; "name": string; "projectId"?: string; "providerRevision"?: number; "readOnly": boolean; "source": string; "userId": string }
export type MembershipDefinition = { "expiresAt"?: string | null }
export type MembershipPage = { "cursor"?: string; "items": (MembershipRecord)[]; "partial": boolean }
export type MembershipRecord = { "data": Membership; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type OwnershipStatus = { "applicationId"?: string; "projectId": string; "state": "system-managed" | "owned" | "owner-unavailable" }
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
export type PermissionExplanation = { "allowed": boolean; "applicationId"?: string; "canInspectOthers": boolean; "checkedAt": string; "cursor"?: string; "items": (PermissionSource)[]; "object": string; "partial": boolean; "permission": string; "permissions": (PermissionDefinition)[]; "projectId": string; "reason": "allowed" | "explicit-deny" | "no-matching-allow" | "user-disabled"; "userDisabled": boolean; "userId": string; "userName": string }
export type PermissionSource = { "assignedRoleId"?: string; "assignedRoleName"?: string; "assignmentApplication"?: string; "assignmentProject"?: string; "effect": "allow" | "deny"; "expiresAt"?: string; "grantId"?: string; "id": string; "kind": "direct" | "role"; "objectScope"?: string; "principalId"?: string; "principalKind"?: "user" | "group"; "principalName"?: string; "projectId"?: string; "restricted": boolean; "restrictedFields"?: (string)[]; "roleId"?: string; "roleName"?: string; "scopeKnown": boolean; "status": "matching" | "expired"; "teamScope"?: string }
export type PlaywrightDraft = { "files": (EditorFile)[]; "id": string; "manifestPath"?: string; "name": string; "targetUrl"?: string }
export type PlaywrightDraftPage = { "cursor"?: string; "items": (PlaywrightDraftRecord)[] }
export type PlaywrightDraftRecord = { "data": PlaywrightDraft; "id": string; "revision": number; "updatedAt": string }
export type PredefinedRole = { "canManage": boolean; "disabled": boolean; "id": string; "name": string; "permissions": (string)[]; "scope": "project" | "installation" }
export type PredefinedRolePage = { "items": (PredefinedRoleRecord)[] }
export type PredefinedRoleRecord = { "data": PredefinedRole; "id": string; "revision": number; "updatedAt"?: string }
export type PredefinedRoleUpdate = { "disabled": boolean; "reason": string }
export type ProbeBinding = { "archived"?: boolean; "endpoint": string; "environmentId": string; "executionLocation"?: string; "id"?: string; "name": string; "suiteId": string; "testId": string }
export type ProbeBindingRecord = { "data": ProbeBinding; "id": string; "revision": number; "updatedAt": string }
export type ProbeHTTPObservation = { "certificateExpires"?: string; "durationMs": number; "status": number; "timingsMs"?: { [key: string]: number }; "url": string }
export type ProbeObservation = { "createdAt": string; "definitionChanged": boolean; "error"?: string; "evidenceState": string; "http"?: ProbeHTTPObservation; "observedAt"?: string; "releaseId": string; "runId": string; "state": string; "targetChanged": boolean }
export type ProbeOverviewPage = { "cursor"?: string; "items": (ProbeSummary)[]; "matched": number; "partial": boolean; "scanned": number; "serverTime": string; "windowStart": string }
export type ProbeSchedule = { "cron"?: string; "enabled": boolean; "id": string; "intervalSeconds"?: number; "kind": string; "lastState"?: string; "name": string; "nextAt"?: string; "shared": boolean }
export type ProbeSummary = { "active"?: ProbeObservation; "activeReleaseId"?: string; "applicationId"?: string; "archived": boolean; "availability": string; "draftChanged"?: boolean; "endpoint": string; "environmentId": string; "executionLocation": string; "freshness": string; "history": (ProbeObservation)[]; "id": string; "latest"?: ProbeObservation; "name": string; "other": number; "partial": boolean; "passed": number; "revision": number; "samples": number; "scheduleState": string; "schedules": (ProbeSchedule)[]; "stage"?: string; "state": string; "successRate"?: number; "suiteId": string; "testId": string; "url"?: string }
export type Problem = { "message": string; "status"?: number }
export type Profile = { "identities": ExternalIdentityPage; "localAccount": boolean; "user": User }
export type Provider = { "autoProvision": boolean; "callbackUrl": string; "canManage": boolean; "clientId": string; "disabled": boolean; "groupClaim"?: string; "groupMappings": ProviderMappings; "id": string; "issuer": string; "logoutCallbackUrl": string; "logoutEndpoint"?: string; "membershipTtlSeconds": number; "name": string; "scopes": (string)[]; "secretRef"?: string }
export type ProviderLogoutOffer = { "expiresAt": string; "providerName": string; "url": string }
export type ProviderMappings = { [key: string]: string }
export type ProviderMutation = { "autoProvision": boolean; "clientId": string; "disabled": boolean; "groupClaim"?: string; "groupMappings": ProviderMappings; "id": string; "issuer": string; "logoutEndpoint"?: string; "membershipTtlSeconds": number; "name": string; "reason": string; "scopes": (string)[]; "secretRef"?: string }
export type ProviderPage = { "canCreate": boolean; "credentialRefs": (string)[]; "cursor"?: string; "items": (ProviderRecord)[]; "partial": boolean }
export type ProviderRecord = { "data": Provider; "id": string; "revision": number; "updatedAt": string }
export type PublicationChange = { "action": "added" | "changed" | "removed"; "after"?: SourceCatalogTest; "before"?: SourceCatalogTest; "fields": (string)[]; "id": string; "safetyChanged": boolean; "title": string }
export type PublicationReview = { "activeReleaseId"?: string; "afterSettings": PublicationSettings; "artifactChanged": boolean; "beforeSettings"?: PublicationSettings; "captureChanged": boolean; "changes": (PublicationChange)[]; "createdAt": string; "draftId": string; "draftRevision": number; "enabled": number; "expiresAt": string; "id": string; "issues": (SourceCatalogIssue)[]; "projectId": string; "schedules": (PublicationSchedule)[]; "suiteId": string; "timingChanged": boolean; "total": number; "unchanged": number }
export type PublicationSchedule = { "enabled": boolean; "followsActive": boolean; "id": string; "name": string; "releaseId"?: string }
export type PublicationSettings = { "artifact": SourceCatalogArtifact; "capture": SourceCapturePolicy; "timing": SourceTimingPolicy }
export type Reason = { "reason": string }
export type RecordingCompanionControl = { "action"?: "stop" | "cancel"; "deadlineAt": string; "sequence": number; "state": string }
export type RecordingCompanionControlRequest = { "instance": string }
export type RecordingCompanionPacket = { "control": RecordingCompanionControl; "input": BrowserRecordingInput; "projectId": string; "suiteId": string }
export type RecordingCompanionPairRequest = { "deviceToken": string; "instance": string; "pairingToken": string }
export type RecordingCompanionSnapshot = { "code"?: string; "failure"?: "recorder" | "capture" | "transport"; "sequence": number; "state": "recording" | "ready" | "failed" }
export type RecordingCompanionSnapshotRequest = { "instance": string; "snapshot": RecordingCompanionSnapshot }
export type ReleaseRegistry = { "activeReleaseId"?: string; "archived": boolean; "canActivate": boolean; "items": (SourceCatalogRelease)[]; "suiteId": string; "suiteName": string; "suiteRevision": number }
export type RetainedTest = { "current": boolean; "lifecycle": string; "projectId": string; "revision": number; "suiteArchived": boolean; "suiteId": string; "suiteName": string; "test": SourceCatalogTest; "updatedAt": string }
export type ReviewedPublicationInput = { "activate": boolean; "reviewId": string }
export type Revoked = { "revoked": boolean }
export type Role = { "builtIn": boolean; "delegatable": boolean; "disabled": boolean; "id": string; "name": string; "permissions": (string)[]; "scope": "project" | "installation" | "application" }
export type RoleChoicePage = { "cursor"?: string; "items": (Role)[]; "partial": boolean }
export type RoleConversion = { "actor": string; "addedPermissions": (string)[]; "applicationId"?: string; "applied": boolean; "before": (AssignmentRecord)[]; "directoryRevision": number; "expiresAt": string; "id": string; "principalId": string; "principalKind": "user" | "group"; "projectId": string; "reason": string; "removedPermissions": (string)[]; "result"?: (AssignmentRecord)[]; "reverses"?: string; "targetRoleIds": (string)[] }
export type RoleConversionRecord = { "data": RoleConversion; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
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
export type SavedSelection = { "archived": boolean; "description": string; "id": string; "name": string; "spec": SavedSelectionSpec }
export type SavedSelectionFilter = { "applicationId"?: string; "category"?: string; "includeCategoryDescendants"?: boolean; "owner"?: string; "query"?: string; "runner"?: string; "safety"?: string; "source"?: string; "tags"?: (string)[]; "type"?: string }
export type SavedSelectionPage = { "cursor"?: string; "items": (SavedSelectionRecord)[] }
export type SavedSelectionPreview = { "actor": string; "createdAt": string; "expiresAt": string; "id": string; "projectId": string; "releaseId": string; "requestedTestIds": (string)[]; "resolvedTestIds": (string)[]; "selection": SavedSelectionSnapshot; "suiteId": string }
export type SavedSelectionPreviewInput = { "releaseId"?: string }
export type SavedSelectionRecord = { "data": SavedSelection; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SavedSelectionRunReference = { "savedSelectionPreviewId"?: string }
export type SavedSelectionSnapshot = { "definition": SavedSelection; "previewId": string; "revision": number }
export type SavedSelectionSpec = { "filter"?: SavedSelectionFilter; "kind": "explicit" | "filter"; "schemaVersion": number; "testIds"?: (string)[] }
export type Session = { "authenticationUntil": string; "client"?: string; "clientName"?: string; "createdAt": string; "current": boolean; "expiresAt": string; "id": string; "lastSeen": string; "provider": string }
export type SessionPage = { "cursor"?: string; "items": (SessionRecord)[]; "partial": boolean }
export type SessionRecord = { "data": Session; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SourceBuild = { "finishedAt": string; "image": string; "log"?: string; "logTruncated"?: boolean; "sourceDigest": string; "startedAt": string }
export type SourceCapturePolicy = { "failureRetentionDays"?: number; "kindRetentionDays"?: { [key: string]: number }; "logs": boolean; "maxBytes"?: number; "maxFileBytes"?: number; "retentionDays"?: number; "screenshots": string; "trace": string; "video": string }
export type SourceCatalogArtifact = { "browser"?: { "endpoint": string; "requests"?: APIRequestPolicy }; "discoveredSelectors"?: { [key: string]: string }; "image"?: string; "provenance"?: SourceOrigin; "sourceProjectId"?: string; "sourceRevision"?: string; "sourceSnapshotId"?: string }
export type SourceCatalogDraft = { "artifact": SourceCatalogArtifact; "capture": SourceCapturePolicy; "id": string; "publishedReleaseId"?: string; "tests": (SourceCatalogTest)[]; "timing": SourceTimingPolicy }
export type SourceCatalogHTTPCheck = { "endpoint": string; "forbiddenText"?: string; "headers"?: { [key: string]: string }; "json"?: { [key: string]: SourceJSONValue }; "maxResponseMs"?: number; "method": string; "minCertificateDays"?: number; "path": string; "requiredText"?: string; "statusCodes": (number)[] }
export type SourceCatalogImportPlan = { "actor": string; "appliedAt"?: string; "artifact"?: SourceCatalogArtifact; "catalogRevision": number; "changes": (SourceCatalogPlannedTest)[]; "createdAt": string; "expiresAt": string; "id": string; "issues": (SourceCatalogIssue)[]; "kind": string; "projectId": string; "sourceSnapshotId"?: string; "suiteId": string; "suiteRevision": number }
export type SourceCatalogIssue = { "field": string; "message": string; "testId"?: string }
export type SourceCatalogPlannedTest = { "action": string; "before"?: SourceCatalogTest; "expectedRevision": number; "test": SourceCatalogTest }
export type SourceCatalogRelease = { "activatedOnPublication": boolean; "actor": string; "catalog": SourceCatalogDraft; "createdAt": string; "draftRevision": number; "id": string; "projectId": string; "suiteId": string }
export type SourceCatalogTest = { "applicationId"?: string; "capture"?: SourceCapturePolicy; "category": string; "check"?: SourceCatalogHTTPCheck; "conditions"?: string; "definitionSourceId"?: string; "dependencies": (string)[] | null; "description": string; "id": string; "owner"?: string; "rationale": string; "requiredBindings": (string)[] | null; "runner": string; "safety": string; "selector"?: string; "source"?: string; "state": string; "tags": (string)[] | null; "timeoutSeconds": number; "timing"?: SourceTimingPolicy; "title": string; "type": string }
export type SourceContent = { "content": string; "digest": string }
export type SourceCredentialReference = { "id": string; "name": string }
export type SourceJSONValue = null | string | number | boolean | (SourceJSONValue)[] | { [key: string]: SourceJSONValue }
export type SourceOrigin = { "commit"?: string; "reference"?: string; "repository"?: string; "snapshotDigest"?: string }
export type SourcePrebuiltInput = { "image": string; "runner": string }
export type SourcePreparationJob = { "actor": string; "assemblyComplete"?: boolean; "assemblyReference"?: SourcePreparationReference; "build"?: SourceBuild; "cancelRequested": boolean; "cleanupPending": boolean; "createdAt": string; "error"?: string; "executionLocation": string; "finishedAt"?: string; "id": string; "image": string; "kind"?: string; "leaseUntil"?: string; "localImageId"?: string; "log"?: string; "logTruncated"?: boolean; "manifestPath"?: string; "phase": string; "projectId": string; "reference"?: SourcePreparationReference; "releaseId"?: string; "repositorySourceId"?: string; "repositorySyncId"?: string; "runner": string; "runtime"?: string; "sourceBundleId"?: string; "sourceDigest"?: string; "sourceDraftId"?: string; "sourceDraftRevision"?: number; "sourceExpired"?: boolean; "sourceGeneration"?: number; "sourceName"?: string; "sourceOrigin"?: SourceOrigin; "sourceSequence"?: number; "startedAt"?: string; "state": string; "suiteId": string; "uploadId"?: string; "worker"?: string }
export type SourcePreparationOptions = { "bundleRunners": (string)[]; "executionLocation": string; "runners": (string)[] }
export type SourcePreparationPage = { "cursor"?: string; "items": (SourcePreparationRecord)[] }
export type SourcePreparationRecord = { "data": SourcePreparationJob; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SourcePreparationReference = { "attempt": number; "container": string; "id": string; "installationId": string; "profile": string; "projectId": string; "runId": string; "testId": string; "volume": string; "workerId": string }
export type SourcePublishInput = { "activate": boolean }
export type SourceRepositoryBinding = { "buildProfile": string; "manifestPath": string; "runner": string; "suiteId": string }
export type SourceRepositoryConfig = { "autoActivate": boolean; "bindings": (SourceRepositoryBinding)[]; "credentialReference"?: string; "enabled": boolean; "name": string; "provider": string; "pushEnabled"?: boolean; "reference": string; "repository": string; "syncIntervalSeconds"?: number; "webhookCredentialReference"?: string }
export type SourceRepositoryOptions = { "credentials": (SourceCredentialReference)[]; "providers": (string)[]; "webhookCredentials": (SourceCredentialReference)[] }
export type SourceRepositoryPage = { "cursor"?: string; "items": (SourceRepositoryRecord)[] }
export type SourceRepositoryRecord = { "data": SourceRepositorySource; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SourceRepositorySource = { "actor": string; "configuration": SourceRepositoryConfig; "generation": number; "id": string; "lastCommit"?: string; "lastError"?: string; "lastState"?: string; "lastSyncAt"?: string; "lastSyncId"?: string; "nextSyncAt"?: string; "projectId": string; "repositoryId"?: string; "sequence": number }
export type SourceRepositorySync = { "actor": string; "configuration": SourceRepositoryConfig; "createdAt": string; "deliveryId"?: string; "error"?: string; "eventDigest"?: string; "finishedAt"?: string; "force": boolean; "id": string; "leaseUntil"?: string; "preparations": { [key: string]: string }; "projectId": string; "repositoryId"?: string; "requestedBy": string; "revision"?: SourceRevision; "sequence": number; "snapshotId"?: string; "sourceGeneration": number; "sourceId": string; "state": string; "trigger"?: string; "worker"?: string }
export type SourceRepositorySyncPage = { "cursor"?: string; "items": (SourceRepositorySyncRecord)[] }
export type SourceRepositorySyncRecord = { "data": SourceRepositorySync; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type SourceRevision = { "commit": string; "private": boolean; "reference": string; "repository": string; "repositoryId": string; "url": string }
export type SourceSyncInput = { "force": boolean }
export type SourceTimingPolicy = { "maxAgeDays": number; "minimumIncreaseMs": number; "minimumSamples": number; "minimumTimeoutMs": number; "mode": string; "slowdownFactor": number; "timeoutFactor": number; "timeoutMarginMs": number; "timeoutMode": string; "window": number }
export type SourceUpload = { "activated": boolean; "actor": string; "artifact"?: SourceCatalogArtifact; "bytes": number; "createdAt": string; "digest": string; "expiresAt": string; "format": string; "id": string; "issues": (SourceCatalogIssue)[]; "kind"?: string; "name": string; "planId"?: string; "preparationId"?: string; "projectId": string; "releaseId"?: string; "repositorySourceId"?: string; "state": string; "suiteId": string }
export type SourceUploadInput = { "content": string; "format": "json" | "yaml"; "name": string }
export type SourceUploadPage = { "cursor"?: string; "items": (SourceUploadRecord)[] }
export type SourceUploadPreview = { "plan"?: SourceCatalogImportPlan; "upload": SourceUploadRecord }
export type SourceUploadRecord = { "data": SourceUpload; "id": string; "projectId"?: string; "revision": number; "updatedAt": string }
export type StandardAssignmentReplacement = { "previousRoleId"?: string; "reason": string; "roleId": "builtin.teststudio.reader" | "builtin.teststudio.member" | "builtin.teststudio.developer" | "builtin.teststudio.contributor" | "builtin.teststudio.owner" }
export type StatusPage = { "background": string; "defaultWindow": "24h" | "7d" | "1m" | "1y" | "all"; "id": string; "maxAgeSeconds": number; "probes": (StatusPageEntry)[]; "publicToken": string; "published"?: StatusPageDefinition; "publishedAt"?: string; "publisher"?: string; "slug"?: string; "title": string }
export type StatusPageAvailability = { "coveragePercent": number; "downSeconds": number; "percent": number | null; "since": string; "unknownSeconds": number; "upSeconds": number }
export type StatusPageDefinition = { "background": string; "defaultWindow": "24h" | "7d" | "1m" | "1y" | "all"; "id": string; "maxAgeSeconds": number; "probes": (StatusPageEntry)[]; "slug"?: string; "title": string }
export type StatusPageEntry = { "description": string; "probeId": string; "title": string }
export type StatusPageList = { "cursor"?: string; "items": (StatusPageRecord)[] }
export type StatusPagePoint = { "at": string; "state": string }
export type StatusPageRecord = { "data": StatusPage; "id": string; "projectId"?: string; "revision": number; "updatedAt"?: string }
export type StatusPageReport = { "asOf": string; "background": string; "maxAgeSeconds": number; "probes": (StatusPageRow)[]; "title": string; "window": string }
export type StatusPageRow = { "availability": StatusPageAvailability; "description": string; "history": (StatusPagePoint)[]; "state": string; "title": string }
export type StatusPageSlugAvailability = { "available": boolean; "slug": string }
export type SuiteDefinition = { "activeReleaseId"?: string; "applications": (string)[] | null; "archived": boolean; "capture"?: SourceCapturePolicy; "description": string; "id": string; "name": string; "requiresTarget": boolean; "timing"?: SourceTimingPolicy }
export type SuiteLifecycle = { "canManage": boolean; "revision": number; "schedules": (SuiteSchedule)[]; "suite": SuiteDefinition; "updatedAt": string }
export type SuiteSchedule = { "enabled": boolean; "id": string; "name": string; "releaseId"?: string }
export type SuiteSettings = { "applications": (string)[] | null; "description": string; "name": string; "requiresTarget": boolean }
export type TestHistoryItem = { "attempt": number; "createdAt": string; "durationMs"?: number; "environmentName"?: string; "error"?: string; "evidenceState": string; "executionLocation": string; "finishedAt"?: string; "releaseId": string; "runId": string; "runOutcome": string; "runState": string; "testState": string; "title": string }
export type TestHistoryPage = { "cursor"?: string; "items": (TestHistoryItem)[]; "partial": boolean; "scanned": number }
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
  readCurrentCapabilities: { query?: { "project"?: string; "application"?: string } }
  explainOwnPermission: { query: { "project"?: string; "object"?: string; "permission": string; "limit"?: number; "cursor"?: string; "application"?: string } }
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
  listAPICredentialReferences: { "project": string }
  readApplication: { "project": string; "application": string }
  readApplicationMemberCandidate: { "project": string; "application": string; "user": string }
  readApplicationOwnership: { "project": string; "application": string }
  listApplicationAssignments: { "project": string; "application": string; query?: { "limit"?: number; "cursor"?: string } }
  createApplicationAssignment: { "project": string; "application": string; body: AssignmentDefinition }
  replaceApplicationStandardAssignment: { "project": string; "application": string; "kind": string; "principal": string; revision: number; body: StandardAssignmentReplacement }
  revokeApplicationAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; revision: number; "application": string; body: Reason }
  readApplicationAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; "application": string }
  previewApplicationRoleConversion: { "project": string; "application": string; body: AssignmentDefinition }
  readApplicationRoleConversion: { "project": string; "application": string; "conversion": string }
  applyApplicationRoleConversion: { "project": string; "application": string; "conversion": string; revision: number }
  previewApplicationRoleConversionRollback: { "project": string; "application": string; "conversion": string; body: Reason }
  readProjectMemberCandidate: { "project": string; "user": string }
  readProjectOwnership: { "project": string }
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
  listHTTPSProbes: { "project": string; query?: { "search"?: string; "application"?: string; "stage"?: string; "state"?: string; "freshness"?: string; "schedule"?: string; "cursor"?: string; "hours"?: number; "limit"?: number; "archived"?: boolean } }
  createHTTPSProbeBinding: { "project": string; body: ProbeBinding }
  readHTTPSProbeBinding: { "project": string; "probe": string }
  updateHTTPSProbeBinding: { "project": string; "probe": string; revision: number; body: ProbeBinding }
  readRepositorySourceOptions: { "project": string }
  listRepositorySources: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  readRepositorySource: { "project": string; "source": string }
  saveRepositorySource: { "project": string; "source": string; revision: number; body: SourceRepositoryConfig }
  listRepositorySyncs: { "project": string; "source": string; query?: { "limit"?: number; "cursor"?: string } }
  enqueueRepositorySync: { "project": string; "source": string; key: string; body: SourceSyncInput }
  readRepositorySync: { "project": string; "source": string; "sync": string }
  listProjectAssignments: { "project": string; query?: { "limit"?: number; "cursor"?: string } }
  createProjectAssignment: { "project": string; body: AssignmentDefinition }
  replaceProjectStandardAssignment: { "project": string; "kind": string; "principal": string; revision: number; body: StandardAssignmentReplacement }
  revokeProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readProjectAssignment: { "project": string; "kind": "user" | "group"; "principal": string; "role": string }
  previewProjectRoleConversion: { "project": string; body: AssignmentDefinition }
  readProjectRoleConversion: { "project": string; "conversion": string }
  applyProjectRoleConversion: { "project": string; "conversion": string; revision: number }
  previewProjectRoleConversionRollback: { "project": string; "conversion": string; body: Reason }
  checkStatusPageSlug: { "project": string; "slug": string; query?: { "pageId"?: string } }
  listStatusPages: { "project": string; query?: { "cursor"?: string } }
  deleteStatusPage: { "project": string; "page": string; revision: number }
  readStatusPage: { "project": string; "page": string }
  saveStatusPage: { "project": string; "page": string; revision: number; body: StatusPageDefinition }
  previewStatusPage: { "project": string; "page": string; query?: { "window"?: string } }
  publishStatusPage: { "project": string; "page": string; revision: number }
  unpublishStatusPage: { "project": string; "page": string; revision: number }
  changeSuiteLifecycle: { "project": string; "suite": string; "action": "archive" | "restore"; revision: number }
  listBrowserRecordings: { "project": string; "suite": string; query?: { "cursor"?: string; "limit"?: number } }
  startBrowserRecording: { "project": string; "suite": string; body: BrowserRecordingInput }
  readBrowserRecordingOptions: { "project": string; "suite": string }
  readBrowserRecording: { "project": string; "suite": string; "id": string }
  changeBrowserRecordingState: { "project": string; "suite": string; "id": string; "action": string }
  createBrowserRecordingPairing: { "project": string; "suite": string; "id": string }
  readBrowserRecordingSource: { "project": string; "suite": string; "id": string }
  listCatalogDrafts: { "project": string; "suite": string }
  createCatalogDraft: { "project": string; "suite": string; body: SourceCatalogArtifact }
  readCatalogDraft: { "project": string; "suite": string; "id": string }
  updateCatalogDraft: { "project": string; "suite": string; "id": string; revision: number; body: SourceCatalogDraft }
  reviewCatalogPublication: { "project": string; "suite": string; "id": string; revision: number }
  publishReviewedCatalog: { "project": string; "suite": string; "id": string; revision: number; key: string; body: ReviewedPublicationInput }
  validateCatalogDraft: { "project": string; "suite": string; "id": string }
  exportCatalogDocument: { "project": string; "suite": string }
  previewCatalogImport: { "project": string; "suite": string; body: CatalogDocument }
  readCatalogPlan: { "project": string; "suite": string; "id": string }
  applyCatalogPlan: { "project": string; "suite": string; "id": string; revision: number }
  readSuiteLifecycle: { "project": string; "suite": string }
  listPlaywrightDrafts: { "project": string; "suite": string; query?: { "cursor"?: string; "limit"?: number } }
  readPlaywrightStarter: { "project": string; "suite": string; query?: { "type"?: "browser" | "api" } }
  readPlaywrightDraft: { "project": string; "suite": string; "id": string }
  savePlaywrightDraft: { "project": string; "suite": string; "id": string; revision: number; body: PlaywrightDraft }
  preparePlaywrightDraft: { "project": string; "suite": string; "id": string; revision: number; key: string }
  readCatalogPublicationReview: { "project": string; "suite": string; "id": string }
  readReleaseRegistry: { "project": string; "suite": string }
  listCatalogReleaseRecords: { "project": string; "suite": string }
  readCatalogReleaseRecord: { "project": string; "suite": string; "id": string }
  activateRetainedRelease: { "project": string; "suite": string; "id": string; revision: number }
  listSavedSelections: { "project": string; "suite": string; query?: { "cursor"?: string; "limit"?: number; "archived"?: boolean } }
  createSavedSelection: { "project": string; "suite": string; body: SavedSelection }
  readSavedSelection: { "project": string; "suite": string; "id": string }
  updateSavedSelection: { "project": string; "suite": string; "id": string; revision: number; body: SavedSelection }
  changeSavedSelectionState: { "project": string; "suite": string; "id": string; "action": "archive" | "restore"; revision: number }
  previewSavedSelection: { "project": string; "suite": string; "id": string; body: SavedSelectionPreviewInput }
  updateSuiteSettings: { "project": string; "suite": string; revision: number; body: SuiteSettings }
  uploadSourceBundle: { "project": string; "suite": string; key: string; query: { "name": string; "manifest": string }; body: Blob }
  readPreparationOptions: { "project": string; "suite": string }
  listSourcePreparations: { "project": string; "suite": string; query?: { "limit"?: number; "cursor"?: string } }
  enqueuePrebuiltPreparation: { "project": string; "suite": string; key: string; body: SourcePrebuiltInput }
  readSourcePreparation: { "project": string; "suite": string; "id": string }
  downloadSourceBundle: { "project": string; "suite": string; "id": string }
  cancelSourcePreparation: { "project": string; "suite": string; "id": string; revision: number }
  readPreparationEditorSource: { "project": string; "suite": string; "id": string }
  listSourceUploads: { "project": string; "suite": string; query?: { "limit"?: number; "cursor"?: string } }
  stageSourceUpload: { "project": string; "suite": string; body: SourceUploadInput }
  readSourceUpload: { "project": string; "suite": string; "id": string }
  readSourceContent: { "project": string; "suite": string; "id": string }
  publishSourceUpload: { "project": string; "suite": string; "id": string; revision: number; body: SourcePublishInput }
  listSuiteCatalog: { "project": string; "suite": string; query?: { "query"?: string; "suite"?: string; "application"?: string; "category"?: string; "tag"?: string; "owner"?: string; "type"?: string; "runner"?: string; "state"?: string; "safety"?: string; "source"?: string; "sort"?: string; "cursor"?: string; "descendants"?: boolean; "uncategorized"?: boolean; "limit"?: number } }
  createCatalogTest: { "project": string; "suite": string; body: SourceCatalogTest }
  previewCatalogBulk: { "project": string; "suite": string; body: CatalogBulkRequest }
  readCatalogTest: { "project": string; "suite": string; "test": string }
  updateCatalogTest: { "project": string; "suite": string; "test": string; revision: number; body: SourceCatalogTest }
  duplicateCatalogTest: { "project": string; "suite": string; "test": string; revision: number; body: CatalogDuplicateInput }
  changeCatalogTestState: { "project": string; "suite": string; "test": string; "action": "disable" | "enable" | "archive" | "restore"; revision: number }
  readRetainedTest: { "project": string; "suite": string; "test": string }
  listCatalogTestRevisions: { "project": string; "suite": string; "test": string }
  readTestRunHistory: { "project": string; "suite": string; "test": string; query?: { "environment"?: string; "location"?: string; "application"?: string; "cursor"?: string; "limit"?: number } }
  listProjectCatalog: { "project": string; query?: { "query"?: string; "suite"?: string; "application"?: string; "category"?: string; "tag"?: string; "owner"?: string; "type"?: string; "runner"?: string; "state"?: string; "safety"?: string; "source"?: string; "sort"?: string; "cursor"?: string; "descendants"?: boolean; "uncategorized"?: boolean; "limit"?: number } }
  readPublicStatusPage: { "token": string; query?: { "window"?: string } }
  readRecordingCompanionControl: { "id": string; body: RecordingCompanionControlRequest }
  pairRecordingCompanion: { "id": string; body: RecordingCompanionPairRequest }
  submitRecordingCompanionSnapshot: { "id": string; body: RecordingCompanionSnapshotRequest }
  listInstallationAssignments: { query?: { "limit"?: number; "cursor"?: string } }
  createInstallationAssignment: { body: AssignmentDefinition }
  revokeInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string; revision: number; body: Reason }
  readInstallationAssignment: { "kind": "user" | "group"; "principal": string; "role": string }
  listRoles: { query?: { "project"?: string; "application"?: string } }
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
  explainUserPermission: { "user": string; query: { "project"?: string; "object"?: string; "permission": string; "limit"?: number; "cursor"?: string; "application"?: string } }
  listUserIdentities: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  readLocalCredential: { "user": string }
  setLocalPassword: { "user": string; revision: number; body: LocalPasswordDefinition }
  disableLocalCredential: { "user": string; revision: number; body: Reason }
  enableLocalCredential: { "user": string; revision: number; body: Reason }
  listUserMemberships: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  restoreUser: { "user": string; revision: number; body: Reason }
  listUserSessions: { "user": string; query?: { "limit"?: number; "cursor"?: string } }
  revokeUserSession: { "user": string; "session": string; revision: number; body: Reason }
  cancelCLISignIn: { body: CLIProof }
  exchangeCLISignIn: { body: CLIProof }
  readCLISignInOptions: {  }
  readCLISignInRequest: { "request": string; query: { "clientName": string } }
  confirmCLISignInRequest: { "request": string; body: CLIConfirmation }
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
  readCurrentCapabilities: Capabilities
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
  listAPICredentialReferences: { "items": ({ "allowedOrigins"?: (string)[]; "id": string; "name": string })[] }
  readApplication: ApplicationRecord
  readApplicationMemberCandidate: MemberCandidate
  readApplicationOwnership: OwnershipStatus
  listApplicationAssignments: AssignmentPage
  createApplicationAssignment: AssignmentRecord
  replaceApplicationStandardAssignment: AssignmentRecord
  revokeApplicationAssignment: Revoked
  readApplicationAssignment: AssignmentRecord
  previewApplicationRoleConversion: RoleConversionRecord
  readApplicationRoleConversion: RoleConversionRecord
  applyApplicationRoleConversion: RoleConversionRecord
  previewApplicationRoleConversionRollback: RoleConversionRecord
  readProjectMemberCandidate: MemberCandidate
  readProjectOwnership: OwnershipStatus
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
  listHTTPSProbes: ProbeOverviewPage
  createHTTPSProbeBinding: ProbeBindingRecord
  readHTTPSProbeBinding: ProbeBindingRecord
  updateHTTPSProbeBinding: ProbeBindingRecord
  readRepositorySourceOptions: SourceRepositoryOptions
  listRepositorySources: SourceRepositoryPage
  readRepositorySource: SourceRepositoryRecord
  saveRepositorySource: SourceRepositoryRecord
  listRepositorySyncs: SourceRepositorySyncPage
  enqueueRepositorySync: SourceRepositorySyncRecord
  readRepositorySync: SourceRepositorySyncRecord
  listProjectAssignments: AssignmentPage
  createProjectAssignment: AssignmentRecord
  replaceProjectStandardAssignment: AssignmentRecord
  revokeProjectAssignment: Revoked
  readProjectAssignment: AssignmentRecord
  previewProjectRoleConversion: RoleConversionRecord
  readProjectRoleConversion: RoleConversionRecord
  applyProjectRoleConversion: RoleConversionRecord
  previewProjectRoleConversionRollback: RoleConversionRecord
  checkStatusPageSlug: StatusPageSlugAvailability
  listStatusPages: StatusPageList
  deleteStatusPage: { "deleted": boolean }
  readStatusPage: StatusPageRecord
  saveStatusPage: StatusPageRecord
  previewStatusPage: StatusPageReport
  publishStatusPage: StatusPageRecord
  unpublishStatusPage: StatusPageRecord
  changeSuiteLifecycle: SuiteLifecycle
  listBrowserRecordings: BrowserRecordingPage
  startBrowserRecording: BrowserRecordingRecord
  readBrowserRecordingOptions: BrowserRecordingOptions
  readBrowserRecording: BrowserRecordingRecord
  changeBrowserRecordingState: BrowserRecordingRecord
  createBrowserRecordingPairing: BrowserRecordingPairing
  readBrowserRecordingSource: PlaywrightDraft
  listCatalogDrafts: CatalogDraftPage
  createCatalogDraft: CatalogDraftRecord
  readCatalogDraft: CatalogDraftRecord
  updateCatalogDraft: CatalogDraftRecord
  reviewCatalogPublication: PublicationReview
  publishReviewedCatalog: SourceCatalogRelease
  validateCatalogDraft: CatalogValidation
  exportCatalogDocument: CatalogDocument
  previewCatalogImport: CatalogPlanRecord
  readCatalogPlan: CatalogPlanRecord
  applyCatalogPlan: CatalogPlanRecord
  readSuiteLifecycle: SuiteLifecycle
  listPlaywrightDrafts: PlaywrightDraftPage
  readPlaywrightStarter: PlaywrightDraft
  readPlaywrightDraft: PlaywrightDraftRecord
  savePlaywrightDraft: PlaywrightDraftRecord
  preparePlaywrightDraft: SourcePreparationRecord
  readCatalogPublicationReview: PublicationReview
  readReleaseRegistry: ReleaseRegistry
  listCatalogReleaseRecords: CatalogReleasePage
  readCatalogReleaseRecord: CatalogReleaseRecord
  activateRetainedRelease: {  }
  listSavedSelections: SavedSelectionPage
  createSavedSelection: SavedSelectionRecord
  readSavedSelection: SavedSelectionRecord
  updateSavedSelection: SavedSelectionRecord
  changeSavedSelectionState: SavedSelectionRecord
  previewSavedSelection: SavedSelectionPreview
  updateSuiteSettings: SuiteLifecycle
  uploadSourceBundle: SourcePreparationRecord
  readPreparationOptions: SourcePreparationOptions
  listSourcePreparations: SourcePreparationPage
  enqueuePrebuiltPreparation: SourcePreparationRecord
  readSourcePreparation: SourcePreparationRecord
  downloadSourceBundle: Blob
  cancelSourcePreparation: SourcePreparationRecord
  readPreparationEditorSource: PlaywrightDraft
  listSourceUploads: SourceUploadPage
  stageSourceUpload: SourceUploadPreview
  readSourceUpload: SourceUploadPreview
  readSourceContent: SourceContent
  publishSourceUpload: SourceCatalogRelease
  listSuiteCatalog: CatalogListPage
  createCatalogTest: CatalogTestRecord
  previewCatalogBulk: CatalogPlanRecord
  readCatalogTest: CatalogTestRecord
  updateCatalogTest: CatalogTestRecord
  duplicateCatalogTest: CatalogTestRecord
  changeCatalogTestState: CatalogTestRecord
  readRetainedTest: RetainedTest
  listCatalogTestRevisions: CatalogRevisionPage
  readTestRunHistory: TestHistoryPage
  listProjectCatalog: CatalogListPage
  readPublicStatusPage: StatusPageReport
  readRecordingCompanionControl: RecordingCompanionControl
  pairRecordingCompanion: RecordingCompanionPacket
  submitRecordingCompanionSnapshot: RecordingCompanionControl
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
  cancelCLISignIn: { "state": string }
  exchangeCLISignIn: CLIExchange
  readCLISignInOptions: CLIOptions
  readCLISignInRequest: CLIView
  confirmCLISignInRequest: CLIView
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
  readCurrentCapabilities: { method: "GET", path: "/api/v1/me/capabilities" },
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
  listAPICredentialReferences: { method: "GET", path: "/api/v1/projects/{project}/api-credential-references" },
  readApplication: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}" },
  readApplicationMemberCandidate: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}/member-candidates/{user}" },
  readApplicationOwnership: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}/ownership" },
  listApplicationAssignments: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}/role-assignments" },
  createApplicationAssignment: { method: "POST", path: "/api/v1/projects/{project}/applications/{application}/role-assignments" },
  replaceApplicationStandardAssignment: { method: "PUT", path: "/api/v1/projects/{project}/applications/{application}/role-assignments/{kind}/{principal}" },
  revokeApplicationAssignment: { method: "DELETE", path: "/api/v1/projects/{project}/applications/{application}/role-assignments/{kind}/{principal}/{role}" },
  readApplicationAssignment: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}/role-assignments/{kind}/{principal}/{role}" },
  previewApplicationRoleConversion: { method: "POST", path: "/api/v1/projects/{project}/applications/{application}/role-conversions/preview" },
  readApplicationRoleConversion: { method: "GET", path: "/api/v1/projects/{project}/applications/{application}/role-conversions/{conversion}" },
  applyApplicationRoleConversion: { method: "POST", path: "/api/v1/projects/{project}/applications/{application}/role-conversions/{conversion}/apply" },
  previewApplicationRoleConversionRollback: { method: "POST", path: "/api/v1/projects/{project}/applications/{application}/role-conversions/{conversion}/rollback-preview" },
  readProjectMemberCandidate: { method: "GET", path: "/api/v1/projects/{project}/member-candidates/{user}" },
  readProjectOwnership: { method: "GET", path: "/api/v1/projects/{project}/ownership" },
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
  listHTTPSProbes: { method: "GET", path: "/api/v1/projects/{project}/probes" },
  createHTTPSProbeBinding: { method: "POST", path: "/api/v1/projects/{project}/probes" },
  readHTTPSProbeBinding: { method: "GET", path: "/api/v1/projects/{project}/probes/{probe}" },
  updateHTTPSProbeBinding: { method: "PUT", path: "/api/v1/projects/{project}/probes/{probe}" },
  readRepositorySourceOptions: { method: "GET", path: "/api/v1/projects/{project}/repository-source-options" },
  listRepositorySources: { method: "GET", path: "/api/v1/projects/{project}/repository-sources" },
  readRepositorySource: { method: "GET", path: "/api/v1/projects/{project}/repository-sources/{source}" },
  saveRepositorySource: { method: "PUT", path: "/api/v1/projects/{project}/repository-sources/{source}" },
  listRepositorySyncs: { method: "GET", path: "/api/v1/projects/{project}/repository-sources/{source}/syncs" },
  enqueueRepositorySync: { method: "POST", path: "/api/v1/projects/{project}/repository-sources/{source}/syncs" },
  readRepositorySync: { method: "GET", path: "/api/v1/projects/{project}/repository-sources/{source}/syncs/{sync}" },
  listProjectAssignments: { method: "GET", path: "/api/v1/projects/{project}/role-assignments" },
  createProjectAssignment: { method: "POST", path: "/api/v1/projects/{project}/role-assignments" },
  replaceProjectStandardAssignment: { method: "PUT", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}" },
  revokeProjectAssignment: { method: "DELETE", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  readProjectAssignment: { method: "GET", path: "/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}" },
  previewProjectRoleConversion: { method: "POST", path: "/api/v1/projects/{project}/role-conversions/preview" },
  readProjectRoleConversion: { method: "GET", path: "/api/v1/projects/{project}/role-conversions/{conversion}" },
  applyProjectRoleConversion: { method: "POST", path: "/api/v1/projects/{project}/role-conversions/{conversion}/apply" },
  previewProjectRoleConversionRollback: { method: "POST", path: "/api/v1/projects/{project}/role-conversions/{conversion}/rollback-preview" },
  checkStatusPageSlug: { method: "GET", path: "/api/v1/projects/{project}/status-page-slugs/{slug}" },
  listStatusPages: { method: "GET", path: "/api/v1/projects/{project}/status-pages" },
  deleteStatusPage: { method: "DELETE", path: "/api/v1/projects/{project}/status-pages/{page}" },
  readStatusPage: { method: "GET", path: "/api/v1/projects/{project}/status-pages/{page}" },
  saveStatusPage: { method: "PUT", path: "/api/v1/projects/{project}/status-pages/{page}" },
  previewStatusPage: { method: "POST", path: "/api/v1/projects/{project}/status-pages/{page}/preview" },
  publishStatusPage: { method: "POST", path: "/api/v1/projects/{project}/status-pages/{page}/publish" },
  unpublishStatusPage: { method: "POST", path: "/api/v1/projects/{project}/status-pages/{page}/unpublish" },
  changeSuiteLifecycle: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/actions/{action}" },
  listBrowserRecordings: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings" },
  startBrowserRecording: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings" },
  readBrowserRecordingOptions: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings/options" },
  readBrowserRecording: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings/{id}" },
  changeBrowserRecordingState: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings/{id}/actions/{action}" },
  createBrowserRecordingPairing: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings/{id}/pairing" },
  readBrowserRecordingSource: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/browser-recordings/{id}/source" },
  listCatalogDrafts: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts" },
  createCatalogDraft: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts" },
  readCatalogDraft: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts/{id}" },
  updateCatalogDraft: { method: "PUT", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts/{id}" },
  reviewCatalogPublication: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts/{id}/publication-reviews" },
  publishReviewedCatalog: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts/{id}/publish" },
  validateCatalogDraft: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-drafts/{id}/validate" },
  exportCatalogDocument: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/catalog-exports" },
  previewCatalogImport: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-imports" },
  readCatalogPlan: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/catalog-imports/{id}" },
  applyCatalogPlan: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/catalog-imports/{id}/apply" },
  readSuiteLifecycle: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/lifecycle" },
  listPlaywrightDrafts: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/playwright-drafts" },
  readPlaywrightStarter: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/playwright-drafts/starter" },
  readPlaywrightDraft: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/playwright-drafts/{id}" },
  savePlaywrightDraft: { method: "PUT", path: "/api/v1/projects/{project}/suites/{suite}/playwright-drafts/{id}" },
  preparePlaywrightDraft: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/playwright-drafts/{id}/prepare" },
  readCatalogPublicationReview: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/publication-reviews/{id}" },
  readReleaseRegistry: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/release-registry" },
  listCatalogReleaseRecords: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/releases" },
  readCatalogReleaseRecord: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/releases/{id}" },
  activateRetainedRelease: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/releases/{id}/activate" },
  listSavedSelections: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections" },
  createSavedSelection: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections" },
  readSavedSelection: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections/{id}" },
  updateSavedSelection: { method: "PUT", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections/{id}" },
  changeSavedSelectionState: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections/{id}/actions/{action}" },
  previewSavedSelection: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/saved-selections/{id}/previews" },
  updateSuiteSettings: { method: "PATCH", path: "/api/v1/projects/{project}/suites/{suite}/settings" },
  uploadSourceBundle: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-bundles" },
  readPreparationOptions: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparation-options" },
  listSourcePreparations: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations" },
  enqueuePrebuiltPreparation: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations" },
  readSourcePreparation: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}" },
  downloadSourceBundle: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}/bundle", responseType: "blob" },
  cancelSourcePreparation: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}/cancel" },
  readPreparationEditorSource: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-preparations/{id}/editor-source" },
  listSourceUploads: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads" },
  stageSourceUpload: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads" },
  readSourceUpload: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}" },
  readSourceContent: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}/content" },
  publishSourceUpload: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/source-uploads/{id}/publish" },
  listSuiteCatalog: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/tests" },
  createCatalogTest: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/tests" },
  previewCatalogBulk: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/tests/bulk-preview" },
  readCatalogTest: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}" },
  updateCatalogTest: { method: "PUT", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}" },
  duplicateCatalogTest: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}/actions/duplicate" },
  changeCatalogTestState: { method: "POST", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}/actions/{action}" },
  readRetainedTest: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}/retained" },
  listCatalogTestRevisions: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}/revisions" },
  readTestRunHistory: { method: "GET", path: "/api/v1/projects/{project}/suites/{suite}/tests/{test}/run-history" },
  listProjectCatalog: { method: "GET", path: "/api/v1/projects/{project}/tests" },
  readPublicStatusPage: { method: "GET", path: "/api/v1/public/status-pages/{token}" },
  readRecordingCompanionControl: { method: "POST", path: "/api/v1/recording-companion/sessions/{id}/control" },
  pairRecordingCompanion: { method: "POST", path: "/api/v1/recording-companion/sessions/{id}/pair" },
  submitRecordingCompanionSnapshot: { method: "POST", path: "/api/v1/recording-companion/sessions/{id}/snapshots" },
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
  cancelCLISignIn: { method: "POST", path: "/auth/cli/cancel" },
  exchangeCLISignIn: { method: "POST", path: "/auth/cli/exchange" },
  readCLISignInOptions: { method: "GET", path: "/auth/cli/options" },
  readCLISignInRequest: { method: "GET", path: "/auth/cli/requests/{request}" },
  confirmCLISignInRequest: { method: "POST", path: "/auth/cli/requests/{request}" },
  signOut: { method: "POST", path: "/auth/logout" },
  signInWithPassword: { method: "POST", path: "/auth/password" },
  changeLocalPassword: { method: "POST", path: "/auth/password/change" }
} as const
