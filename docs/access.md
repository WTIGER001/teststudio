# Identity administration

The backend implements the [System and scoped role plan](roles-plan.md) through API contract 0.37.0.
This implementation is unreleased. The existing hosted release and membership screens remain unchanged.
Use the standard role APIs below for new integrations. Legacy roles and custom permissions retain their stored assignments.

TestStudio stores user profiles, external identity links, groups, and membership provenance in its selected database.
The permission library evaluates access. The directory does not infer project authority from an email address or group name.

Users and Groups screens use the administrative APIs for profiles, account state, group metadata, local membership, and session revocation.
Managed password screens use the same directory, sessions, and permission checks.
Managed custom role APIs support scoped rules, inheritance, disabled state, and assignment.
Custom role screens use the same typed API, scope checks, and revision guards.
Direct permission screens manage user and group rules through the same generated client.
Explain access screens show current decisions and authorized source details through the same permission evaluator.
Identity provider screens manage sign-in configuration and exact group mappings.
User-owned identity linking and project-owned pending access use explicit, audited workflows.
The full acceptance ledger remains in [Implementation verification](implementation.md).
The [OpenAPI contract](../contracts/openapi.yaml) describes identity, role policy, direct grants, permission catalog, and assignment routes. Other route families remain pending.

## Standard Project and Application roles

| Role identifier | Includes |
| --- | --- |
| builtin.teststudio.reader | Read permitted definitions, results, trends, schedules, and evidence. |
| builtin.teststudio.member | Reader access plus published execution, cancellation, and execution schedule management. |
| builtin.teststudio.developer | Member access plus test authoring, publication, suites, sources, and repository synchronization. |
| builtin.teststudio.contributor | Developer access plus settings, environments, notifications, pins, and non-Owner membership management. |
| builtin.teststudio.owner | Contributor access plus deletion and adding, removing, promoting, or demoting Owners. |

Project roles inherit into every application in the project. Application roles apply to one application.
Combine direct and group permissions through permissions.Service. Denies, role disables, current membership, and expiry still apply.
A lower Application role does not reduce an inherited Project role.
System Admin keeps the compatible builtin.teststudio.installation-admin identifier and System-wide authority.
Standard roles grant neither System administration nor protected execution or capture overrides.
Changing a protected environment to unprotected requires separate policies.manage authority.

Contributor checks protect all Owner paths, including group membership and custom policies, even when another Owner remains usable.
Contributors cannot delegate deletion or ownership through compatibility roles, custom role rules, or direct grants.
Project group ownership and System group authority remain separate from Application membership administration.
Owners still need authority over a shared group's stored scope before assigning it Application access.

The server resolves stored resource context before evaluating Application roles.
It filters project discovery, application and environment lists, catalogs, runs, trends, and automation by current access.
Application-only members receive only the parent project's identifier and name, without project settings.
Shared suites remain Project-owned. Filtered catalog reads expose authorized tests; complete source access, authoring, publication, activation, and link changes require Project authority.
Full mixed-application run payloads and their logs, exports, and artifacts remain unavailable to Application-only members.
Runs keep frozen application context. Later suite relinking cannot expand historical run access.
Scheduled execution and artifact downloads recheck current authority. Removing a role rejects the next request from the existing session and blocks the next schedule firing.

### Scoped membership endpoints

Use /api/v1/projects/{project}/role-assignments for Project roles.
Use /api/v1/projects/{project}/applications/{application}/role-assignments for Application roles.
The Application family supports the five standard roles. Project routes also retain compatibility and custom roles.
GET lists bounded direct assignments. Inherited sources appear through effective-permission explanations.
POST assigns a role with principalKind, principalId, roleId, and reason.
GET or DELETE /{kind}/{principal}/{role} inspects or revokes an exact assignment. Deletion requires its current If-Match revision and a reason.
Routes derive bindings. Bodies cannot supply project, application, or binding overrides.

PUT /{kind}/{principal} atomically replaces a standard role. Supply roleId, previousRoleId, reason, and the previous assignment's If-Match revision.
Both the previous role and revision must match. Different role records can have the same numeric revision.
To create the first standard role with PUT, use If-Match: 0 and omit previousRoleId.
Only one direct standard role is allowed for each principal and exact scope. Custom and compatibility paths remain independent.
Self changes are authorized against the complete proposed batch before committing. Concurrent replacements produce one winner and an explicit conflict.

GET /api/v1/roles?project={project}&application={application} lists standard Application choices and current delegation capability.
GET /api/v1/me/capabilities accepts the same optional application parameter.
GET /api/v1/me/effective-permissions and the authorized user explanation endpoint also accept application.
Explanation results include applicationId and readable assignmentApplication provenance. Cursors bind the complete selected context.

GET /api/v1/projects/{project}/member-candidates/{user} returns an authorized exact user's identifier, display name, and disabled state.
The Application equivalent sits under /applications/{application}. This lookup grants no System directory listing, email, credential, or identity-link access.
GET ownership at either scope returns system-managed, owned, or owner-unavailable.
A usable Owner needs an enabled account and role, current membership, an available sign-in path, and effective management authority.
Application ownership can rely on a usable Project Owner.
Guarded mutations preserve the last usable Owner. External expiry can still create an owner-unavailable state; System Admin can repair it through normal audited assignments.

### Reviewed role conversion

Use /api/v1/projects/{project}/role-conversions. The Application equivalent sits under /applications/{application}.
POST /preview accepts the assignment definition and target standard role. It records the principal's current predefined assignments and potential added and removed role actions.
These action lists describe the role definitions. Existing custom roles, direct grants, and denies still determine effective access.
Preview changes no assignments. It expires after ten minutes and binds the actor, exact scope, directory revision, and prior assignment revisions.

GET /{conversion} reads the actor's retained review or receipt.
POST /{conversion}/apply uses the preview record's If-Match revision. Apply rechecks current authority and commits assignments, metadata, and audit records together.
A changed directory, disabled target role, stale assignment, or expired review requires a new preview.
Repeated successful apply requests return the receipt while the actor retains read authority.
An actor who removes their own management access may need another authorized manager to inspect current membership.

POST /{conversion}/rollback-preview with a reason prepares a new review that restores the original predefined role set.
Apply that new review through its normal apply endpoint. Current delegation, revision, and usable-Owner safeguards still apply.
A rollback that would remove the final usable Owner requires another usable Owner first.
The conversion never removes custom roles or direct grants and never runs automatically at startup.
Legacy Operator is a Member conversion candidate, but its schedule-delete action appears as a loss requiring review.
Legacy Project administrator does not automatically become Contributor or Owner.

Focused local API, permission-service, creator, conversion, and compatibility checks pass on SQLite and PostgreSQL.
See [implementation verification](implementation.md) for the retained check totals and separate target acceptance.

## Manage membership in the interface

Open Project members under Project administration, or Application members under Application administration.
System administration contains Users, System groups, System roles, and System access.
System Admin manages the installation. Project and Application roles use Reader, Member, Developer, Contributor, and Owner.
Group remains the shared collection of users; assigning a group grants its active members the selected scoped role.

Select + Add role, choose User or Group, and enter its permanent registered identifier.
Choose a role and enter an Assignment reason. Standard roles require Review assignment before Assign role.
The review verifies an exact user name without exposing the System directory and displays the scope and new powers.
Choose Member to run published tests, Developer to author and publish, or Contributor to delegate scoped management without deletion or Owner changes.
Assign an Application role to limit new powers to that application. Project roles inherit into every application.

Change role replaces one direct standard role after review. Remove role removes only the selected assignment.
Application members shows readable inherited project assignments as read-only. Inspect access includes group paths and exact application context.
A lower direct role does not reduce inherited access. Current server delegation determines which rows and choices can change.
To transfer ownership, create another usable Owner first. Then change or remove the original assignment.

Advanced roles retain compatibility and custom policies. Review conversion previews all predefined assignments replaced for that principal and scope.
Inspect added and removed permissions, then apply the exact review. Custom roles and grants remain.
Reload conversion receipt after an unconfirmed response. Advanced recovery prepares a new rollback review using current authority and Owner safeguards.
Conflicts preserve form values and require inspection before reopening a changed assignment or preparing a new conversion.
Help preserves edits. Leaving unfinished membership work requires confirmation.

## Grant acceptance coverage

Acceptance 12 covers the same permission-service and managed API behavior on SQLite and local PostgreSQL.
The selected checks use the pinned permission library. They do not substitute a separate permission evaluator.

