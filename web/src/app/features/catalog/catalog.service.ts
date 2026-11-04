import { randomId } from '../../core/random-id'
import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiError, ApiService, type Page, type RecordValue } from '../../core/api.service'
import { AccessClientService } from '../../core/access-client.service'
import type { CatalogListPage, PublicationReview } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import type { Application, Suite } from '../configuration/configuration.service'
import type { CatalogItem, Draft, Issue } from './catalog.models'
import { catalogFacetOptions, catalogTypeOptions, catalogListRequest, catalogListItem, mergeCatalogItems, emptyCatalogPage, categoryLabel, categoryInputValue, type CatalogFilters } from './catalog-browse.utils'
import { CatalogTestEditorService } from './catalog-test-editor.service'

@Injectable()
export class CatalogService {
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly testEditor = inject(CatalogTestEditorService)
  readonly timingEditor = this.testEditor.timingEditor
  readonly checkEditor = this.testEditor.checkEditor
  readonly projectId = signal('')
  readonly items = signal<CatalogItem[]>([])
  readonly applicationNotice = signal('')
  readonly applications = signal<RecordValue<Application>[]>([])
  readonly suites = signal<RecordValue<Suite>[]>([])
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
  readonly releaseSuite = signal('')
  readonly draft = signal<RecordValue<Draft> | null>(null)
  readonly review = signal<PublicationReview | null>(null)
  readonly publicationConflict = signal(false)
  readonly publicationUncertain = signal(false)
  readonly publicationAttempt = signal<boolean | null>(null)
  readonly publicationDenied = signal(false)
  readonly reviewStale = computed(() => this.publicationConflict() || !!this.review() && !this.draft() && !this.publicationUncertain())
  readonly issues = signal<Issue[]>([])
  readonly form = this.testEditor.form
  private loadTimer: ReturnType<typeof setTimeout> | undefined
  private applicationGeneration = 0
  private generation = 0
  private publicationGeneration = 0
  private publicationKey = ''

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? ''); this.suiteFilter.set(this.route.snapshot.queryParamMap.get('suite') ?? ''); this.typeFilter.set(''); this.query.set(''); this.state.set('active'); this.filters.set({}); this.view.set('table'); this.review.set(null); this.publicationConflict.set(false); this.publicationUncertain.set(false); this.publicationAttempt.set(null); this.publicationDenied.set(false); this.publicationGeneration++; this.releaseSuite.set(''); this.draft.set(null)
      this.editorOpen.set(false); this.editing.set(null); this.items.set([]); this.visible.set([]); this.page.set(emptyCatalogPage()); this.suites.set([]); this.applications.set([]); this.success.set(this.route.snapshot.queryParamMap.has('created') ? 'Draft changes saved. Publish a release to use the updated definition.' : '')
    })
    effect(() => {
      const project = this.projectId()
      this.query(); this.state(); this.suiteFilter(); this.typeFilter(); this.filters()
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
    this.destroy.onDestroy(() => { this.generation++; this.applicationGeneration++; this.publicationGeneration++; clearTimeout(this.loadTimer) })
  }

  filter(field: keyof CatalogFilters, value: string | boolean): void { this.filters.update(filters => ({ ...filters, [field]: value })) }
  clearFilters(): void { this.query.set(''); this.state.set('active'); this.suiteFilter.set(''); this.typeFilter.set(''); this.filters.set({}) }

  async load(more = false): Promise<void> {
    if (more && (this.loading() || this.loadingMore() || !this.page().cursor)) return
    clearTimeout(this.loadTimer)
    const generation = ++this.generation, project = this.projectId()
    const query = catalogListRequest(this.query(), this.state(), this.suiteFilter(), this.typeFilter(), this.filters(), more ? this.page().cursor : '')
    this.loading.set(!more); this.loadingMore.set(more); this.error.set('')
    try {
      const [page, suites] = await Promise.all([
        this.client.call('listProjectCatalog', { project, query }),
        more ? Promise.resolve({ items: this.suites() }) : this.api.request<Page<RecordValue<Suite>>>(`/api/v1/projects/${project}/suites`)
      ])
      if (generation !== this.generation) return
      const incoming = page.items.map(catalogListItem)
      this.visible.set(more ? mergeCatalogItems(this.visible(), incoming) : incoming)
      this.items.set(mergeCatalogItems(this.items(), incoming)); this.page.set(page); this.suites.set(suites.items)
      if (!this.releaseSuite()) this.releaseSuite.set(suites.items[0]?.id ?? '')
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
    this.success.set('Draft changes saved. Publish a release to use the updated definition.'); this.draft.set(null)
    await this.load()
  }

  async action(item: CatalogItem, action: 'disable' | 'enable' | 'archive' | 'restore'): Promise<void> {
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      await this.api.request(`/api/v1/projects/${this.projectId()}/suites/${item.suiteId}/tests/${item.id}/actions/${action}`, 'POST', {}, item.revision)
      this.success.set('Test state updated. Retained releases and history remain available.')
      this.draft.set(null)
      await this.load()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'The test state could not be changed.') }
    finally { this.saving.set(false) }
  }

  changeReleaseSuite(suite: string): void {
    if (this.publicationUncertain()) return
    this.publicationGeneration++; this.releaseSuite.set(suite); this.draft.set(null); this.review.set(null); this.publicationConflict.set(false); this.publicationAttempt.set(null)
  }

  async prepareRelease(): Promise<void> {
    if (!this.releaseSuite() || this.saving() || this.publicationUncertain()) return
    const generation = ++this.publicationGeneration, project = this.projectId(), suite = this.releaseSuite()
    this.saving.set(true); this.error.set(''); this.success.set(''); this.review.set(null); this.draft.set(null); this.publicationConflict.set(false); this.publicationDenied.set(false); this.publicationAttempt.set(null)
    try {
      const draft = await this.api.request<RecordValue<Draft>>(`/api/v1/projects/${project}/suites/${suite}/catalog-drafts`, 'POST', {})
      const review = await this.client.call('reviewCatalogPublication', { project, suite, id: draft.id, revision: draft.revision })
      if (generation !== this.publicationGeneration) return
      this.draft.set(draft); this.review.set(review); this.issues.set(review.issues); this.publicationKey = randomId()
    } catch (error) { if (generation === this.publicationGeneration) this.error.set(error instanceof Error ? error.message : 'The release preview could not be prepared.') }
    finally { if (generation === this.publicationGeneration) this.saving.set(false) }
  }

  async publish(activate = true): Promise<void> {
    const review = this.review(), generation = this.publicationGeneration
    if (!review || review.issues.length || this.saving() || this.reviewStale() || this.publicationAttempt() !== null && this.publicationAttempt() !== activate) return
    this.saving.set(true); this.error.set(''); this.success.set(''); this.publicationAttempt.set(activate)
    try {
      await this.client.call('publishReviewedCatalog', { project: review.projectId, suite: review.suiteId, id: review.draftId, revision: review.draftRevision, key: this.publicationKey, body: { activate, reviewId: review.id } })
      if (generation !== this.publicationGeneration) return
      this.draft.set(null); this.review.set(null); this.publicationUncertain.set(false); this.publicationAttempt.set(null)
      this.success.set(activate ? 'Release published and activated. Existing run snapshots retain their original definitions.' : 'Release published. The active release stays unchanged. Open Suite releases to activate it later.')
      await this.load()
    } catch (error) {
      if (generation !== this.publicationGeneration) return
      if (error instanceof ApiError && error.status < 500) {
        this.publicationUncertain.set(false); this.publicationConflict.set(true)
        if (error.status === 403 || error.status === 401) { this.review.set(null); this.draft.set(null); this.publicationDenied.set(true); this.error.set('Publication access is unavailable. Refresh access and prepare a new review.') }
        else this.error.set('The review is outdated or no longer executable. Validate and preview again before publishing.')
      } else {
        this.publicationUncertain.set(true)
        this.error.set('The publication response is unavailable. Retry the same publication action below. This page retains its request key to recover an existing release. Reloading loses this continuation; inspect Suite releases before publishing again.')
      }
    } finally { if (generation === this.publicationGeneration) this.saving.set(false) }
  }
}
