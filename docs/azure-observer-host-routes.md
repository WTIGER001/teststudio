# Azure observer controller route absence

## Selected profile

The staged core API profile uses each app's own ingress and revision traffic settings.
It now requires an explicitly empty HTTP route configuration collection in the selected controller environment.
This matches the no-route contract already used by the dedicated exchange and backend environments.
It checks reported absence, rather than interpreting route conditions or declaring a route harmless.

The requirement applies to the complete selected environment, including routes for unrelated apps.
Use a controller environment without those overrides. The inspector offers no route allowlist or per-rule exception.
The core package creates no HTTP route configuration. It also removes or changes no existing environment route.
Installation tooling must select a suitable existing environment before completing host admission.
This restriction does not change ordinary test targets or their application environments in the TestStudio catalog.

Microsoft's [HTTP route list contract](https://learn.microsoft.com/en-us/rest/api/resource-manager/containerapps/http-route-config/list?view=rest-resource-manager-containerapps-2025-07-01) defines the environment-scoped collection and its continuation field.

## Required observation

ObserverHostReader now requires RoutesAbsent with the exact Config.controllerEnvironment identifier.
Each successful host inspection must receive a response containing an explicit empty value array and no nonempty nextLink.
Any non-null route object rejects ownership, even if incomplete, pending, or apparently unrelated to the selected API app.
Missing or null arrays, null-only rows, provider errors, denied reads, missing collections, and incomplete empty pages remain unavailable.
A route object takes precedence over a continuation because that response already contradicts absence.

The reader fetches exactly one page. It never follows nextLink, including a link to the same collection.
It returns no route names, hostnames, targets, rule conditions, provider messages, or response bodies.
Successful absence is not cached or persisted. A later observation can reject newly reported state and accept restored absence.

## SDK request boundary

SDKObserverHostReader reuses its existing HTTPRouteConfig client and API 2025-07-01.
It permits one additional selected GET path:

```text
SELECTED_CONTROLLER_ENVIRONMENT/httpRouteConfigs
```

The request contains only api-version. Filter, top, skip token, route-item reads, alternate scopes, and mutations reject before transport.
No pagination permission is added to this path. The existing revision collection retains its separate bounded paging contract.
The operation uses a five-second context and the shared four-megabyte response limit, with retries and provider registration disabled.
Redirects are never followed. Canceled or failed operations cannot establish absence.
The method returns only a fixed public error category or success.

## Bound host integration and cost

Each host Check completes the preceding protection, coverage, identity, and inventory chain once.
It then reads app, revisions, current authentication, and controller route absence concurrently through the selected observer credential.
A failed parent chain prevents all four host reads. Check repeats no bootstrap inspection and constructs no new credential.
The existing fixed runtime, lifecycle, selected image, and healthy revision contracts remain required.

The ordinary host cost is now four GETs. The maximum host-specific cost is six requests with bounded revision paging.
The composed ceiling becomes 2222 requests within the unchanged 135-second context.
The session retains 22 readers. Bootstrap remains 125 seconds and 2200 requests; diagnostic read-deny sessions remain 130 seconds and 2216 requests.
No SDK dependency, grant, new reader, application setting, or package build is added.

## Local verification and limits

Three new focused race checks cover explicit absence, rejected state, same and foreign continuations, exact request scope, cancellation, fresh bound checks, and parent gating.
Seven affected host and protection regressions update their expected read counts. All ten pass in one targeted local selection.
No browser, UI build, package compilation, image build, container, registry push, deployment, or provider call runs.

A complete reported empty collection does not prove authorization coverage or prevent later administrative writes.
It does not establish external proxy behavior, actual ingress enforcement, certificate trust, or protection-management authority.
This API-host observer does not inspect every artifact-app setting or establish exclusive observer use across all resource types.
Full visibility, clean installation provenance, execution isolation, trusted preparation networking, and native wiring remain required.
Azure coded execution remains disabled. Acceptance 51 advances; the total remains 48 verified of 57, with nine open.
