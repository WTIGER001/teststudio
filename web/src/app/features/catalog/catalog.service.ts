import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { combineLatest } from 'rxjs'
import { applicationSuites } from '../../core/application-scope.utils'
import { ActivatedRoute, Router } from '@angular/router'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { AccessClientService } from '../../core/access-client.service'
import type { CatalogListPage } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import type { Application, Suite } from '../configuration/configuration.service'
import type { CatalogItem } from './catalog.models'
import { catalogFacetOptions, catalogTypeOptions, catalogListRequest, catalogListItem, mergeCatalogItems, emptyCatalogPage, categoryLabel, categoryInputValue, type CatalogFilters } from './catalog-browse.utils'
import { CatalogTestEditorService } from './catalog-test-editor.service'

@Injectable()
export class CatalogService {
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly testEditor = inject(CatalogTestEditorService)
  readonly timingEditor = this.testEditor.timingEditor
  readonly checkEditor = this.testEditor.checkEditor
  readonly probeOverview = this.route.snapshot.data['testType'] === 'https'
  readonly projectId = signal('')
  readonly applicationId = signal('')
  readonly items = signal<CatalogItem[]>([])
  readonly applicationNotice = signal('')
  readonly applications = signal<RecordValue<Application>[]>([])
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly hasSharedSuites = computed(() => !!this.applicationId() && this.suites().some(row => row.data.applications.length > 1))
  readonly loading = signal(true)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly success = signal('')
  readonly query = signal('')
  readonly state = signal('active')
  readonly suiteFilter = signal('')
  readonly typeFilter = signal('')
  readonly filters = signal<CatalogFilters>({})
  readonly view = signal<'table' | 'categories'>('table')
  readonly page = signal<CatalogListPage>(emptyCatalogPage())
  readonly visible = signal<CatalogItem[]>([])
  readonly loadingMore = signal(false)
  readonly selectionEpoch = signal(0)
  readonly categories = computed(() => this.page().categories)
  readonly categoryInput = computed(() => categoryInputValue(this.filters().category ?? ''))
  readonly categoryTitle = computed(() => categoryLabel(this.filters().category ?? ''))
  readonly options = computed(() => catalogFacetOptions(this.page().options, this.applications()))
  readonly types = computed(() => catalogTypeOptions(this.page().options.type))
  readonly counts = computed(() => this.page().counts)
  readonly editorOpen = this.testEditor.editorOpen
  readonly editing = this.testEditor.editing
  readonly editingHTTP = this.testEditor.editingHTTP
  readonly form = this.testEditor.form
  private loadTimer: ReturnType<typeof setTimeout> | undefined
  private applicationGeneration = 0
  private generation = 0

  constructor() {
    combineLatest([this.route.paramMap, this.route.queryParamMap]).pipe(takeUntilDestroyed(this.destroy)).subscribe(([params, query]) => {
      this.generation++; this.applicationId.set(params.get('applicationId') ?? '')
      this.projectId.set(params.get('projectId') ?? ''); this.suiteFilter.set(query.get('suite') ?? ''); this.typeFilter.set(this.probeOverview ? 'https' : query.get('type') ?? ''); this.query.set(''); this.state.set('active'); this.filters.set({}); this.view.set('table')
      this.editorOpen.set(false); this.editing.set(null); this.items.set([]); this.visible.set([]); this.page.set(emptyCatalogPage()); this.suites.set([]); this.applications.set([]); this.success.set(this.route.snapshot.queryParamMap.has('created') ? 'Test saved. Changes apply to the current version.' : '')
    })
    effect(() => {
      const project = this.projectId()
      this.query(); this.state(); this.suiteFilter(); this.typeFilter(); this.filters(); this.applicationId()
      untracked(() => {
        clearTimeout(this.loadTimer); this.generation++; this.loading.set(true); this.loadingMore.set(false); this.visible.set([])
        if (project) this.loadTimer = setTimeout(() => { void this.load() }, 120)
      })
    })
    effect(() => {
      const project = this.projectId(), allowed = this.workspace.projectId() === project && this.workspace.can('projects.read')
      untracked(() => {
        this.applicationGeneration++; this.applications.set([]); this.applicationNotice.set('')
        if (allowed) void this.loadApplications(project)
      })
    })
    effect(() => {
      const applications = this.workspace.applications(), application = this.applicationId(), project = this.projectId()
      if (!application && applications.length === 1 && project === this.workspace.projectId()) untracked(() => { void this.router.navigate(['/projects', project, 'applications', applications[0].id, 'catalog'], { queryParams: this.route.snapshot.queryParams, replaceUrl: true }) })
    })
    this.destroy.onDestroy(() => { this.generation++; this.applicationGeneration++; clearTimeout(this.loadTimer) })
  }

