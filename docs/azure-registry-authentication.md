# Trusted Azure registry authentication

## Current boundary

The adapters/registry/acr package supplies managed-credential authentication for the existing OCI inspector and source publisher.
It supports the Azure worker profile's Linux AMD64 Go and Playwright images.
It exchanges an explicit trusted Azure credential for registry and repository tokens. It does not use Docker or registry administrator passwords.
The Azure API and preparation runtime can use this client as their Inspector or source registry client.
Native Azure worker command wiring and identity role packaging remain unfinished. Coded Azure execution stays disabled.
The [preparation registry path observer](azure-preparation-registry.md) now inspects selected private network and DNS resources.
It does not automatically gate this client or enable the source queue; native integration remains required.

The [registry service inspector](azure-worker-registry.md) now verifies private access settings, regional data authority, and service-side endpoint approval.
It preserves platform ARM-audience authentication while this trusted client continues to request only the registry audience.

See [Azure job lifecycle](azure-worker-lifecycle.md), [registry inspection](registry-inspection.md), and [source publication](registry-preparation.md).
No cloud connection or registry publication occurs during local verification.

## Explicit installation inputs

| Option | Purpose |
| --- | --- |
| Registry | One lowercase public-cloud ACR login hostname ending in azurecr.io. |
| Tenant | The selected lowercase Entra tenant identifier. |
| RepositoryPrefix | The permitted repository and its descendants. Neighboring prefixes are excluded. |
| Credential | One explicit azcore.TokenCredential for this trusted process. No implicit credential discovery occurs in the adapter. |
| Publish | False for inspection. True only for the separate trusted preparation writer. |
| BlobRedirectOrigins | Up to 16 exact HTTPS Azure Blob or registry data endpoint origins. |
| Transport | An optional trusted HTTP transport for installation trust or local protocol fixtures. |
| Now | An optional trusted clock for targeted verification. |

The caller selects the managed identity explicitly when it constructs the Azure credential.
Use the API's registry reader identity for inspection and a separate preparation writer identity for publication.
Neither credential belongs to the untrusted job container or its environment.
The platform's image-pull identity remains separate and unavailable to container code through lifecycle None.

AcrPull applies to the baseline registry RBAC model. AcrPush supplies the separate preparation writer's pull and push authority.
ABAC registries instead require their corresponding repository Reader and Writer roles with installation-specific conditions.
These client restrictions supplement cloud roles. They do not grant a permission that the selected identity lacks.
Actual role assignment and managed identity behavior still require the separate deployment verification phase.

## Token exchange

The client requests one token from its Azure credential using https://containerregistry.azure.net/.default and the selected tenant.
It uses no ARM token fallback and does not read a registry administrator password.
POST /oauth2/exchange sends access_token grant type, registry service, tenant, and the credential's access token as form fields.
POST /oauth2/token then requests a repository grant using the returned registry refresh token.
These form requests have no Basic or Bearer Authorization header. They stay on the exact configured HTTPS registry.

The ordinary OCI request first receives a Bearer challenge.
The shared parser validates the exact token realm, service, repository scope, and permitted actions before invoking the token callback.
Foreign realms, different services, wider repository scopes, duplicate parameters, and unsupported challenge fields fail before obtaining a credential.
The ACR callback requests pull for inspection or pull,push for publication. It requests no catalog, delete, or wildcard grant.
The resulting repository token authorizes subsequent OCI requests through the existing Bearer path.

The generic OCI Tokens callback is optional. It requires exact TokenURL and TokenService settings and excludes the Credentials callback.
Existing Basic and Bearer credential callbacks retain their previous behavior when Tokens is absent.
No registry-specific workaround changes OCI content validation, upload journals, source approval, or run execution.

## In-memory lifetime and concurrency

The authentication operation has a five-second context bound, including waiting for another refresh.
The Azure credential is consulted before a cached token is reused.
A changed source access token clears the older registry refresh token and all repository tokens.
Missing, expired, or failed source credentials clear the cache. There is no fallback to an earlier cached identity.

Each cache lifetime stays within these bounds:

- The source Azure token's expiry, minus 30 seconds.
- The returned registry token's expiry, minus 30 seconds.
- Five minutes from the registry refresh exchange.

