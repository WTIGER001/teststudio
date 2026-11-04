# Azure worker private endpoint ownership

## Current boundary

EndpointGuard checks actual private endpoints, NIC addresses, endpoint subnet attachments, and subnet security rules after [private DNS inspection](azure-worker-private-dns.md).
It binds the selected DNS addresses to the configured registry and exchange endpoint resources.
The production reader uses scoped Azure SDK reads. Local fixtures supply the responses during verification.

This stage does not implement Network, authorize dispatch, or release worker input.
Azure coded execution remains disabled in native commands and infrastructure packaging.
Service-side connection approval, actual registry configuration, isolated exchange authority, effective DNS visibility, and cross-job isolation remain required.
Local fixture success does not establish service reachability or deployment compatibility.

## Explicit settings

EndpointSettings requires these values:

| Setting | Requirement |
| --- | --- |
| privateDns | The exact settings used by the preceding private DNS guard. |
| subnet | A separate endpoint subnet in the worker VNet. |
| subnetPrefix | A canonical private IPv4 prefix inside that VNet. |
| securityGroup | A separate NSG attached only to the endpoint subnet. |
| registry | The registry resource identifier matching Config.Registry. |
| registryEndpoint | The selected worker registry private endpoint. |
| registryNic | The exact NIC owned by that endpoint. |
| exchangeEnvironment | A managed environment distinct from worker and controller environments. |
| exchangeEndpoint | The selected worker exchange private endpoint. |
| exchangeNic | The exact NIC owned by that endpoint. |

Resource identifiers use the selected subscription and explicit resource groups.
Endpoint and NIC pairs must be distinct.
The endpoint subnet cannot overlap worker or firewall subnets.
Its prefix must provide at least three usable addresses after Azure reservations.
All three selected DNS addresses must belong to that subnet.
The first four and last subnet addresses cannot serve endpoint roles.

SubnetDefinition and SecurityGroupDefinition return fresh validated SDK values for later packaging.
They issue no write request. Caller mutations cannot change the guard.
The final network profile must include these settings before execution enablement.

## Endpoint and address ownership

Both endpoint resources must report the expected identifier, name, type, worker region, and successful provisioning.
Each endpoint must identify exactly its selected NIC and endpoint subnet.
Each has one automatic private-link connection to the selected service and group.
Manual connections, extra services, extra NICs, application security groups, and non-IPv4 endpoint modes reject inspection.

The registry connection targets the configured Microsoft.ContainerRegistry registry with group registry.
Its NIC has exactly two IPv4 configurations.
Member registry owns the login address. Member registry_data_REGION owns the selected regional data address.
The explicit registry data region can differ from the worker region.
Swapping login and data addresses or selecting another regional member rejects inspection.

The exchange connection targets the selected Microsoft.App managed environment with group managedEnvironment.
Its NIC has exactly one IPv4 configuration with the exchange address and matching group.
The single exchange member name must be present. This stage does not impose an undocumented literal member name.
Service-side required-member discovery and the isolated exchange application's hostname binding remain separate checks.

The consumer-side connection must report Approved with no outstanding action.
Pending approval, missing state, unfinished provisioning, and required updates leave admission unavailable.
Rejected or removed connections reject ownership.
Consumer-side approval alone does not prove the service-side connection inventory.

NIC identity, private-endpoint backlink, IP configuration child identity, group, member, address, and subnet must agree.
IPv6, prefix allocation, public IPs, load balancers, forwarding, taps, VM attachments, and alternate networking reject inspection.
Inactive None values for NIC auxiliary and migration fields do not change the boundary.
Active extensions and custom NIC DNS settings reject inspection.
Observed addresses can use dynamic or static allocation.
Optional static endpoint declarations must agree with the observed group, role, and address when present.

Endpoint custom-DNS metadata and NIC FQDN metadata do not establish the configured exchange or registry authority.
Actual selected-zone records provide that address mapping.
Service configuration and hostname ownership still require separate inspection.

## Endpoint subnet policy

