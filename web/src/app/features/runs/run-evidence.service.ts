import { inject, Injectable, signal } from '@angular/core'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { Artifact, Attempt, LogPage, RunEvent } from './run.models'
import { errorMessage, queryString } from './run.utils'

@Injectable()
export class RunEvidenceService {
  private readonly api = inject(ApiService)
  readonly base = signal('')
  readonly attempts = signal<RecordValue<Attempt>[]>([])
  readonly events = signal<RecordValue<RunEvent>[]>([])
  readonly artifacts = signal<RecordValue<Artifact>[]>([])
  readonly logs = signal<LogPage>({ segments: [], truncated: false })
  readonly cursors = signal<Record<string, string>>({})
  readonly stream = signal('')
  readonly test = signal('')
  readonly error = signal('')
  readonly logError = signal('')
  readonly loading = signal(false)
  private generation = 0
  private logGeneration = 0

  reset(project: string, run: string): void {
    this.generation++; this.logGeneration++
    this.base.set(`/api/v1/projects/${encodeURIComponent(project)}/runs/${encodeURIComponent(run)}`)
    this.attempts.set([]); this.events.set([]); this.artifacts.set([]); this.cursors.set({})
    this.logs.set({ segments: [], truncated: false }); this.error.set(''); this.logError.set('')
    this.stream.set(''); this.test.set('')
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    const results = await Promise.allSettled([
      this.api.request<Page<RecordValue<Attempt>>>(`${this.base()}/attempts?limit=100`),
      this.api.request<Page<RecordValue<RunEvent>>>(`${this.base()}/events?limit=100`),
      this.api.request<Page<RecordValue<Artifact>>>(`${this.base()}/artifacts?limit=100`)
    ])
    if (generation !== this.generation) return
    const [attempts, events, artifacts] = results
    if (attempts.status === 'fulfilled') this.attempts.set(attempts.value.items)
    if (events.status === 'fulfilled') this.events.set(events.value.items)
    if (artifacts.status === 'fulfilled') this.artifacts.set(artifacts.value.items)
    this.cursors.set({ attempts: attempts.status === 'fulfilled' ? attempts.value.cursor ?? '' : '', events: events.status === 'fulfilled' ? events.value.cursor ?? '' : '', artifacts: artifacts.status === 'fulfilled' ? artifacts.value.cursor ?? '' : '' })
    const failed = results.find(r => r.status === 'rejected')
    if (failed?.status === 'rejected') this.error.set(errorMessage(failed.reason, 'Some evidence could not be loaded.'))
    await this.loadLogs()
    if (generation === this.generation) this.loading.set(false)
  }

  async more(kind: 'attempts' | 'events' | 'artifacts'): Promise<void> {
    const cursor = this.cursors()[kind]
    if (!cursor || this.loading()) return
    const generation = this.generation
    this.loading.set(true); this.error.set('')
    try {
      const page = await this.api.request<Page<RecordValue<Attempt | RunEvent | Artifact>>>(`${this.base()}/${kind}?limit=100&cursor=${encodeURIComponent(cursor)}`)
      if (generation !== this.generation) return
      if (kind === 'attempts') this.attempts.update(items => [...items, ...page.items as RecordValue<Attempt>[]])
      if (kind === 'events') this.events.update(items => [...items, ...page.items as RecordValue<RunEvent>[]])
      if (kind === 'artifacts') this.artifacts.update(items => [...items, ...page.items as RecordValue<Artifact>[]])
      this.cursors.update(values => ({ ...values, [kind]: page.cursor ?? '' }))
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'More evidence could not be loaded.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async loadLogs(): Promise<void> {
    const generation = ++this.logGeneration
    this.logError.set('')
    try {
      const query = queryString({ stream: this.stream(), test: this.test(), tail: 'true', limit: '50' })
      const logs = await this.api.request<LogPage>(`${this.base()}/logs?${query}`)
      if (generation === this.logGeneration) this.logs.set(logs)
    } catch (error) { if (generation === this.logGeneration) this.logError.set(errorMessage(error, 'Logs could not be loaded.')) }
  }

  invalidate(): void { this.generation++; this.logGeneration++ }
}
