import { DatePipe, DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, input, output } from '@angular/core'
import type { StatusPageReport, StatusPageRow } from '../../core/generated/access.types'

import { statusWindows, statusTextColor, monitoredLabel } from './status.utils'
export { statusWindows, statusTextColor, monitoredLabel } from './status.utils'

@Component({
  selector: 'ts-status-view', standalone: true, imports: [DatePipe, DecimalPipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="status-view" [style.background-color]="report().background" [style.color]="textColor(report().background)" aria-label="Status page">
      <header><h1>{{ report().title }}</h1><div class="range"><label for="status-range">Availability period</label><select id="status-range" [value]="report().window" (change)="windowChanged.emit($any($event.target).value)">@for (window of windows; track window.id) { <option [value]="window.id" [selected]="window.id === report().window">{{ window.label }}</option> }</select></div></header>
      <p class="as-of">Updated {{ report().asOf | date:'medium' }} · Refreshes every minute</p>
      @for (row of report().probes; track $index; let index = $index) {
        <article class="status-row" [attr.aria-label]="row.title"><div class="probe-content"><h2>{{ row.title }}</h2>@if (row.description) { <p class="description">{{ row.description }}</p> }<p class="current">{{ state(row.state) }}</p>
          <div class="pills" aria-label="Recent probe results">@for (point of row.history; track $index) {
            <button type="button" class="pill" [class.passed]="point.state === 'passed'" [class.failed]="point.state === 'failed'" [class.timeout]="point.state === 'timed-out'" [class.flaky]="point.state === 'flaky'" [attr.aria-label]="state(point.state) + ' · ' + (point.at | date:'medium')" [attr.title]="state(point.state) + ' · ' + (point.at | date:'medium')"><span class="tooltip" role="tooltip">{{ state(point.state) }} · {{ point.at | date:'medium' }}</span></button>
          } @empty { <span>No observations in this period</span> }</div>
        </div><figure class="availability" [attr.aria-label]="row.availability.percent === null ? 'No monitored data' : (row.availability.percent | number:'1.2-2') + '% availability; ' + coverage(row.availability.coveragePercent) + ' monitored'">
          <svg viewBox="0 0 100 100" aria-hidden="true"><circle class="unknown" cx="50" cy="50" r="40" pathLength="100" /><circle class="up" cx="50" cy="50" r="40" pathLength="100" [attr.stroke-dasharray]="arc(row, 'up') + ' 100'" /><circle class="down" cx="50" cy="50" r="40" pathLength="100" [attr.stroke-dasharray]="arc(row, 'down') + ' 100'" [attr.stroke-dashoffset]="-arc(row, 'up')" /></svg>
          <div class="donut-label">@if (row.availability.percent !== null) { <strong>{{ row.availability.percent | number:'1.2-2' }}%</strong><span>availability</span> } @else { <strong>—</strong><span>No data</span> }</div>
          <figcaption>{{ coverage(row.availability.coveragePercent) }} monitored</figcaption>
        </figure></article>
      }
      <footer><span class="legend up-key">Available</span><span class="legend down-key">Unavailable</span><span class="legend unknown-key">Unknown</span><p>Estimated time available during monitored periods. Missing or stale checks are unknown. All time starts at each probe's first retained observation.</p></footer>
    </section>
  `,
  styleUrl: './status-view.css'
})
export class StatusViewComponent {
  readonly report = input.required<StatusPageReport>(); readonly windowChanged = output<string>(); readonly windows = statusWindows; readonly textColor = statusTextColor; readonly coverage = monitoredLabel
  state(value: string): string { return ({ passed: 'Available', failed: 'Unavailable', 'timed-out': 'Timed out', flaky: 'Unstable', unknown: 'Unknown or stale' } as Record<string, string>)[value] || 'Unknown' }
  arc(row: StatusPageRow, kind: 'up' | 'down'): number { const value = row.availability, total = value.upSeconds + value.downSeconds + value.unknownSeconds; return total ? 100 * (kind === 'up' ? value.upSeconds : value.downSeconds) / total : 0 }
}
