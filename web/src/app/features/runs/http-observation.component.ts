import { DatePipe, DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core'
import type { HTTPObservation } from './run.models'
import { certificateObservation, httpTimingRows } from './http-observation.utils'

@Component({
  selector: 'ts-http-observation', standalone: true, imports: [DatePipe, DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <span>HTTP {{ http().status || 'unavailable' }}</span><span class="table-subtitle wrap-anywhere">{{ http().url }}</span>
    <details class="http-observation"><summary>HTTP diagnostics</summary><dl>
      <dt>Probe duration</dt><dd>{{ http().durationMs | number:'1.0-2' }} ms</dd>
      @for (phase of phases(); track phase.label) { <dt>{{ phase.label }}</dt><dd>@if (phase.milliseconds != null) { {{ phase.milliseconds | number:'1.0-2' }} ms } @else { Unavailable }</dd> }
      <dt>Certificate expiry</dt><dd>@if (certificate().expiresAt; as expiry) { <time [attr.datetime]="expiry">{{ expiry | date:'medium':'UTC' }} UTC</time> } @else { Unavailable }</dd>
      <dt>Certificate lifetime at observation</dt><dd>@if (certificate().remainingDays != null) { {{ certificate().remainingDays | number:'1.0-2' }} days } @else { Unavailable }</dd>
      <dt>TLS trust</dt><dd>@if (http().trustBundle; as bundle) { {{ bundle.name }} <span class="table-subtitle">{{ bundle.id }}</span><span class="table-subtitle wrap-anywhere">SHA-256 {{ bundle.digest }}</span> } @else if (http().certificateExpires) { System certificate authorities } @else { Unavailable }</dd>
    </dl><p class="small muted">Timing phases can overlap. Unavailable phases are not zero measurements. Certificate lifetime uses this attempt's finish time.</p></details>
  `,
  styles: `
    :host { display: block; }
    .http-observation { min-width: 230px; max-width: 420px; margin-top: 10px; }
    summary { cursor: pointer; }
    dl { display: grid; grid-template-columns: minmax(100px, 1fr) minmax(0, 1fr); gap: 9px 15px; margin: 14px 0; font-size: 11px; }
    dt { color: var(--muted); }
    dd { margin: 0; overflow-wrap: anywhere; }
  `
})
export class HTTPObservationComponent {
  readonly http = input.required<HTTPObservation>()
  readonly observedAt = input.required<string>()
  readonly phases = computed(() => httpTimingRows(this.http()))
  readonly certificate = computed(() => certificateObservation(this.http(), this.observedAt()))
}
