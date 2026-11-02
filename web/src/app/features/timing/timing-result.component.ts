import { DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core'
import { measuredAttemptDuration } from './timing.utils'
import type { Attempt } from '../runs/run.models'

@Component({
  selector: 'ts-timing-result', standalone: true, imports: [DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (result(); as result) {
      @if (duration() != null) { <span>{{ duration() | number:'1.0-1' }} ms</span> } @else { <span>Duration unavailable</span> }
      @if (result.workerDurationMs != null) { <span class="table-subtitle muted">Worker elapsed {{ result.workerDurationMs | number:'1.0-1' }} ms</span> }
      @if (result.timing; as timing) {
        <span class="table-subtitle" [class.warning]="timing.status === 'slow'">{{ timing.status }}@if (timing.ratio != null) { · {{ timing.ratio | number:'1.1-2' }}× expected }</span>
        <span class="table-subtitle muted">@if (timing.baseline.expectedMs != null) { Expected {{ timing.baseline.expectedMs | number:'1.0-1' }} ms · }{{ timing.baseline.sampleCount }} observations</span>
        <span class="table-subtitle muted">Timeout {{ timing.timeoutMs / 1000 | number:'1.0-1' }} s@if (timing.censored) { · interrupted duration }</span>
      }
      @if (result.attempt > 1 && result.totalDurationMs != null) { <span class="table-subtitle">All attempts: {{ result.totalDurationMs | number:'1.0-1' }} ms</span> }
      @if (result.timingRegressions) { <span class="table-subtitle">{{ result.timingRegressions }} slow {{ result.timingRegressions === 1 ? 'attempt' : 'attempts' }}</span> }
    }
  `
})
export class TimingResultComponent { readonly result = input<Attempt>(); readonly duration = computed(() => measuredAttemptDuration(this.result())) }
