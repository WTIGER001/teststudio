import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { WizardController } from '../../shared/wizard/wizard-controller'
import { suggestedIdentifier, validWebAddress } from '../../shared/wizard/creation.utils'
import { ConfigurationService } from './configuration.service'

@Injectable()
export class RegistrationWizardService {
  readonly registration = inject(ConfigurationService)
  readonly error = signal('')
  readonly flow = new WizardController(3, step => this.validate(step))
  private readonly suffix = crypto.randomUUID().slice(0, 8)

  constructor() {
    const destroy = inject(DestroyRef), form = this.registration.form
    form.controls.name.valueChanges.pipe(takeUntilDestroyed(destroy)).subscribe(name => { if (form.controls.id.pristine) form.controls.id.setValue(suggestedIdentifier(name, this.suffix)) })
    if (!form.controls.id.value && form.controls.name.value) form.controls.id.setValue(suggestedIdentifier(form.controls.name.value, this.suffix))
  }

  validate(step: number): boolean {
    const form = this.registration.form, kind = this.registration.kind()
    this.error.set('')
    const fields = step === 0 ? [form.controls.name, form.controls.id] : kind === 'environments' ? [form.controls.application, form.controls.url, form.controls.endpoint, form.controls.stage] : []
    for (const field of fields) field.markAsTouched()
    const valid = fields.every(field => field.disabled || field.valid) && (step !== 0 || !!form.controls.name.value.trim()) && (step !== 1 || kind !== 'environments' || !!form.controls.application.value && validWebAddress(form.controls.url.value))
    if (!valid) this.error.set(step === 0 ? 'Enter a name and a valid identifier. Identifier options are under Advanced details.' : 'Choose an application and enter a valid HTTP or HTTPS address and endpoint name.')
    return valid
  }

  submit(): boolean { return this.flow.ready() && this.registration.form.valid }
  applicationName(): string { const id = this.registration.form.controls.application.value; return this.registration.applications().find(row => row.id === id)?.data.name || id || 'None' }
}