| Required behavior | Verified coverage |
| --- | --- |
| User and group grants | Direct managed grants, group membership, inherited role access, and group-deny precedence. |
| Role grants | Custom parent and child rules, disabled policies, exact assignments, and atomic policy changes. |
| Object scope | Exact suite rules reject other suites, unscoped requests, and the same relative object in another project. |
| Project scope | Server-derived team bindings preserve independent alpha and beta assignments. |
| Explicit denies | Applicable direct user, direct group, and inherited role denies override allows. |
| Expiry | Expired allows withdraw access. Expired denies remain inspectable and stop blocking access. |
| Inheritance | Alternate paths retain access. Removing the final path or disabling a required policy withdraws access. |
| Revocation | Removing an exact binding updates an existing evaluator without removing another project's binding. |
| Administrative boundaries | Read-only access, object denies, filtered lists, conflict revisions, and audited mutations remain enforced. |

The focused nine-check selection runs with race detection on both local databases.
Evidence is output/access-acceptance/summary.json and its database-specific logs.
It includes managed grant, custom role, assignment, pending policy, and inheritance-removal checks.
The check removes its owned PostgreSQL container. No browser or cloud deployment is involved.
See [Implementation verification](implementation.md) for completion counts and separate remaining acceptance scope.

## Typed client contract

Generate identity request types, response types, operation methods, and route templates from the OpenAPI contract:

```sh
go run ./cmd/teststudio-client
go run ./cmd/teststudio-client --check
```

Run these commands from the repository root.
The generated module contains the identity and access operations, including providers, explanations, role state, grants, previews, and custom assignment choices.
The shared client adapter supplies same-origin sessions, request-forgery protection, and revision headers.
The client escapes path identifiers and preserves explicit query values.
Generator checks reject stale output. Nullable expiry and required fields retain their contract types.
User read views fill missing legacy session generations without rewriting stored records or revisions.

## Users and sessions

Installation user permissions control directory listing, profile edits, account state, external identity inspection, and active session inspection.
Project administration does not grant installation user authority.
Lists apply object permission checks before returning records.

User creation registers a profile with a permanent identifier and optional email.
It does not send an invitation, link an external identity, or grant project access.
Matching email addresses never link accounts.
Email changes clear verification. Clients cannot write verification state, creation time, sign-in time, disabled state, or session generations through profile edits.

Sign-in saves its time and session in one transaction.
Disabling a user increments its session generation. Old sessions fail authentication and private artifact checks.
Restoration does not revive those sessions. A new sign-in uses the current generation.
Single-session revocation checks the session owner and current revision.
Session responses and audit records omit cookies, provider tokens, and request-forgery tokens.

Authenticated users can read their own profile and linked identities through `/api/v1/me` without installation directory permissions.
Inspecting another user's identity links requires authorized directory access.

## Group ownership and provenance

Groups belong to the installation or one project. Ownership stays permanent after creation.
An empty group-list project filter selects installation groups. It does not select every project.
Project administrators can manage only authorized groups in their project.
Each resource check uses the stored group scope. A request cannot reparent another project's group to acquire authority.

Membership records retain their user, group, source, and optional expiry.
Local membership editors can change only local membership in locally managed groups.
Provider-managed membership remains read-only. Update its source provider instead.
Expired membership does not grant effective access.

Membership alone grants no project permissions. Role assignments and explicit grants determine access through the permission library.
Archive removes a group's effective membership access while preserving its records and assignments.
Restore reactivates retained, unexpired membership. It can restore the associated role access.

## Interface

Open Users from System administration to register profiles and inspect user details.
User details show account state, email verification, linked identities, readable memberships, and active sessions.
Use Edit profile to change permitted fields. Account actions use separate reason forms.
Session revocation identifies the current browser before confirmation.

Open System groups for shared groups owned by the installation.
Select a workspace and open Project groups for that project's directory.
System navigation uses System capabilities. Project navigation uses the selected project's capabilities. Application navigation uses the exact selected application capabilities.
The server still checks each requested resource.

Group details show permanent ownership, lifecycle state, description, membership source, and member activity.
Local membership forms accept a registered user identifier and optional future expiry.
Expiry inputs use UTC in both themes, independent of the computer's local timezone.
Provider-managed membership displays its source and remains read-only.
Archived groups retain visible membership records and require restoration before local membership edits.

Profile, group, and membership conflicts keep unsaved form values.
Explicit reload actions retrieve the current record and revision before another save.
Lifecycle and removal conflicts require cancellation, refresh, and a newly opened action.
Changing ownership scope closes the previous editor and prevents stale responses from replacing the current view.

Each collection provides explicit refresh and bounded continuation controls.
Read failures remain separate from valid empty pages. Authorization failures clear cached collection data.
The interface uses shared light and dark theme tokens. Tables scroll inside narrow screens.
Users, User details, Groups, and Group details each provide contextual Introduction, Concepts, and Tutorials help.

## Revisions, audit, and pagination

Updates, lifecycle actions, membership changes, and session revocation require `If-Match`.
Use zero to create local membership. Existing records require their current positive revision.
Conflicts return 409. Read the current record before retrying.
Lifecycle changes, membership removal, and session revocation require a nonblank reason.
Identity changes save actor, scope, resource, prior value, new value, and time in the audit transaction.

Lists use bounded scans and apply current resource permissions.
A filtered page can contain no records while its cursor permits another scan.
Coverage appears through the cursor and, for filtered detail collections, the `partial` field.
The default scan limit is 30. The maximum is 1000.
Opaque cursors use authenticated encryption and a durable installation key.
They bind to the caller, collection, selected resource, ownership scope, and archive filter.
Cursors cannot reveal filtered identifiers or move between query contexts.

## Administrator protection

Ordinary account disabling, group archival, membership removal, and installation role revocation preserve the last usable installation administrator.
Managed installation role and direct grant changes use the same protection.
Predefined role overrides also use this protection.
Usable administration requires system management, user management, permission inspection and delegation, role inspection and management, and membership inspection and management.
The check evaluates the proposed directory state through the same permission library and built-in definitions used for requests.
Explicit denies and active membership remain authoritative. Alternate group paths can preserve access.
Unlinked accounts and accounts with disabled sign-in providers do not count as usable recovery accounts.
The configured loopback administrator provides an independent local sign-in path.

A shared directory revision prevents competing administrative mutations from passing stale checks and removing both administrators.
Directory provisioning, membership synchronization, profile changes, and local recovery also advance that revision.
Role and direct grant administration join this coordination. Future provider administration must also join it before its write APIs become available.
Provider outages and external membership expiration can still require operator recovery.

## Operator recovery

Stop the local service before recovery. Use the same database and private artifact directory:

```sh
go run ./cmd/teststudio --local --recover-local-admin \
  --database .data/teststudio.db --artifact-directory .data/artifacts
```

The command requires explicit local mode and a loopback listen address.
It enables the dedicated local administrator, restores its administrator assignment, advances its session generation, and records the recovery action.
It also enables the predefined installation administrator role when disabled and advances its managed override revision.
Ordinary startup preserves disabled predefined roles.
These changes share one database transaction.
Select Open local workspace to create a fresh session.
After recovery, stop this process and return to the installation's normal startup command.

Ordinary startup does not restore revoked administrator assignments.
Local and external bootstrap operations persist initialization markers.
They grant administration only when they create a new bootstrap profile.
Existing profiles retain their account state and assignments when an older installation adopts these markers.
A disabled or revoked external bootstrap account does not prevent startup.
Use authorized role management or explicit operator recovery to restore administration.
Changing an existing profile's bootstrap configuration does not elevate that profile.
Ordinary local startup does not restore disabled users or advance session generations.
Keep operator access to the database and host under installation control.

## Routes