  canLeave(): boolean { return !(this.editorOpen() && this.testEditor.dirty() || this.saving()) || window.confirm('This catalog has unsaved edits or a save in progress. Leave this page?') }

  filter(field: keyof CatalogFilters, value: string | boolean): void { this.filters.update(filters => ({ ...filters, [field]: value })) }
  clearFilters(): void { this.query.set(''); this.state.set('active'); this.suiteFilter.set(''); this.typeFilter.set(this.probeOverview ? 'https' : ''); this.filters.set({}) }

  async load(more = false): Promise<void> {
    if (!this.applicationId() && !this.probeOverview) { this.loading.set(false); return }
    if (more && (this.loading() || this.loadingMore() || !this.page().cursor)) return
    clearTimeout(this.loadTimer)
    const generation = ++this.generation, project = this.projectId()
    const query = catalogListRequest(this.query(), this.state(), this.suiteFilter(), this.typeFilter(), { ...this.filters(), application: this.applicationId() }, more ? this.page().cursor : '')
    this.loading.set(!more); this.loadingMore.set(more); this.error.set('')
    try {
      const [page, suites] = await Promise.all([
        this.client.call('listApplicationCatalog', { project, application: this.applicationId(), query }),
        more ? Promise.resolve({ items: this.suites() }) : this.api.request<Page<RecordValue<Suite>>>(`/api/v1/projects/${project}/suites`)
      ])
      if (generation !== this.generation) return
      const incoming = page.items.map(catalogListItem)
      this.visible.set(more ? mergeCatalogItems(this.visible(), incoming) : incoming)
      this.items.set(mergeCatalogItems(this.items(), incoming)); this.page.set(page); this.suites.set(applicationSuites(suites.items, this.applicationId()))
    } catch (error) {
      if (generation !== this.generation) return
      this.visible.set([])
      if (error instanceof ApiError && [401, 403, 409].includes(error.status)) {
        this.items.set([]); this.page.set(emptyCatalogPage()); this.selectionEpoch.update(value => value + 1)
        if (error.status === 409) this.error.set('The catalog or your access changed between pages. Reload results and select tests again.')
        else this.error.set('Catalog access is unavailable. Refresh access before reloading results.')
      } else this.error.set(error instanceof Error ? error.message : 'The catalog could not be loaded.')
    } finally { if (generation === this.generation) { this.loading.set(false); this.loadingMore.set(false) } }
  }

  private async loadApplications(project: string): Promise<void> {
    const generation = this.applicationGeneration
    try {
      const page = await this.api.request<Page<RecordValue<Application>>>(`/api/v1/projects/${encodeURIComponent(project)}/applications`)
      if (generation === this.applicationGeneration && project === this.projectId()) this.applications.set(page.items)
    } catch {
      if (generation === this.applicationGeneration) this.applicationNotice.set('Application names are unavailable. You can still filter by identifier.')
    }
  }

  edit(item: CatalogItem): void {
    this.testEditor.configure(this.projectId(), this.suites()); this.testEditor.edit(item); this.error.set(''); this.success.set('')
  }

  async afterEdit(): Promise<void> {
    this.success.set('Test saved. Changes apply to the current version.')
    await this.load()
  }

  async action(item: CatalogItem, action: 'disable' | 'enable' | 'archive' | 'restore'): Promise<void> {
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      await this.api.request(`/api/v1/projects/${this.projectId()}/suites/${item.suiteId}/tests/${item.id}/actions/${action}?save=true`, 'POST', {}, item.revision)
      this.success.set('Test state updated. Retained versions and history remain available.')
      await this.load()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'The test state could not be changed.') }
    finally { this.saving.set(false) }
  }

}
