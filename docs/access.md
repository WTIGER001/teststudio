# Identity administration

TestStudio stores user profiles, external identity links, groups, and membership provenance in its selected database.
The permission library evaluates access. The directory does not infer project authority from an email address or group name.

This checkpoint provides administrative APIs. Users and Groups screens, invitations, identity linking, custom role management, and assignment editors remain pending.
The full acceptance ledger remains in [Implementation verification](implementation.md).
The [OpenAPI contract](../contracts/openapi.yaml) describes the implemented identity routes. Other route families remain pending in that contract.

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

Ordinary account disabling, group archival, and membership removal preserve the last usable installation administrator.
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
It enables the dedicated local administrator, advances its session generation, and records the recovery action.
Select Open local workspace to create a fresh session.
After recovery, stop this process and return to the installation's normal startup command.
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
| PUT, DELETE | `/api/v1/groups/{group}/members/{user}` | Save or remove local membership with a revision. |

## Local verification

Run the focused API and concurrency checks:

```sh
go test -race ./internal/access ./internal/httpapi ./internal/sessions \
  ./internal/authorization ./adapters/auth/oidc ./cmd/teststudio \
  -run 'TestAccess|TestSession|TestPermission|TestOIDC|TestIncompleteGroups|TestLocalRecovery' -count=1
```

Run the same checks on a disposable local PostgreSQL service:

```sh
bash scripts/test-postgres.sh ./internal/access ./internal/httpapi ./internal/sessions \
  ./internal/authorization ./adapters/auth/oidc ./cmd/teststudio \
  -run 'TestAccess|TestSession|TestPermission|TestOIDC|TestIncompleteGroups|TestLocalRecovery'
```

Run the built service with a fresh SQLite installation and actual restart and recovery operations:

```sh
bash scripts/test-access.sh
```

The smoke script also continues an encrypted history cursor after restart.
It uses no cloud credentials. It stops its owned controllers and retains an isolated database and redacted summary.
