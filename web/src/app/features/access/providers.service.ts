import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { ProviderRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessCollection } from './access-collection'
import { providerMappings } from './provider.utils'

@Injectable()
export class ProvidersService {
  private readonly client = inject(AccessClientService)
  private readonly workspace = inject(WorkspaceService)
  private readonly destroy = inject(DestroyRef)
  readonly id = signal('')
  readonly detail = signal<ProviderRecord | null>(null)
  readonly providers = new AccessCollection<ProviderRecord>()
  readonly credentialRefs = signal<string[]>([])
  readonly canCreate = signal(false)
  readonly readable = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly mappings = computed(() => providerMappings(this.detail()?.data.groupMappings ?? {}))
  onAccessLost: (() => void) | undefined
  private generation = 0
  private listGeneration = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.generation++; this.clearPrivate(); this.error.set(''); this.message.set(''); this.id.set(params.get('providerId') ?? ''); void this.refresh() })
    this.destroy.onDestroy(() => { this.generation++; this.clearPrivate() })
  }
  private clearPrivate(): void { this.listGeneration++; this.detail.set(null); this.providers.clear(); this.credentialRefs.set([]); this.canCreate.set(false); this.readable.set(false); this.onAccessLost?.() }
  async refresh(): Promise<void> {
    const generation = ++this.generation; this.loading.set(true); this.error.set('')
    try {
      const caps = await this.workspace.readCapabilities(''); if (generation !== this.generation) return
      if (!caps['teststudio.providers.read']) { this.clearPrivate(); this.error.set('Installation provider read authority is required.'); return }
      this.readable.set(true)
      await Promise.all([this.load(), this.id() ? this.readDetail(generation) : Promise.resolve()])
    } catch (error) { if (generation === this.generation) { this.clearPrivate(); this.error.set(error instanceof Error ? error.message : 'Provider access could not be loaded.') } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  private async readDetail(generation: number): Promise<void> {
    try { const detail = await this.client.call('readIdentityProvider', { provider: this.id() }); if (generation === this.generation) { this.detail.set(detail); if (!detail.data.canManage) this.onAccessLost?.() } }
    catch (error) { if (generation === this.generation) { this.detail.set(null); this.onAccessLost?.(); this.error.set(error instanceof Error ? error.message : 'The provider could not be loaded.') } }
  }
  async load(more = false): Promise<void> {
    if (!this.readable()) return
    const generation = ++this.listGeneration
    await this.providers.load(async cursor => {
      try { const page = await this.client.call('listIdentityProviders', { query: { limit: 30, cursor: cursor || undefined } }); if (generation === this.listGeneration) { this.canCreate.set(page.canCreate); this.credentialRefs.set(page.credentialRefs); if (!page.canCreate && !this.id()) this.onAccessLost?.() }; return page }
      catch (error) { if (generation === this.listGeneration) this.accessError(error); throw error }
    }, more)
    if (generation === this.listGeneration && this.providers.error()) { this.canCreate.set(false); this.credentialRefs.set([]) }
  }
  accessError(error: unknown): void {
    if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.generation++; this.loading.set(false); this.clearPrivate(); this.error.set('Access changed. Refresh providers to load current authority.'); void this.workspace.refreshAccess() }
  }
}
