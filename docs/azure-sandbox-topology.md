# Azure sandbox VNet and firewall admission

## Introduction

OpenRuntime now includes SDKTopology in creation, image import, and live admission.
It checks the selected VNet, embedded subnet contents, default-route firewall, public IP attachment, and complete firewall rule-group inventory.
It uses the existing metadata identity, bounded ARM transport, and positive evidence cache.
Existing firewall resource and policy verifiers are reused through independent bindings.
Ordinary Jobs configuration and its platform allow rules are not required.

This verifies selected reported topology and firewall policy. It does not prove effective routing or deployed traffic enforcement.
Native commands and deployment templates still do not enable sandbox execution.

## Concepts

### Settings and ownership

placement.network.topology requires these fields:

| Field | Contract |
| --- | --- |
| prefix | One canonical private IPv4 prefix containing the execution, endpoint, and firewall subnets. |
| firewall | Exact selected Azure Firewall resource in the subscription. |
| policy | Exact selected firewall policy resource in the subscription. |
| publicIp | Exact selected public IP resource attached only to the firewall's egress configuration. |
| firewallPrefix | Canonical private IPv4 prefix for AzureFirewallSubnet, with a size of /26 or larger. |

The firewall subnet belongs to the sandbox connection's VNet.
It cannot overlap the execution or endpoint subnet.
placement.network.firewallIp must be a usable address in that subnet, outside Azure's reserved addresses.
Invalid scope, resource type, address space, overlap, subnet size, or firewall address rejects before credential construction.
The sandbox-placement/v4 profile binds these settings with the existing network, private endpoint, DNS, and placement bindings.
Older staged profiles require regeneration. No native command currently loads them.

TopologyDefinitions returns five reviewable bodies: firewall, firewall policy, owned rule group, public IP, and firewall subnet.
It makes no service request or mutation. Complete deployment packaging remains unfinished.

### VNet contract

The VNet must match its identifier, name, type, region, and successful provisioning state.
Its address space contains exactly the configured prefix.
Peerings, custom DHCP DNS servers, default public NAT, IP allocations, BGP communities, and extended locations reject.
It contains exactly the execution, endpoint, and firewall subnets.
Each embedded subnet must match a fresh selected subnet observation.
This prevents a selected subnet identifier from hiding different embedded network contents or additional gateway subnets.

The firewall subnet has the selected prefix and disabled default outbound access.
It has no route table, security group, NAT gateway, delegation, service endpoint, private endpoint, or unrelated allocation.
Its sole IP configuration reference belongs to the firewall's egress configuration.
Unrelated configuration profiles, navigation links, and service associations reject.
The independent placement and private stages still verify execution and endpoint subnet policy.

### Firewall and public IP contract

The firewall must match its identifier, name, type, region, and successful provisioning state.
It uses the Standard AZFW_VNet SKU, threat intelligence in Deny mode, and exactly the selected policy.
Its sole egress configuration attaches the selected firewall subnet and public IP.
Its reported private address must equal placement.network.firewallIp and therefore the checked default-route next hop.
Additional inline rules, alternate configurations, virtual hubs, management configurations, and different policy contents reject.

The public IP uses Standard Regional SKU, static IPv4 allocation, and a reported public IPv4 address.
Its attachment points to the exact selected firewall egress configuration.
NAT gateways, linked addresses, service addresses, prefixes, and unrelated IP tags reject.

### Policy and complete rule groups

The firewall policy uses Standard SKU, Deny threat intelligence, disabled DNS proxy, and the fixed private-range SNAT contract.
It attaches only to the selected firewall and has no parent policy, child policy, or managed identity.
Its reported rule-group references must match the complete bounded inventory.

The owned sandbox-deny group contains exactly one filter collection and one network rule.
The action is Deny. Source addresses, destination addresses, and destination ports are each the wildcard; protocol is Any.
The group priority is 400 and its collection priority is 100.
No platform, registry, identity, application, network, or NAT allow rule enters this policy.

The three known service default groups may also appear, but only at their fixed priorities and with empty rule collections.
Unknown groups, nonempty defaults, changed priorities, broader or partial rules, and missing or duplicate references reject.
The three defaults may span four inventory pages with the owned group.
All four groups must be fully observed before admission succeeds.

Private endpoint traffic uses the VNet's local routing and the independently checked endpoint rules.
The firewall policy denies traffic that reaches it through the configured default route.
Actual preview-service traffic behavior and compatibility remain separate deployment work.

### Observation and lifecycle

Initial creation and image import each obtain fresh observations of seven resources and the selected rule-group collection.
All paths use API version 2025-03-01.
The observation has a five-second bound and joins its selected reads.
Rule-group inventory allows at most four items across four pages.
The shared inventory guard enforces exact host, collection path, version, bounded skip tokens, duplicate detection, and complete pagination.
It returns no partial evidence after a missing page, error, cycle, or limit breach.

Successful evidence can supply concurrent live callers for at most 15 seconds from observation start.
Failed current full observations invalidate previous success. Failures are not cached.
Canceled or closed readers return unavailable. Metadata absence never establishes sandbox absence.
Remaining protection callbacks, per-sandbox definitions, and current worker authorization remain uncached.
The shared reader disables redirects, automatic retries, provider registration, and mutations.
It bounds responses to 1 MiB and omits provider and credential diagnostics from returned errors.
Runtime.Close closes the topology reader after joining owned command work.

## Tutorials

### Verify the changed boundary locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxTopology|^TestSandboxRuntimeTopology' -count=1 -timeout=40s
```

Six new focused checks cover deny-all definitions, topology and policy drift, invalid settings, four-page inventory, bounded evidence, and automatic runtime composition.
Seventeen selected native checks pass with race detection in 2.208 seconds.
Three affected shared firewall checks pass with race detection in 3.120 seconds.
The shared checks cover both existing worker and relay policies after the verifier extraction.
Evidence is output/azure-sandbox-topology/summary.json, native.log, and shared-firewall.log.
The checks use local SQLite, fake credentials, and in-memory ARM responses.
No cloud service, browser, container, UI build, demo restart, resource mutation, or remote push is involved.

### Complete activation

Complete resolver policy and other linked-zone observations, trusted exchange service ownership, additional grants, deny effects, and credential host attachment.
Then wire scoped execution and discovery commands and package their dependencies.
Verify actual routing, preview compatibility, private conversion, role acceptance, timer execution, token audience, and proxy trust during separate deployment acceptance.

Primary schemas: [Microsoft Azure Firewall](https://learn.microsoft.com/en-us/azure/templates/microsoft.network/2025-03-01/azurefirewalls)
and [Microsoft firewall rule groups](https://learn.microsoft.com/en-us/azure/templates/microsoft.network/2025-03-01/firewallpolicies/rulecollectiongroups).
Related guides: [connection rules and routes](azure-sandbox-network.md), [private bindings](azure-sandbox-private.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.
