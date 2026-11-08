import { isPlatformBrowser } from '@angular/common'
import { DestroyRef, inject, Injectable, PLATFORM_ID, signal } from '@angular/core'
import { from, Subscription } from 'rxjs'
import { TestStudioStatusClient } from './status-client.service'
import type { StatusCheck, StatusCheckLoader, StatusCheckQuery, StatusSource } from './models'

@Injectable()
export class StatusChecksService {
  private readonly client = inject(TestStudioStatusClient)
  private readonly browser = isPlatformBrowser(inject(PLATFORM_ID))
  private request?: Subscription
  private generation = 0
  private query: StatusCheckQuery | null = null
  private source: StatusSource | null = null
  private loader: StatusCheckLoader | null = null
  readonly items = signal<StatusCheck[]>([])
  readonly cursor = signal('')
  readonly partial = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.request?.unsubscribe() }) }
  configure(query: StatusCheckQuery, source: StatusSource | null, loader: StatusCheckLoader | null): void {
    this.query = query; this.source = source; this.loader = loader
    this.load()
  }
  load(more = false): void {
    this.request?.unsubscribe()
    const generation = ++this.generation
    if (!this.query || !this.browser) return
    this.error.set(''); this.loading.set(true)
    if (!more) { this.items.set([]); this.cursor.set(''); this.partial.set(false) }
    const query = { ...this.query, cursor: more ? this.cursor() : undefined }
    if (Date.parse(query.since) >= Date.parse(query.until)) { this.items.set([]); this.cursor.set(''); this.loading.set(false); return }
    try {
      const request = this.loader ? from(this.loader(query)) : this.client.checks(this.source!, query)
      this.request = request.subscribe({
        next: page => {
          if (generation !== this.generation) return
          this.items.set(more ? [...this.items(), ...page.items] : page.items)
          this.cursor.set(page.cursor || ''); this.partial.set(page.partial); this.loading.set(false)
        },
        error: () => {
          if (generation !== this.generation) return
          this.items.set([]); this.cursor.set(''); this.loading.set(false)
          this.error.set('Individual checks are unavailable. Refresh the report or retry. Published access may have been withdrawn.')
        }
      })
    } catch {
      this.items.set([]); this.cursor.set(''); this.loading.set(false); this.error.set('Individual checks are unavailable. Refresh the report or retry.')
    }
  }
}