| Method | Route | Operation |
| --- | --- | --- |
| GET | `/api/v1/me` | Read the current profile and a bounded identity-link page. |
| GET, POST | `/api/v1/users` | List authorized users or register a pending profile. |
| GET, PUT | `/api/v1/users/{user}` | Read or edit profile fields. |
| POST | `/api/v1/users/{user}/disable` | Disable and invalidate earlier sessions. |
| POST | `/api/v1/users/{user}/restore` | Restore without reviving sessions. |
| GET | `/api/v1/users/{user}/identities` | Inspect linked issuer and subject pairs. |
| GET | `/api/v1/users/{user}/memberships` | Inspect readable local and external group memberships. |
| GET | `/api/v1/users/{user}/sessions` | Inspect active sessions without credentials. |
| DELETE | `/api/v1/users/{user}/sessions/{session}` | Revoke one session with its revision. |
| GET, POST | `/api/v1/groups` | List one ownership scope or create a local group. |
| GET, PUT | `/api/v1/groups/{group}` | Read or edit group metadata. |
| POST | `/api/v1/groups/{group}/archive` | Archive while preserving records and assignments. |
| POST | `/api/v1/groups/{group}/restore` | Restore retained membership access. |
| GET | `/api/v1/groups/{group}/members` | Inspect membership origin, expiry, and activity. |
| GET | `/api/v1/groups/{group}/members/{user}` | Read one membership and its current revision. Optional `source` defaults to `local`. |
| PUT, DELETE | `/api/v1/groups/{group}/members/{user}` | Save or remove local membership with a revision. |
| GET | `/api/v1/roles?project={project}` | List readable predefined project roles and delegation capabilities. Empty scope selects installation roles. |
| POST | `/api/v1/roles` | Create a managed custom role with scoped rules, inheritance, and a reason. |
| GET | `/api/v1/roles/custom?project={project}` | Scan readable custom roles in one exact ownership scope. |
| GET | `/api/v1/roles/choices?project={project}` | Scan custom assignment choices with current delegation authority. |
| POST | `/api/v1/roles/preview` | Compare current and proposed user access without committing policy changes. |
| GET, PUT | `/api/v1/roles/{role}` | Read or update a custom role with its current revision. |
| GET | `/api/v1/predefined-roles` | Inspect all readable predefined definitions and their installation-wide state. |
| GET, PUT | `/api/v1/predefined-roles/{role}` | Inspect or change predefined disabled state with its current positive revision. |
| GET | `/api/v1/permissions?project={project}&object={object}` | Read action descriptions and current grantability for an exact context. |
| GET, POST | `/api/v1/projects/{project}/role-assignments` | Scan or create predefined and managed custom project assignments. |
| GET, DELETE | `/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}` | Read or revoke one exact project assignment. |
| GET, POST | `/api/v1/role-assignments` | Scan or create predefined and managed custom installation assignments. |
| GET, DELETE | `/api/v1/role-assignments/{kind}/{principal}/{role}` | Read or revoke one exact installation assignment. |
| GET, POST | `/api/v1/projects/{project}/permissions` | Scan or create managed direct project grants. |
| GET, PUT, DELETE | `/api/v1/projects/{project}/permissions/{grant}` | Read, change expiry, or remove one managed project grant. |
| GET, POST | `/api/v1/permissions/grants` | Scan or create managed direct installation grants. |
| GET, PUT, DELETE | `/api/v1/permissions/grants/{grant}` | Read, change expiry, or remove one managed installation grant. |


## Local verification

Run the focused API and concurrency checks:

```sh
go test -race ./internal/access ./internal/httpapi ./internal/sessions \
  ./internal/authorization ./adapters/auth/oidc ./cmd/teststudio \
  -run 'TestAccess|TestSession|TestPermission|TestOIDC|TestIncompleteGroups|TestLocalRecovery|TestBootstrap' -count=1
```

Run the same checks on a disposable local PostgreSQL service:

```sh
bash scripts/test-postgres.sh ./internal/access ./internal/httpapi ./internal/sessions \
  ./internal/authorization ./adapters/auth/oidc ./cmd/teststudio \
  -run 'TestAccess|TestSession|TestPermission|TestOIDC|TestIncompleteGroups|TestLocalRecovery|TestBootstrap'
```

Run the built service with a fresh SQLite installation and actual restart and recovery operations:

```sh
bash scripts/test-access.sh
```

The smoke script continues an encrypted group cursor and verifies assignment persistence, revocation, and recreation revisions across actual service restarts.
It uses no cloud credentials. It stops its owned controllers and retains an isolated database and redacted summary.

Run the focused interface workflow through normal TestStudio publication, execution, timing, and evidence storage:

```sh
bash scripts/test-self.sh --test=STUDIO-ACCESS-001
```

The workflow uses new local profiles and groups. It verifies real revision conflicts, reason forms, UTC expiry, archive, restore, and session revocation.
It also verifies scope changes, bounded pages, final-administrator protection, preserved help edits, light and dark layouts, and mobile document width.
An isolated product-issued fixture session verifies project-administrator navigation, foreign-project rejection, and provider-managed membership controls through the real server.
The expiry check runs in a non-UTC browser timezone and compares the stored UTC timestamp.
Expected rejected requests bind to exact methods, addresses, statuses, and individual browser resource diagnostics.
Other browser runtime errors still fail the workflow.
The worker retains diagnostic evidence and removes its owned browser container after verification.

## Atomic assignment storage

Assignment changes use a shared database transaction for permission assignments, metadata, and audit records.
The transaction advances the directory revision before it reads authorization or administrator state.
Competing directory changes use revision checks against that same record.
A rejected workflow rolls back its assignment changes, audit records, and directory revision together.
The permission library reads pending assignments through the transaction before it evaluates access.
PostgreSQL keeps assignments in its existing native permission tables.
Its transaction adapter supports assignment changes and rejects other policy writes.
SQLite uses transaction-bound record storage.
An exact revocation removes one role and binding combination.
An empty binding map does not remove assignments with project bindings.
The assignment API and interface use this transaction. Custom policy storage uses the separate transaction described below.

## Atomic policy storage

`authorization.Service.WithPolicyTransaction` binds directory metadata and the complete permission store to one database transaction.
It advances the shared directory revision before authorization reads and policy changes.
Custom roles, grants, inheritance, assignments, role overrides, and audit records share that transaction.
The permission service evaluates pending writes against the transaction-bound directory and store.
Rejected changes roll back all policy writes, metadata, and the directory revision.
Competing changes through independent database connections produce a revision conflict instead of partial policy state.

PostgreSQL uses the native permission tables in `teststudio_permissions` through the metadata transaction.
SQLite uses its existing permission records through the same bound storage transaction.
No alternate permission evaluator is introduced.
Both stores require existing custom inheritance endpoints and reject cycles.
Custom role display names can repeat. Permanent identifiers remain unique.
Grant inspection includes literal project and object scopes on both databases.
The permission library still evaluates exact scope, explicit denies, disabled roles, and expiry.

The pinned library has no single-grant removal or inheritance removal service operation.
TestStudio supplies these storage operations through `authorization.Service.RemovePolicyGrant` and `RemoveRoleInheritance`.
Both operations require an active policy transaction.
Single-grant removal preserves other grant identifiers.
Inheritance removal preserves alternate paths and rebuilds PostgreSQL closure within the transaction.
All access decisions continue through the permission library.

Focused SQLite and local PostgreSQL checks cover pending evaluation, complete rollback, exact object boundaries, direct reports, and grant pagination.
They also cover disabled overrides, expired denies, independent connections, retained grant identifiers, and alternate inheritance paths.
Managed role APIs use this storage foundation.
Direct principal grant APIs and editors use this transaction. Read-only explanations use the same permission evaluator.

## Managed custom role policies

Create a role with `POST /api/v1/roles`. Read or update it through `/api/v1/roles/{role}`.
Updates require a positive `If-Match` revision. Every mutation requires a nonblank reason.
The identifier and owning project remain permanent. An empty `projectId` selects installation ownership.
Identifiers are globally unique. The `builtin.` prefix and the `custom` and `choices` route identifiers are reserved.
Names permit 200 bytes. Descriptions permit 4096 bytes.

Each rule contains a full registered `permission`, an `allow` or `deny` effect, and optional `object` and `expiresAt` values.
Project rules use the owning project as an exact native team scope. Project policies reject installation actions.
Installation rules require installation authority. They use the native installation-wide scope.
Object rules use exact relative identifiers. Supported forms include `suite/health`, `test/health/check`, and `run/{id}`.
Environment, application, group, user, role, and managed grant identifiers are also supported where their stored ownership matches the rule scope.
The service loads each resource before it accepts a new object rule.
User directory objects require installation scope. Wildcards and caller-supplied native binding fields are rejected.
The service prefixes project objects with their permanent project identifier.
Object rules do not imply access to descendant resources.

Expiry normalizes to UTC microseconds on both databases. Expired rules remain inspectable and grant no access.
Unchanged rules retain their native grant identifiers during metadata edits.
One role permits 100 rules and 16 directly inherited roles. One ownership scope permits 256 managed custom roles.
Inheritance uses existing managed roles in the same ownership scope. Cycles and paths beyond 16 inheritance edges are rejected.
The logical graph includes disabled definitions, so disabling cannot conceal a cycle for later activation.
Changes require management and delegation authority for affected parent roles as well as the edited role.

