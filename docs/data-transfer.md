# Installation backup and database transfer

Use teststudio-data to export a TestStudio installation and restore it to an empty SQLite or PostgreSQL database.
The tool preserves public identifiers, metadata revisions, timestamps, run snapshots, audit history, artifact references, and access settings.
The archive also preserves native grant identifiers and their sequence high-water mark.
Permission conversion rebuilds the destination's internal role and inheritance structures.

This operation moves TestStudio data. It does not import another application's database or tests.
Catalog JSON transfer remains a separate tester workflow. See [Catalog management](catalog.md).

## Build the operator tool

```sh
go build -o bin/teststudio-data ./cmd/teststudio-data
bin/teststudio-data --help
```

Use the same TestStudio version for export and import.
The current format is teststudio.installation/v1 with metadata version 1 and the pinned go-permissions/v0.0.31 adapter format.
Unsupported versions fail before installation data commits.

## Prepare a cutover

1. Stop new run submissions and pause external publication or execution triggers.
2. Wait for active runs, recordings, preparation jobs, and worker recovery to finish.
3. Stop the source API, schedulers, notification workers, and preparation workers.
4. Keep the destination service stopped. Prepare an empty destination database.
5. Export metadata and copy its corresponding retained storage while the source remains stopped.

Export refuses active run leases, execution recovery work, unfinished worker exchanges, preparation leases, and open browser recordings.
The tool cannot stop other service instances. The operator keeps both installations offline during cutover.
After import, start only the destination as the authoritative installation.
Keep the source available for rollback without serving requests from both copies.

## Export SQLite

```sh
bin/teststudio-data export \
  --database .data/teststudio.db \
  --archive .data/backups/installation.ndjson
```

The source must exist. Export does not create or initialize a missing source database.
SQLite opens read-only. PostgreSQL exports use a consistent read transaction.
The output publishes only after the complete archive is written and synced.
An existing archive is never overwritten. A failed export removes its owned temporary file.
The summary prints counts rather than record contents or credentials.

Archives contain private installation metadata, password hashes, sessions, server-side authentication state, and the private worker ticket signing key.
Export creates the archive with permissions 0600. Import requires a private regular file.
Store it with access restricted to the installation's backup operators.

## Import into PostgreSQL

Set TESTSTUDIO_TARGET_DATABASE in the operator's protected environment to the new PostgreSQL connection URL.
Then run:

```sh
bin/teststudio-data import \
  --database-env TESTSTUDIO_TARGET_DATABASE \
  --archive .data/backups/installation.ndjson
```

The destination account must initialize TestStudio's metadata tables and teststudio_permissions schema.
Do not start the service against that database before import.
The tool creates required schema objects and rejects existing installation data or native permission state.
It does not delete, replace, or merge a nonempty destination.

Import restores metadata and native permissions in one database transaction.
The final checksum and section counts must match before commit.
Corrupt, truncated, incompatible, or invalid archives leave installation data uncommitted.
Schema initialization can remain after a failed import. An empty target can accept a corrected retry.
No PostgreSQL extension is required by this transfer.

Native grants keep their original identifiers, including managed grant references.
The sequence advances beyond the exported high-water mark, including identifiers no longer present in current grants.
New grants cannot reuse those previous identifiers.
User and group memberships, custom roles, inheritance, scope bindings, explicit denies, and disabled built-in overrides remain effective.

## Restore SQLite or export PostgreSQL

A new local database can receive the same archive:

```sh
bin/teststudio-data import \
  --database .data/restored/teststudio.db \
  --archive .data/backups/installation.ndjson
```

The path can be new. An existing database must contain no installation data.
A PostgreSQL installation can also export through --database-env and restore to SQLite with the same format.
Internal permission bookkeeping differs by backend. Public metadata and native grant references remain stable.
The SQLite profile still supports one service instance and requires no database server.

## Preserve artifact and source storage

The archive contains relational metadata and references. It does not contain artifact file bodies.
Copy or retain the artifact store's exact project, run, artifact, and checksum keys before starting the destination.
Preserve its installation prefix when using an object store.
Select the matching artifact profile on the destination. See [Artifact storage](artifact-storage.md).

Also preserve source bundle files, retained source snapshots, and required execution images under their existing storage configuration.
The archive does not copy Docker images, mounted secret files, environment variables, or deployment settings.
Keep credential references resolvable and provide the intended local-account and OIDC runtime configuration.
Preserve configured HTTP trust bindings and mounted CA files. Their certificate fingerprints affect queued dispatch and comparable timing history.
A different public origin requires users to sign in at that origin.

After restore, inspect a retained run and download a pinned artifact.
Compare its recorded checksum with the copied bytes. Missing files retain an explicit missing-content state.
The original test outcome and immutable run snapshot remain unchanged.
Refresh existing pages after cutover. Backend-specific pagination cursors do not form part of the migration contract.

## Verification and format limits

The archive uses newline-delimited JSON with a versioned header, typed permission sections, raw metadata payload strings, counters, and a checksum footer.
Metadata payload bytes, revisions, and timestamps do not pass through a domain rewrite.
Each entry has a 64 MiB limit. The reader processes one entry at a time inside the restoration transaction.
Large installations need sufficient database transaction capacity and private archive disk space.

Focused local checks export a newly created installation with users, group-bound custom roles, inheritance, an exact deny, and a disabled built-in role.
They preserve a password hash, a live session, a published release, an actual completed HTTP run, a pin, and retained evidence bytes.
SQLite-to-PostgreSQL conversion and PostgreSQL-to-SQLite restoration pass against a disposable local PostgreSQL instance.
Checks compare raw metadata and verify effective access, native grant updates, sequence high-water marks, and artifact references.
Damaged archives roll back. Existing destinations and active work fail explicitly. No cloud deployment is tested.

## Shared database behavior

Acceptance 06 verifies the same storage and claim contracts on SQLite and local PostgreSQL.
This verification is separate from installation export and import under acceptance 08.

| Behavior | Verified contract |
| --- | --- |
| Revision conflicts | One concurrent writer wins. Stale writes and assertions fail without partial changes. |
| Transactions | Metadata, sequence values, pending reads, and rollback share the same atomic boundary. |
| Paging and reopen | Project-bound ascending and descending pages retain ordering. Cursor continuation survives closing and reopening storage. |
| Startup and identity | Concurrent migrations converge. Concurrent identity creation returns one durable installation identifier. |
| Worker checkpoints | Claim fencing, cleanup failure, retry state, and queue completion remain durable. |
| Schedule claims | Concurrent claimants create one logical run. Live claims remain fenced and expired claims recover after reopening. |

The same eleven targeted checks pass with race detection on both databases.
Evidence is output/storage-acceptance/summary.json and its database-specific logs.
The check removes its owned local PostgreSQL container. It needs no browser or runner-image build.
Existing identity, transfer, preparation, and evidence recovery checks remain separate retained evidence.
SQLite supports one local service instance. Use PostgreSQL for hosted replicas.
Cloud deployment and provider-specific recovery verification remain separate acceptance work.
