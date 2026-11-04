# Private Playwright trace viewer

## Inspect a trace

Open Test runs and select a run. Select Inspect test and an attempt to narrow the evidence.
Select View on an available trace. The preview contains the installation-hosted Playwright viewer.
Select an action to inspect its recorded page snapshot, network activity, and available source.
The viewer uses recorded evidence. It does not open the original website or repeat the test.
Select Refresh trace access when access expires. Current permissions determine whether a new view can open.
Close preview returns keyboard focus to the original View button.

Download trace requests a separate download grant. It requires the current evidence download permission.
Use a local Playwright viewer when inline viewing is unavailable or the trace format is unsupported.
Do not upload private traces to a public service.

Inline viewing requires HTTPS or localhost and browser service-worker support.
The current installation supports the pinned Playwright 1.64.1 viewer. Newer trace formats can require an installation update.
Invalid archives retain their authorized download fallback. The viewer accepts archives up to 64 MiB, 10,000 entries, and 256 MiB expanded.

## Install the viewer

Run `npm --prefix web ci` and `npm --prefix web run build`.
The postbuild task creates `web/dist/trace-viewer` beside the main UI output.
The bundle includes the pinned viewer, its license notices, a session controller, and an isolated service-worker wrapper.
The generated manifest records the pinned version and each asset checksum.

Pass `--trace-viewer-directory /absolute/path/to/web/dist/trace-viewer` to the Go server.
Keep this directory outside the main UI directory. Serve it only through the separate artifact listener.
The API verifies the manifest and loads immutable asset bytes at startup. Invalid configured bundles stop startup.
An omitted directory disables inline viewing and preserves authorized downloads.
The Go server requires no Node process to serve the viewer.
The local demo launcher and self-test image include the bundle automatically.
Hosted container packaging remains separate deployment work.

## Access and isolation

The normal project artifact access endpoint issues a session-bound grant for one artifact and action.
View grants expire within one minute and can expire sooner with the session or current authorization.
A trace response adds `traceViewerUrl` when inline viewing is available.
Otherwise, `traceViewerUnavailable` explains the fallback. Existing `url` and `expiresAt` fields remain available.
A download grant cannot become a viewer grant.

Viewer assets and trace content use the separate artifact origin. They do not execute on the main application origin.
Every server read rechecks the session, current project permission, artifact identity, availability, and grant expiry.
The service worker also checks current authorization before returning recorded resources from its in-memory trace context.
Each grant has its own worker scope. Captured resource responses cannot execute scripts.
Recorded resource URLs remain intact for snapshot rendering. The worker serves their archived bytes and blocks external network fetches.
The viewer does not contact the recorded website or an external trace service.

The session controller closes expired views. It checks revoked access every five seconds while the viewer is open.
A rejected resource read also closes the viewer. Refresh trace access requests a new grant through the application API.
Evidence already delivered to a browser cannot be recalled. These checks prevent further reads after access is lost.

## Reverse proxy configuration

Use HTTPS for both hosted origins. Preserve JavaScript MIME types and the service-worker scope below each grant path.
Do not cache viewer routes, authorization responses, or artifact content. Preserve the server security and no-store headers.
Redact `/trace/{token}` and `/access/{token}` paths and query values from proxy logs and telemetry.
Grant URLs are temporary bearer credentials. Do not publish them or include them in support tickets.
The artifact origin must remain separate from the main application origin.
This milestone has local verification only. It does not establish AWS, Kubernetes, or Azure deployment acceptance.