Custom rule changes require delegation authority and the actual rule action in the rule's exact context.
Custom role assignment checks those conditions through inherited rules and each inherited role.
Project administrators receive no implicit custom-policy management permission.
Delegating custom-policy administration requires an explicit `teststudio.roles.manage` grant in the appropriate scope.
Existing user, group ownership, project, and assignment revision guards still apply.

Disabled policies retain their rules and inheritance in metadata.
Their native grants and outgoing inheritance are withdrawn atomically, including access through child roles.
Enabling revalidates the stored policy and restores its grants and inheritance.
Installation policy changes evaluate the proposed native state before commit and preserve the last usable administrator.
Policy, ownership index, audit, and directory revision changes commit together.
Rejected changes leave all those records unchanged.

`GET /api/v1/roles/custom` scans bounded pages in the selected scope.
Object checks filter unreadable records. An empty page can retain a continuation cursor.
Cursors bind to the actor and owning scope. Installation pages contain no project-owned policies.
`GET /api/v1/roles` remains the predefined role-choice endpoint. Custom choices use the separate paginated catalog.

`GET /api/v1/permissions` provides registered action descriptions and server-computed `canGrant` values.
Supply an exact project and optional object to inspect that context.
The service checks stored object ownership and current delegation and action permissions.
Custom role details return server-computed `canManage` and `delegatable` values.
Catalog pages return `canCreate` for the current scope. Archived projects suppress mutation capabilities.
Custom assignment choice pages use a separate actor-bound cursor and current delegation checks.
Direct principal grants and predefined role overrides use their dedicated administration routes.
The effective-access explanation workflow appears below.

## Proposed policy preview

`POST /api/v1/roles/preview` accepts `policy`, its current `revision`, a registered `userId`, and an optional exact `object`.
Use revision zero for a new policy. Existing policies require their opened revision.
The preview validates the same proposed policy and administrator protections as an actual save.
It also requires scoped user membership management, delegation authority, and permission inspection authority for the selected context.

The service evaluates current access before it writes temporary policy state.
It then evaluates the proposal with the enabled role assigned directly to the selected user.
A disabled policy adds no new assignment. Existing assigned roles still reflect the proposed disabling.
Disabled users remain denied.
Both comparisons call the common permission service. Existing assignments, groups, denies, and expiry remain effective.
The preview always rolls back temporary policy, assignment, audit, and directory revision writes.
It returns one current and proposed decision for each registered action in the selected scope.
These comparisons do not replace the permission-source explanation workflow.

## Custom role interface

Open Project roles inside a selected workspace, or Installation roles from installation navigation.
Roles and permissions shows bounded custom policy pages and code-managed predefined definitions.
Create role uses the server's current creation capability. Role details use current management and delegation flags.
The editor preserves permanent ownership and captures its opened revision.
Refresh role updates the stored view while unsaved editor fields and the opened revision remain unchanged.
Reload stored role explicitly replaces editor fields after review.

Permission rules provide registered action descriptions, allow and deny effects, exact object fields, and full UTC expiry timestamps.
Refresh rule permissions retrieves grantability for the entered object.
The editor preserves timestamp precision and permits inspection of expired rules.
Inheritance controls select existing managed roles and provide bounded continuation choices.
State actions use separate target, impact, reason, and confirmation controls.

Preview effective access accepts a registered user and optional exact object.
It shows current and proposed decisions without saving the policy.
Changed decisions appear first. Show unchanged permissions includes the complete comparison.
Changed policy or preview inputs mark the result out of date. Save remains a separate action with an audit reason.
Conflicts preserve edited fields and block another save until explicit reload or reopening.
Unconfirmed write responses also require state review before retrying.
Lost read access clears private role details, open editor fields, and obsolete asynchronous responses.

Project access and Installation access combine predefined choices with bounded custom choices.
The server supplies assignment authority. Custom choices link to their exact policies instead of flattening deny rules into allow lists.
Both new pages provide Introduction, Concepts, and Tutorials help. Help preserves unsaved policy fields.
Light and dark themes share the existing tokens. Narrow tables scroll inside labeled, keyboard-focusable regions.
Table row headers retain readable case for exact permission identifiers.

Run the focused browser workflow through normal TestStudio execution and evidence capture:

```sh
bash scripts/test-self.sh --test=STUDIO-POLICY-001
```

Run focused checks on SQLite and disposable local PostgreSQL:

```sh
go test -race ./internal/access ./internal/httpapi -run 'CustomRole|RolePolicyGraph' -count=1
bash scripts/test-postgres.sh ./internal/access ./internal/httpapi -run 'CustomRole|RolePolicyGraph'
bash scripts/test-access.sh
```

The built-service check verifies custom policies, inherited group assignment, exact-object access, and disabled state across actual SQLite restarts.
It uses a new managed local user and random passwords held in memory.
All owned verification controllers exit after the check.

## Initial project creator access

Creating a project requires installation-scoped projects.create permission.
Ordinary creators receive one managed Owner role, bound only to the new project's identifier.
The project and its Owner assignment commit in one transaction with audit records.
If Owner is disabled or the assignment fails, the project and partial access do not commit.
The error identifies the unavailable role. An installation operator must repair that policy before another create attempt.
Explicit denies remain effective. Initial assignments grant no installation privileges or protected-target execution permission.
Assignment routes expose these records with their normal revisions, management timestamps, and revocation controls.
Revoking a role remains effective after project edits. Updating a project never repeats its initial assignments.
A duplicate project identifier returns a conflict without assigning access or changing its owner experience.

An actor with every installation permission already has complete installation administration.
Such actors keep their existing authority without automatic project self-assignment.
They can assign explicit project roles before withdrawing installation administration when retained project access is required.
Existing projects receive no automatic access changes during startup or upgrade.
Existing four-role creator bundles stay unchanged until reviewed conversion.
The legacy Project administrator role still excludes editing, publishing, and running tests.

### Project creation API

POST /api/v1/projects accepts id, name, description, archived, and the existing optional project policies.
The request selects no role identifiers, principals, or assignment bindings.
The response remains the normal project record. Creation returns HTTP 200.
Current session, Origin, and request-forgery protection apply.
Missing authority returns 403. Duplicate identity returns 409. Invalid or unavailable setup policy returns 422.
Use the existing project read route to reconcile an interrupted successful response before retrying creation.
Initial assignment inspection and removal use the ordinary project role-assignment routes.

## Predefined role assignments

Create an assignment with `principalKind`, `principalId`, `roleId`, and a nonblank `reason`.
Use an existing user or group. Local and OIDC profiles use the same permanent internal user identifiers.
The API creates no credentials, identity links, or invitation messages.
The project route determines the exact `team` binding. Request bodies cannot supply bindings or widen the scope.
The installation route supports the installation administrator role.
Project routes support Reader, Member, Developer, Contributor, and Owner, plus legacy viewer, operator, editor, publisher, and project administrator roles.
The same routes also accept managed custom roles in their exact owning scope.
Their literal rule scopes require no caller-supplied bindings. Arbitrary object binding templates remain unavailable.

Adding an assignment requires membership management and permission delegation in the selected scope.
Checks include the target role and principal objects. Explicit denies remain authoritative.
Project administrators can delegate these predefined project roles without receiving every delegated action themselves.
They cannot assign installation administration through a project route.
Group assignments also require group management in the group's stored ownership scope.
Project groups cannot receive roles outside their project.
Installation groups require installation group authority, even when their assignment targets one project.
An authorized installation manager can assign the same installation group different roles in different projects.

Assignment inspection requires membership and permission read authority in the assignment scope.
It exposes the assigned principal's identifier, display name, and activity state.
It does not grant access to that principal's directory profile, group membership, or identity links.
Assignment pages scan a bounded number of native records before applying scope and object permissions.
An empty filtered page can have another continuation cursor. Cursors bind to actor, route family, and exact scope.

Creation rejects disabled users, archived groups, disabled roles, and archived projects.
Revocation remains available for retained inactive assignments when the caller has current authority.
Use `If-Match` and a reason to revoke the exact role and route-derived binding combination.
Removing one project assignment preserves the principal's assignments in other projects.
Revocation retains revision history. Recreating the same assignment does not reuse its previous revision.
Legacy bootstrap assignments remain readable before their first managed mutation.
Operator recovery cannot make an old removal revision valid again.
Assignments without a matching management record omit the management timestamp.

