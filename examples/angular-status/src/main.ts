import { provideHttpClient } from '@angular/common/http'
import { bootstrapApplication } from '@angular/platform-browser'
import { Component, signal } from '@angular/core'
import { TestStudioStatusBarComponent, TestStudioStatusPageComponent, type StatusReport, type StatusSource } from '@wtiger001/teststudio-angular'

const counts = { passed: 10, failed: 1, timedOut: 1, flaky: 1, skipped: 1, unknown: 1 }
const availability = { percent: null, coveragePercent: 0, upSeconds: 0, downSeconds: 0, unknownSeconds: 3600, since: '2026-11-07T00:00:00Z' }
const totals = { counts, upSeconds: 0, downSeconds: 0, unknownSeconds: 0, durationSumMs: 1200, durationSampleCount: 12, censoredCount: 1, durationUnavailableCount: 2 }
const buckets = ['passed', 'failed', 'timed-out', 'flaky', 'skipped', 'unknown'].map((state, i) => {
  const key = state === 'timed-out' ? 'timedOut' : state
  const stateCounts = { passed: 0, failed: 0, timedOut: 0, flaky: 0, skipped: 0, unknown: 0, [key]: state === 'passed' ? 10 : 1 }
  const samples = state === 'passed' ? 10 : ['failed', 'flaky'].includes(state) ? 1 : 0
  const stateTotals = { ...totals, counts: stateCounts, durationSumMs: samples * 100, durationSampleCount: samples, censoredCount: state === 'timed-out' ? 1 : 0, durationUnavailableCount: ['skipped', 'unknown'].includes(state) ? 1 : 0 }
  return { since: `2026-11-07T0${i}:00:00Z`, until: `2026-11-07T0${i + 1}:00:00Z`, state, partial: false, totals: stateTotals, availability: null, passRate: state === 'passed' ? 100 : ['failed', 'timed-out', 'flaky'].includes(state) ? 0 : null }
})
const report: StatusReport = { title: 'Release checks', background: '', window: '24h', maxAgeSeconds: 0, asOf: '2026-11-07T12:00:00Z', probes: [{ title: 'Checkout browser test', description: 'Host-supplied outcome history', state: 'passed', metric: 'outcomes', history: [], totals, passRate: 1000 / 13, contextCount: 1, availability, buckets }] }


@Component({
  selector: 'status-example', standalone: true, imports: [TestStudioStatusBarComponent, TestStudioStatusPageComponent],
  template: `<main><h1>Independent Angular host</h1><div class="controls"><label>Component<select aria-label="Component" [value]="mode()" (change)="mode.set($any($event.target).value)"><option value="page">Host data · full page</option><option value="bar">Host data · status bar</option><option value="remote">Public API · full page</option></select></label><label>Appearance<select aria-label="Appearance" [value]="theme()" (change)="theme.set($any($event.target).value)"><option value="light">Light</option><option value="dark">Dark</option><option value="brand">Host brand</option></select></label><button type="button" (click)="visible.set(!visible())">{{ visible() ? 'Remove components' : 'Show components' }}</button></div>
    <div class="host" [class.dark]="theme() === 'dark'" [class.brand]="theme() === 'brand'">
      @if (visible()) {
        @switch (mode()) {
          @case ('bar') { <teststudio-status-bar [row]="report.probes[0]" (bucketSelected)="selected.set($event.bucket.state)" (drilldownRequested)="selected.set('Drill-down: ' + $event.bucket.since)" /> }
          @case ('remote') { <teststudio-status-page [source]="source" (bucketSelected)="selected.set($event.bucket.state)" /> }
          @default { <teststudio-status-page [report]="report" (windowChanged)="selected.set('Requested period: ' + $event)" /> }
        }
      }
    </div><p role="status">{{ selected() }}</p></main>`
})
class StatusExample {
  readonly report = report
  readonly mode = signal(new URLSearchParams(location.search).has('page') ? 'remote' : 'page')
  readonly theme = signal('light')
  readonly visible = signal(true)
  readonly selected = signal('')
  readonly source: StatusSource = { url: 'http://127.0.0.1:8040/api/v1/public/status-pages/' + encodeURIComponent(new URLSearchParams(location.search).get('page') || 'unpublished'), refreshSeconds: 15 }
}

bootstrapApplication(StatusExample, { providers: [provideHttpClient()] }).catch(error => { throw error })
