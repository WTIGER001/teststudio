import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import { FormControl } from '@angular/forms'
import { ScheduleReviewComponent } from '../../shared/probes/schedule-review.component'
import { ProbeURLFieldComponent } from '../../shared/probes/probe-url-field.component'
import { ProbeReviewComponent } from '../../shared/probes/probe-review.component'
import { HTTPObservationComponent } from '../runs/http-observation.component'

export type HelpExample = 'probe-request' | 'probe-review' | 'probe-result' | 'probe-schedule' | 'probe-failure'
@Component({
  selector: 'ts-help-example', standalone: true, imports: [ScheduleReviewComponent, ProbeURLFieldComponent, ProbeReviewComponent, HTTPObservationComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<figure><figcaption>Example data · {{ captions[kind()] }}</figcaption>
    @switch (kind()) {
      @case ('probe-request') { <ts-probe-url-field [control]="address" id="help-example-url" [readonly]="true" /><p>GET · Accepted status codes: 200</p><p>Example choices: 10-second timeout, 14-day certificate lifetime. Review safety for your actual target.</p> }
      @case ('probe-review') { <ts-probe-review title="Example homepage" suite="Website availability" url="https://example.com/" /><p>Save makes no website request. Save and run also explicitly starts a run on this installation.</p> }
      @case ('probe-schedule') { <ts-schedule-review name="Homepage checks" suite="Website availability" target="Example production · Protected" cadence="Every 300 seconds · UTC" tests="homepage" options="This installation · 60 seconds · No retries · Skip overlaps" /><p>Review the owner and target. Enable this schedule only when you want repeated execution.</p> }
      @case ('probe-failure') { <p class="notice error">Failed · Unexpected HTTP status 503</p><ts-http-observation [http]="failure" observedAt="2026-01-01T12:00:00Z" /><p>This is a retained endpoint failure. A timeout, stale observation, or missing evidence has its own state.</p> }
      @case ('probe-result') { <ts-http-observation [http]="observation" observedAt="2026-01-01T12:00:00Z" /><p>Passed · Observed 1 January 2026, 12:00 UTC. A public entry-page check does not test signing in.</p> }
    }
  </figure>`,
  styles: `figure { margin: 1rem 0; padding: 1rem; border: 1px solid var(--border); border-radius: 8px; background: var(--surface-muted); min-width: 0; } figcaption { font-size: 12px; font-weight: 650; margin-bottom: 1rem; } p { font-size: 13px; line-height: 1.7; }`
})
export class HelpExampleComponent {
  readonly captions = { 'probe-request': 'Request', 'probe-review': 'Review', 'probe-result': 'Result', 'probe-schedule': 'Schedule review', 'probe-failure': 'Failed result' }
  readonly kind = input.required<HelpExample>()
  readonly address = new FormControl('https://example.com/', { nonNullable: true })
  readonly failure = { url: 'https://example.com/', status: 503, durationMs: 120 }
  readonly observation = { url: 'https://example.com/', status: 200, durationMs: 120, timingsMs: { dns: 5, connect: 12, tls: 30, firstByte: 105 }, certificateExpires: '2026-04-01T00:00:00Z' }
}
