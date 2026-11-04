import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import { TimingResultComponent } from '../timing/timing-result.component'
import { GoObservationComponent } from './go-observation.component'
import { HTTPObservationComponent } from './http-observation.component'
import { RunStepsComponent } from './run-steps.component'
import type { Attempt } from './run.models'

@Component({
  selector: 'ts-run-attempt', standalone: true, imports: [DatePipe, TimingResultComponent, GoObservationComponent, HTTPObservationComponent, RunStepsComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <article class="attempt-card" [attr.aria-label]="'Attempt ' + attempt().attempt + ' for ' + attempt().testId">
      <div class="section-top"><h3>Attempt {{ attempt().attempt }} <code class="small muted">{{ attempt().testId }}</code></h3><span class="badge" [class.timeout]="attempt().state === 'timed-out'" [class.success]="attempt().state === 'passed'" [class.warning]="attempt().state !== 'passed'">{{ attempt().state }}</span></div>
      <p class="small muted">Started {{ attempt().startedAt | date:'medium' }} · <ts-timing-result [result]="attempt()" /></p>
      @if (attempt().failureKind) { <p class="small">Reason: {{ attempt().failureKind }}</p> }
      @if (attempt().recovered) { <p class="notice">Recovered observation. The original run outcome stays unchanged.</p> }
      @if (attempt().observedState) { <p class="small muted">Functional result: {{ attempt().observedState }}</p> }
      @if (attempt().error) { <details open><summary>Failure details</summary><pre class="step-error">{{ attempt().error }}</pre></details> }
      @if (attempt().http; as http) { <ts-http-observation [http]="http" [observedAt]="attempt().finishedAt" /> }
      @if (attempt().go; as go) { <ts-go-observation [go]="go" /> }
      <ts-run-steps [steps]="attempt().steps ?? []" [revealFailures]="true" />
    </article>
  `,
  styles: [`.attempt-card { min-width: 0; border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin: 12px 0; } h3 { margin: 0; font-size: 14px; } h3 code { display: block; margin-top: 4px; } .section-top { flex-wrap: wrap; gap: 10px; } details { margin: 12px 0; }`]
})
export class RunAttemptComponent { readonly attempt = input.required<Attempt>() }
