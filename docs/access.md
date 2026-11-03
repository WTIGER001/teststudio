# Identity administration

TestStudio stores user profiles, external identity links, groups, and membership provenance in its selected database.
The permission library evaluates access. The directory does not infer project authority from an email address or group name.

Users and Groups screens use the administrative APIs for profiles, account state, group metadata, local membership, and session revocation.
Managed password screens use the same directory, sessions, and permission checks.
Managed custom role APIs support scoped rules, inheritance, disabled state, and assignment.
Custom role screens use the same typed API, scope checks, and revision guards.
Identity linking and pending project access remain pending.
The full acceptance ledger remains in [Implementation verification](implementation.md).
The [OpenAPI contract](../contracts/openapi.yaml) describes identity, role policy, direct grants, permission catalog, and assignment routes. Other route families remain pending.

## Typed client contract

Generate identity request types, response types, operation methods, and route templates from the OpenAPI contract:

```sh
go run ./cmd/teststudio-client
go run ./cmd/teststudio-client --check
```

Run these commands from the repository root.
The generated module contains all 54 implemented identity and access operations, including direct grants, policy previews, and custom assignment choices.
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
Managed installation role and direct grant changes use the same protection.
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
Direct principal grant APIs use this transaction. Their editor workflows and complete effective-access explanations remain pending.

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
Direct principal grants, built-in role override administration, and complete effective-access explanations remain pending.

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
These comparisons do not replace the pending permission-source explanation workflow.

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

## Predefined role assignments

Create an assignment with `principalKind`, `principalId`, `roleId`, and a nonblank `reason`.
Use an existing user or group. Local and OIDC profiles use the same permanent internal user identifiers.
The API creates no credentials, identity links, or invitation messages.
The project route determines the exact `team` binding. Request bodies cannot supply bindings or widen the scope.
The installation route supports the installation administrator role.
Project routes support viewer, operator, editor, publisher, and project administrator roles.
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
Identity linking and personal profile editing remain pending.

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
Direct grant screens and complete source explanations remain pending.
