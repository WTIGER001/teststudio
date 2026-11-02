import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { TrendChartComponent } from './trend-chart.component'
import { TrendRankingsComponent } from './trend-rankings.component'
import { TrendsService } from './trends.service'

@Component({
  standalone: true, imports: [DatePipe, ReactiveFormsModule, TrendChartComponent, TrendRankingsComponent], providers: [TrendsService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Trends</h1><p class="muted">See how final test results change across runs.</p></div><button class="button secondary" type="button" [disabled]="service.loading()" (click)="service.load()">Refresh trends</button></div>
    <section class="panel" aria-labelledby="trend-scope-title"><h2 id="trend-scope-title">History scope</h2>
      <form [formGroup]="service.form" (ngSubmit)="service.apply()"><div class="trend-filter-grid">
        <div class="field"><label for="trend-from">From date (UTC)</label><input id="trend-from" type="date" formControlName="from" required></div>
        <div class="field"><label for="trend-to">Through date (UTC)</label><input id="trend-to" type="date" formControlName="to" required></div>
        <div class="field"><label for="trend-suite">Suite identifier</label><input id="trend-suite" list="trend-suites" formControlName="suite" placeholder="All suites"><datalist id="trend-suites">@for (item of service.report()?.options.suites; track item.id) { <option [value]="item.id">{{ item.name }}</option> }</datalist></div>
        <div class="field"><label for="trend-environment">Environment identifier</label><input id="trend-environment" list="trend-environments" formControlName="environment" placeholder="All environments"><datalist id="trend-environments">@for (item of service.report()?.options.environments; track item.id) { <option [value]="item.id">{{ item.name }}</option> }</datalist><small>Use none for targetless runs.</small></div>
        <div class="field"><label for="trend-category">Category</label><input id="trend-category" list="trend-categories" formControlName="category" placeholder="All categories"><datalist id="trend-categories">@for (item of service.report()?.options.categories; track item) { <option [value]="item"></option> }</datalist></div>
        <div class="field"><label for="trend-test">Permanent test identifier</label><input id="trend-test" formControlName="test" placeholder="All tests"><small>Choose its suite to keep test identities separate.</small></div>
        <div class="field"><label for="trend-location">Execution location</label><input id="trend-location" list="trend-locations" formControlName="location" placeholder="All locations"><datalist id="trend-locations">@for (item of service.report()?.options.locations; track item) { <option [value]="item"></option> }</datalist></div>
      </div><button class="button primary" type="submit" [disabled]="service.loading() || service.form.invalid">Apply scope</button></form>
    </section>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.loading()) { <p class="loading-state" role="status">Loading trend history…</p> }
    @if (service.report(); as report) {
      <p class="notice" role="status">{{ report.sampleCount }} result-bearing runs · {{ report.totals.total }} logical results · {{ report.contextRuns - report.sampleCount }} selection gaps. Period: {{ report.filter.since | date:'mediumDate':'UTC' }} through {{ report.filter.until | date:'mediumDate':'UTC' }} UTC. Project: {{ report.projectId }}. Environment: {{ report.filter.environment || 'all' }}. Location: {{ report.filter.location || 'all' }}.</p>
      @if (report.partial || service.cursor()) { <p class="notice warning">This is a bounded history page, not the complete period. Counts and rankings cover only the displayed page. Use older history or return to the newest page.</p> }
      @if (report.options.partial) { <p class="notice">Filter suggestions are limited. Enter an exact identifier or category to search beyond the suggestions.</p> }
      @if (report.points.length) { <section class="panel"><ts-trend-chart [points]="report.points" [project]="service.projectId()" [title]="service.title()" /></section> } @else { <div class="empty-state"><h2>No finished runs in this history page.</h2><p>Choose another scope or browse older history. Active runs do not enter result-state trends.</p></div> }
      <div class="row-actions wrap-actions trend-pagination">@if (report.cursor) { <button class="button secondary" type="button" (click)="service.older()">Older history</button> }@if (service.cursor()) { <button class="button secondary" type="button" (click)="service.newest()">Newest history</button> }</div>
      <ts-trend-rankings [project]="service.projectId()" [filters]="service.applied()" [rankings]="report.failureRankings" />
      <ts-trend-rankings [project]="service.projectId()" [filters]="service.applied()" [rankings]="report.volatilityRankings" [volatility]="true" />
    }
  `,
  styleUrl: './trends.css'
})
export class TrendsPage { readonly service = inject(TrendsService) }
