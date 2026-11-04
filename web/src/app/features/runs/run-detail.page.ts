import { RunWorkersComponent } from './run-workers.component'
import { NotificationDeliveriesComponent } from '../notifications/notification-deliveries.component'
import { RunWorkersService } from './run-workers.service'
import { RunTimingComponent } from '../timing/run-timing.component'
import { RunTimingService } from '../timing/run-timing.service'
import { TimingResultComponent } from '../timing/timing-result.component'
import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, afterNextRender, inject, Injector, viewChild } from '@angular/core'
import { RouterLink } from '@angular/router'
import type { Attempt } from './run.models'
import { RunDetailService } from './run-detail.service'
import { RunEvidenceService } from './run-evidence.service'
import { RunEvidenceComponent } from './run-evidence.component'
import { RunArtifactsComponent } from './run-artifacts.component'
import { RunRetentionComponent } from './run-retention.component'
import { RunRetentionService } from './run-retention.service'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, RunEvidenceComponent, RunArtifactsComponent, RunRetentionComponent, RunTimingComponent, TimingResultComponent, RunWorkersComponent, NotificationDeliveriesComponent], providers: [RunDetailService, RunWorkersService, RunTimingService, RunEvidenceService, RunRetentionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.shared()) { <p class="notice">Shared run: these details show the complete saved run across applications. Application lists and trends count only matching tests.</p> }
    <a class="small" [routerLink]="service.workspace.path('runs')">← Run history</a>
    <div class="page-heading run-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Run details</h1><code class="small muted wrap-anywhere">{{ service.runId() }}</code></div><div class="row-actions">
      @if (service.canCancel()) { <button type="button" class="button secondary" [disabled]="service.mutating()" (click)="service.cancel()">Cancel run</button> }
      @if (service.canRerun()) { <button type="button" class="button primary" [disabled]="service.mutating()" title="Rerun failed, timed-out, and flaky tests" (click)="service.rerunFailures()">Rerun failures</button> }
      <button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.load()">Refresh</button>
    </div></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.actionError()) { <div class="notice error" role="alert"><p>{{ service.actionError() }}</p><button type="button" class="button secondary" [disabled]="service.refreshingAccess() || service.mutating()" (click)="service.refreshAccess()">Refresh access</button></div> }
    @if (service.run(); as run) {
      <section class="metrics" aria-label="Run summary">
        <article class="metric"><span class="eyebrow">State</span><strong class="metric-status">{{ run.state }}</strong><span class="small muted">{{ run.phase }}{{ run.cancelRequested ? ' · Cancellation requested' : '' }}</span></article>
        <article class="metric"><span class="eyebrow">Outcome</span><strong class="metric-status">{{ run.outcome }}</strong><span class="small muted">{{ run.counts.completed }} / {{ run.counts.total }} tests complete</span></article>
        <article class="metric"><span class="eyebrow">Results</span><strong>{{ run.counts.passed }}<small class="small muted"> passed</small></strong><span class="small muted">{{ run.counts.failed }} failed · {{ run.counts.timedOut }} timed out · {{ run.counts.flaky }} flaky · {{ run.counts.skipped }} skipped</span></article>
        <article class="metric"><span class="eyebrow">Capture</span><strong class="metric-status">{{ run.evidenceState }}</strong><span class="small muted">File retention is tracked separately.</span></article>
      </section>
      @if (!service.finished()) { <div class="notice" role="status"><label for="run-progress">{{ run.currentTestId ? (run.phase === 'preparation' ? 'Preparing ' : 'Running ') + run.currentTestId : 'Waiting for worker progress' }}</label><progress id="run-progress" [value]="run.counts.completed" [max]="run.counts.total || 1">{{ run.counts.completed }} / {{ run.counts.total }}</progress><span class="small muted">Updates every two seconds.</span></div> }
      @if (run.error) { <p class="notice error" role="alert">{{ run.error }}</p> }
      @for (error of run.evidenceErrors; track $index) { <p class="notice warning">Evidence: {{ error }}</p> }
      <section class="panel" aria-labelledby="snapshot-title">
        <h2 id="snapshot-title">Execution snapshot</h2><dl class="detail-grid">
          <dt>Suite</dt><dd>{{ run.request.suiteId }} · release <code>{{ run.release.id }}</code></dd><dt>Environment</dt><dd>{{ run.environment?.name || 'No target' }}{{ run.environment?.protected ? ' · Protected' : '' }}<span class="small muted"> {{ run.environment?.stage }}</span></dd>
          @if (run.request.parentRunId) { <dt>Rerun of</dt><dd><a [routerLink]="service.workspace.path('runs', run.request.parentRunId!)">Original run</a><span class="table-subtitle muted">The original results and evidence remain available.</span></dd> }
          <dt>Selection</dt><dd>{{ run.request.testIds?.length ? 'Selected tests and dependencies' : 'Full eligible suite' }} · {{ run.tests.length }} tests</dd><dt>Location</dt><dd>{{ run.request.executionLocation }}</dd>
          @if (run.savedSelection; as saved) { <dt>Saved selection</dt><dd>{{ saved.definition.name }} · revision {{ saved.revision }}<code class="table-subtitle">{{ saved.definition.id }} · {{ saved.definition.spec.kind }} · schema {{ saved.definition.spec.schemaVersion }}</code></dd> }
          <dt>Requested by</dt><dd>{{ run.actor }} · {{ run.trigger }}</dd><dt>Created</dt><dd><time [attr.datetime]="run.createdAt">{{ run.createdAt | date:'medium' }}</time></dd>
          @if (run.schedule; as schedule) { <dt>Schedule</dt><dd><a [routerLink]="['/projects', run.projectId, 'schedules']">{{ schedule.id }}</a><span class="table-subtitle muted">Due <time [attr.datetime]="schedule.dueAt">{{ schedule.dueAt | date:'medium':'UTC' }} UTC</time></span></dd> }
          <dt>Finished</dt><dd>{{ run.finishedAt ? (run.finishedAt | date:'medium') : 'In progress' }}</dd><dt>Limits</dt><dd>{{ run.request.timeoutSeconds }} seconds · {{ run.request.retries ?? 0 }} retries per test</dd>
        </dl>
      </section>
      <ts-run-timing />
      <section class="panel" aria-labelledby="results-title"><h2 id="results-title">Final test results</h2>
        <div class="table-scroll"><table><caption class="sr-only">One final result for each completed test</caption><thead><tr><th scope="col">Test</th><th scope="col">State</th><th scope="col">Last attempt</th><th scope="col">Duration</th><th scope="col">Details</th></tr></thead><tbody>
          @for (result of run.results; track result.testId) { <tr><td><strong>@if (service.workspace.can('catalog.read')) { <a [routerLink]="service.workspace.path('tests', run.request.suiteId, result.testId, 'history')" [queryParams]="{ run: run.id }">{{ service.testLabels()[result.testId] || result.testId }}</a> } @else { {{ service.testLabels()[result.testId] || result.testId }} }</strong><code class="table-subtitle">{{ result.testId }}</code><button type="button" class="button small secondary" [attr.aria-label]="'Inspect test ' + result.testId" (click)="inspect(result)">Inspect test</button></td><td><span class="badge" [class.timeout]="result.state === 'timed-out'" [class.success]="result.state === 'passed'" [class.warning]="result.state === 'flaky' || result.state === 'failed'">{{ result.state }}</span></td><td>{{ result.attempt }}</td><td><ts-timing-result [result]="result" /></td><td>@if (result.observedState) { <span class="table-subtitle">Functional: {{ result.observedState }}</span> }<p class="small wrap-anywhere">{{ service.resultSummaries()[result.testId] || '—' }}</p></td></tr> } @empty { <tr><td colspan="5">No final results are recorded yet.</td></tr> }
        </tbody></table></div>
      </section>
      <ts-run-workers /><ts-run-evidence />
      @if (service.workspace.can('notifications.read')) { <ts-notification-deliveries /> }
      @if (service.workspace.can('evidence.read')) { <ts-run-retention /><ts-run-artifacts /> }
    } @else if (service.loading()) { <p class="loading-state" role="status">Loading run details…</p> }
  `
})
export class RunDetailPage {
  readonly service = inject(RunDetailService)
  private readonly injector = inject(Injector)
  private readonly investigation = viewChild(RunEvidenceComponent)
  inspect(result: Attempt): void {
    void this.service.evidence.inspectTest(result.testId)
    afterNextRender(() => this.investigation()?.focus(), { injector: this.injector })
  }
}
