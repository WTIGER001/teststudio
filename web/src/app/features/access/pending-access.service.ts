import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { PendingAccessRecord, PendingProviderChoice } from '../../core/generated/access.types'
import { AccessCollection } from './access-collection'
import { accessFocus } from './access.utils'
import { AssignmentsService } from './assignments.service'
import { pendingAccessDefinition, pendingAccessTarget, pendingAccessValidation } from './pending-access.utils'

@Injectable()
export class PendingAccessService {
  private readonly client = inject(AccessClientService)
  readonly access = inject(AssignmentsService)
  readonly records = new AccessCollection<PendingAccessRecord>()
  readonly providers = signal<PendingProviderChoice[]>([])
  readonly reviewedProviders = signal<PendingProviderChoice[]>([])
  readonly loading = signal(false)
  readonly busy = signal(false)
  readonly opened = signal(false)
  readonly needsReview = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly canAdd = computed(() => this.access.canAdd() && !this.loading() && this.providers().length > 0)
  readonly form = new FormGroup({ providerId: new FormControl('', { nonNullable: true }), email: new FormControl('', { nonNullable: true }), roleId: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }) })
  private readonly providerId = toSignal(this.form.controls.providerId.valueChanges, { initialValue: '' })
  readonly selectedProvider = computed(() => this.reviewedProviders().find(provider => provider.id === this.providerId()) ?? null)
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => { this.generation++; this.form.reset(); this.records.clear(); this.providers.set([]); this.reviewedProviders.set([]) }); void this.refresh() }
  async refresh(): Promise<void> {
    if (this.busy() || this.loading()) return
    const generation = this.generation, project = this.access.projectId()
    this.loading.set(true); this.error.set('')
    try {
      await this.load(false)
      const choices = this.access.rights().manage ? await this.client.call('pendingAccessProviders', { project }) : { items: [] }
      if (generation === this.generation) this.providers.set(choices.items)
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'Pending access could not be refreshed.'); this.accessError(error) } }
    finally { if (generation === this.generation) this.loading.set(false) }
  }
  async load(more = false): Promise<void> {
    const project = this.access.projectId(), generation = this.generation
    await this.records.load(cursor => this.client.call('listPendingAccess', { project, query: { limit: 30, cursor: cursor || undefined } }).catch(error => { if (generation === this.generation) this.accessError(error); throw error }), more)
  }
  open(): void {
    if (!this.canAdd() || this.busy()) return
    this.form.reset(); this.reviewedProviders.set(this.providers()); this.error.set(''); this.message.set(''); this.needsReview.set(false); this.opened.set(true); this.access.action.close(); accessFocus('pending-editor-heading')
  }
  close(): void { if (this.busy()) return; this.form.reset(); this.opened.set(false); this.needsReview.set(false); this.error.set(''); this.reviewedProviders.set([]); accessFocus('pending-access-heading') }
  async review(): Promise<void> {
    await this.refresh()
    if (!this.error() && !this.records.error()) { this.reviewedProviders.set(this.providers()); this.needsReview.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.needsReview() || !this.canAdd()) return
    const body = pendingAccessDefinition(this.form.getRawValue(), this.reviewedProviders()), validation = pendingAccessValidation(body, this.access.roleChoices())
    this.error.set(validation); if (validation) return
    const generation = this.generation, project = this.access.projectId(); this.busy.set(true); this.message.set('')
    try {
      await this.client.call('createPendingAccess', { project, body })
      if (generation !== this.generation) return
      this.busy.set(false); this.close(); this.message.set('Pending project access prepared. It activates only after the allowed provider verifies this email during sign-in.'); await this.refresh()
    } catch (error) {
      if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The pending access request could not be confirmed.'); this.needsReview.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500); this.accessError(error) }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
  cancel(row: PendingAccessRecord): void {
    if (!row.data.canCancel || !this.access.rights().manage || this.busy() || this.access.action.busy()) return
    const generation = this.generation, project = this.access.projectId()
    this.access.action.open({ title: 'Cancel pending project access', target: pendingAccessTarget(row), returnFocus: 'pending-access-heading', impact: 'Prevent this approval from activating at sign-in. Existing roles remain available through ordinary role assignment controls.', submit: async reason => {
      try { await this.client.call('cancelPendingAccess', { project, access: row.id, revision: row.revision, body: { reason } }) }
      catch (error) { if (generation === this.generation) this.accessError(error); throw error }
      if (generation === this.generation) await this.refresh()
      return 'Pending project access cancelled. Existing role assignments remain available.'
    } })
  }
  private accessError(error: unknown): void {
    if (error instanceof ApiError && [401, 403].includes(error.status)) { this.form.reset(); this.opened.set(false); this.reviewedProviders.set([]); this.providers.set([]); this.records.clear(); this.access.mutationError(error) }
  }
}
