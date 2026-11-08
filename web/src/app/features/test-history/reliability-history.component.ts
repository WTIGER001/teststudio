import { ChangeDetectionStrategy, Component, effect, inject, input } from '@angular/core'
import { TestStudioStatusPageComponent } from '@wtiger001/teststudio-angular'
import { ReliabilityHistoryService } from './reliability-history.service'

@Component({
  selector: 'ts-reliability-history', standalone: true, imports: [TestStudioStatusPageComponent], providers: [ReliabilityHistoryService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<section class="panel" aria-label="Reliability history"><h2>Reliability history</h2><p class="small muted">Each pill summarizes an interval. Pass rate excludes skipped and unknown outcomes. Enable a monitoring policy to estimate availability.</p>
    <div class="filters"><label>Reliability environment<select aria-label="Reliability environment" [value]="service.environment()" (change)="service.environment.set($any($event.target).value); service.range.set(null); service.load()"><option value="">No target</option>@for (environment of service.environments(); track environment.id) { <option [value]="environment.id">{{ environment.name }}</option> }</select></label></div>
    <details><summary>Advanced reliability settings</summary><div class="filters"><label>Execution location<input placeholder="Installation default" [value]="service.location()" (change)="service.location.set($any($event.target).value); service.load()"></label><label class="check"><input type="checkbox" [checked]="service.monitoring()" (change)="service.monitoring.set($any($event.target).checked); service.load()">Treat this check as a monitor</label>@if (service.monitoring()) { <label>Monitoring freshness (seconds)<input type="number" min="60" max="86400" [value]="service.freshness()" (change)="service.freshness.set(+$any($event.target).value); service.load()"></label> }</div><p class="small muted">This changes the report policy. It does not create a schedule, execute a check, or change frozen test criteria.</p></details>
    @if (service.loading()) { <p role="status">Loading reliability history…</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p><button type="button" class="button secondary" (click)="service.load()">Retry reliability</button> }
    @if (service.range()) { <button type="button" class="button secondary" (click)="service.resetRange()">Return to full period</button> }
    @if (service.report(); as report) { <teststudio-status-page [report]="report" (windowChanged)="service.load($event)" (drilldownRequested)="service.drilldown($event)" /> }
  </section>`,
  styles: `:host {display:block;} .filters {display:flex;gap:12px;flex-wrap:wrap;margin:12px 0;} label {display:grid;gap:6px;} .check {display:flex;align-items:center;} select,input {max-width:100%;} details {margin:12px 0;} teststudio-status-page {--ts-status-padding:0;--ts-status-row-padding:16px;}`
})
export class ReliabilityHistoryComponent {
  readonly project = input.required<string>()
  readonly suite = input.required<string>()
  readonly test = input.required<string>()
  readonly environment = input('')
  readonly location = input('')
  readonly service = inject(ReliabilityHistoryService)
  constructor() { effect(() => { void this.service.configure(this.project(), this.suite(), this.test(), this.environment(), this.location()) }) }
}
