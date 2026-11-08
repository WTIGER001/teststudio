# TestStudio Angular status components

Version 0.3.0-dev provides standalone, OnPush Angular status components.
Angular 22.2.4 is the verified peer version. Other Angular versions remain unverified.
The library uses Angular core, common, RxJS, and tslib. It requires no TestStudio application shell, router, global stylesheet, or iframe.

## Build and install

From the TestStudio repository:

```sh
node scripts/build-status-library.mjs --consumer --pack
```

This builds the library, installs it into the independent example, checks compilation and server rendering, and creates a package under output/angular-status.
Install that package in another Angular application:

```sh
npm install ./wtiger001-teststudio-angular-0.3.0-dev.tgz
```

Package creation does not publish to an npm registry or deploy a site.

## Render host-supplied data

```ts
import { Component, input } from '@angular/core'
import {
  TestStudioStatusBarComponent,
  TestStudioStatusPageComponent,
  type StatusReport
} from '@wtiger001/teststudio-angular'

@Component({
  selector: 'app-service-status',
  standalone: true,
  imports: [TestStudioStatusBarComponent, TestStudioStatusPageComponent],
  template: `
    <teststudio-status-bar [row]="report().probes[0]" />
    <teststudio-status-page [report]="report()" />
  `
})
export class ServiceStatusComponent {
  readonly report = input.required<StatusReport>()
}
```

Supply at least one row before rendering that bar. A page accepts an empty row list.
The report's probes array can contain any test type. Its name preserves the public report contract.
Use the host's normal authenticated service for private data. The library does not establish identity or store service credentials.

## Read a published page

Provide Angular HttpClient in the host application. Then use the optional source input:

```ts
readonly source = {
  url: 'https://teststudio.example.test/api/v1/public/status-pages/customer-health',
  refreshSeconds: 60
}
```

```html
<teststudio-status-page [source]="source" />
```

Sources accept public status-report endpoints only. They send no browser cookies.
The public client bypasses host authentication interceptors and explicitly omits Fetch credentials.
The server permits cross-origin reads of that public projection. Private endpoints retain their normal restrictions.
The public URL must already be published. Installing this library does not publish checks.
Withdrawal or access rejection clears the displayed report on the next refresh.
Transient errors also clear the report and show an unavailable message.
Refresh defaults to 60 seconds. Positive intervals clamp to 15–3600 seconds. Zero loads once.
Identical active subscriptions share requests within the client instance. Destruction cancels subscriptions, requests, and polling.
Rendering never executes a check or starts sign-in.

TestStudioStatusClient.watch exposes the same report stream for host services that feed a compact bar.
TestStudioStatusClient.publicURL constructs the public endpoint from an installation URL and page name or token.
Render-only components need no HttpClient provider. Public sources make no requests during server rendering.

## Style the components

```css
teststudio-status-page,
teststudio-status-bar {
  --ts-status-font: Georgia, serif;
  --ts-status-passed: #7544c7;
  --ts-status-pill-width: 12px;
  --ts-status-pill-height: 32px;
  --ts-status-pill-radius: 4px;
  --ts-status-gap: 6px;
}
```

Supported status tokens are passed, failed, timeout, flaky, and unknown under the ts-status prefix.
Use --ts-status-foreground and --ts-status-background for surfaces and text.
The defaults inherit the host font and foreground, so light, dark, and system themes can supply their own values.
Other tokens control padding, top padding, minimum height, radius, heading size, title size, description size, row spacing, summary width, option colors, and tooltip colors. Their names appear in the component styles under the same prefix.
Optional logo and logoAlt report fields preserve uploaded images or public HTTPS references. The page centers the logo above its title, preserves proportions and transparency, and shows a readable fallback after an image load failure. Optional logoHeight sets a height from 1 through 1024 CSS pixels. Zero or omission keeps automatic sizing through --ts-status-logo-width and --ts-status-logo-height. A positive height takes precedence over those automatic bounds, preserves proportions, and shrinks to fit its container.
The page uses host styling by default. Set useReportBackground to true to honor the published background and choose contrasting text.
Scoped styles require no private CSS selectors or deep overrides.
Title, description, and selected-detail templates are available through titleTemplate, descriptionTemplate, and detailTemplate inputs on both components.

## Interaction and accessibility

Both components emit bucketSelected and drilldownRequested with the selected row and bucket.
The page emits windowChanged. Hosts own routing and private run navigation.
Public-source pages fetch the requested period automatically. Host-data pages expect the host to fetch a replacement report after windowChanged.
The page emits loadingChanged and loadFailed for source-driven views.
Use locale and timezone inputs for presentation. Bucket boundaries retain the server's original timestamps.
Use showSummaries on the page or showHeading on the bar to fit the host layout.
Native controls, keyboard focus, accessible bucket details, and touch targets remain available with custom styling.
Unknown monitoring coverage remains visible alongside downtime. Historical bucket state remains separate from the current result.

The independent example lives in examples/angular-status. STUDIO-RELIABILITY-001 verifies it through normal TestStudio execution.

## Individual checks without aggregation

Source-driven public pages include Individual checks on every row. Select a pill and Inspect period for the individual results inside it, or use Individual checks for the whole report period. Return to full period clears the pill filter. Load older checks fetches the next page. The table preserves completion time, outcome, duration, and censoring. It never groups executions or extrapolates uptime.

Host-data bars and pages accept a checkLoader input. It receives since, until, row, limit, and an optional opaque cursor, and returns Promise<StatusCheckPage>. Use your normal authenticated service for private requests. For example:

```ts
readonly loadChecks: StatusCheckLoader = query =>
  this.history.readChecks(this.testId, query)
```

```html
<teststudio-status-page [report]="report()" [checkLoader]="loadChecks"
  [inspectableChecks]="true" (checkRequested)="openRun($event.runId)" />
```

The host owns openRun and rechecks private run access on navigation. Public requests never return runId or evidence. A standalone bar can use checksSource with the public report URL and rowIndex for its published row. A host-data page can also supply checksSource while retaining host-managed report loading. Both modes bypass host authentication interceptors for public reads.
The exported teststudio-status-checks component supports a standalone table with range, source or loader, row, locale, and timezone inputs. Its checkRequested and resetRange outputs leave navigation to the host. Table requests stop on component destruction; late host-promise results are ignored. Server rendering starts no detail requests.
The API returns at most 100 results per page and bounds filtered scans. Follow cursor even when a page has no readable matches. Cursors expire after 15 minutes; retry from the report if one expires. Compact observations have no automatic expiry, but summaries can outlive removed detail. The table reports missing fine detail without inventing rows. Three-hour and six-hour summary sizes do not restrict individual history.

The page centers its contents inside a maximum width of 72rem while retaining its configured full-width background. Set --ts-status-content-width on the component to override that bound for a host layout. Pill widths stay controlled by --ts-status-pill-width; a wide container does not stretch them.
