# Azure sandbox identities and selected authority

## Introduction

SDKAuthority checks the configured control, registry inspection, metadata, and timer-stop identities and their selected Azure role assignments.
OpenRuntime now includes this stage after placement admission and before the remaining protection checks.
The factory constructs it with the explicit metadata credential. Callers cannot omit the stage.

These are reported identity and selected-grant checks. They do not establish complete effective authorization or exclude other assignments.
Host attachment, identity reuse, inherited authority, deny effects, and actual service acceptance still need verification.
Native commands and deployment templates remain disabled for sandbox execution.

## Concepts

### Settings

Settings now requires authority alongside placement.
The control, registry inspection, metadata, import, and timer-stop identities must have distinct resources, clients, and principals.
All identity resources belong to the configured subscription. Reported tenant and region must match the installation.
Invalid bindings reject before credential construction.

| authority field | Required binding |
| --- | --- |
| control, registry, metadata | Each contains resource and principalId. Its client comes from the matching top-level client setting. |
| stopClientId and stopPrincipalId | Exact identifiers for timer.stopIdentity. |
| controlGrant | Custom role and exact assignment at the sandbox group for the control principal. |
| stopGrant | Custom stop-only role and exact assignment at the same group for the timer principal. |
| workflowGrant | Custom workflow administration role assigned to the controller at the dedicated timer resource group. |
| assignGrant | Custom identity attachment role assigned to the controller at exactly timer.stopIdentity. |
| metadataGrant | Custom metadata read role assigned to the metadata principal at the selected subscription. |
| registryAssignment | Registry-scoped AcrPull assignment for the registry inspection principal. It differs from the import assignment. |

Each custom grant contains assignment and role resource identifiers.
Role definitions belong to the selected subscription and have distinct identifiers.
Assignments must use their exact expected scope. A broader parent assignment does not substitute for the selected grant.
Grant conditions, delegated identities, a different principal, another role, or an incorrect principal type reject.

### Fixed profiles

The implementation compares each custom role's name, type, assignable scope, actions, data actions, and exclusions against a fixed profile.
Additional permissions, wildcard substitutions, duplicate permissions, conditions, and exceptions reject.
Sandbox operations belong in dataActions. Workflow and identity administration belong in actions.

| Profile | Permitted authority |
| --- | --- |
| Execution controller | Read converted images; read, create, delete, execute the supervised command, and stop sandboxes. Group scope only. |
| Discovery controller | Execution controller permissions plus image import. No image deletion or other sandbox features. |
| Timer stop | Read and stop sandboxes. No create, command execution, import, resume, policy changes, ports, secrets, or file access. |
| Workflow administration | Read, write, and delete workflows in the dedicated timer resource group. No broad Logic App Contributor role. |
| Timer identity attachment | Read and assign the selected timer identity. The assignment targets that identity, not its resource group. |
| Metadata reader | The existing fixed metadata contract: read management metadata and observe DNS resolver policy associations. No data actions or write authority. |

The metadata read profile has subscription scope because installation resources span several resource groups.
The SDK transport still permits only its configured selected GET paths.
The selected registry inspection assignment uses AcrPull, not AcrPush.
The existing registry client also keeps publication disabled and requests pull tokens only.
These selected grants do not prove absence of other effective write privileges.

### Reviewable definitions

Settings.AuthorityDefinitions returns five canonical custom-role definitions and six expected assignment objects.
The same profiles drive verification and reviewable packaging inputs.
This method changes no Azure resource and acquires no credential.
Its objects include expected resource and readback metadata. They are not a complete deployment template.
Deployment packaging must place each definition and assignment at the corresponding ARM scope.
Use separate controller role identifiers for execution and discovery. Their permissions and role names differ.

The independent timer definition attaches only timer.stopIdentity.
The controller's identity-assignment grant targets only that identity.
Even that narrow grant does not prove the identity is absent from other hosts. Actual placement protection remains required.

### Observations and reuse

One authority observation reads fifteen selected resources in parallel:

- Four user-assigned identity resources.
- Five custom role definitions.
- Six exact role assignments.

The import identity and built-in AcrPull definition remain in the separate placement stage.
Identity reads use 2024-11-30. Role definitions use 2022-05-01-preview. Assignments use 2022-04-01.
The shared bounded ARM reader permits these exact GET paths and versions only.
It does not list assignments, mutate roles, follow redirects, register providers, or obtain unrelated metadata.
Missing or inaccessible authority returns unavailable. It never establishes worker absence or authorizes another launch.

Successful authority evidence is shared for at most 15 seconds from observation start.
Concurrent callers share a successful refresh. Failures create no cached success.
Cancellation and closure reject even when earlier evidence remains within the window.
This is a selected Azure metadata window, not a TestStudio authorization cache.
Worker permission checks, command reservations, revocation, per-sandbox definition checks, and the remaining protection callback stay uncached.
Azure also authorizes each actual data-plane and workflow request. Metadata success cannot replace service acceptance.
The placement and authority stages share one bounded-cache implementation and one scoped ARM reader implementation.

## Tutorials

### Verify the boundary locally

```sh
go test -race ./adapters/execution/azuresandbox -run '^TestSandboxAuthority|^TestSandboxRuntimeAuthority' -count=1 -timeout=35s
```

Five new checks cover canonical permissions, mode separation, changed identities and scopes, role broadening, malformed bindings, cache bounds, and factory composition.
They verify a stop identity cannot execute commands and an execution controller cannot import images.
Twelve concurrent callers share one fifteen-resource observation.
Missing metadata, cancellation, closure, and a changed uncached protection result remain explicit failures.
Affected placement, runtime, shared execution, and command authorization checks also pass.
Evidence is output/azure-sandbox-authority/summary.json and targeted.log.
These checks use local SQLite, private filesystem evidence, fake credentials, and in-memory ARM transports.
No cloud account, identity endpoint, registry service, role mutation, browser, container, demo restart, or remote push is involved.

### Complete deployment admission

Verify effective assignments and exclusions, deny effects, identity host attachment, and protection from unintended identity reuse.
Complete private exchange ownership, security-rule contents, route contents, DNS, and protected network bindings.
Then wire the execution and discovery roles to native commands and package their resources.
Verify custom sandbox data-action roles and the narrow timer roles with the actual service during the separate deployment phase.
The published operation catalog supplies permission names. It does not prove preview-service compatibility or effective installation access.
Keep registry conversion routing, token audience, workflow execution, and proxy trust in that separate verification scope.

Primary contracts: [Microsoft sandbox permissions](https://learn.microsoft.com/en-us/azure/role-based-access-control/permissions/compute) and [Logic App role definitions](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/integration).
Related implementation: [placement](azure-sandbox-placement.md), [runtime](azure-sandbox-runtime.md), and [independent timers](azure-sandbox-deadlines.md).
Acceptance 51 advances. The verified count remains 48 of 57, with nine open.
