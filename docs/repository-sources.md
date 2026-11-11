# Repository sources

## Introduction

Connect GitHub or an operator-approved self-managed GitLab server from Import and sync tests.
A repository can supply several suites. Each suite belongs to one application.
Use existing [suite declarations](test-repositories.md) or discover existing Playwright tests without metadata or permanent tags.
The [Git source specification](git-test-sources.md) records the ownership and execution rules.

Connecting a repository saves settings. It does not fetch, build, or run tests.
Inspect repository files reads a pinned archive without evaluating test code.
Sync now queues isolated preparation. Successful automatic setup makes a saved version available without starting tests.

## Concepts

### Metadata ownership

| Mode | Authoritative metadata | Saving changes |
| --- | --- | --- |
| Read-only | Manifest and catalog JSON in TestStudio's database. | Save metadata, then synchronize. The connection never commits or pushes. |
| Read/write | Manifest and catalog files on the configured Git branch. | Save pending edits, review the diff, and commit a review branch and pull or merge request. |

Both modes read executable source from Git.
Read/write mode requires a branch and an approved write credential.
Source workers request read access. Write access is used only for the explicit metadata commit.
GitHub App credentials must approve contents and pull request writes for this action.
GitLab credentials need permission to create a branch, commit metadata, and open a merge request.

Pending read/write edits do not change committed test metadata.
Merge their review request before running the configured branch, or explicitly configure their review branch.
A remote synchronization retains differing pending edits.
Review latest Git metadata shows committed data separately. Use reviewed Git base updates the pending base after explicit review; it does not merge or overwrite the local JSON.
A changed base blocks a new write. No workflow force-pushes or merges automatically.

Catalog Edit opens repository metadata for repository-owned tests.
The repository page provides a test review form and an Advanced metadata JSON section.
Git-owned catalog edits and lifecycle changes must use this workflow. Legacy catalog snapshots cannot bypass ownership.
Read-only synchronization preserves service-managed catalog metadata.
Switching ownership requires a review and confirmation. Earlier metadata snapshots and runs remain retained.

Environment values, secret payloads, execution history, and evidence stay outside Git metadata.
Metadata snapshots retain reference names and requirements, not resolved secrets.

### Existing repositories without metadata

Choose Discover tests without metadata for a Playwright binding.
Enter its source folder and configuration path. Leave the project blank to discover all projects, or select one exact project name.
Inspect repository files locates candidate configurations. Use this configuration opens the saved connection editor with a detected folder and configuration.
Set a named environment endpoint for environment-backed tests. Leave it blank for targetless tests.

Discovery lists tests in an isolated worker. It does not run test bodies, setup hooks, or application assertions.
Configuration and module imports still execute inside that isolation. They must not require an application session or fetch secrets.
The runtime receives no repository credential.

Discovery generates permanent IDs and exact project, file, and title-path mappings.
A source tag is optional for discovered tests. Existing declared tags retain their original IDs.
Newly discovered tests start disabled with Needs review safety.
Review setup, helpers, cleanup, side effects, dependencies, required parameter and secret names, and capture policy before enabling them.
The application configuration contract and each selected environment supply actual values.
A declared browser secret does not add a browser login credential bridge.

A line change preserves identity. A rename, move, removal, or ambiguous case mapping requires explicit metadata review before preparation can complete.
Correct the mapping while preserving the intended permanent ID. Do not assign old history to an ambiguous new test.

### Source freshness and reproducibility

A new run following a repository reference resolves that reference during run creation.
Manual and scheduled starts use the same check. Timing estimates and schedule saves do not contact the source provider or queue synchronization.
Webhooks and periodic synchronization can prepare versions early. They do not replace the start-time check.

A changed commit or database metadata revision queues setup and blocks the start.
Wait for setup, then refresh manual run review. Failed setup, unavailable workers, repository outages, and revoked credentials cannot silently select an older image.
Scheduled firings retain a blocked reason when the source is not ready; synchronization itself starts no tests.
An explicit retained version or failure rerun keeps its original source pin and remains subject to current authority, lifecycle, safety, and environment requirements.

Fetch uses an exact commit archive. A matching immutable source snapshot and prepared runtime can be reused without another full download or build.
Tests and retries never pull a moving branch or execute from a shared mutable checkout.
Run details show the requested reference, commit, resolution time, original source digest, prepared source digest, metadata digest, and artifact digest.
Database metadata revisions are recorded separately from the Git commit.
Source overlays retain the original ZIP unchanged and give the combined preparation its own digest.

### Conflicts, retries, and retention

Metadata saves use the current metadata and catalog revisions. A conflict retains local edits.
Reload metadata asks before replacing unsaved work.
Metadata revision history stores immutable snapshots. Load retained metadata for review creates local edits; saving creates a new revision.
Exact test read restrictions also apply to metadata and its history.

Each Git commit operation keeps a stable operation identifier, base commit, and review branch.
An uncertain response retains pending edits. Reconcile Git commit inspects that operation's branch and review request before attempting further work.
Changed branch content is a conflict. Confirmed operations return their stored result without repeating writes.

Unchanged source settings and commits reuse preparation. Prepare again even when the commit is unchanged explicitly requests another preparation.
Failed synchronization preserves earlier working versions and history.
Disabling a connection stops new synchronization. Explicit retained versions remain separate choices.

### Provider events

Push webhooks use a separate project-scoped source-webhook secret reference.
Complete one sync to pin repository identity before configuring the webhook.
The displayed address includes the provider key. Select push events and the same secret at the provider.

