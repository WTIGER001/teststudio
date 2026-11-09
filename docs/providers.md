# Identity providers

Open Identity providers from installation navigation.
Provider read authority permits inspection. Provider management authority permits authorized configuration changes.
Project administration alone grants no installation provider authority.
Local accounts and OIDC accounts use the same internal directory and project permissions.

## Generic OIDC boundary

TestStudio integrates with an OIDC issuer through discovery, authorization-code flow, PKCE, token verification, and a managed application session.
The issuer owns its sign-in page, account chooser, upstream federation, and authentication policy.
Cognito can offer Google or other methods without a Google integration in TestStudio.
Configure the issuer and client identifier in Identity providers. The display name is configuration, not application logic.
TestStudio does not choose an upstream social provider or accept its password.
It validates the configured issuer, audience, signature, expiry, nonce, state, and transaction binding before creating a session.
A signed-in identity grants no project or installation authority by itself. Current TestStudio permissions still apply.

Use Your account and Link another sign-in to add an organization identity to an existing account.
Complete fresh proof of the current and new identities, then explicitly confirm the link.
This preserves the current internal account and its roles. Matching email addresses never merge accounts.

## Register and configure

Select + Register provider.
Enter a permanent identifier, display name, exact issuer URL, and client identifier.
Use HTTPS outside loopback. A loopback provider can use HTTP during local development.
The issuer must not contain credentials, a query, or a fragment.

Keep Provider disabled while preparing the configuration.
Requested scopes use space-separated tokens. OpenID is always requested once.
An empty scope list uses OpenID, profile, and email.
Create pending user profiles at first sign-in controls automatic provisioning.
Successful sign-in alone grants no project or installation roles.

Enter Provider change reason and select Register provider.
Open the saved provider. Register its displayed Callback URL with the external provider.
The callback follows the configured public installation origin and permanent provider identifier.

## Sign-in page and automatic SSO

Local password sign-in appears only when the operator enables managed local accounts with --local-accounts. Disabled local password sign-in has no username or password form.
Enabled providers appear as SSO buttons unless Hide from sign-in page is selected. Use that setting to remove a duplicate or recovery-provider button. Hiding preserves existing identities and permissions and permits an explicit /auth/login/PROVIDER sign-in link. Disabling the provider also blocks that explicit sign-in.
The loopback --local development administrator is a separate operator mode. Its shortcut is omitted when the password form is enabled. Keep it out of hosted installations.

Open the normal organization provider and select Automatically open this SSO provider. Save with a reason. When managed local passwords and the loopback administrator are both disabled, a new unauthenticated visit opens this provider directly without displaying the TestStudio login form.
Choose one visible, enabled provider for automatic sign-in. With no selection or multiple automatic selections, the normal choices remain. Local password sign-in always keeps the login form visible, even when SSO bypass is selected.
Cancellation, expired requests, provider failures, and sign-out keep manual sign-in and readable feedback. Dismissing feedback does not automatically start another sign-in. The requested local path, query, and fragment survive recovery. Existing sessions never trigger an automatic login.
Public status documents and embedded pages never start automatic SSO. Provider tokens and credentials stay on the server. Bypass changes navigation only; it grants no access or membership.
Provider edits retain current authority checks, audit reasons, revision conflicts, and authentication invalidation. Editing your current provider requires a fresh sign-in.

## Server credentials

The browser accepts no client secret values.
Select No client secret for a public client, or select an operator-approved credential reference.
The server resolves the reference during provider sign-in.
Provider responses, audit records, and public choices never contain the resolved credential value.
The credential enters only the server's token exchange. Browser redirects, cookies, sessions, and stored provider settings contain no resolved value.
Discovery and credential-resolution failures return fixed error messages. They exclude upstream response bodies and resolver diagnostics from application errors.
A configured reference must resolve to a nonempty value. An empty result blocks sign-in rather than selecting public-client behavior.
Token exchange and identity verification also use fixed failures. Browser recovery continues through the readable sign-in outcome.

The operator supplies the environment variable through the server's protected runtime configuration.
Approve its reference when starting the service:

```sh
bin/teststudio --oidc-secret-refs env:TESTSTUDIO_OIDC_SECRET
```

Separate multiple references with commas.
References declared in an operator-supplied `--oidc-config` file are also approved.
That file registers initial providers. Ordinary startup preserves existing managed records.
Removing a reference prevents new sign-in that needs it. Review affected providers before changing operator configuration.
A disabled existing provider can retain its previous unavailable reference for inspection and repair.
Enabling requires an approved reference or no client secret.

## Private provider certificate authorities

Use --oidc-ca-file for a private provider's mounted PEM certificate authorities.
The service adds those authorities to system trust only for OIDC discovery, token exchange, keys, and provider logout discovery.
Certificate hostname, lifetime, and chain validation remain enabled. The minimum protocol is TLS 1.2.
The file must contain only valid certificate authorities within 64 KiB. Invalid configuration blocks startup without exposing file contents or paths.
The service loads the file at startup. Restart the service after certificate changes.
This setting does not configure browser trust or native HTTPS probe trust.
Configure probe authorities separately by project and exact origin in [HTTP checks](http-checks.md).

The Kubernetes chart accepts oidc.caSecret as an existing namespace Secret containing ca.crt.
The chart mounts that certificate file read-only and supplies --oidc-ca-file.
The chart creates no certificate or credential value. See [Kubernetes](kubernetes.md).

## Provider-owned groups

Open a saved provider and use Create a provider group.
Enter its identifier, name, and optional existing project identifier.
An empty project selects installation ownership.
The service stores `oidc:PROVIDER_ID` as its permanent membership owner.
Group creation requires installation provider management and appropriate group management in the selected scope.

