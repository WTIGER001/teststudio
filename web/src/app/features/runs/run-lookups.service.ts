import { inject, Injectable, signal } from '@angular/core'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { Suite, Environment } from '../configuration/configuration.service'

@Injectable()
export class RunLookupsService {
  private readonly api = inject(ApiService)
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly environments = signal<RecordValue<Environment>[]>([])
  private generation = 0

  async load(project: string): Promise<void> {
    const generation = ++this.generation
    this.suites.set([]); this.environments.set([])
    const base = `/api/v1/projects/${encodeURIComponent(project)}`
    const [suites, environments] = await Promise.all([
      this.api.request<Page<RecordValue<Suite>>>(`${base}/suites`),
      this.api.request<Page<RecordValue<Environment>>>(`${base}/environments`)
    ])
    if (generation !== this.generation) return
    this.suites.set(suites.items); this.environments.set(environments.items)
  }
}
