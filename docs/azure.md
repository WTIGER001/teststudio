# Azure core service profile

The user selects a direct installation into their own Azure subscription as the first delivery on 2026-11-08.
Follow [the deployment plan](azure-deployment-plan.md). The target selection, implementation, and live verification remain pending.
Keep a future Managed Application viable. Marketplace publication and billing remain deferred.
This guide describes staged implementation. It does not establish deployed acceptance.

The Azure package runs TestStudio in two Container Apps.
One app serves the UI and API. The second app serves private evidence and the trace viewer on a separate HTTPS origin.
The API app owns identity initialization, schedules, execution coordination, and maintenance.
The artifact app validates existing grants against current stored sessions, credentials, permissions, and evidence manifests.

The current package supports the catalog, identity, native HTTP execution, schedules, and private evidence service.
The separate [assembly profile](azure-native-assembly.md) now supplies a trusted source preparation command and headless template.
Complete source discovery, Go execution, Playwright execution, and recording remain unfinished.
The active coded execution path uses the [sandbox client and launch contract](azure-sandboxes.md) and [durable dispatch controller](azure-sandbox-controller.md).
The [sandbox deadline SDK](azure-sandbox-deadlines.md) now supplies owned independent timers and checkpointed cleanup.
The [command bridge](azure-sandbox-commands.md) now supplies independent exit receipts, guarded Linux supervision, and hosted status mapping.
The [image manager](azure-sandbox-images.md) now supplies durable source conversion and verified image lookup.
The [provider bridge](azure-sandbox-provider.md) now connects the shared execution and recovery lifecycle, including pending source conversion.
The [runtime factory](azure-sandbox-runtime.md) now supplies independent settings and concrete SDK assembly.
The [placement stage](azure-sandbox-placement.md) now checks selected reported group, network, and source-pull metadata.
The [controller dependency binding](azure-sandbox-host-access.md) now checks selected secret and evidence grants, vault RBAC metadata, and private container access.
The [authority stage](azure-sandbox-authority.md) now checks configured identity resources, custom control and timer permissions, and selected grants.
The [network stage](azure-sandbox-network.md) now checks complete selected security rules and route contents.
The [private stage](azure-sandbox-private.md) now checks selected endpoints, NIC addresses, endpoint subnet rules, and complete private DNS inventories.
The [topology stage](azure-sandbox-topology.md) now checks selected VNet contents, firewall and public IP attachments, and complete firewall policy groups.
The [resolver stage](azure-sandbox-resolver.md) now checks selected policy rules, actual association, and reported subscription private-zone links.
The [service stage](azure-sandbox-services.md) now checks fixed relay and backend definitions, private environments, identity exposure, certificate metadata, and complete revisions.
The [backend endpoint stage](azure-sandbox-backend-endpoint.md) now joins approval, NIC address, network policy, and the existing relay readiness address.
The [relay network stage](azure-sandbox-relay-network.md) now checks the trusted VNet, egress policy, DNS, private registry endpoint, and selected pull grant.
The [registry service stage](azure-sandbox-registry-service.md) now checks access policies and complete reported consumer, member, and replication inventories.
The [controller source stage](azure-sandbox-host-network.md) now checks VNet contents, source rules, firewall routing, derived HTTPS dependencies, and resolver association.
The [controller database stage](azure-sandbox-host-database.md) now checks private PostgreSQL endpoint ownership, DNS, source rules, and reported approvals.
The [controller registry stage](azure-sandbox-host-registry.md) now checks its private endpoint, DNS, selected pull grant, and reported approval.
The [controller host stage](azure-sandbox-host.md) now checks fixed execution or discovery definitions, private environments, identity lifecycles, and complete revisions.
The [service workload stage](azure-sandbox-service-workloads.md) now checks reported relay, backend, and controller occupancy, identity placement, identity resources, and federation absence.
Effective routing, resolution, deployed readiness, and effective service authority remain unverified.
Sandboxes and the selected API remain preview. Complete authority and network admission, timer authority, command wiring, and packaging remain unfinished.
Ordinary Jobs stay disabled. Their contracts below remain reference material and do not establish sandbox isolation.
The [Azure job contract](azure-worker-contract.md) now builds and verifies SDK requests for execution and discovery.
The [durable provider](azure-worker-lifecycle.md) now passes local dispatch, status, cancellation, and checkpointed recovery checks.
The [trusted ACR adapter](azure-registry-authentication.md) now supports separate inspection and preparation credentials.
The [independent deadline adapter](azure-worker-deadlines.md) now supplies durable timer resources.
The [topology inspector](azure-worker-network.md) now verifies environment and subnet bindings through scoped reads.
The [perimeter inspector](azure-worker-perimeter.md) now verifies the fixed NSG rules and default route.
The [firewall inspector](azure-worker-firewall.md) now checks actual attachment, dedicated policy, and complete group inventory.
The [DNS inspector](azure-worker-dns.md) now checks actual resolver policy, VNet association, complete rules, and default block.
The [selected private-zone inspector](azure-worker-private-dns.md) now verifies exact records, completed links, and absence of public fallback.
Complete network admission, identity role binding, and command wiring remain required before enabling coded jobs.
Do not enable local Docker runner flags or mount a Docker socket inside these apps.
Local packaging checks do not close Azure deployment acceptance.

