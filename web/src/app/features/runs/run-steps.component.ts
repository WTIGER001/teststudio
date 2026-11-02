import { DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input } from '@angular/core'
import type { TestStep } from './run.models'

@Component({
  selector: 'ts-run-steps', standalone: true, imports: [DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (steps().length) {
      <details class="attempt-steps"><summary>{{ steps().length }} recorded steps</summary>
        <ol class="attempt-step-list" tabindex="0" aria-label="Recorded test steps">
          @for (step of steps(); track $index) {
            <li class="wrap-anywhere"><strong>{{ step.title }}</strong><span class="table-subtitle muted">{{ step.category }} · {{ step.durationMs | number:'1.0-1' }} ms</span>@if (step.error) { <pre class="step-error">{{ step.error }}</pre> }</li>
          }
        </ol>
      </details>
    }
  `
})
export class RunStepsComponent { readonly steps = input<TestStep[]>([]) }
