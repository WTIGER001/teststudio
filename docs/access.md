# Identity administration

TestStudio stores user profiles, external identity links, groups, and membership provenance in its selected database.
The permission library evaluates access. The directory does not infer project authority from an email address or group name.

Users and Groups screens use the administrative APIs for profiles, account state, group metadata, local membership, and session revocation.
Managed local credentials, identity linking, custom role management, assignment editors, pending project access, and the personal profile screen remain pending.
The full acceptance ledger remains in [Implementation verification](implementation.md).
The [OpenAPI contract](../contracts/openapi.yaml) describes the implemented identity and predefined assignment routes. Other route families remain pending.

## Typed client contract

Generate identity request types, response types, operation methods, and route templates from the OpenAPI contract:

```sh
go run ./cmd/teststudio-client
go run ./cmd/teststudio-client --check
```

Run these commands from the repository root.
The generated module contains all 30 implemented identity and access operations, including direct membership reads and scoped predefined assignments.
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

Open Users from the installation navigation to register profiles and inspect user details.
User details show account state, email verification, linked identities, readable memberships, and active sessions.
Use Edit profile to change permitted fields. Account actions use separate reason forms.
Session revocation identifies the current browser before confirmation.

Open Installation groups for shared groups owned by the installation.
Select a workspace and open Project groups for that project's directory.
Installation navigation uses installation capabilities. Project navigation uses the selected project's capabilities.
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
The check evaluates the proposed directory state through the same permission library and built-in definitions used for requests.
Explicit denies and active membership remain authoritative. Alternate group paths can preserve access.
Unlinked accounts and accounts with disabled sign-in providers do not count as usable recovery accounts.
The configured loopback administrator provides an independent local sign-in path.

A shared directory revision prevents competing administrative mutations from passing stale checks and removing both administrators.
Directory provisioning, membership synchronization, profile changes, and local recovery also advance that revision.
Future role, grant, and provider administration must join this coordination before those write APIs become available.
Provider outages and external membership expiration can still require operator recovery.

## Operator recovery

Stop the local service before recovery. Use the same database and private artifact directory:

```sh
go run ./cmd/teststudio --local --recover-local-admin \
  --database .data/teststudio.db --artifact-directory .data/artifacts
```

The command requires explicit local mode and a loopback listen address.
It enables the dedicated local administrator, restores its administrator assignment, advances its session generation, and records the recovery action.
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
| GET, POST | `/api/v1/projects/{project}/role-assignments` | Scan or create predefined project assignments. |
| GET, DELETE | `/api/v1/projects/{project}/role-assignments/{kind}/{principal}/{role}` | Read or revoke one exact project assignment. |
| GET, POST | `/api/v1/role-assignments` | Scan or create predefined installation assignments. |
| GET, DELETE | `/api/v1/role-assignments/{kind}/{principal}/{role}` | Read or revoke one exact installation assignment. |


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
The assignment API uses this transaction. The assignment interface and custom policy transactions remain pending.

## Predefined role assignments

Create an assignment with `principalKind`, `principalId`, `roleId`, and a nonblank `reason`.
Use an existing user or group. Local and OIDC profiles use the same permanent internal user identifiers.
The API creates no credentials, identity links, or invitation messages.
The project route determines the exact `team` binding. Request bodies cannot supply bindings or widen the scope.
The installation route supports the installation administrator role.
Project routes support viewer, operator, editor, publisher, and project administrator roles.
Custom roles and object bindings remain pending in this API slice.

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
