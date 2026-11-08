# TestStudio Angular status components

Version 0.1.0-dev provides standalone, OnPush Angular status components.
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
npm install ./wtiger001-teststudio-angular-0.1.0-dev.tgz
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
