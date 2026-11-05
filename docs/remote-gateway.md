# Remote gateway transport

## Current scope

The remote transport connects a trusted worker supervisor to the existing browser and API gateway engine.
It implements the first component of the [hosted networking plan](deployment-readiness.md).
Hosted dispatch remains disabled for environment-backed coded tests and credential bindings.
Provider resources, initialization authority, supervisor wiring, network policies, and durable cleanup remain unfinished.
This component does not establish Kubernetes, ECS, or Azure deployment acceptance.
Azure implementation remains deferred.

## Connection boundary

The runner keeps its existing Unix socket protocol. The supervisor owns that socket and its remote TLS connection.
`ServeRemoteBridge` accepts the frozen target and a private `RemoteConnection`.
The connection contains one canonical HTTPS gateway origin, an optional certificate authority, and one execution session.
The bridge verifies the gateway certificate and hostname. It permits no TLS validation bypass.
The private session contains an execution identifier, a 32-byte capability, and a deadline within two hours.
The capability uses unpadded base64url encoding.
The gateway requires both the capability and execution identifier on every TLS proxy request.
Duplicate authorization headers, a wrong execution, plaintext requests, and expired sessions fail before target delivery.

Application credentials belong only in the gateway's private `Configuration`.
The bridge connection contains no application credential.
Future supervisor integration must keep the transport capability outside child input, environment variables, source, reports, and durable metadata.
Connection, session, and configuration diagnostic strings omit private values. JSON serialization remains private transport data.

## Policy and lifecycle

The remote server delegates accepted requests to the existing `NewAuthenticated` gateway engine.
Browser requests retain exact-origin HTTP and CONNECT enforcement and guarded destination resolution.
API requests retain reviewed methods, paths, body limits, request counts, credential injection, and reflected-secret redaction.
API CONNECT requests remain forbidden. Relative API paths resolve against the frozen target origin.
Remote transport headers are removed before target delivery.

The bridge uses bounded TLS connection setup, request transfer, and response header parsing.
The session deadline bounds both listeners and active connections.
Cancellation closes owned connections, including hijacked CONNECT streams.
Both target and runner tunnel connections close when either bridge or gateway stops.
This also fixes cancellation of hijacked connections in the shared local server implementation.

## Gateway process

The gateway binary accepts `--remote` for the TLS server and `--ready-remote` for its local TCP readiness check.
The remote listener uses port 8044. Certificates mount at `/tls/tls.crt` and `/tls/tls.key`.
Initialize it through the existing private configuration socket and `--configure` standard input.
The private configuration includes `target`, optional `credential`, and required `remote` session fields.
No credentials or transport capabilities belong in process arguments or deployment annotations.
Default local mode rejects a remote session. Unrecognized command arguments fail immediately.
The gateway image declares `teststudio.gateway-remote=tls-execution/v1`.
The existing image health command remains local. Future hosted packaging must select `--ready-remote` explicitly.

## Targeted local verification

New protocol checks use loopback TLS fixtures and an owned Unix socket.
They verify allowed HTTPS API delivery, credential injection, reflected-secret redaction, denied methods and paths, and foreign-origin rejection.
They also verify scoped authorization, duplicate-header rejection, plaintext rejection, and untrusted certificate rejection.
Active browser tunnels close after gateway cancellation, bridge cancellation, and the session deadline.
The selected gateway checks pass with Go's race detector.
Existing command configuration checks and selected canonical browser and API policy checks also pass.
No cloud account, cluster deployment, browser process, image rebuild, or new catalog run is required for these transport checks.

## Continuation

Add a separate private gateway initialization authority to the worker exchange.
Bind it to the frozen execution, dispatch authority, gateway resource identity, and deadline.
Keep it separate from the ordinary runner packet and activation token.
Wire the supervisor-owned bridge before launching uploaded code.
Extend Kubernetes admission, durable references, inventory, observation, stop, and cleanup to own both workloads.
Verify both workload identities and the enforced network boundary before releasing the activation gate.
Keep the current hosted rejection guards until that complete path works.
Use the dedicated local cluster for the first environment-backed browser and API scenarios.
Port the shared contract to ECS afterward. Keep Azure deferred.
