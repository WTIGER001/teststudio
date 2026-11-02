import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { errorMessage } from '../runs/run.utils'
import type { Firing, Schedule } from './schedule.models'
import { scheduleDefinition, scheduleRows } from './schedule.utils'

@Injectable()
export class SchedulesService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly projectId = signal('')
  readonly items = signal<RecordValue<Schedule>[]>([])
  readonly rows = computed(() => scheduleRows(this.items()))
  readonly cursor = signal('')
  readonly archived = signal(false)
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly history = signal<RecordValue<Schedule> | null>(null)
  readonly firings = signal<RecordValue<Firing>[]>([])
  readonly historyCursor = signal('')
  readonly historyLoading = signal(false)
  readonly historyError = signal('')
  private generation = 0
  private historyGeneration = 0
  private paged = false
  private historyPaged = false

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? ''); this.items.set([]); this.closeHistory(); this.message.set(''); void this.load()
    })
    const poll = setInterval(() => {
      if (!this.loading() && !this.busy() && !this.paged) void this.load(false, true)
      if (this.history() && !this.historyLoading() && !this.historyPaged) void this.loadHistory()
    }, 5000)
    this.destroy.onDestroy(() => { clearInterval(poll); this.generation++; this.historyGeneration++ })
  }
  base(): string { return `/api/v1/projects/${encodeURIComponent(this.projectId())}/schedules` }
  async load(more = false, quiet = false): Promise<void> {
    if (more && (!this.cursor() || this.loading())) return
    const generation = ++this.generation
    if (!quiet) this.loading.set(true)
    this.error.set(''); if (!more && !quiet) this.paged = false
    try {
      const page = await this.api.request<Page<RecordValue<Schedule>>>(`${this.base()}?limit=30&archived=${this.archived()}${more ? `&cursor=${encodeURIComponent(this.cursor())}` : ''}`)
      if (generation !== this.generation) return
      this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? '')
      if (more) this.paged = true
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Schedules could not be loaded.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async action(item: RecordValue<Schedule>, action: 'pause' | 'enable' | 'archive' | 'restore'): Promise<void> {
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      if (action === 'pause' || action === 'enable') {
        await this.api.request(`${this.base()}/${encodeURIComponent(item.id)}`, 'PUT', { ...scheduleDefinition(item.data), enabled: action === 'enable' }, item.revision)
      } else await this.api.request(`${this.base()}/${encodeURIComponent(item.id)}/${action}`, 'POST', {}, item.revision)
      this.message.set(action === 'restore' ? 'Schedule restored and paused.' : action === 'archive' ? 'Schedule archived. Firing history remains available.' : action === 'pause' ? 'Schedule paused.' : 'Schedule enabled.')
      await this.load(); this.focus('schedules-heading')
    } catch (error) { this.error.set(errorMessage(error, 'The schedule action failed. Reload before retrying a revision conflict.')) }
    finally { this.busy.set(false) }
  }
  openHistory(item: RecordValue<Schedule>): void { this.closeHistory(); this.history.set(item); this.focus('schedule-history-heading'); void this.loadHistory() }
  closeHistory(): void { this.historyGeneration++; this.history.set(null); this.firings.set([]); this.historyCursor.set(''); this.historyError.set(''); this.historyLoading.set(false); this.historyPaged = false }
  async loadHistory(more = false): Promise<void> {
    const selected = this.history(); if (!selected || (more && !this.historyCursor())) return
    const generation = ++this.historyGeneration; this.historyLoading.set(true); this.historyError.set('')
    try {
      const page = await this.api.request<Page<RecordValue<Firing>>>(`${this.base()}/${encodeURIComponent(selected.id)}/firings?limit=30${more ? `&cursor=${encodeURIComponent(this.historyCursor())}` : ''}`)
      if (generation !== this.historyGeneration) return
      this.firings.set(more ? [...this.firings(), ...page.items] : page.items); this.historyCursor.set(page.cursor ?? ''); if (more) this.historyPaged = true
    } catch (error) { if (generation === this.historyGeneration) this.historyError.set(errorMessage(error, 'Firing history could not be loaded.')) }
    finally { if (generation === this.historyGeneration) this.historyLoading.set(false) }
  }
  focus(id: string): void { setTimeout(() => document.getElementById(id)?.focus(), 0) }
}
