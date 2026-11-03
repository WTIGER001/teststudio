import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup } from '@angular/forms'
import { ApiError } from '../../core/api.service'
import { AccessClientService } from '../../core/access-client.service'
import type { RolePolicyMutation, RolePreview } from '../../core/generated/access.types'
import { RolePoliciesService } from './role-policies.service'
import { previewFingerprint, previewValidation, rolePreviewInput } from './role-policy.utils'

@Injectable()
export class RolePreviewService {
  private readonly client = inject(AccessClientService)
  private readonly roles = inject(RolePoliciesService)
  readonly form = new FormGroup({ userId: new FormControl('', { nonNullable: true }), object: new FormControl('', { nonNullable: true }) })
  readonly fields = toSignal(this.form.valueChanges, { initialValue: this.form.getRawValue() })
  readonly result = signal<RolePreview | null>(null)
  readonly fingerprint = signal('')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly showUnchanged = signal(false)
  private generation = 0
  constructor() { inject(DestroyRef).onDestroy(() => this.reset()) }
  reset(): void { this.generation++; this.form.reset(); this.result.set(null); this.fingerprint.set(''); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.showUnchanged.set(false) }
  async run(policy: RolePolicyMutation, revision: number): Promise<void> {
    if (this.busy()) return
    const { userId, object } = this.form.getRawValue(), validation = previewValidation(userId, object)
    this.error.set(validation); if (validation) return
    const generation = ++this.generation, key = previewFingerprint(policy, revision, userId, object); this.busy.set(true); this.result.set(null)
    try { const result = await this.client.call('previewRolePolicy', { body: rolePreviewInput(policy, revision, userId, object) }); if (generation === this.generation) { this.result.set(result); this.fingerprint.set(key) } }
    catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The preview could not be evaluated.'); this.conflict.set(error instanceof ApiError && error.status === 409); await this.roles.mutationError(error) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
