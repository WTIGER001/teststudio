import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed, toSignal } from '@angular/core/rxjs-interop'
import { FormArray, FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import type { ProviderRecord } from '../../core/generated/access.types'
import { accessFocus } from './access.utils'
import { ProvidersService } from './providers.service'
import { providerCredentialChoices, providerDefinition, providerFields, providerValidation } from './provider.utils'

const text = (value = '') => new FormControl(value, { nonNullable: true })
const mapping = (external = '', groupId = '') => new FormGroup({ external: text(external), groupId: text(groupId) })

@Injectable()
export class ProviderEditorService {
  private readonly client = inject(AccessClientService)
  private readonly destroy = inject(DestroyRef)
  readonly providers = inject(ProvidersService)
  readonly opened = signal(false)
  readonly editing = signal<ProviderRecord | null>(null)
  readonly busy = signal(false)
  readonly conflict = signal(false)
  readonly error = signal('')
  readonly form = new FormGroup({ id: text(), name: text(), issuer: text(), clientId: text(), secretRef: text(), scopes: text(), disabled: new FormControl(true, { nonNullable: true }), autoProvision: new FormControl(false, { nonNullable: true }), groupClaim: text(), membershipTtlSeconds: new FormControl(1800, { nonNullable: true }), reason: text(), mappings: new FormArray<ReturnType<typeof mapping>>([]) })
  private readonly values = toSignal(this.form.valueChanges, { initialValue: this.form.getRawValue() })
  readonly credentials = computed(() => providerCredentialChoices(this.providers.credentialRefs(), this.values().secretRef ?? ''))
  private generation = 0
  constructor() {
    inject(ActivatedRoute).paramMap.pipe(takeUntilDestroyed(this.destroy)).subscribe(() => this.reset())
    this.providers.onAccessLost = () => this.reset()
    this.destroy.onDestroy(() => this.reset())
  }
  private reset(): void { this.generation++; this.opened.set(false); this.editing.set(null); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.form.reset(); this.form.controls.mappings.clear() }
  open(selected?: ProviderRecord): void {
    if (selected ? !selected.data.canManage : !this.providers.canCreate()) return
    this.reset(); this.editing.set(selected ?? null)
    const fields = providerFields(selected); this.form.reset(fields)
    for (const item of fields.mappings) this.form.controls.mappings.push(mapping(item.external, item.groupId))
    this.opened.set(true); this.providers.message.set(''); accessFocus('provider-editor-heading')
  }
  close(): void { this.reset(); accessFocus(this.providers.id() ? 'provider-details-heading' : 'providers-heading') }
  addMapping(): void { if (this.form.controls.mappings.length < 40) this.form.controls.mappings.push(mapping()) }
  removeMapping(index: number): void { this.form.controls.mappings.removeAt(index) }
  async reload(): Promise<void> {
    const selected = this.editing(); if (!selected || this.busy()) return
    const generation = ++this.generation; this.busy.set(true)
    try { const record = await this.client.call('readIdentityProvider', { provider: selected.id }); if (generation === this.generation) { if (record.data.canManage) this.open(record); else { this.reset(); await this.providers.refresh() } } }
    catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The stored provider could not be loaded.'); this.providers.accessError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async submit(): Promise<void> {
    if (this.busy() || this.conflict()) return
    const fields = this.form.getRawValue(), validation = providerValidation(fields, this.providers.credentialRefs(), this.editing()); this.error.set(validation); if (validation) return
    const selected = this.editing(), generation = ++this.generation, body = providerDefinition(fields); this.busy.set(true)
    try {
      if (selected) await this.client.call('editIdentityProvider', { provider: selected.id, revision: selected.revision, body }); else await this.client.call('createIdentityProvider', { body })
      if (generation !== this.generation) return
      this.close(); this.providers.message.set('Provider saved. Earlier provider sessions and memberships require fresh sign-in.'); await this.providers.refresh()
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The provider could not be saved.'); this.conflict.set(error instanceof ApiError && (error.status === 409 || error.status >= 500 || error.message.includes('unreadable')) || !(error instanceof ApiError)); this.providers.accessError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
