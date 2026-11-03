# Native HTTP checks

Native checks run in the Go service without a browser worker or a test image.
Select Create test from the catalog, then choose HTTPS probe on the dedicated creation page.
Edit existing checks from the catalog. Validate and publish the suite release after saving.
Each run retains its selected definition and target snapshot.
Later edits and publications do not change existing runs.

## Request settings

Choose a registered environment endpoint and a relative path beginning with one slash.
The path can include a query string. It cannot select another origin or include a fragment.
Plain HTTP requires an explicitly permitted environment, including the local stage.

Choose GET or HEAD. The default is GET with accepted status 200.
Configure 1 through 32 accepted status codes from 100 through 599.
Redirects remain disabled. An unexpected redirect status is a normal HTTP status failure.
HEAD never falls back to GET.
Keep a separate safety label and rationale for every check.

## Response assertions

All configured assertions must pass:

- Required text must occur in the response body.
- Forbidden text must not occur in the response body.
- Expected headers compare exact string values.
- Expected JSON fields compare exact JSON values at dot-separated paths.

Use numeric path segments for array indexes, such as `items.0.id`.
JSON expectations support strings, numbers, booleans, null, arrays, and objects.
The editor accepts JSON data. It does not evaluate expressions.
Expected headers describe the response. They do not configure authentication credentials.

HEAD supports status and header assertions. It cannot assert a response body.
The editor preserves conflicting body fields and blocks saving until the user clears them or chooses GET.
It does not silently discard assertions when the method changes.

Each text assertion has an 8192-byte limit.
Configure at most 32 expected headers with unique case-insensitive names.
Header names have a 256-byte limit. Values must be single-line strings within 8192 bytes.
Configure at most 32 JSON paths, with a 256-byte limit per path and 65536 bytes of combined JSON expectations.
Body assertions read at most one MiB of response content.

## Response time and TLS

Maximum response time accepts whole milliseconds from zero through 3600000.
Zero disables this assertion. An exceeded response limit is a response-time failure.
Attempt timeouts remain separate and use the timed-out status.
Historical slowdown rules also remain separate from this fixed response-time assertion.

Minimum certificate lifetime accepts whole days from zero through 36500.
Zero disables the additional lifetime threshold.
TLS trust, hostname, and certificate dates still require validation.
A positive lifetime threshold requires TLS certificate information from the endpoint.

Publication rejects invalid assertion settings and limits before activation.
An invalid new draft does not replace the active release.
The executor uses the same validation when it receives a frozen definition.

## Definition contract

The test definition uses runner `http`, type `https`, and a `check` object:

```json
{
  "endpoint": "api",
  "path": "/health",
  "method": "GET",
  "statusCodes": [200, 204],
  "requiredText": "up",
  "forbiddenText": "down",
  "headers": {"Content-Type": "application/json"},
  "json": {"status": "up", "items.0.id": 7},
  "maxResponseMs": 1000,
  "minCertificateDays": 14
}
```

Use the normal catalog API to save, validate, and publish these definitions.
The catalog type filter separates HTTPS checks from API, browser, and other test types.
Metadata-only edits preserve assertion settings.

## Retained diagnostics

Expand HTTP diagnostics in attempt history to inspect probe duration and available network phases.
The view includes DNS lookup, connection, TLS handshake, and time to first byte.
Unavailable phases stay unavailable. A measured zero remains a valid measurement.
Phases can overlap and must not be added together as a second execution duration.

Certificate expiry uses the retained certificate date.
Displayed certificate lifetime uses the attempt finish time, so historical views do not change as time passes.
Missing certificate information remains unavailable.
The interface reads normal result records and needs no HTML report.

## Local verification

```sh
bash scripts/test-self.sh --test=STUDIO-HTTPS-001
```

This workflow creates and publishes assertions through the UI, runs GET and HEAD, and verifies frozen history and retained diagnostics.
Focused Go checks verify publication limits, HEAD behavior, response-time failures, certificate trust, and certificate lifetime thresholds.

Redirect following, secret references, project trust-bundle configuration, response excerpts, schedules, and response-time history remain pending.
