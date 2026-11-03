import { DestroyRef, inject, Injectable, signal } from '@angular/core'
import { takeUntilDestroyed } from '@angular/core/rxjs-interop'
import { FormControl, FormGroup, Validators } from '@angular/forms'
import { ActivatedRoute } from '@angular/router'
import { combineLatest } from 'rxjs'
import { AccessClientService } from '../../core/access-client.service'
import { SessionService } from '../../core/session.service'
import type { CLIView } from '../../core/generated/access.types'

@Injectable()
export class CLISignInService {
  readonly account = inject(SessionService)
  private readonly client = inject(AccessClientService)
  readonly view = signal<CLIView | null>(null)
  readonly busy = signal(false)
  readonly error = signal('')
  readonly message = signal('')
  readonly code = new FormControl('', { nonNullable: true, validators: [Validators.required, Validators.pattern('^[A-Fa-f0-9]{4}-[A-Fa-f0-9]{4}$')] })
  readonly form = new FormGroup({ code: this.code })
  private request = ''
  private clientName = ''
  private generation = 0

  constructor() {
    const route = inject(ActivatedRoute)
    combineLatest([route.paramMap, route.queryParamMap]).pipe(takeUntilDestroyed(inject(DestroyRef))).subscribe(([params, query]) => {
      this.request = params.get('requestId') ?? ''; this.clientName = query.get('clientName') ?? ''
      this.generation++; this.view.set(null); this.error.set(''); this.message.set(''); this.code.reset(); void this.refresh()
    })
  }

  async refresh(): Promise<void> {
    const generation = this.generation
    this.busy.set(true); this.error.set('')
    try { const view = await this.client.call('readCLISignInRequest', { request: this.request, query: { clientName: this.clientName } }); if (generation === this.generation) this.view.set(view) }
    catch (error) { if (generation === this.generation) this.fail(error) }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  async confirm(approve: boolean): Promise<void> {
    if (this.busy() || this.view()?.state !== 'pending' || approve && this.code.invalid) return
    const generation = this.generation
    this.busy.set(true); this.error.set(''); this.message.set('')
    try {
      const view = await this.client.call('confirmCLISignInRequest', { request: this.request, body: { clientName: this.clientName, code: this.code.value, approve } })
      if (generation !== this.generation) return
      this.view.set(view); this.code.reset(); this.message.set(approve ? 'CLI sign-in approved. Return to the terminal to finish.' : 'CLI sign-in declined. No CLI session was issued.')
    } catch (error) { if (generation === this.generation) { this.fail(error); this.code.reset() } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }

  private fail(error: unknown): void { this.error.set(error instanceof Error ? error.message : 'CLI sign-in could not be read. Start it again in the terminal.') }
}