Creation grants no membership or roles.
Ordinary group editors can change permitted display fields but cannot change membership ownership.
Local membership editors reject provider-owned groups.
Existing local groups cannot be converted into provider groups.

Assign roles through Project access or Installation access.
Project-owned groups remain restricted to their project.
Installation groups can receive separately authorized project assignments.

## Exact external mappings

Select Edit provider. Enter the top-level Group claim name.
Select + Add group mapping and enter an exact external value and destination group identifier.
Each external value can appear once. Multiple values can map to the same group.
Use at most 40 mappings. Wildcard expansion is not supported.
The destination must belong to this provider.
Managing mappings requires group read, group management, and membership management at the destination's actual ownership scope.
Exact group-object denies remain authoritative.

Membership lifetime is at most 3600 seconds. Zero uses 1800 seconds.
Token expiry can shorten the lifetime.
Missing, invalid, or incomplete required claims withdraw provider membership instead of preserving earlier elevated access.
Unmapped external values grant no access.
Local membership remains independent.

Disabled configuration can retain archived destinations for review.
Restore mapped groups and projects before enabling the provider.
Clear Provider disabled, enter a reason, and select Save provider.
Verify access through a fresh provider sign-in.

## Changes and recovery

Every save invalidates earlier login transactions, provider sessions, and synchronized memberships.
This also applies to a display-name edit.
Restoration does not revive earlier cookies or earlier membership proof.
A fresh validated sign-in updates the provider revision and membership provenance.
The last usable installation administrator remains protected.
A group-only administrator can require an independent administration path before editing that group's provider.

Refresh provider updates the stored view without replacing an opened editor revision.
A conflict preserves your fields and blocks another save.
Select Reload stored provider to inspect its current revision before retrying.
An unconfirmed response also blocks repeat submission until explicit review.
Revoked authority clears private editors and stored views.
Retained read authority permits a new read-only inspection.

Editing your current provider can invalidate your own browser session.
The application clears private pages and reloads current public sign-in choices.
Use a fresh provider sign-in or another configured sign-in method.

Disabling withdraws provider authentication while retaining its configuration and permanent identifier.
The API provides no provider deletion or identifier reuse operation.
Changing the issuer does not link existing accounts or transfer their roles.
Equal email addresses never merge different issuer-subject pairs.
Users without project access see Access not granted and can ask a project owner to add their account.

## API

| Method | Path | Purpose |
| --- | --- | --- |
| GET | `/api/v1/identity-providers` | Scan readable provider configuration and approved credential references. |
| POST | `/api/v1/identity-providers` | Register one provider with explicit state and reason. |
| GET | `/api/v1/identity-providers/{provider}` | Inspect stored configuration and current management capability. |
| PUT | `/api/v1/identity-providers/{provider}` | Save configuration with a positive current `If-Match` revision and reason. |
| POST | `/api/v1/groups` | Create a group with optional permanent `managedBy` provider ownership. |

Provider scans use opaque actor-bound cursors and bounded limits from 1 through 1000.
The default limit is 30. A filtered page can be empty while more readable records remain.
Public `/auth/providers` exposes only enabled provider names, identifiers, and available sign-in modes.

Provider changes use one metadata and permission transaction for authorization, configuration, audit, directory revision, and administrator protection.
A rejected change preserves earlier state and revisions.

## Local verification

```sh
bash scripts/test-self.sh --test=STUDIO-PROVIDERS-001
bash scripts/test-access.sh
```

The browser workflow uses two new local issuers and a disposable TestStudio instance.
It requires no cloud account or remote identity provider.
Provider logout, explicit account linking, and cross-profile provider acceptance remain separate pending work.

## Initial operator configuration

The API accepts an initial JSON provider array through --oidc-config FILE or --oidc-config-env VARIABLE.
Choose one source. The environment form supports secret-integrated container installations without a configuration file or shell.
Each source has a 64 KiB limit. Selected environment variables must exist and contain nonblank JSON.
The complete list must contain one to 100 valid providers with unique identifiers.
Unknown fields, inline client secret fields, malformed JSON, and trailing JSON fail before database initialization.
Configuration read errors exclude values and parser diagnostics.

Each provider uses the same fields and validation as provider administration.
Use env:VARIABLE references for confidential clients. Supply resolved values through the protected runtime environment.
Initial configuration approves its declared credential references. --oidc-secret-refs can approve additional mappings for later provider administration.
Initial provider records are inserted only when absent. Restarting does not replace managed configuration or restore disabled providers.
Keep each configured client callback aligned with the installation's public origin.
See the [AWS core profile](aws.md) for ECS secret mappings and current deployment limits.

## Actual local Kubernetes profile

Two independent issuers now pass normal configured sign-in and project-scoped access through the deployed Kubernetes installation.
Provider-owned group mapping permits one project's private reports. Foreign-project and installation administration remain denied.
Existing-session and previously granted artifact access reject while the project role is removed.
The role is restored after verification. No test runs are repeated.
See [local hosted identity](local-kubernetes-identity.md) for evidence, issuer configuration, and remaining cross-profile scope.

## Hosted selection

On 2026-11-09 the user selects SSO-only automatic Organization sign-in for MMDev. Local password sign-in is off. The duplicate TestStudio sign-in provider is hidden from the page. Organization is the automatic SSO target. Recovery after cancellation, expiry, or sign-out shows one Organization button.
Release 0.12.1-dev verifies the actual hosted redirect, protocol parameters, cancelled transaction, preserved destination, reload, mobile help, and complete owned cleanup. The hidden provider's explicit recovery flow remains available to operators. See [MMDev](aws-mmdev.md).