## Staged component references

These references preserve staged implementation details. Use the current deployment plan to select work; do not restart historical inspector work.

The [Azure worker admission guide](azure-worker-network.md) links the network, endpoint, and registry service checks under development.
The [standalone worker relay](worker-relay.md) supplies a bounded frontend without installation credentials, [pinned backend readiness](worker-relay-readiness.md), and a UI-independent image builder. Provider service wiring remains unfinished.
The [staged Azure worker backend](azure-worker-backend.md) defines a dedicated private environment, worker-only app, and scoped installation identities. Admission and deployment remain separate.
The [backend observation stage](azure-worker-backend-admission.md) checks actual app configuration, reported placement, identities, and revisions through scoped reads. Complete admission remains unfinished.
The [backend private endpoint observer](azure-worker-backend-endpoint.md) binds service approval, NIC and address ownership, and the dedicated endpoint subnet policy. Effective routing and TLS remain unfinished.
The [relay VNet observer](azure-relay-topology.md) checks address space, peering, DNS settings, and subnet consistency with one additional read. Complete native admission remains required.
The [relay perimeter observer](azure-relay-perimeter.md) binds the source NSG and default route through shared policy checks. Firewall and effective DNS controls remain required.
The [relay firewall observer](azure-relay-firewall.md) binds the routed appliance, policy, subnet, and public IP. Effective DNS and registry endpoint ownership remain required.
The [relay DNS policy observer](azure-relay-dns.md) checks exact lists, enabled rules, exclusive links, and actual VNet association. Effective private-zone visibility remains required.
The [relay registry endpoint observer](azure-relay-registry-endpoint.md) binds an explicit approved consumer to its NIC, member addresses, subnet, and security rules. Private records remain required.
The [relay private DNS observer](azure-relay-private-dns.md) binds selected registry records to those member addresses and one relay-only link. Effective VNet-wide visibility remains required.
The [reported private DNS inventory](azure-private-dns-inventory.md) detects additional subscription-visible links into worker and relay VNets. Cross-scope coverage and attachment authority remain required.
The [network and DNS deny observer](azure-network-deny.md) checks enforced, unconditional protection metadata on both VNets and three selected zones. Deployed enforcement and lifecycle authority remain required.
The [observer identity and read grant](azure-read-identity.md) bind a separate managed identity to one selected subscription read role. Bound sessions use that credential throughout inspection; effective access and native installation remain required.
The [reported read-deny scan](azure-read-deny.md) gates bound inspection on reported subscription conflicts and required protection visibility. Hosted coverage and enforcement remain unverified.
The [reported protection stacks](azure-protection-stacks.md) must match fixed policy and protected roots; direct assignment ownership and management authority remain unproven.
The [assembly-only source mode](source-assembly-handoff.md) separates registry publication from mandatory discovery through a durable handoff. The [Azure assembly profile](azure-native-assembly.md) supplies native configuration, a persistent-journal template, and startup checks for selected host, storage, and reader metadata. Actual deployment and isolated discovery remain unfinished.
The [Azure preparation network](azure-preparation-network.md) adds source egress, a private journal account and endpoint, native policy checks, and an actual writable SMB mount probe. Its separate template retrieves no storage keys.
The [preparation registry observer](azure-preparation-registry.md) checks the selected controller-side endpoint, NIC, subnets, security group, and private DNS. The [writer publisher](azure-preparation-writer.md) composes those checks with an explicit identity and selected push grant. Actual placement, effective grants, and admitted discovery remain unfinished.
The [observer host stage](azure-observer-host.md) checks reported controller attachment and revisions. Optional core packaging declares its separate API-only identity; native startup remains required. Its [fixed runtime contract](azure-observer-host-runtime.md) additionally checks API arguments, secret references, storage, ingress, probes, and scaling. The [platform authentication check](azure-observer-host-auth.md) requires an explicit disabled current configuration. The [controller route check](azure-observer-host-routes.md) requires reported absence of environment routing overrides.
The [trusted worker backend role](worker-backend.md) serves private transport without UI, sign-in, or background workers. It supports optional direct TLS; private routing, certificate trust, and provider wiring remain required.
The [private worker exchange](worker-exchange.md) and [hosted supervisor](job-supervisor.md) supply scoped transport and runner supervision for the provider adapters under development.

