import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute, Router } from '@angular/router'
import { combineLatest } from 'rxjs'
import { ApiService } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { errorMessage } from '../runs/run.utils'
import type { TrendReport } from './trends.models'
import { defaultTrendFilters, readTrendFilters, trendFilterIssue, trendRequest, trendTitle } from './trends.utils'

@Injectable()
export class TrendsService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly projectId = signal('')
  readonly applicationId = signal('')
  readonly applied = signal(defaultTrendFilters(Date.now()))
  readonly title = computed(() => trendTitle(this.applied()))
  readonly cursor = signal('')
  readonly report = signal<TrendReport | null>(null)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({
    from: new FormControl('', { nonNullable: true }), to: new FormControl('', { nonNullable: true }),
    suite: new FormControl('', { nonNullable: true }), environment: new FormControl('', { nonNullable: true }),
    category: new FormControl('', { nonNullable: true }), test: new FormControl('', { nonNullable: true }), location: new FormControl('', { nonNullable: true })
  })
  private generation = 0

  constructor() {
    combineLatest([this.route.paramMap, this.route.queryParams]).pipe(takeUntilDestroyed(this.destroy)).subscribe(([params, query]) => {
      this.applicationId.set(params.get('applicationId') ?? ''); this.projectId.set(params.get('projectId') ?? '')
      const filters = readTrendFilters(query, Date.now())
      this.applied.set(filters); this.form.reset(filters)
      this.cursor.set(typeof query['cursor'] === 'string' ? query['cursor'] : '')
      void this.load()
    })
    this.destroy.onDestroy(() => { this.generation++ })
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.report.set(null); this.error.set('')
    const issue = trendFilterIssue(this.applied())
    if (issue) { this.loading.set(false); this.error.set(issue); return }
    this.loading.set(true)
    try {
      const report = await this.api.request<TrendReport>(`/api/v1/projects/${encodeURIComponent(this.projectId())}/trends?${trendRequest(this.applied(), this.cursor())}&application=${encodeURIComponent(this.applicationId())}`)
      if (generation === this.generation) this.report.set(report)
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Trend history could not be loaded.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async apply(): Promise<void> {
    const filters = this.form.getRawValue()
    const issue = trendFilterIssue(filters)
    if (issue) { this.error.set(issue); return }
    const changed = await this.router.navigate([], { relativeTo: this.route, queryParams: filters })
    if (!changed) await this.load()
  }

  older(): void {
    const cursor = this.report()?.cursor
    if (cursor) void this.router.navigate([], { relativeTo: this.route, queryParams: { ...this.applied(), cursor } })
  }

  newest(): void { void this.router.navigate([], { relativeTo: this.route, queryParams: this.applied() }) }
}