Permission checks use the library's service and current identity records.
Changes reach existing service instances without another sign-in.
The shared directory revision serializes competing role revocation and account-state changes.
The last-administrator check uses the proposed assignment state inside the transaction.
A rejected change leaves its assignment, management record, audit history, and directory revision unchanged.

## Assignment interface

Open Project access from a selected workspace or Installation access from installation navigation.
These navigation entries require the corresponding membership and permission read authority.
Role choices show predefined roles, their registered permissions, disabled state, and current delegation authority.
Select Add role, choose User or Group, enter the registered identifier, select a role, and enter a reason.
The form shows the exact scope. It creates no credentials or identity links.
Local and OIDC profiles use the same internal identifiers.

Assignment rows show principal activity, role state, exact scope, and the last managed change when available.
The server returns each assignment's current removal capability as `canRevoke`.
The interface uses that capability for removal controls. It does not infer shared-group authority from project access.
Removal displays the exact target and impact before it accepts a reason.
The last-administrator safeguard still runs when the action reaches the server.

A creation conflict preserves entered values and blocks another submission until explicit review and reopening.
A removal conflict preserves the reason. Cancel, refresh access, inspect the current record, and open a new action.
Revoking your own access refreshes capabilities and the project directory, then clears unavailable assignment data.
Scope changes close old editors and actions. Late responses cannot replace the new scope.
Refresh and More assignments provide bounded scans. Empty pages and request errors remain distinct.

Both pages provide contextual Introduction, Concepts, and Tutorials help.
Help preserves open edits. Shared theme tokens support light and dark layouts.
Narrow screens keep the table inside a keyboard-focusable horizontal scroll region.
The `STUDIO-ROLES-001` self-test verifies actual creation, stale removal after recreation, role delegation, shared groups, read-only access, and self-revocation.
It also verifies administrator protection, pagination, preserved help edits, both themes, and mobile document width.

Run the focused workflow with:

```sh
bash scripts/test-self.sh --test=STUDIO-ROLES-001
```

Run selected pure interface utility checks with:

```sh
node scripts/test-ui-utils.mjs web/src/app/features/access/assignment.utils.test.ts
```

The utility runner compiles temporary CommonJS files through the pinned TypeScript dependency.
It preserves runtime relative dependencies, accepts Node built-ins, rejects framework imports, and removes temporary output after execution.

## Managed local account API

Enable managed local passwords with `--local-accounts`.
This option can operate alone or alongside configured OIDC providers.
It is separate from `--local`, which enables the explicit loopback development administrator.
Normal password-only operation does not require `--local`.
Use the internal user identifier for password sign-in. Email matching never selects or links an account.
Local credential creation grants no project permissions.

The public origin must use HTTPS. Loopback HTTP is available only with a loopback listener.
A TLS-terminating proxy can forward requests to the configured private service listener.
Use the same authentication flags on replicas that share a database.
The database stores the enabled mode and its revision.
Changing this mode revokes earlier password sessions, including after later reactivation.
OIDC sessions keep their own authentication method and do not inherit password restrictions from a provider's name.

