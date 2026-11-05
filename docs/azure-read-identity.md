# Azure observer identity and selected read grant

## Current boundary

ReadIdentityGuard binds one observer identity to its reported principal, tenant, client identifier, subscription assignment, and custom role.
It follows the [selected network and DNS deny observer](azure-network-deny.md). Each inspection refreshes that complete preceding chain once.
The observer identity remains separate from worker pull, backend pull, backend startup, and backend runtime identities.

This stage implements no Network and enables no native Azure coded execution.
It observes the selected grant. It does not prove effective authorization, complete visibility, or absence of other grants.
The SDK readers still require an explicit bootstrap credential. This stage does not prove that credential uses the selected observer identity.
Identity attachment, token availability, deny coverage, protection-management authority, and cross-scope visibility remain installation requirements.
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.

## Explicit settings

ReadIdentitySettings requires these fields:

| Field | Requirement |
| --- | --- |
| protection | Complete preceding NetworkDenySettings. |
| identity | Selected user-assigned identity resource identifier in the configured subscription. |
| clientId | Canonical, nonzero client GUID. |
| principalId | Canonical, nonzero service principal GUID, distinct from clientId. |
| tenantId | Canonical, nonzero tenant GUID. |
| assignment | Selected subscription-root role assignment identifier with a GUID name. |
| role | Selected subscription-root role definition identifier with a GUID name. |

The identity resource must differ from the four known pull and backend identity resources.
Its observed name, resource type, location, identifier, client, principal, and tenant must match the selected configuration.
The assignment must report that principal, ServicePrincipal type, selected role, and exact subscription scope.
Assignment conditions, condition versions, and delegated managed identity references are rejected.
Missing required identity or grant metadata remains unavailable. A conflicting identity or policy rejects ownership.

## Fixed selected role

The role must report CustomRole type and exactly the selected subscription as its assignable scope.
One permission entry permits exactly these actions:

- */read
- Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action

The second action supports the existing bodyless DNS-association observation.
The role has no exclusions, data permissions, permission conditions, write actions, join actions, or role-management actions.
Action order and letter case do not change semantics. Duplicate actions reject observation.
The cosmetic role name does not determine authority.

RoleDefinition and AssignmentDefinition return fresh SDK values for installation tooling.
They perform no creation, assignment, update, or other cloud operation.
The assignment definition omits response-only scope metadata. Installation tooling uses the selected assignment identifier to select its scope.
Microsoft documents [role definitions](https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions) and [role assignments](https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments).

## Credential and SDK boundary

Credential performs one complete inspection and constructs ManagedIdentityCredential with the verified, observed client identifier.
It provides no default-credential fallback. This method does not request or redeem a token.
SDK calls made later with that credential can request tokens. Host identity attachment and successful authentication require separate verification.
The [managed identity credential API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity#ManagedIdentityCredentialOptions) defines explicit user-assigned identity selection.

SDKReadIdentityReader requires an explicit credential and permits only three selected GET paths.
The identity reader pins armmsi v1.3.0 and API 2024-11-30.
The existing armauthorization/v3 v3.0.0-beta.3 supplies role assignment API 2022-04-01 and role definition API 2022-05-01-preview.
The scoped role Get method preserves the exact query. GetByID adds an SDK disambiguation query outside this transport contract.
The shared transport rejects mutations, lists, alternate resources, extra query parameters, secret operations, and redirects.
Provider registration and retries remain disabled. Each selected read has a five-second context and four-megabyte response limit.

Three selected reads run concurrently after protection succeeds. Parent observations do not repeat within one inspection.
The composed guard has a 125-second context and at most 2196 requests, including the bounded reported-subscription scan.
Normal additional cost is exactly three GETs. Successful observations are not cached or persisted.
Credential construction uses the identity from that same inspection.
Returned errors use fixed public categories and omit provider messages, condition text, and credential factory details.

## Targeted local verification

Five new focused race checks and one selected existing protection regression pass in 4.003 seconds.
They cover fresh grants, parent gating, changed principals, foreign scopes, conditions, extra authority, exact SDK paths, unknown metadata, and settings isolation.
A private credential factory spy checks the verified client selection without constructing or requesting a real managed identity token.
The initial targeted command exposes the GetByID query mismatch. The scoped Get correction preserves the transport boundary and passes the same checks.
No unchanged broad suite repeats.
Fixtures use explicit fake credentials and in-memory SDK transports. They open no network connection or listener.
No cloud request, UI build, browser, image build, container, registry push, deployment, or demo restart runs.

```sh
go test -race ./adapters/execution/azure -run '^(TestAzureReadIdentity.*|TestAzureNetworkDenyRequiresFreshEnforcedProtectionAfterInventory)$' -count=1
```

Evidence is output/azure-read-identity/summary.json and the targeted command results.
Next bind the observer credential to actual observation clients and verify effective access rather than inferring it from one selected grant.
Keep clean creation provenance, existing cross-scope visibility, deployed protection, attachment controls, preparation networking, and execution isolation in the native admission scope.
