# Identity providers

Open Identity providers from installation navigation.
Provider read authority permits inspection. Provider management authority permits authorized configuration changes.
Project administration alone grants no installation provider authority.
Local accounts and OIDC accounts use the same internal directory and project permissions.

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
