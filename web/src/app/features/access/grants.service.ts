import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { GrantRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { grantReadRights, grantTarget } from './grant.utils'

@Injectable()
export class GrantsService {
  private readonly client = inject(AccessClientService)
  private readonly workspace = inject(WorkspaceService)
  private readonly destroy = inject(DestroyRef)
  readonly action = inject(AccessActionService)
  readonly projectId = signal('')
  readonly readable = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly grants = new AccessCollection<GrantRecord>()
  private readonly serverCanCreate = signal(false)
  readonly canCreate = computed(() => this.serverCanCreate() && this.readable() && !this.loading() && !this.grants.loading())
  onAccessLost: (() => void) | undefined
  private generation = 0
  private scopeGeneration = 0
  private listGeneration = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.scopeGeneration++; this.generation++; this.clearPrivate(); this.readable.set(false); this.error.set(''); this.message.set(''); this.projectId.set(params.get('projectId') ?? ''); void this.refresh() })
    this.destroy.onDestroy(() => { this.scopeGeneration++; this.generation++; this.clearPrivate() })
  }
  private clearPrivate(): void { this.listGeneration++; this.grants.clear(); this.serverCanCreate.set(false); this.action.reset(); this.onAccessLost?.() }
  async refresh(): Promise<void> {
    const generation = ++this.generation, project = this.projectId(); this.loading.set(true); this.error.set('')
    try {
      const readable = grantReadRights(await this.workspace.readCapabilities(project)); if (generation !== this.generation) return
      this.readable.set(readable)
      if (!readable) { this.clearPrivate(); this.error.set('Membership and permission read authority is required for this scope.'); return }
      await this.load()
    } catch (error) { if (generation === this.generation) { this.clearPrivate(); this.readable.set(false); this.error.set(error instanceof Error ? error.message : 'Permission access could not be loaded.') } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async load(more = false): Promise<void> {
    if (!this.readable()) return
    const generation = ++this.listGeneration, project = this.projectId()
    await this.grants.load(async cursor => {
      try {
        const page = project ? await this.client.call('listProjectGrants', { project, query: { limit: 30, cursor: cursor || undefined } }) : await this.client.call('listInstallationGrants', { query: { limit: 30, cursor: cursor || undefined } })
        if (generation === this.listGeneration) this.serverCanCreate.set(page.canCreate)
        return page
      } catch (error) { if (generation === this.listGeneration) this.mutationError(error); throw error }
    }, more)
    if (generation === this.listGeneration && this.grants.error()) this.serverCanCreate.set(false)
  }
  mutationError(error: unknown): void {
    if (!(error instanceof ApiError) || ![401, 403].includes(error.status)) return
    this.generation++; this.loading.set(false); this.clearPrivate(); this.readable.set(false); this.error.set('Access changed. Refresh permissions to load your current access.'); void this.workspace.refreshAccess()
  }
  async afterMutation(): Promise<void> { await this.refresh(); await this.workspace.refreshAccess() }
  remove(row: GrantRecord): void {
    if (!row.data.canRemove || this.action.busy()) return
    const generation = this.scopeGeneration, project = this.projectId(); this.message.set('')
    this.action.open({ title: 'Remove direct grant', target: grantTarget(row), returnFocus: 'grants-heading', impact: 'Remove this exact allow or deny rule. Other grants and role assignments remain effective. Removing a deny can restore access. The last usable administrator remains protected.', submit: async reason => {
      try {
        if (project) await this.client.call('removeProjectGrant', { project, grant: row.id, revision: row.revision, body: { reason } })
        else await this.client.call('removeInstallationGrant', { grant: row.id, revision: row.revision, body: { reason } })
      } catch (error) { if (generation === this.scopeGeneration) { this.mutationError(error) }; throw error }
      if (generation === this.scopeGeneration) await this.afterMutation()
      return 'Direct grant removed. Its revision history remains retained.'
    } })
  }
}
