import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import type { Suite } from '../configuration/configuration.service'
import type { CatalogItem, Draft, Issue } from './catalog.models'
import { catalogCounts, catalogTypes, filterCatalog } from './catalog.utils'
import { CatalogTestEditorService } from './catalog-test-editor.service'

@Injectable()
export class CatalogService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly testEditor = inject(CatalogTestEditorService)
  readonly timingEditor = this.testEditor.timingEditor
  readonly checkEditor = this.testEditor.checkEditor
  readonly projectId = signal('')
  readonly items = signal<CatalogItem[]>([])
  readonly suites = signal<RecordValue<Suite>[]>([])
  readonly loading = signal(true)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly success = signal('')
  readonly query = signal('')
  readonly state = signal('active')
  readonly suiteFilter = signal('')
  readonly typeFilter = signal('')
  readonly types = computed(() => catalogTypes(this.items()))
  readonly visible = computed(() => filterCatalog(this.items(), this.query(), this.state(), this.suiteFilter(), this.typeFilter()))
  readonly counts = computed(() => catalogCounts(this.items()))
  readonly editorOpen = this.testEditor.editorOpen
  readonly editing = this.testEditor.editing
  readonly editingHTTP = this.testEditor.editingHTTP
  readonly releaseSuite = signal('')
  readonly draft = signal<RecordValue<Draft> | null>(null)
  readonly issues = signal<Issue[]>([])
  readonly form = this.testEditor.form
  private generation = 0
  private publicationKey = ''

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? ''); this.suiteFilter.set(''); this.typeFilter.set(''); this.releaseSuite.set(''); this.draft.set(null)
      this.editorOpen.set(false); this.editing.set(null); this.items.set([]); this.suites.set([]); this.success.set(this.route.snapshot.queryParamMap.has('created') ? 'Draft changes saved. Publish a release to use the updated definition.' : '')
      void this.load()
    })
  }

  async load(): Promise<void> {
    const generation = ++this.generation
    this.loading.set(true); this.error.set('')
    try {
      const base = `/api/v1/projects/${this.projectId()}`
      const [tests, suites] = await Promise.all([
        this.api.request<Page<CatalogItem>>(`${base}/tests`),
        this.api.request<Page<RecordValue<Suite>>>(`${base}/suites`)
      ])
      if (generation !== this.generation) return
      this.items.set(tests.items); this.suites.set(suites.items)
      if (!this.releaseSuite()) this.releaseSuite.set(suites.items[0]?.id ?? '')
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The catalog could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
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

  async prepareRelease(): Promise<void> {
    if (!this.releaseSuite() || this.saving()) return
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      const base = `/api/v1/projects/${this.projectId()}/suites/${this.releaseSuite()}/catalog-drafts`
      const draft = await this.api.request<RecordValue<Draft>>(base, 'POST', {})
      const validation = await this.api.request<{ issues: Issue[] }>(`${base}/${draft.id}/validate`, 'POST', {})
      this.draft.set(draft); this.issues.set(validation.issues); this.publicationKey = crypto.randomUUID()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'The release preview could not be prepared.') }
    finally { this.saving.set(false) }
  }

  async publish(): Promise<void> {
    const draft = this.draft()
    if (!draft || this.issues().length || this.saving()) return
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      await this.api.request(`/api/v1/projects/${this.projectId()}/suites/${this.releaseSuite()}/catalog-drafts/${draft.id}/publish`, 'POST', { activate: true }, draft.revision, this.publicationKey)
      this.draft.set(null); this.success.set('Release published and activated. Existing run snapshots retain their original definitions.')
      await this.load()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'Publication failed.') }
    finally { this.saving.set(false) }
  }
}
