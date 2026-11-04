import { RunAttemptComponent } from './run-attempt.component'
import { RunStepsComponent } from './run-steps.component'
import { TimingResultComponent } from '../timing/timing-result.component'
import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, ElementRef, inject, viewChild } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { diagnosticAttempts } from './run-diagnostics.utils'
import { RunDetailService } from './run-detail.service'
import { RunEvidenceService } from './run-evidence.service'
import { HTTPObservationComponent } from './http-observation.component'
import { GoObservationComponent } from './go-observation.component'

@Component({
  selector: 'ts-run-evidence', standalone: true, imports: [RunAttemptComponent, DatePipe, FormsModule, TimingResultComponent, RunStepsComponent, HTTPObservationComponent, GoObservationComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="investigate-title">
      <div class="section-top"><div><h2 #heading tabindex="-1" id="investigate-title">Investigate a test</h2><p class="small muted">Choose a test to keep its attempts, logs, and artifacts together. Final counts stay unchanged.</p></div><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.load()">Refresh evidence</button></div>
      <div class="form-grid"><div class="field"><label for="evidence-test">Test evidence scope</label><select id="evidence-test" [ngModel]="service.focusedTest()" (ngModelChange)="service.inspectTest($event)"><option value="">All tests</option>@for (test of detail.testChoices(); track test.id) { <option [value]="test.id">{{ test.label }}</option> }</select></div>
        @if (service.focusedTest()) { <div class="field"><label for="evidence-attempt">Attempt evidence scope</label><select id="evidence-attempt" [ngModel]="service.focusedAttempt()" (ngModelChange)="service.inspectTest(service.focusedTest(), $event)"><option [ngValue]="0">All attempts</option>@for (attempt of attempts(); track attempt) { <option [ngValue]="attempt">Attempt {{ attempt }}</option> }</select></div> }
      </div>
      @if (service.focusedTest()) { <p class="notice" role="status">Inspecting {{ detail.testLabels()[service.focusedTest()] || service.focusedTest() }} · {{ service.focusedAttempt() ? 'Attempt ' + service.focusedAttempt() : 'All attempts, including unsuccessful retries' }}. Run-level events remain available under Advanced execution events.</p> }
      @if (service.loading()) { <p class="small muted" role="status">Loading evidence for this selection…</p> }
    </section>
    @if (service.error()) { <p class="notice warning" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="attempts-title">
      <div class="section-top"><h2 id="attempts-title">Attempt history</h2><span class="small muted">Retries remain separate from final test counts.</span></div>
      @if (service.attempts().length) {
        @if (service.focusedTest()) { @for (item of service.attempts(); track item.id) { <ts-run-attempt [attempt]="item.data" /> } } @else {
        <div class="table-scroll"><table><caption class="sr-only">Every retained test attempt</caption><thead><tr><th scope="col">Test / attempt</th><th scope="col">Result</th><th scope="col">Duration</th><th scope="col">Observation</th><th scope="col">Started</th></tr></thead><tbody>
          @for (item of service.attempts(); track item.id) { <tr><td><code>{{ item.data.testId }}</code><span class="table-subtitle muted">Attempt {{ item.data.attempt }}</span></td><td><span class="badge" [class.timeout]="item.data.state === 'timed-out'" [class.success]="item.data.state === 'passed'" [class.warning]="item.data.state !== 'passed'">{{ item.data.state }}</span><span class="table-subtitle">{{ item.data.failureKind }}</span>@if (item.data.recovered) { <span class="table-subtitle muted">Recovered observation · original run outcome retained</span> }@if (item.data.observedState) { <span class="table-subtitle muted">Functional: {{ item.data.observedState }}</span> }</td><td><ts-timing-result [result]="item.data" /></td><td>@if (item.data.http; as http) { <ts-http-observation [http]="http" [observedAt]="item.data.finishedAt" /> } @if (item.data.go; as go) { <ts-go-observation [go]="go" /> } @if (item.data.error) { <details [open]="!!service.focusedTest()"><summary>Failure details</summary><pre class="step-error">{{ item.data.error }}</pre></details> }<ts-run-steps [steps]="item.data.steps ?? []" [revealFailures]="!!service.focusedTest()" /></td><td><time [attr.datetime]="item.data.startedAt">{{ item.data.startedAt | date:'medium' }}</time></td></tr> }
        </tbody></table></div>
        }
      } @else { <p class="small muted">{{ service.loading() ? 'Loading attempts…' : service.error() ? 'Attempt data is unavailable. Read the evidence warning and retry.' : service.cursors()['attempts'] ? 'No matching attempts found in this scan. Load more attempts to continue.' : service.focusedTest() ? 'No attempts are recorded for this selection.' : 'No attempts are recorded yet.' }}</p> }
      @if (service.cursors()['attempts']) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.more('attempts')">Load more attempts</button> }
    </section>
    <section class="panel" aria-labelledby="logs-title">
      <div class="section-top"><h2 id="logs-title">Run logs</h2><button type="button" class="button secondary" [disabled]="service.logLoading()" (click)="service.loadLogs()">Refresh logs</button></div>
      <details><summary>Advanced log filters</summary><div class="run-filters">
        <div class="field"><label for="log-stream">Stream</label><input id="log-stream" [ngModel]="service.stream()" (ngModelChange)="service.stream.set($event)" placeholder="All streams"></div>
        @if (!service.focusedTest()) { <div class="field"><label for="log-test">Test identifier</label><input id="log-test" [ngModel]="service.test()" (ngModelChange)="service.test.set($event)" placeholder="All tests"></div> }
        <button type="button" class="button secondary" [disabled]="service.logLoading()" (click)="service.loadLogs()">Filter logs</button>
      </div></details>
      @if (service.logError()) { <p class="notice error" role="alert">{{ service.logError() }}</p> }
      @if (service.logs().truncated) { <p class="notice">This view shows the latest bounded log segments. Download log artifacts to inspect retained content.</p> }
      <div class="log-view" tabindex="0" role="region" aria-label="Latest run log segments">
        @for (segment of service.logs().segments; track segment.artifact.id) {
          <div class="log-heading"><time [attr.datetime]="segment.artifact.createdAt">{{ segment.artifact.createdAt | date:'mediumTime' }}</time> · {{ segment.artifact.stream }} · {{ segment.artifact.testId || 'Run' }}@if (segment.artifact.attempt) { · attempt {{ segment.artifact.attempt }} }</div>
          @if (segment.artifact.state === 'available') { <pre>{{ segment.text }}</pre> } @else { <p class="small">Content {{ segment.artifact.state }}. {{ segment.artifact.error }}</p> }
        } @empty { <p class="small muted">{{ service.logLoading() ? 'Loading logs…' : service.logError() ? 'Log content is unavailable. Read the log warning and retry.' : 'No log segments match this selection.' }}</p> }
      </div>
    </section>
    <details class="panel"><summary>Advanced execution events</summary><section aria-labelledby="events-title">
      <h2 id="events-title">Execution events</h2>
      <ol class="event-list">@for (event of service.events(); track event.id) { <li><span class="badge">{{ event.data.kind }}</span><time class="small muted" [attr.datetime]="event.data.at">{{ event.data.at | date:'mediumTime' }}</time><p>{{ event.data.message }}</p>@if (event.data.testId) { <code>{{ event.data.testId }} · attempt {{ event.data.attempt }}</code> }</li> } @empty { <li>No events are recorded yet.</li> }</ol>
      @if (service.cursors()['events']) { <button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.more('events')">Load more events</button> }
    </section></details>
  `
})
export class RunEvidenceComponent {
  readonly service = inject(RunEvidenceService)
  readonly detail = inject(RunDetailService)
  readonly attempts = computed(() => diagnosticAttempts(this.detail.run(), this.service.focusedTest(), this.service.attempts()))
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  focus(): void { this.heading()?.nativeElement.focus() }
}
