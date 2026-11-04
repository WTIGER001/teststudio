# Azure trusted backend observation

## Purpose and boundary

BackendGuard observes the application defined by the [staged private backend package](azure-worker-backend.md).
It binds the relay's configured backend origin to a selected app, dedicated environment, worker-only configuration, and healthy active revision.
It refreshes the preceding [workload inventory stage](azure-worker-workloads.md) before backend inspection.
It reuses that call's transient placement snapshot to reject other reported backend occupants without repeating subscription scans.

This stage observes configuration and reported placement. It does not implement Network or enable coded Azure execution.
It does not establish effective read coverage, prevent later attachments, or prove complete network isolation.
The private relay route, endpoint approval, effective DNS, actual TLS trust, identity grants, and backend storage paths remain unverified.
Keep acceptance 51 and all related hosted criteria open.

## Explicit configuration

BackendSettings requires these fields:

| Field | Requirement |
| --- | --- |
| exchange | The exact ExchangeSettings bound by the preceding inventory guard. |
| application | Selected backend Container App identifier. |
| environment | Dedicated backend environment identifier, distinct from worker, controller, and exchange environments. |
| subnet | Dedicated backend subnet identifier in the configured controller VNet. It must differ from the controller infrastructure subnet. |
| infrastructureGroup | Separate backend platform infrastructure group name. |
| image | Immutable service image in the installation registry, outside the untrusted source repository prefix. |
| pullIdentity | Backend image-pull identity identifier. |
| startupIdentity | Backend database-secret resolution identity identifier. |
| runtimeIdentity | Backend artifact runtime identity identifier. |
| runtimeClientId | The selected runtime identity's client identifier. |
| databaseSecretUrl | Exact unversioned secret reference under a public Azure Key Vault secrets URL. |
| blobServiceUrl | Exact public Azure Blob service URL, including its final slash. |
| evidenceContainer | Existing installation evidence container. |
| replicas | One through ten fixed replicas. Zero selects the package default of one. |

Select the controller's existing database reference, artifact account, container, and teststudio artifact prefix.
References identify intended resources. The stage retrieves no database secret or artifact credential value.
All resource identifiers belong to the selected subscription. Separate resource groups remain supported.
The three backend identity identifiers must differ from each other and from the untrusted worker pull identity.
Changed settings cannot reuse a differently configured inventory guard.

ApplicationDefinition returns a fresh SDK definition matching the staged backend command and storage contract.
It returns no runtime credential value and issues no resource write request.
Image approval and non-root package inspection remain separate prerequisites.
The complete admission profile must bind these settings before native enablement.

## Observed environment and application

The environment must report successful provisioning, the selected infrastructure subnet and group, internal networking, and disabled public network access.
It must report one Consumption profile without environment identity or telemetry credentials.
Its default domain must use the configured region's azurecontainerapps.io namespace.
The relay backend origin must exactly equal HTTPS plus the selected application name and that observed default domain.
Custom backend domains and explicit backend ports are outside this staged platform-ingress profile.
The generic relay and direct TLS command retain their separate supported configurations.

The backend app must appear in the fresh subscription inventory with its expected environment.
Other reported apps or jobs in that backend environment reject observation, including resources in other groups.
Missing placement or incomplete subscription observations remain unavailable.
Reported absence does not establish unrestricted inventory read authority or stop a later workload join.

The app must report a ready or running state, its exact environment, and its expected ingress hostname.
The complete known writable application specification must match the fixed worker-only definition.
The comparison preserves command, arguments, replicas, containers, probes, storage references, identity lifecycles, and ingress configuration.
API mode, extra target bindings, secret values, alternate storage, startup identity exposure, extra listeners, and changed commands reject observation.
The observed runtime identity client identifier must match artifact configuration. Missing identity metadata remains unavailable.
Repeated identity client or principal identifiers reject inconsistent observations.

The environment HTTP route inventory must explicitly complete with no routes.
Platform user authentication must explicitly be disabled on the selected app's current auth configuration.
Missing or null route and revision arrays do not establish empty inventories.
The same bounded reader now applies that rule to the exchange service.

One healthy latest revision must receive all traffic and match the expected template.
Inactive revisions must explicitly report no traffic and no replicas.
At most four revision observations and three revision pages are permitted.
Unknown states, incomplete rollout metadata, or unfinished pages remain unavailable.

## Comparison and read limits

The backend and relay share the existing service comparison and SDK read implementation.
Read-only metadata, revision suffixes, and explicit inactive defaults normalize away.
Identity, registry, secret, environment variable, and probe ordering do not change their meaning.
Artifact JSON permits formatting and key-order differences. Duplicate keys, excessive structure, and unknown content do not disappear during comparison.

The SDK reader accepts an explicit credential and only five selected GET resources or collections:

- The backend environment.
- The backend application.
- Its current auth configuration.
- Its environment HTTP routes.
- Its application revisions.

It uses armappcontainers/v4 with management API 2025-07-01 and the existing public ARM transport restrictions.
Secret actions, certificate reads, other apps, resource changes, filters, extra queries, redirects, and foreign pages reject inspection.
Individual reads have five-second contexts and four-megabyte response limits. No automatic provider registration or retry runs.
The full guard has a forty-five-second context and no successful-admission cache.
A normal composed check performs fifty reads. Complete bounded paging permits at most 101 reads.
Placement snapshots contain only resource and environment identifiers. They remain local to one check and are not persisted or cached.

## Targeted local verification

Four new focused race checks cover exact resource binding, current backend specification, reported occupants, identity and storage changes, read scope, and uncertain lists.
Four selected existing exchange or inventory checks pass after shared reader and comparison extraction.
Only checks affected by transient inventory reuse and stronger occupant coverage repeat.
The fixtures use explicit fake credentials and in-memory SDK transports. They open no network connection.
No browser, UI build, image build, container, live service listener, cloud request, registry push, or deployment runs.

```sh
go test -race ./adapters/execution/azure -run '^TestAzureBackend' -count=1
go test -race ./adapters/execution/azure -run '^(TestAzureExchangeVerifiesFreshSelectedBindingAndSafeDefaults|TestAzureExchangeSDKRestrictsResourceKindsSecretsAndInventoryScope|TestAzureExchangeRejectsRuntimeAuthorityAndRouteChanges|TestAzureWorkloadInventoryChecksFreshPlacementAndDropsConfiguration)$' -count=1
```

Evidence is output/azure-worker-backend-admission/summary.json and the targeted command results.
The verified count remains 48 of 57, with nine open. Continue private backend routing and the remaining native admission controls.
