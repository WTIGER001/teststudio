import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { UserRecord } from '../../core/generated/access.types'
import { UsersService } from './users.service'
import { accessFocus, userDefinition, userValidation } from './access.utils'

@Injectable()
export class UserEditorService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly users = inject(UsersService)
  readonly opened = signal(false)
  readonly editing = signal<UserRecord | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({ id: new FormControl('', { nonNullable: true }), name: new FormControl('', { nonNullable: true }), email: new FormControl('', { nonNullable: true }) })
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.editing.set(null); this.busy.set(false); this.error.set('') })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  open(item?: UserRecord): void {
    this.generation++; this.editing.set(item ?? null); this.form.reset({ id: item?.id ?? '', name: item?.data.name ?? '', email: item?.data.email ?? '' }); this.opened.set(true); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.users.message.set(''); accessFocus('user-editor-heading')
  }
  close(): void { this.generation++; this.opened.set(false); this.busy.set(false); accessFocus(this.users.userId() ? 'user-details-heading' : 'users-heading') }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected || this.busy()) return
    const generation = ++this.generation; this.busy.set(true)
    try { const stored = await this.client.call('readUser', { user: selected.id }); if (generation === this.generation) this.open(stored) }
    catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The stored profile could not be loaded.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict()) return
    const definition = userDefinition(this.form.getRawValue()), validation = userValidation(definition)
    this.error.set(validation); if (validation) return
    const selected = this.editing(), generation = ++this.generation
    this.busy.set(true)
    try {
      const saved = selected ? await this.client.call('editUser', { user: selected.id, revision: selected.revision, body: definition }) : await this.client.call('createUser', { body: definition })
      if (generation !== this.generation) return
      this.close(); this.users.message.set(selected ? 'User profile saved.' : 'User profile registered. No identity link or project access was added.')
      if (this.users.userId()) await this.users.refreshDetails(); else await this.users.loadUsers()
      if (saved.id === this.users.session.user()?.id) await this.users.session.refresh()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The profile could not be saved.'); this.conflict.set(error instanceof ApiError && error.status === 409) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