## Required installation resources

Provision these dedicated resources before a later deployment:

| Input | Requirement |
| --- | --- |
| Target resource group | An installation resource group in the selected subscription. |
| location | The same Azure region as the selected Container Apps environment. |
| environmentName and environmentResourceGroup | An existing Container Apps environment with suitable networking, logs, and outbound connectivity. |
| registryName and registryResourceGroup | An existing Azure Container Registry in the subscription. Use registry RBAC permissions compatible with AcrPull. |
| imageRepository | The registry repository containing the service image. |
| imageDigest | The pushed distribution digest, including sha256: and its 64 hexadecimal characters. |
| keyVaultName and keyVaultResourceGroup | An existing Key Vault with Azure RBAC authorization enabled. |
| databaseSecretName | A Key Vault secret containing the complete PostgreSQL connection URL as its plain string value. |
| oidcConfigurationSecretName | A Key Vault secret containing the initial provider JSON array as its plain string value. |
| bootstrapIssuer and bootstrapSubject | The exact configured issuer and external subject for the initial administrator. |

Use an installationName containing lowercase letters, numbers, and hyphens. Start with a letter and end with a letter or number.
The name has three to 20 characters. It determines the two app names and independent managed identity names.
Existing environment, registry, and vault resources can reside in different resource groups in the same subscription.
The deployment identity needs resource creation and scoped role-assignment authority in the relevant resource groups.
The package retrieves no vault secret values or storage keys during template evaluation.

The environment must reach PostgreSQL, Key Vault, the registry, Blob storage, identity providers, and registered HTTP targets.
Use verified PostgreSQL TLS and installation-specific routing and firewall rules.
Configure the database's trusted root certificates through the supported PostgreSQL connection settings.
The package creates no database, environment, virtual network, registry, vault, or external identity provider.
Environment networking and logs remain operator responsibilities. No AWS service or credential is an Azure installation input.

