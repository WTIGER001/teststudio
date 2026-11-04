# Coded API request policies

Use test type api with the Playwright runner for request-based TypeScript tests.
The new request policy extends the existing single-origin environment execution profile.
The browser recording profile remains separate. It does not inspect encrypted request methods or paths.

## Current delivery

The policy, trusted gateway, source preparation, API starter endpoint, and runtime compatibility checks are implemented.
The dedicated API choice and guided authoring controls remain pending in Create test.
Full worker verification, automatic diagnostic redaction, credential injection, and environment-backed Go clients remain pending.
Acceptance 35 stays open.

Code authors can upload complete source bundles or registered repository manifests through Import and sync tests.
The normal reviewed preparation, publication, optional activation, and explicit run workflow applies.
Use the Playwright editor's retained-source copy to edit an uploaded API workspace.
Keep source code, request rules, and safety labels under review together.

## API starter endpoint

Read GET /api/v1/projects/{project}/suites/{suite}/playwright-drafts/starter?type=api.
Omit type or use type=browser for the existing offline browser starter.
Unknown or repeated type parameters fail validation. The endpoint requires the normal source-read authority and session.

The API starter contains a complete TypeScript workspace, package declarations, manifest, and catalog.
Its test uses Playwright's request fixture with a relative /health path and explicit assertion steps.
It does not request a page or launch a browser.
The generated catalog starts with needs-review safety and turns off screenshots, traces, and video.
Review the endpoint semantics before assigning safe status.

Choose a permanent draft identifier. Save through the normal revision-controlled Playwright draft endpoint.
Prepare the exact saved revision with its request key. Review and publish the prepared source separately.
Reading, saving, and preparing source do not start a run.

## Manifest policy

The suite uses type api, runner playwright, and buildProfile playwright-environment.
Its browser field identifies the registered endpoint and its request restrictions:

```json
{
  "endpoint": "api",
  "requests": {
    "rules": [
      {"method": "GET", "pathPrefix": "/v1/items", "readOnly": true, "maxBodyBytes": 0},
      {"method": "POST", "pathPrefix": "/v1/search", "readOnly": true, "maxBodyBytes": 4096}
    ],
    "maxRequests": 20,
    "maxResponseBytes": 65536
  }
}
```

An API policy accepts 1 through 64 unique method and path-prefix rules.
Prefixes match a complete path or its slash-separated descendants. /v1/items does not permit /v1/items-other.
Paths must be canonical. Encoded paths, dot segments, repeated separators, fragments, and backslashes fail closed.
When several rules match, the most specific path prefix sets the payload limit.

GET, HEAD, and OPTIONS prohibit request bodies. POST can declare reviewed read-only semantics.
PUT, PATCH, and DELETE cannot be labeled read-only. CONNECT and protocol upgrades are unavailable.
Read-only rules reject common method-override headers.
The review label describes application semantics. A GET method alone does not prove that a request has no side effects.

Request bodies have per-rule limits up to 1 MiB. The gateway reads and validates the complete bounded body before upstream transmission.
Request trailers and compressed payloads are unavailable. The gateway requests identity response encoding and rejects compressed upstream responses.
Responses have limits up to 4 MiB. Each attempt permits 1 through 1000 upstream requests and at most four concurrent gateway requests.
The gateway does not follow redirects. Redirect statuses remain visible, while Location and the original redirect body are withheld.
The caller receives a fixed policy message instead. Nonredirect response headers and bounded bodies remain available for assertions.

## Direct request transport

The execution container retains network=none. The trusted gateway connects through an owned Unix socket and validates every upstream destination.
The runtime supplies a local relay as the request fixture's baseURL.
Use relative paths with the configured request fixture. TESTSTUDIO_API_BASE_URL and TESTSTUDIO_BASE_URL identify that guarded relay.
Independent clients can also use the relay. They cannot obtain an unrestricted CONNECT tunnel through it.

The gateway sets the registered upstream origin and validates its TLS trust, hostname, and certificate dates.
Its image includes the pinned distribution's standard certificate authorities.
Private-address access still requires an explicitly permitted environment. Metadata and link-local addresses stay blocked.
Native HTTP custom CA bindings do not apply to this separate execution profile.

Protected environments require safe test and dependency labels, protected-execution authority, and a valid policy whose every rule is reviewed read-only.
A later protection change rechecks that policy before dispatch and each attempt.
Origin-only API profiles cannot claim protected request enforcement.
Mutating request policies remain available only for permitted unprotected targets.

## Runtime and recovery

Both the source runtime and gateway image must declare teststudio.api-policy=request-policy/v1.
Admission and timing checks reject older images rather than silently ignoring request rules.
Build the updated trusted images, then restart the configured API and source preparation worker with their immutable identifiers.
Reprepare API suites that use an older runtime.

The worker compares the prepared manifest policy with its frozen execution input.
Policy changes enter a new prepared release. They do not rewrite existing runs or timing definitions.
Cancellation, deadlines, owned-resource cleanup, result steps, attempts, and evidence use the existing local execution contracts.
Hosted deployment verification remains separate work.

## Focused local checks

Policy, gateway, compatibility, starter, and source-contract checks use local fixtures only.
Run the targeted commands in [Self-testing](self-testing.md).
Full API worker and authoring workflow verification remains the next delivery step.
