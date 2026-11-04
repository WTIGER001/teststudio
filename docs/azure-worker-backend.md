# Staged Azure trusted worker backend

## Scope and current status

The deploy/azure/worker-backend.bicep package defines the [trusted worker backend](worker-backend.md) in a dedicated internal Container Apps environment.
The environment disables public network access. Its only application runs the worker-only service role.
The package reuses the existing registry, secret, and Blob grant modules. It creates no new database, vault, registry, or artifact store.
The existing Azure core deployment remains separate and unchanged.

This is staged backend infrastructure. It is not wired into the native Azure execution provider or the core deployment.
It creates no test job, worker environment, standalone relay, private endpoint, DNS zone, firewall, or deadline workflow.
Do not enable coded Azure jobs from this package alone.
Native resource binding, effective read coverage, attachment authority, network admission, and separate deployment verification remain required.
Acceptance 51 advances. The verified total remains 48 of 57, with nine open.

## Private ingress and service identity

The environment uses a selected dedicated infrastructure subnet with internal networking and public network access disabled.
It defines one Consumption workload profile and no environment identity, Log Analytics connection, or Dapr telemetry credential.
Use a new dedicated backend subnet in the trusted network. Do not select the worker or exchange infrastructure subnet.
Do not share a Container Apps infrastructure subnet between environments. See [dedicated subnet requirements](https://learn.microsoft.com/en-us/azure/container-apps/custom-virtual-networks).

The app enables environment-level ingress on port 8042 and rejects insecure external HTTP.
Azure terminates HTTPS at its environment ingress and forwards HTTP to the private worker listener.
Azure distinguishes app-level external ingress from a public environment. App-level internal ingress permits only calls from the same environment.
See [Azure ingress visibility](https://learn.microsoft.com/en-us/azure/container-apps/ingress) and [private environment networking](https://learn.microsoft.com/en-us/azure/container-apps/networking).

This profile uses the environment's default DNS domain and platform HTTPS certificate.
It does not mount a server key or select direct backend TLS. Direct TLS remains available for other approved backend arrangements.
The generated backendOrigin uses HTTPS, the selected app name, and that environment's reported default domain.
Normal certificate verification and the actual private route still need separate proof.

The container overrides the service image entrypoint with /app/teststudio and these exact arguments:

```text
--serve-role=worker
--worker-addr=0.0.0.0:8042
--database-env=TESTSTUDIO_DATABASE
--require-postgres=true
--artifact-store-env=TESTSTUDIO_ARTIFACT_CONFIGURATION
--execution-location=azure-container-apps
```

It starts no UI, account sign-in, artifact viewer, scheduler, notification processor, source preparation, or test executor.
Platform user authentication is explicitly disabled. Attempt tickets and existing project authorization retain authority over private transport operations.
The app has one container, fixed CPU and memory, one active revision, three lifecycle probes, and a bounded shutdown grace period.
Replica counts remain between one and ten. Startup and readiness probe /healthz; liveness probes /livez.
Database readiness does not prove artifact availability, complete ingress, or provider admission.

## Configuration and scoped access

| Parameter | Required value |
| --- | --- |
| installationName | Three through twelve characters for the backend resource prefix. |
| location | Selected Azure region. |
| infrastructureSubnetId | Dedicated trusted backend infrastructure subnet. |
| infrastructureResourceGroup | Selected separate platform infrastructure resource group name. |
| registryName, registryResourceGroup | Existing installation registry and its resource group. |
| imageRepository, imageDigest | Approved service repository and full digest, including sha256:. |
| keyVaultName, keyVaultResourceGroup | Existing installation vault and its resource group. |
| databaseSecretName | Existing named secret containing the shared PostgreSQL connection. |
| storageAccountName, storageResourceGroup | Existing artifact account and its resource group. |
| evidenceContainer | Existing installation artifact container. |
| replicas | Fixed minimum and maximum backend replicas. The default is one. |

Use the same database, account, container, and teststudio artifact prefix as the controller.
Select an approved immutable service image containing the worker role. Current cached service images can predate that code.
Image publication, approval, complete digest validation, and protected repository checks remain prerequisites for native wiring.
Template length constraints do not inspect an image or establish that it is trusted.

Three new user-assigned identities have separate purposes:

| Identity | Granted scope | Runtime availability |
| --- | --- | --- |
| backend-pull | AcrPull on the selected registry. | None. Platform image pull only. |
| backend-startup | Key Vault secret read on the one database secret. | None. Platform secret resolution only. |
| backend-runtime | Blob contributor on the selected evidence container. | Main container only. |

The application receives one database secret reference and non-secret Azure artifact settings with the exact runtime client identifier.
The package retrieves no secret value or storage key during compilation or through ARM template functions.
It adds no OIDC credential, target credential, source builder, registry writer, Docker socket, or AWS configuration.
These are intended role assignments. Actual permissions, unexpected inherited grants, identity placement, and effective network access still require admission.

The package outputs only backendOrigin, backendApplicationId, and backendEnvironmentId.
Native setup must bind all three to observed state. A configured URL alone does not establish the selected application or environment.
Do not change the controller environment reference merely because a separate backend environment now exists.
Its relationship to the trusted network must remain explicit in later admission settings.

## Remaining installation work

Before native enablement, bind and inspect the actual backend app, environment, revision, runtime configuration, and effective identities.
Verify exclusive backend occupancy and restrictions on creating or joining other workloads in its environment.
The intended resource definition does not prove future attachment authority.

Provide the relay network with a dedicated private backend path and correctly scoped DNS.
Verify endpoint approval, interface ownership, selected destination, normally trusted TLS, and all effective ingress and egress restrictions.
Do not expose the API, reports, database, or storage through that path.
Keep worker, relay, backend, and preparation network relationships explicit. Do not introduce worker-to-controller peering.
Provide backend access to PostgreSQL, private artifacts, image pull, and platform secret resolution through their approved trusted paths.
The current package provisions none of those endpoint, routing, or DNS controls.

Complete the remaining Azure admission stages and bind the complete policy into the execution profile before dispatch wiring.
Run actual Azure installation and execution verification in the separate hosted phase.

## Targeted local verification

The checker compiles only this backend package and its existing scoped grant modules with a local Bicep binary.
It uses no Azure CLI, account, live endpoint, registry restoration, image build, UI build, or browser.
Six checks cover private environment intent, HTTPS listener configuration, exact worker command, identity lifecycles, scoped grants, and probes or outputs.
The backend process and its HTTPS protocol retain the focused local evidence from the preceding backend deliveries.
This packaging batch does not repeat those unchanged Go checks or the existing core package checks.

```sh
python3 scripts/verify-azure-worker-backend.py \
  --bicep .data/azure-tools/bicep \
  --output output/azure-worker-backend
```

The generated template, compiler log, and summary remain under output/azure-worker-backend.
Compilation and structural checks establish package intent only. They do not establish deployed isolation or complete acceptance.
