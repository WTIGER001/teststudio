import { randomId } from '../../core/random-id'
import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier } from '../../shared/wizard/creation.utils'
import { apiBodyMethod, apiPathPrefix, apiReadOnlyMethod, apiStarterProblem } from './api-draft.utils'
import { PlaywrightEditorService } from './playwright-editor.service'

@Injectable()
export class APIDraftWizardService {
  readonly editor = inject(PlaywrightEditorService)
  readonly error = signal('')
  readonly flow = new WizardController(3, step => this.validate(step))
  readonly bodyMethod = apiBodyMethod
  readonly readOnlyMethod = apiReadOnlyMethod
  readonly form = new FormGroup({
	 authScheme: new FormControl('none', { nonNullable: true }), credentialBinding: new FormControl('api-login', { nonNullable: true }), credentialHeader: new FormControl('X-API-Key', { nonNullable: true }),
    title: new FormControl('', { nonNullable: true }), id: new FormControl('', { nonNullable: true }),
    method: new FormControl('GET', { nonNullable: true }), path: new FormControl('/health', { nonNullable: true }),
    endpoint: new FormControl('api', { nonNullable: true }), prefix: new FormControl('/health', { nonNullable: true }),
    status: new FormControl(200, { nonNullable: true }), readOnly: new FormControl(false, { nonNullable: true }),
    body: new FormControl('', { nonNullable: true }), maxBodyBytes: new FormControl(65536, { nonNullable: true }),
    maxRequests: new FormControl(10, { nonNullable: true }), maxResponseBytes: new FormControl(65536, { nonNullable: true })
  })

  constructor() {
    const destroy = inject(DestroyRef), suffix = randomId().slice(0, 8), controls = this.form.controls
    controls.id.setValue('API-' + suggestedIdentifier('api-test', suffix), { emitEvent: false })
    controls.title.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(title => { if (controls.id.pristine) controls.id.setValue('API-' + suggestedIdentifier(title, suffix)) })
    controls.path.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(value => { if (controls.prefix.pristine) controls.prefix.setValue(apiPathPrefix(value)) })
    controls.method.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(method => { if (!apiReadOnlyMethod(method)) controls.readOnly.setValue(false) })
    this.form.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(() => this.editor.sources.creationDirty.set(true))
    destroy.onDestroy(() => this.editor.sources.creationDirty.set(false))
  }

  validate(step: number): boolean {
    for (const control of Object.values(this.form.controls)) control.updateValueAndValidity({ emitEvent: false })
    const problem = apiStarterProblem(this.form.getRawValue(), step)
    this.error.set(problem?.message ?? '')
    if (problem) { const field = this.form.controls[problem.field]; field.markAsTouched(); field.setErrors({ apiPolicy: true }) }
    return !problem
  }
  async submit(): Promise<boolean> {
    if (!this.flow.ready()) return false
    return this.editor.create('api', this.form.getRawValue())
  }
}
