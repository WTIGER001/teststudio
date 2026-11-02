import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { ApiService, type Page, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import type { Suite } from '../configuration/configuration.service'
import type { CatalogItem, Draft, Issue, TestDefinition } from './catalog.models'
import { TimingPolicyEditorService } from '../timing/timing-policy-editor.service'
import { catalogCounts, catalogTypes, filterCatalog } from './catalog.utils'
import { HTTPCheckEditorService } from './http-check-editor.service'

@Injectable()
export class CatalogService {
  private readonly api = inject(ApiService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly timingEditor = inject(TimingPolicyEditorService)
  readonly checkEditor = inject(HTTPCheckEditorService)
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
  readonly editorOpen = signal(false)
  readonly editing = signal<CatalogItem | null>(null)
  readonly editingHTTP = computed(() => !this.editing() || this.editing()?.data.runner === 'http')
  readonly releaseSuite = signal('')
  readonly draft = signal<RecordValue<Draft> | null>(null)
  readonly issues = signal<Issue[]>([])
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    title: new FormControl('', { nonNullable: true, validators: Validators.required }),
    category: new FormControl('Availability', { nonNullable: true, validators: Validators.required }),
    timeout: new FormControl(10, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600), Validators.pattern(/^\d+$/)] }),
    safety: new FormControl('needs-review', { nonNullable: true }),
    rationale: new FormControl('', { nonNullable: true, validators: Validators.required })
  })
  private generation = 0
  private publicationKey = ''

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.projectId.set(params.get('projectId') ?? ''); this.suiteFilter.set(''); this.typeFilter.set(''); this.releaseSuite.set(''); this.draft.set(null)
      this.editorOpen.set(false); this.editing.set(null); this.items.set([]); this.suites.set([]); this.success.set('')
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

  create(): void {
    this.editing.set(null)
    this.timingEditor.reset()
    this.checkEditor.reset()
    this.form.reset({ suite: this.suites()[0]?.id ?? '', id: '', title: '', category: 'Availability', timeout: 10, safety: 'needs-review', rationale: '' })
    this.form.controls.id.enable(); this.form.controls.suite.enable()
    this.editorOpen.set(true); this.error.set(''); this.success.set('')
  }

  edit(item: CatalogItem): void {
    this.editing.set(item)
    this.timingEditor.reset(item.data.timing)
    this.checkEditor.reset(item.data.check)
    this.form.reset({ suite: item.suiteId, id: item.id, title: item.data.title, category: item.data.category, timeout: item.data.timeoutSeconds, safety: item.data.safety, rationale: item.data.rationale })
    this.form.controls.id.disable(); this.form.controls.suite.disable()
    this.editorOpen.set(true); this.error.set(''); this.success.set('')
  }

  async save(): Promise<void> {
    this.form.markAllAsTouched()
    this.checkEditor.form.markAllAsTouched()
    if (this.form.invalid || !this.timingEditor.valid() || (this.editingHTTP() && !this.checkEditor.valid()) || this.saving()) return
    this.saving.set(true); this.error.set(''); this.success.set('')
    try {
      const value = this.form.getRawValue()
      const existing = this.editing()
      const test: TestDefinition = {
        ...(existing?.data ?? { description: '', tags: [], dependencies: [], requiredBindings: [], type: 'https', runner: 'http', state: 'enabled', timeoutSeconds: 10 }),
        id: value.id, title: value.title, category: value.category, timeoutSeconds: value.timeout, timing: this.timingEditor.policy(), safety: value.safety, rationale: value.rationale
      }
      if (this.editingHTTP()) test.check = { ...test.check, ...this.checkEditor.check()! }
      const base = `/api/v1/projects/${this.projectId()}/suites/${value.suite}/tests`
      await this.api.request(existing ? `${base}/${value.id}` : base, existing ? 'PUT' : 'POST', test, existing?.revision)
      this.editorOpen.set(false); this.success.set('Draft changes saved. Publish a release to use the updated definition.')
      this.draft.set(null)
      await this.load()
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'The test could not be saved.') }
    finally { this.saving.set(false) }
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
