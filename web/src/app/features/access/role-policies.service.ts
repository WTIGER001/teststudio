import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { PermissionChoice, Role, RolePolicyDetail, RolePolicyRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { roleRights, roleRoute, roleStateTarget } from './role-policy.utils'

@Injectable()
export class RolePoliciesService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  private readonly workspace = inject(WorkspaceService)
  readonly action = inject(AccessActionService)
  readonly projectId = signal('')
  readonly roleId = signal('')
  readonly detail = signal<RolePolicyDetail | null>(null)
  readonly rights = signal(roleRights({}))
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly canCreate = signal(false)
  readonly policies = new AccessCollection<RolePolicyRecord>()
  readonly builtin = new AccessCollection<Role>()
  readonly permissions = new AccessCollection<PermissionChoice>()
  readonly listRoute = computed(() => roleRoute(this.projectId()))
  onAccessLost: (() => void) | undefined
  private scopeGeneration = 0
  private generation = 0
  private policyGeneration = 0
  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => { this.scopeGeneration++; this.generation++; this.policyGeneration++; this.action.reset(); this.detail.set(null); this.policies.clear(); this.builtin.clear(); this.permissions.clear(); this.canCreate.set(false); this.rights.set(roleRights({})); this.error.set(''); this.message.set(''); this.projectId.set(params.get('projectId') ?? ''); this.roleId.set(params.get('roleId') ?? ''); void this.refresh() })
    this.destroy.onDestroy(() => { this.scopeGeneration++; this.generation++; this.policyGeneration++; this.action.reset(); this.policies.clear(); this.permissions.clear(); this.builtin.clear() })
  }
  async refresh(): Promise<void> {
    const generation = ++this.generation, project = this.projectId(); this.loading.set(true); this.error.set('')
    try {
      const rights = roleRights(await this.workspace.readCapabilities(project)); if (generation !== this.generation) return
      this.rights.set(rights)
      if (!rights.read) { this.clearPrivate(); this.error.set('Role read authority is required for this scope.'); return }
      await Promise.all([this.loadPolicies(), this.builtin.load(() => this.client.call('listRoles', { query: { project } })), this.permissions.load(() => this.client.call('permissionCatalog', { query: { project } })), this.roleId() ? this.readDetail(generation) : Promise.resolve()])
    } catch (error) { if (generation === this.generation) { this.clearPrivate(); this.error.set(error instanceof Error ? error.message : 'Role permissions could not be loaded.') } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  private clearPrivate(): void { this.policyGeneration++; this.action.reset(); this.detail.set(null); this.policies.clear(); this.builtin.clear(); this.permissions.clear(); this.canCreate.set(false); this.onAccessLost?.() }
  private async readDetail(generation: number): Promise<void> {
    try { const detail = await this.client.call('readRolePolicy', { role: this.roleId() }); if (generation !== this.generation) return; if (detail.data.projectId !== this.projectId()) { this.detail.set(null); this.error.set('This role belongs to another ownership scope.'); this.action.reset(); this.onAccessLost?.(); return }; this.detail.set(detail) }
    catch (error) { if (generation === this.generation) { this.detail.set(null); this.error.set(error instanceof Error ? error.message : 'The role could not be loaded.'); if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.action.reset(); this.onAccessLost?.() } } }
  }
  async loadPolicies(more = false): Promise<void> {
    if (!this.rights().read) return
    const project = this.projectId(), generation = ++this.policyGeneration
    await this.policies.load(async cursor => { const page = await this.client.call('listRolePolicies', { query: { project, limit: 30, cursor: cursor || undefined } }); if (generation === this.policyGeneration) this.canCreate.set(page.canCreate); return page }, more)
    if (generation === this.policyGeneration && this.policies.error()) this.canCreate.set(false)
  }
  async afterMutation(): Promise<void> { await this.refresh(); await this.workspace.refreshAccess() }
  async mutationError(error: unknown): Promise<void> { if (error instanceof ApiError && [401, 403].includes(error.status)) { await this.refresh(); await this.workspace.refreshAccess() } }
  state(disabled: boolean): void {
    const selected = this.detail(); if (!selected?.canManage || this.action.busy()) return
    const generation = this.scopeGeneration
    this.action.open({ title: disabled ? 'Disable role' : 'Enable role', target: roleStateTarget(selected), returnFocus: 'role-details-heading', impact: 'This changes effective access for assigned users, groups, and parent roles. The definition and assignments remain retained. The last usable administrator remains protected.', submit: async reason => {
      try { await this.client.call('editRolePolicy', { role: selected.id, revision: selected.revision, body: { ...selected.data, disabled, reason } }) }
      catch (error) { if (generation === this.scopeGeneration) await this.mutationError(error); throw error }
      if (generation === this.scopeGeneration) await this.afterMutation()
      return disabled ? 'Role disabled. Its rules and inherited access are withdrawn.' : 'Role enabled after policy validation.'
    } })
  }
}
