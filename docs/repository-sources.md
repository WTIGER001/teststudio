# Repository sources

The Sources page registers one GitHub repository with one or more suite manifest paths.
Each binding selects a targetless suite and the offline Go build profile.
Application source, shared libraries, and several suites can share the same repository snapshot.

Registration stores settings. It does not fetch source, build images, or execute tests.
Manual sync resolves an explicit branch, tag, or commit to one immutable commit.
A separate worker fetches the archive once and queues each suite preparation.
Review and publication use the existing source workflow.

## Local setup

Enable registration on the API:

```sh
bin/teststudio --local --github-sources --secret-bindings .data/source-bindings.json
```

Build the trusted runtime as described in [Go source bundles](source-bundles.md).
Start the separate worker against the same database and local Docker daemon:

```sh
bin/teststudio-prepare --database .data/teststudio.db \
  --github-sources --secret-bindings .data/source-bindings.json \
  --go-runtime sha256:TRUSTED_RUNTIME_IMAGE \
  --execution-location local --build-directory .data/source-builds
```

Public repositories need no credential binding. Omit secret-bindings when none exist.
The API and worker must use the same source credentials and approved references.
The API never runs assembly commands. The worker never runs test bodies during preparation.
Omit go-runtime to recover recorded preparation resources without starting new builds or source fetches.

## Private access

An operator supplies a project-scoped source-repository secret reference through the shared secret provider.
The UI selects the reference. It never accepts a private key or token.
A GitHub App installation must grant access to the selected repositories and read access to contents.

For local operation, a binding file can contain:

```json
[
  {
    "id": "github-read",
    "name": "Application repository reader",
    "projectId": "PROJECT_ID",
    "purpose": "source-repository",
    "file": "/absolute/private/path/github-app.json",
    "network": "public"
  }
]
```

Keep the referenced file private to the operator account.
The secret value uses this JSON structure:

```json
{
  "clientId": "APP_CLIENT_ID",
  "installationId": 12345,
  "privateKeyPem": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
  "repositories": ["owner/repository"]
}
```

The repositories list is required. Each entry names an explicitly approved repository.
The adapter requests a short-lived installation token for only that repository and contents read access.
The adapter uses HTTPS and guarded network connections. It validates archive redirects before following them.
It does not forward the API authorization header to the archive download host.
Private keys, installation tokens, and signed archive URLs stay outside browser responses and preparation logs.
Build contexts receive source files and provenance. They receive no source credentials.

See the [GitHub App token documentation](https://docs.github.com/en/rest/apps/apps#create-an-installation-access-token-for-an-app) for installation permissions.

## Register and synchronize

Open Sources and select Register repository.
Enter a permanent source identifier, display name, and owner/repository.
Choose a branch, tag, or exact commit. The API represents named references as heads/name or tags/name.
Select public access or an approved private credential reference.
Add each suite and manifest path relative to the repository root.
Save the source, inspect it, and select Sync now.

The worker pins the repository identity and resolved commit before fetching source.
A renamed or replaced repository cannot silently change the pinned identity.
Each downloaded GitHub wrapper directory is removed from logical paths after archive validation.
The digest covers the exact original ZIP bytes. All bound suites share that retained snapshot.
Suite preparation keeps its manifest, commit, source digest, runtime, image, diagnostics, and discovered catalog.
Get original source ZIP returns the unchanged archive, including its wrapper directory.

Preparing source does not publish a catalog or start a run.
Inspect suite preparation, open its review, and publish with optional activation.
Alternatively, enable automatic publication in source settings.
Automatic publication checks current source, catalog, and suite authority before activation.
An older sync cannot activate after a newer sync request.
A publication conflict leaves the ready preparation available for review.

## Reuse and conflicts

A repeated request with the same idempotency key returns its original sync.
An unchanged commit and source configuration reuse existing preparations and source bytes.
Ready preparations refresh their review against the current catalog without repeating assembly or discovery.
Service-managed titles, labels, timing rules, safety settings, and other managed metadata remain preserved.
Ordinary catalog imports cannot take ownership of source-managed definitions.

Choose Prepare again even when the commit is unchanged to request fresh preparation explicitly.
Concurrent source edits require the current record revision. Stale writes return a conflict.
Settings changes increment the source generation. Old workers cannot publish against changed settings.
Every sync records its settings, owner, requester, commit, state, and child preparations.
Interrupted synchronization records a failed outcome and retained children without repeating builds or tests.

## Authority and retention

Background synchronization uses the account that last saves the source settings.
Both the requester and the source owner need current access to the source and its bound suites.
A source-scoped deny uses source/SOURCE_ID and protects details, preparations, reviews, and original source downloads.
The existing suite and catalog permissions still apply.
Disabled sources cannot start new synchronization. Current workers stop when their next authority check fails.
Published releases and retained source remain available after disabling a source.

Unpublished source expires after the staging period when no retained preparation references it.
Published snapshots remain retained for release provenance and history.
Failed synchronization and preparation leave the active release unchanged.
Tests start through the normal run API, UI, or schedule after publication.

## Signed push synchronization

Choose an approved Push webhook access reference and enable signed push synchronization in source settings.
The reference uses purpose source-webhook. Its secret value contains 32 through 4096 bytes.
Use a high-entropy random secret. Keep repository App credentials and webhook secrets in separate bindings.

Complete one manual sync to pin the repository identity.
Inspect the source and configure its Webhook address as a GitHub repository webhook URL.
Select JSON content and push events. Use the same signing secret in GitHub and the server-side binding.
Hosted endpoints use HTTPS. A local server can use periodic sync when GitHub cannot reach it.

The receiver authenticates the exact body bytes with HMAC-SHA256 before parsing JSON.
It compares signatures in constant time and checks the registered repository identity and watched reference.
It accepts no browser session or CSRF token as a substitute for the signature.
The body limit is 2 MiB. Larger deliveries need periodic or manual sync.
Unsupported events, unwatched references, and reference deletions create no work.

A repeated delivery identifier returns its original sync. A changed payload with the same identifier is rejected.
A signed push is a change notification. The worker resolves the current watched reference after queueing.
Delayed deliveries cannot force execution of their old payload commit.
Every queued sync records its push trigger, delivery identifier, and body digest without retaining credentials or the raw payload.

The provider endpoint has a separate [webhook contract](../contracts/source-webhooks.openapi.yaml).
See [GitHub signature validation](https://docs.github.com/en/webhooks/using-webhooks/validating-webhook-deliveries) for provider configuration.

## Periodic synchronization

Set Periodic sync interval to 60 through 86400 seconds. Set 0 to disable periodic synchronization.
The main service scheduler processes these timers when scheduler is enabled.
It queues ordinary source synchronization. It performs no source fetch, build, or test within the scheduler tick.
The separate preparation worker still needs github-sources and the approved Go runtime.

Timers persist in the metadata database. Competing replicas atomically advance one due timer and queue one sync.
Missed intervals coalesce into one request. An existing pending sync advances the timer without adding more work.
Current source ownership, permissions, settings generation, and enabled state apply at each firing.
A blocked timer displays its problem and advances to the next interval without creating a test result.
Disabling the source removes its timer. Changing settings schedules the next interval from the settings change.
The source details show the next due time. Sync details distinguish manual, push, and periodic triggers.

## Remaining source work

Playwright source assembly and online dependency preparation remain pending.
The current implementation supports manual, push, and periodic public or GitHub App synchronization with offline Go source assembly.
All verification for this milestone runs locally. No live GitHub or cloud deployment is required for these checks.
