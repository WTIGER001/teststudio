import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { RunDetailService } from './run-detail.service'
import type { WorkerExecution } from './run-workers.models'
import { pendingWorkers } from './run-workers.utils'
import { errorMessage } from './run.utils'

@Injectable()
export class RunWorkersService {
  private readonly api = inject(ApiService)
  private readonly destroy = inject(DestroyRef)
  readonly detail = inject(RunDetailService)
  readonly records = signal<RecordValue<WorkerExecution>[]>([])
  readonly pending = computed(() => pendingWorkers(this.records()))
  readonly loading = signal(false)
  readonly error = signal('')
  readonly cursor = signal('')
  readonly nextCursor = signal('')
  private readonly previous = signal<string[]>([])
  readonly canGoBack = computed(() => this.previous().length > 0)
  private base = ''
  private generation = 0
  private timer: ReturnType<typeof setTimeout> | undefined

  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.generation++; clearTimeout(this.timer)
      this.base = `/api/v1/projects/${encodeURIComponent(params.get('projectId') ?? '')}/runs/${encodeURIComponent(params.get('runId') ?? '')}/executions`
      this.records.set([]); this.cursor.set(''); this.nextCursor.set(''); this.previous.set([])
      void this.load()
    })
    this.destroy.onDestroy(() => { this.generation++; clearTimeout(this.timer) })
  }

  async load(): Promise<void> {
    clearTimeout(this.timer)
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    let forbidden = false
    try {
      const page = await this.api.request<Page<RecordValue<WorkerExecution>>>(`${this.base}?limit=50&cursor=${encodeURIComponent(this.cursor())}`)
      if (generation !== this.generation) return
      this.records.set(page.items); this.nextCursor.set(page.cursor ?? '')
    } catch (error) {
      if (generation === this.generation) this.error.set(errorMessage(error, 'Worker recovery status could not be loaded.'))
      forbidden = error instanceof ApiError && (error.status === 401 || error.status === 403 || error.status === 404)
    } finally {
      if (generation === this.generation) {
        this.loading.set(false)
        if (!forbidden && (this.pending() || !this.detail.finished() || this.error())) this.timer = setTimeout(() => { void this.load() }, 5000)
      }
    }
  }

  next(): void {
    if (!this.nextCursor() || this.loading()) return
    this.previous.update(previous => [...previous, this.cursor()]); this.cursor.set(this.nextCursor()); void this.load()
  }
  back(): void {
    if (!this.previous().length || this.loading()) return
    this.cursor.set(this.previous().at(-1) ?? ''); this.previous.update(previous => previous.slice(0, -1)); void this.load()
  }
}
