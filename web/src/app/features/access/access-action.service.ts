import { Injectable, signal } from '@angular/core'
import { ApiError } from '../../core/api.service'
import { accessFocus, reasonValidation } from './access.utils'

interface AccessAction { title: string; impact: string; target: string; submit: (reason: string) => Promise<string>; returnFocus: string }

@Injectable()
export class AccessActionService {
  readonly pending = signal<AccessAction | null>(null)
  readonly reason = signal('')
  readonly busy = signal(false)
  readonly error = signal('')
  readonly conflict = signal(false)
  readonly message = signal('')
  private generation = 0

  open(action: AccessAction): void { this.reset(); this.pending.set(action); accessFocus('access-action-heading') }
  reset(): void { this.generation++; this.pending.set(null); this.reason.set(''); this.busy.set(false); this.error.set(''); this.conflict.set(false); this.message.set('') }
  close(): void { const target = this.pending()?.returnFocus; this.reset(); if (target) accessFocus(target) }
  async submit(): Promise<void> {
    const action = this.pending(); if (!action || this.busy() || this.conflict()) return
    const validation = reasonValidation(this.reason()); this.error.set(validation); if (validation) return
    const generation = ++this.generation
    this.busy.set(true)
    try {
      const message = await action.submit(this.reason().trim())
      if (generation !== this.generation) return
      this.pending.set(null); this.message.set(message); accessFocus(action.returnFocus)
    } catch (error) { if (generation === this.generation) { this.error.set(error instanceof Error ? error.message : 'The action failed.'); this.conflict.set(error instanceof ApiError && error.status === 409) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
}
