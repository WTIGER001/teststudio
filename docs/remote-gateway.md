# Remote gateway transport

## Current scope

The remote transport connects a trusted worker supervisor to the existing browser and API gateway engine.
It implements the first component of the [hosted networking plan](deployment-readiness.md).
Kubernetes now has opt-in environment executor admission and installation packaging. Actual network execution on the dedicated cluster remains unverified.
ECS and Azure environment-backed coded execution remain disabled.
Shared private initialization and supervisor bridge delivery are implemented.
The [Kubernetes resource lifecycle](kubernetes-gateway.md) now implements bundle ownership, instance binding, network policies, stop, inventory, and cleanup.
Normal Kubernetes admission, discovery, image approval, and installation wiring now pass targeted local checks.
Actual network execution remains unverified.
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
The supervisor keeps the transport capability outside child input and environment variables.
Source, reports, and durable metadata must also exclude that capability.
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

The process also accepts `--remote-exchange=HTTPS_ORIGIN` for private initialization through the exchange.
Supply `TESTSTUDIO_GATEWAY_EXECUTION`, `TESTSTUDIO_GATEWAY_TOKEN`, and `TESTSTUDIO_GATEWAY_INSTANCE` only to the trusted gateway workload.
The instance must match the provider-confirmed workload identity.
Initialization retries temporary unavailability for at most 30 seconds. Authorization denial stops initialization immediately.
The exchange certificate uses normal system trust. A private installation must mount its approved CA bundle.
This process mode is implemented and compiles. The provider installation path remains unverified.

## Separate initialization authority

The controller calls `PrepareGateway` after preparing the gated execution.
This freezes the gateway HTTPS origin and optional certificate authority.
The application credential reference is already frozen separately by execution preparation.
Neither this reference nor its resolved value enters the runner packet.
Preparation retries cannot replace the gateway origin, CA, or application credential reference.

The provider must confirm the actual gateway workload before calling `BindGateway` with its immutable instance identifier.
Binding another instance fails. Configuration delivery waits until this identity is bound.
The private `gateway-input` route accepts only the separate initialization ticket and that bound instance.
It checks current execution authority before resolving the frozen application credential.
The initialization ticket cannot fetch uploaded code or the runner bridge connection.

The `gateway-connection` route accepts only the normal worker ticket after activation and input claim.
It returns the credential-free TLS connection to that same worker instance.
The proxy transport capability uses another HMAC purpose. It cannot fetch private initialization or uploaded code.
All three capabilities remain distinct and reconstruct consistently after service restart.
Resolved application credentials and raw capabilities are absent from durable exchange records.
Revocation, completed submissions, expired deadlines, or withdrawn run authority deny new gateway configuration and bridge access.

Activation remains closed until a gateway identity is bound and private configuration is delivered.
That exchange check does not prove Pod readiness or network isolation.
The provider must independently verify both workload identities, readiness, and network policies before activation.

## Supervisor delivery

For target-backed work, the supervisor fetches the separate bridge connection after it receives the activated frozen packet.
It verifies the execution identifier and exact packet deadline before launching the child.
It creates the private socket at `/output/.browser-egress.sock`.
The child receives the existing canonical work item and fixed runtime environment.
It receives no gateway ticket or proxy capability from that input or environment.
The supervisor closes owned connections and removes the socket when delivery finishes or fails.

New runtime overlays advertise `teststudio.job-gateway=tls-execution/v1`.
Registry inspection exposes this optional capability. Existing offline runtime images remain valid for offline work.
Hosted network admission requires this capability, the supervisor build revision, and a distinct timing profile.
No runtime image is rebuilt for this shared-component milestone.

## Targeted local verification

New protocol checks use loopback TLS fixtures and an owned Unix socket.
They verify allowed HTTPS API delivery, credential injection, reflected-secret redaction, denied methods and paths, and foreign-origin rejection.
They also verify scoped authorization, duplicate-header rejection, plaintext rejection, and untrusted certificate rejection.
Active browser tunnels close after gateway cancellation, bridge cancellation, and the session deadline.
The selected gateway checks pass with Go's race detector.
Existing command configuration checks and selected canonical browser and API policy checks also pass.
Private exchange checks verify separate roles, instance binding, closed activation, frozen references, restart, and authority withdrawal.
Those exchange checks pass with the race detector.
A new child-process check verifies supervisor socket use, stripped gateway environment, prelaunch scope rejection, and socket removal.
The child-process check also passes with the race detector.
No cloud account, cluster deployment, browser process, image rebuild, or new catalog run is required for these transport checks.

## Continuation

Use the implemented private initialization and supervisor contracts when wiring the provider.
Configure the trusted private exchange's secret provider before admitting authenticated suites.
Mount the gateway server certificate and approved exchange trust through installation configuration.
Use the implemented Kubernetes bundle lifecycle for admission and installation wiring.
Verify both workload identities and the enforced network boundary before releasing the activation gate.
Require the inspected supervisor gateway capability for environment-backed images.
Extend hosted source-discovery validation for the reviewed environment build profiles while keeping discovery network-isolated.
Keep unsupported runtime and provider profiles rejected. Kubernetes opt-in admission now connects the complete shared contract.
Use the dedicated local cluster for the first environment-backed browser and API scenarios.
Port the shared contract to ECS afterward. Keep Azure deferred.
