# Git test source specification

Status: Implemented locally on 2026-11-10. Live provider and hosted deployment acceptance remain separate. See [repository sources](repository-sources.md) for the current workflow, configuration, and supported limits.

## Current behavior

The [repository integration](repository-sources.md) supports GitHub and operator-approved self-managed GitLab servers.
Connections select read-only database metadata or read/write Git metadata with review branches.
Playwright discovery can generate metadata and exact untagged mappings. New tests start disabled with Needs review safety.
New starts resolve their configured reference and reuse only matching immutable preparation. Changed source queues setup and blocks stale execution until ready and reviewed. Scheduled firings retain a blocked reason while setup is incomplete.
Explicit retained versions and reruns preserve their original pin. A manual reviewed version also guards against a silently changed catalog.
GitHub API token authentication and validated replacement, optional App installation token renewal and key reference replacement, GitLab mounted-file project token self-rotation, and durable credential health are implemented.
The Playwright runner supports TypeScript and JavaScript. Python, online dependency installation, submodules, and Git LFS preparation remain unsupported.

## Repository connections

A project manages repository connections. Each connection identifies its provider, approved server, repository identity, credential reference, and access mode.
Support GitHub and the user's self-managed GitLab instance. GitLab needs a configurable HTTPS server, nested repository namespaces, and approved private certificate authorities and network access.
Server approval belongs to the operator. User-entered repository addresses cannot grant new network access.
Connection credentials stay separate from application environment secrets.
Show access status, credential expiry when known, last successful validation and synchronization, and actionable failures.
Support credential replacement and provider-specific rotation without changing suite bindings. Preserve audit records and keep secret values server-side.
GitHub App installation token renewal remains automatic. Private key rollover needs a guided validation and replacement workflow.
GitLab rotation must account for the provider's permissions, version, immediate revocation, and interrupted-response recovery. Do not blindly retry a rotation request.

## Two access modes

| Mode | Authoritative metadata | Repository changes |
| --- | --- | --- |
| Read-only | TestStudio stores the suite manifest, catalog JSON, revision history, and permanent test mappings in its database. | Read source only. Never commit or push. |
| Read/write | The configured Git branch stores the manifest and catalog JSON. TestStudio retains indexed copies, pending edits, and immutable history. | Commit reviewed metadata changes to a review branch and open a pull or merge request. |

Both modes read executable test source from the repository. Metadata includes identity, descriptions, categories, safety, dependencies, required configuration names, timing, and capture policies.
Environment values, credentials, result history, and evidence remain outside Git metadata.
Existing repository metadata can seed read-only setup. Subsequent source synchronization preserves the database metadata and reports incompatible source changes.
Read/write synchronization adopts committed metadata changes and preserves pending local edits separately.

Read/write saves show a diff and expected base commit. Commit related metadata files atomically.
Recheck TestStudio authority and provider access before writing. Preserve remote edits, branch protection, and revision conflicts. Never force-push to resolve a conflict.
Retain local edits after a rejected or interrupted write and recover the provider outcome before retrying. An uncertain response must not create duplicate commits or review requests.
Show pending local changes and pending review clearly. A database save does not imply that a Git commit succeeded.
Normal runs use metadata committed to their selected source reference. Pending changes become runnable when their commit is explicitly selected or reaches that reference.
Saving metadata, committing, and merging never start tests. A run never commits pending edits automatically.

Switching modes requires explicit confirmation of the new metadata authority. Review both copies in the metadata page before switching. Retained database revisions preserve earlier edits.

## Onboard repositories without metadata

1. Connect the repository and select its reference, application, test directory, and framework project.
2. Inspect candidate configurations. Run discovery in an isolated worker without test bodies, application sessions, or repository credentials inside test code.
3. Generate the manifest, catalog, and permanent test identifiers. Suggest names and categories from discovered source.
4. Review application ownership, endpoints, parameters, secret requirements, dependencies, safety, and evidence defaults.
5. Save metadata in the database for read-only mode or commit it through the read/write workflow.

Each suite has one application owner. A repository can supply several application-owned suites.
Newly discovered tests are disabled and carry needs-review safety until reviewed. Do not infer that arbitrary source is safe or that all secrets and dependencies have been discovered.
Display missing setup separately from actual test outcomes. Repository registration, discovery, and saving never execute tests.