The initial profile uses the environment's managed HTTPS domains.
The UI host is INSTALLATION-api.ENVIRONMENT_DEFAULT_DOMAIN.
The artifact host is INSTALLATION-artifacts.ENVIRONMENT_DEFAULT_DOMAIN.
The template derives these origins before either app starts. Both apps receive the same origin values.
Custom DNS and certificate bindings are not part of this initial package.

## Image architecture and local build

Container Apps requires a Linux AMD64 image for this profile.
Build that architecture explicitly:

```sh
node scripts/build-service-image.mjs --arch=amd64
```

Cross-architecture packaging requires Docker buildx.
If the plugin is not installed, select a trusted standalone buildx executable:

```sh
node scripts/build-service-image.mjs --arch=amd64 --buildx=.data/docker-tools/docker-buildx
```

The producer uses the current Docker context's built-in builder and loads the image locally.
It does not push a registry image or create a builder daemon.
The certificate bundle comes from a pinned image through a stopped extraction container. No foreign-architecture stage executes.
The producer verifies removal of that container and removes its temporary build context.
UI, private viewer, and unchanged Go binaries reuse their cached outputs.

Push the produced image to the dedicated registry only when that action is authorized.
Use the resulting registry digest in the deployment parameters. A local Docker image identifier is not a registry distribution digest.
Both apps use the same image version and immutable reference.
The image runs as 65532:65532 and includes no shell, Docker CLI, test suite, or application database.

This Container Apps API exposes no container read-only root or Linux capability-drop setting.
The template does not invent those properties or claim the Kubernetes and ECS filesystem controls apply unchanged.
The core service uses its non-root image identity and container-scoped temporary storage.
PostgreSQL and Blob storage hold persistent state. No task-local SQLite fallback is allowed.
Untrusted execution requires its own isolation design and adapter verification.

## Identity configuration

