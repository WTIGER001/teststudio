import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { GrantRecord, PermissionChoice } from '../../core/generated/access.types'
import { accessFocus } from './access.utils'
import { grantDefinition, grantFields, grantUpdate, grantValidation } from './grant.utils'
import { GrantsService } from './grants.service'
import { roleObjectValid } from './role-policy.utils'
import { RolePermissionCatalog } from './role-permissions.service'

@Injectable()
export class GrantEditorService {
  private readonly client = inject(AccessClientService)
  private readonly catalog = inject(RolePermissionCatalog)
  private readonly destroy = inject(DestroyRef)
  readonly access = inject(GrantsService)
  readonly opened = signal(false)
  readonly selected = signal<GrantRecord | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly permissions = signal<PermissionChoice[]>([])
  readonly permissionsLoading = signal(false)
  readonly permissionsError = signal('')
  readonly loadedObject = signal<string | null>(null)
  readonly form = new FormGroup({ principalKind: new FormControl<'user' | 'group'>('user', { nonNullable: true }), principalId: new FormControl('', { nonNullable: true }), permission: new FormControl('', { nonNullable: true }), effect: new FormControl<'allow' | 'deny'>('allow', { nonNullable: true }), object: new FormControl('', { nonNullable: true }), expiresAt: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  private readonly object = toSignal(this.form.controls.object.valueChanges, { initialValue: '' })
  readonly permissionsCurrent = computed(() => this.loadedObject() !== null && this.loadedObject() === this.object().trim() && !this.permissionsLoading() && !this.permissionsError())
  private generation = 0
  private permissionGeneration = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.reset())
    this.access.onAccessLost = () => this.reset()
    this.destroy.onDestroy(() => this.reset())
  }
  reset(): void { this.generation++; this.permissionGeneration++; this.opened.set(false); this.selected.set(null); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.permissions.set([]); this.loadedObject.set(null); this.permissionsLoading.set(false); this.permissionsError.set(''); this.form.reset(grantFields()); this.catalog.reset() }
  open(row?: GrantRecord): void {
    if (row ? !row.data.canUpdate : !this.access.canCreate()) return
    this.reset(); this.selected.set(row ?? null); this.form.reset(grantFields(row)); this.opened.set(true); this.access.action.reset(); this.access.message.set(''); accessFocus('grant-editor-heading')
    if (!row) void this.loadPermissions()
  }
  close(): void { this.reset(); accessFocus('grants-heading') }
  remove(row: GrantRecord): void { this.close(); this.access.remove(row) }
  async loadPermissions(): Promise<void> {
    const object = this.form.controls.object.value.trim(), project = this.access.projectId()
    if (!roleObjectValid(object)) { this.permissionsError.set('Enter an exact supported object or leave the scope empty.'); return }
    const generation = ++this.permissionGeneration; this.permissionsLoading.set(true); this.permissionsError.set(''); this.permissions.set([]); this.loadedObject.set(null)
    try { const choices = await this.catalog.read(project, object, true); if (generation === this.permissionGeneration) { this.permissions.set(choices); this.loadedObject.set(object) } }
    catch (error) { if (generation === this.permissionGeneration) { this.permissionsError.set(error instanceof Error ? error.message : 'Permission choices could not be loaded.'); this.access.mutationError(error) } }
    finally { if (generation === this.permissionGeneration) this.permissionsLoading.set(false) }
  }
  async reload(): Promise<void> {
    const row = this.selected(); if (!row || this.busy()) return
    const generation = ++this.generation, project = this.access.projectId(); this.busy.set(true)
    try {
      const fresh = project ? await this.client.call('readProjectGrant', { project, grant: row.id }) : await this.client.call('readInstallationGrant', { grant: row.id })
      if (generation !== this.generation) return
      if (!fresh.data.canUpdate) { this.reset(); await this.access.refresh(); return }
      this.open(fresh)
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The grant could not be reloaded.'); if (error instanceof ApiError && error.status === 404) { this.reset(); await this.access.refresh(); this.access.message.set('The selected grant is no longer available.') }; this.access.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    const selected = this.selected(); if (this.busy() || this.conflict() || (selected ? !selected.data.canUpdate : !this.access.canCreate() || !this.permissionsCurrent())) return
    const fields = this.form.getRawValue(), validation = grantValidation(fields, selected, this.permissions()); this.error.set(validation); if (validation) return
    const generation = ++this.generation, project = this.access.projectId(); this.busy.set(true)
    try {
      if (selected) {
        const params = { grant: selected.id, revision: selected.revision, body: grantUpdate(fields) }
        if (project) await this.client.call('updateProjectGrant', { ...params, project }); else await this.client.call('updateInstallationGrant', params)
      } else {
        const body = grantDefinition(fields)
        if (project) await this.client.call('createProjectGrant', { project, body }); else await this.client.call('createInstallationGrant', { body })
      }
      if (generation !== this.generation) return
      this.close(); this.access.message.set(selected ? 'Grant expiry saved. Current access uses the new expiry.' : 'Direct grant created. Current permissions apply without another sign-in.'); await this.access.afterMutation()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof ApiError ? error.message : 'The save could not be confirmed. Refresh before retrying.'); this.conflict.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500); this.access.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
