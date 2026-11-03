import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { AssignmentRecord, Role } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { assignableRoles, assignmentRights, assignmentTarget } from './assignment.utils'

@Injectable()
export class AssignmentsService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  private readonly workspace = inject(WorkspaceService)
  readonly action = inject(AccessActionService)
  readonly projectId = signal('')
  readonly rights = signal(assignmentRights({}))
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly assignments = new AccessCollection<AssignmentRecord>()
  readonly roles = new AccessCollection<Role>()
  readonly customRoles = new AccessCollection<Role>()
  readonly roleChoices = computed(() => [...this.roles.items(), ...this.customRoles.items()])
  readonly rolesLoading = computed(() => this.roles.loading() || this.customRoles.loading())
  readonly rolesError = computed(() => this.roles.error() || this.customRoles.error())
  readonly canAdd = computed(() => this.rights().manage && !this.loading() && !this.rolesLoading() && !this.rolesError() && assignableRoles(this.roleChoices()).length > 0)
  private scopeGeneration = 0
  private refreshGeneration = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.scopeGeneration++; this.refreshGeneration++; this.action.reset(); this.assignments.clear(); this.roles.clear(); this.customRoles.clear(); this.rights.set(assignmentRights({})); this.message.set(''); this.error.set('')
      this.projectId.set(params.get('projectId') ?? ''); void this.refresh()
    })
    this.destroy.onDestroy(() => { this.scopeGeneration++; this.refreshGeneration++; this.action.reset(); this.assignments.clear(); this.roles.clear(); this.customRoles.clear() })
  }
  async refresh(): Promise<void> {
    const generation = ++this.refreshGeneration, project = this.projectId()
    this.loading.set(true); this.error.set('')
    try {
      const rights = assignmentRights(await this.workspace.readCapabilities(project))
      if (generation !== this.refreshGeneration) return
      this.rights.set(rights)
      if (!rights.read) { this.assignments.clear(); this.roles.clear(); this.customRoles.clear(); this.error.set('Membership and permission read authority is required for this scope.'); return }
      await Promise.all([this.loadAssignments(), this.loadRoles()])
    } catch (error) {
      if (generation === this.refreshGeneration) { this.assignments.clear(); this.roles.clear(); this.customRoles.clear(); this.rights.set(assignmentRights({})); this.error.set(error instanceof Error ? error.message : 'Access permissions could not be loaded.') }
    } finally { if (generation === this.refreshGeneration) this.loading.set(false) }
  }
  async loadAssignments(more = false): Promise<void> {
    if (!this.rights().read) return
    const project = this.projectId()
    await this.assignments.load(cursor => project
      ? this.client.call('listProjectAssignments', { project, query: { limit: 30, cursor: cursor || undefined } })
      : this.client.call('listInstallationAssignments', { query: { limit: 30, cursor: cursor || undefined } }), more)
  }
  async loadRoles(): Promise<void> {
    if (!this.rights().readRoles) { this.roles.clear(); this.customRoles.clear(); this.roles.error.set('Role read authority is required to inspect choices or add access.'); return }
    const project = this.projectId()
    await Promise.all([this.roles.load(() => this.client.call('listRoles', { query: { project } })), this.loadCustomRoles()])
  }
  async loadCustomRoles(more = false): Promise<void> {
    if (!this.rights().readRoles) return
    const project = this.projectId()
    await this.customRoles.load(cursor => this.client.call('customRoleChoices', { query: { project, limit: 30, cursor: cursor || undefined } }), more)
  }
  mutationError(error: unknown): void {
    if (!(error instanceof ApiError) || ![401, 403].includes(error.status)) return
    this.refreshGeneration++; this.loading.set(false); this.assignments.clear(); this.roles.clear(); this.customRoles.clear(); this.rights.set(assignmentRights({})); this.error.set('Access changed. Refresh access to load your current permissions.'); void this.workspace.refreshAccess()
  }
  remove(row: AssignmentRecord): void {
    if (!this.rights().manage || !row.data.canRevoke || this.action.busy()) return
    const project = this.projectId(), generation = this.scopeGeneration
    this.message.set('')
    this.action.open({ title: 'Remove role assignment', target: assignmentTarget(row), returnFocus: 'assignments-heading',
      impact: 'Remove this exact role in the displayed scope. Access can remain through other roles or groups. The last usable installation administrator remains protected.',
      submit: async reason => {
        const params = { kind: row.data.principalKind, principal: row.data.principalId, role: row.data.roleId, revision: row.revision, body: { reason } }
        try {
          if (project) await this.client.call('revokeProjectAssignment', { ...params, project })
          else await this.client.call('revokeInstallationAssignment', params)
        } catch (error) { if (generation === this.scopeGeneration) this.mutationError(error); throw error }
        if (generation === this.scopeGeneration) { await this.refresh(); await this.workspace.refreshAccess() }
        return 'Role assignment removed. Other assignments and membership records remain available.'
      } })
  }
}