Read-only onboarding must work without editing source to add permanent tags.
Persist a framework mapping using project, source location, title path, and case identity where available. Line numbers alone are not permanent identities.
The discovery and execution adapters resolve each selected mapping to exactly one test. Tagged suites keep their existing exact-tag execution path.
Preserve reviewed IDs across unambiguous changes. Require review for ambiguous renames, moves, duplicate titles, and changed parameterized cases.
Read/write metadata commits do not implicitly rewrite executable tests. Adding permanent tags or editing test bodies requires a separate reviewed source change.

## Source acquisition when a run starts

For a new run following a branch or tag, resolve the configured reference against the provider during run-start preparation. Manual and scheduled starts use the same path.
Push notifications and periodic synchronization may prepare versions early. They do not replace the run-start freshness check.
Pin the resolved repository identity and commit for that run. Record when the reference was resolved.
If a manual review predates a changed source version, refresh its source and selection review before accepting execution. Saved schedules use their configured selection and current safety and configuration checks against the resolved version.

Fetch a source archive or check out that exact commit in a clean isolated workspace.
Clone and archive retrieval are equivalent acquisition choices when the required source is complete and its provenance is verified.
Support for submodules, large-file storage, additional dependencies, or unsupported runtime versions must be declared explicitly. Missing source or assets block preparation with a named reason.
Never execute from a shared mutable working directory or use pull inside the test process.
Source credentials belong to acquisition workers and never enter the test runtime.

Combine the source commit with the selected metadata revision. In read-only mode, retain the database metadata snapshot and digest. In read/write mode, read committed metadata from that exact source version.
Validate discovery, permanent mappings, selected tests and dependencies, application ownership, required environment configuration, safety, and current authority before dispatch.
Missing or ambiguous mappings block source preparation until reviewed. New discovered mappings remain disabled until reviewed. An explicit ID selection cannot silently expand to newly discovered tests.

Reuse a retained source snapshot and prepared runtime only when repository identity, commit, metadata digest, dependency inputs, runtime profile, and preparation settings match.
Reference resolution still happens for each new run following a branch or tag. A full clone or rebuild for every individual test is unnecessary.
Once pinned, every test and retry uses that run's exact source and metadata. A later branch update cannot change an accepted run.
Idempotent start retries and worker recovery preserve the original pin and do not create duplicate runs.

If current source cannot be resolved, fetched, validated, or prepared, block the latest-source run. Never silently fall back to an older prepared version.
Keep repository and preparation failures distinct from assertion failures. Cancellation closes owned preparation work and prevents later dispatch.
An explicitly selected retained version or rerun uses its original source pin, subject to current authority and environment checks. Show this choice distinctly from a new run following the current reference.

Run details retain provider and repository identity, requested reference, resolved commit, source digest, metadata revision and digest, prepared artifact digest, and resolution time.
Retained versions and earlier results remain immutable in both modes.

## Required verification

These scenarios extend acceptance 21–29, with permission and environment coverage under 12, 13, 20, and 38. Local evidence refines these numbers without closing broad live-provider or hosted deployment criteria.

- Onboard an untagged repository without writing source; preserve permanent IDs through a reviewed rename and reject an ambiguous mapping.
- Save and execute read-only metadata after restart while issuing no Git writes.
- Commit read/write metadata, adopt a remote edit, detect a changed base, and retain edits after a failed write.
- Recover an uncertain commit or rotation response without duplicate writes or unsafe retries.
- Resolve a changed branch at a new manual or scheduled start; reuse unchanged prepared inputs without skipping provider resolution.
- Keep in-progress tests, retries, recovered work, and explicit reruns pinned when the branch moves.
- Block latest-source execution on repository outage, revoked credentials, missing assets, discovery mismatch, unsafe selections, or missing configuration.
- Verify provider credential rotation and private GitLab trust without exposing credentials to browsers, logs, source bundles, or test runtimes.
- Preserve existing GitHub source behavior, explicit execution, conflict recovery, history, and application ownership.

Use focused provider and run-orchestration checks first. Verify changed UI through explicit normal self-test IDs and keep live provider and hosted deployment acceptance separate.
