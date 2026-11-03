import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { Router } from '@angular/router'
import { RecordCreationService } from '../../core/record-creation.service'
import { WorkspaceService } from '../../core/workspace.service'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier, validWebAddress } from '../../shared/wizard/creation.utils'

@Injectable()
export class ProjectsService {
  readonly workspace = inject(WorkspaceService)
  private readonly creation = inject(RecordCreationService)
  private readonly router = inject(Router)
  readonly saving = signal(false)
  readonly locked = signal(false)
  readonly error = signal('')
  readonly completed = signal<string[]>([])
  readonly flow = new WizardController(4, step => this.validate(step))
  private readonly suffix = crypto.randomUUID().slice(0, 8)
  readonly form = new FormGroup({
    id: new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern(/^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$/)] }),
    name: new FormControl('', { nonNullable: true, validators: Validators.required }),
    description: new FormControl('', { nonNullable: true }),
    addApplication: new FormControl(true, { nonNullable: true }),
    applicationName: new FormControl('', { nonNullable: true }),
    addSuite: new FormControl(true, { nonNullable: true }),
    addTarget: new FormControl(false, { nonNullable: true }),
    targetName: new FormControl('Development', { nonNullable: true }),
    url: new FormControl('', { nonNullable: true }),
    stage: new FormControl('development', { nonNullable: true }),
    protected: new FormControl(false, { nonNullable: true })
  })
  private disposed = false

  constructor() {
    const destroy = inject(DestroyRef), fields = this.form.controls
    fields.name.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(name => {
      if (fields.id.pristine) fields.id.setValue(suggestedIdentifier(name, this.suffix))
      if (fields.applicationName.pristine) fields.applicationName.setValue(name)
    })
    fields.addApplication.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(enabled => { if (!enabled) fields.addTarget.setValue(false) })
    destroy.onDestroy(() => { this.disposed = true })
  }

  applicationId(): string { return suggestedIdentifier(this.form.controls.applicationName.value, this.suffix) }
  suiteId(): string { return `availability-${this.suffix}` }
  environmentId(): string { return suggestedIdentifier(this.form.controls.targetName.value, this.suffix) }

  validate(step: number): boolean {
    const value = this.form.getRawValue()
    this.error.set('')
    if (step === 0) {
      this.form.controls.name.markAsTouched(); this.form.controls.id.markAsTouched()
      if (this.form.controls.name.invalid || !value.name.trim() || this.form.controls.id.invalid) { this.error.set('Enter a project name. Check the identifier under Advanced details if you changed it.'); return false }
    }
    if (step === 1 && value.addApplication && !value.applicationName.trim()) { this.error.set('Enter an application name or choose to add it later.'); return false }
    if (step === 2 && value.addTarget && (!value.addApplication || !value.targetName.trim() || !validWebAddress(value.url))) { this.error.set('Enter a target name and a valid HTTP or HTTPS address.'); return false }
    return true
  }

  async create(): Promise<void> {
    if (!this.flow.ready() || this.saving()) return
    this.saving.set(true); this.error.set('')
    const value = this.form.getRawValue(), base = `/api/v1/projects/${value.id}`
    try {
      if (!this.completed().includes('Project')) {
        await this.creation.create('/api/v1/projects', { id: value.id, name: value.name, description: value.description, archived: false }, 'POST', base)
        this.completed.update(rows => [...rows, 'Project']); this.locked.set(true)
      }
      if (value.addApplication && !this.completed().includes('Application')) {
        const path = `${base}/applications/${this.applicationId()}`
        await this.creation.create(path, { id: this.applicationId(), name: value.applicationName, description: '' }, 'PUT', path)
        this.completed.update(rows => [...rows, 'Application'])
      }
      if (value.addTarget && !this.completed().includes('Target environment')) {
        const path = `${base}/environments/${this.environmentId()}`
        await this.creation.create(path, { id: this.environmentId(), name: value.targetName, applicationId: this.applicationId(), stage: value.stage, protected: value.protected, disabled: false, endpoints: { api: value.url } }, 'PUT', path)
        this.completed.update(rows => [...rows, 'Target environment'])
      }
      if (value.addSuite && !this.completed().includes('Availability suite')) {
        const path = `${base}/suites/${this.suiteId()}`
        await this.creation.create(path, { id: this.suiteId(), name: 'Availability', description: '', applications: value.addApplication ? [this.applicationId()] : [], requiresTarget: value.addApplication, archived: false }, 'PUT', path)
        this.completed.update(rows => [...rows, 'Availability suite'])
      }
      await this.workspace.load()
      if (!this.disposed) {
        this.form.markAsPristine()
        await this.router.navigate(value.addSuite ? ['/projects', value.id, 'tests', 'new'] : ['/projects', value.id, 'applications'], value.addSuite ? { queryParams: { suite: this.suiteId() } } : {})
      }
    } catch (error) { this.error.set(`${this.completed().length ? `${this.completed().join(', ')} created. Retry continues the remaining setup; it does not create these resources again. ` : ''}${error instanceof Error ? error.message : 'Project setup failed.'}`) }
    finally { this.saving.set(false) }
  }
  canLeave(): boolean { return !this.form.dirty || window.confirm(this.locked() ? 'Some resources are already created. Leave setup and manage them from the project?' : 'This setup has unsaved changes. Leave without saving?') }
}
