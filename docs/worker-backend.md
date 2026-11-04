# Trusted worker backend

## Service boundary

The worker service role exposes the four [private worker exchange](worker-exchange.md) operations on one listener.
It also provides local readiness and liveness probes.
It starts no UI, user API, sign-in flow, artifact viewer, scheduler, notification worker, preparation worker, or test executor.
It creates no users or external providers and preserves the stored password sign-in mode.

The [standalone relay](worker-relay.md) forwards worker requests to this trusted backend through a fixed HTTPS origin.
The backend holds database and private artifact access. The relay holds neither.
The backend checks the existing attempt ticket and current execution or discovery authority for every operation.
Disabled users, cancellation, expired leases, changed ownership, and archived resources retain their existing rejection behavior.
The preparation execution location selects the discovery lease registry.

This role implements the backend process and optional direct TLS. It does not establish provider routing, placement, or installation authority.
The [staged Azure backend package](azure-worker-backend.md) supplies a private environment, worker-only app, and scoped identities.
It uses platform HTTPS termination rather than direct TLS. Provider admission and deployed routing remain unfinished.
Azure coded execution remains disabled until complete admission and provider wiring pass.

## Startup

A local process can use SQLite and filesystem artifacts:

```sh
go run ./cmd/teststudio \
  --serve-role=worker \
  --worker-addr=127.0.0.1:8042 \
  --execution-location=local \
  --database=.data/teststudio.db \
  --artifact-directory=.data/artifacts
```

Use the same installation database and artifact settings as the dispatching controller.
Do not start an independent empty database and expect existing tickets to work.
Hosted instances require PostgreSQL and the installation's shared private artifact adapter.
This example uses a separate private TLS ingress:

```sh
/app/teststudio \
  --serve-role=worker \
  --worker-addr=0.0.0.0:8042 \
  --execution-location=azure \
  --database-env=TESTSTUDIO_DATABASE \
  --require-postgres \
  --artifact-store-env=TESTSTUDIO_ARTIFACTS
```

Database connection values and artifact settings remain in private configuration.
Workers and the relay receive no installation database, artifact, OIDC, registry, or target credential.
The backend does not load target secret bindings, HTTP trust bindings, OIDC configuration, execution providers, or source builders.

The role accepts only these explicit options:

| Option | Purpose |
| --- | --- |
| serve-role | Select worker. |
| worker-addr | Required listener address with a numeric port. |
| execution-location | Preparation location identifier. The default is local. |
| database or database-env | Existing installation database. Select one source. |
| require-postgres | Reject a SQLite fallback for hosted installations. |
| artifact-directory | Private filesystem artifact root for local operation. |
| artifact-store-config or artifact-store-env | Existing private artifact adapter settings. |
| worker-origin | Exact HTTPS origin for optional direct TLS. |
| worker-tls-certificate | Mounted PEM certificate chain for direct TLS. |
| worker-tls-key | Mounted PEM private key for direct TLS. |

Other explicit flags reject startup before storage opens. This includes scheduler=false and notification-worker=false.
Those services never start in this role, so their flags are unnecessary.
UI, artifact listener, public origin, provider, recording, and identity flags also reject startup.
Positional arguments reject startup. Role validation errors omit option values.

The existing service image entrypoint supplies UI and artifact listener flags.
For this role, override that entrypoint with /app/teststudio and provide only the options above.
Build a new service image containing this change before using the role. Existing cached images do not acquire new code automatically.
Override the default image health probe to use the worker port.

## Private ingress and probes

The process listener uses HTTP behind trusted private HTTPS termination unless direct TLS is configured.
Permit only the standalone relay and approved installation probes to reach this listener.
Do not expose it through the UI or artifact origin.
The relay requires normal backend certificate verification and a fixed HTTPS backend origin.
Deployment setup must prove the backend route, certificate authority, placement, and network access restrictions separately.

GET /livez checks process availability.
GET /healthz checks database connectivity with a two-second bound.
Both use the standard teststudio health response and no-store caching policy.
Readiness does not prove artifact availability, relay reachability, TLS validity, or complete provider admission.
The existing teststudio-health command accepts an explicit worker health URL.

