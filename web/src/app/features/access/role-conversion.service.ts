import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { AssignmentRecord, RoleConversionRecord } from '../../core/generated/access.types'
import { WorkspaceService } from '../../core/workspace.service'
import { AssignmentsService } from './assignments.service'
import { accessFocus, reasonValidation } from './access.utils'

@Injectable()
export class RoleConversionService {
  private readonly client = inject(AccessClientService)
  private readonly workspace = inject(WorkspaceService)
  private readonly destroy = inject(DestroyRef)
  readonly access = inject(AssignmentsService)
  readonly principal = signal<AssignmentRecord | null>(null)
  readonly receipt = signal<RoleConversionRecord | null>(null)
  readonly busy = signal(false)
  readonly uncertain = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({ roleId: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  readonly rollbackReason = new FormControl('', { nonNullable: true })
  private generation = 0

  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.reset())
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { if (!this.receipt()?.data.applied) this.receipt.set(null) })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  private reset(): void { this.generation++; this.principal.set(null); this.receipt.set(null); this.error.set(''); this.busy.set(false); this.uncertain.set(false); this.form.reset(); this.rollbackReason.reset() }
  open(row: AssignmentRecord): void {
    if (!this.access.rights().manage || !this.access.projectId()) return
    if (this.dirty() && !window.confirm('Discard this unfinished role conversion?')) return
    this.reset(); this.principal.set(row); accessFocus('conversion-heading')
  }
  close(force = false): void { if (!force && this.dirty() && !window.confirm('Discard this unfinished role conversion?')) return; this.reset(); accessFocus('assignments-heading') }
  dirty(): boolean { return !!this.principal() && (this.receipt()?.data.applied ? this.rollbackReason.dirty : this.form.dirty || !!this.receipt()) }
  roleNames(ids: string[]): string { return ids.map(id => this.access.roleChoices().find(role => role.id === id)?.name || id).join(', ') }
  async preview(): Promise<void> {
    const row = this.principal(), value = this.form.getRawValue()
    if (!row || this.busy() || this.uncertain()) return
    const validation = reasonValidation(value.reason)
    if (validation || !this.access.standardChoices().some(role => role.id === value.roleId && role.delegatable && !role.disabled)) { this.error.set(validation || 'Select a standard role you can assign.'); return }
    await this.request(async () => {
      const project = this.access.projectId(), application = this.access.applicationId(), body = { principalKind: row.data.principalKind, principalId: row.data.principalId, roleId: value.roleId, reason: value.reason.trim() }
      return application ? this.client.call('previewApplicationRoleConversion', { project, application, body }) : this.client.call('previewProjectRoleConversion', { project, body })
    })
  }
  async apply(): Promise<void> {
    const receipt = this.receipt()
    if (!receipt || receipt.data.applied || this.busy() || this.uncertain()) return
    await this.request(async () => {
      const project = this.access.projectId(), application = this.access.applicationId(), params = { project, conversion: receipt.id, revision: receipt.revision }
      return application ? this.client.call('applyApplicationRoleConversion', { ...params, application }) : this.client.call('applyProjectRoleConversion', params)
    }, true)
  }
  async reload(): Promise<void> {
    const receipt = this.receipt()
    if (!receipt || this.busy()) return
    await this.request(() => this.access.applicationId() ? this.client.call('readApplicationRoleConversion', { project: this.access.projectId(), application: this.access.applicationId(), conversion: receipt.id }) : this.client.call('readProjectRoleConversion', { project: this.access.projectId(), conversion: receipt.id }))
  }
  async rollback(): Promise<void> {
    const receipt = this.receipt(), reason = this.rollbackReason.value.trim()
    if (!receipt?.data.applied || this.busy()) return
    const validation = reasonValidation(reason); this.error.set(validation); if (validation) return
    await this.request(() => this.access.applicationId() ? this.client.call('previewApplicationRoleConversionRollback', { project: this.access.projectId(), application: this.access.applicationId(), conversion: receipt.id, body: { reason } }) : this.client.call('previewProjectRoleConversionRollback', { project: this.access.projectId(), conversion: receipt.id, body: { reason } }))
  }
  private async request(operation: () => Promise<RoleConversionRecord>, mutation = false): Promise<void> {
    const generation = ++this.generation
    this.busy.set(true); this.error.set('')
    try {
      const value = await operation()
      if (generation !== this.generation) return
      this.receipt.set(value); this.uncertain.set(false); this.rollbackReason.reset()
      if (value.data.applied) { this.form.markAsPristine(); await this.access.refresh(); await this.workspace.refreshAccess() }
      accessFocus('conversion-review-heading')
    } catch (error) {
      if (generation !== this.generation) return
      this.error.set(error instanceof Error ? error.message : 'The role conversion could not be confirmed.')
      this.uncertain.set(mutation && (!(error instanceof ApiError) || error.status >= 500))
      if (error instanceof ApiError && error.status === 409) { this.receipt.set(null); this.error.set('The reviewed access changed or expired. Your entries remain. Preview the current conversion again.') }
      this.access.mutationError(error)
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
}
