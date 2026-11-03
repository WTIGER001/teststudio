import { computed, inject, Injectable } from '@angular/core'
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
  readonly feedback = computed(() => authenticationFeedback(this.route()))
  providerPath(id: string): string { return providerLoginPath(id, this.route()) }
  async dismiss(): Promise<void> { this.session.dismissLogout(); await this.router.navigate([], { queryParams: { signIn: null, providerLogout: null }, queryParamsHandling: 'merge', replaceUrl: true }) }
}
