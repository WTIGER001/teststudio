# Azure assembly worker

## Supported boundary

The native preparation command now supports Azure through an explicit assembly-only profile.
It consumes validated source jobs, publishes immutable registry images, and queues discovery through the existing durable handoff.
It never starts Container Apps Jobs or executes uploaded source. Discovery and source approval remain separate.
The [publisher](azure-preparation-writer.md) checks the selected registry path, writer grant, and reported host before startup and every new assembly attempt.
Azure coded execution and complete deployment acceptance remain unfinished.

## Native command

Use PostgreSQL shared with the API, the Azure execution location, and one private Azure settings document:

```sh
teststudio-prepare --assembly-only \
  --execution-location=azure-container-apps \
  --azure-assembly-env=TESTSTUDIO_AZURE_ASSEMBLY \
  --database-env=TESTSTUDIO_DATABASE \
  --require-postgres=true \
  --build-directory=/private/source-journals \
  --registry-go-runtime=REGISTRY/REPOSITORY/go@sha256:DISTRIBUTION_DIGEST
```

The registry reference above is a placeholder. Supply the actual immutable distribution digest under the configured registry and repository prefix.
Add --registry-playwright-runtime for Playwright assembly. Runtime flags must match the host settings and deployed template exactly.
Recovery-only preparation requires empty runtime flags, host runtime fields, and template runtime inputs.
Use --azure-assembly-config with a private file instead of the environment reference when suitable.
Do not supply both forms, another provider, local runtime flags, or a different execution location.
The command rejects incompatible modes and invalid settings syntax before database creation or directory creation.
Hosted operation requires PostgreSQL. It never falls back to local image preparation or another provider.

