import { DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import type { GoObservation } from './run.models'

@Component({
  selector: 'ts-go-observation', standalone: true, imports: [DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <span class="wrap-anywhere">{{ go().package }} · {{ go().test }}</span>
    <details><summary>Go diagnostics</summary><dl>
      <dt>Toolchain</dt><dd>{{ go().toolchain || 'Unavailable' }}</dd>
      <dt>Platform</dt><dd>{{ go().os || 'Unavailable' }} / {{ go().architecture || 'Unavailable' }}</dd>
      <dt>Build duration</dt><dd>@if (go().buildExitCode != null) { {{ go().buildDurationMs | number:'1.0-2' }} ms } @else { Unavailable }</dd>
      <dt>Build exit code</dt><dd>{{ go().buildExitCode ?? 'Not started' }}</dd>
      <dt>Test process exit code</dt><dd>{{ go().exitCode ?? 'Not started' }}</dd>
    </dl><p class="small muted">Build preparation stays separate from test duration. Registered subtests produce one logical result each.</p>
    @if (go().outputTruncated) { <p class="small">Raw output reached its capture limit. The event parser continued reading.</p> }
    </details>
  `,
  styles: `
    :host { display: block; min-width: 230px; max-width: 420px; }
    details { margin-top: 10px; }
    summary { cursor: pointer; }
    dl { display: grid; grid-template-columns: minmax(100px, 1fr) minmax(0, 1fr); gap: 9px 15px; margin: 14px 0; font-size: 11px; }
    dt { color: var(--muted); }
    dd { margin: 0; overflow-wrap: anywhere; }
  `
})
export class GoObservationComponent { readonly go = input.required<GoObservation>() }
