import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import type { TimingEstimate } from '../timing/timing.models'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { Router } from '@angular/router'
import { ApiError, ApiService, type RecordValue } from '../../core/api.service'
import { AccessClientService } from '../../core/access-client.service'
import type { SavedSelectionPreview, SavedSelectionRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { RunLookupsService } from './run-lookups.service'
import type { Run } from './run.models'
import { errorMessage, requestFromForm } from './run.utils'
import { WizardController } from '../../shared/wizard/wizard-controller'

@Injectable()
export class NewRunService {
  private readonly api = inject(ApiService)
  private readonly client = inject(AccessClientService)
  private readonly router = inject(Router)
  readonly workspace = inject(WorkspaceService)
  readonly lookups = inject(RunLookupsService)
  readonly saving = signal(false)
  readonly estimating = signal(false)
  readonly estimate = signal<TimingEstimate | undefined>(undefined)
  private estimateGeneration = 0
  private readonly destroy = inject(DestroyRef)
  readonly error = signal('')
  readonly flow = new WizardController(2, () => { this.form.markAllAsTouched(); if (this.form.invalid || this.needsSelectionPreview()) { this.error.set('Choose a suite, check the options, and preview any saved selection before continuing.'); return false }; this.error.set(''); return true })
  readonly selections = signal<SavedSelectionRecord[]>([])
  readonly selectionCursor = signal('')
  readonly selectionLoading = signal(false)
  readonly selectionResolving = signal(false)
  readonly selectionPreview = signal<SavedSelectionPreview | null>(null)
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    selection: new FormControl('', { nonNullable: true }),
    environment: new FormControl('', { nonNullable: true }), tests: new FormControl('', { nonNullable: true }),
    timeout: new FormControl(900, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600)] }),
    retries: new FormControl(0, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(2)] })
  })
  readonly selectionId = toSignal(this.form.controls.selection.valueChanges, { initialValue: '' })
  readonly needsSelectionPreview = computed(() => !!this.selectionId() && !this.selectionPreview())
  private selectionGeneration = 0
  private selectionLoadGeneration = 0
  private key = crypto.randomUUID()
  private fingerprint = ''

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.estimateGeneration++; this.estimate.set(undefined); this.estimating.set(false) })
    this.form.controls.suite.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.selectionGeneration++; this.selectionLoadGeneration++; this.selectionLoading.set(false); this.form.controls.selection.setValue(''); this.selections.set([]); this.selectionCursor.set(''); void this.loadSelections() })
    this.form.controls.selection.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(selection => { this.selectionGeneration++; this.selectionPreview.set(null); this.selectionResolving.set(false); this.error.set(''); this.form.controls.tests.setValue('', { emitEvent: false }); if (selection) this.form.controls.tests.disable({ emitEvent: false }); else this.form.controls.tests.enable({ emitEvent: false }) })
    this.destroy.onDestroy(() => { this.estimateGeneration++; this.selectionGeneration++; this.selectionLoadGeneration++ })
  }

  async loadSelections(more = false): Promise<void> {
    const suite = this.form.controls.suite.value
    if (!suite || more && !this.selectionCursor()) return
    const generation = ++this.selectionLoadGeneration
    this.selectionLoading.set(true)
    try {
      const page = await this.client.call('listSavedSelections', { project: this.workspace.projectId(), suite, query: { cursor: more ? this.selectionCursor() : undefined, limit: 30 } })
      if (generation === this.selectionLoadGeneration) { this.selections.set(more ? [...this.selections(), ...page.items] : page.items); this.selectionCursor.set(page.cursor ?? '') }
    } catch (error) { if (generation === this.selectionLoadGeneration) this.error.set(errorMessage(error, 'Saved selections could not be loaded.')) }
    finally { if (generation === this.selectionLoadGeneration) this.selectionLoading.set(false) }
  }

  async previewSelection(): Promise<void> {
    const selection = this.selectionId()
    if (!selection || this.selectionResolving()) return
    const generation = ++this.selectionGeneration
    this.selectionResolving.set(true); this.selectionPreview.set(null); this.error.set(''); this.estimate.set(undefined)
    try {
      const preview = await this.client.call('previewSavedSelection', { project: this.workspace.projectId(), suite: this.form.controls.suite.value, id: selection, body: {} })
      if (generation === this.selectionGeneration) this.selectionPreview.set(preview)
    } catch (error) { if (generation === this.selectionGeneration) this.error.set(errorMessage(error, 'Saved selection could not be resolved.')) }
    finally { if (generation === this.selectionGeneration) this.selectionResolving.set(false) }
  }

  async estimateTiming(): Promise<void> {
    if (this.form.invalid || this.estimating() || this.saving() || this.needsSelectionPreview()) return
    const generation = ++this.estimateGeneration
    this.estimating.set(true); this.error.set('')
    try {
      const estimate = await this.api.request<TimingEstimate>(`/api/v1/projects/${encodeURIComponent(this.workspace.projectId())}/runs/estimate`, 'POST', requestFromForm(this.form.getRawValue(), this.selectionPreview()?.id))
      if (generation === this.estimateGeneration) this.estimate.set(estimate)
    } catch (error) { if (generation === this.estimateGeneration) this.error.set(errorMessage(error, 'Timing could not be estimated.')) }
    finally { if (generation === this.estimateGeneration) this.estimating.set(false) }
  }

  suiteName(): string { const id = this.form.controls.suite.value; return this.lookups.suites().find(row => row.id === id)?.data.name || id }
  environmentName(): string { const id = this.form.controls.environment.value; const target = this.lookups.environments().find(row => row.id === id); return target ? `${target.data.name} · ${target.data.stage}${target.data.protected ? ' · Protected' : ''}` : id || 'No target' }
  targetWarning(): string { const target = this.lookups.environments().find(row => row.id === this.form.controls.environment.value); return target?.data.protected ? 'Protected target: only safe tests can run, with protected execution permission.' : target?.data.stage === 'production' ? 'Production target: this environment is not protected. Review the selected tests before starting.' : 'Starting creates a run from the published release. Review the target and selected tests before continuing.' }

  async create(): Promise<void> {
    if (!this.flow.ready() || this.saving()) return
    this.saving.set(true); this.error.set('')
    try {
      const input = requestFromForm(this.form.getRawValue(), this.selectionPreview()?.id)
      const fingerprint = JSON.stringify(input)
      if (fingerprint !== this.fingerprint) { this.key = crypto.randomUUID(); this.fingerprint = fingerprint }
      const run = await this.api.request<RecordValue<Run>>(`/api/v1/projects/${encodeURIComponent(this.workspace.projectId())}/runs`, 'POST', input, undefined, this.key)
      await this.router.navigate(['/projects', this.workspace.projectId(), 'runs', run.id])
    } catch (error) { if (this.selectionPreview() && error instanceof ApiError && error.status === 409) { this.selectionPreview.set(null); this.error.set('The saved selection changed. Preview the saved selection again before starting the run.') } else this.error.set(errorMessage(error, 'The run could not be started.')) }
    finally { this.saving.set(false) }
  }
}
