# Azure relay VNet observation

## Scope

RelayTopologyGuard observes the selected relay VNet after [backend endpoint observation](azure-worker-backend-endpoint.md).
It binds the reported address space, DNS settings, peering, and embedded subnet inventory to explicit installation settings.
It reuses the endpoint stage's validated relay and backend endpoint subnets. It adds one VNet GET and no duplicate subnet reads.

This stage does not implement Network, enable coded jobs, or authorize dispatch.
It checks reported configuration. It sends no backend packet, DNS query, or TLS probe.
Acceptance 51 advances. The verified count remains 48 of 57, with nine open.

## Explicit settings

RelayTopologySettings requires these values:

| Field | Requirement |
| --- | --- |
| backendEndpoint | Exact BackendEndpointSettings used by the preceding guard. |
| virtualNetworkPrefix | One canonical private IPv4 address space containing both selected relay and backend endpoint subnet prefixes. |

The address space must not overlap the configured worker VNet.
Public ranges, mapped IPv6, host bits, and prefixes that exclude either selected subnet reject configuration.
The guard binds the complete supplied Config and parent settings. Caller substitution cannot change its selected resources.
The existing exchange settings require separate worker, controller, and relay VNet identifiers.
This stage does not establish nonoverlap with every other installed network.

## Reported VNet and subnet requirements

The VNet must report its selected identifier, name, resource type, region, and successful provisioning.
Its address space must equal the single configured prefix. Extended locations reject observation.
Peering, custom DHCP DNS servers, default public NAT gateways, BGP communities, and external IP allocations reject observation.

The embedded subnet array must contain between two and 256 entries.
Each entry must identify a unique child of the selected VNet and report successful provisioning.
Each subnet must report one canonical private IPv4 prefix within the VNet. Subnet prefixes must not overlap.
Missing inventories and unfinished provisioning remain unavailable. Foreign or duplicate identifiers reject observation.

Both selected subnets must match their direct observations from the preceding endpoint check.
The comparison includes reported bindings and attachments. Missing selected entries or conflicting observations remain unavailable.
No successful observation is cached or persisted. Every check repeats the parent checks and selected VNet read.

Additional bounded subnets can support later platform and registry paths.
Their presence does not approve their workloads, attachment authority, security rules, or traffic paths.
The [relay perimeter observer](azure-relay-perimeter.md) now binds the source NSG and route table without repeating these subnet reads.
Complete firewall, effective DNS, identity grants, and workload isolation remain required before native enablement.

## Read boundary and availability

SDKRelayTopologyReader requires an explicit credential. It permits only the selected VNet GET through armnetwork/v8 and API 2025-03-01.
The shared transport rejects writes, actions, subnet reads, collections, expansions, paging, alternate API versions, and foreign hosts.
Provider registration, retries, secret reads, and credential discovery remain disabled.
The read has a five-second context and four-megabyte response limit. Redirects and unavailable resources cannot establish approval.

The composed guard has a 55-second context, 58 normal reads, and at most 111 reads with bounded parent paging.
The endpoint stage still issues its existing seven reads. The backend stage still issues its existing five reads.
The transient subnet observations exist only during the current composed call.

## Targeted local verification

Four new focused race checks cover fresh state, parent reuse, alternate paths, inventory bounds, prefix conflicts, uncertain state, and SDK scope.
Two existing endpoint checks cover the changed observation return path and read-scope behavior.
Fixtures use explicit fake credentials and in-memory SDK responses. They open no network connection or listener.
No cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureRelayTopology.*|TestAzureBackendEndpointBindsFreshApprovalWithoutAnotherEnvironmentRead|TestAzureBackendEndpointSDKPreservesUncertaintyAndReadScope)$' -count=1
```

Evidence is output/azure-relay-topology/summary.json and the targeted command result.
Continue relay firewall controls, effective DNS visibility, attachment authority, and complete native admission before enabling Azure coded execution.
