import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import { CatalogService } from './catalog.service'
import type { CatalogItem } from './catalog.models'
import type { TestRevision, CatalogRevision } from './catalog-operations.models'
import { newestRevisions } from './catalog-operations.utils'

@Injectable()
export class CatalogEntryService {
  private readonly api = inject(ApiService)
  readonly catalog = inject(CatalogService)
  readonly entry = signal<CatalogItem | null>(null)
  readonly mode = signal<'history' | 'duplicate'>('history')
  readonly revisions = signal<CatalogRevision[]>([])
  readonly loading = signal(false)
  readonly error = signal('')
  readonly newID = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] })
  readonly form = new FormGroup({ id: this.newID })
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => this.close())
    destroy.onDestroy(() => { this.generation++ })
  }

  close(): void { this.generation++; this.entry.set(null); this.revisions.set([]); this.error.set(''); this.loading.set(false) }

  async open(item: CatalogItem, mode: 'history' | 'duplicate'): Promise<void> {
    this.close(); this.entry.set(item); this.mode.set(mode); this.newID.setValue('')
    if (mode !== 'history') return
    const generation = ++this.generation
    this.loading.set(true)
    try {
      const page = await this.api.request<Page<RecordValue<TestRevision>>>(`${this.base(item)}/revisions`)
      if (generation === this.generation) this.revisions.set(newestRevisions(page.items))
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Revision history could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async duplicate(): Promise<void> {
    const item = this.entry()
    if (!item || this.newID.invalid || this.loading()) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    try {
      const id = this.newID.value
      await this.api.request(`${this.base(item)}/actions/duplicate`, 'POST', { id }, item.revision)
      if (generation !== this.generation) return
      this.close(); this.catalog.success.set(`Created ${id} as a new test identity with independent history.`)
      await this.catalog.load()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The test could not be duplicated.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  private base(item: CatalogItem): string { return `/api/v1/projects/${encodeURIComponent(this.catalog.projectId())}/suites/${encodeURIComponent(item.suiteId)}/tests/${encodeURIComponent(item.id)}` }
}
