# Headless execution controller

## Introduction

The server accepts --serve-role=controller to process queued test runs without the UI, user API, sign-in bootstrap, or worker exchange listener.
The controller uses the same database, authorization, run lifecycle, evidence contracts, and completion planning as the API process.
It exposes only GET /healthz and GET /livez on an explicit --addr.

The default combined local server keeps its existing behavior.
An API instance can select --execution-worker=false when a separate controller owns queued execution.
That option preserves the API's other configured background services.

## Responsibilities

| Process | Responsibility |
| --- | --- |
| API | User requests, accounts, project and catalog workflows, and its configured scheduling and notification workers. |
| Controller | Run claims, execution, execution recovery, result persistence, and completion delivery planning. |
| Discovery worker | Source discovery and discovery recovery through --discovery-only. |
| Preparation writer | Source assembly, registry publication, and assembly recovery. |
| Worker backend | Private worker input, reports, logs, and evidence transport. |
| Artifact service | Authorized report and artifact access. |

The controller starts no scheduler, notification sender, source preparation loop, repository synchronization, recording maintenance, or session maintenance.
Its completion hook retains notification delivery records for the configured notification worker.
--public-url supplies the UI origin used in those report links. It does not enable a controller UI listener.

## Local native HTTP execution

Use the same SQLite database and artifact directory as the API for a single-instance local installation:

```sh
bin/teststudio --serve-role=controller \
  --addr=127.0.0.1:8043 \
  --execution-location=local \
  --database=.data/teststudio.db \
  --artifact-directory=.data/artifacts \
  --public-url=http://127.0.0.1:8040
```

Add --execution-worker=false to the API's existing launch options when using this arrangement.
The local controller profile supports native HTTP checks.
Local Go and Playwright coded execution retains the existing combined local-server profile.
Use a hosted controller profile for isolated coded execution.
Project and exact-origin private certificate authorities use the existing --http-trust-bundles configuration.

## Hosted coded execution

The controller accepts one existing ECS or Kubernetes execution profile and its matching execution location.
Hosted profiles require PostgreSQL and the installation's shared artifact adapter.
The optional [--database-host binding](database-host-binding.md) validates the selected database destination and verified TLS before storage opens.
For Kubernetes, the role selection looks like this:

```sh
bin/teststudio --serve-role=controller \
  --addr=0.0.0.0:8043 \
  --execution-location=kubernetes \
  --kubernetes-workers-config=/private/execution-workers.json \
  --database-env=TESTSTUDIO_DATABASE --require-postgres \
  --artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION \
  --public-url=https://studio.example.test
```

For ECS, select --ecs-workers-config and execution-location=ecs instead.
In the service container, override the entrypoint with /app/teststudio.
The default image entrypoint supplies UI and artifact flags that the controller role rejects.
The role uses only execution registration. It registers no discovery inspector or recovery inspector.
Workers use the profile's existing private exchange origin, served by the installation's worker backend.
The controller starts no second exchange listener.
The optional [Kubernetes controller package](kubernetes-controllers.md) supplies its dedicated deployment, configuration, identity, probes, and replica settings.
Keep provider credentials, identity permissions, network controls, and durable storage aligned with the selected deployment profile.
The staged [Azure host contract](azure-sandbox-host.md) now binds this role and its reported identity attachment.
Native Azure sandbox command selection, controller network admission, effective authority, and deployment packaging remain unfinished.

## Startup and shutdown

The role requires an explicit numeric listen address.
Only controller address, location, database, artifact storage, UI origin, HTTPS trust, and one supported provider configuration are accepted.
Account, OIDC, bootstrap, UI, local coded runtime, source preparation, recording, scheduling, and worker transport flags reject before storage opens.
--execution-worker applies only to the API and combined service roles.

Readiness checks the database. Liveness remains independent of database availability.
The shared server lifecycle binds its listener before starting execution workers.
Cancellation closes the listener, cancels and joins run workers, then releases provider, artifact, and database resources.
It preserves the API's sign-in configuration.

## Local verification

Three new focused checks and eight affected checks pass with race detection.
The actual controller command consumes one queued HTTPS probe while the API execution worker is disabled.
The check uses SQLite, a private filesystem artifact store, an owned loopback TLS target, and project-scoped certificate trust.
It verifies one logical result, retained notification planning, unchanged sign-in configuration, child-process exit, and listener closure.
Affected checks retain worker backend health, direct TLS transport, startup, role parsing, provider separation, and server cleanup behavior.
Evidence is output/headless-controller/summary.json and local.log.
Owned processes, listeners, and clients close through cleanup.
No cloud service, browser, container, UI build, demo restart, or remote push is involved.

Related guides: [service image](service-image.md), [hosted discovery](hosted-discovery.md), and [sandbox runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
