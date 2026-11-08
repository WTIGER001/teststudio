import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import type { StatusReport, StatusCheck, StatusCheckLoader } from '@wtiger001/teststudio-angular'
import { Router } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'

@Injectable()
export class ReliabilityHistoryService {
  private readonly router = inject(Router)
  private readonly client = inject(AccessClientService)
  private readonly api = inject(ApiService)
  readonly report = signal<StatusReport | null>(null)
  readonly error = signal('')
  readonly loading = signal(false)
  readonly environments = signal<{ id: string; name: string }[]>([])
  readonly environment = signal('')
  readonly location = signal('')
  readonly window = signal('24h')
  readonly monitoring = signal(false)
  readonly freshness = signal(900)
  private scope = { project: '', suite: '', test: '' }
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++ }) }
  async configure(project: string, suite: string, test: string, environment: string, location: string): Promise<void> {
    this.scope = { project, suite, test }; this.environment.set(environment); this.location.set(location)
    const generation = ++this.generation; this.report.set(null)
    try {
      const page = await this.api.request<Page<RecordValue<{ id: string; name: string }>>>('/api/v1/projects/' + encodeURIComponent(project) + '/environments')
      if (generation !== this.generation) return
      this.environments.set(page.items.map(row => row.data))
    } catch { if (generation === this.generation) this.environments.set([]) }
    if (generation === this.generation) await this.load()
  }
  async load(window?: string): Promise<void> {
    if (window) { this.window.set(window) }
    const generation = ++this.generation
    this.loading.set(true); this.report.set(null); this.error.set('')
    try {
      const report = await this.client.call('readTestReliability', { ...this.scope, query: { environment: this.environment() || undefined, location: this.location() || undefined, window: this.window(), maxAgeSeconds: this.monitoring() ? this.freshness() : 0 } })
      if (generation === this.generation) this.report.set(report)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Reliability history is unavailable.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  readonly checks: StatusCheckLoader = query => this.client.call('readTestReliabilityChecks', { ...this.scope, query: { environment: this.environment() || undefined, location: this.location() || undefined, since: query.since, until: query.until, cursor: query.cursor, limit: query.limit } })
  inspect(check: StatusCheck): void { if (check.runId) void this.router.navigate(['/projects', this.scope.project, 'runs', check.runId], { queryParams: { test: this.scope.test } }) }
}
