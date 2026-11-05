# Local Kubernetes identity and private evidence

## Introduction

Two independent OIDC providers now authenticate against the same PostgreSQL directory in the dedicated Kubernetes installation.
Normal code flow, PKCE, private TLS validation, issuer separation, mapped project access, and private evidence authorization pass actual verification.
No test runs are created. The check reuses an existing screenshot and its retained checksum.
Acceptance 55 is complete. The total is 50 verified of 57, with seven open items.
Acceptance 54 remains open for its required cross-profile scope. Azure implementation and live cloud verification remain deferred.

## Concepts

### Configured issuers

| Provider | Issuer | Client | Callback path |
| --- | --- | --- | --- |
| local-k3s | https://identity.teststudio.test | teststudio-local-k3s | /auth/callback/local-k3s |
| local-k3s-review | https://identity-two.teststudio.test/review | teststudio-local-review | /auth/callback/local-k3s-review |

The original fixture deployment and signing key remain unchanged.
The second fixture runs separately with a new RSA signing key and its own TLS certificate.
It uses a newly authored public configuration file in tests/kubernetes/fixture/secondary-identity.json.
Both fixtures emit the same verified email and subject values. Their distinct issuers resolve to different directory users.
They also use the same key identifier. Each issuer still uses its own signing key and keys endpoint.
The initial bootstrap profile keeps its existing metadata. Sign-in does not overwrite that profile to match a fixture email.
The reviewer profile receives its configured email. Identity separation follows issuer and subject, not a profile email comparison.

The second address resolves only inside the dedicated cluster and the verifier's owned loopback connection.
No host DNS or system certificate trust changes occur.
The service already trusts the local fixture authority through the installation's mounted OIDC CA setting.
Discovery, code exchange, keys, and callback handling retain normal certificate and hostname validation.
The TLS private key stays in a protected local directory and a namespace Secret. It enters no image or committed configuration.

The identity fixture automatically authorizes a fixed test subject. It is a local protocol fixture, not a production identity service.
Its new optional configuration supports a fixed issuer realm path, client, callback, identity claims, and group claim values.
One focused Go check covers the default and second realm protocol, including emitted issuer, audience, email, subject, nonce, and callback state.

### Directory and project access

The second provider is registered through the normal managed provider API with an audit reason.
Its first sign-in creates an unprivileged user. The same subject value does not transfer the first provider's administrator authority.
Project reads and installation directory access reject that initial user.

An installation administrator creates group k3s-provider-reviewers inside project k3s-coded.
Its permanent owner is oidc:local-k3s-review. The provider maps its exact reviewers claim to that group.
The group receives the project's predefined Viewer role through the ordinary role-assignment API.
A fresh second-provider sign-in resolves the same reviewer user and synchronizes provider-owned membership.
It can read that project's existing run and private evidence.
It cannot read project k3s-integration, list installation users, or manage identity providers.
The first administrator retains its independent project and installation authority.

### Private report authorization

The reviewer reads the existing browser run 20261105T151447.417956930-cfc4c366f3c9ee354dd21551.
The normal artifact-access API grants a private screenshot download through the separate evidence origin.
The downloaded bytes match the artifact's stored size and SHA-256 checksum.
The verifier then removes the project role with its current assignment revision and an audit reason.
The same reviewer session immediately loses project, run, and new artifact-grant access.
The previously issued artifact URL also rejects access while that permission is withdrawn.
The check does not infer permanent URL invalidation after later permission restoration.

The Viewer assignment is restored after the check. Current authorized report access works again without creating another run.
No provider token, request-forgery token, session cookie, or artifact URL is written into the evidence summary.
Private download URLs remain in memory. All owned verification sessions sign out afterward.

### Acceptance 55 audit

The actual hosted service uses installation-specific origins, client identifiers, callback addresses, provider registration, and mounted private certificate trust.
Both managed identities resolve through the shared directory and permission service.
Project authorization protects report metadata, grant creation, and private artifact delivery, including an existing session after role revocation.
This proves the criterion's hosted identity and private report contract through actual Kubernetes routing and PostgreSQL.
Other deployment profiles remain under the separate cross-profile criteria, including 54 and 57.
The local result does not establish AWS or Azure deployment acceptance.

## Tutorials

### Inspect the retained result

Evidence is output/local-k3s/identity/summary.json and references.json.
issuer-links.json records the distinct issuer links and directory identifiers without tokens.
initial-profile.json and mapped-profile.json retain the before and after directory views.
reviewer-private-evidence.png contains the checksum-verified existing screenshot.
fixture-protocol-check.log records the focused default and realm protocol check.
installation.json records the second fixture image, issuer, service address, and setup state.
TLS keys remain under .data/local-k3s/secondary-identity-tls with private permissions.

### Resume setup or inspect the same scenario

```sh
python3 scripts/setup-kubernetes-identity.py
python3 scripts/verify-kubernetes-identity.py
```

The setup command builds only the small identity fixture and imports its verified ARM64 image through the shared image helper.
It preserves the original issuer deployment and application service image.
The verifier reuses the registered provider, user, mapped group, assignment, and existing test evidence.
Completed revocation proof remains recorded. Ordinary reinspection does not withdraw the restored role again.
No UI build, browser automation, cloud connection, or test execution is required.

### Continue the plan

Audit the remaining seven deployment-specific criteria against implementation files and retained evidence before adding more checks.
Separate missing implementation from verification that requires deferred cloud targets.
Keep the full 57-item scope and Azure deferral intact.

Related guides: [provider management](providers.md), [local controller failover](local-kubernetes-controllers.md), and [acceptance ledger](implementation.md).
