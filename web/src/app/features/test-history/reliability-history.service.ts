import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import type { StatusReport, StatusSelection } from '@wtiger001/teststudio-angular'
import { AccessClientService } from '../../core/access-client.service'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'

@Injectable()
export class ReliabilityHistoryService {
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
  readonly range = signal<{ since: string; until: string } | null>(null)
  private scope = { project: '', suite: '', test: '' }
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++ }) }
  async configure(project: string, suite: string, test: string, environment: string, location: string): Promise<void> {
    this.scope = { project, suite, test }; this.environment.set(environment); this.location.set(location); this.range.set(null)
    const generation = ++this.generation; this.report.set(null)
    try {
      const page = await this.api.request<Page<RecordValue<{ id: string; name: string }>>>('/api/v1/projects/' + encodeURIComponent(project) + '/environments')
      if (generation !== this.generation) return
      this.environments.set(page.items.map(row => row.data))
    } catch { if (generation === this.generation) this.environments.set([]) }
    if (generation === this.generation) await this.load()
  }
  async load(window?: string): Promise<void> {
    if (window) { this.window.set(window); this.range.set(null) }
    const generation = ++this.generation
    this.loading.set(true); this.report.set(null); this.error.set('')
    try {
      const report = await this.client.call('readTestReliability', { ...this.scope, query: { environment: this.environment() || undefined, location: this.location() || undefined, window: this.window(), maxAgeSeconds: this.monitoring() ? this.freshness() : 0, ...this.range() } })
      if (generation === this.generation) this.report.set(report)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Reliability history is unavailable.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  drilldown(selection: StatusSelection): void { this.range.set({ since: selection.bucket.since, until: selection.bucket.until }); void this.load() }
  resetRange(): void { this.range.set(null); void this.load() }
}
