# Azure sandbox controller host binding

## Introduction

OpenRuntime now verifies the selected controller application before sandbox creation, source import, and live admission.
Execution and discovery use separate host definitions and identity sets.
The definitions are staged packaging inputs. Commands do not accept the native --azure-sandbox-env option yet.
This change does not activate Azure execution or establish deployment acceptance.

## Concepts

### Host settings and profiles

Settings.host contains these required bindings:

| Field | Contract |
| --- | --- |
| application, environment | Selected controller application and private environment, separate from relay and backend services. |
| subnetPrefix, registry | Controller source prefix and private registry path described in the [registry guide](azure-sandbox-host-registry.md). |
| subnet, infrastructureGroup | Selected environment subnet and infrastructure group, separate from sandbox, relay, and backend networks. |
| image | Immutable trusted service image outside the source-image namespace. |
| pull, startup | Distinct identity resource, client, and principal pins. |
| databaseSecretUrl, configurationSecretUrl | Distinct Key Vault secret references for PostgreSQL and native runtime configuration. |
| replicas | Fixed minimum and maximum replica count between one and ten. |
| profile | Settings.HostProfile(), calculated after all other settings are assigned. |
| runtime | Required Blob runtime identity for execution. Discovery omits it. |
| blobServiceUrl, evidenceContainer, publicOrigin | Required execution evidence settings and UI origin. Discovery omits them. |

The sandbox-controller-host/v1 hash binds the complete role-specific settings, with host.profile cleared during calculation.
The application exposes that nonsecret hash through TESTSTUDIO_HOST_PROFILE.
Host configuration stays outside the sandbox-placement/v9 worker network profile.
Execution and discovery retain the same converted-image pool when their worker configuration and import identity match.

### Fixed process and identity boundaries

| Host | Process | Main identities | Platform-only identities |
| --- | --- | --- | --- |
| Execution | /app/teststudio --serve-role=controller | Control, registry inspection, metadata, evidence runtime | Image pull, secret startup |
| Discovery | /app/teststudio-prepare --discovery-only | Control, registry inspection, metadata | Image pull, secret startup |

Platform-only identities use lifecycle None. Main identities use lifecycle Main.
The fixed execution command selects PostgreSQL, shared Blob evidence, an explicit health address, and the UI origin for completion links.
The fixed discovery command selects PostgreSQL and native configuration without an artifact adapter or assembly options.
Neither host has ingress, sidecars, init containers, volumes, or arbitrary environment variables.
The execution host has startup and readiness probes on /healthz and a liveness probe on /livez.
Discovery has no HTTP listener or probes in its expected definition.

Settings.HostDefinitions returns the application and disabled platform-authentication bodies.
It does not create Azure resources or generate a complete deployment.

### Reported admission

SDKHost reads six selected paths with API version 2025-07-01:

- Controller application and private environment.
- Application authentication configuration and complete revisions.
- Environment HTTP routes and private endpoint connections, both explicitly empty.

The environment must match its subnet, infrastructure group, region, and private configuration.
The application must retain its exact identities, lifecycles, secret references, command, resources, scaling, environment variables, and probes.
One active healthy revision must match the fixed template. Inactive revisions have zero traffic and replicas.
The headless revision contract reuses the existing service verifier.

The expanded [workload stage](azure-sandbox-service-workloads.md) also checks controller occupancy and reported identity sharing.
It verifies nine identities for discovery and ten for execution, including identity resources and empty federation inventories.
Controller identities cannot appear on another reported app or job in the selected subscription.
Import and timer host attachment remain separate work.

Each host observation has a five-second joined deadline and existing response bounds.
Revision inventory permits four items and four pages. Required empty collections permit one page.
Creation and source import use fresh reads. Successful live evidence has a 15-second bound from observation start.
A current failure invalidates prior success. Remaining protection and authorization stay uncached.
Runtime.Close closes the host reader after owned work joins.

## Tutorials

### Local verification

Four new checks and eleven affected checks pass locally with race detection in 4.223 seconds.
They cover both host modes, shared image-pool identity, configuration rejection, template and identity drift, runtime admission, and shutdown.
The affected selection retains central composition, scoped modes, settings, credentials, selected authority, workload census, and trusted service checks.
Evidence is output/azure-sandbox-host/summary.json and local.log.
Checks use fake credentials, in-memory metadata, and local SQLite.
No cloud service, browser, container, UI build, demo restart, or remote push runs.

### Continue native activation

The [registry stage](azure-sandbox-host-registry.md) now binds the selected private registry path and reported pull authority.
Complete controller VNet contents, source network policy, dependency access, effective grants, deny effects, and protected writes.
Bind remaining import and timer identities and declared registry consumers to their actual hosts and networks.
Then add native command selection and deployment packaging.
Actual token issuance, cross-subscription visibility, deny-hidden resources, routing, preview compatibility, and independent timer execution require separate deployment verification.

Related guides: [runtime](azure-sandbox-runtime.md), [headless controller](execution-controller.md), and [selected authority](azure-sandbox-authority.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
