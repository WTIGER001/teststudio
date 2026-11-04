# Azure selected worker exchange binding

## Current boundary

ExchangeGuard inspects the selected relay environment, application, certificate, authentication configuration, service approval, routes, and revisions.
It runs after [registry service admission](azure-worker-registry.md).
The [standalone relay](worker-relay.md) supplies the bounded worker-facing process and its independent image package.

This stage does not implement Network, authorize dispatch, or release worker input.
A selected application lookup does not prove exclusive environment occupancy.
The [workload inventory stage](azure-worker-workloads.md) now checks reported app and job placement across the configured subscription.
Complete application and job inventories, attachment authority, effective network state, backend placement, and actual TLS verification remain required.
Azure coded execution stays disabled in native commands and infrastructure packaging.

## Explicit settings

ExchangeSettings requires these values:

| Setting | Requirement |
| --- | --- |
| registry | The exact settings used by the preceding registry guard. |
| application | The selected Container App resource. |
| certificate | An uploaded certificate child of the selected exchange environment. |
| virtualNetwork | The intended exchange VNet, distinct from worker and controller VNets. |
| subnet | The exchange infrastructure subnet in that VNet. |
| infrastructureGroup | The selected separate platform infrastructure group. |
| image | One immutable relay image in the configured registry, outside the test repository prefix. |
| backendOrigin | The exact HTTPS origin selected for the trusted worker backend. |

The application name follows the Container Apps length and naming profile.
The image requires an explicit repository and SHA-256 registry manifest digest.
A local Docker image ID cannot substitute for a published registry manifest reference.
Source suites cannot select this service image through the configured test repository prefix.
Trusted publication and installed image approval remain required before native enablement.

The relay and Azure settings share the same pure origin validator.
HTTP, credentials, paths, queries, invalid ports, and equivalent frontend/backend authorities reject configuration.
An explicit backend TLS port remains supported.
A valid origin is an intended destination. It does not prove that the destination serves only the private worker listener.

ApplicationDefinition and AuthDefinition return fresh SDK values for later packaging.
They issue no Azure write request. Caller changes cannot alter a saved guard.
The complete network profile must bind these settings before native execution enablement.

## Environment and service approval

The selected environment must match its configured identity, worker region, infrastructure subnet, and infrastructure group.
Its reported provisioning state must succeed, and public network access must be disabled.
It must report one Consumption workload profile and no environment runtime identity.
Application logs select none, without Log Analytics or Dapr telemetry credentials.
A custom environment DNS suffix or configured environment certificate authority cannot replace the selected app binding.

The default environment domain must match the expected regional azurecontainerapps.io namespace.
The app's reported default ingress hostname must derive from its name and that environment domain.
Actual VNet, subnet, routing, and backend attachment inspection remains separate work.
The configured VNet distinction alone does not establish effective network separation.

The complete service-side endpoint connection inventory must contain exactly the selected worker exchange endpoint.
It must report successful provisioning, Approved status, and no outstanding update action.
Reported groups must match managedEnvironment.
The environment's embedded connection inventory must independently agree with the complete collection.
Missing, pending, and inconsistent observations remain unavailable.
Foreign, rejected, duplicate, or extra connections reject ownership.

The private-link resource inventory must report one managedEnvironment group, one required member, and the expected regional private-link zone.
The actual DNS path remains the exact exchange hostname A record inspected by the preceding private DNS stage.
The regional service zone declaration does not replace that custom-hostname record.

## Relay application configuration

The app must report the selected environment, successful provisioning, and running or ready status.
Its complete configured specification must match the generated relay definition.
That definition uses these controls:

| Area | Required behavior |
| --- | --- |
| Container | One relay container with the exact immutable service image. |
| Command | /app/teststudio-exchange with fixed listen, public-origin, and backend-origin arguments. |
| Runtime authority | No app secrets, environment values, volumes, init containers, sidecars, service binds, or runtime identity. |
| Image pull | Only the selected pull identity, with lifecycle None. No registry username or password reference. |
| Ingress | HTTPS edge access to target port 8043, one SNI hostname, one selected certificate, and no extra ports. |
| Revisions | Single mode with one active latest revision receiving all traffic. |
| Scale | One through two replicas, fixed resources, and explicit bounded scale settings. |
| Probes | Local process health, readiness, and liveness endpoints from the relay package. |

External app ingress here publishes through the environment edge. Disabled environment public access keeps the selected path private.
The app's Easy Auth resource must explicitly report platform authentication disabled.
This prevents a separate sign-in or platform-authentication flow from replacing the worker ticket contract.
The complete environment HTTP-route inventory must be empty. An alternate route cannot override the app's fixed worker hostname path.

The guard ignores resource annotations and documented read-only metadata.
It accepts equivalent identifier case, probe order, inactive Dapr defaults, ignored client certificates, empty exposed-port defaults, and disabled sticky sessions.
It does not ignore traffic-bearing fields, extra configuration, different commands, changed backend arguments, or credentials.
Reported ephemeral container storage and revision suffix metadata do not replace the configured execution specification.

