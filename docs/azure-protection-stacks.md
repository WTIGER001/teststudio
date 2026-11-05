# Azure reported protection stacks

## Boundary

The identity observer now checks two selected deployment stacks alongside the selected deny assignments.
This checks reported policy and resource membership before credential construction and each bound inspection.
It implements no Network and enables no native Azure coded execution.

The stack API provides no exact deny-assignment identifier link.
Matching stack metadata therefore does not prove direct ownership of the selected deny assignments.
Names, descriptions, and creator metadata cannot supply that missing link.
Authority to change or delete the stacks also remains unproven.

## Required settings

[ReadIdentitySettings](azure-read-identity.md#explicit-settings) adds two mandatory fields:

| Field | Selected resource set |
| --- | --- |
| networkProtectionStack | Worker VNet, relay VNet, worker exchange zone, worker registry zone, and relay registry zone. |
| identityProtectionStack | The selected observer user-assigned identity. |

The stack identifiers must differ and belong to the configured subscription.
Each identifier names a subscription-scoped or resource-group-scoped Microsoft.Resources/deploymentStacks resource.
Names follow the existing canonical resource-name rules. Each stack scope must contain all its selected resources.
A resource-group stack cannot cover roots in another resource group. Use a subscription stack for those roots.
Reported identifiers, names, resource types, and locations must match configuration. Location matches Config.Region.

This profile requires exactly the five network roots in one stack and the identity root in the other.
Extra managed resources, duplicate roots, extension references, and alternate identifier forms reject observation.
Generic existing stacks with additional managed resources do not satisfy this dedicated-root profile.

## Reported policy

Each stack must report Succeeded provisioning, denyWriteAndDelete, and applyToChildScopes set to true.
No excluded principal is permitted. Every required root must report Managed status and denyWriteAndDelete protection.
Missing metadata or roots, unknown status, failed resources, provider errors, and bypassStackOutOfSyncError block observation.
Conflicting resource identity or policy rejects ownership. An explicit deploymentScope must match the stack parent scope.

The effective identity exception set contains only */read and Microsoft.Authorization/locks/delete.
The network stack additionally permits Microsoft.Network/virtualNetworks/listDnsResolverPolicies/action.
Azure adds the two default exceptions automatically. The verifier accepts their implicit or explicit representation.
It normalizes letter case and duplicates before comparing the effective set. Additional exceptions reject observation.
The separately selected deny assignments still pass their existing enforcement, principal, scope, and exception checks.

Microsoft documents the reported fields and defaults in the [subscription GET contract](https://learn.microsoft.com/en-us/rest/api/resources/deployment-stacks/get-at-subscription?view=rest-resources-2025-07-01).
The [resource-group GET contract](https://learn.microsoft.com/en-us/rest/api/resources/deployment-stacks/get-at-resource-group?view=rest-resources-2025-07-01) defines the alternate selected scope.

## SDK and observation limits

The reader pins armdeploymentstacks/v2 v2.0.0 and API 2025-07-01.
It permits exactly two selected stack GETs through the existing restricted transport.
No list, export, validation, what-if, mutation, alternate resource, redirect, or extra query is permitted.
Each GET has a five-second context and the shared four-megabyte response limit. Retries and provider registration remain disabled.

The reader returns only identity metadata, deny settings, deployment scope, provisioning state, and reduced managed-resource rows.
It drops templates, parameters, outputs, deployment links, descriptions, and system metadata before returning the observation.
Successful observations are not cached or persisted.

Seven identity-stage operations now run concurrently after the preceding protection chain.
The identity and bootstrap ceiling becomes 2200 requests within the existing 125-second context.
Read-deny sessions allow 2216 requests in 130 seconds. Host sessions allow 2222 requests in 135 seconds.
These are maximum bounds across the composed inventories. This change adds two normal GETs and no repeated parent scan.
The host session retains 22 readers. Binding still performs one separate bootstrap inspection.
Failed stack checks block credential construction and later host reads. Bound checks refresh state through the selected observer credential.

## Installation requirements and remaining work

The inspector creates no stack and attempts no prohibited write.
The optional core package supplies no protection arrangement or automatic stack references.
Installation tooling must establish supported protection, select both stack identifiers, and inspect the generated assignments.

Reported membership does not establish actual enforcement, resource-type support, complete visibility, or clean creation provenance.
It does not remove existing foreign consumers or prevent an authorized administrator from changing protection later.
Direct assignment ownership and authority over stack management require separate authoritative evidence.
Complete trusted host configuration, execution isolation, preparation networking, and native command wiring remain open.
See [Azure worker admission](azure-worker-network.md) for the broader boundary.

The [host authentication check](azure-observer-host-auth.md) accounts for the host session's additional selected GET.

The [controller route check](azure-observer-host-routes.md) adds another host-specific collection GET without changing bootstrap or diagnostic session limits.
