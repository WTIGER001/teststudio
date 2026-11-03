import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { GroupRecord } from '../../core/generated/access.types'
import { GroupsService } from './groups.service'
import { accessFocus, groupDefinition, groupValidation } from './access.utils'

@Injectable()
export class GroupEditorService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  readonly groups = inject(GroupsService)
  readonly opened = signal(false)
  readonly editing = signal<GroupRecord | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({ id: new FormControl('', { nonNullable: true }), name: new FormControl('', { nonNullable: true }), description: new FormControl('', { nonNullable: true }) })
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.editing.set(null); this.busy.set(false); this.error.set('') })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  open(item?: GroupRecord): void {
    this.generation++; this.editing.set(item ?? null); this.form.reset({ id: item?.id ?? '', name: item?.data.name ?? '', description: item?.data.description ?? '' }); this.opened.set(true); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.groups.message.set(''); accessFocus('group-editor-heading')
  }
  close(): void { this.generation++; this.opened.set(false); this.busy.set(false); accessFocus(this.groups.groupId() ? 'group-details-heading' : 'groups-heading') }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected || this.busy()) return
    const generation = ++this.generation; this.busy.set(true)
    try { const record = await this.client.call('readGroup', { group: selected.id }); if (generation === this.generation) this.open(record) }
    catch (error) { if (generation === this.generation) this.error.set(error instanceof Error ? error.message : 'The stored group could not be loaded.') }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict()) return
    const definition = groupDefinition({ ...this.form.getRawValue(), projectId: this.groups.projectId() }), validation = groupValidation(definition)
    this.error.set(validation); if (validation) return
    const selected = this.editing(), generation = ++this.generation
    this.busy.set(true)
    try {
      if (selected) await this.client.call('editGroup', { group: selected.id, revision: selected.revision, body: definition })
      else await this.client.call('createGroup', { body: definition })
      if (generation !== this.generation) return
      this.close(); this.groups.message.set('Group configuration saved. Ownership scope remains permanent.')
      if (this.groups.groupId()) await this.groups.refreshDetails(); else await this.groups.loadGroups()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The group could not be saved.'); this.conflict.set(error instanceof ApiError && error.status === 409) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
