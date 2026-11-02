# TestStudio

TestStudio is an independent test catalog, execution, and evidence platform.
The implementation follows [the product plan](docs/plan.md).
The [acceptance ledger](docs/implementation.md) records verified work and pending requirements.

## Local development

Requirements: Go 1.26.2 or later and Node 22.22.3, Node 24.15, or Node 26 or later.

Build the Angular UI:

```sh
npm --prefix web ci
npm --prefix web run build
```

Start the local server:

```sh
go run ./cmd/teststudio --local
```

Open http://127.0.0.1:8040 and select Open local workspace.
The explicit local mode binds to loopback and provides local installation administrator access.
The server starts with an empty project registry.
Data stays under `.data/` by default.

Create a project, application, suite, and environment through the UI.
Add native HTTPS checks, save catalog changes, validate a draft, and publish a release.
Native HTTP runs use durable local workers and retained attempt records.
The API provides run creation, progress, cancellation, failure reruns, logs, and private evidence access.
The artifact listener uses `http://127.0.0.1:8041` by default.
Keep the artifact directory outside the public UI directory.
Use `--artifact-origin`, `--artifact-addr`, and `--artifact-directory` to change these settings.
Select Test runs to start a run, inspect progress, cancel execution, or rerun failed tests.
The detail page shows attempts, filtered logs, events, and private artifact previews or downloads.
Other plan areas remain tracked in the acceptance ledger.

## Authentication

Hosted installations configure OIDC providers and use server-managed browser sessions.
Use `--oidc-config` with a JSON array of provider definitions for initial registration.
Existing provider records remain unchanged when the server restarts.

Set the initial administrator through explicit issuer and subject identifiers:

```sh
go run ./cmd/teststudio --public-url https://tests.example.com --addr 127.0.0.1:8040 \
  --artifact-origin https://evidence.example.com --artifact-addr 127.0.0.1:8041 \
  --oidc-config config/oidc.example.json \
  --bootstrap-issuer https://identity.example.com --bootstrap-subject SUBJECT
```

Register the callback `/auth/callback/PROVIDER_ID` at the identity provider.
Provider secret references use `env:VARIABLE_NAME` for local configuration.
The browser receives no provider tokens or client secret.

## Databases

SQLite is the default for a single local service.
Use `--database postgres://USER:PASSWORD@HOST/DATABASE` for PostgreSQL.
Use a protected configuration mechanism for production database credentials.
The permission engine uses a separate `teststudio_permissions` schema in PostgreSQL.

## Validation

Run tests for the changed feature:

```sh
go test ./internal/storage -count=1
go test ./adapters/permissions/sqlite -count=1
go test ./internal/catalog -count=1
go test ./adapters/auth/oidc ./internal/sessions -count=1
go test ./internal/httpapi -count=1
```

Build the server with `go build -o bin/teststudio ./cmd/teststudio`.
The user authorizes frequent feature commits. Remote pushes require a separate request.
