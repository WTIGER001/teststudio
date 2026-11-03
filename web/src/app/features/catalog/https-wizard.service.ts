import { computed, DestroyRef, effect, inject, Injectable, signal, untracked } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormControl } from '@angular/forms'
import { Router } from '@angular/router'
import { ApiService, Page, RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import type { Environment } from '../configuration/configuration.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { CatalogTestEditorService } from './catalog-test-editor.service'
import { CreateTestService } from './create-test.service'
import type { Draft, Issue } from './catalog.models'
import { probeTargets, probeURL } from './https-wizard.utils'

@Injectable()
export class HTTPSWizardService {
  readonly editor = inject(CatalogTestEditorService)
  readonly creation = inject(CreateTestService)
  readonly workspace = inject(WorkspaceService)
  private readonly api = inject(ApiService)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly flow = new WizardController(3, step => this.validate(step))
  readonly environment = new FormControl('', { nonNullable: true })
  private readonly selected = toSignal(this.environment.valueChanges, { initialValue: '' })
  private readonly suite = toSignal(this.editor.form.controls.suite.valueChanges, { initialValue: this.editor.form.controls.suite.value })
  private readonly allEnvironments = signal<RecordValue<Environment>[]>([])
  readonly environments = computed(() => probeTargets(this.allEnvironments(), this.editor.suites(), this.suite()))
  readonly target = computed(() => this.environments().find(row => row.id === this.selected()))
  readonly endpoints = computed(() => Object.keys(this.target()?.data.endpoints ?? {}))
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly saved = signal(false)
  readonly error = signal('')
  readonly progress = signal('')
  readonly canRun = computed(() => this.workspace.can('catalog.publish') && this.workspace.can('runs.create'))
  private readonly suffix = crypto.randomUUID().slice(0, 8)
  private draft: RecordValue<Draft> | undefined
  private release = ''
  private readonly publishKey = crypto.randomUUID()
  private readonly runKey = crypto.randomUUID()
  private disposed = false

  constructor() {
    this.editor.form.controls.title.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(title => { if (this.editor.form.controls.id.pristine) this.editor.form.controls.id.setValue(suggestedIdentifier(title, this.suffix)) })
    this.environment.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { const endpoint = Object.keys(this.environments().find(row => row.id === this.environment.value)?.data.endpoints ?? {})[0]; if (endpoint) this.editor.checkEditor.form.controls.endpoint.setValue(endpoint) })
    if (!this.editor.form.controls.rationale.value) this.editor.form.controls.rationale.setValue('Production safety has not been reviewed.')
    effect(() => { const project = this.creation.sources.project(), allowed = this.workspace.can('environments.read'); if (project && allowed) untracked(() => { void this.loadTargets(project) }) })
    effect(() => { const targets = this.environments(); untracked(() => { if (!targets.some(row => row.id === this.environment.value)) this.environment.setValue(targets.length === 1 ? targets[0].id : '') }) })
    this.destroy.onDestroy(() => { this.disposed = true })
  }

  private async loadTargets(project: string): Promise<void> {
    this.loading.set(true)
    try {
      const page = await this.api.request<Page<RecordValue<Environment>>>(`/api/v1/projects/${project}/environments`)
      if (this.disposed || project !== this.creation.sources.project()) return
      this.allEnvironments.set(page.items)
    } catch (error) { if (!this.disposed) this.error.set(error instanceof Error ? error.message : 'Targets could not be loaded. You can still save a draft.') }
    finally { if (!this.disposed) this.loading.set(false) }
  }

  validate(step: number): boolean {
    this.error.set('')
    if (step === 0) {
      const fields = this.editor.form.controls
      for (const field of [fields.title, fields.id, fields.suite, fields.category]) field.markAsTouched()
      if (![fields.title, fields.id, fields.suite, fields.category].every(field => field.valid) || !fields.title.value.trim()) { this.error.set('Enter a title and choose a suite. Check Advanced details if you changed the identifier or category.'); return false }
    } else {
      this.editor.checkEditor.form.markAllAsTouched()
      if (!this.editor.checkEditor.valid() || !this.editor.timingEditor.valid() || this.editor.form.controls.timeout.invalid) { this.error.set('Check the request and Advanced options before continuing.'); return false }
    }
    return true
  }

  suiteName(): string { const id = this.editor.form.controls.suite.value; return this.editor.suites().find(row => row.id === id)?.data.name || id }
  requestURL(): string { const base = this.target()?.data.endpoints[this.editor.checkEditor.form.controls.endpoint.value]; return probeURL(base, this.editor.checkEditor.form.controls.path.value) }

  async save(run: boolean): Promise<void> {
    if (this.busy() || !this.flow.ready()) return
    this.editor.form.markAllAsTouched()
    if (this.editor.form.invalid) { this.error.set('Choose a safety label and provide its rationale before saving.'); return }
    if (run && (!this.canRun() || !this.target() || !this.endpoints().includes(this.editor.checkEditor.form.controls.endpoint.value))) { this.error.set('Choose a registered target with this endpoint before saving and running. Publication and run permissions are required.'); return }
    if (run && this.target()?.data.protected && (this.editor.form.controls.safety.value !== 'safe' || !this.workspace.can('runs.execute-protected'))) { this.error.set('This target is protected. A safe label with an accurate rationale and protected execution permission are required. You can still save a draft.'); return }
    this.busy.set(true); this.error.set('')
    try {
      if (!this.saved()) {
        this.progress.set('Saving test…')
        if (!await this.editor.save()) { this.error.set(this.editor.error()); return }
        this.saved.set(true)
      }
      if (!run) { this.creation.afterCreate(); return }
      const value = this.editor.form.getRawValue(), base = `/api/v1/projects/${this.creation.sources.project()}/suites/${value.suite}/catalog-drafts`
      if (!this.release) {
        this.progress.set('Validating a release…')
        this.draft ??= await this.api.request<RecordValue<Draft>>(base, 'POST', {})
        const validation = await this.api.request<{ valid: boolean; issues: Issue[] }>(`${base}/${this.draft.id}/validate`, 'POST', {})
        if (!validation.valid) throw new Error(validation.issues.map(issue => `${issue.testId || 'Suite'}: ${issue.message}`).join('; ') || 'Release validation failed. Open the catalog to review this saved test.')
        this.progress.set('Publishing without changing the active release…')
        const release = await this.api.request<{ id: string }>(`${base}/${this.draft.id}/publish`, 'POST', { activate: false }, this.draft.revision, this.publishKey)
        this.release = release.id
      }
      this.progress.set('Starting local run…')
      const result = await this.api.request<{ id: string }>(`/api/v1/projects/${this.creation.sources.project()}/runs`, 'POST', { suiteId: value.suite, releaseId: this.release, environmentId: this.environment.value, testIds: [value.id], timeoutSeconds: 900, retries: 0 }, undefined, this.runKey)
      if (!this.disposed) await this.router.navigate(['/projects', this.creation.sources.project(), 'runs', result.id])
    } catch (error) { this.error.set(`${this.saved() ? 'The test is saved. ' : ''}${error instanceof Error ? error.message : 'The action could not be completed.'}`) }
    finally { this.busy.set(false); this.progress.set('') }
  }
}