The service uses bounded headers and read, write, idle, and shutdown timeouts.
SIGINT or SIGTERM closes the owned listener through the shared graceful shutdown path.
It exposes no HTTP operation for preparing, activating, collecting, or revoking exchanges.
Trusted controllers retain those operations.

## Direct TLS

The worker role can terminate HTTPS itself instead of using a separate TLS proxy.
Set all three TLS options together:

```sh
/app/teststudio \
  --serve-role=worker \
  --worker-addr=0.0.0.0:8042 \
  --worker-origin=https://controller-workers.internal.example.test:8042 \
  --worker-tls-certificate=/private/tls/chain.pem \
  --worker-tls-key=/private/tls/key.pem \
  --execution-location=azure \
  --database-env=TESTSTUDIO_DATABASE \
  --require-postgres \
  --artifact-store-env=TESTSTUDIO_ARTIFACTS
```

Mount the certificate chain and key read-only on the trusted backend. Keep the private key inaccessible to test workers and the relay.
The relay backend-origin must equal the selected worker-origin.
The certificate must cover that hostname, match the key, remain current, and permit server authentication.
Validation runs before database or artifact initialization. Incomplete, malformed, mismatched, expired, and future-dated leaf material rejects startup with fixed errors.
The backend loads its certificate once. Restart it to rotate the certificate before expiry.

Direct TLS requires TLS 1.2 or later. Plain HTTP receives no worker operation.
The TLS server name must match the configured hostname. A literal IP origin permits clients that omit SNI.
Each HTTP request must retain the exact configured Host authority, including an explicit configured port.
A different Host returns a fixed 404 before health or worker handling. A different SNI rejects the TLS handshake.
Do not add a health probe that changes the Host or disables certificate verification.
Use the selected HTTPS origin for teststudio-health, with working private DNS and normal certificate trust.

The relay still verifies the normal certificate chain. Startup validation of the leaf does not prove the relay trusts its issuer or complete chain.
Use an issuer trusted by the approved relay image. Locally injected test roots do not alter the production relay trust configuration.
Direct TLS does not replace attempt tickets or network access restrictions.
Provider admission must still bind the backend destination, private routing, certificate authority, and placement.
These TLS options require the worker role. The existing API worker listener continues to use external TLS termination.

## Targeted local verification

Six new focused race checks cover role selection, option rejection, live execution authority, discovery authority, health behavior, and actual command startup.
The command check preserves stored sign-in state and confirms graceful process exit and listener closure.
Three existing artifact-role and listener lifecycle checks pass after storage initialization becomes shared.
No browser, image build, deployment, registry push, or cloud request runs.

```sh
go test -race ./cmd/teststudio ./internal/runtimeconfig -run '^(TestWorkerBackend|TestServiceRoles)' -count=1
go test -race ./cmd/teststudio -run '^(TestArtifactRolePreservesPasswordModeAndRechecksExistingGrants|TestListenerFailureDoesNotStartWorkersAndReleasesEarlierPorts|TestServiceCancellationStopsItsOwnedListenersAndWorkers)$' -count=1
```

Evidence is output/worker-backend/summary.json and the targeted command results.
Acceptance 51 advances. The count remains 48 verified of 57, with nine open.

### Direct TLS delivery

Two new focused local race checks cover certificate configuration and the real HTTPS backend command through the standalone relay.
The end-to-end check carries input, start, report, and attachment operations into shared durable storage.
It rejects a foreign Host, foreign SNI that the certificate also covers, untrusted roots, legacy TLS, plaintext HTTP, and revoked write authority.
Three existing command and listener lifecycle checks also pass. Owned subprocesses and listeners exit through cleanup.
Only changed certificate and end-to-end checks repeat after review strengthens purpose, SNI, and plaintext coverage.
No UI, browser, image build, cloud request, registry push, or deployment runs.

```sh
go test -race ./cmd/teststudio -run '^(TestDirectWorkerTLS|TestWorkerBackendStartupPreservesSignInStateAndStopsItsListener|TestListenerFailureDoesNotStartWorkersAndReleasesEarlierPorts|TestServiceCancellationStopsItsOwnedListenersAndWorkers)' -count=1
```

Evidence is output/worker-backend-tls/summary.json. Acceptance 51 advances; the count remains 48 of 57.
