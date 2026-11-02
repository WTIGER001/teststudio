import { inject, Injectable, signal } from '@angular/core'
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
  readonly error = signal('')
  readonly form = new FormGroup({
    suite: new FormControl('', { nonNullable: true, validators: Validators.required }),
    environment: new FormControl('', { nonNullable: true }), tests: new FormControl('', { nonNullable: true }),
    timeout: new FormControl(900, { nonNullable: true, validators: [Validators.required, Validators.min(1), Validators.max(3600)] }),
    retries: new FormControl(0, { nonNullable: true, validators: [Validators.required, Validators.min(0), Validators.max(2)] })
  })
  private key = crypto.randomUUID()
  private fingerprint = ''

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
