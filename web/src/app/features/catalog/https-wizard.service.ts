import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { Router } from '@angular/router'
import { ApiService, RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { CatalogTestEditorService } from './catalog-test-editor.service'
import { CreateTestService } from './create-test.service'
import type { Draft, Issue } from './catalog.models'
import { ProbeTargetService } from './probe-target.service'
import { parseProbeAddress } from './probe-address.utils'

@Injectable()
export class HTTPSWizardService {
  readonly editor = inject(CatalogTestEditorService)
  readonly creation = inject(CreateTestService)
  readonly workspace = inject(WorkspaceService)
  private readonly api = inject(ApiService)
  private readonly router = inject(Router)
  private readonly destroy = inject(DestroyRef)
  readonly flow = new WizardController(3, step => this.validate(step))
  readonly targets = inject(ProbeTargetService)
  readonly environment = this.targets.environment
  readonly environments = this.targets.environments
  readonly target = this.targets.target
  readonly endpoints = this.targets.endpoints
  readonly loading = this.targets.loading
  readonly locked = computed(() => this.saved() || this.targets.locked())
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
    if (!this.editor.form.controls.rationale.value) this.editor.form.controls.rationale.setValue('Production safety has not been reviewed.')
    this.destroy.onDestroy(() => { this.disposed = true })
  }

  validate(step: number): boolean {
    this.error.set('')
    if (step === 0) {
      const fields = this.editor.form.controls
      for (const field of [fields.title, fields.id, fields.suite, fields.category]) field.markAsTouched()
      if (![fields.title, fields.id, fields.suite, fields.category].every(field => field.valid) || !fields.title.value.trim()) { this.error.set('Enter a title and choose a suite. Check Advanced details if you changed the identifier or category.'); return false }
      if (!this.targets.address.value && this.targets.address.pristine && parseProbeAddress(fields.title.value)) this.targets.address.setValue(fields.title.value)
    } else {
      this.targets.address.markAsTouched()
      const issue = this.targets.issue()
      if (issue) { this.error.set(issue); return false }
      this.editor.checkEditor.form.markAllAsTouched()
      if (!this.editor.checkEditor.valid() || !this.editor.timingEditor.valid() || this.editor.form.controls.timeout.invalid) { this.error.set('Check the request and Advanced options before continuing.'); return false }
    }
    return true
  }

  suiteName(): string { const id = this.editor.form.controls.suite.value; return this.editor.suites().find(row => row.id === id)?.data.name || id }
  requestURL(): string { return this.targets.requestURL() }

  async save(run: boolean): Promise<void> {
    if (this.busy() || !this.flow.ready()) return
    this.editor.form.markAllAsTouched()
    if (this.editor.form.invalid) { this.error.set('Choose a safety label and provide its rationale before saving.'); return }
    const issue = this.targets.issue(run)
    if (issue) { this.error.set(issue); return }
    if (run && !this.canRun()) { this.error.set('Publication and run permissions are required.'); return }
    if (run && (this.target()?.data.protected || this.targets.needsTarget() && this.targets.protected.value) && (this.editor.form.controls.safety.value !== 'safe' || !this.workspace.can('runs.execute-protected'))) { this.error.set('This target is protected. A safe label with an accurate rationale and protected execution permission are required. You can still save a draft.'); return }
    this.busy.set(true); this.error.set('')
    const project = this.creation.sources.project(), value = this.editor.form.getRawValue()
    try {
      if (this.targets.needsTarget()) { this.progress.set('Saving reviewed target configuration…'); await this.targets.ensureTarget(project) }
      if (this.disposed || project !== this.creation.sources.project()) return
      if (!this.saved()) {
        this.progress.set('Saving test…')
        if (!await this.editor.save()) { this.error.set(this.editor.error()); return }
        this.saved.set(true)
      }
      if (!run) { this.creation.afterCreate(); return }
      if (this.disposed || project !== this.creation.sources.project()) return
      const base = `/api/v1/projects/${project}/suites/${value.suite}/catalog-drafts`
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
      const result = await this.api.request<{ id: string }>(`/api/v1/projects/${project}/runs`, 'POST', { suiteId: value.suite, releaseId: this.release, environmentId: this.targets.environmentId(), testIds: [value.id], timeoutSeconds: 900, retries: 0 }, undefined, this.runKey)
      if (!this.disposed && project === this.creation.sources.project()) await this.router.navigate(['/projects', project, 'runs', result.id])
    } catch (error) { this.error.set(`${this.saved() ? 'The test is saved. ' : this.targets.targetSaved() || this.targets.applicationSaved() ? 'Target configuration is saved. Retry to continue saving the test. ' : ''}${error instanceof Error ? error.message : 'The action could not be completed.'}`) }
    finally { this.busy.set(false); this.progress.set('') }
  }
}
