# First Azure installation

## Delivery direction

On 2026-11-08 the user selects a direct installation into their own Azure subscription as the first Azure delivery.
Keep a future Azure Managed Application in a customer's subscription viable.
Marketplace publication, offer configuration, licensing integration, and billing are deferred.
AKS is outside this profile. Preserve the separate Kubernetes distribution.

This plan records scope. It establishes no deployed Azure acceptance.
The tenant, subscription, region, installation name, identity provider, and monthly budget remain unselected.
Do not infer an Azure target from the authorized AWS installation or an inherited CLI account.
Historical Azure inspector work remains outside this milestone.

## Architecture

Use Container Apps for the Go API and packaged Angular UI, with a separate artifact service and HTTPS origin.
Use Azure Database for PostgreSQL Flexible Server for durable state and private Blob Storage for evidence.
Use Azure Container Registry, Key Vault, scoped managed identities, and Azure Monitor.
Preserve generic OIDC, internal identities, project permissions, and empty initial testing registries.
Keep PostgreSQL schedules, leases, and recovery as the durable coordination model.

Container Apps Sandboxes remain the preferred candidate for isolated tests and recording sessions.
Deploy the sandbox group with infrastructure; create individual workers through the controller at runtime.
Keep uploaded code outside trusted services and withhold installation credentials from workers.
Verify the selected sandbox API, runtime, network enforcement, deadlines, quotas, and regional availability before activation.
Ordinary Jobs remain disabled for coded execution under the current implementation.
Read the [sandbox contract](azure-sandboxes.md) and [core guide](azure.md) before implementation.

## Milestones

| Milestone | Acceptance numbers | Required evidence |
| --- | --- | --- |
| 1. Installation contract and cost | 46, 51, 53, 54, 56, 57 | Select the exact target, infrastructure scope, identity inputs, network design, budget, and cleanup ownership. Review staged packaging gaps and documentation links. |
| 2. Foundation and core | 46, 53, 54, 57 | Review the Bicep deployment changes. Deploy the selected committed release. Verify sign-in, project denial, a new native HTTPS probe, private evidence, and service restart. |
| 3. Isolated execution and preparation | 41, 51, 53, 56, 57 | Newly authored Go and Playwright suites prepare and run. Verify timing, cancellation, absolute deadlines, lost acknowledgments, recovery, evidence, and cleanup. |
| 4. Hosted recording | 12, 13, 45, 51, 56, 57 | A scoped recording session captures, saves, prepares, and executes a generic journey. Verify expiry, revocation, disconnect, capacity, and cleanup. |
| 5. Installation operations | 46, 51, 54, 57 | Verify upgrades, restore, credential rotation, two-provider authority, retention, quotas, and evidence pins. Record removal and data preservation behavior. |

Core health does not establish isolated execution or recording readiness.
Keep the seven broad criteria open until their complete cross-profile scope passes.
The ledger remains 50 verified of 57.

## Preserve future Marketplace delivery

Keep reusable Bicep modules separate from the direct-deployment entry point.
Parameterize installation identity, resource names, region, sizing, networking, and secret references.
Avoid dependencies on publisher credentials, a sibling checkout, or pre-existing application resources.
Use immutable images from committed source and record the selected deployment version.
Design upgrade, restore, export, and removal behavior before relying on a managed application wrapper.
Defer the Marketplace portal wizard and managed-application permissions until that delivery is requested.
Future Marketplace eligibility, sandbox support, image distribution, and commercial terms require separate validation.

## Cost planning

Estimate a 730-hour month in USD using public pay-as-you-go prices for a stated region.
Separate the two-service core, trusted execution services, sandbox compute, and network infrastructure.
Price Front Door, Container Apps private endpoint infrastructure, registry tiers, firewalls, and DNS separately.
Do not treat an inexpensive core as the complete isolated worker installation.
Use current regional rates when the target and topology are selected.
