import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { combineLatest } from 'rxjs'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { PermissionDefinition, PermissionExplanation } from '../../core/generated/access.types'
import { SessionService } from '../../core/session.service'
import { WorkspaceService } from '../../core/workspace.service'
import { appendExplanation, explanationPermissions, explanationValidation } from './permission-explanation.utils'

@Injectable()
export class PermissionExplanationService {
  private readonly client = inject(AccessClientService)
  private readonly session = inject(SessionService)
  private readonly workspace = inject(WorkspaceService)
  private readonly destroy = inject(DestroyRef)
  readonly projectId = signal('')
  readonly applicationId = signal('')
  readonly permissions = signal<PermissionDefinition[]>([])
  readonly report = signal<PermissionExplanation | null>(null)
  readonly busy = signal(false)
  readonly initializing = signal(false)
  readonly error = signal('')
  readonly canInspectOthers = signal(false)
  readonly form = new FormGroup({ userId: new FormControl('', { nonNullable: true }), object: new FormControl('', { nonNullable: true }), permission: new FormControl('', { nonNullable: true }) })
  readonly ownId = computed(() => this.session.user()?.id ?? '')
  private generation = 0

  constructor() {
    this.form.valueChanges.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.clear())
    const route = inject(ActivatedRoute)
    combineLatest([route.paramMap, route.queryParamMap]).pipe(takeUntilDestroyed(this.destroy)).subscribe(([params, query]) => {
      this.clear(); this.canInspectOthers.set(false); this.permissions.set([]); this.projectId.set(params.get('projectId') ?? ''); this.applicationId.set(params.get('applicationId') ?? '')
      this.form.reset({ userId: query.get('user') || this.ownId(), object: query.get('object') ?? '', permission: query.get('permission') || (this.projectId() ? 'teststudio.projects.read' : 'teststudio.system.read') }, { emitEvent: false })
      void this.initialize()
    })
    this.destroy.onDestroy(() => this.clear())
  }
  private clear(): void { this.generation++; this.report.set(null); this.busy.set(false); this.initializing.set(false); this.error.set('') }
  async initialize(): Promise<void> {
    const generation = ++this.generation; this.initializing.set(true); this.error.set('')
    try {
      const catalog = await this.client.call('explainOwnPermission', { query: { permission: 'teststudio.system.read', limit: 1 } })
      if (generation !== this.generation) return
      this.permissions.set(explanationPermissions(catalog.permissions, this.projectId()))
      this.initializing.set(false)
      await this.check()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'Permission choices could not be loaded.'); this.initializing.set(false) } }
  }
  async check(more = false): Promise<void> {
    if (this.busy() || this.initializing()) return
    const fields = this.form.getRawValue(), validation = explanationValidation(fields, this.permissions())
    this.error.set(validation); if (validation) return
    const previous = this.report()
    if (more && !previous?.cursor) return
    const generation = ++this.generation, project = this.projectId(), user = fields.userId.trim()
    this.busy.set(true); if (!more) this.report.set(null)
    try {
      const query = { project: project || undefined, application: this.applicationId() || undefined, object: fields.object.trim() || undefined, permission: fields.permission, limit: 30, cursor: more ? previous?.cursor : undefined }
      const report = user === this.ownId() ? await this.client.call('explainOwnPermission', { query }) : await this.client.call('explainUserPermission', { user, query })
      if (generation !== this.generation) return
      this.permissions.set(explanationPermissions(report.permissions, project)); this.canInspectOthers.set(report.canInspectOthers)
      this.report.set(more && previous ? appendExplanation(previous, report) : report)
    } catch (error) {
      if (generation !== this.generation) return
      this.report.set(null)
      this.error.set(error instanceof ApiError && error.status === 409 ? 'Access changed during inspection. Check access again to load current sources.' : error instanceof Error ? error.message : 'Access could not be explained. Check access again.')
      if (error instanceof ApiError && [401, 403, 404].includes(error.status)) { this.canInspectOthers.set(false); void this.workspace.refreshAccess() }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
  inspectSelf(): void { this.form.controls.userId.setValue(this.ownId()); void this.check() }
}
