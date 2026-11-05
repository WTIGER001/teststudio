# Azure sandbox registry service admission

## Introduction

OpenRuntime includes SDKRegistryService in creation, source import, and live admission.
It verifies registry access settings and complete reported connection, private-link member, and replication inventories.
It reuses RegistryServiceBinding, the pure verifier also used by the existing registry guard.
Sandbox settings remain independent of ordinary Jobs settings.

The existing private and relay stages verify the two selected network paths.
This service stage verifies registry-side metadata for all declared consumers.
Network ownership of additional trusted consumers remains separate work.

## Concepts

### Declared consumers

The expected connection inventory always includes the sandbox registry endpoint and relay registry endpoint.
placement.registryConsumers adds up to six explicit trusted private endpoint identifiers.
Use that list for preparation, backend, and inspection paths when those roles need separate registry endpoints.
Do not repeat either derived endpoint in this list.

Each additional endpoint must belong to the configured subscription.
Duplicates, foreign subscriptions, wrong resource types, and exchange or backend-service endpoint aliases reject configuration.
The backend-service endpoint targets Container Apps, so it cannot serve as a registry endpoint.
The constructor copies the additional list to preserve the admitted configuration against later caller edits.

The sandbox-placement/v9 profile binds the additional consumers with all existing placement and service settings.
Changed consumers require profile regeneration. Older staged profiles reject before credential construction.
Native commands do not load these settings yet.
Listing an endpoint does not verify its NIC, subnet, private DNS, workload ownership, or image-pull authority.
Those bindings remain required in the complete protection composition.

### Registry contract

The selected registry must report its exact identifier, login host, region, Premium SKU, and successful provisioning.
An absent optional SKU tier remains absent during SDK model conversion.
An explicitly incompatible tier rejects admission.
Dedicated data endpoints must be enabled, with exactly the configured regional data hostname.

| Setting | Required value |
| --- | --- |
| Public network access | Disabled. |
| Administrator user and anonymous pull | Disabled. |
| Trusted-service network bypass | None. |
| Task network bypass | Disabled. |
| Network rule default | Deny, with no IP exceptions. |
| Artifact export policy | Disabled. |
| Role assignment mode | LegacyRegistryPermissions. |
| ARM-audience authentication | Enabled. |

The complete private-link resource inventory contains exactly the registry group.
Its required members are registry and registry_data_REGION.
Its required private DNS zone is privatelink.azurecr.io.
The complete replication inventory is empty or contains one successfully provisioned home-region replica with its regional endpoint enabled.
Other regions, additional members, unknown policy values, and incomplete observations reject or remain unavailable.

### Complete reported approval inventory

The complete registry-side connection list must approve exactly the declared endpoints.
The embedded registry connection list must independently approve the same set.
Both lists must associate each endpoint with the same connection identifier.
Ordering does not affect the comparison.

Extra, duplicate, foreign, rejected, or disconnected consumers reject admission.
Missing or pending approvals remain unavailable.
Different connection identifiers across otherwise valid snapshots remain unavailable until the observations agree.
No partial list establishes complete consumer ownership.

### Read bounds and lifecycle

The reader uses API version 2025-11-01 for four selected GET paths:
the registry, privateEndpointConnections, privateLinkResources, and replications.
The same explicit metadata identity supplies every read.
Construction obtains no token and calls no service.

The complete connection list permits at most eight endpoints and nine pages.
The member and replication lists each permit one entry and two pages.
The whole observation has a five-second bound and joins all reads before returning.
Continuation host, path, version, query, duplicate identifiers, and pagination limits retain the shared metadata guards.
Redirects, automatic retries, request mutations, oversized responses, and provider diagnostic exposure remain blocked.
Metadata gaps remain unavailable and never establish sandbox absence.

Creation and import obtain fresh observations.
Concurrent live checks share successful evidence for at most 15 seconds from observation start.
Current failures invalidate prior success. Failures remain uncached.
Remaining protection, sandbox-definition, and authorization checks remain uncached.
Runtime.Close closes the registry reader after joining owned command work.

## Tutorials

### Verify locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxRegistry|^TestSandboxRuntimeRegistryService' -count=1 -timeout=30s
```

Six new checks cover complete consumers, configuration binding, drift, pagination, shared evidence, and automatic runtime composition.
Seven affected native checks and four existing shared registry checks also pass with race detection.
Evidence is output/azure-sandbox-registry-service/summary.json, native.log, and shared.log.
Checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Complete native activation

Bind additional declared consumers to their actual trusted workloads and network paths.
Complete identity host attachment, effective grants, deny effects, and protected write boundaries.
Then wire execution and discovery commands separately and package their dependencies.
Verify private image conversion, publication, actual pulls, service compatibility, routing, and independent deadlines during separate deployment acceptance.

Related guides: [runtime](azure-sandbox-runtime.md), [relay network](azure-sandbox-relay-network.md), and [shared registry contract](azure-worker-registry.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
