import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { combineLatest } from 'rxjs'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { errorMessage } from '../runs/run.utils'
import type { Delivery, DeliveryPage, NotificationAttempt } from './notification.models'
import { deliveryRows } from './notification.utils'

@Injectable()
export class NotificationDeliveriesService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly projectId = signal('')
  readonly runId = signal('')
  readonly items = signal<RecordValue<Delivery>[]>([])
  readonly now = signal(Date.now())
  readonly rows = computed(() => deliveryRows(this.items(), this.now()))
  readonly cursor = signal('')
  readonly partial = signal(false)
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly selected = signal<RecordValue<Delivery> | null>(null)
  readonly attempts = signal<RecordValue<NotificationAttempt>[]>([])
  readonly attemptsCursor = signal('')
  readonly attemptsLoading = signal(false)
  readonly attemptsError = signal('')
  private generation = 0
  private historyGeneration = 0
  private paged = false
  private historyPaged = false

  constructor() {
    combineLatest([this.route.paramMap, this.route.queryParamMap]).pipe(takeUntilDestroyed(this.destroy)).subscribe(([params, query]) => {
      const project = params.get('projectId') ?? '', run = params.get('runId') ?? query.get('run') ?? ''
      if (this.projectId() === project && this.runId() === run) return
      this.projectId.set(project); this.runId.set(run); this.items.set([]); this.cursor.set(''); this.message.set(''); this.closeHistory(); void this.load()
    })
    const poll = setInterval(() => {
      this.now.set(Date.now())
      if (!this.loading() && !this.busy() && !this.paged) void this.load(false, true)
      if (this.selected() && !this.attemptsLoading() && !this.historyPaged) void this.loadHistory()
    }, 5000)
    this.destroy.onDestroy(() => { clearInterval(poll); this.generation++; this.historyGeneration++ })
  }
  base(): string { return `/api/v1/projects/${encodeURIComponent(this.projectId())}/notifications/deliveries` }
  async load(more = false, quiet = false): Promise<void> {
    if (more && (!this.cursor() || this.loading())) return
    const generation = ++this.generation
    this.loading.set(true); if (!quiet) { this.error.set(''); if (!more) this.paged = false }
    const query = new URLSearchParams({ limit: '30' })
    if (this.runId()) query.set('run', this.runId())
    if (more) query.set('cursor', this.cursor())
    try {
      const page = await this.api.request<DeliveryPage>(`${this.base()}?${query}`)
      if (generation !== this.generation) return
      this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? ''); this.partial.set(page.partial); this.error.set('')
      if (more) this.paged = true
      const selected = this.selected(), updated = page.items.find(item => item.id === selected?.id)
      if (updated) this.selected.set(updated)
    } catch (error) {
      if (generation !== this.generation) return
      this.error.set(errorMessage(error, 'Delivery history could not be loaded.'))
      if (error instanceof ApiError && [401, 403].includes(error.status)) { this.items.set([]); this.cursor.set(''); this.closeHistory(); this.paged = true }
    } finally { if (generation === this.generation) this.loading.set(false) }
  }
  async action(item: RecordValue<Delivery>, action: 'retry' | 'cancel'): Promise<void> {
    if (this.busy()) return
    const base = this.base()
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const updated = await this.api.request<RecordValue<Delivery>>(`${base}/${encodeURIComponent(item.id)}/${action}`, 'POST', {}, item.revision)
      if (base !== this.base()) return
      if (this.selected()?.id === item.id) this.selected.set(updated)
      this.message.set(action === 'retry' ? 'Delivery retry requested. Test results remain unchanged.' : 'Delivery retries stopped. A message already sent cannot be recalled.')
      await this.load(); if (this.selected()) await this.loadHistory(); this.focus('notification-deliveries-heading')
    } catch (error) { if (base === this.base()) this.error.set(errorMessage(error, 'The delivery action failed. Refresh and check the current channel before retrying.')) }
    finally { this.busy.set(false) }
  }
  openHistory(item: RecordValue<Delivery>): void { this.closeHistory(); this.selected.set(item); this.focus('notification-attempts-heading'); void this.loadHistory() }
  closeHistory(): void { this.historyGeneration++; this.selected.set(null); this.attempts.set([]); this.attemptsCursor.set(''); this.attemptsError.set(''); this.attemptsLoading.set(false); this.historyPaged = false }
  async loadHistory(more = false): Promise<void> {
    const selected = this.selected(); if (!selected || (more && (!this.attemptsCursor() || this.attemptsLoading()))) return
    const generation = ++this.historyGeneration
    if (!more) this.historyPaged = false
    this.attemptsLoading.set(true); this.attemptsError.set('')
    try {
      const base = `${this.base()}/${encodeURIComponent(selected.id)}`
      const [row, page] = await Promise.all([
        this.api.request<RecordValue<Delivery>>(base),
        this.api.request<Page<RecordValue<NotificationAttempt>>>(`${base}/attempts?limit=30${more ? `&cursor=${encodeURIComponent(this.attemptsCursor())}` : ''}`)
      ])
      if (generation !== this.historyGeneration) return
      this.selected.set(row); this.attempts.set(more ? [...this.attempts(), ...page.items] : page.items); this.attemptsCursor.set(page.cursor ?? '')
      if (more) this.historyPaged = true
    } catch (error) {
      if (generation !== this.historyGeneration) return
      this.attemptsError.set(errorMessage(error, 'Delivery attempts could not be loaded.'))
      if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.attempts.set([]); this.selected.set(null) }
    } finally { if (generation === this.historyGeneration) this.attemptsLoading.set(false) }
  }
  focus(id: string): void { setTimeout(() => document.getElementById(id)?.focus(), 0) }
}
