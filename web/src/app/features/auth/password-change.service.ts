import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { ApplicationService } from '../../core/application.service'
import { ApiError } from '../../core/api.service'
import { SessionRefreshError } from '../../core/session.service'
import { passwordChangeDefinition, passwordChangeValidation } from './password.utils'

@Injectable()
export class PasswordChangeService {
  private readonly app = inject(ApplicationService)
  readonly session = this.app.session
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly needsRefresh = signal(false)
  readonly completed = signal(false)
  readonly form = new FormGroup({ currentPassword: new FormControl('', { nonNullable: true }), newPassword: new FormControl('', { nonNullable: true }), confirmation: new FormControl('', { nonNullable: true }) })
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.form.reset() }) }
  async submit(): Promise<boolean> {
    if (this.busy() || this.needsRefresh() || this.session.authenticating()) return false
    const value = this.form.getRawValue(), body = passwordChangeDefinition(value), validation = passwordChangeValidation(body, value.confirmation)
    this.error.set(validation); if (validation) return false
    const generation = ++this.generation; this.busy.set(true); this.message.set(''); this.completed.set(false)
    try {
      await this.app.changePassword(body)
      if (generation !== this.generation) return false
      this.completed.set(true); this.message.set('Password changed. Earlier sessions are invalid.'); return true
    } catch (error) {
      if (generation === this.generation) {
        this.completed.set(error instanceof SessionRefreshError)
        this.needsRefresh.set(error instanceof SessionRefreshError || !(error instanceof ApiError) || error.status === 409 || error.status >= 500)
        this.error.set(error instanceof Error ? error.message : 'The password change could not be confirmed. Refresh your session before retrying.')
      }
      return false
    } finally { this.form.reset(); if (generation === this.generation) this.busy.set(false) }
  }
  async refresh(): Promise<boolean> {
    if (this.busy() || this.session.authenticating()) return false
    const generation = ++this.generation; this.busy.set(true)
    try {
      await this.app.refreshAuthentication()
      if (generation !== this.generation) return false
      this.needsRefresh.set(false); this.error.set(''); this.message.set(this.completed() ? 'Password changed. Earlier sessions are invalid.' : 'Session refreshed. Enter current credentials before another password change.')
      return this.completed()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Session refresh failed.'); return false }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
