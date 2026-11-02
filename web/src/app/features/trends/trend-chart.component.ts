import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, input, signal } from '@angular/core'
import { RouterLink } from '@angular/router'
import type { TrendPoint } from './trends.models'
import { stackedChart, trendStates } from './trend-chart.utils'

@Component({
  selector: 'ts-trend-chart', standalone: true, imports: [DatePipe, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <figure class="trend-figure">
      <figcaption><h2 id="trend-chart-title">{{ title() }}</h2><p class="small muted">Final logical results per finished run, in creation order. Vertical axis: test counts. Dates use UTC.</p></figcaption>
      <ul class="trend-legend" aria-label="Result states">@for (state of states; track state.key) { <li><span class="trend-swatch" [class]="'trend-swatch state-' + state.key" aria-hidden="true"></span>{{ state.label }}</li> }</ul>
      <div class="trend-chart-scroll"><svg viewBox="0 0 900 294" role="group" aria-labelledby="trend-chart-title" aria-describedby="trend-chart-description">
        <desc id="trend-chart-description">Stacked area chart. Each band shows a final result state. Striped gaps indicate unavailable selections. Focus a run to inspect counts or open its details.</desc>
        <defs><pattern id="trend-gap-pattern" width="8" height="8" patternUnits="userSpaceOnUse"><path d="M0,8 L8,0" class="trend-gap-line" /></pattern></defs>
        @for (tick of chart().ticks; track tick.value) { <line [attr.x1]="chart().left" [attr.x2]="chart().right" [attr.y1]="tick.y" [attr.y2]="tick.y" class="trend-grid" /><text x="38" [attr.y]="tick.y + 4" text-anchor="end" class="trend-axis">{{ tick.value }}</text> }
        @for (column of chart().columns; track column.point.runId) { @if (column.point.gap) { <rect [attr.x]="column.x" [attr.y]="chart().top" [attr.width]="column.width" [attr.height]="chart().bottom - chart().top" fill="url(#trend-gap-pattern)" /> } }
        @for (area of chart().areas; track area.key) { <path [attr.d]="area.path" [class]="'trend-area state-' + area.key" /> }
        @for (column of chart().columns; track column.point.runId) {
          <a [routerLink]="['/projects', project(), 'runs', column.point.runId]" [attr.aria-label]="column.summary" (mouseenter)="selected.set(column.index)" (focus)="selected.set(column.index)">
            <rect [attr.x]="column.x" [attr.y]="chart().top" [attr.width]="column.width" [attr.height]="chart().bottom - chart().top" class="trend-hit" />
          </a>
        }
        @if (points().length) { <text x="48" y="281" class="trend-axis">{{ points()[0].at | date:'MMM d, HH:mm':'UTC' }}</text><text x="884" y="281" text-anchor="end" class="trend-axis">{{ points()[points().length - 1].at | date:'MMM d, HH:mm':'UTC' }}</text> }
      </svg></div>
      <div class="trend-tooltip" role="tooltip">@if (active(); as column) { <strong>{{ column.point.at | date:'medium':'UTC' }} UTC · {{ column.point.suiteId }}</strong><p>{{ column.summary }}</p> } @else { <p>Hover or focus a run to inspect state counts. Select it to open run details.</p> }</div>
      <p class="small muted">On narrow screens, scroll the chart horizontally to see all runs.</p>
      <details class="trend-values"><summary>View chart values</summary><div class="table-scroll"><table><caption>Accessible trend values, oldest first. Gaps have no measured counts.</caption><thead><tr><th scope="col">Run / created UTC</th>@for (state of states; track state.key) { <th scope="col">{{ state.label }}</th> }<th scope="col">Total</th></tr></thead><tbody>
        @for (point of points(); track point.runId) { <tr><th scope="row"><a [routerLink]="['/projects', project(), 'runs', point.runId]"><code>{{ point.runId }}</code></a><span class="table-subtitle">{{ point.at | date:'medium':'UTC' }} UTC</span></th>@if (point.gap) { <td colspan="7">{{ point.gap }}</td> } @else { @for (state of states; track state.key) { <td>{{ point.counts[state.key] }}</td> }<td>{{ point.counts.total }}</td> }</tr> }
      </tbody></table></div></details>
    </figure>
  `,
  styleUrl: './trends.css'
})
export class TrendChartComponent {
  readonly project = input.required<string>()
  readonly title = input.required<string>()
  readonly points = input.required<TrendPoint[]>()
  readonly states = trendStates
  readonly chart = computed(() => stackedChart(this.points()))
  readonly selected = signal<number | null>(null)
  readonly active = computed(() => this.chart().columns[this.selected() ?? -1])
}
