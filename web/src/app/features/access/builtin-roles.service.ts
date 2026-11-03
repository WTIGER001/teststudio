import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { PredefinedRoleRecord } from '../../core/generated/access.types'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { RolePoliciesService } from './role-policies.service'

@Injectable()
export class BuiltinRolesService {
  private readonly client = inject(AccessClientService)
  private readonly policies = inject(RolePoliciesService)
  readonly action = inject(AccessActionService)
  readonly records = new AccessCollection<PredefinedRoleRecord>()
  readonly error = signal('')
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.records.clear(); this.action.reset() }); void this.refresh() }
  async refresh(): Promise<void> {
    this.error.set('')
    const generation = this.generation
    await this.records.load(async () => {
      try { return await this.client.call('listPredefinedRoles', {}) }
      catch (error) { if (generation === this.generation) await this.mutationError(error); throw error }
    })
  }
  private async mutationError(error: unknown): Promise<void> {
    if (error instanceof ApiError && [401, 403].includes(error.status)) { this.records.clear(); this.action.reset(); this.error.set('Access changed. Refresh predefined roles to load current controls.'); await this.policies.mutationError(error) }
  }
  state(row: PredefinedRoleRecord): void {
    if (!row.data.canManage || this.records.loading() || this.records.error() || this.action.busy()) return
    const generation = this.generation, disabled = !row.data.disabled
    this.action.open({ title: disabled ? 'Disable predefined role' : 'Enable predefined role', target: `${row.data.name} (${row.id}) · Entire installation`, returnFocus: 'builtin-roles-heading', impact: 'This changes the role across this installation, including existing project assignments. Definitions and assignments remain retained. Enabling can restore access. The last usable administrator remains protected.', submit: async reason => {
      try { await this.client.call('setPredefinedRoleState', { role: row.id, revision: row.revision, body: { disabled, reason } }) }
      catch (error) { if (generation === this.generation) await this.mutationError(error); throw error }
      if (generation === this.generation) { await this.refresh(); await this.policies.afterMutation() }
      return disabled ? 'Predefined role disabled across this installation. Its assignments remain retained.' : 'Predefined role enabled across this installation. Its retained assignments can grant access again.'
    } })
  }
}
