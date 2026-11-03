import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApplicationService } from '../../core/application.service'
import { ApiError } from '../../core/api.service'
import type { LocalCredentialRecord } from '../../core/generated/access.types'
import { AccessActionService } from './access-action.service'
import { accessFocus, reasonValidation } from './access.utils'
import { passwordPolicy } from '../auth/password.utils'

@Injectable()
export class LocalCredentialService {
  private readonly client = inject(AccessClientService)
  private readonly app = inject(ApplicationService)
  readonly action = inject(AccessActionService)
  readonly userId = signal('')
  readonly name = signal('')
  readonly record = signal<LocalCredentialRecord | null>(null)
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly opened = signal(false)
  readonly editing = signal<LocalCredentialRecord | null>(null)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly form = new FormGroup({ password: new FormControl('', { nonNullable: true }), confirmation: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  onStateChanged: (() => void) | undefined
  private generation = 0
  private readGeneration = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.readGeneration++; this.form.reset() }) }
  select(user: string, name: string): void {
    this.name.set(name); if (user === this.userId()) return
    this.generation++; this.readGeneration++; this.userId.set(user); this.record.set(null); this.editing.set(null); this.form.reset(); this.opened.set(false); this.error.set(''); this.message.set(''); this.busy.set(false); this.conflict.set(false); void this.load()
  }
  async load(): Promise<void> {
    const user = this.userId(), generation = ++this.readGeneration; this.loading.set(true); this.error.set('')
    try { const record = await this.client.call('readLocalCredential', { user }); if (generation === this.readGeneration) this.record.set(record) }
    catch (error) { if (generation === this.readGeneration) { this.record.set(null); this.editing.set(null); this.opened.set(false); this.form.reset(); this.generation++; this.error.set(error instanceof Error ? error.message : 'The credential state could not be loaded.') } }
    finally { if (generation === this.readGeneration) this.loading.set(false) }
  }
  open(): void { if (!this.record()?.data.canManage) return; this.generation++; this.editing.set(this.record()); this.form.reset(); this.opened.set(true); this.conflict.set(false); this.error.set(''); this.message.set(''); accessFocus('local-password-editor-heading') }
  close(): void { this.generation++; this.form.reset(); this.editing.set(null); this.opened.set(false); this.busy.set(false); this.conflict.set(false); accessFocus('local-credential-heading') }
  async submit(): Promise<boolean> {
    const selected = this.editing(); if (!selected?.data.canManage || !this.record()?.data.canManage || this.busy() || this.conflict()) return false
    const value = this.form.getRawValue(), validation = value.password !== value.confirmation ? 'The passwords must match.' : passwordPolicy(value.password) || reasonValidation(value.reason)
    this.error.set(validation); if (validation) return false
    const generation = ++this.generation, user = this.userId(); this.busy.set(true)
    try {
      const result = await this.client.call('setLocalPassword', { user, revision: selected.revision, body: { password: value.password, reason: value.reason.trim() } })
      if (generation !== this.generation) return false
      this.record.set(result); this.opened.set(false); this.form.reset(); this.message.set('Local password saved. A password change is required at password sign-in. Earlier sessions are invalid.'); accessFocus('local-credential-heading')
      if (user === this.app.session.user()?.id) { this.app.message.set('Local credentials changed. Sign in again using an enabled sign-in method.'); await this.app.refreshAuthentication(); return false }
      return generation === this.generation
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The password update could not be confirmed. Refresh credential state before retrying.'); this.conflict.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500) }; return false }
    finally { if (generation === this.generation) { this.form.controls.password.reset(); this.form.controls.confirmation.reset(); this.busy.set(false) } }
  }
  state(disabled: boolean): void {
    const selected = this.record(); if (!selected?.data.canManage || !selected.data.configured) return
    const user = this.userId(), generation = this.generation
    this.action.open({ title: disabled ? 'Disable local sign-in' : 'Enable local sign-in', target: `${this.name()} (${user})`, returnFocus: 'local-credential-heading',
      impact: 'This changes local password sign-in and invalidates earlier sessions for the account. The last usable installation administrator remains protected.',
      submit: async reason => {
        const record = await this.client.call(disabled ? 'disableLocalCredential' : 'enableLocalCredential', { user, revision: selected.revision, body: { reason } })
        if (generation === this.generation) { this.record.set(record); if (user === this.app.session.user()?.id) { this.app.message.set('Local credentials changed. Sign in again using an enabled sign-in method.'); await this.app.refreshAuthentication(); return 'Local credentials changed. Sign in again.' } }
        if (generation === this.generation) this.onStateChanged?.()
        return disabled ? 'Local sign-in disabled. Earlier sessions remain invalid.' : 'Local sign-in enabled. Earlier sessions remain invalid.'
      } })
  }
}
