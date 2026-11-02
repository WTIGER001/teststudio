import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import type { TimingEstimate } from '../timing/timing.models'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { Router } from '@angular/router'
import { ApiService, type RecordValue } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { RunLookupsService } from './run-lookups.service'
import type { Run } from './run.models'
import { errorMessage, requestFromForm } from './run.utils'

@Injectable()
export class NewRunService {
  private readonly api = inject(ApiService)
  private readonly router = inject(Router)
  readonly workspace = inject(WorkspaceService)
  readonly lookups = inject(RunLookupsService)
  readonly saving = signal(false)
  readonly estimating = signal(false)
  readonly estimate = signal<TimingEstimate | undefined>(undefined)
  private estimateGeneration = 0
  private readonly destroy = inject(DestroyRef)
  readonly error = signal('')
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    environment: new FormControl('', { nonNullable: true }), tests: new FormControl('', { nonNullable: true }),
    timeout: new FormControl(900, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600)] }),
    retries: new FormControl(0, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(2)] })
  })
  private key = crypto.randomUUID()
  private fingerprint = ''

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.estimateGeneration++; this.estimate.set(undefined); this.estimating.set(false) })
    this.destroy.onDestroy(() => { this.estimateGeneration++ })
  }

  async estimateTiming(): Promise<void> {
    if (this.form.invalid || this.estimating() || this.saving()) return
    const generation = ++this.estimateGeneration
    this.estimating.set(true); this.error.set('')
    try {
      const estimate = await this.api.request<TimingEstimate>(`/api/v1/projects/${encodeURIComponent(this.workspace.projectId())}/runs/estimate`, 'POST', requestFromForm(this.form.getRawValue()))
      if (generation === this.estimateGeneration) this.estimate.set(estimate)
    } catch (error) { if (generation === this.estimateGeneration) this.error.set(errorMessage(error, 'Timing could not be estimated.')) }
    finally { if (generation === this.estimateGeneration) this.estimating.set(false) }
  }

  async create(): Promise<void> {
    if (this.form.invalid || this.saving()) return
    this.saving.set(true); this.error.set('')
    try {
      const input = requestFromForm(this.form.getRawValue())
      const fingerprint = JSON.stringify(input)
      if (fingerprint !== this.fingerprint) { this.key = crypto.randomUUID(); this.fingerprint = fingerprint }
      const run = await this.api.request<RecordValue<Run>>(`/api/v1/projects/${encodeURIComponent(this.workspace.projectId())}/runs`, 'POST', input, undefined, this.key)
      await this.router.navigate(['/projects', this.workspace.projectId(), 'runs', run.id])
    } catch (error) { this.error.set(errorMessage(error, 'The run could not be started.')) }
    finally { this.saving.set(false) }
  }
}
