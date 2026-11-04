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

This role implements the backend process. It does not establish provider routing, TLS termination, placement, or installation authority.
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
Hosted instances require PostgreSQL and the installation's shared private artifact adapter:

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

Other explicit flags reject startup before storage opens. This includes scheduler=false and notification-worker=false.
Those services never start in this role, so their flags are unnecessary.
UI, artifact listener, public origin, provider, recording, and identity flags also reject startup.
Positional arguments reject startup. Role validation errors omit option values.

The existing service image entrypoint supplies UI and artifact listener flags.
For this role, override that entrypoint with /app/teststudio and provide only the options above.
Build a new service image containing this change before using the role. Existing cached images do not acquire new code automatically.
Override the default image health probe to use the worker port.

## Private ingress and probes

The process listener uses HTTP behind trusted private HTTPS termination.
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
