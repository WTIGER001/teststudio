import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { GroupRecord, MembershipRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { accessRights } from './access.utils'

@Injectable()
export class GroupsService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly action = inject(AccessActionService)
  readonly projectId = signal('')
  readonly groupId = signal('')
  readonly group = signal<GroupRecord | null>(null)
  readonly rights = signal(accessRights({}, ''))
  readonly listRoute = computed(() => this.projectId() ? ['/projects', this.projectId(), 'groups'] : ['/settings', 'groups'])
  readonly archived = signal(false)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly groups = new AccessCollection<GroupRecord>()
  readonly members = new AccessCollection<MembershipRecord>()
  private generation = 0
  private profileGeneration = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.generation++; this.profileGeneration++; this.action.reset(); this.groups.clear(); this.members.clear(); this.group.set(null); this.error.set(''); this.message.set(''); this.rights.set(accessRights({}, ''))
      this.projectId.set(params.get('projectId') ?? ''); this.groupId.set(params.get('groupId') ?? ''); void this.initialize()
    })
    this.destroy.onDestroy(() => { this.generation++; this.profileGeneration++; this.action.reset(); this.groups.clear(); this.members.clear() })
  }
  private async initialize(): Promise<void> {
    const generation = this.generation
    this.loading.set(true)
    try {
      const rights = accessRights(await this.workspace.readCapabilities(this.projectId()), this.projectId())
      if (generation !== this.generation) return
      this.rights.set(rights)
      if (!rights.readGroups) { this.error.set('Group read permission is required for this ownership scope.'); return }
      if (this.groupId()) await this.refreshDetails(); else await this.loadGroups()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'Group permissions could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async loadGroups(more = false): Promise<void> {
    const project = this.projectId(), archived = this.archived()
    await this.groups.load(cursor => this.client.call('listGroups', { query: { project, archived, limit: 30, cursor: cursor || undefined } }), more)
  }
  async readGroup(): Promise<GroupRecord | null> {
    const generation = ++this.profileGeneration, group = this.groupId()
    this.loading.set(true); this.error.set('')
    try {
      const record = await this.client.call('readGroup', { group })
      if (generation !== this.profileGeneration) return null
      if ((record.data.projectId ?? '') !== this.projectId()) { this.group.set(null); this.members.clear(); this.error.set('This group belongs to another ownership scope. Open it from the correct group directory.'); return null }
      this.group.set(record); return record
    } catch (error) {
      if (generation === this.profileGeneration) { this.error.set(error instanceof Error ? error.message : 'The group could not be loaded.'); if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.group.set(null); this.members.clear() } }
      return null
    } finally { if (generation === this.profileGeneration) this.loading.set(false) }
  }
  async refreshDetails(): Promise<void> { if (await this.readGroup()) await this.loadMembers() }
  async loadMembers(more = false): Promise<void> {
    if (!this.rights().readMembers) { this.members.clear(); this.members.error.set('Membership read permission is required.'); return }
    const group = this.groupId()
    await this.members.load(cursor => this.client.call('listGroupMembers', { group, query: { limit: 30, cursor: cursor || undefined } }), more)
  }
  state(archived: boolean): void {
    const selected = this.group(); if (!selected) return
    const generation = this.generation
    this.action.open({ title: archived ? 'Archive group' : 'Restore group', target: selected.data.name, returnFocus: 'group-details-heading',
      impact: archived ? 'Remove effective membership access while preserving membership and assignment records. The last usable installation administrator remains protected.' : 'Reactivate retained, unexpired membership and its assigned role access. Review the group before restoring it.',
      submit: async reason => {
        await this.client.call(archived ? 'archiveGroup' : 'restoreGroup', { group: selected.id, revision: selected.revision, body: { reason } })
        if (generation === this.generation) await this.refreshDetails()
        return archived ? 'Group archived. Membership and assignment records remain available.' : 'Group restored. Retained, unexpired memberships are active again.'
      } })
  }
}