GitHub authenticates exact body bytes with HMAC-SHA256.
GitLab compares X-Gitlab-Token in constant time over HTTPS and uses X-Gitlab-Event-UUID for delivery identity.
The endpoint rejects mismatched provider keys, changed repository identities, and replayed identifiers with changed payloads.
Deletions, unsupported events, and unwatched references create no work.
A push is a notification. The worker resolves the current watched reference; an old delivery cannot force its payload commit into a run.
The body limit is 2 MiB. See the [webhook contract](../contracts/source-webhooks.openapi.yaml).

Periodic synchronization accepts 60 through 86400 seconds. Use 0 to disable it.
Timers persist, coalesce missed intervals, and avoid duplicating pending work.
Current source ownership, settings generation, suite permissions, and enabled state apply at each firing.

## Tutorials

### Connect and review tests

1. Open Import and sync tests and select Register repository.
2. Choose an enabled provider, metadata ownership, repository path, reference, and approved credential.
3. Choose the destination suites. Use discovery for existing Playwright configurations or select existing manifest paths and runners.
4. Review ownership, bindings, enabled state, and optional push or periodic settings. Save repository source returns to its details.
5. Inspect repository files when configuration paths are unknown. Edit the saved binding if necessary.
6. Select Sync now. Inspect its preparation and named diagnostics.
7. Open Review metadata for the suite. Review each imported test, keep its changes, and review before Save metadata.
8. Synchronize read-only metadata, or commit read/write metadata and merge its review request before synchronizing the configured branch.
9. Start tests explicitly through the normal run workflow.

Help preserves edits. Creation and metadata editing use dedicated routes with unsaved-change protection.

### Configure providers and workers

Enable GitHub with github-sources on the API and preparation worker.
For self-managed GitLab, provide the same operator-owned git-sources-config file to both processes:

```json
[
  {
    "id": "gitlab-company",
    "url": "https://gitlab.example.test",
    "network": "private",
    "caFile": "/absolute/private/path/gitlab-ca.pem"
  }
]
```

The URL is an exact HTTPS origin without credentials, a path prefix, query, or fragment.
Use public or private network scope. The CA file is optional when system trust is sufficient.
Configuration cannot replace the reserved github provider key or introduce duplicate keys.
Users choose approved provider keys; a repository address cannot grant new network access.

```sh
bin/teststudio --local --github-sources \
  --git-sources-config .data/git-servers.json \
  --secret-bindings .data/source-bindings.json --source-bundles

bin/teststudio-prepare --database .data/teststudio.db \
  --github-sources --git-sources-config .data/git-servers.json \
  --secret-bindings .data/source-bindings.json \
  --playwright-runtime sha256:TRUSTED_PLAYWRIGHT_RUNTIME \
  --go-runtime sha256:TRUSTED_GO_RUNTIME \
  --execution-location local --build-directory .data/source-builds
```

Use a matching API execution profile and trusted gateway for environment-backed tests.
The API queues source work. It does not build source or run discovery in its request process.
See [source bundles](source-bundles.md) for runtimes and dependency preparation.

### Private access and rotation

An operator registers a project-scoped source-repository binding through the shared secret provider.
Use a private file or server environment variable. The UI selects the reference and never accepts or returns its secret payload.
Each credential requires an explicit approved repository list.
GitHub entries use owner/repository. GitLab entries support group/subgroup/repository.

A GitHub App secret uses clientId, installationId, privateKeyPem, and repositories.
Set write to true only when the operator approves metadata commits and pull requests.
Read operations still request a fresh repository-limited installation token with contents read access.

A GitLab secret uses this shape:

```json
{
  "token": "OPERATOR_SUPPLIED_PROJECT_TOKEN",
  "tokenType": "project",
  "repositories": ["group/subgroup/repository"],
  "write": false,
  "expiresAt": "2027-04-01T00:00:00Z"
}
```

Expiry is optional when not supplied. Write defaults to false.
Project, group, or personal tokens can read approved repositories with suitable provider permissions.
The native rotation action is offered only for declared project tokens in replaceable mounted-file bindings.
It requires an instance and token scope that support self-rotation. Environment-variable and read-only mounted bindings use operator replacement.

Validate repository access records the attempt, last successful validation, current configuration match, known expiry, and GitHub public-key fingerprint.
Credential warnings identify expiry within fourteen days. This status is evidence of the recorded check, not a guarantee that later access will succeed.

For GitHub key rollover, generate and register a replacement App key as another approved reference.
Select it under Replace repository credential. Validate and save replacement checks access before switching the connection.
Revoke the old key at GitHub after all consumers have switched. Installation tokens renew automatically; App key generation and revocation remain provider administration.

Rotate GitLab project token invalidates the old token immediately.
The service durably records the attempt, stores the replacement atomically through the mounted-file provider, and retains the same reference.
An interrupted or failed rotation is not blindly retried. It requires validation and operator replacement or recovery because the provider may already have revoked the old token.
No secret values enter browser responses, logs, test runtimes, source bundles, or Git commits.

See [GitHub App keys](https://docs.github.com/en/apps/creating-github-apps/authenticating-with-a-github-app/managing-private-keys-for-github-apps), [GitLab project token rotation](https://docs.gitlab.com/api/project_access_tokens/#rotate-a-project-access-token), and [GitLab commits](https://docs.gitlab.com/api/commits/).

## Supported scope and target verification

The current runtime supports offline Go and Playwright TypeScript or JavaScript preparation with pinned dependencies.
Python, arbitrary dependency installation, Git submodules, and Git LFS asset preparation are not implemented.
Source with missing prerequisites stops with explicit diagnostics. Registration or successful discovery does not prove application assertions pass.

Provider, storage, and execution checks use local fixtures and local runtimes.
Live acceptance against the user's GitLab server, actual GitHub App permissions and rollover, and hosted worker installation remains separate.
MMDev's deployed core still lacks coded preparation and execution workers. Do not describe saved browser tests as runnable there based on these local checks.