Local passwords use Argon2id with independent random salts.
The parameters use 64 MiB, three passes, four lanes, and a 32-byte output.
These values follow the memory-constrained option in [RFC 9106](https://www.rfc-editor.org/rfc/rfc9106.html#section-7.4).
The implementation uses the pinned [Go crypto package](https://pkg.go.dev/golang.org/x/crypto/argon2).
At most two password operations run concurrently in each process.

Passwords require 15 through 256 Unicode characters and at most 1024 UTF-8 bytes.
NFC normalization applies before hashing. Spaces remain significant.
Control characters, one-character repetitions, and a small built-in list of common values are rejected.
The minimum length follows [NIST password guidance](https://pages.nist.gov/800-63-4/sp800-63b.html#passwordver).
No composition rule requires a particular mixture of character types.
Hashes stay in private credential records. Profile views, credential views, audit records, and application logs do not return them.
Request bodies for password sign-in and password change have an 8 KiB limit.

| Method | Route | Operation |
| --- | --- | --- |
| POST | `/auth/password` | Sign in with `userId` and `password`. Require the configured Origin. |
| POST | `/auth/password/change` | Verify `currentPassword`, set `newPassword`, revoke earlier sessions, and rotate the cookie. |
| GET | `/api/v1/me/local-credential` | Read your own redacted credential state without directory authority. |
| GET | `/api/v1/users/{user}/local-credential` | Read redacted state with installation user read authority. |
| PUT | `/api/v1/users/{user}/local-credential` | Provision or reset a password with installation user management authority, reason, and revision. |
| POST | `/api/v1/users/{user}/local-credential/disable` | Disable local sign-in with a reason and current revision. |
| POST | `/api/v1/users/{user}/local-credential/enable` | Enable an existing valid local credential with a reason and current revision. |

An unconfigured credential reads as revision zero.
Use `If-Match: 0` to provision it. Resets and state changes require the current positive revision.
Administrator provisioning and resets always require a password change.
Reset preserves a credential's disabled state. Enabling remains a separate action.
Password changes, resets, and credential state changes invalidate earlier sessions for that account.
New password sessions bind to the current credential and mode revisions.
A reset cannot reuse password proof obtained against an earlier credential revision.

A reset password creates a restricted session.
That session can inspect its own session, change its password, or sign out.
It cannot use ordinary project, administration, or evidence APIs.
Password change requires the current password and the session's request-forgery token.
Successful change updates the credential, session generation, audit record, and replacement session in one transaction.
The server sends the new cookie only after that transaction commits.
Existing cookie tokens cannot revive after password changes, credential restoration, or authentication mode reactivation.

Sign-in errors use the same response for unknown users, incorrect passwords, and inactive accounts.
Durable counters limit each identifier bucket to ten attempts per fifteen minutes.
Peer buckets permit one hundred attempts in that interval.
The counters include unknown identifiers and use secret-keyed bounded buckets.
There are at most 4096 identifier buckets and 256 peer buckets. Raw identifiers and peer addresses are not stored in those counters.
Peer selection uses the connection address. Forwarded headers do not override it.
A shared proxy can aggregate clients into one peer bucket.
Limits apply across instances and restarts. A limited response includes `Retry-After`.

Enabled local credentials count as administrator sign-in paths only when managed local authentication is active.
Last-administrator checks include local, configured OIDC, and explicit development paths.
Credential disabling, account disabling, group changes, and role revocation use the same directory revision.
Identity transactions lock that revision before they change individual records.
A rejected credential change rolls back its session generation, credential state, audit, and directory revision.

For initial setup, run an explicit loopback administrator with managed local accounts enabled:

```sh
go run ./cmd/teststudio --local --local-accounts
```

Use User details to provision credentials. Use Installation access or Project access to assign permitted roles.
Then restart normal local operation without the development administrator:

```sh
go run ./cmd/teststudio --local-accounts
```

Keep the same database and private artifact directory.
Operator recovery remains available through the explicit loopback recovery command.
Include `--local-accounts` when recovery must manage local credentials.

## Managed local account interface

When managed local accounts are enabled, Sign-in provides User identifier and Password fields beside configured OIDC choices.
Open local workspace remains a separate development action. Normal password-only operation does not display it.
Wrong sign-in retains the identifier and clears the password.
Passwords retain significant spaces. Forms do not store passwords in browser storage.

A required password change replaces the workspace with a restricted form.
The interface makes no ordinary private page requests until the change succeeds.
The form requires the current temporary password and a confirmed new password.
Help remains available and preserves unsaved fields.
Successful changes rotate the session and invalidate earlier sessions.

Open Your account from navigation on desktop or mobile.
The page shows your own profile, local credential state, and bounded external identity pages.
These reads require no installation directory authority.
An enabled credential provides a password-change form that requires the current password.
Your account also supports explicit identity linking. Personal profile editing remains pending.

Authorized User details provide credential provisioning, reset, disabling, and enabling.
Credential reads expose no password or hash.
Editors retain the revision captured when opened. Refreshing visible state does not change that revision.
Conflicts clear submitted passwords, preserve the reason, and require close, refresh, and a new editor.
Unconfirmed server responses also require explicit state review before another save.
Changing the current account's credentials signs it out and clears private workspace state.
State actions identify their target, explain session invalidation, and require a reason.
Reset preserves disabled state. The last usable installation administrator remains protected.

Your account, required password change, and User details provide contextual Introduction, Concepts, and Tutorials help.

Verify native local account operation and actual restarts with:

```sh
bash scripts/test-local-accounts.sh
```

The script uses fresh passwords held in memory and an isolated database.
It preserves a redacted summary and stops every controller it starts.
It verifies forced change, cookie rotation, ordinary restart continuity, local-only administration, mode revocation, and protected transport requirements.

Run the browser workflow through TestStudio with:

```sh
bash scripts/test-self.sh --test=STUDIO-LOCAL-001
```

The isolated worker supports authentication mode restarts through a test-only wrapper. The production service exposes no such route.

## Managed direct grant API

Direct grants assign one registered permission to a user or group with an explicit allow or deny effect.
The route fixes the project. Installation routes use installation-wide native scope.
Optional exact objects use the same validated resource forms as custom role rules.
Project grants reject installation actions, foreign project resources, and wildcard objects.
The permission library evaluates direct grants together with assignments, inherited roles, active groups, explicit denies, and expiry.

Creation requires `principalKind`, `principalId`, `permission`, `effect`, and a nonblank `reason`.
Optional `object` and `expiresAt` fields refine the rule.
The service rejects disabled users, archived groups, and archived projects when creating or renewing grants.
Expiry uses UTC microseconds and UTC years from 1 through 9999. Expired grants remain inspectable and grant no access.

Inspection requires membership and permission read authority at the scope, principal, grant, and exact object.
Mutation requires membership management, permission delegation, and the actual rule action in its exact context.
Group grants also require group authority in the group's stored ownership scope.
Project administrators cannot manage shared installation groups through a project route.
A caller denied the rule action needs another authorized administrator to change that rule.

Each grant has a stable public identifier derived from its permanent scope, principal, permission, effect, and object.
Expiry does not change that identifier.
Use `PUT` with the current positive `If-Match` revision and a reason to change expiry.
Omitted or null expiry removes the expiry limit.
Identity changes require removal and creation. Caller-supplied native identifiers and other server fields are rejected.

Removal requires the current positive revision and a reason.
Removal remains available for retained inactive principals and archived projects when the caller has current authority.
A retained management record preserves revision history after removal.
Recreating the same rule advances its revision. Earlier removal requests cannot remove the recreated grant.

Managed pages include current principal activity, expiry state, and server-computed update and removal capabilities.
They scan bounded management records and filter current object permissions.
Expired rules remain visible. Removed rules do not appear.
Continuation binds to the actor, collection, and exact scope. Empty filtered pages can still permit continuation.
These pages list managed direct grants. They do not claim to list every native grant or explain complete effective access.

Native grants, management records, reasons, audit events, and directory revisions share one policy transaction.
Unrelated native grant identifiers remain stable during expiry changes and removal.
The service rejects unmanaged duplicate adoption and inconsistent native grant records.
Project revision assertions reject an archive that races with a new or renewed grant.
Installation changes evaluate the proposed native state before preserving the last usable administrator.
Rejected changes leave no partial management, permission, audit, or directory state.

Verify the API and independent connection guards on SQLite:

```sh
go test -race ./internal/access ./internal/httpapi -run TestDirectGrant -count=1
```

Run the same checks on disposable local PostgreSQL:

```sh
bash scripts/test-postgres.sh ./internal/access ./internal/httpapi -run TestDirectGrant
```

The built-service access verifier checks actual SQLite restarts, retained expiry, deny precedence, revoked access through existing sessions, and recreation revisions.
The effective-access explanation workflow appears below.

## Direct grant interface

Open Project permissions from the selected workspace or Installation permissions from installation navigation.
Each page requires membership and permission read authority in its displayed scope.
The table shows the principal, activity, permission, exact object, effect, precise expiry, last change, and permitted actions.
Grant identity expands to show the public identifier and revision.
More grants continues a bounded scan. Refresh permissions returns to the first scan.

Select Add direct grant to open the creation form.
Choose User or Group and enter its registered identifier.
Set an optional exact object, then select Refresh grant permissions.
Permission choices show registered descriptions and current delegation authority for that object.
Changing the object invalidates loaded choices until refresh.
Registered action inspection accepts role read authority or permission read authority.
The server still checks current delegation, actual rule actions, principal ownership, and lifecycle state when saving.

Select an effect, optional complete expiry timestamp, and a reason. Select Create grant to save.
An installation rule shows that its scope includes all projects.
An active deny overrides matching allows. Removing a deny can restore access through other grants and roles.

Edit expiry preserves the revision captured when opened.
Refreshing the visible list does not change that revision or unsaved fields.
Conflicts preserve values and require explicit Reload stored grant before another save.
Identity fields remain fixed. Expiry updates send only expiry and reason.
Empty expiry removes the time limit. Past expiry remains visible and grants no access.

Remove grant shows the exact target and impact, then requires a reason.
After a conflict, cancel, refresh, and open a new action against the current revision.
Unconfirmed saves and removals preserve reasons and block repeat submission until explicit state review.
An unreadable rejection retains its original HTTP status. Authorization failures clear private list and editor data.
Scope changes discard previous editors and invalidate late responses.

Both pages provide Introduction, Concepts, and Tutorials help without discarding unsaved values.
Shared theme tokens support light and dark modes. Tables scroll within labeled keyboard-focusable regions on narrow screens.

Run the focused browser workflow through normal TestStudio publication, execution, timing, and evidence storage:

```sh
bash scripts/test-self.sh --test=STUDIO-GRANTS-001
```

Run the pure form utility checks:

```sh
node scripts/test-ui-utils.mjs web/src/app/features/access/grant.utils.test.ts
```

## Predefined role state administration

Open Installation roles to inspect all six predefined definitions and their current disabled state.
Project roles shows the predefined project choices and current state without global mutation controls.
Definitions remain code managed. State changes retain permissions, role identifiers, and existing assignments.
Disabling applies across this installation, including assignments in every project.
Enabling can restore access through retained assignments.

The predefined-role API requires installation role read authority at the scope and the selected role object.
The fixed catalog filters unreadable definitions. It requires no continuation cursor.
State mutation requires installation system management, role management, permission delegation, and authority for every default action.
Exact role-object denies remain authoritative. Project administration cannot change installation-wide role state.
The server returns `canManage` for each readable definition. Inspection alone does not grant state-management authority.

Use `PUT /api/v1/predefined-roles/{role}` with explicit `disabled`, a reason, and the current positive `If-Match` revision.
The API rejects definition edits, missing state, unknown identifiers, and stale revisions.
Public revisions include unmanaged native state. Enabling and disabling retain managed history and advance the public revision.
The response omits a managed timestamp until the first managed state change.
Inconsistent managed metadata and native state produce a conflict instead of silent adoption.

Native overrides, metadata, audit reasons, and directory revisions share one policy transaction.
The administrator guard evaluates the proposed state through the permission library.
Two accounts assigned the same predefined administrator role are both affected by disabling that role.
A complete custom administrator can retain an independent administration path.
Competing role disabling and account removal cannot remove both usable paths.

Every permission check reads current native predefined state before using an evaluation engine.
The engine receives that immutable disabled-role set for its cached library lookups.
A state change therefore reaches existing service instances on their next check.
A storage error prevents evaluation from reusing an earlier allowed state.
Role choices, assignment state, and administrator guards use the same fresh state.
All permission decisions still use the common permission library.

Reason forms retain the role revision captured when opened.
Refresh predefined roles updates visible cards without replacing a pending action's revision or reason.
Conflicts and unconfirmed responses block repeat submission until cancellation, refresh, and a new action.
Authority loss clears private controls and refreshes the page's capabilities.
Introduction, Concepts, and Tutorials help explains the global effect while preserving pending reasons.

Explicit loopback recovery enables the predefined installation administrator role when required.
It advances override metadata, records recovery, restores the administrator assignment, and invalidates earlier local administrator sessions atomically.
Ordinary startup does not enable disabled roles or overwrite override metadata.
The built-service access verifier checks this behavior across actual SQLite restarts and rejects a pre-recovery override revision.

Run the focused browser workflow:

```sh
bash scripts/test-self.sh --test=STUDIO-BUILTIN-001
```

Run the focused database and recovery checks:

```sh
go test -race ./internal/access ./internal/httpapi ./internal/authorization ./cmd/teststudio \
  -run 'TestBuiltinRole|TestLocalRecovery' -count=1
```

Use the disposable PostgreSQL profile for the same contract:

```sh
bash scripts/test-postgres.sh ./internal/access ./internal/httpapi ./internal/authorization ./cmd/teststudio \
  -run 'TestBuiltinRole|TestLocalRecovery'
```

## Effective-access explanations

Open Explain project access for the selected project. Open Explain installation access for installation context.
Both pages start with your own account. Inspection grants no new access.
Select a registered permission. Leave the object empty for the displayed scope or enter one exact existing resource.
Changing a field clears the earlier report and invalidates late responses.
Check access evaluates the current context. Use my account resets the inspected user.

The API provides these read-only endpoints:

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/v1/me/effective-permissions` | Explain your own permission in the requested context. |
| GET | `/api/v1/users/{user}/effective-permissions` | Explain an authorized registered user's permission. |

Supply the full registered `permission` query value. Optional parameters are `project`, `object`, `limit`, and `cursor`.
The default scan limit is 30. Accepted limits are 1 through 1000.
Project context rejects installation-only actions. Exact objects require authoritative ownership in the supplied scope.
Another user's inspection requires scoped membership reading, membership management, and permission reading, including the subject and requested object.
Self-inspection requires project reading for a whole project, or appropriate reading authority for the exact resource.
Self-inspection of installation scope requires an active authenticated account.

The decision calls the same permission service as live requests.
Reasons distinguish an allow, an explicit deny, no matching allow, and a disabled account.
The report includes its check time. It does not promise unchanged future access.
Synthetic public, authenticated, and administrator roles remain disabled. Product access requires explicit assignments or grants.

Each source identifies a direct grant or an assigned role path.
Authorized role details include the principal, assigned root, inherited grant owner, configured scopes, project binding, and expiry.
Expired rules appear only when they otherwise match the context. They grant no access.
Field restrictions follow the library's permission-level evaluation.
Disabled roles and inactive groups contribute no active paths.

Source inspection uses isolated read-only library engines. It does not implement a second policy matcher.
Each role probe retains its actual principal, root assignment, binding, inherited grant owner, and native role state.
Deny probes use an in-memory baseline allow to verify actual deny precedence.
The probes write no assignments, native grants, role definitions, or audit changes.

Source details require reading authority over their principal, role, grant, and ownership scope.
Group sources also require reading authority over the stored group and membership context.
Exact object read denies remain authoritative.
Restricted sources expose only an opaque identity, source kind, effect, and matching or expired status.
They omit names, identifiers, scopes, bindings, expiry timestamps, and field restrictions.
Opaque source identities bind the inspector and context. They do not expose native grant identifiers.

Source scans are bounded. They can return an empty partial page before later matching sources.
Scan more sources continues the same context. The decision remains complete even while its source list is incomplete.
Continuation binds the inspector, subject, project, object, action, directory revision, group state, role paths, and current decision.
Directory changes or processed grant expiry invalidate continuation with HTTP 409.
Rejected continuations and failed refreshes clear private reports. Check access starts a new scan.

Run the browser workflow through normal publication and execution:

```sh
bash scripts/test-self.sh --test=STUDIO-PERMISSIONS-001
```


## Provider configuration binding

OIDC login transactions bind the exact stored provider revision.
Creating the transaction asserts that revision after discovery completes.
Callbacks check it before discovery, after discovery, and after token validation.
A changed configuration rejects the callback before account provisioning or access synchronization.
New-account creation also asserts the provider revision with the user, exact issuer-subject link, and audit records.
The rejected transaction cannot be reused. Start a new sign-in.

Validated provider group memberships record `providerRevision` with their existing source and expiry.
Synchronization asserts that provider revision in the membership and audit transaction.
An outdated callback cannot restore membership after a provider change.
Directory lookup rejects outdated, disabled, missing, or incorrectly scoped providers.
Local membership remains independent.
Membership screens and administrator guards use the same current-membership check.
Outdated memberships remain inspectable with inactive state.

New OIDC sessions record their authentication method and provider revision.
Session creation asserts the provider revision with the session and user sign-in update.
Browser requests and private evidence access require that revision to remain enabled and current.
Any provider edit invalidates earlier OIDC sessions and provider memberships, including a display-name edit.
Restoration does not reactivate earlier cookies or membership records. A validated sign-in refreshes their provenance.
Active session lists use the same session validator as requests.
Provider identifiers named local or password do not select those authentication methods.

An upgrade rejects older sessions that have no explicit authentication method.
Sign in again after this upgrade. New sessions retain their method across ordinary restarts.
Older OIDC memberships without a provider revision remain inactive until validated refresh.
Pending login transactions without a provider revision also require a new sign-in.

Focused loopback provider tests use actual discovery, signed tokens, state, nonce, and PKCE exchanges.
They verify changes during discovery and exchange without external provider accounts.
The [provider guide](providers.md) describes managed provider configuration.
Optional provider sign-out and focused OIDC acceptance are implemented below. Deployment-wide identity acceptance remains pending.

## Explicit identity linking

Open Your account and use Link another sign-in.
Choose an enabled provider and enter an audit reason.
The request belongs to the current browser session. It expires after five minutes.

Password sign-in requires the current password again. The durable sign-in limiter also applies to this verification.
OIDC sign-in requires fresh authentication through the current provider.
The returned issuer and subject must already identify the same internal user.
The loopback operator can confirm local control only from a loopback connection.
Internal service or fixture sessions cannot start linking.

Next, select Verify new identity. Authenticate through the target provider.
Both OIDC steps use code flow, PKCE, state, nonce, and normal token verification.
They request `max_age=0` and `prompt=login`.
The token must contain an integer `auth_time`.
Authentication must follow the authorization request, with one minute of clock tolerance and a five-minute maximum age.
The protocol requirements are in [OpenID Connect Core](https://openid.net/specs/openid-connect-core-1_0-errata2.html#AuthRequest).

Proof returns to Your account. It creates no identity link, user, membership, or role assignment.
Review the exact issuer and subject. Select Confirm identity link to add the sign-in method.
Email matching never links accounts. An existing identity keeps its owner, including when that owner is the current user.
This workflow cannot merge internal accounts or transfer an identity.
Provider group mappings apply during later normal sign-in. The linking proof never synchronizes groups.

Confirmation requires the current operation revision and the original active session.
One database transaction inserts the link, records the audit event, increments the user session epoch, and completes the operation.
Earlier sessions become invalid. This browser receives a new opaque session cookie and request-forgery token.
The original session expiry and authentication deadline remain unchanged.
The replacement session can read the completed operation after an unconfirmed response.

Provider edits, credential changes, disabled users, revoked sessions, and expiry invalidate pending operations.
Starting another proof supersedes its earlier browser transaction.
Consumed, superseded, cancelled, and cross-session callbacks cannot complete linking.
Cancel identity link discards pending proof.
Use Review current link state after conflicts or unconfirmed responses. Repeated mutations stay disabled until that review succeeds.
Password values clear after submission and never enter operation records or audit data.

### Identity link API

These operations require an active session. Mutations also require the configured Origin and current request-forgery token.

| Method | Route | Behavior |
| --- | --- | --- |
| GET | `/api/v1/me/identity-linking` | Read the source method and enabled target choices. |
| POST | `/api/v1/me/identity-links` | Start with `providerId`, `reason`, and source authentication. |
| GET | `/api/v1/me/identity-links/{operation}` | Read the operation owned by this exact session. |
| GET | `/auth/identity-links/{operation}/authorize` | Navigate to the required source or target provider. |
| POST | `/api/v1/me/identity-links/{operation}/confirm` | Confirm proven identity with positive `If-Match`. |
| POST | `/api/v1/me/identity-links/{operation}/cancel` | Cancel with positive `If-Match`. |

Source authentication uses `password` for a password session, or `confirmLocal: true` for the loopback operator.
OIDC source proof follows the authorize route.
The existing `/auth/callback/{provider}` route handles purpose-bound linking callbacks.
No HTTP operation accepts an arbitrary issuer and subject as a linking mutation.
JSON operation responses contain no session identifiers, protocol secrets, provider credentials, or passwords.
OpenAPI 0.10 and the generated identity client include the five JSON operations.


## Pending project access

Project owners can prepare access before a user signs in.
Open Project access and select + Prepare project access.
Choose Allowed sign-in provider, enter Required verified email, and select a permitted project role.
Enter Pending access reason and select Prepare access.
The operation creates no user, password, external identity, or invitation message.

The record stores the exact project, provider identifier, issuer, provider revision, email, role, owner, reason, and creation time.
Email matching preserves the exact mailbox before `@` and ignores domain letter case.
The allowed provider must confirm `email_verified: true` for that address during normal OIDC sign-in.
Internal profile email or verification fields cannot activate pending access.
Pending access cannot link identities, merge accounts, or grant installation administration.
Explicit identity-linking proof does not activate pending access.

The configured provider controls first-sign-in profile provisioning.
When automatic provisioning is disabled, sign-in requires an existing exact identity link.
An authenticated user can add that link through Your account.
Preparing project access does not override that configuration.

Activation checks the current provider revision and exact external identity owner.
It checks the project, the approving owner’s active account and current delegation authority, the role, and the actual recipient.
The approved role fingerprint includes inherited policy revisions and predefined role state revisions.
Provider or role changes block the old approval. Restoring a changed definition does not silently restore its approval.
An archived project or lost owner authority also blocks activation.

The permission library assigns the normal project-bound role through the same transaction used for direct assignments.
Assignment metadata, activation records, match-index removal, and audit events commit together.
A failed batch rolls back native roles and activation metadata.
Concurrent cancellation and sign-in cannot produce an activated role with a cancelled approval.
Existing exact assignments are retained without duplicate assignment events.

Successful activation consumes the pending match and records the permanent user and assignment identifiers.
Later role revocation remains effective after another sign-in. A consumed approval cannot restore it.
The activated record remains in history. Remove the resulting role through Role assignments.
Cancel pending access removes only a pending or blocked approval. It requires the current positive record revision and a reason.
A new approval creates a separate record. Cancelled and blocked history remains available.

A provider and email permit at most 100 pending project-role matches across the installation.
One project, provider, email, and role permit only one pending approval at a time.
The bound prevents silent truncation during sign-in.

### Pending access API and interface

All routes require an active session and the route’s exact project authority.
Mutations require the configured Origin and current request-forgery token.

| Method | Route | Behavior |
| --- | --- | --- |
| GET | `/api/v1/projects/{project}/pending-access/providers` | Read enabled provider choices without credentials or installation provider management. |
| GET, POST | `/api/v1/projects/{project}/pending-access` | Scan readable approvals or prepare project access. |
| GET | `/api/v1/projects/{project}/pending-access/{access}` | Read one approval and its retained state. |
| POST | `/api/v1/projects/{project}/pending-access/{access}/cancel` | Cancel pending or blocked access with positive `If-Match` and a reason. |

Creation accepts `providerId`, `providerRevision`, `email`, `roleId`, and `reason`.
Verified email is mandatory and cannot be disabled in the request.
Read authority requires project membership and permission inspection. Management requires membership management and delegation.
Exact `pending-access/{id}`, role, and activated-recipient denies remain authoritative.
Project access managers can read curated provider choices without installation directory or provider authority.
Choices contain names, issuers, revisions, and provisioning mode. They contain no client credentials or secret references.
Cursor values bind to the reader and exact project. Unreadable rows can leave an empty page with a continuation cursor.

The interface retains the provider revision captured when its editor opens.
Refresh pending access updates records and available choices without replacing that opened revision.
After a conflict or unconfirmed response, repeated submission stays disabled.
Use Review pending records and reload choices to review current records and explicitly adopt current provider choices.
Cancel stale approvals before replacing them.
The interface clears private edits after revoked access and supports read-only project inspection.
Contextual help covers preparation, verification, blocked states, cancellation, and ordinary role revocation.
OpenAPI 0.11 and the generated identity client include 73 JSON operations.

## OIDC verification and group refresh

Provider discovery and key requests use bounded HTTP clients. The maximum client timeout is ten seconds.
Authorization, token, and key endpoints require HTTPS outside loopback. Redirects cannot downgrade an HTTPS request.
A shorter operator-supplied timeout remains unchanged. Provider logout discovery has a five-second request deadline.
The SDK verifies the original signed token against discovered signing keys. New sign-ins fetch current keys and accept valid key rotation.
Retired or unknown signing keys fail verification.

TestStudio uses the SDK for signature, issuer, audience, expiry, and basic token validation.
It also checks nonce, issue time, authorized party, token use, and access-token hash when present.
An authorized-party claim must match the configured client, including for a single audience.
Multiple audiences require that claim. An explicit access-token type or use cannot identify a TestStudio browser user.
Registered claim names cannot use ambiguous case variants.
Provider ID tokens cannot replace the opaque session cookie on API requests.

The signature key-set adapter preserves the original verified claims for application validation.
It removes distributed-claim envelopes only from the payload used for SDK identity-claim parsing.
It never rewrites the signed input before signature verification.
The SDK validates the identity fields before TestStudio changes group membership.
This permits safe membership withdrawal when unsupported distributed group metadata would otherwise prevent identity parsing.
An invalid nonce or identity binding cannot withdraw another account’s membership.

Group mappings require a complete string array. An explicit empty array is complete and grants no mapped membership.
Missing, null, malformed, overflowing, distributed, or paged group claims fail the refresh and withdraw that provider’s earlier memberships.
External identifiers are bounded to 512 bytes. A complete claim permits at most 4096 values.
Unmapped identifiers grant nothing. Local membership remains independent.
Only explicit configured mappings can select destination groups. External role names never become executable permissions.
Expired membership loses project permissions on the next check, even while another valid session remains open.
The configured membership lifetime also bounds the OIDC session authentication deadline.

Unresolved distributed email or email-verification claims cannot activate pending project access.
Provider-confirmed inline email remains mandatory for that workflow.

Normal OIDC and local sign-in replace the earlier browser session identifier.
Other browser sessions remain independent unless account or credential changes revoke them.
Idle, absolute, and authentication deadlines apply to normal requests and private evidence authorization.
Authentication and API responses, including redirects, use `Cache-Control: no-store`.

Sign-in links preserve the current local page, query, and fragment.
Authentication server actions and encoded external-authority paths cannot become return routes.
Validated cancellation and failure responses preserve the approved destination and add a fixed sign-in status.
Messages never display raw provider diagnostics.

## Optional provider sign-out

The default Sign out action revokes the current TestStudio session and clears its cookie first.
It also clears any unfinished browser sign-in cookie.
Provider sign-out is a separate optional action. Provider errors cannot restore the closed TestStudio session.

Open a provider’s settings and copy Post-logout callback URL into its registered logout redirects.
Set Provider logout endpoint to the exact discovered `end_session_endpoint`.
Leave the field empty to disable the integration.
Endpoints require HTTPS outside loopback. Reserved authentication parameters cannot appear in the configured URL.

This integration uses the client-identifier mode from [OpenID Connect RP-Initiated Logout](https://openid.net/specs/openid-connect-rpinitiated-1_0.html).
It sends `client_id`, the registered `post_logout_redirect_uri`, and an opaque state.
It sends no ID-token hint, access token, refresh token, or client secret through the browser.
The provider must support this mode and validate the registered return destination.
The provider asks the user to confirm sign-out when no identity-token hint is supplied.
A provider that requires an identity-token hint can retain TestStudio-only sign-out.

After local sign-out, the interface offers Sign out of the configured provider.
The offer expires after five minutes and belongs to the initiating browser.
Its opaque URL and separate HttpOnly cookie bind the launch. The server keeps protocol state in the selected database.
Successful launch consumes the offer and requires current provider configuration and matching discovery metadata.
The callback checks provider, browser, state, expiry, and one-time transaction state.
It cannot clear or revoke a newer TestStudio session.
Completion, denial, and unavailable-provider outcomes return readable messages.
Provider configuration changes invalidate earlier offers.

| Method | Route | Behavior |
| --- | --- | --- |
| POST | `/auth/logout` | Revoke the local session and return an optional browser-bound provider offer. |
| GET | `/auth/logout/{offer}` | Launch the configured provider logout after browser and discovery checks. |
| GET | `/auth/logout/callback/{provider}` | Complete a matching logout transaction without changing newer sessions. |

The POST requires the current session, configured Origin, and request-forgery token.
Launch and callback occur after local revocation. They use their separate short-lived browser binding.
No provider credentials enter JSON logout responses, browser storage, or audit events.
OpenAPI 0.12 and the generated identity client include 74 JSON operations.

Deployment-wide identity parity and privilege-change session behavior beyond the verified credential and account operations remain in the full plan.

## Repository source objects

Repository policies use source/SOURCE_ID within a project scope.
The object resolves against the stored repository source before policy evaluation.
Source read denies also protect child preparations, retained reviews, and original archives.
Suite and catalog permissions remain required for each binding.
Background source work checks the current source owner and current grants before preparing or publishing.
