# Azure observer host runtime contract

## Boundary

The [host observer](azure-observer-host.md) now checks a fixed API runtime contract against app configuration and its active revision.
Matching app and revision templates alone cannot establish this contract. Both must match the independently constructed expected definition.
The selected identity metadata must also match the configured Blob runtime client.
No new SDK read, permission, reader, dependency, or request allowance is added.
Azure coded execution remains disabled. Acceptance 51 and supporting packaging under 57 advance; 48 of 57 remain verified.

## Required runtime settings

ObserverHostSettings now requires runtime with these fields:

| Field | Requirement |
| --- | --- |
| runtimeClientId | Nonzero canonical GUID for the selected core runtime identity. It differs from observer and backend runtime clients. |
| databaseSecretUrl | Unversioned HTTPS Key Vault database secret reference. |
| oidcConfigurationSecretUrl | Distinct unversioned Key Vault sign-in configuration reference. |
| oidcClientOneSecretUrl | Optional Key Vault reference for the first confidential OIDC client. |
| oidcClientTwoSecretUrl | Optional Key Vault reference for the second confidential OIDC client. |
| blobServiceUrl | Canonical public Azure Blob service URL without a query or credential. |
| evidenceContainer | Valid dedicated Blob container name. The artifact prefix remains teststudio. |
| uiOrigin | Canonical HTTPS origin whose hostname starts with the selected app name. |
| artifactOrigin | Separate app hostname under the same configured domain. |
| bootstrapIssuer | HTTPS bootstrap issuer, with an optional issuer path and no credentials, query, or fragment. |
| bootstrapSubject | Nonempty bootstrap subject, at most 512 bytes, without control characters. |
| localAccounts | Explicit expected local sign-in policy. False remains the default. |
| replicas | Expected fixed replica count from one through ten. Zero selects the existing default of one. |

All selected secret references use the same vault and distinct names, matching the current core package.
An omitted optional client reference creates no secret or environment entry. Public OIDC clients therefore need no client secret.
The API and artifact origins differ from the worker exchange and trusted backend origins.
This profile follows the core package's separate app hostnames. Custom-domain profiles require their own reviewed contract.
Invalid configuration rejects before bootstrap inspection or credential construction.

## Expected service definition

ApplicationDefinition constructs the expected SDK resource without sending a request.
It uses the existing immutable service image and preserves that image's entrypoint. An app command override rejects observation.
The exact arguments select the API role, PostgreSQL requirement, installation origins, Azure execution location, artifact configuration, and sign-in references.
The expected bootstrap identity and local-account policy also form part of that argument contract.
Additional arguments, including local runner enablement, reject observation.

The only environment variables are the database and OIDC secret references, non-secret Blob configuration, and selected optional OIDC client references.
The Blob configuration selects Azure, the configured service and container, prefix teststudio, and the exact runtime client.
Inline credentials, alternate secret references, duplicate or additional variables, proxy injection, and substituted storage settings reject observation.
Secret entries use only the selected URLs and startup identity. Registry pull uses only the configured registry and pull identity.
The observer identity receives no startup secret or registry role through this definition.

The template requires one service container, no init container or volume, 0.5 CPU, 1Gi memory, and 90-second termination grace.
Startup and readiness probe /healthz. Liveness probes /livez. All use port 8040 with the core package's fixed timing values.
Replica minimum and maximum match the selected count. Additional scale rules reject observation.
Ingress uses external HTTP transport with HTTPS required, target port 8040, the configured hostname, and latest-revision traffic at 100 percent.
Extra ingress paths, ports, identities, app services, Dapr, runtime configuration, or unrecognized SDK properties cannot substitute for this definition.
The existing lifecycle and healthy Single-revision checks remain required.

The comparison normalizes environment, secret, identity, and probe order, plus Blob JSON formatting and documented shared service defaults.
It accepts the reported Consumption workload profile and inactive-revision limit from one through 100.
The complete revision collection still permits at most four records; inactive rows require explicit zero traffic and zero replicas.
It retains the existing service normalization for reported ephemeral storage. This stage does not establish filesystem or image-entrypoint behavior.

Microsoft's [Container App GET contract](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps/get?view=rest-resource-manager-containerapps-2025-07-01) describes the reported configuration and template fields.

## Package output and freshness

The optional observerBinding output now includes runtime using the same core parameters as the API app.
It contains resource and secret references, origins, identity client, bootstrap identity, local-account policy, and replica count.
It contains no database value, OIDC configuration body, client secret, access token, or storage key.
The observer option remains disabled by default. The output remains incomplete until installation supplies the preceding network and protection bindings.

Each bound Check reads fresh app and revision state through the selected observer credential.
Changed app and revision settings reject even when they agree with each other. Restored valid state can pass the next inspection.
The host session still uses 22 readers, a 135-second context, and at most 2222 requests.
Its ordinary host-specific cost is now four GETs, including [platform authentication](azure-observer-host-auth.md) and [controller route absence](azure-observer-host-routes.md). Initial binding retains one separate bootstrap inspection.

## Remaining installation evidence

These checks inspect reported configuration. They do not retrieve or verify database or OIDC secret contents.
They do not establish actual image behavior, token issuance, deployed platform authentication behavior, effective grants, or authority to alter the app.
Public hostname ownership and certificate behavior require separate deployment verification.
Complete isolation, other identity consumers, protection ownership and management authority, preparation networking, and native command wiring remain open.
No provider service or deployment is contacted during the local fixture and package checks.
