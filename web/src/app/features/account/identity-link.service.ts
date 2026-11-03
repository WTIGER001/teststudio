import { computed, DestroyRef, inject, Injectable, signal } from '@angular/core'
import { FormControl, FormGroup } from '@angular/forms'
import { ActivatedRoute, Router } from '@angular/router'
import { AccessClientService } from '../../core/access-client.service'
import { ApiError } from '../../core/api.service'
import { ApplicationService } from '../../core/application.service'
import type { IdentityLinkOperation, IdentityLinkOptions } from '../../core/generated/access.types'
import { AccountService } from './account.service'
import { identityLinkAuthorizePath, identityLinkDefinition, identityLinkTerminal, identityLinkValidation } from './identity-link.utils'

@Injectable()
export class IdentityLinkService {
  private readonly client = inject(AccessClientService)
  private readonly app = inject(ApplicationService)
  private readonly account = inject(AccountService)
  private readonly router = inject(Router)
  readonly options = signal<IdentityLinkOptions | null>(null)
  readonly operation = signal<IdentityLinkOperation | null>(null)
  readonly authorizePath = computed(() => { const operation = this.operation(); return operation ? identityLinkAuthorizePath(operation) : '' })
  readonly busy = signal(false)
  readonly error = signal('')
  readonly needsReview = signal(false)
  readonly form = new FormGroup({ providerId: new FormControl('', { nonNullable: true }), reason: new FormControl('', { nonNullable: true }), password: new FormControl('', { nonNullable: true }), confirmLocal: new FormControl(false, { nonNullable: true }) })
  private generation = 0
  private operationId = ''
  constructor() {
    const subscription = inject(ActivatedRoute).queryParamMap.subscribe(params => { this.operationId = params.get('identityLink') ?? ''; void this.load(false) })
    inject(DestroyRef).onDestroy(() => { this.generation++; subscription.unsubscribe(); this.form.reset(); this.operation.set(null); this.options.set(null) })
  }
  private show(operation: IdentityLinkOperation): void {
    this.operationId = operation.id
    this.operation.set(operation)
  }
  async load(refreshSession = true): Promise<void> {
    if (this.busy()) return
    const generation = ++this.generation; this.busy.set(true); this.error.set(''); this.form.controls.password.reset()
    try {
      if (refreshSession) await this.app.refreshAuthentication()
      const options = await this.client.call('identityLinkOptions', {})
      const operation = this.operationId ? await this.client.call('readIdentityLink', { operation: this.operationId }) : null
      if (generation !== this.generation) return
      this.options.set(options); this.operation.set(null)
      if (operation) this.show(operation)
      this.needsReview.set(false)
      if (operation?.state === 'complete') await this.account.refresh()
    } catch (error) {
      if (generation === this.generation) { this.options.set(null); this.operation.set(null); this.needsReview.set(true); this.error.set(error instanceof Error ? error.message : 'Link state could not be loaded. Refresh your session and review the current state.') }
    } finally { if (generation === this.generation) this.busy.set(false) }
  }
  async start(): Promise<void> {
    const options = this.options()
    if (this.busy() || this.needsReview() || !options?.canStart || this.operation()) return
    const body = identityLinkDefinition(this.form.getRawValue(), options.sourceMethod), validation = identityLinkValidation(body, options.sourceMethod)
    this.error.set(validation); if (validation) return
    const generation = ++this.generation; this.busy.set(true)
    try {
      const operation = await this.client.call('startIdentityLink', { body })
      if (generation !== this.generation) return
      this.show(operation)
      await this.router.navigate([], { queryParams: { identityLink: operation.id }, queryParamsHandling: 'merge', replaceUrl: true })
    } catch (error) { if (generation === this.generation) this.reject(error) }
    finally { this.form.controls.password.reset(); if (generation === this.generation) this.busy.set(false) }
  }
  async mutate(action: 'confirm' | 'cancel'): Promise<void> {
    const operation = this.operation()
    if (this.busy() || this.needsReview() || !operation || identityLinkTerminal(operation.state) || (action === 'confirm' && operation.state !== 'review')) return
    const generation = ++this.generation; this.busy.set(true); this.error.set('')
    try {
      const result = await this.client.call(action === 'confirm' ? 'confirmIdentityLink' : 'cancelIdentityLink', { operation: operation.id, revision: operation.revision })
      if (generation !== this.generation) return
      this.show(result)
      if (action === 'confirm') { await this.app.refreshAuthentication(); if (generation === this.generation) await this.account.refresh() }
    } catch (error) { if (generation === this.generation) { this.reject(error); this.needsReview.set(true) } }
    finally { if (generation === this.generation) this.busy.set(false) }
  }
  async reset(): Promise<void> {
    const operation = this.operation()
    if (this.busy() || (operation && (this.needsReview() || !identityLinkTerminal(operation.state)))) return
    this.operationId = ''; this.operation.set(null); this.form.reset(); this.error.set(''); this.needsReview.set(false)
    await this.router.navigate([], { queryParams: { identityLink: null }, queryParamsHandling: 'merge', replaceUrl: true }); await this.load(false)
  }
  guardNavigation(event: Event): void { if (this.busy() || this.needsReview()) event.preventDefault() }
  private reject(error: unknown): void {
    this.needsReview.set(!(error instanceof ApiError) || error.status === 409 || error.status >= 500)
    this.error.set(error instanceof Error ? error.message : 'The request outcome is unknown. Review the current link state before continuing.')
  }
}
