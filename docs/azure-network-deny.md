# Azure selected network and DNS deny observation

## Scope

NetworkDenyGuard observes five selected deny assignments after [reported private DNS inventory](azure-private-dns-inventory.md).
It checks protection metadata for the worker VNet, relay VNet, and their three selected private DNS zones.
VNet protection alone does not protect records in an already-linked zone. The selected zones therefore require independent protection.
The check reads actual assignment resources. A deployment option or installation declaration cannot substitute for those observations.

This stage implements no Network, authorizes no dispatch, and enables no Azure coded execution.
It does not create assignments, test a forbidden mutation, or establish deployed enforcement and platform compatibility.
Existing cross-scope links, clean installation provenance, effective read coverage, stack-management authority, and environment attachment controls remain unproven.
Other perimeter resources, trusted preparation networking, execution isolation, and complete native wiring remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Explicit resource bindings

NetworkDenySettings requires the exact preceding RelayPrivateDNSSettings and these assignment identifiers:

| Field | Protected resource |
| --- | --- |
| workerAssignment | Selected worker VNet. |
| relayAssignment | Selected relay VNet. |
| workerExchangeZoneAssignment | Selected worker exchange private zone. |
| workerRegistryZoneAssignment | Selected worker registry private zone. |
| relayRegistryZoneAssignment | Selected relay registry private zone. |

Every identifier must name a distinct resource-scoped Microsoft.Authorization/denyAssignments extension with a valid GUID name.
Its parent must equal that exact protected resource. A broader subscription or resource-group assignment cannot substitute in this profile.
All protected resources derive from the preceding configuration. There is no scope, principal, condition, or exception override.
Constructor validation preserves canonical resource ownership and the complete parent identity.

The observed assignment must report its selected identity, name, resource type, and exact protected scope.
Required enforcement, child-scope, protection, permission, and principal fields must be available.
Missing required state remains unavailable. Foreign identity, scope substitution, or a conflicting policy rejects ownership.

## Fixed protection contract

Each assignment must report these properties:

- denyAssignmentEffect is enforced. Audit mode cannot establish protection.
- isSystemProtected is explicitly true.
- doNotApplyToChildScopes is explicitly false.
- Exactly one SystemDefined principal uses the all-principals zero GUID.
- There are no excluded principals or groups.
- Assignment conditions and condition versions are empty.
- Exactly one permission entry denies all management actions through the literal star action.
- Permission conditions and condition versions are empty. Data permission fields are empty.

The permission entry contains exactly these three notActions exceptions:

| Exception | Purpose |
| --- | --- |
| */read | Retain management reads. |
| Microsoft.Authorization/locks/delete | Retain the fixed lock-removal exception used by the selected protection profile. |
| Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action | Retain the existing bodyless DNS-association inventory operation. |

No network join, network write, zone write, record write, wildcard provider action, or additional exception is permitted.
Exception order and action letter case do not change semantics. Duplicate exceptions reject observation.
The DNS-association exception adds no zone mutation authority.
These denies do not grant any permission. The reader's grants remain a separate installation requirement.

The [Microsoft deny-assignment model](https://learn.microsoft.com/en-us/azure/templates/microsoft.authorization/2024-07-01-preview/denyassignments) defines enforcement mode, conditions, principals, and permission fields.
The [SDK model documentation](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v3) describes the corresponding typed response.

## Pinned preview read model and request boundary

The reader pins armauthorization/v3 v3.0.0-beta.3 in [go.mod](../go.mod) and uses API 2024-07-01-preview.
The older stable model omits assignment-level conditions and explicit enforcement mode.
This observer requires those fields and provides no fallback that assumes an omitted mode is enforced.
Preview API availability and complete service response shape require separate hosted verification before native adoption.
The dependency change adds only this pinned module and its checksum entries. Existing dependency versions remain unchanged.

SDKNetworkDenyReader requires an explicit credential and permits GET on only the five selected assignment identifiers.
The shared transport rejects mutations, assignment lists, role or permission discovery, secret actions, alternate APIs, filters, foreign resources, and redirects.
Retries and provider registration remain disabled. Each read has a five-second context and four-megabyte response limit.
The controller requires denyAssignments/read on these selected resources. The reader requires no create, delete, role-management, or protection-management authority.

Five reads run concurrently after the existing chain succeeds. No parent VNet, zone, registry, or inventory read is repeated.
The composed guard has a 120-second context and a maximum of 2193 reads including the bounded subscription scan.
Normal additional cost is exactly five assignment GETs. No successful observation is cached or persisted.

## Targeted local verification and next work

Four new focused race checks pass for the complete five-resource contract in 3.831 seconds.
Two selected existing inventory regression checks also pass in the earlier targeted command.
Only the changed deny checks repeat when review extends protection from VNets to the selected zones.
The checks cover fresh state, parent gating, audit mode, scope changes, excluded groups, conditions, child-scope bypasses, join exceptions, record-write exceptions, and unknown metadata.
They also verify action ordering, case semantics, exact SDK paths, configuration binding, unavailable reads, and redirects.
An initial compile attempt corrects the SDK's value-shaped response handling before checks pass.
Fixtures use explicit fake credentials and in-memory responses. They open no network connection or listener.
No broad suite, cloud request, browser, UI build, image build, container, registry push, deployment, or demo restart runs.
External requests read only public SDK metadata and Microsoft documentation.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureNetworkDeny' -count=1
```

Evidence is output/azure-network-deny/summary.json and the targeted command results.
Verify actual resource-scoped assignment provisioning and enforcement before installation uses this contract.
Establish authoritative existing cross-scope visibility or clean creation provenance. These assignments do not remove an existing foreign zone link.
Inspect who can change protection through its owning installation or deployment-stack workflow. The reported system-protection flag alone does not prove that authority boundary.
Continue environment attachment control, other perimeter protection, effective read coverage, trusted preparation networking, and complete native admission.
