import { DatePipe, DecimalPipe, JsonPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { TestRevisionsComponent } from '../catalog/test-revisions.component'
import { TestHistoryService } from './test-history.service'
import { ReliabilityHistoryComponent } from './reliability-history.component'

@Component({
  standalone: true, imports: [DatePipe, DecimalPipe, JsonPipe, RouterLink, TestRevisionsComponent, ReliabilityHistoryComponent], providers: [TestHistoryService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Test history</h1><p class="muted">Follow this permanent test identity through its saved runs and definitions.</p></div><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.load()">Refresh history</button></div>
    <div class="row-actions history-links"><a [routerLink]="service.workspace.path('catalog')" [queryParams]="{ suite: service.suite() }">Test catalog</a><a [routerLink]="['/projects', service.project(), 'releases']" [queryParams]="{ suiteId: service.suite() }">Suite versions</a>@if (service.returnRun()) { <a [routerLink]="service.workspace.path('runs', service.returnRun())" [queryParams]="{ test: service.test() }">Return to run</a> }</div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }} Saved history is not changed.</p><button type="button" class="button secondary" (click)="service.refreshAccess()">Refresh access</button> }
    @if (service.loading()) { <p role="status">Loading retained history…</p> }
    @if (service.context(); as context) {
      <section class="panel" aria-labelledby="retained-test-title"><h2 id="retained-test-title">{{ context.test.title }}</h2><p><code>{{ service.test() }}</code> · {{ context.suiteName }} · <span class="badge">{{ context.lifecycle }}</span></p>
        @if (context.suiteArchived) { <p class="notice">This suite is archived. Its definitions, runs, and retained evidence remain readable. Restore the suite before new execution.</p> }
        @if (!context.current) { <p class="notice">The working definition is removed. This page shows its last saved definition. Existing runs keep their original snapshots.</p> } @else if (context.lifecycle === 'archived') { <p class="notice">This test is archived. Its existing runs and retained evidence remain available.</p> }
        @if (context.test.description) { <p>{{ context.test.description }}</p> }
        <dl class="detail-grid"><dt>Type / runner</dt><dd>{{ context.test.type }} · {{ context.test.runner }}</dd><dt>Category</dt><dd>{{ context.test.category || 'Uncategorized' }}</dd><dt>Safety</dt><dd>{{ context.test.safety }} · {{ context.test.rationale }}</dd>@if (context.test.conditions) { <dt>Conditions</dt><dd>{{ context.test.conditions }}</dd> }<dt>Last saved</dt><dd>{{ context.updatedAt | date:'medium' }}</dd></dl>
        <details><summary>Advanced definition</summary><pre class="log-view">{{ context.test | json }}</pre></details>
      </section>
      @if (service.runAccess()) { <ts-reliability-history [project]="service.project()" [suite]="service.suite()" [test]="service.test()" [environment]="service.environment()" [location]="service.location()" /> }
      <section class="panel" aria-labelledby="test-runs-title"><div class="section-top"><h2 id="test-runs-title">Runs containing this test</h2>@if (service.runAccess()) { <a [routerLink]="service.workspace.path('trends')" [queryParams]="{ suite: service.suite(), test: service.test() }">Test trends</a> }</div>
        <p class="small muted">Run titles, results, version, and timing come from saved run snapshots. Evidence follows its normal access and retention rules.</p>
        @if (!service.runAccess()) { <p class="notice">Run history requires permission to read project runs.</p> }
        @if (service.runError()) { <p class="notice error" role="alert">{{ service.runError() }}</p><button type="button" class="button secondary" (click)="service.loadRuns()">Retry run history</button> }
        @for (run of service.runs(); track run.runId) {
          <article class="history-run" [attr.aria-label]="'Run ' + run.runId"><div class="section-top"><h3>{{ run.title }}</h3><span class="badge" [class.timeout]="run.testState === 'timed-out'">{{ run.testState }}</span></div><p class="small muted">{{ run.createdAt | date:'medium' }} · {{ run.environmentName || 'No target' }} · {{ run.executionLocation }}</p><p class="small">Run {{ run.runState }} · {{ run.runOutcome }} · Attempt {{ run.attempt || 'Not recorded' }}@if (run.durationMs !== undefined) { · {{ run.durationMs | number:'1.0-1' }} ms }</p>@if (run.error) { <p class="small failure">{{ run.error }}</p> }<p class="small muted">Capture: {{ run.evidenceState }}</p><a class="button secondary" [routerLink]="service.workspace.path('runs', run.runId)" [queryParams]="{ test: service.test() }">Inspect run and evidence</a><details><summary>Run and version identifiers</summary><p><code>{{ run.runId }}</code></p><p><code>{{ run.releaseId }}</code></p></details></article>
        } @empty { @if (service.runAccess() && !service.runError() && !service.loading() && !service.runLoading()) { <p class="muted">{{ service.cursor() ? 'No readable matches in this scan. Load older runs to continue.' : 'No readable saved runs contain this test.' }}</p> } }
        @if (service.partial()) { <p class="notice">This scan reached its limit. Load older runs to continue searching stored history.</p> }
        @if (service.runLoading()) { <p role="status">Loading run history…</p> }
        @if (service.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.runLoading()" (click)="service.loadRuns(true)">Load older runs</button> }
      </section>
      <section class="panel"><details (toggle)="service.loadRevisions($any($event.target).open)"><summary>Advanced revision history</summary>@if (service.revisionError()) { <p class="notice error" role="alert">{{ service.revisionError() }}</p><button type="button" class="button secondary" (click)="service.loadRevisions(true)">Retry revisions</button> } @else if (service.revisionLoading()) { <p role="status">Loading saved revisions…</p> } @else { <ts-test-revisions [revisions]="service.revisions()" /> }</details></section>
    }
  `,
  styles: [`:host { display: block; min-width: 0; } .page-heading, .section-top, .history-links { flex-wrap: wrap; gap: 12px; } .history-links { margin-bottom: 20px; } .history-run { border: 1px solid var(--border); border-radius: 8px; padding: 16px; margin: 16px 0; min-width: 0; } .history-run h3 { margin: 0; font-size: 15px; } code, .failure { overflow-wrap: anywhere; } .failure { white-space: pre-wrap; max-height: 180px; overflow: auto; } summary { cursor: pointer; } details { margin-top: 16px; }`]
})
export class TestHistoryPage { readonly service = inject(TestHistoryService) }
