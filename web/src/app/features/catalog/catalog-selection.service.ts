import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type RecordValue } from '../../core/api.service'
import { CatalogService } from './catalog.service'
import { CatalogPlanService } from './catalog-plan.service'
import { CatalogBulkMetadataService } from './catalog-bulk-metadata.service'
import type { CatalogItem } from './catalog.models'
import type { CatalogPlan } from './catalog-operations.models'
import { bulkInput, catalogKey, selectedItems } from './catalog-operations.utils'

@Injectable()
export class CatalogSelectionService {
  private readonly api = inject(ApiService)
  private readonly plans = inject(CatalogPlanService)
  readonly catalog = inject(CatalogService)
  readonly metadata = inject(CatalogBulkMetadataService)
  readonly keys = signal<string[]>([])
  readonly items = computed(() => selectedItems(this.catalog.items(), this.keys()))
  readonly canSelect = computed(() => this.catalog.workspace.can('catalog.update') || this.catalog.workspace.can('catalog.archive') || this.catalog.workspace.can('catalog.restore'))
  readonly action = signal('')
  readonly loading = signal(false)
  readonly error = signal('')
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.invalidate(); this.keys.set([]); this.action.set(''); this.metadata.reset() })
    destroy.onDestroy(() => { this.generation++ })
  }

  contains(item: CatalogItem): boolean { return this.keys().includes(catalogKey(item)) }
  toggle(item: CatalogItem): void { this.invalidate(); const key = catalogKey(item); this.keys.update(keys => keys.includes(key) ? keys.filter(value => value !== key) : [...keys, key]) }
  selectVisible(): void { this.invalidate(); this.keys.set(this.catalog.visible().map(catalogKey)) }
  clear(): void { this.invalidate(); this.keys.set([]) }
  selectAction(action: string): void { this.invalidate(); this.action.set(action) }
  private invalidate(): void { this.generation++; this.plans.clear(); this.error.set(''); this.loading.set(false) }

  async preview(): Promise<void> {
    if (this.loading()) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set(''); this.plans.clear()
    try {
      const edits = this.metadata.edits(), action = this.action()
      const { suite, input } = bulkInput(this.items(), action, action === 'metadata' ? this.metadata.input() : undefined)
      const plan = await this.api.request<RecordValue<CatalogPlan>>(`/api/v1/projects/${encodeURIComponent(this.catalog.projectId())}/suites/${encodeURIComponent(suite)}/tests/bulk-preview`, 'POST', input)
      if (generation === this.generation && (action !== 'metadata' || edits === this.metadata.edits())) this.plans.show(plan)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Bulk preview failed.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
}
