# Coded API request policies

Use test type api with the Playwright runner for request-based TypeScript tests.
The new request policy extends the existing single-origin environment execution profile.
The browser recording profile remains separate. It does not inspect encrypted request methods or paths.

## Current delivery

Create test includes an API test choice with Name, Request, and Review steps, followed by the shared source editor.
Manual and scheduled request suites execute through the trusted gateway and retain normal results, timing, history, and trends.
API diagnostics now redact sensitive text and JSON before retention, including during worker recovery.
Environment-backed credential injection now supports Bearer, Basic, and approved API-key headers through the trusted local gateway.
Environment-backed Go clients remain pending.
Acceptance 35 stays open.

## Create an API draft

1. Select Create test, then API test. Choose a destination suite.
2. Select New API draft. Enter its title. The identifier is generated under Advanced details.
3. Continue to Request. Choose the method, relative path, and expected response status.
4. Add JSON request data for methods that permit a body. The data remains in the saved source workspace.
5. Review read-only semantics separately from the method. Writes remain ineligible for protected targets.
6. Use Advanced request policy only when changing the endpoint, allowed path prefix, request budget, or payload limits.
7. Continue to Review. Inspect the request, policy, identifier, and needs-review safety label.
8. Select Create source draft. Review the generated TypeScript, catalog, and manifest in the API test editor.
9. Select Save API draft. Then validate and prepare the saved revision, review its catalog, and publish explicitly.
10. Start a run separately. Select a worker-reachable registered environment with the reviewed endpoint name.

Back preserves your entries. Help does not reset them. Leaving unfinished setup or source changes requires an explicit discard choice.
An invalid Advanced value opens its disclosure and receives keyboard focus.
The review creates no server record until Save API draft. Saving, preparing, and publishing do not start a run.
The source catalog retains needs-review until a reviewer changes and publishes its safety label.
A read-only request policy does not automatically label the test safe.
The initial request asserts its expected status. Add further steps and assertions in the code editor.
Choose Authentication when the API needs a credential. Enter its environment binding name and review it before creating source.
Configure references under Advanced credentials when registering an environment. See [API credentials](api-credentials.md).
Request JSON is serialized literally with application/json content type. The wizard evaluates no scripts or expressions.

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
    "maxResponseBytes": 65536,
    "redactFields": ["privateContact"]
  }
}
```

An API policy accepts 1 through 64 unique method and path-prefix rules.
Prefixes match a complete path or its slash-separated descendants. /v1/items does not permit /v1/items-other.
Paths must be canonical. Encoded paths, dot segments, repeated separators, fragments, and backslashes fail closed.
Requests can retain one trailing slash. Rule prefixes omit a trailing slash except at the root.
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

## Diagnostic redaction

The execution collector applies api-diagnostics/v1 before the result checkpoint, evidence upload, or worker cleanup.
Recovery uses the frozen test and release policy. It applies the same redaction before restoring an observation.
Assertion messages, step titles, step errors, framework status text, stdout, stderr, and evidence warnings pass through this boundary.
Redaction changes no test state, failure kind, duration, retry eligibility, history count, or trend count.

Default sensitive fields include authorization, proxy authorization, cookies, passwords, secrets, tokens, API keys, credentials, and sessions.
Matching ignores case and field-name separators. URL user information and sensitive query values are also removed.
The collector replaces detected values in the other text from the same observation, including reflected assertion values.
Known secret values also support literal, JSON-escaped, and URL-encoded replacement.
The gateway injects selected credentials and removes common reflections before responses reach the worker.
See [API credentials](api-credentials.md#runtime-and-recovery) for the private channel and recovery boundary.

Add up to 32 field names with requests.redactFields in the source manifest.
Each name starts with a letter and contains at most 64 letters, digits, periods, underscores, or hyphens.
These names identify object properties, not JSON paths. Nested objects and arrays retain the same protection.
The prepared release freezes this policy. Prepare and publish a new release to change it.

Only bounded UTF-8 text/plain, application/json, and application/*+json attachments enter API evidence.
Each diagnostic file has a 2 MiB input limit and the configured capture limits still apply.
JSON must parse completely and remain within 64 nesting levels. Numbers retain their JSON precision.
Diagnostic scanning permits 2048 field assignments and 512 learned values within a 1 MiB private-value budget.
Malformed structured sensitive values and exceeded redaction limits omit the affected diagnostic content.
Invalid, opaque, oversized, or unsupported files produce an explicit evidence gap without changing the observed test result.
Request-restricted API workers disable automatic screenshots, video, and traces. The reporter rejects media and binary attachments before copying them.
The collector independently rejects unsupported files before transfer. Text redaction does not sanitize browser images or trace archives.
Browser-typed tests retain their existing media capture behavior.

Results expose diagnosticsPolicy. Retained worker artifacts expose redactionPolicy. Both record api-diagnostics/v1.
Artifact retries cannot replace that policy marker. Generated diagnostic filenames avoid retaining source attachment names.
Existing stored evidence remains unchanged. Its absent marker does not claim redaction.

Use dedicated test data and scoped credentials. Keep sensitive application fields in the reviewed redaction list.
This policy filters conventional structured diagnostics. It does not sandbox published code or detect deliberately encoded or unlabeled private data.

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
The local worker verification checks protected manual and scheduled runs, rejected requests, retained redaction, history, and trend counts.
The targeted STUDIO-API-AUTHOR-001 workflow checks the creation wizard, source persistence, help, focus, unsaved navigation, and narrow layout.
