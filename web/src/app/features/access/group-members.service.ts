import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { MembershipRecord } from '../../core/generated/access.types'
import { accessFocus, canEditMembership, expiryForm, membershipExpiry, membershipValidation } from './access.utils'
import { GroupsService } from './groups.service'

@Injectable()
export class GroupMembersService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly groups = inject(GroupsService)
  readonly opened = signal(false)
  readonly editing = signal<MembershipRecord | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({ user: new FormControl('', { nonNullable: true }), expiry: new FormControl('', { nonNullable: true }) })
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.editing.set(null); this.busy.set(false); this.error.set('') })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  open(item?: MembershipRecord): void {
    if (!canEditMembership(this.groups.group()?.data ?? null, this.groups.rights().manageMembers, item?.data)) return
    this.generation++; this.editing.set(item ?? null); this.form.reset({ user: item?.data.userId ?? '', expiry: expiryForm(item?.data.expiresAt) }); this.opened.set(true); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.groups.message.set(''); accessFocus('member-editor-heading')
  }
  close(): void { this.generation++; this.opened.set(false); this.busy.set(false); accessFocus('group-members-heading') }
  async reload(): Promise<void> {
    const item = this.editing(); if (!item || this.busy()) return
    const generation = ++this.generation; this.busy.set(true)
    try {
      const stored = await this.client.call('readGroupMember', { group: this.groups.groupId(), user: item.data.userId, query: { source: 'local' } })
      if (generation === this.generation) this.open(stored)
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The stored membership could not be loaded.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict()) return
    const value = this.form.getRawValue(), expiry = membershipExpiry(value.expiry, Date.now()), validation = membershipValidation(value.user.trim()) || expiry.error
    this.error.set(validation); if (validation) return
    const item = this.editing(), generation = ++this.generation
    this.busy.set(true)
    try {
      await this.client.call('saveLocalMembership', { group: this.groups.groupId(), user: value.user.trim(), revision: item?.revision ?? 0, body: { expiresAt: expiry.expiresAt } })
      if (generation !== this.generation) return
      this.close(); this.groups.message.set('Local membership saved. Role assignments determine access.'); await this.groups.loadMembers()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The membership could not be saved.'); this.conflict.set(error instanceof ApiError && error.status === 409) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  remove(item: MembershipRecord): void {
    if (!canEditMembership(this.groups.group()?.data ?? null, this.groups.rights().manageMembers, item.data)) return
    const group = this.groups.groupId(), generation = this.generation
    this.groups.action.open({ title: 'Remove local membership', target: item.data.name, returnFocus: 'group-members-heading', impact: 'Remove this local group membership. Other groups and direct assignments remain unchanged.',
      submit: async reason => {
        await this.client.call('removeLocalMembership', { group, user: item.data.userId, revision: item.revision, body: { reason } })
        if (generation === this.generation) { this.close(); await this.groups.loadMembers() }
        return 'Local membership removed.'
      } })
  }
}
