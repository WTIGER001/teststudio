import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type RecordValue } from '../../core/api.service'
import { CatalogService } from './catalog.service'
import type { CatalogPlan } from './catalog-operations.models'

@Injectable()
export class CatalogPlanService {
  private readonly api = inject(ApiService)
  readonly catalog = inject(CatalogService)
  readonly plan = signal<RecordValue<CatalogPlan> | null>(null)
  readonly saving = signal(false)
  readonly error = signal('')
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => this.clear())
    destroy.onDestroy(() => { this.generation++ })
  }

  clear(): void { this.generation++; this.plan.set(null); this.error.set(''); this.saving.set(false) }
  show(plan: RecordValue<CatalogPlan>): void { this.clear(); this.plan.set(plan) }

  async apply(): Promise<void> {
    const row = this.plan()
    if (!row || row.data.issues.length || this.saving()) return
    const generation = ++this.generation
    this.saving.set(true); this.error.set('')
    try {
      await this.api.request(`/api/v1/projects/${encodeURIComponent(row.data.projectId)}/suites/${encodeURIComponent(row.data.suiteId)}/catalog-imports/${encodeURIComponent(row.id)}/apply`, 'POST', {}, row.revision)
      if (generation !== this.generation) return
      this.plan.set(null); this.catalog.draft.set(null)
      this.catalog.success.set('Catalog changes applied with revision history. Publish a release to use new definitions. Lifecycle restrictions apply immediately.')
      await this.catalog.load()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The catalog plan could not be applied.') }
    finally { if (generation === this.generation) this.saving.set(false) }
  }
}
