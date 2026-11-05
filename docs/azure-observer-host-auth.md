# Azure observer host platform authentication

## Required reported state

TestStudio manages sign-in, server sessions, project permissions, and private report authorization.
The core Azure API and artifact apps therefore explicitly disable Container Apps platform authentication in authConfigs/current.
The core app module declares that child resource for both roles, including installations with the optional observer disabled.
This does not disable TestStudio authentication or change its OIDC and local-account configuration.

ObserverHostGuard now requires the selected API app's current authentication resource on every successful inspection.
Its identifier, name, and type must match the selected app and authConfigs/current.
Platform.enabled must be explicitly false. True rejects ownership. Missing identity or platform metadata remains unavailable.
A missing resource, denied read, provider error, redirect, or canceled operation cannot substitute for an explicit disabled response.
The profile offers no setting to accept platform authentication in place of TestStudio's own session contract.

Microsoft's [authentication GET contract](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/container-apps-auth-configs/get?view=rest-resource-manager-containerapps-2025-07-01#authplatform) defines the enabled flag and resource endpoint.
The check observes that reported flag. It does not prove the behavior of deployed proxies, tokens, or session endpoints.
Inactive provider settings are not evaluated as an alternative authentication policy.

## SDK boundary and returned metadata

ObserverHostReader requires Auth as part of its contract. It has no optional missing-method fallback.
SDKObserverHostReader.Auth takes the selected application identifier, not a child-resource identifier.
It uses the existing Container Apps SDK and API 2025-07-01 to GET exactly SELECTED_APP/authConfigs/current.
The transport permits no auth collection, alternate app, alternate configuration, mutation, secret operation, or extra query.
Environment and route paths remain outside this host reader's allowlist.

The call has a five-second context and the existing four-megabyte response limit.
Retries and provider registration remain disabled. Redirects are never followed.
A provider 404 becomes unavailable for this configuration check. It does not imply authoritative worker absence.
The general job SDK keeps its existing missing-worker semantics.

The reader returns only resource identifier, name, type, and the platform enabled flag.
It drops provider configuration, registration and encryption references, login settings, system metadata, and other authentication properties.
It retrieves no secret values and returns fixed public error categories without provider messages.
Successful state is not cached or persisted.

## Bound session cost and packaging

Each host Check completes the existing read-deny and identity chain once.
It then reads app, revisions, and current authentication concurrently through the same verified observer credential.
Changed authentication rejects the next check. A restored explicit false value can pass a later fresh inspection.
A failed parent chain prevents all three host reads. Binding creates no observer request and repeats no bootstrap inspection during Check.

The host-specific cost is now three ordinary GETs, with at most five requests including bounded revision paging.
The composed host ceiling increases from 2220 to 2221 requests within the unchanged 135-second context.
The session retains 22 readers. No SDK dependency or role grant is added.
The separate bootstrap remains 125 seconds and 2200 requests. The diagnostic read-deny session remains 130 seconds and 2216 requests.

ObserverHostSettings.AuthDefinition constructs the fixed disabled SDK resource without a provider operation.
The core package declares the corresponding child resource after its parent app.
The existing [runtime contract](azure-observer-host-runtime.md), identity lifecycles, image, healthy revision, and installation references remain required.
This change enables no native coded execution and installs nothing during local verification.

## Local evidence and remaining work

Three new focused checks cover explicit selected state, request restrictions, reduced metadata, fresh bound checks, and parent gating.
Seven affected host and parent-gating regressions update only the changed read contract and request counts.
One reader check exposes the missing-resource mapping. After correction, only that check and its bound-session check repeat.
The changed Bicep module compiles once with pinned 0.48.1 and passes package contracts without warnings or registry restoration.

These observations do not prove deployed authentication behavior, secret contents, effective grants, or app-management authority.
The artifact app receives an explicit disabled resource in packaging, but this API-host observer does not inspect its live state.
Other identity consumers, protection ownership and management authority, trusted networking, execution isolation, and full native wiring remain open.
Keep real Azure verification separate. Acceptance 51 and supporting packaging under 57 advance; the total remains 48 verified of 57.
