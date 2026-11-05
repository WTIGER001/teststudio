# Relay backend readiness and address binding

## Runtime contract

The [standalone relay](worker-relay.md) now exposes GET /readyz for backend readiness.
The request runs inside the relay process, so it uses the relay network and its normal certificate trust.
It opens HTTPS to the configured backend and requests only /healthz without a ticket, cookie, user identity, or forwarded caller header.
The backend must report status up, service teststudio, and role worker.
The [dedicated worker backend](worker-backend.md) now includes that role in its successful health and liveness replies.

GET /healthz and GET /livez continue to report local relay process health.
A backend outage changes readiness without changing process liveness.
Readiness sends no test operation, creates no run, and authorizes no worker attempt.
Existing private transport still checks its own ticket, ownership, permission, lease, and deadline for each operation.

The readiness client requires normal hostname and chain verification with TLS 1.2 or later.
The peer certificate must remain valid for more than two hours and five minutes.
That horizon matches the current maximum work window and startup allowance used by service admission.
Each probe uses HTTP/1.1 on a new TLS connection with keep-alive and multiplexing disabled.
Certificate rotation cannot remain hidden behind a reused readiness connection.

The response must have status 200, JSON content type, and a body of at most 1 KiB.
Only the three health fields are accepted. Missing fields, duplicate keys, unknown content, wrong role, redirects, and errors return fixed 503 replies.
Backend bodies, certificate errors, addresses, and redirect destinations do not become response details.
Caller credentials, query strings, request bodies, and noncanonical probe paths do not enter the backend probe.

Readiness has a three-second deadline, a dedicated connection, and one in-flight probe slot.
Worker transport retains its separate eight-request limit and normal connection pool.
Saturated readiness returns 503 without another backend call. Request cancellation reaches the backend request.
Probe responses retain no-store headers. No readiness result is cached or persisted.

## Private address selection

The command accepts an optional backend-address:

```sh
go run ./cmd/teststudio-exchange \
  --addr=127.0.0.1:8043 \
  --public-origin=https://workers.studio.example.test \
  --backend-origin=https://controller-workers.internal.example.test \
  --backend-address=10.40.2.4
```

The address must be a canonical private IPv4 literal without a port.
Loopback, public addresses, mapped IPv6, hostnames, and alternate address spellings reject configuration.
The HTTPS origin still supplies the HTTP Host, TLS server name, and selected port. The default port is 443.
Both readiness and worker transport dial only the selected private address while preserving normal certificate verification.
The transport disables environment proxies and redirects and rejects any dial target outside its configured backend authority.

Pinned mode uses the selected endpoint directly and performs no backend hostname lookup.
This removes a backend DNS lookup dependency. It does not prove global DNS isolation or replace worker, exchange, registry, or platform DNS controls.
A changed private endpoint address requires matching relay configuration and new admission observation.
Standalone operation without backend-address retains normal resolution of the fixed backend origin.
Production exposes no insecure TLS or arbitrary transport option. Trusted local tests can inject their owned transport.

## Azure profile binding

ExchangeSettings now requires backendAddress in addition to backendOrigin.
The expected relay command includes --backend-address, and backend endpoint settings must select that same observed address.
An endpoint observation with a different relay address rejects configuration.

The relay's startup probe uses /healthz. Its liveness probe uses /livez.
Its readiness probe now uses /readyz with a four-second platform timeout around the three-second backend check.
The selected app and healthy active revision must retain those exact command and probe settings.
The [exchange observer](azure-worker-exchange.md) compares them through the existing full specification checks.

This implementation makes the readiness check available from the correct network location.
Local TLS checks do not establish an installed relay route, actual Azure readiness, effective VNet policy, or complete provider admission.
Continue those native controls and separate deployment verification before coded execution enablement.
The count remains 48 verified of 57, with nine open.

## Installation and upgrades

Build and approve updated relay and service images before deploying these settings.
Earlier relay images do not accept backend-address. Earlier backend images can omit the required worker role field.
The service image's generic health command continues to accept the additional worker health field.
The local demonstration uses the ordinary API role and needs no restart for this private protocol change.

Use a trusted certificate issuer available in the approved relay image's public CA bundle.
The relay has no database or artifact credential and opens no identity directory or cloud SDK.
The backend readiness reply reports database availability. It does not inspect artifact storage, target applications, or test outcomes.

## Targeted local verification

Five new focused race checks cover pinned worker traffic, fresh credential-free TLS, health schema and role rejection, certificate lifetime, normal trust, cancellation, concurrency, and private dial scope.
Ten selected existing relay, command, backend, and Azure definition or endpoint checks pass for the changed contracts.
Only changed certificate, worker health, transport, and configuration cases repeat after review. No broad runner suite, UI build, or image build repeats.
The production dependency audit still permits only the standard library and the existing wire and identifier helpers.

Fixtures use owned local TLS listeners and fake SDK transports. The private dial hook redirects only to those owned loopback listeners.
All listeners, request goroutines, idle transports, and the command-check subprocess close through cleanup.
No cloud credential, cloud request, browser, container, registry push, or deployment runs.
Evidence is output/worker-relay-readiness/summary.json and targeted command results.
Acceptance 51 advances. Provider admission and all nine remaining broad acceptance items stay open.
