import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import { ProvidersService } from './providers.service'
import { providerGroupValidation } from './provider.utils'

@Injectable()
export class ProviderGroupsService {
  private readonly client = inject(AccessClientService)
  private readonly providers = inject(ProvidersService)
  readonly form = new FormGroup({ id: new FormControl('', { nonNullable: true }), name: new FormControl('', { nonNullable: true }), projectId: new FormControl('', { nonNullable: true }) })
  readonly busy = signal(false)
  readonly blocked = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.form.reset() }) }
  reset(): void { this.generation++; this.busy.set(false); this.blocked.set(false); this.error.set(''); this.message.set(''); this.form.reset() }
  async submit(): Promise<void> {
    if (this.busy() || this.blocked()) return
    const fields = this.form.getRawValue(), validation = providerGroupValidation(fields); this.error.set(validation); if (validation) return
    const generation = ++this.generation, provider = this.providers.id(); this.busy.set(true); this.message.set('')
    try {
      await this.client.call('createGroup', { body: { id: fields.id.trim(), name: fields.name.trim(), projectId: fields.projectId.trim(), managedBy: `oidc:${provider}`, description: '' } })
      if (generation !== this.generation) return
      this.form.reset(); this.message.set(`Provider group ${fields.id.trim()} created. Add its identifier to an external group mapping. No roles or memberships were granted.`)
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The group could not be created.'); this.blocked.set(error instanceof ApiError && (error.status === 409 || error.status >= 500 || error.message.includes('unreadable')) || !(error instanceof ApiError)); this.providers.accessError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
