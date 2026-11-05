# Azure assembly worker

## Supported boundary

The native preparation command now supports Azure through an explicit assembly-only profile.
It consumes validated source jobs, publishes immutable registry images, and queues discovery through the existing durable handoff.
It never starts Container Apps Jobs or executes uploaded source. Discovery and source approval remain separate.
The [publisher](azure-preparation-writer.md) checks the selected registry path and writer grant before startup and every new assembly attempt.
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
Add --registry-playwright-runtime for Playwright assembly. Omitting both runtime flags starts recovery-only preparation.
Use --azure-assembly-config with a private file instead of the environment reference when suitable.
Do not supply both forms, another provider, local runtime flags, or a different execution location.
The command rejects incompatible modes and invalid settings syntax before database creation or directory creation.
Hosted operation requires PostgreSQL. It never falls back to local image preparation or another provider.

The settings document contains the existing worker Config, complete writer settings, and readerClientId.
The reader client identifier must be explicit, nonzero, and different from the writer client identifier.
Library callers supplying their own observation credential can omit it; the native command cannot.
The native reader uses ManagedIdentityCredential with that exact client identifier. It uses no default credential or login fallback.
See [writer configuration](azure-preparation-writer.md#configuration) and [private path settings](azure-preparation-registry.md#explicit-settings) for the nested fields.

Both Go and Playwright recovery builders retain their existing journal ownership checks.
Cleanup and confirmed handoff use their owned journal and selected registry client without repeating control-plane admission.
This allows cleanup when new publication is blocked and avoids adding a 45-second observation to the short handoff lease.
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
Set readerClientId from assemblyBinding.readerClientId, then save the complete settings as the named vault secret.
Keep worker.pullIdentity bound to the untrusted worker pull identity; do not replace it with the preparation platform pull identity.
Confirm the PostgreSQL secret, storage mount, configured source subnet, and private registry route before enabling the worker.
Enable workerEnabled only for the trusted assembly role. Do not advertise a complete Azure source workflow without compatible isolated discovery.

The binding output contains resource identifiers and identity metadata only. It contains no secret values or registry tokens.
The selected managed identity credential must succeed in the actual deployment; the template output does not prove effective authorization.
Build and publish the service image from this revision during the separate deployment work. Earlier preparation binaries do not recognize the Azure flags.

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
The existing core package checker also passes against its retained compiled template.
These checks use local code and fake SDK transports. They make no Azure service call and rebuild no image.

## Remaining deployment evidence

Actual worker placement, complete source egress, effective read authority, and protection from identity reuse still require installation verification.
The native command observes configured private resources; it does not yet compare its selected app and environment with deployed workload metadata.
SMB private access, mount permissions, atomic journal operations, directory synchronization, and restart recovery need actual deployment evidence.
The template declares the mount contract but does not prove remote filesystem behavior.
Complete Azure isolated discovery and coded execution remain unfinished. Assembly publication alone enables neither review nor test execution.
Acceptance 51 advances; the ledger remains 48 verified of 57, with nine open.
