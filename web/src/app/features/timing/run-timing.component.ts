import { DatePipe, DecimalPipe } from '@angular/common'
import { afterRenderEffect, ChangeDetectionStrategy, Component, computed, ElementRef, inject, viewChild } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { TimingEstimateComponent } from './timing-estimate.component'
import { RunTimingService } from './run-timing.service'
import { predictionRows } from './timing.utils'

@Component({
  selector: 'ts-run-timing', standalone: true, imports: [DatePipe, DecimalPipe, ReactiveFormsModule, TimingEstimateComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.detail.run(); as run) {
      <section class="panel" aria-labelledby="run-timing-title"><h2 id="run-timing-title">Test timing</h2>
        @if (service.detail.timingProgress(); as progress) {
          @if (progress.elapsedMs != null) { <p>Worker elapsed: {{ progress.elapsedMs / 1000 | number:'1.1-1' }} s.@if (progress.remainingMs != null) { Estimated remaining test execution: {{ progress.remainingMs / 1000 | number:'1.1-1' }} s. } @else if (progress.overdue) { The current attempt is taking longer than its typical duration. } @else if (!run.finishedAt) { A remaining duration estimate is unavailable. }</p> }
        }
        <p class="small muted">Predictions and thresholds below were captured when this run was queued. Elapsed worker time also includes coordination and capture.</p>
        @if (run.timing?.totalTests) { <ts-timing-estimate [estimate]="run.timing" /> } @else { <p class="small muted">This historical run has no captured timing baseline. Attempt durations remain available below.</p> }
        @if (service.detail.workspace.can('runs.read') && rows().length) {
          <details class="timing-baselines"><summary>Manage comparable timing history</summary><p class="small muted">Current history can differ from this run's snapshot. Reset starts learning again and preserves previous results.</p>
            <div class="row-actions wrap-actions">@for (prediction of rows(); track prediction.context.testId) { <button class="button secondary small" type="button" [disabled]="service.loading() || service.saving()" (click)="service.open(prediction)">{{ prediction.context.testId }} baseline</button> }</div>
          </details>
        }
        @if (service.selected(); as prediction) {
          <section class="timing-baseline" aria-labelledby="baseline-title"><div class="section-top"><h3 #heading tabindex="-1" id="baseline-title">{{ prediction.context.testId }} · current baseline</h3><button class="button subtle" type="button" [disabled]="service.saving()" (click)="service.close()">Close baseline</button></div>
            @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
            @if (service.success()) { <p class="notice success" role="status">{{ service.success() }}</p> }
            @if (service.loading()) { <p role="status">Loading baseline…</p> }
            @if (service.baseline(); as baseline) {
              <p>{{ baseline.data.samples.length }} stored clean observations · generation {{ baseline.data.generation }} · revision {{ baseline.revision }}</p>
              <p class="small muted">{{ baseline.data.context.environmentId || 'No target' }} · {{ baseline.data.context.executionLocation }} · {{ baseline.data.context.profile }}. Policy windows can use fewer observations.</p>
              @if (baseline.data.resetAt) { <p class="small">Last reset {{ baseline.data.resetAt | date:'medium' }}: {{ baseline.data.resetReason }}</p> }
              @if (service.detail.workspace.can('catalog.publish')) {
                <form [formGroup]="service.form" (ngSubmit)="service.reset()"><div class="field"><label for="baseline-reset-reason">Reason to reset timing history</label><textarea id="baseline-reset-reason" formControlName="reason" rows="2" maxlength="500" required></textarea><small>Use after an intentional performance change. Runs already queued cannot repopulate this new generation.</small></div><div class="row-actions"><button class="button secondary" type="button" [disabled]="service.loading() || service.saving()" (click)="service.open(prediction)">Reload baseline</button><button class="button danger" type="submit" [disabled]="service.reason.invalid || service.saving()">{{ service.saving() ? 'Resetting…' : 'Reset timing history' }}</button></div></form>
              }
            }
          </section>
        }
      </section>
    }
  `
})
export class RunTimingComponent {
  readonly service = inject(RunTimingService)
  readonly rows = computed(() => predictionRows(this.service.detail.run()?.timing))
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { if (this.service.selected()) this.heading()?.nativeElement.focus() }) }
}
