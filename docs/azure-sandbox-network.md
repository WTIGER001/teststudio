# Azure sandbox connection rules and routes

## Introduction

OpenRuntime now includes SDKNetwork in sandbox creation, live admission, and source import.
It reads the selected security group and route table through the existing bounded ARM reader.
The metadata identity supplies these reads. No additional credential or cloud mutation is required.

This checks reported rule and route contents. It does not prove effective routing or deployed sandbox compatibility.
Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Configuration and ownership

placement.network requires firewallIp and privateEndpointIps.
Each value is a canonical private IPv4 address outside the execution subnet.
The endpoint list contains one to eight unique addresses in lexical order.
The firewall address cannot also be an allowed endpoint.
These address checks do not establish endpoint ownership, DNS resolution, or firewall ownership.
The required [private stage](azure-sandbox-private.md) now checks selected endpoint ownership, exact NIC addresses, and complete selected zone inventories.
The remaining protection dependencies must establish those relationships before activation.

The sandbox-placement/v4 profile includes these settings, private endpoint and DNS bindings, topology, and the existing policyProfile fingerprint.
Changing a firewall or endpoint address changes launch and converted-image ownership.
Settings reject a stale profile before credentials are constructed.
Earlier staged profiles require regeneration. No native command currently loads those profiles.
SDKNetwork retains its own endpoint-list copy. Caller mutation cannot change an accepted policy.

NetworkDefinitions returns two reviewable resource bodies. It makes no Azure request.
Their map keys identify the selected security group and route table.
Packaging must supply the resources and complete their ownership and protection bindings.

### Security group contract

The group must match its identifier, name, type, region, and successful provisioning state.
It attaches only to the execution subnet and has no network-interface attachments.
flushConnection must be true.
The complete custom rule list contains exactly these rules:

| Direction | Priority | Decision | Scope |
| --- | --- | --- | --- |
| Inbound | 4096 | Deny | All protocols, sources, destinations, and ports. |
| Outbound | 100 | Deny | All traffic to AzurePlatformIMDS. |
| Outbound | 120 | Allow | Execution subnet to AzurePlatformDNS on UDP port 53. |
| Outbound | 121 | Allow | Execution subnet to AzurePlatformDNS on TCP port 53. |
| Outbound | 300 | Allow | Execution subnet to the selected private endpoint addresses on TCP port 443. |
| Outbound | 4096 | Deny | All remaining outbound traffic. |

No ordinary Jobs platform, registry, identity, load-balancer, or internal-subnet allow rules enter this sandbox connection policy.
Actual preview-service compatibility with this connection policy remains separate deployment work.
Required service changes must update the reviewed policy and its ownership version before activation.
The sandbox's own Full inspection and exact exchange-host allow rule remain independently required.

Rules must match names, priorities, protocols, address forms, ports, access, and direction.
Duplicate rules, duplicate endpoint addresses, extra rules, application security groups, and unknown writable fields reject.
Endpoint-list ordering in service responses is immaterial. Rule descriptions are ignored.
Reported child identifiers and types must match when present. Every child must report successful provisioning.
Unknown group properties reject. Known read-only resource identifiers and default-rule metadata do not replace the custom deny rules.

### Route table contract

The table must match its identifier, name, type, region, and successful provisioning state.
It attaches only to the execution subnet and disables BGP route propagation.
Exactly one sandbox-egress route selects 0.0.0.0/0, VirtualAppliance, and the configured firewall address.
Additional routes, Internet next hops, changed next hops, BGP overrides, and unknown properties reject.
The route must report successful provisioning.

This verifies configured user-defined routes. It does not establish the effective route set.
These rule and route checks do not establish VNet contents, effective DNS paths, firewall enforcement, trusted service ownership, or complete authority.
The required [topology stage](azure-sandbox-topology.md) now checks selected VNet contents, firewall ownership, attachments, and the complete firewall policy inventory.
Actual routing and traffic enforcement remain separate deployment verification.
The independent timer and selected identity stages remain required.

### Observation bounds and shutdown

Initial creation and image import always obtain a fresh two-resource observation.
Successful observations can supply shared live evidence for at most 15 seconds from observation start.
Concurrent live callers share one successful refresh after expiry.
A failed current full observation invalidates prior successful evidence. Failures are not cached.
The remaining protection callback, sandbox-definition checks, and current worker authorization remain uncached.

Each observation has a five-second bound and joins its selected reads.
Only the two exact management.azure.com GET paths with API version 2025-03-01 are permitted.
Redirects, mutations, filters, other resources, automatic retries, and provider registration reject.
The shared reader bounds responses to 1 MiB and closes their bodies.
Missing metadata remains unavailable. A policy 404 cannot establish worker absence or permit a replacement execution.
Returned errors omit provider bodies and credential diagnostics.
Runtime.Close closes the network reader after joining owned command work.

## Tutorials

### Verify the changed boundary locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxNetwork|^TestSandboxRuntimeNetwork' -count=1 -timeout=35s
```

Five focused checks cover fixed policy, rule and route drift, address bindings, cache invalidation, and automatic runtime integration.
They verify fresh creation and import checks, bounded shared live observations, caller mutation, canceled reads, and shutdown.
Affected placement, authority, runtime, shared execution, and command authorization checks also pass.
The combined targeted run passes 23 test functions with race detection in 2.211 seconds.
Evidence is output/azure-sandbox-network/summary.json and targeted.log.
The checks use fake credentials, in-memory HTTP responses, and local SQLite fixtures.
No cloud service, browser, container, UI build, demo restart, or remote push is involved.

### Complete native activation

Bind the selected endpoint addresses to the private exchange and registry endpoints.
Observe resolver policy, other linked zones, effective DNS resolution, trusted service ownership, identity host attachment, additional grants, and deny effects.
Then wire the scoped execution and discovery commands and package their dependencies.
Verify actual preview-service behavior, registry conversion access, timer execution, token audience, and proxy trust during separate deployment acceptance.

Schema references: [Microsoft security groups](https://learn.microsoft.com/en-us/azure/templates/microsoft.network/2025-03-01/networksecuritygroups)
and [Microsoft route tables](https://learn.microsoft.com/en-us/azure/templates/microsoft.network/2025-03-01/routetables).
The independent sandbox policy follows [Microsoft sandbox egress controls](https://learn.microsoft.com/en-us/azure/container-apps/sandboxes-egress-policies).
Related contracts: [placement](azure-sandbox-placement.md), [authority](azure-sandbox-authority.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.