The subnet must report the configured prefix, explicit disabled default outbound access, and the selected NSG.
Private endpoint and private link service network policies must report Enabled.
NAT, route tables, delegations, service endpoints, service policies, sharing, and unrelated subnet attachments reject inspection.
The subnet must report exactly the two selected endpoints and all three observed NIC IP configuration references.
Missing inventory remains unavailable. Foreign or duplicate references reject ownership.

The NSG must report successful provisioning, FlushConnection enabled, one endpoint subnet attachment, and no direct NIC attachments.
Its complete custom rule set contains exactly these rules:

| Direction | Priority | Behavior |
| --- | --- | --- |
| Inbound | 100 | Allow worker subnet TCP port 443 to the three selected endpoint addresses. |
| Inbound | 4096 | Deny all other incoming traffic. |
| Outbound | 4096 | Deny initiated outbound traffic. |

The shared rule comparator checks every traffic-bearing field.
Rule order, endpoint address order, and descriptions do not alter the expected policy.
Explicit deny rules override permissive platform default NSG rules.
Stateful response traffic remains subject to Azure's NSG behavior.
Packet behavior and native service compatibility require separate deployment verification.

This endpoint subnet is exclusive to the worker path.
Trusted preparation needs separate private registry access and DNS resources in trusted networking.
Do not add trusted-controller exceptions to this NSG or peer worker and controller VNets.

## SDK boundary

SDKEndpointReader uses the pinned armnetwork/v8 8.0.0 package and management API 2025-03-01.
It requires an explicit TokenCredential.
Its transport permits GET only on the two selected endpoints, two NICs, endpoint subnet, and endpoint NSG.
Wrong resource kinds, other services, collection discovery, child discovery, mutations, extra queries, and paging reject before transport.
The client does not follow provider redirects, register resource providers, or retry failed reads automatically.
Each SDK operation has a five-second context and four-megabyte response bound.
Provider diagnostics become fixed control errors.

The six new reads run concurrently after all prior stages succeed.
A normal complete check makes thirty-one reads. Prior-stage paging can increase the bound to forty-two reads.
The complete guard has a twenty-five-second context and no successful-result cache.
These observations do not prevent later resource changes. Attachment authority and effective-state checks remain required.

The controller needs read access to the selected endpoints, interfaces, subnet, and NSG.
This stage needs no registry token, secret, worker identity, endpoint approval, or network write permission.
Close the reader to close idle transport connections.

## Targeted local verification

Seven new focused race checks cover ownership, fresh reads, prior-stage failures, address roles, configuration, reserved addresses, and SDK scope.
They cover foreign services, changed data regions, pending approval, public attachments, permissive rules, missing inventory, and inactive NIC defaults.
Two selected existing perimeter checks verify the extracted shared NSG comparator.
No unchanged broad suite, UI build, browser, container, image build, registry push, deployment, or live cloud call runs.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureEndpoints' -count=1
```

The retained targeted command results and output/azure-worker-endpoints/summary.json record local evidence.
Acceptance 51 advances. The total remains 48 verified items out of 57.

## Next implementation

Inspect actual registry service configuration, regional data endpoints, and service-side private endpoint approval.
Inspect exchange environment configuration, service-side approval, private application inventory, and HTTPS hostname authority.
Resolve effective VNet-wide DNS visibility and attachment controls.
Establish cross-job isolation and execution placement before composing Network.
Then add complete profile binding, native role configuration, packaging, and separate deployment verification.

## Primary references

[Registry private endpoints](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-private-endpoints) documents login and regional NIC member addresses.
[Container Apps private endpoints](https://learn.microsoft.com/en-us/azure/container-apps/private-endpoints-with-dns) documents the managedEnvironment group and custom-domain private addressing.
[Private endpoint network policies](https://learn.microsoft.com/en-us/azure/private-link/disable-private-endpoint-network-policy) documents subnet policy requirements for NSG enforcement.
The pinned network SDK defines the inspected resource and attachment fields.
