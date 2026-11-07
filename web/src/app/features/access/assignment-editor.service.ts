import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { accessFocus } from './access.utils'
import type { AssignmentRecord, MemberCandidate, StandardAssignmentReplacement } from '../../core/generated/access.types'
import { assignmentDefinition, assignmentValidation, selectedRole, standardRole, roleDescription } from './assignment.utils'
import { AssignmentsService } from './assignments.service'

@Injectable()
export class AssignmentEditorService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  private readonly workspace = inject(WorkspaceService)
  readonly access = inject(AssignmentsService)
  readonly opened = signal(false)
  readonly editing = signal<AssignmentRecord | null>(null)
  readonly reviewed = signal(false)
  readonly candidate = signal<MemberCandidate | null>(null)
  readonly description = roleDescription
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({ principalKind: new FormControl<'user' | 'group'>('user', { nonNullable: true }), principalId: new FormControl('', { nonNullable: true }), roleId: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  private readonly roleId = toSignal(this.form.controls.roleId.valueChanges, { initialValue: '' })
  readonly role = computed(() => selectedRole(this.access.roleChoices(), this.roleId()))
  private generation = 0

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.reviewed.set(false); this.candidate.set(null) })
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.form.reset() })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  open(row?: AssignmentRecord): void {
    if (!this.access.canAdd() || row && (!row.data.canRevoke || !standardRole(row.data.roleId))) return
    if (this.opened() && this.form.dirty && !window.confirm('Discard the unfinished role assignment?')) return
    this.generation++; this.form.enable(); this.editing.set(row ?? null); this.reviewed.set(false); this.candidate.set(null)
    this.form.reset(row ? { principalKind: row.data.principalKind, principalId: row.data.principalId, roleId: row.data.roleId, reason: '' } : undefined)
    if (row) { this.form.controls.principalKind.disable(); this.form.controls.principalId.disable() }
    this.error.set(''); this.conflict.set(false); this.busy.set(false); this.opened.set(true); this.access.message.set(''); this.access.action.close(); accessFocus('assignment-editor-heading')
  }
  close(force = false): void { if (!force && this.form.dirty && !window.confirm('Discard the unfinished role assignment?')) return; this.generation++; this.form.markAsPristine(); this.opened.set(false); this.busy.set(false); accessFocus('assignments-heading') }
  async review(): Promise<void> {
    if (this.busy() || this.conflict()) return
    const definition = assignmentDefinition(this.form.getRawValue()), validation = assignmentValidation(definition, this.access.roleChoices())
    this.error.set(validation); if (validation) return
    const generation = this.generation, project = this.access.projectId(), application = this.access.applicationId()
    this.busy.set(true)
    try {
      if (definition.principalKind === 'user' && project) {
        const user = application ? await this.client.call('readApplicationMemberCandidate', { project, application, user: definition.principalId }) : await this.client.call('readProjectMemberCandidate', { project, user: definition.principalId })
        if (generation !== this.generation) return
        if (user.disabled) { this.error.set('Restore this user before assigning access.'); return }
        this.candidate.set(user)
      }
      if (generation === this.generation) { this.reviewed.set(true); accessFocus('assignment-review-heading') }
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The user could not be verified.'); this.access.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict() || !this.access.canAdd()) return
    const definition = assignmentDefinition(this.form.getRawValue()), validation = assignmentValidation(definition, this.access.roleChoices())
    this.error.set(validation); if (validation) return
    if (standardRole(definition.roleId) && !this.reviewed()) { await this.review(); return }
    const project = this.access.projectId(), application = this.access.applicationId(), generation = ++this.generation
    this.busy.set(true)
    try {
      if (this.editing()) {
        const row = this.editing()!, params = { project, kind: definition.principalKind, principal: definition.principalId, revision: row.revision, body: { roleId: definition.roleId as StandardAssignmentReplacement['roleId'], previousRoleId: row.data.roleId, reason: definition.reason } }
        if (application) await this.client.call('replaceApplicationStandardAssignment', { ...params, application })
        else await this.client.call('replaceProjectStandardAssignment', params)
      } else if (application) await this.client.call('createApplicationAssignment', { project, application, body: definition })
      else if (project) await this.client.call('createProjectAssignment', { project, body: definition })
      else await this.client.call('createInstallationAssignment', { body: definition })
      if (generation !== this.generation) return
      this.close(true); this.access.message.set('Role assigned in the displayed scope. Permissions apply without another sign-in.')
      await this.access.refresh(); await this.workspace.refreshAccess()
    } catch (error) {
      if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The role could not be assigned.'); this.conflict.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500); this.access.mutationError(error) }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
}