Use the [initial provider configuration](providers.md#initial-operator-configuration) format in the selected Key Vault secret.
Provider entries contain credential references, not resolved client secrets.
OidcClientOneSecretName maps to env:TESTSTUDIO_OIDC_CLIENT_ONE.
OidcClientTwoSecretName maps to env:TESTSTUDIO_OIDC_CLIENT_TWO.
Each optional secret contains one plain client credential string. Public clients can omit both mappings and their secretRef fields.
The configuration can contain multiple providers. Additional independent client credentials require extending the explicit mappings.

Register https://UI_HOST/auth/callback/PROVIDER_ID with each external provider.
Use TestStudio's displayed callback to confirm the registration.
The initial administrator requires one exact issuer and subject. An email address is not an inferred subject.
LocalAccounts optionally enables managed password sign-in alongside OIDC on the API app.
Project owners add authorized users directly. Initial user provisioning alone grants no project authority.

The API validates the entire initial provider list before database initialization.
Existing provider records remain unchanged on restart. Use provider administration for later record changes.
The initial administrator's bootstrap runs once and does not restore revoked access on later starts.
The artifact app receives neither provider configuration nor client credentials.
It follows the database's current password-authentication state without enabling or disabling that state itself.
Disabling password sign-in invalidates an existing password session's artifact grant on its next use.

Key Vault references select versionless secret URIs. Use the platform's documented refresh behavior and controlled revision changes for rotation.
Verify the replacement revision's database and identity behavior before treating a rotation as complete.
Do not put connection values or client credentials in Bicep parameter files or command arguments.
Authorized infrastructure operators can inspect resolved app secrets and environments. Restrict those administrative permissions.

## Managed identities and storage

The package creates five user-assigned managed identities:

| Identity | Authority and lifecycle |
| --- | --- |
| API runtime | Blob Data Contributor on the dedicated evidence container. Available to the main API container. |
| Artifact runtime | Blob Data Reader on the same container. Available to the main artifact container. |
| API startup | Key Vault Secrets User on the database, provider settings, and mapped client secrets. Unavailable through the runtime identity endpoint. |
| Artifact startup | Key Vault Secrets User on the database secret only. Unavailable through the runtime identity endpoint. |
| Image pull | AcrPull on the selected registry. Unavailable through the runtime identity endpoint. |

ObserverIdentityEnabled optionally adds a sixth identity and a fixed subscription read role. It defaults to false.
Only the API main container receives that observer identity. The artifact role cannot expose it through the shared module parameter.
The observerBinding output supplies its identity, grant, and host references without secrets or tokens.
The optional subscription module requires role-definition and role-assignment authority at that scope.
This option prepares identity resources. It enables neither native observer startup nor coded execution.
The observer inspector also requires a resource-scoped Azure-managed protection assignment. The current core package does not create that protection arrangement.
See [observer host attachment](azure-observer-host.md) for settings, reported lifecycle checks, request bounds, and remaining admission requirements.

Blob configuration selects each app's exact runtime client identifier.
The platform resolves Key Vault and registry references with their separate identities.
Role assignments use explicit container, named-secret, or registry scopes.
Role propagation and real managed identity behavior still require deployment verification.
An existing registry that uses ABAC repository permissions needs a corresponding repository-reader role configuration instead of this AcrPull profile.

The template creates one dedicated StorageV2 account, Blob service, and private evidence container.
It disables shared-key authorization and public Blob access. HTTPS and TLS 1.2 are required.
The service reads and writes through the Azure Blob adapter. It does not mount Blob storage as a filesystem.
No SAS credential or storage account key enters the app settings.
The initial storage endpoint is publicly routable with identity authorization. Private endpoint integration remains installation networking work.

A CanNotDelete lock protects the storage account's control-plane deletion.
The lock does not prevent application-controlled Blob retention and deletion.
The package installs no lifecycle expiration rule or soft-delete retention that bypasses product retention decisions.
Keep project retention, pins, manifests, quotas, and interrupted-upload recovery within the common evidence service.
Review PostgreSQL and evidence backups before removing the storage deletion lock or changing the installation's ownership.

## Service lifecycle

Both apps reject insecure ingress. Each app exposes one target port.
The API app uses port 8040 and --serve-role=api.
The artifact app uses port 8041 and --serve-role=artifacts.
No additional public TCP port exposes the other listener.
The artifact role starts no scheduler, run worker, preparation worker, notification worker, or maintenance loop.

Startup and readiness probes call /healthz on the selected listener.
Liveness calls /livez on that listener. A database outage fails readiness without changing process liveness.
The artifact origin serves no UI, authentication route, or catalog API.
Evidence and trace responses keep their existing private grant and content policies.
The profile adds no CDN response caching layer.

Single revision mode sends traffic to the latest healthy revision.
The termination grace period is 90 seconds.
ApiReplicas and ArtifactReplicas independently set fixed counts from one to ten.
Both default to one and remain above zero. Schedules and maintenance do not depend on incoming HTTP traffic.
Shared PostgreSQL and Blob storage support multiple instances, but hosted replica recovery acceptance remains unverified.

## Compile and check locally

Use a trusted Bicep 0.48.1 executable. The current local copy is .data/azure-tools/bicep.
Its release URL and verified download checksum are recorded in .data/azure-tools/download.json.
Compile and check the package without Azure CLI or account credentials:

```sh
python3 scripts/verify-azure-package.py --bicep .data/azure-tools/bicep --output output/azure-package/check
```

The checker compiles local modules with provider type validation and no registry restore.
It checks role separation, origins, ingress, probes, identities, permissions, private storage, and immutable image references.
It makes no Azure account or service call. The compiled template and summary stay in the output directory.
Use --compiled-template PATH instead of --bicep PATH to check an already-current compiled template without compiling it again.
Recompile when the Bicep source changes. Contract checks alone do not establish that a saved template matches edited source.

The current AMD64 image passes separate local role and configuration checks:

```sh
python3 scripts/verify-service-roles.py --image IMAGE_ID --output output/azure-package/roles
python3 scripts/verify-service-image.py --configuration-only --image IMAGE_ID --output output/azure-package/configuration
```

The role check uses one fresh SQLite installation sequentially, with one process at a time.
It verifies the two listeners, unused address handling, artifact route boundaries, persisted sign-in mode, and invalid role rejection.
This local check does not substitute SQLite for the profile's required shared PostgreSQL database.
All owned containers and volumes are removed. No browser or hosted deployment runs.
A focused Go check also validates a password session's retained artifact access and later sign-in revocation through the artifact role.

During the separate deployment session, review the compiled template and an Azure what-if result before applying changes.
Then verify the real identity providers, network reachability, private evidence, replica recovery, and common acceptance scenarios.
Complete the isolated preparation and execution adapters before claiming hosted Go or browser support.

## Provider references

[Container support](https://learn.microsoft.com/en-us/azure/container-apps/containers) specifies the supported image architecture.
[Container Apps schema](https://learn.microsoft.com/en-us/azure/templates/microsoft.app/2025-07-01/containerapps) defines the selected resource properties.
[Managed identities](https://learn.microsoft.com/en-us/azure/container-apps/managed-identity) describes runtime identity lifecycle controls.
[Key Vault references](https://learn.microsoft.com/en-us/azure/container-apps/manage-secrets) describes protected secret resolution and refresh.
[Storage behavior](https://learn.microsoft.com/en-us/azure/container-apps/storage-mounts) describes temporary storage and Blob mount limitations.

## Staged private worker backend

The separate [worker backend package](azure-worker-backend.md) defines a dedicated internal environment and the worker-only service role.
It reuses existing PostgreSQL, artifact storage, vault, and registry references through three separate managed identities.
It overrides the service image entrypoint and adds only database and artifact configuration.
It exposes no UI, sign-in, source preparation, scheduler, notification worker, or test executor.
The core deployment does not include or invoke this package.

Local compilation and six packaging checks pass without Azure account access or deployment.
Private relay access, effective identity authority, DNS, routing, complete admission, and separate hosted verification remain unfinished.
Keep coded Azure execution disabled. Backend infrastructure intent does not complete acceptance 51 or 57.

The [reported protection-stack checks](azure-protection-stacks.md) require two explicit stack references and fixed managed-resource sets.
The core package creates no such arrangement. Direct assignment ownership and protection-management authority remain unproven.

The observerBinding output includes non-secret [API runtime settings](azure-observer-host-runtime.md) derived from the same installation parameters as the API app.
It supplies references rather than secret values. It enables no coded execution or automatic admission.

Both core apps explicitly disable [Container Apps platform authentication](azure-observer-host-auth.md) so TestStudio handles sessions and private report authorization.
The observer requires fresh reported disabled state on the selected API app. This enables no native coded execution.

The [host observer profile](azure-observer-host-routes.md) requires an explicit empty controller-environment HTTP route collection.
The core package creates no route overrides and removes none from the supplied existing environment. Select a suitable environment before admission.

The [preparation registry path](azure-preparation-registry.md) now has selected resource and DNS observation through the existing SDKs.
The core package creates none of those preparation resources. The separate [assembly template](azure-native-assembly.md) declares a trusted worker and persistent journal mount.

The [assembly-only source mode](source-assembly-handoff.md) now provides a durable queue boundary for a future separate Azure registry writer.
It supports existing ECS and Kubernetes native registry settings and the separate Azure assembly settings.
The [Azure preparation publisher](azure-preparation-writer.md) now loads explicit library settings and composes selected network, identity, and push-grant checks.
It opens a separate managed-identity ACR client without executing source.
Native startup now checks selected app, environment, storage, and reader metadata against the fixed assembly profile before claiming jobs.
The [preparation network package](azure-preparation-network.md) adds source egress, private journal resources, and required native network and mount checks.
Complete effective grants, actual placement enforcement, and isolated discovery remain unfinished.
