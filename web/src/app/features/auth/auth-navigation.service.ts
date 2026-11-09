import { computed, effect, inject, Injectable, signal } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { NavigationEnd, Router } from '@angular/router'
import { filter, map } from 'rxjs'
import { SessionService } from '../../core/session.service'
import { authenticationFeedback, providerLoginPath } from './auth-navigation.utils'

@Injectable({ providedIn: 'root' })
export class AuthNavigationService {
  private readonly router = inject(Router)
  readonly session = inject(SessionService)
  private readonly route = toSignal(this.router.events.pipe(filter(event => event instanceof NavigationEnd), map(event => event.urlAfterRedirects)), { initialValue: this.router.url })
  private readonly navigationFailed = signal(false)
  private readonly manualSignIn = signal(false)
  readonly redirecting = computed(() => {
    const options = this.session.providers(), query = new URL(this.route(), 'http://teststudio.local').searchParams
    return window.self === window.top && !new URL(this.route(), 'http://teststudio.local').pathname.startsWith('/status/') && !this.manualSignIn() && !this.session.loading() && !this.session.signedIn() && !this.session.authenticating() && !this.session.error() && !this.session.signOutCompleted() && !this.navigationFailed() && !query.has('signIn') && !query.has('providerLogout') && !options.localEnabled && !options.localAccountsEnabled && options.providers.some(provider => provider.id === options.autoRedirectProvider)
  })
  constructor() {
    effect(() => {
      const query = new URL(this.route(), 'http://teststudio.local').searchParams
      if (query.has('signIn') || query.has('providerLogout') || this.session.signOutCompleted() || this.session.error()) this.manualSignIn.set(true)
      if (this.redirecting()) {
        try { window.location.replace(this.providerPath(this.session.providers().autoRedirectProvider)) }
        catch { this.navigationFailed.set(true); this.session.error.set('Automatic sign-in could not open. Use the SSO button to continue.') }
      }
    })
  }
  readonly feedback = computed(() => authenticationFeedback(this.route()))
  providerPath(id: string): string { return providerLoginPath(id, this.route()) }
  async dismiss(): Promise<void> { this.manualSignIn.set(true); this.session.dismissLogout(); await this.router.navigate([], { queryParams: { signIn: null, providerLogout: null }, queryParamsHandling: 'merge', preserveFragment: true, replaceUrl: true }) }
}
