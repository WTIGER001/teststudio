import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import { WorkspaceService } from '../../core/workspace.service'
import { accessFocus } from './access.utils'
import { assignmentDefinition, assignmentValidation, selectedRole } from './assignment.utils'
import { AssignmentsService } from './assignments.service'

@Injectable()
export class AssignmentEditorService {
  private readonly client = inject(AccessClientService)
  private readonly route = inject(ActivatedRoute)
  private readonly destroy = inject(DestroyRef)
  private readonly workspace = inject(WorkspaceService)
  readonly access = inject(AssignmentsService)
  readonly opened = signal(false)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly form = new FormGroup({ principalKind: new FormControl<'user' | 'group'>('user', { nonNullable: true }), principalId: new FormControl('', { nonNullable: true }), roleId: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  private readonly roleId = toSignal(this.form.controls.roleId.valueChanges, { initialValue: '' })
  readonly role = computed(() => selectedRole(this.access.roles.items(), this.roleId()))
  private generation = 0

  constructor() {
    this.route.paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => { this.generation++; this.opened.set(false); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.form.reset() })
    this.destroy.onDestroy(() => { this.generation++ })
  }
  open(): void {
    if (!this.access.canAdd()) return
    this.generation++; this.form.reset(); this.error.set(''); this.conflict.set(false); this.busy.set(false); this.opened.set(true); this.access.message.set(''); this.access.action.close(); accessFocus('assignment-editor-heading')
  }
  close(): void { this.generation++; this.opened.set(false); this.busy.set(false); accessFocus('assignments-heading') }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict() || !this.access.canAdd()) return
    const definition = assignmentDefinition(this.form.getRawValue()), validation = assignmentValidation(definition, this.access.roles.items())
    this.error.set(validation); if (validation) return
    const project = this.access.projectId(), generation = ++this.generation
    this.busy.set(true)
    try {
      if (project) await this.client.call('createProjectAssignment', { project, body: definition })
      else await this.client.call('createInstallationAssignment', { body: definition })
      if (generation !== this.generation) return
      this.close(); this.access.message.set('Role assigned in the displayed scope. Permissions apply without another sign-in.')
      await this.access.refresh(); await this.workspace.refreshAccess()
    } catch (error) {
      if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The role could not be assigned.'); this.conflict.set(error instanceof ApiError && error.status === 409); this.access.mutationError(error) }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
}
