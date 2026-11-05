# Azure sandbox controller registry path

## Introduction

OpenRuntime now includes SDKHostRegistry before the selected controller host and workload checks.
It binds the controller's private image registry endpoint, NIC addresses, endpoint subnet rules, private DNS, and selected pull authority.
Creation and image import require fresh observations. Live admission uses bounded successful evidence.
Native command selection and complete controller network admission remain unfinished.

## Concepts

### Required settings

host.subnetPrefix supplies the controller environment subnet prefix.
host.registry uses these endpoint settings:

| Field | Required binding |
| --- | --- |
| endpoint, interface | Distinct selected private endpoint and NIC. |
| subnet, subnetPrefix | Separate endpoint subnet in the controller VNet. |
| securityGroup | Exclusive endpoint security group. |
| loginIp, dataIp | Distinct usable private addresses for registry login and regional data. |
| zone | Separate privatelink.azurecr.io zone. |
| connection | Registry-side approval for the selected endpoint. |
| pullAssignment | Registry-scoped AcrPull assignment for host.pull. |

The endpoint must appear in placement.registryConsumers.
Keep that shared list consistent across execution and discovery configurations that use the same converted-image pool.
Their host profiles can differ without changing the worker profile when shared placement and worker settings match.
Regenerate worker.networkProfile after changing declared consumers. Then calculate host.profile from Settings.HostProfile().

Missing fields, foreign resources, aliased protected resources, overlapping subnets, reserved addresses, and undeclared consumers reject configuration.
Controller prefixes cannot overlap the sandbox or relay VNet prefixes.
Endpoint and source subnets must differ and belong to the same controller VNet.

### Reported path and authority

The source subnet must report its exact prefix and the Container Apps environment delegation.
The selected endpoint targets the configured registry and contains exactly the selected NIC.
The NIC binds the registry and registry_data_REGION members to the two configured addresses.
The endpoint subnet contains only this endpoint and its exact NIC configurations.
Its security group permits inbound HTTPS only from the controller subnet to those two addresses.
Remaining inbound and outbound traffic is denied by the fixed endpoint rules.

The separate registry-side approval and the selected embedded approval must target this endpoint.
The [registry service stage](azure-sandbox-registry-service.md) independently checks the complete reported consumer inventory.
The pull identity must report its exact resource, region, tenant, client, and principal.
Its registry-scoped assignment must reference the unchanged built-in AcrPull role and contain no condition or delegation.
Reported assignment metadata does not establish complete effective permissions or actual token issuance.

### Private DNS

The selected zone has two exact A records: registry login and regional data.
Its only VNet link is controller-network targeting the controller VNet, with registration disabled and no public fallback.
Complete record and link inventories retain existing count, alias, wildcard, paging, and response guards.
The subscription [DNS inventory](azure-sandbox-resolver.md) now also guards the controller VNet.
An additional reported zone linked to that VNet rejects admission.
Visibility outside the selected subscription and deny-hidden resources remain separate requirements.

### Bounds and packaging

The reader has 13 selected GET paths and a five-second joined observation bound.
Network resources use API 2025-03-01, registry metadata uses 2025-11-01, and DNS uses 2024-06-01.
Identity, assignment, and role reads reuse the existing pinned versions and pure verifiers.
The reader acquires no credentials during construction and permits no resource mutation.
Successful live evidence expires 15 seconds after observation start. A current failure invalidates prior success.
Remaining protection and authorization remain uncached. Runtime.Close closes the reader after owned work joins.

HostRegistryDefinitions returns seven staged bodies: endpoint rules, endpoint subnet, pull assignment, zone, link, and two records.
These bodies do not create the endpoint, NIC, controller VNet, source network policy, or a complete deployment.

## Tutorials

### Verify locally

Three new checks and eleven affected checks pass with race detection in 4.277 seconds.
They cover both controller modes, invalid settings, endpoint and DNS drift, grants, cached evidence, fresh failure, admission composition, and shutdown.
Affected checks retain controller definitions, image-pool sharing, complete registry inventory, additional DNS links, backend endpoint checks, and relay networking.
The central discovery composition observes 128 unique selected paths.
Evidence is output/azure-sandbox-host-registry/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata.
No cloud service, browser, container, UI build, demo restart, or remote push runs.

### Continue activation

The [source perimeter](azure-sandbox-host-network.md) now checks VNet contents, source rules, firewall routing, derived HTTPS dependencies, and resolver association.
The [database stage](azure-sandbox-host-database.md) now checks reported private PostgreSQL transport.
Complete actual SQL access, native HTTPS-probe routing, and remaining dependency access.
Complete effective authority, deny effects, protected writes, remaining identity attachment, native command selection, and deployment packaging.
Keep deployed pulls, private conversion, token behavior, routing, preview compatibility, and independent timer execution in separate deployment acceptance.

Related guides: [controller host](azure-sandbox-host.md), [runtime](azure-sandbox-runtime.md), and [selected authority](azure-sandbox-authority.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
