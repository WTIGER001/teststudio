import { RunStepsComponent } from './run-steps.component'
import { TimingResultComponent } from '../timing/timing-result.component'
import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { RunEvidenceService } from './run-evidence.service'

@Component({
  selector: 'ts-run-evidence', standalone: true, imports: [DatePipe, FormsModule, TimingResultComponent, RunStepsComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.error()) { <p class="notice warning" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="attempts-title">
      <div class="section-top"><h2 id="attempts-title">Attempt history</h2><span class="small muted">Retries remain separate from final test counts.</span></div>
      @if (service.attempts().length) {
        <div class="table-scroll"><table><caption class="sr-only">Every retained test attempt</caption><thead><tr><th scope="col">Test / attempt</th><th scope="col">Result</th><th scope="col">Duration</th><th scope="col">Observation</th><th scope="col">Started</th></tr></thead><tbody>
          @for (item of service.attempts(); track item.id) { <tr><td><code>{{ item.data.testId }}</code><span class="table-subtitle muted">Attempt {{ item.data.attempt }}</span></td><td><span class="badge" [class.timeout]="item.data.state === 'timed-out'" [class.success]="item.data.state === 'passed'" [class.warning]="item.data.state !== 'passed'">{{ item.data.state }}</span><span class="table-subtitle">{{ item.data.failureKind }}</span>@if (item.data.recovered) { <span class="table-subtitle muted">Recovered observation · original run outcome retained</span> }@if (item.data.observedState) { <span class="table-subtitle muted">Functional: {{ item.data.observedState }}</span> }</td><td><ts-timing-result [result]="item.data" /></td><td>@if (item.data.http; as http) { <span>HTTP {{ http.status || 'unavailable' }}</span><span class="table-subtitle wrap-anywhere">{{ http.url }}</span> } @if (item.data.error) { <span class="table-subtitle">{{ item.data.error }}</span> }<ts-run-steps [steps]="item.data.steps ?? []" /></td><td><time [attr.datetime]="item.data.startedAt">{{ item.data.startedAt | date:'medium' }}</time></td></tr> }
        </tbody></table></div>
      } @else { <p class="small muted">No attempts are recorded yet.</p> }
      @if (service.cursors()['attempts']) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.more('attempts')">Load more attempts</button> }
    </section>
    <section class="panel" aria-labelledby="logs-title">
      <div class="section-top"><h2 id="logs-title">Run logs</h2><button type="button" class="button secondary" (click)="service.loadLogs()">Refresh logs</button></div>
      <div class="run-filters">
        <div class="field"><label for="log-stream">Stream</label><input id="log-stream" [ngModel]="service.stream()" (ngModelChange)="service.stream.set($event)" placeholder="All streams"></div>
        <div class="field"><label for="log-test">Test identifier</label><input id="log-test" [ngModel]="service.test()" (ngModelChange)="service.test.set($event)" placeholder="All tests"></div>
        <button type="button" class="button secondary" (click)="service.loadLogs()">Filter logs</button>
      </div>
      @if (service.logError()) { <p class="notice error" role="alert">{{ service.logError() }}</p> }
      @if (service.logs().truncated) { <p class="notice">This view shows the latest bounded log segments. Download log artifacts to inspect retained content.</p> }
      <div class="log-view" tabindex="0" role="region" aria-label="Latest run log segments">
        @for (segment of service.logs().segments; track segment.artifact.id) {
          <div class="log-heading"><time [attr.datetime]="segment.artifact.createdAt">{{ segment.artifact.createdAt | date:'mediumTime' }}</time> · {{ segment.artifact.stream }} · {{ segment.artifact.testId || 'Run' }}@if (segment.artifact.attempt) { · attempt {{ segment.artifact.attempt }} }</div>
          @if (segment.artifact.state === 'available') { <pre>{{ segment.text }}</pre> } @else { <p class="small">Content {{ segment.artifact.state }}. {{ segment.artifact.error }}</p> }
        } @empty { <p class="small muted">No log segments match this selection.</p> }
      </div>
    </section>
    <section class="panel" aria-labelledby="events-title">
      <h2 id="events-title">Execution events</h2>
      <ol class="event-list">@for (event of service.events(); track event.id) { <li><span class="badge">{{ event.data.kind }}</span><time class="small muted" [attr.datetime]="event.data.at">{{ event.data.at | date:'mediumTime' }}</time><p>{{ event.data.message }}</p>@if (event.data.testId) { <code>{{ event.data.testId }} · attempt {{ event.data.attempt }}</code> }</li> } @empty { <li>No events are recorded yet.</li> }</ol>
      @if (service.cursors()['events']) { <button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.more('events')">Load more events</button> }
    </section>
  `
})
export class RunEvidenceComponent { readonly service = inject(RunEvidenceService) }