Repository tokens use separate repository and action keys. At most 32 entries remain in the cache.
Concurrent inspections serialize refreshes and reuse eligible grants.
JWT expiry claims bound caching only. They do not replace issuer authentication, signature validation, or registry authorization.
The selected credential and fixed trusted HTTPS exchange establish the authentication path. The registry validates issued Bearer tokens.

Malformed, expired, not-yet-valid, duplicate-key, or oversized token responses fail explicitly.
OAuth responses have a 128 KiB limit. JWT payloads have additional size, depth, and node limits.
Raw tokens and provider diagnostics do not enter serialized client state or normal error messages.
Close seals new authentication requests, clears cached credentials, and closes idle connections. Close is idempotent.
No token cache is stored in the database, source bundle, upload journal, settings file, browser, or worker input.

## Network and write restrictions

The default transport uses system TLS trust, TLS 1.2 or later, bounded connection setup, and no environment proxy.
Registry requests stay within the configured repository namespace and immutable digest paths.
Inspection permits only GET and HEAD metadata requests. It cannot publish through a broader identity's accidental write authority.
Publication additionally permits immutable manifest writes and bounded upload-session operations in that namespace.
It creates no mutable image tags. Upload-session cancellation remains available without manifest deletion authority.

OAuth redirects are not followed. Registry credentials never accompany a configuration-blob redirect.
Blob redirects require an explicit exact origin under Azure Blob Storage or the selected registry's regional data endpoint.
Each redirected request uses GET without Authorization, Cookie, or Proxy-Authorization.
The existing OCI size and SHA-256 checks still validate the returned configuration content.
Do not configure a wildcard endpoint list. Discover and review exact required origins during installation setup.

## Integration

Construct one client per trusted role:

```go
reader, err := acr.New(acr.Options{
    Registry: "installationregistry.azurecr.io",
    Tenant: tenantID,
    RepositoryPrefix: "teststudio",
    Credential: readerCredential,
    BlobRedirectOrigins: reviewedBlobOrigins,
})
```

Pass reader as the Azure provider's Inspector. Pass reader.Client where the existing builder requires an OCI client.
Use a separate credential and Publish: true only when constructing the preparation writer.
Close each ACR client after its owning process stops its operations.
Creating this client alone does not enable worker dispatch or approve a source artifact.

The future native setup must bind actual reader and writer managed identities and roles.
It must retain private journals, source approval, current project authorization, and separate worker network admission.
Independent stop resources and private exchange service packaging remain required before Azure coded execution can start.

## Targeted local verification

Five new focused race checks pass with in-memory OAuth and OCI protocol responses.
They cover both runners, normal supervised-image inspection, source publication, scope separation, and upload-journal completion.
They also cover concurrent reuse, changed identities, cache age and size, source expiry, denied credentials, malformed responses, and foreign redirects.
An approved configuration-blob redirect retains its content checksum and sends no registry credential.
The read-only client performs no source writes. Closing the client prevents further requests.

Four selected existing OCI inspection, namespace, digest, and redirect checks also pass with race detection.
No browser, container, UI build, source-runtime build, image build, real registry, or cloud deployment runs.
The publication fixture validates protocol and metadata behavior. It creates no runnable suite image or real test execution.
Evidence is output/azure-registry-auth/summary.json and the focused command results.
Acceptance 51 advances. The verified total remains 48 of 57.
Live role enforcement, identity endpoints, registry tokens, private data paths, publication, and deployment parity remain unverified.

## Primary references

[Managed registry identities](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-authentication-managed-identity) describes separate reader and writer roles.
[ACR-specific Entra scopes](https://learn.microsoft.com/en-us/azure/container-registry/container-registry-disable-authentication-as-arm) describes the registry resource audience.
[Refresh-token exchange](https://learn.microsoft.com/en-us/rest/api/registry-dataplane/authentication/exchange-aad-access-token-for-acr-refresh-token?view=rest-registry-dataplane-2021-07-01) defines the form exchange.
[ACR OAuth integration](https://github.com/Azure/acr/blob/main/docs/AAD-OAuth.md) describes refresh and repository token requests, JWT expiry, and Bearer challenges.
