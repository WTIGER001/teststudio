import { inject, Injectable, signal } from '@angular/core'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import type { Artifact, Attempt, LogPage, RunEvent } from './run.models'
import { errorMessage, queryString } from './run.utils'

type EvidenceKind = 'attempts' | 'events' | 'artifacts'

@Injectable()
export class RunEvidenceService {
  private readonly api = inject(ApiService)
  readonly base = signal('')
  readonly attempts = signal<RecordValue<Attempt>[]>([])
  readonly events = signal<RecordValue<RunEvent>[]>([])
  readonly artifacts = signal<RecordValue<Artifact>[]>([])
  readonly logs = signal<LogPage>({ segments: [], truncated: false })
  readonly cursors = signal<Record<string, string>>({})
  readonly focusedTest = signal('')
  readonly focusedAttempt = signal(0)
  readonly stream = signal('')
  readonly test = signal('')
  readonly error = signal('')
  readonly logError = signal('')
  readonly restricted = signal(false)
  readonly loading = signal(false)
  readonly logLoading = signal(false)
  private generation = 0
  private logGeneration = 0

  reset(project: string, run: string): void {
    this.generation++; this.logGeneration++
    this.base.set(`/api/v1/projects/${encodeURIComponent(project)}/runs/${encodeURIComponent(run)}`)
    this.clear(); this.restricted.set(false); this.error.set(''); this.logError.set(''); this.loading.set(false); this.logLoading.set(false)
    this.focusedTest.set(''); this.focusedAttempt.set(0); this.stream.set(''); this.test.set('')
  }

  async inspectTest(test: string, attempt = 0): Promise<void> {
    this.generation++; this.logGeneration++
    this.focusedTest.set(test); this.focusedAttempt.set(test ? attempt : 0); this.test.set(test); this.stream.set('')
    this.clear()
    await this.load()
  }

  private clear(): void { this.attempts.set([]); this.events.set([]); this.artifacts.set([]); this.cursors.set({}); this.logs.set({ segments: [], truncated: false }) }

  private query(kind: EvidenceKind, cursor = ''): string {
    return queryString({ limit: '100', cursor, test: kind === 'events' ? '' : this.focusedTest(), attempt: kind === 'events' || !this.focusedAttempt() ? '' : String(this.focusedAttempt()) })
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    const results = await Promise.allSettled([
      this.api.request<Page<RecordValue<Attempt>>>(`${this.base()}/attempts?${this.query('attempts')}`),
      this.api.request<Page<RecordValue<RunEvent>>>(`${this.base()}/events?${this.query('events')}`),
      this.api.request<Page<RecordValue<Artifact>>>(`${this.base()}/artifacts?${this.query('artifacts')}`)
    ])
    if (generation !== this.generation) return
    const [attempts, events, artifacts] = results
    this.attempts.set(attempts.status === 'fulfilled' ? attempts.value.items : [])
    this.events.set(events.status === 'fulfilled' ? events.value.items : [])
    this.artifacts.set(artifacts.status === 'fulfilled' ? artifacts.value.items : [])
    this.cursors.set({ attempts: attempts.status === 'fulfilled' ? attempts.value.cursor ?? '' : '', events: events.status === 'fulfilled' ? events.value.cursor ?? '' : '', artifacts: artifacts.status === 'fulfilled' ? artifacts.value.cursor ?? '' : '' })
    const failed = results.find(r => r.status === 'rejected')
    if (failed?.status === 'rejected') this.error.set(this.feedback(failed.reason, 'Some evidence could not be loaded. Retry Refresh evidence. Unavailable sections do not prove that no evidence exists.'))
    await this.loadLogs()
    if (generation === this.generation && results.every(result => result.status === 'fulfilled') && !this.logError()) this.restricted.set(false)
    if (generation === this.generation) this.loading.set(false)
  }

  async more(kind: EvidenceKind): Promise<void> {
    const cursor = this.cursors()[kind]
    if (!cursor || this.loading()) return
    const generation = this.generation
    this.loading.set(true); this.error.set('')
    try {
      const page = await this.api.request<Page<RecordValue<Attempt | RunEvent | Artifact>>>(`${this.base()}/${kind}?${this.query(kind, cursor)}`)
      if (generation !== this.generation) return
      if (kind === 'attempts') this.attempts.update(items => [...items, ...page.items as RecordValue<Attempt>[]])
      if (kind === 'events') this.events.update(items => [...items, ...page.items as RecordValue<RunEvent>[]])
      if (kind === 'artifacts') this.artifacts.update(items => [...items, ...page.items as RecordValue<Artifact>[]])
      this.cursors.update(values => ({ ...values, [kind]: page.cursor ?? '' }))
    } catch (error) {
      if (generation === this.generation) {
        if (error instanceof ApiError && (error.status === 403 || error.status === 404)) {
          if (kind === 'attempts') this.attempts.set([])
          if (kind === 'events') this.events.set([])
          if (kind === 'artifacts') this.artifacts.set([])
          this.cursors.update(values => ({ ...values, [kind]: '' }))
        }
        this.error.set(this.feedback(error, 'More evidence could not be loaded. Retry the same page or refresh evidence.'))
      }
    } finally { if (generation === this.generation) this.loading.set(false) }
  }

  async loadLogs(): Promise<void> {
    const generation = ++this.logGeneration
    this.logError.set(''); this.logLoading.set(true)
    try {
      const query = queryString({ stream: this.stream(), test: this.test(), attempt: this.focusedAttempt() ? String(this.focusedAttempt()) : '', tail: 'true', limit: '50' })
      const logs = await this.api.request<LogPage>(`${this.base()}/logs?${query}`)
      if (generation === this.logGeneration) this.logs.set(logs)
    } catch (error) {
      if (generation === this.logGeneration) { this.logs.set({ segments: [], truncated: false }); this.logError.set(this.feedback(error, 'Logs could not be loaded. Retry Refresh logs. This does not mean the test produced no logs.')) }
    } finally { if (generation === this.logGeneration) this.logLoading.set(false) }
  }

  private feedback(error: unknown, fallback: string): string {
    if (error instanceof ApiError && error.status === 403) { this.restricted.set(true); return 'Access to this evidence is no longer allowed. Previously loaded content from the rejected section is hidden. Check current project access before retrying.' }
    return `${fallback} ${errorMessage(error, '')}`.trim()
  }

  restrict(): void { this.invalidate(); this.clear(); this.restricted.set(true); this.loading.set(false); this.logLoading.set(false) }

  invalidate(): void { this.generation++; this.logGeneration++ }
}
