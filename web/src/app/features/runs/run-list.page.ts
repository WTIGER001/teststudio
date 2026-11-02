import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { NewRunComponent } from './new-run.component'
import { RunListService } from './run-list.service'
import { RunLookupsService } from './run-lookups.service'

@Component({
  standalone: true, imports: [DatePipe, ReactiveFormsModule, RouterLink, NewRunComponent], providers: [RunListService, RunLookupsService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Test runs</h1><p class="muted">See what happened, with the evidence to explain it.</p></div>
      @if (service.workspace.can('runs.create')) { <button type="button" class="button primary" (click)="service.newRunOpen.set(!service.newRunOpen())">{{ service.newRunOpen() ? 'Close form' : '+ New run' }}</button> }
    </div>
    @if (service.newRunOpen()) { <ts-new-run /> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="runs-title">
      <div class="section-top"><h2 id="runs-title">Run history</h2><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.load()">Refresh</button></div>
      <form class="run-filters" [formGroup]="service.filters" (ngSubmit)="service.load()">
        <div class="field"><label for="filter-environment">Environment</label><select id="filter-environment" formControlName="environment"><option value="">All environments</option><option value="none">No target</option>@for (environment of service.lookups.environments(); track environment.id) { <option [value]="environment.id">{{ environment.data.name }}</option> }</select></div>
        <div class="field"><label for="filter-suite">Suite</label><select id="filter-suite" formControlName="suite"><option value="">All suites</option>@for (suite of service.lookups.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div>
        <div class="field"><label for="filter-state">Run state</label><select id="filter-state" formControlName="state"><option value="">All states</option><option value="requested">Queued</option><option value="running">Running</option><option value="completed">Completed</option><option value="cancelled">Cancelled</option><option value="timed-out">Timed out</option><option value="blocked">Blocked</option><option value="incomplete">Incomplete</option></select></div>
        <div class="field"><label for="filter-selection">Selection</label><select id="filter-selection" formControlName="selection"><option value="">All selections</option><option value="full">Full suite</option><option value="selection">Selected tests</option></select></div>
        <button type="submit" class="button secondary" [disabled]="service.loading()">Apply filters</button>
      </form>
      @if (service.items().length) {
        <div class="table-scroll"><table><caption class="sr-only">Runs, newest first</caption><thead><tr><th scope="col">Run / suite</th><th scope="col">Environment</th><th scope="col">State / outcome</th><th scope="col">Results</th><th scope="col">Created</th><th scope="col">Evidence</th></tr></thead><tbody>
          @for (item of service.items(); track item.id) { <tr><td><a [routerLink]="['/projects', service.projectId(), 'runs', item.id]">{{ item.id }}</a><span class="table-subtitle muted">{{ item.data.suiteId }} · {{ item.data.selection }}</span></td><td>{{ item.data.environmentName || 'No target' }}<span class="table-subtitle muted">{{ item.data.stage }}{{ item.data.protected ? ' · Protected' : '' }}</span></td><td><span class="badge">{{ item.data.state }}</span><span class="table-subtitle">{{ item.data.outcome }}</span></td><td>{{ item.data.counts.completed }} / {{ item.data.counts.total }}<span class="table-subtitle muted">{{ item.data.counts.passed }} passed · {{ item.data.counts.failed }} failed · {{ item.data.counts.flaky }} flaky</span></td><td><time [attr.datetime]="item.data.createdAt">{{ item.data.createdAt | date:'medium' }}</time></td><td>{{ item.data.evidenceState }}</td></tr> }
        </tbody></table></div>
      } @else if (!service.loading()) { <div class="empty-state"><h3>No runs found</h3><p>Publish a catalog release, then start a run against a registered environment.</p></div> }
      @if (service.loading()) { <p role="status" class="loading-state">Loading runs…</p> }
      @if (service.partial()) { <p class="notice">The scan reached its page limit. Continue to search older runs.</p> }
      @if (service.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.load(true)">Load older runs</button> }
    </section>
  `
})
export class RunListPage { readonly service = inject(RunListService) }
