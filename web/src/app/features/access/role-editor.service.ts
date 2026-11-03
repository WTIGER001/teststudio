import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormControl } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { RolePolicyDetail } from '../../core/generated/access.types'
import { accessFocus } from './access.utils'
import { rolePolicyForm, roleRuleForm } from './role-policy.forms'
import { inheritanceChoices, previewFingerprint, previewRows, rolePolicyDefinition, rolePolicyValidation } from './role-policy.utils'
import { RolePoliciesService } from './role-policies.service'
import { RolePermissionCatalog } from './role-permissions.service'
import { RolePreviewService } from './role-preview.service'

@Injectable()
export class RoleEditorService {
  private readonly client = inject(AccessClientService)
  private readonly catalog = inject(RolePermissionCatalog)
  private readonly destroy = inject(DestroyRef)
  readonly roles = inject(RolePoliciesService)
  readonly preview = inject(RolePreviewService)
  readonly opened = signal(false)
  readonly editing = signal<RolePolicyDetail | null>(null)
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly form = rolePolicyForm()
  private readonly fields = toSignal(this.form.valueChanges, { initialValue: this.form.getRawValue() })
  readonly definition = computed(() => { this.fields(); return rolePolicyDefinition(this.form.getRawValue(), this.roles.projectId(), this.editing()) })
  readonly inheritanceOptions = computed(() => inheritanceChoices(this.roles.policies.items(), this.fields().inherits ?? [], this.definition().id))
  readonly previewCurrent = computed(() => { const fields = this.preview.fields(); return !!this.preview.result() && this.preview.fingerprint() === previewFingerprint(this.definition(), this.editing()?.revision ?? 0, fields.userId ?? '', fields.object ?? '') })
  readonly comparisons = computed(() => previewRows(this.preview.result(), this.preview.showUnchanged()))
  private generation = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.reset())
    this.roles.onAccessLost = () => this.reset()
    this.destroy.onDestroy(() => { this.generation++; this.catalog.reset(); this.preview.reset() })
  }
  private reset(): void { this.generation++; this.opened.set(false); this.editing.set(null); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.catalog.reset(); this.preview.reset(); this.form.reset(); this.form.controls.rules.clear(); this.form.controls.inherits.clear() }
  open(selected?: RolePolicyDetail): void {
    if (selected ? !selected.canManage : !this.roles.canCreate()) return
    this.reset(); this.roles.action.reset(); this.editing.set(selected ?? null); this.form.reset({ id: selected?.id ?? '', name: selected?.data.name ?? '', description: selected?.data.description ?? '', reason: '' })
    for (const rule of selected?.data.rules ?? []) this.form.controls.rules.push(roleRuleForm(rule))
    for (const id of selected?.data.inherits ?? []) this.form.controls.inherits.push(new FormControl(id, { nonNullable: true }))
    this.opened.set(true); this.roles.message.set(''); accessFocus('role-editor-heading')
  }
  close(): void { this.reset(); accessFocus(this.roles.roleId() ? 'role-details-heading' : 'roles-heading') }
  addRule(): void { if (this.form.controls.rules.length < 100) this.form.controls.rules.push(roleRuleForm()) }
  removeRule(index: number): void { this.form.controls.rules.removeAt(index) }
  addInheritance(): void { if (this.form.controls.inherits.length < 16) this.form.controls.inherits.push(new FormControl('', { nonNullable: true })) }
  removeInheritance(index: number): void { this.form.controls.inherits.removeAt(index) }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected || this.busy()) return
    const generation = ++this.generation; this.busy.set(true)
    try { const record = await this.client.call('readRolePolicy', { role: selected.id }); if (generation === this.generation) { if (record.data.projectId !== this.roles.projectId() || !record.canManage) { this.reset(); await this.roles.refresh(); return }; this.open(record) } }
    catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The role could not be reloaded.'); await this.roles.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async inspect(): Promise<void> { const validation = rolePolicyValidation(this.definition(), this.roles.permissions.items(), false); this.error.set(validation); if (!validation && !this.conflict()) { await this.preview.run(this.definition(), this.editing()?.revision ?? 0); if (this.preview.conflict()) this.conflict.set(true) } }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict() || this.preview.busy()) return
    const body = this.definition(), validation = rolePolicyValidation(body, this.roles.permissions.items()); this.error.set(validation); if (validation) return
    const selected = this.editing(), generation = ++this.generation; this.busy.set(true)
    try {
      if (selected) await this.client.call('editRolePolicy', { role: selected.id, revision: selected.revision, body })
      else await this.client.call('createRolePolicy', { body })
      if (generation !== this.generation) return
      this.close(); this.roles.message.set('Role policy saved. Current assignments use the new definition.'); await this.roles.afterMutation()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The policy save could not be confirmed. Refresh state before retrying.'); this.conflict.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500); await this.roles.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
