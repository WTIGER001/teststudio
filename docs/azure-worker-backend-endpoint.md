# Azure private backend endpoint observation

## Scope

BackendEndpointGuard observes the selected relay-to-backend private endpoint after [backend service observation](azure-worker-backend-admission.md).
It binds consumer approval, service approval, NIC ownership, the private IP, the dedicated endpoint subnet, and its security group.
The allow rule uses the relay infrastructure subnet's observed prefix and permits TCP 443 to only the selected backend address.
The stage checks reported configuration. It sends no packet, DNS query, or TLS probe to the backend.

This stage does not implement Network, authorize dispatch, or enable coded Azure execution.
Complete relay routing, VNet policy, DNS, actual TLS trust, effective identity authority, attachment controls, and execution isolation remain required.
It does not create or approve an endpoint. The [staged backend package](azure-worker-backend.md) still provisions no relay access path.
The verified count remains 48 of 57, with nine open.

## Explicit settings

BackendEndpointSettings requires these values:

| Field | Requirement |
| --- | --- |
| backend | Exact BackendSettings used by the preceding service guard. |
| endpoint | Selected private endpoint identifier in the installation subscription. |
| interface | Selected NIC identifier for that endpoint. |
| subnet | Dedicated endpoint subnet identifier in the exchange VNet. |
| subnetPrefix | Canonical private IPv4 prefix with at least eight addresses. |
| relaySubnetPrefix | Canonical private IPv4 prefix for the selected relay infrastructure subnet, with at least 32 addresses. |
| securityGroup | Dedicated endpoint NSG identifier. |
| address | Exact private IPv4 address assigned to the backend endpoint NIC. It must equal the relay backendAddress. |

The endpoint subnet differs from the relay infrastructure subnet and overlaps neither its prefix nor the configured worker VNet.
The relay prefix also differs from the worker address space.
The first four subnet addresses and the final address cannot identify an endpoint.
Mapped IPv6, public addresses, noncanonical prefixes, and inconsistent parent identifiers reject configuration.
The dedicated NSG cannot reuse either existing worker NSG identifier.

SecurityGroupDefinition and SubnetDefinition return fresh SDK values without issuing resource writes.
Native setup must bind all settings into the complete admission profile before execution enablement.
Stable selected NIC naming and the installed private IP must match later observed state.

## Approval and address ownership

The endpoint must report successful provisioning, one selected NIC, one selected subnet, and one approved automatic service connection.
That connection must target the selected backend environment and its managedEnvironment group.
Manual connections, other services, extra interfaces, alternate groups, and pending update actions reject or leave observation unavailable.

The backend environment's complete reported service connection collection must contain exactly that approved consumer endpoint.
Its embedded environment connections must independently agree.
The guard reuses the preceding call's validated environment observation. It issues no duplicate environment GET.

The backend private-link resource must declare one managedEnvironment group, one required member, and the selected regional private-link zone.
The NIC connection member must equal the service's reported required member.
Any declared endpoint IP configuration must agree with the same member and selected address.
The [Azure private endpoint guide](https://learn.microsoft.com/en-us/azure/container-apps/private-endpoints-with-dns) documents the service group and regional zone.

The NIC must point back to the selected endpoint, use its exact private IPv4 address and subnet, and contain one IP configuration.
Public IPs, forwarding, independent NIC security groups, compute attachments, and unrelated workload configuration reject observation.
The dedicated subnet must contain exactly that endpoint and the NIC's observed IP configuration reference.
Missing, duplicate, and unrelated attachment references cannot establish exclusive endpoint ownership.

## Endpoint subnet and security policy

Both private endpoint and private-link service network policies must explicitly be enabled.
The subnet must disable default outbound access and attach its selected NSG.
It must have no route table, NAT gateway, delegation, service endpoint, sharing configuration, or unrelated allocation.
The [Azure network policy guide](https://learn.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy) explains policy support for endpoint security rules.

The NSG must attach only to this endpoint subnet, flush updated connections, and contain exactly three custom rules:

1. Allow inbound TCP 443 from the selected relay subnet prefix to the backend endpoint address.
2. Deny all remaining inbound traffic before platform default rules.
3. Deny all remaining outbound traffic before platform default rules.

The selected relay subnet must report the configured prefix, successful provisioning, and its Microsoft.App/environments delegation.
This source-prefix check does not inspect the relay's effective route table, NSG contents, peering, DNS, firewall, or alternative outbound paths.
Those remain mandatory later admission controls. An endpoint NSG alone does not establish isolated relay networking.

## Read boundary and availability

The reader accepts an explicit credential and GET access to seven selected resources or collections:

- The private endpoint and its NIC.
- The dedicated endpoint subnet and NSG.
- The relay infrastructure subnet.
- The backend environment's service connection and private-link resource collections.

Network resources use armnetwork/v8 and management API 2025-03-01.
Service collections use armappcontainers/v4 and management API 2025-07-01.
The shared transport permits neither writes nor secret reads, unrelated endpoints, a duplicate environment read, alternate API versions, filters, or foreign pages.
Each service collection permits one entry and at most two pages. Unfinished pages cannot establish complete approval.
Individual reads have five-second contexts and four-megabyte response limits.
The full composed guard has a fifty-second context, fifty-seven normal reads, and a maximum of 110 reads with bounded paging.
It stores no successful-admission cache and persists no environment observation.

## Targeted local verification

Four new focused race checks cover fresh approval, environment reuse, foreign services, NIC and IP changes, member consistency, subnet sharing, security bypass, read scope, and incomplete observations.
Four selected existing endpoint and backend checks pass after shared endpoint reader and verifier extraction.
Only the changed definition check repeats after review strengthens its exact destination assertion.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection.
No cloud request, cloud credential, browser, UI build, image build, container, live service listener, registry push, or deployment runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureBackendEndpoint.*|TestAzureEndpointsVerifyFreshAddressOwnershipAndPriorStages|TestAzureEndpointsRejectForeignServicesAddressesAndPolicyBypass|TestAzureEndpointsSDKRestrictsResourceKindsAndTransportAuthority|TestAzureBackendObservesFreshWorkerBindingAndSemanticConfiguration)$' -count=1
```

Evidence is output/azure-worker-backend-endpoint/summary.json and targeted command results.
The [relay readiness implementation](worker-relay-readiness.md) now verifies backend TLS and worker-role health through that pinned address.
Local checks do not establish the installed route. Continue effective relay networking, DNS isolation, and deployed trust verification before native dispatch wiring.
Acceptance 51 advances.
