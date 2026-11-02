import { inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { Router } from '@angular/router'
import { ApiService } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'

@Injectable()
export class ProjectsService {
  readonly workspace = inject(WorkspaceService)
  private readonly api = inject(ApiService)
  private readonly router = inject(Router)
  readonly saving = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    name: new FormControl('', { nonNullable: true, validators: Validators.required }),
    description: new FormControl('', { nonNullable: true })
  })

  async create(): Promise<void> {
    this.form.markAllAsTouched()
    if (this.form.invalid || this.saving()) return
    this.saving.set(true)
    this.error.set('')
    try {
      const data = this.form.getRawValue()
      await this.api.request('/api/v1/projects', 'POST', { ...data, archived: false })
      await this.workspace.load()
      await this.router.navigate(['/projects', data.id, 'configuration'])
    } catch (error) { this.error.set(error instanceof Error ? error.message : 'Project creation failed.') }
    finally { this.saving.set(false) }
  }
}
