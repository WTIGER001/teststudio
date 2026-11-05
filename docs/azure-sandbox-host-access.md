# Azure sandbox controller dependency authority

## Introduction

The existing SDKAuthority reader now checks the controller's selected startup-secret and execution-evidence grants.
It also checks the named vaults and the execution Blob account and container metadata.
This extends the existing authority stage. It adds no separate cache, credential chain, secret retrieval, or resource mutation.
Native sandbox command selection and complete effective authorization remain unfinished.

## Concepts

### Required settings

host.access contains these bindings:

| Field | Contract |
| --- | --- |
| database.vault | Vault resource whose name matches databaseSecretUrl. |
| database.grant | Custom role and assignment scoped to that database secret. |
| configuration.vault | Vault resource whose name matches configurationSecretUrl. |
| configuration.grant | Separate custom role and assignment scoped to that configuration secret. |
| blob.account | Execution-only account whose Blob endpoint matches blobServiceUrl. |
| blob.grant | Execution-only custom role and assignment scoped to evidenceContainer in that account. |

Each grant contains role and assignment resource identifiers.
Roles belong to the installation subscription. Assignments use the exact selected secret or container scope.
Roles and assignments cannot reuse the existing native control, timer, metadata, registry, or other dependency grants.
Discovery omits blob entirely. Execution requires it and the existing separate runtime identity.
The host profile binds these settings without changing the shared worker or converted-image profile.

### Fixed dependency roles

| Principal | Scope | Fixed permissions |
| --- | --- | --- |
| Host startup | Database secret | Get secret contents and read secret metadata. No management actions. |
| Host startup | Runtime-configuration secret | Get secret contents and read secret metadata. No management actions. |
| Execution runtime | Evidence container | Read container metadata; read, write, and delete its blobs. |

Startup and runtime principals remain separate. Discovery receives no evidence grant.
The evidence role grants no storage-key access, container creation or deletion, delegation-key generation, or role administration.
The startup roles grant no secret writes or vault administration.
The existing artifact adapter uses ordinary block-blob upload, download, and delete operations.

Each custom role retains an exact name, one resource-group assignable scope, fixed actions, fixed data actions, and no exclusions or conditions.
Each assignment retains its exact principal, ServicePrincipal type, scope, and role, without conditions or delegated identities.
This uses Key Vault's supported [individual-secret assignment scope](https://learn.microsoft.com/en-us/azure/key-vault/general/rbac-guide).
The secret data-action names follow the [Key Vault role contract](https://learn.microsoft.com/en-us/azure/role-based-access-control/built-in-roles/security#key-vault-secrets-user).

### Reported dependency resources

Each vault must report the installation tenant, selected region, exact vault URI, enabled RBAC authorization, and enabled public network access.
This controller profile uses the existing selected HTTPS dependency path. It does not configure private vault endpoints.
The reader retrieves vault management metadata only. It does not GET either secret resource or its value.

The execution account must report successful provisioning, the selected region and Blob endpoint, HTTPS-only transport, and TLS1_2.
Anonymous Blob access and shared-key access must be explicitly disabled. Public network access must be enabled for this profile's selected HTTPS path.
The selected evidence container must exist, report publicAccess=None, and remain undeleted.
Container metadata follows the [Blob container resource contract](https://learn.microsoft.com/en-us/rest/api/storagerp/blob-containers/get?view=rest-storagerp-2024-01-01).
The reader retrieves no artifact bytes, secret contents, or storage keys.

These observations establish selected configuration and grants only.
Inherited assignments, group membership, deny effects, policy changes, data ACLs, resource visibility, and actual token behavior still require complete effective-authority admission.
Actual startup secret resolution, Blob operations, and network reachability remain separate deployment verification.

### Definitions, bounds, and cache

AuthorityDefinitions now returns seven custom roles and eight assignments for discovery, or eight roles and nine assignments for execution.
These are reviewable inputs. They create no Azure resources or grants.
The existing core Container Apps packaging keeps its separate built-in role bindings.

For one shared vault, authority admission reads 20 selected resources in discovery or 24 in execution.
Using separate vaults adds one selected resource.
Vault metadata uses API 2023-07-01. Account and container metadata use 2024-01-01.
Existing identity and role API versions remain unchanged.
Reads retain the existing five-second joined bounds, scoped transport, response limits, and redacted errors.
Construction acquires no token and calls no service.
The reader copies execution account and runtime-identity pins to preserve admitted settings against later caller mutation.

Successful authority evidence has the existing 15-second bound from observation start, including creation and import.
After expiry, a failed current observation cannot reuse earlier success. Failures remain uncached.
Domain authorization and remaining effective-authority protection remain uncached.
Runtime.Close closes the existing authority reader and rejects further cached results.

## Tutorials

### Local verification

Three new checks and eleven affected checks pass with race detection in 4.191 seconds.
They cover mode-specific grants, copied settings, scoped permissions, vault and account drift, expired evidence, admission composition, and shutdown.
Affected checks retain native control and timer separation, identity bindings, cache sharing, uncached protection, explicit credentials, and database admission.
The central discovery composition observes 157 unique selected paths.
Evidence is output/azure-sandbox-host-access/summary.json and local.log.
Checks use local SQLite, fake credentials, and in-memory metadata. Owned readers close through cleanup.
No cloud service, secret endpoint, Blob operation, database connection, browser, container, UI build, demo restart, or remote push runs.

### Continue activation

Complete remaining import and timer identity attachment, inherited authority, deny effects, protected writes, and trusted-consumer ownership.
Bind native HTTPS-probe target routing to registered-target and request-policy contracts.
Then complete native command selection and deployment packaging.
Verify actual SQL access, secret startup, Blob operations, DNS, routing, preview compatibility, and independent timers in the separate deployment phase.

Related guides: [selected authority](azure-sandbox-authority.md), [controller host](azure-sandbox-host.md), and [runtime](azure-sandbox-runtime.md).
Acceptance 51 advances. The total remains 48 verified of 57, with nine open.
