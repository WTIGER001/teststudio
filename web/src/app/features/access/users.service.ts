import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { ExternalIdentityRecord, MembershipRecord, SessionRecord, UserRecord } from '../../core/generated/access.types'
import { SessionService } from '../../core/session.service'
import { WorkspaceService } from '../../core/workspace.service'
import { AccessActionService } from './access-action.service'
import { AccessCollection } from './access-collection'
import { accessRights } from './access.utils'

@Injectable()
export class UsersService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly workspace = inject(WorkspaceService)
  readonly session = inject(SessionService)
  readonly action = inject(AccessActionService)
  readonly userId = signal('')
  readonly profile = signal<UserRecord | null>(null)
  readonly loading = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly manage = signal(false)
  readonly users = new AccessCollection<UserRecord>()
  readonly identities = new AccessCollection<ExternalIdentityRecord>()
  readonly memberships = new AccessCollection<MembershipRecord>()
  readonly sessions = new AccessCollection<SessionRecord>()
  private generation = 0
  private profileGeneration = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(params => {
      this.generation++; this.profileGeneration++; this.action.reset(); this.users.clear(); this.clearDetails(); this.message.set(''); this.error.set(''); this.manage.set(false)
      this.userId.set(params.get('userId') ?? ''); void this.initialize()
    })
    this.destroy.onDestroy(() => { this.generation++; this.profileGeneration++; this.action.reset(); this.users.clear(); this.clearDetails() })
  }
  private clearDetails(): void { this.profile.set(null); this.identities.clear(); this.memberships.clear(); this.sessions.clear() }
  private async initialize(): Promise<void> {
    const generation = this.generation
    this.loading.set(true)
    try {
      const rights = accessRights(await this.workspace.readCapabilities(''), '')
      if (generation !== this.generation) return
      this.manage.set(rights.manageUsers)
      if (!rights.readUsers) { this.error.set('Installation user read permission is required.'); return }
      if (this.userId()) await this.refreshDetails(); else await this.loadUsers()
    } catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'User permissions could not be loaded.') }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async loadUsers(more = false): Promise<void> { await this.users.load(cursor => this.client.call('listUsers', { query: { limit: 30, cursor: cursor || undefined } }), more) }
  async readProfile(): Promise<UserRecord | null> {
    const generation = ++this.profileGeneration, user = this.userId()
    this.loading.set(true); this.error.set('')
    try {
      const profile = await this.client.call('readUser', { user })
      if (generation !== this.profileGeneration) return null
      this.profile.set(profile); return profile
    } catch (error) {
      if (generation === this.profileGeneration) { this.error.set(error instanceof Error ? error.message : 'The user could not be loaded.'); if (error instanceof ApiError && [401, 403, 404].includes(error.status)) this.clearDetails() }
      return null
    } finally { if (generation === this.profileGeneration) this.loading.set(false) }
  }
  async refreshDetails(): Promise<void> {
    const profile = await this.readProfile(); if (!profile) return
    await Promise.all([this.loadIdentities(), this.loadMemberships(), this.loadSessions()])
  }
  async loadIdentities(more = false): Promise<void> { const user = this.userId(); await this.identities.load(cursor => this.client.call('listUserIdentities', { user, query: { limit: 30, cursor: cursor || undefined } }), more) }
  async loadMemberships(more = false): Promise<void> { const user = this.userId(); await this.memberships.load(cursor => this.client.call('listUserMemberships', { user, query: { limit: 30, cursor: cursor || undefined } }), more) }
  async loadSessions(more = false): Promise<void> { const user = this.userId(); await this.sessions.load(cursor => this.client.call('listUserSessions', { user, query: { limit: 30, cursor: cursor || undefined } }), more) }
  state(disabled: boolean): void {
    const selected = this.profile(); if (!selected) return
    const generation = this.generation
    this.action.open({ title: disabled ? 'Disable user' : 'Restore user', target: selected.data.name, returnFocus: 'user-details-heading',
      impact: disabled ? 'Disable sign-in and invalidate earlier sessions. The last usable installation administrator cannot be disabled.' : 'Restore sign-in eligibility. Earlier sessions remain invalid; a fresh sign-in is required.',
      submit: async reason => {
        await this.client.call(disabled ? 'disableUser' : 'restoreUser', { user: selected.id, revision: selected.revision, body: { reason } })
        if (generation === this.generation) { await this.refreshDetails(); if (selected.id === this.session.user()?.id) await this.session.refresh() }
        return disabled ? 'User disabled. Earlier sessions are invalid.' : 'User restored. Earlier sessions remain invalid.'
      } })
  }
  revoke(selected: SessionRecord): void {
    const user = this.userId(), generation = this.generation
    this.action.open({ title: 'Revoke session', target: selected.id, returnFocus: 'user-sessions-heading',
      impact: selected.data.current ? 'This is your current session. Revoking it signs this browser out.' : 'The selected session loses access on its next request. Other sessions remain active.',
      submit: async reason => {
        await this.client.call('revokeUserSession', { user, session: selected.id, revision: selected.revision, body: { reason } })
        if (generation === this.generation) { if (selected.data.current) await this.session.refresh(); else await this.loadSessions() }
        return 'Session revoked.'
      } })
  }
}
