import { PercentPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, computed, inject, input } from '@angular/core'
import { WorkspaceService } from '../../core/workspace.service'
import { RouterLink } from '@angular/router'
import type { TrendFilters, TrendRanking } from './trends.models'
import { rankingRows } from './trends.utils'

@Component({
  selector: 'ts-trend-rankings', standalone: true, imports: [PercentPipe, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush, styleUrl: './trends.css',
  template: `
    <section class="panel" [attr.aria-labelledby]="volatility() ? 'volatility-title' : 'failure-ranking-title'">
      <h2 [id]="volatility() ? 'volatility-title' : 'failure-ranking-title'">{{ volatility() ? 'Most volatile tests' : 'Most failed tests' }}</h2>
      <p class="small muted">{{ volatility() ? 'State changes between adjacent known results for the same target and location. Gaps, skipped tests, and unknowns break comparisons.' : 'Ranked by failed plus timed-out logical results. Retries do not add counts.' }} Rankings cover this history page.</p>
      @if (rows().length) { <div class="table-scroll"><table class="trend-ranking-table"><caption class="sr-only">{{ volatility() ? 'Test volatility ranking' : 'Test failure ranking' }}</caption><thead><tr><th scope="col">Test / category</th><th scope="col">Samples</th><th scope="col">Failed</th><th scope="col">Timed out</th>@if (volatility()) { <th scope="col">Changes / comparisons</th><th scope="col">Volatility</th> }</tr></thead><tbody>
        @for (row of rows(); track row.key) { <tr><th scope="row"><a [routerLink]="workspace.path('trends')" [queryParams]="row.testQuery">{{ row.rank.title || row.rank.testId }}</a><code class="table-subtitle">{{ row.rank.suiteId }} / {{ row.rank.testId }}</code><a class="table-subtitle" [routerLink]="workspace.path('trends')" [queryParams]="row.categoryQuery">{{ row.rank.category }}</a></th><td>{{ row.rank.counts.total }}</td><td>{{ row.rank.counts.failed }}</td><td>{{ row.rank.counts.timedOut }}</td>@if (volatility()) { <td>{{ row.rank.changes }} / {{ row.rank.comparisons }}</td><td>{{ row.rank.volatility | percent:'1.0-1' }}</td> }</tr> }
      </tbody></table></div> } @else { <p class="notice">{{ volatility() ? 'At least two adjacent known observations are needed for volatility.' : 'No failed or timed-out logical results exist in this history page.' }}</p> }
    </section>
  `
})
export class TrendRankingsComponent {
  readonly workspace = inject(WorkspaceService)
  readonly project = input.required<string>()
  readonly filters = input.required<TrendFilters>()
  readonly rankings = input.required<TrendRanking[]>()
  readonly volatility = input(false)
  readonly rows = computed(() => rankingRows(this.filters(), this.rankings()))
}
