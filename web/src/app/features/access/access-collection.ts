import { signal } from '@angular/core'
import { ApiError } from '../../core/api.service'

export class AccessCollection<T> {
  readonly items = signal<T[]>([])
  readonly cursor = signal('')
  readonly partial = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  private generation = 0

  clear(): void { this.generation++; this.items.set([]); this.cursor.set(''); this.partial.set(false); this.loading.set(false); this.error.set('') }

  async load(fetch: (cursor: string) => Promise<{ items: T[]; cursor?: string; partial?: boolean }>, more = false): Promise<void> {
    if (more && (this.loading() || !this.cursor())) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    try {
      const page = await fetch(more ? this.cursor() : '')
      if (generation !== this.generation) return
      this.items.set(more ? [...this.items(), ...page.items] : page.items); this.cursor.set(page.cursor ?? ''); this.partial.set(page.partial === true || !!page.cursor)
    } catch (error) {
      if (generation !== this.generation) return
      this.error.set(error instanceof Error ? error.message : 'The collection could not be loaded.')
      if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.items.set([]); this.cursor.set(''); this.partial.set(false) }
    } finally { if (generation === this.generation) this.loading.set(false) }
  }
}
