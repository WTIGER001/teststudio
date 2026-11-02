# Interface design and verification

The Angular application uses standalone components, OnPush change detection, and signal-based services.
Components delegate workflows to focused feature services.
Pure utilities handle catalog filtering and counts.

## Themes

Light, dark, and system preferences use shared semantic CSS tokens.
The browser stores the selected preference under `teststudio.theme`.
An external initialization script applies the theme before the first render.
System mode follows operating-system changes without a page reload.
The build disables inline critical-style scripts to preserve the server content policy.

## HTML and access

The shell uses header, navigation, main, and complementary landmarks.
A keyboard skip link moves focus to the main content.
Forms use associated native labels, required fields, and compatible identifier patterns.
Tables use captions and scoped column headings.
Status labels include text. Color is not the only signal.
Visible focus and reduced-motion preferences apply in both themes.
Narrow layouts wrap navigation and controls. Wide tables scroll inside their own container.

## Verified local workflows

A browser check creates a new generic project, application, suite, environment, and HTTPS definition.
It validates and publishes a catalog release, then archives and restores the test.
It verifies theme persistence, operating-system theme changes, visible keyboard focus, and a 390-pixel viewport.
The final page has no console errors.
The test browser and its automation process exit after verification.
Run history provides environment, suite, state, and selection filters with bounded pagination.
The run form selects a published suite, environment, test identifiers, time limit, and retries.
The time limit accepts 1 through 3600 seconds, consistent with the run API.
Run details show the execution snapshot, progress, final counts, attempts, events, and filtered logs.
Active runs refresh every two seconds. Cancellation and failure reruns use explicit actions.
Artifact previews support text, raster images, and video. Other content uses a download link.
The browser retrieves artifact content through short-lived access links on a separate origin.
Local browser checks verify successful runs, failure reruns, cancellation, text previews, and file downloads.
The run pages support light and dark themes and a 390-pixel viewport without document overflow.
