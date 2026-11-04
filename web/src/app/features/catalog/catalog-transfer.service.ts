import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { CatalogService } from './catalog.service'
import { CatalogPlanService } from './catalog-plan.service'
import type { CatalogDocument, CatalogPlan } from './catalog-operations.models'
import { parseCatalogDocument } from './catalog-operations.utils'

@Injectable()
export class CatalogTransferService {
  private readonly api = inject(ApiService)
  private readonly plans = inject(CatalogPlanService)
  readonly catalog = inject(CatalogService)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly downloadURL = signal('')
  readonly form = new FormGroup({ suite: new FormControl('', { nonNullable: true, validators: Validators.required }), document: new FormControl('', { nonNullable: true }) })
  private generation = 0

  constructor() {
    const destroy = inject(DestroyRef)
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.generation++; this.plans.clear(); this.clearDownload(); this.loading.set(false) })
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(destroy)).subscribe(() => { this.form.reset(); this.error.set('') })
    destroy.onDestroy(() => { this.generation++; this.clearDownload() })
  }

  private clearDownload(): void { if (this.downloadURL()) URL.revokeObjectURL(this.downloadURL()); this.downloadURL.set('') }

  async file(file: File | undefined): Promise<void> {
    if (!file) return
    this.error.set('')
    if (file.size > 2 << 20) { this.error.set('The catalog file exceeds the 2 MiB limit.'); return }
    const generation = ++this.generation
    this.loading.set(true); this.plans.clear(); this.clearDownload()
    try { const text = await file.text(); if (generation === this.generation) this.form.controls.document.setValue(text) }
    catch { if (generation === this.generation) this.error.set('The catalog file could not be read.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async export(): Promise<void> {
    if (this.form.controls.suite.invalid || this.loading()) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set(''); this.clearDownload()
    try {
      const document = await this.api.request<CatalogDocument>(`${this.base()}/catalog-exports`)
      if (generation !== this.generation) return
      const text = JSON.stringify(document, null, 2)
      this.form.controls.document.setValue(text, { emitEvent: false })
      this.clearDownload(); this.downloadURL.set(URL.createObjectURL(new Blob([text], { type: 'application/json' })))
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'Export failed.'); if (error instanceof ApiError && [401, 403].includes(error.status)) { this.form.controls.document.setValue('', { emitEvent: false }); this.plans.clear() } } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  async preview(): Promise<void> {
    if (this.form.controls.suite.invalid || this.loading()) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set(''); this.plans.clear()
    try {
      const document = parseCatalogDocument(this.form.controls.document.value)
      const plan = await this.api.request<RecordValue<CatalogPlan>>(`${this.base()}/catalog-imports`, 'POST', document)
      if (generation === this.generation) this.plans.show(plan)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Import validation failed.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }

  private base(): string { return `/api/v1/projects/${encodeURIComponent(this.catalog.projectId())}/suites/${encodeURIComponent(this.form.controls.suite.value)}` }
}
