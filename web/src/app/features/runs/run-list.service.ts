import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunLookupsService } from './run-lookups.service'
import type { RunSummary } from './run.models'
import { errorMessage, queryString } from './run.utils'

@Injectable()
export class RunListService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly lookups = inject(RunLookupsService)
  readonly projectId = signal('')
  readonly items = signal<RecordValue<RunSummary>[]>([])
  readonly cursor = signal('')
  readonly partial = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly newRunOpen = signal(false)
  readonly filters = new FormGroup({
    environment: new FormControl('', { nonNullable: true }), suite: new FormControl('', { nonNullable: true }),
    state: new FormControl('', { nonNullable: true }), selection: new FormControl('', { nonNullable: true })
  })
  private generation = 0
  private activeFilters = ''

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? '')
      this.filters.reset(); this.items.set([]); this.newRunOpen.set(false)
      void this.initialize()
    })
    this.destroy.onDestroy(() => { this.generation++ })
  }

  private async initialize(): Promise<void> {
    const project = this.projectId()
    try { await this.lookups.load(project) }
    catch (error) { if (project === this.projectId()) this.error.set(errorMessage(error, 'Run options could not be loaded.')) }
    if (project === this.projectId()) await this.load()
  }

  async load(more = false): Promise<void> {
    if (more && (!this.cursor() || this.loading())) return
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    if (!more) this.activeFilters = queryString(this.filters.getRawValue())
    const position = more ? `&cursor=${encodeURIComponent(this.cursor())}` : ''
    try {
      const page = await this.api.request<Page<RecordValue<RunSummary>> & { partial: boolean }>(`/api/v1/projects/${encodeURIComponent(this.projectId())}/runs?limit=30&${this.activeFilters}${position}`)
      if (generation !== this.generation) return
      this.items.set(more ? [...this.items(), ...page.items] : page.items)
      this.cursor.set(page.cursor ?? ''); this.partial.set(page.partial)
    } catch (error) { if (generation === this.generation) this.error.set(errorMessage(error, 'Runs could not be loaded.')) }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
}