The settings document contains the existing worker Config, complete writer settings, readerClientId, and host.
The reader client identifier must be explicit, nonzero, and different from the writer client identifier.
Library callers without host observation can supply their own observation credential and omit readerClientId and host. The native command requires both.
The native reader uses ManagedIdentityCredential with that exact client identifier. It uses no default credential or login fallback.
See [writer configuration](azure-preparation-writer.md#configuration) and [private path settings](azure-preparation-registry.md#explicit-settings) for the nested fields.

Both Go and Playwright recovery builders retain their existing journal ownership checks.
Cleanup and confirmed handoff use their owned journal and selected registry client without repeating control-plane admission.
This allows cleanup when new publication is blocked and avoids adding host observation to the short handoff lease.
The registry still enforces authorization for any owned upload cancellation. Cleanup removes no published manifest or source image.

## Separate deployment template

[preparation.bicep](../deploy/azure/preparation.bicep) declares a standalone headless Container App.
It uses the existing service image's /app/teststudio-prepare binary, with one main container, no ingress, and one always-on replica.
Single active revisions and a 90-second termination grace period support orderly worker shutdown.
The process shares the installation's PostgreSQL database and uses the Azure assembly-only flags above.
It receives no job dispatch, deadline administration, Docker runtime, or Docker socket configuration.

The template requires these installation inputs:

| Input | Requirement |
| --- | --- |
| installationName and location | Installation naming and Azure region, following the core profile. |
| environmentName and environmentResourceGroup | Existing dedicated trusted preparation environment in the configured source subnet and controller VNet. |
| journalStorageName | Existing read-write Azure Files SMB storage binding in that environment, dedicated to this installation's source journals. |
| journalAccount and journalShare | Exact storage account and SMB share reported by that environment storage binding. |
| registryName and registryResourceGroup | Existing registry selected by the private path configuration. |
| imageRepository and imageDigest | Current AMD64 service image repository and immutable sha256 distribution digest. |
| keyVaultName and keyVaultResourceGroup | Existing vault with the two named secrets and scoped role-assignment support. |
| databaseSecretName | Secret containing the shared PostgreSQL connection URL. |
| assemblyConfigurationSecretName | Secret containing the complete Azure assembly settings JSON. |
| goRuntime and playwrightRuntime | Optional immutable supervised source-runtime references; an empty value enables no new builds for that runner. |
| workerEnabled | False by default. Enable after identity bindings, secrets, journal storage, and network prerequisites are configured. |

The template creates four dedicated managed identities and their separate grants:

| Identity | Grant and lifecycle |
| --- | --- |
| source-writer | Registry-scoped AcrPush; available to the main process. |
| source-reader | Fixed subscription metadata read role plus DNS policy observation action; available to the main process. |
| source-pull | Registry-scoped AcrPull; platform-only lifecycle None. |
| source-startup | Read access to the two named vault secrets; platform-only lifecycle None. |

The database and private configuration reach the process through platform Key Vault references, never literal command arguments.
The template retrieves no secret value or storage account key during evaluation.
The existing environment storage binding owns its SMB credential and network setup; this template creates neither.
Keep preparation identities separate from the controller, backend, observer, and untrusted job identities.

## Persistent journal and bootstrap

The Azure Files volume mounts /private/source-journals with directory mode 0700, file mode 0600, and uid and gid 65532.
Those identifiers match the current service image user. The mount disables client data caching.
The normal startup and registry builder reject a public or symbolic-link journal directory.
The volume survives replica replacement. Do not substitute EmptyDir or delete it during source recovery.
Microsoft documents the environment storage and volume relationship in [storage mounts](https://learn.microsoft.com/en-us/azure/container-apps/storage-mounts).
An ingress-free worker needs an explicit nonzero minimum replica count; see [scaling](https://learn.microsoft.com/en-us/azure/container-apps/scale-app).

For the separate deployment session, first provision the environment, private registry path, SMB storage binding, and current images.
Apply the reviewed template with workerEnabled=false to establish identities and grants without starting the worker.
Merge assemblyBinding.writer into the existing writer fields while retaining its registry settings.
Set readerClientId and host from the corresponding assemblyBinding fields, then save the complete settings as the named vault secret.
Keep worker.pullIdentity bound to the untrusted worker pull identity; do not replace it with the preparation platform pull identity.
Confirm the PostgreSQL secret, storage mount, configured source subnet, and private registry route before enabling the worker.
Enable workerEnabled only for the trusted assembly role. Do not advertise a complete Azure source workflow without compatible isolated discovery.

The binding output contains resource identifiers and identity metadata only. It contains no secret values or registry tokens.
The selected managed identity credential must succeed in the actual deployment; the template output does not prove effective authorization.
Build and publish the service image from this revision during the separate deployment work. Earlier preparation binaries do not recognize the Azure flags.
During upgrades, refresh the host image, runtime, identity, storage, and secret-reference settings with the matching template bindings.
Preserve the database and journal volume when replacing the worker. A stale host image or runtime setting blocks new assembly.

## Host settings and observation

The host object uses these fields from assemblyBinding.host:

| Field | Requirement |
| --- | --- |
| application and environment | Exact preparation app and dedicated environment identifiers. The environment differs from controller, worker, and exchange environments. |
| image | Current immutable service image reference in the selected registry. |
| readerIdentity and readerPrincipalId | Exact managed identity resource and principal matching readerClientId and the writer's installation tenant. |
| readerAssignment and readerRole | Selected subscription assignment and fixed custom metadata read role. |
| pullIdentity and startupIdentity | Distinct platform identities, separate from reader, writer, and untrusted worker pull identities. |
| journalStorageName, journalAccount, and journalShare | Exact selected environment storage binding and its read-write SMB account and share. |
| databaseSecretUrl and configurationSecretUrl | Distinct versionless secret URLs in the same installation vault. |
| goRuntime and playwrightRuntime | Optional immutable source runtime references matching the command and template inputs. |

The native command requires /private/source-journals. It rejects a different path or mismatched runtime flags before state creation.
The publisher adds one owned SDK reader for the host, bringing the native reader count to eleven.
After the existing writer chain succeeds, seven concurrent operations read app, environment, revisions, storage, reader identity, assignment, and role.
The selected read role uses the same fixed permission contract as the existing observer. Additional actions or data grants in that role reject.
This verifies the selected reported grant, not the absence of other effective authority.

The environment must report the configured source subnet, successful provisioning, internal networking, disabled public access, and one Consumption profile.
The selected environment storage must report the exact account and share with read-write AzureFile access. NFS and read-only bindings reject.
The SDK projection drops storage keys, key-vault credential references, and unrelated storage metadata before returning the observation.
See Microsoft's [selected storage GET contract](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/managed-environments-storages/get?view=rest-resource-manager-containerapps-2025-07-01).

The app must report the four exact identities and their configured lifecycles. Reader and writer client and principal identifiers remain pinned.
The shared app verifier compares command, arguments, image, environment, secret references, registry identity, mount, resources, and scaling against an independent definition.
Ingress, sidecars, init containers, additional flags, alternate volumes, and changes mirrored into both app and revision reject that definition.
One healthy current active revision must match the fixed template and have running replicas.
Headless revisions permit absent or zero HTTP traffic weight. HTTP services retain their existing traffic requirements.

The host check has a 55-second total bound, including the existing 45-second writer chain.
The added stage uses seven requests normally and at most nine with bounded revision paging. Each SDK operation has a five-second bound.
Only seven selected GET paths are permitted. Lists beyond bounded revisions, credential operations, mutations, redirects, and unrelated queries reject.
Native startup waits up to three minutes for unavailable platform observations, with two seconds between attempts.
Definite ownership mismatches stop immediately. Cancellation stops the wait. No job is claimed until startup succeeds.
This keeps the process alive while a new headless replica reaches platform readiness; actual activation behavior still needs deployment verification.

## Targeted local verification

Compile and inspect the standalone package without Azure CLI, credentials, registry restoration, or deployment:

```sh
python3 scripts/verify-azure-package.py --profile=preparation \
  --bicep=.data/azure-tools/bicep \
  --output=output/azure-preparation-package
```

After checker-only edits, use --compiled-template with the emitted preparation.json instead of compiling again.
Six packaging checks cover identities, grants, command, secrets, image reference, scaling, and the private persistent mount.
Targeted Go checks cover native mode rejection, runner registration, fresh admission, cleanup, and explicit reader selection.
Host fixtures cover placement, exact runtime and mount, selected read grants, synchronized revision drift, unknown metadata, sanitized storage observations, and reader closure.
The startup wait has a focused local readiness, ownership-error, and cancellation check.
The existing core package checker also passes against its retained compiled template.
These checks use local code and fake SDK transports. They make no Azure service call and rebuild no image.

## Remaining deployment evidence

Actual worker placement, complete source egress, effective read authority, and protection from identity reuse still require installation verification.
The native command now compares selected reported app, environment, storage, and reader metadata with its configuration.
Those responses do not attest the current process's origin or prove complete effective authorization and network enforcement.
SMB private access, mount permissions, atomic journal operations, directory synchronization, and restart recovery need actual deployment evidence.
The template declares the mount contract but does not prove remote filesystem behavior.
Complete Azure isolated discovery and coded execution remain unfinished. Assembly publication alone enables neither review nor test execution.
Acceptance 51 advances; the ledger remains 48 verified of 57, with nine open.
