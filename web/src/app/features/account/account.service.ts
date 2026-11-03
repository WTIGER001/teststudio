import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { AccessClientService } from '../../core/access-client.service'
import type { ExternalIdentityRecord, LocalCredentialRecord, User } from '../../core/generated/access.types'
import { AccessCollection } from '../access/access-collection'

@Injectable()
export class AccountService {
  private readonly client = inject(AccessClientService)
  readonly user = signal<User | null>(null)
  readonly credential = signal<LocalCredentialRecord | null>(null)
  readonly identities = new AccessCollection<ExternalIdentityRecord>()
  readonly loading = signal(false)
  readonly error = signal('')
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.identities.clear() }); void this.refresh() }
  async refresh(): Promise<void> {
    const generation = ++this.generation; this.loading.set(true); this.error.set('')
    try {
      const [profile, credential] = await Promise.all([this.client.call('currentProfile', { query: { limit: 30 } }), this.client.call('readOwnLocalCredential', {})])
      if (generation !== this.generation) return
      this.user.set(profile.user); this.credential.set(credential)
      await this.identities.load(() => Promise.resolve(profile.identities))
    } catch (error) { if (generation === this.generation) { this.user.set(null); this.credential.set(null); this.identities.clear(); this.error.set(error instanceof Error ? error.message : 'Your account could not be loaded.') } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async moreIdentities(): Promise<void> { await this.identities.load(cursor => this.client.call('currentProfile', { query: { limit: 30, cursor } }).then(profile => profile.identities), true) }
}
