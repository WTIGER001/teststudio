import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { ApplicationService } from '../../core/application.service'
import { loginDefinition, loginValidation } from './password.utils'

@Injectable()
export class PasswordSigninService {
  private readonly app = inject(ApplicationService)
  readonly session = this.app.session
  readonly busy = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({ userId: new FormControl('', { nonNullable: true }), password: new FormControl('', { nonNullable: true }) })
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.form.reset() }) }
  async submit(): Promise<void> {
    if (this.busy() || this.session.authenticating()) return
    const body = loginDefinition(this.form.getRawValue()), validation = loginValidation(body)
    this.error.set(validation); if (validation) return
    const generation = ++this.generation; this.busy.set(true)
    try { await this.app.signInWithPassword(body) }
    catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Sign-in failed.') }
    finally { this.form.controls.password.reset(); if (generation === this.generation) this.busy.set(false) }
  }
  async refresh(): Promise<void> {
    if (this.busy() || this.session.authenticating()) return
    const generation = ++this.generation; this.busy.set(true)
    try { await this.app.refreshAuthentication(); if (generation === this.generation && !this.session.signedIn()) this.error.set('No sign-in is active. Enter your credentials to try again.') }
    catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Sign-in state could not be refreshed.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
