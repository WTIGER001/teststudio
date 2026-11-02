import { DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, input } from '@angular/core'
import type { TimingEstimate } from './timing.models'
import { predictionRows } from './timing.utils'

@Component({
  selector: 'ts-timing-estimate', standalone: true, imports: [DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (estimate(); as estimate) {
      <div class="notice" role="status"><strong>{{ estimate.readyTests }} / {{ estimate.totalTests }} timing baselines ready</strong>
        <p>@if (estimate.expectedExecutionMs != null) { Expected test execution: {{ estimate.expectedExecutionMs / 1000 | number:'1.1-1' }} s · high estimate: {{ (estimate.highExecutionMs ?? 0) / 1000 | number:'1.1-1' }} s. } @else { A complete duration estimate needs history for every selected test. }
        @if (estimate.provisional) { Estimates are provisional while baselines learn. }</p>
        <p class="small muted">Execution estimates exclude queue delay, preparation, evidence capture, and retries. High estimates sum historical 95th percentiles.</p>
        <p class="small">Attempt timeout budget, including permitted retries: {{ estimate.testTimeoutBudgetMs / 1000 | number:'1.0-1' }} s.</p>
        @if (estimate.exceedsRunLimit) { <p class="notice warning">The attempt timeout budget exceeds the overall run limit. Some tests or retries may not finish.</p> }
      </div>
      <div class="table-scroll"><table><caption class="sr-only">Timing predictions captured for the selected tests</caption><thead><tr><th scope="col">Test</th><th scope="col">History</th><th scope="col">Expected</th><th scope="col">Slow at</th><th scope="col">Attempt timeout</th><th scope="col">Rule</th></tr></thead><tbody>
        @for (row of rows(); track row.context.testId) { <tr><td><code>{{ row.context.testId }}</code></td><td>{{ row.sampleCount }} / {{ row.policy.minimumSamples }} minimum<span class="table-subtitle muted">{{ row.ready ? 'Ready' : 'Learning' }}</span></td><td>{{ row.expectedMs == null ? 'Unknown' : (row.expectedMs | number:'1.0-1') + ' ms' }}</td><td>{{ row.slowThresholdMs == null ? 'Learning' : (row.slowThresholdMs | number:'1.0-1') + ' ms' }}</td><td>{{ row.effectiveTimeoutMs / 1000 | number:'1.0-1' }} s<span class="table-subtitle muted">{{ row.policy.timeoutMode }}</span></td><td>{{ row.policy.mode }} · {{ row.policy.slowdownFactor }}×</td></tr> }
      </tbody></table></div>
    }
  `
})
export class TimingEstimateComponent {
  readonly estimate = input<TimingEstimate>()
  readonly rows = computed(() => predictionRows(this.estimate()))
}
