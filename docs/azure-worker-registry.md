# Azure worker registry service admission

## Current boundary

RegistryGuard inspects the actual registry after [endpoint ownership checks](azure-worker-endpoints.md) succeed.
It checks private access settings, regional data authority, required endpoint members, and complete service-side connection inventories.
Its production SDK reader performs bounded management reads. Local fixtures supply responses during current verification.

This stage does not implement Network, dispatch work, or release worker input.
Azure coded execution stays disabled in native commands and infrastructure packaging.
Exchange authority, effective DNS visibility, trusted preparation networking, cross-job isolation, execution placement, and identity role bindings remain unfinished.
Local management responses do not prove actual image pulls or publication.

## Settings and profile

RegistrySettings contains two values:

| Setting | Requirement |
| --- | --- |
| endpoints | The exact EndpointSettings used by the preceding guard. |
| preparationEndpoint | The separate private endpoint selected for trusted preparation registry access. |

The preparation endpoint uses the selected subscription and an explicit resource group.
It cannot reuse either selected worker endpoint.
The constructor preserves the complete preceding configuration and its canonical address list.
Changed configuration cannot reuse an earlier guard as authority.

This profile uses one registry data region, matching the preceding two-address registry NIC contract.
The registry's actual primary location must equal registryDataRegion.
The worker region can differ from that registry region.
Additional regional data members or replicas require corresponding NIC, DNS, and policy profile support before enablement.

This profile uses LegacyRegistryPermissions, matching the existing AcrPull infrastructure arrangement.
An ABAC-enabled registry cannot silently reuse those role assumptions.
The trusted token client can support repository-conditioned roles, but native ABAC role packaging remains separate work.

## Actual registry settings

The registry must report the configured resource identity, primary region, Premium SKU, and successful provisioning.
Its login server must equal Config.Registry.
Data endpoints must be enabled, with exactly the selected REGION.data.azurecr.io hostname for that registry.
The private-link resource inventory must contain exactly the registry group.
Its required members must equal registry and registry_data_REGION.
Its required private DNS zone must equal privatelink.azurecr.io.

The complete replication inventory may be empty or contain one ready home-region observation.
An explicit home observation must report the selected region and enabled regional endpoint.
Foreign regions, extra replicas, and unfinished observations cannot establish this profile.

The service must explicitly report these access settings:

| Setting | Required value |
| --- | --- |
| Public network access | Disabled. |
| Administrator user | Disabled. |
| Anonymous pull | Disabled. |
| Trusted-service network bypass | None. |
| Task network bypass | Disabled. |
| Network default action | Deny, with no IP exceptions. |
| Artifact export policy | Disabled. |
| Registry role assignment mode | LegacyRegistryPermissions. |
| ARM-audience authentication | Enabled for platform managed-identity image pulls. |

Missing access fields or policy status leave admission unavailable.
Metadata and unrelated retention settings do not replace these checks.
Registry identity permissions and effective role assignments remain separate inspection requirements.

Container Apps managed-identity image pulls require ARM-audience authentication at the registry.
Keep that policy enabled for the platform pull identity.
The [trusted registry client](azure-registry-authentication.md) separately requests the registry-specific token audience without ARM fallback.
The job contract keeps the pull identity unavailable to untrusted container code through lifecycle None.
Actual platform behavior still requires separate deployment verification.

## Service-side approval

The complete private endpoint connection list must contain exactly the selected worker registry endpoint and preparation endpoint.
Each connection must have its own child identity under the registry and the correct endpoint resource identifier.
Resource identity comparison ignores letter case. Duplicate identities cannot count as separate connections.
Both connections must report successful provisioning, Approved status, and no outstanding update action.

The registry's embedded connection inventory must independently satisfy those rules and agree with the complete collection.
Different snapshots leave admission unavailable.
Missing connections, pending approval, missing status, and required updates also remain unavailable.
Rejected, disconnected, duplicate, foreign, and extra connections reject ownership.
Descriptions and inventory order do not change authority.

Service-side preparation approval does not prove the preparation endpoint's NIC, subnet, DNS, or trusted process placement.
Inspect that separate trusted path before native preparation wiring.
Keep its resources in trusted networking. Do not add preparation exceptions to the worker endpoint subnet.

## SDK and read scope

SDKRegistryReader pins armcontainerregistry/v3 3.0.0 and management API 2025-11-01.
This stable model exposes anonymous access, task bypass, role mode, and authentication policy fields needed by this profile.
It requires an explicit TokenCredential and uses the existing public ARM endpoint and bounded transport.
It permits GET only on the selected registry and its privateEndpointConnections, privateLinkResources, and replications collections.

Credential listing, credential regeneration, approval writes, deletion, subscription discovery, filters, alternate versions, and foreign resources reject before transport.
Automatic retries, resource-provider registration, and redirects remain disabled.
Each operation has a five-second context and four-megabyte response bound.
Provider diagnostics become fixed control errors.

Connections permit at most two items and three pages.
Private-link resources and replications each permit at most one item and two pages.
The shared pager rejects foreign origins, other collections, repeated tokens, malformed queries, and excess inventory.
Unfinished bounded paging remains unavailable.

The four new reads run concurrently after endpoint inspection.
A normal full check makes thirty-five reads. The complete paging bound is fifty reads.
The guard has a thirty-second context and no successful-result cache.
The controller needs read access only to these selected registry resources.
This reader needs no registry password, image token, approval authority, or write permission.
Close it to close idle transport connections.

## Targeted local verification

Six new focused race checks cover fresh observations, access settings, regional authority, approval, incomplete state, paging, and SDK scope.
They cover anonymous and public access, task bypass, incompatible roles, extra regions, foreign connections, duplicate identity case, and changed snapshots.
Constructor checks preserve the distinct preparation endpoint and preceding profile.
All six checks pass across targeted commands. Two changed checks repeat after review fixes.
No unchanged broad suite or UI build runs.
No live cloud credential, cloud call, DNS probe, packet probe, registry push, deployment, browser, container, or image build runs.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureRegistryService' -count=1
```

Evidence is output/azure-worker-registry/summary.json and the targeted command results.
Acceptance 51 advances. The total remains 48 verified items out of 57.

## Next implementation

Inspect the separate exchange environment, service-side approval, private application inventory, and HTTPS hostname authority.
Resolve effective VNet-wide private DNS visibility and attachment controls.
Inspect trusted preparation registry networking and effective managed identity role assignments.
Establish cross-job isolation and execution placement before composing Network.
Then complete native configuration, infrastructure packaging, and separate deployment verification.

## Primary references

[Registry private endpoints](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-endpoints) describes Premium private connectivity and disabled public access.
[Required private-link members](https://learn.microsoft.com/en-us/rest/api/container-registry/registries/list-private-link-resources?view=rest-container-registry-2025-11-01) defines the group, regional members, and required zones.
[Image pulls with managed identity](https://learn.microsoft.com/azure/container-apps/managed-identity-image-pull) requires ARM-audience authentication for platform pulls.
[Artifact export policy](https://learn.microsoft.com/en-us/azure/container-registry/data-loss-prevention) describes disabled exports and their private-access prerequisite.
The pinned SDK defines inspected access settings, service-side connection fields, and replication inventory.