The complete selected-app revision inventory permits at most four observations.
Exactly one active revision must match both latest and latest-ready names.
It must be provisioned, healthy, running, have a replica, receive all traffic, and retain the exact relay template.
Inactive revisions must explicitly report zero replicas and zero traffic. Unknown counts remain unavailable.
Revision identifiers that differ only by letter case cannot count as separate observations.
Unsettled rollout or inactive replica state remains unavailable.
Revision readiness does not prove backend availability. The relay's probes check its own process only.

## HTTPS certificate metadata

The certificate must belong to the selected environment, report successful provisioning, and be marked valid.
Its issue date cannot be future, and its remaining life must exceed two hours plus five minutes.
That margin covers the current maximum worker exchange deadline with a small admission allowance.
Unknown, pending, invalid, or near-expiry observations remain unavailable.

Standard hostname verification checks the reported DNS subject alternative names against the exact worker-facing hostname.
Single-label wildcard SANs retain normal Go hostname semantics.
A different certificate binding, unmatched SAN, foreign certificate resource, returned private key material, or Key Vault certificate reference rejects this profile.
This initial profile uses an uploaded certificate without giving the relay a certificate-store identity.

These are management-plane metadata checks.
They do not verify a served certificate chain, actual SNI behavior, recursive DNS, or packet reachability.
Those checks belong to the separate deployment verification phase.
Private endpoint custom-domain setup also requires the documented domain ownership verification record.
Public ownership validation records are distinct from the selected worker private zone's runtime A record.

## SDK scope and uncertainty

SDKExchangeReader uses the existing armappcontainers/v4 4.0.0 SDK and management API 2025-07-01.
It requires an explicit TokenCredential and the existing bounded ARM transport.
It permits only eight selected resources: environment, application, certificate, current auth configuration, and four complete child collections.
Secret listing, certificate export, app auth tokens, writes, deletion, subscription discovery, resource-group discovery, filters, and foreign resources reject before transport.
It neither requests broad workload inventory nor claims that inventory is complete.
A separate inventory and attachment-control stage must establish exclusivity before Network can authorize execution.

Connections and private-link resources permit one item and two pages each.
HTTP routes permit zero items in one complete response.
Revisions permit four items and three pages.
Foreign pages, other collections, repeated tokens, excess inventory, redirects, and altered queries reject inspection.
Incomplete bounded paging remains unavailable.

Eight observations run concurrently after the registry stage succeeds.
A normal complete check makes forty-three reads. The full paging bound is sixty-two reads.
The guard has a thirty-five-second context. Each SDK operation retains its five-second bound and four-megabyte response ceiling.
No successful-result cache hides later changes.
Provider diagnostics become fixed control errors, without response bodies or credentials.
The controller needs read access only to these selected resources and collections for this stage.
Close the reader to close idle transport connections.

## Targeted local verification

Six new focused race checks cover selected authority, service approval, app specification, revision state, certificate metadata, incomplete observations, paging, and SDK scope.
They reject runtime credentials, extra identities, API commands, changed backend origins, additional ports, Easy Auth, alternate routes, foreign certificates, and unrelated approval.
One existing relay configuration check passes after the shared pure origin-validator extraction.
The relay production dependency audit still reports standard-library and wire-helper packages only.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureExchange' -count=1
go test -race ./adapters/workerexchange/relay -run '^TestWorkerRelayIsFixedOriginAndBoundsConcurrentForwarding$' -count=1
python3 scripts/verify-worker-relay-dependencies.py
```

Evidence is output/azure-worker-exchange/summary.json and the focused command results.
No cloud credential, live cloud API, TLS probe, DNS probe, packet probe, registry push, deployment, UI build, browser, container, or image build runs.
Existing relay image workflow evidence remains retained. This metadata batch repeats no image or broad runner fixture.
Acceptance 51 advances. The total remains 48 verified items out of 57.

## Next implementation

Compose workload inventory observation. Verify effective read coverage, exclusive attachment authority, and actual exchange network configuration.
Verify that the selected backend reaches only the trusted worker listener with the intended TLS authority.
Complete effective VNet-wide DNS visibility, trusted preparation networking, cross-job isolation, execution placement, managed identity roles, and full profile binding.
Then add native configuration and infrastructure service wiring before separate deployment acceptance.

## Primary references

[Container Apps private endpoints](https://learn.microsoft.com/en-us/azure/container-apps/private-endpoints-with-dns) defines the private managedEnvironment service and disabled public access.
[Custom domains and certificates](https://learn.microsoft.com/en-us/azure/container-apps/custom-domains-certificates) defines SNI binding and private endpoint DNS configuration.
[Container Apps operations](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps?view=rest-resource-manager-containerapps-2025-07-01) documents the available inventory scopes.
The pinned SDK defines the inspected environment, application, certificate, route, authentication, and revision fields.
