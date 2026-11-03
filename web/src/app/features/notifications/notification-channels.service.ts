import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { errorMessage } from '../runs/run.utils'
import type { Channel } from './notification.models'
import { channelDefinition } from './notification.utils'

@Injectable()
export class NotificationChannelsService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly projectId = signal('')
  readonly items = signal<RecordValue<Channel>[]>([])
  readonly cursor = signal('')
  readonly archived = signal(false)
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  private generation = 0
  private paged = false

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? ''); this.items.set([]); this.cursor.set(''); this.message.set(''); void this.load()
    })
    const poll = setInterval(() => { if (!this.loading() && !this.busy() && !this.paged) void this.load(false, true) }, 5000)
    this.destroy.onDestroy(() => { clearInterval(poll); this.generation++ })
  }
  base(): string { return `/api/v1/projects/${encodeURIComponent(this.projectId())}/notifications` }
  async load(more = false, quiet = false): Promise<void> {
    if (more && (!this.cursor() || this.loading())) return
    const generation = ++this.generation
    this.loading.set(true); if (!quiet) { this.error.set(''); if (!more) this.paged = false }
    try {
      const page = await this.api.request<Page<RecordValue<Channel>>>(`${this.base()}/channels?limit=30&archived=${this.archived()}${more ? `&cursor=${encodeURIComponent(this.cursor())}` : ''}`)
      if (generation !== this.generation) return
      this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? ''); this.error.set('')
      if (more) this.paged = true
    } catch (error) {
      if (generation !== this.generation) return
      this.error.set(errorMessage(error, 'Notification channels could not be loaded.'))
      if (error instanceof ApiError && [401, 403].includes(error.status)) { this.items.set([]); this.cursor.set(''); this.paged = true }
    } finally { if (generation === this.generation) this.loading.set(false) }
  }
  async action(item: RecordValue<Channel>, action: 'pause' | 'enable' | 'archive' | 'restore'): Promise<void> {
    if (this.busy()) return
    const project = this.projectId(), base = this.base()
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      if (action === 'pause' || action === 'enable') await this.api.request(`${base}/channels/${encodeURIComponent(item.id)}`, 'PUT', { ...channelDefinition(item.data), enabled: action === 'enable' }, item.revision)
      else await this.api.request(`${base}/channels/${encodeURIComponent(item.id)}/${action}`, 'POST', {}, item.revision)
      if (project !== this.projectId()) return
      this.message.set(action === 'restore' ? 'Channel restored and paused.' : action === 'archive' ? 'Channel archived. Delivery history remains available.' : action === 'pause' ? 'Channel paused. Earlier queued messages require explicit retry after reenabling.' : 'Channel enabled for future runs.')
      await this.load(); this.focus('notification-channels-heading')
    } catch (error) { if (project === this.projectId()) this.error.set(errorMessage(error, 'The channel action failed. Reload before retrying a revision conflict.')) }
    finally { this.busy.set(false) }
  }
  focus(id: string): void { setTimeout(() => document.getElementById(id)?.focus(), 0) }
}
