# Azure relay registry endpoint observation

## Scope

RelayRegistryEndpointGuard checks the selected relay registry endpoint after [relay DNS policy observation](azure-relay-dns.md).
It binds consumer approval, NIC ownership, registry and regional data addresses, dedicated subnet policy, and endpoint security rules.
The preceding registry service stage independently requires approval for this exact relay consumer.
The endpoint addresses must equal the relay source NSG's selected private registry destinations.

The observer reuses the current registry service checks and validated relay VNet observation.
It adds four selected resource reads without another registry collection, registry resource, or VNet read.
It implements no Network, authorizes no dispatch, and enables no Azure coded execution.
The [relay private DNS observer](azure-relay-private-dns.md) now checks records derived from these member addresses and an exclusive completed relay link.
Effective zone visibility, identity and attachment authority, preparation networking, and complete native wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit registry consumer contract

RegistrySettings now accepts optional relayEndpoint in addition to the required worker and preparation endpoint settings.
Without relayEndpoint, the complete service inventory must still contain exactly the two selected consumers.
With relayEndpoint, it must contain exactly three selected consumers: worker, preparation, and relay.
The relay identifier must belong to the installation subscription and cannot reuse the worker registry, worker exchange, or preparation endpoint.
Omitting this field does not authorize another consumer. Foreign, duplicate, disconnected, rejected, and additional connections still reject ownership.
Missing or pending approvals remain unavailable. The embedded registry inventory must independently agree with the complete collection.

The shared SDK bounds connection inventory to the configured consumer count and one additional page.
The original profile retains two entries and three pages. The explicit relay profile permits three entries and four pages.
Registry read paths, credential scope, private access requirements, required members, and selected data region remain unchanged.
No service connection is created or approved by these observers.

## Endpoint settings and address binding

RelayRegistryEndpointSettings requires these values:

| Field | Requirement |
| --- | --- |
| dns | Exact RelayDNSSettings used by the preceding guard. |
| endpoint | Exact nonempty relayEndpoint selected in the preceding registry settings. |
| interface | Dedicated endpoint NIC identifier. |
| subnet | Dedicated registry endpoint subnet in the relay VNet. |
| subnetPrefix | Canonical private IPv4 prefix with at least eight addresses. |
| securityGroup | Dedicated endpoint NSG identifier. |
| registryIp | Exact registry member address. |
| dataIp | Exact selected regional data member address. |

The subnet must lie within the relay VNet and avoid the source, backend endpoint, and firewall subnet prefixes.
It cannot reuse the source or backend endpoint subnet identifier.
Both private addresses must be unique and usable in that subnet. The first four addresses and final address cannot identify endpoint members.
Their sorted set must equal the preceding relay perimeter's complete registryEndpointIps set.
The selected data region comes from the preceding registry profile. The endpoint cannot add another region.

NIC and NSG identifiers must belong to the installation subscription and differ from the selected backend and worker counterparts.
The NSG also cannot reuse the relay source NSG. Settings cannot substitute the preparation endpoint as the relay consumer.
SecurityGroupDefinition and SubnetDefinition return fresh validated SDK values without resource writes.

## Consumer ownership and dedicated network policy

The endpoint must report successful provisioning, one selected NIC, one selected subnet, and one approved automatic connection.
The connection must target the selected registry service and registry group.
Manual connections, extra interfaces, other services, alternate groups, and unfinished update actions cannot establish approval.

The NIC must point back to the endpoint and contain exactly two successful private IPv4 configurations.
Their required member names must identify registry and registry_data_REGION from the preceding service profile.
Each must use its exact selected address and dedicated subnet.
Public IPs, forwarding, compute attachments, unrelated workload configuration, alternate members, and additional addresses reject observation.
The shared worker endpoint verifier remains canonical for these checks.

The subnet must contain exactly the selected endpoint and both observed NIC configuration references.
Private endpoint and private-link service network policies must explicitly be enabled. Default outbound access must explicitly be disabled.
The subnet must attach its selected NSG and contain no route table, NAT gateway, delegation, service endpoint, sharing, or unrelated allocation.
Its direct observation must match the current embedded relay VNet subnet. Missing or inconsistent observations remain unavailable.

The dedicated NSG attaches only to this subnet, attaches to no individual NIC, and enables FlushConnection.
It contains exactly three custom rules:

1. Allow inbound TCP 443 from the exact relay source prefix to only the registry and regional data addresses.
2. Deny all remaining inbound traffic before platform defaults.
3. Deny all remaining outbound traffic before platform defaults.

This establishes reported endpoint configuration. It does not prove effective DNS resolution, packet isolation, image-pull identity authority, or platform startup.
Independent worker, preparation, and relay networking remains mandatory.

## Read scope and targeted local verification

SDKRelayRegistryEndpointReader uses the shared scoped endpoint SDK with an explicit credential.
It permits GET only on the selected endpoint, NIC, subnet, and NSG through armnetwork/v8 and API 2025-03-01.
It permits no registry collection reread, credential action, write, child action, paging, expansion, or foreign resource.
Individual operations have five-second contexts and four-megabyte response limits. Retries and provider registration remain disabled.
The four reads run concurrently after the full parent chain succeeds.
The explicit three-consumer composed profile has an 80-second context, 75 normal reads, and at most 134 reads with bounded paging.
The optional consumer adds one maximum parent page. It adds no normal parent read.
No successful observation is cached or persisted.

Six new focused race checks and four selected existing checks pass locally in 4.395 seconds.
They cover fresh service approval, member and address ownership, parent reuse, subnet consistency, policy bypasses, SDK scope, and immutable definitions.
They also cover explicit three-consumer approval, missing or unknown consumers, aliases, and retained two-consumer rejection behavior.
Only two changed checks repeat after review strengthens reserved-address coverage and the explicit four-page limit. They pass in 1.623 seconds.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureRelayRegistryEndpoint.*|TestAzureRegistryServiceSupportsOnlyExplicitRelayConsumer|TestAzureRegistryRelayConsumerSettingsRejectAliases|TestAzureRegistryServiceRejectsForeignOrUnapprovedConnections|TestAzureRegistryServicePreservesIncompleteStateAndBoundedPaging|TestAzureRegistryServiceSDKRejectsCredentialsMutationsAndForeignScope|TestAzureRelayDNSBindsFreshVNetAssociationWithoutParentRereads)$' -count=1
```

Evidence is output/azure-relay-registry-endpoint/summary.json and targeted command results.
Continue effective VNet-wide zone visibility, attachment authority, and trusted preparation networking before native admission and image-pull wiring.
