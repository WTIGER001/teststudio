import { computed, inject, Injectable } from '@angular/core'
import { toSignal } from '@angular/core/rxjs-interop'
import { NavigationEnd, Router } from '@angular/router'
import { filter, map } from 'rxjs'
import { SessionService } from '../../core/session.service'
import { helpPages } from './help.pages'

@Injectable({ providedIn: 'root' })
export class HelpService {
  private readonly router = inject(Router)
  private readonly session = inject(SessionService)
  private readonly topic = toSignal(this.router.events.pipe(
    filter(event => event instanceof NavigationEnd),
    map(() => this.currentTopic())
  ), { initialValue: this.currentTopic() })
  private currentTopic(): string | undefined { const route = this.router.routerState.snapshot.root.firstChild; return route?.data['helpTopic'] === 'create-test' && route.paramMap.get('kind') === 'https' ? 'https-create' : route?.data['helpTopic'] }
  readonly page = computed(() => this.session.mustChangePassword() ? helpPages['password-required'] : this.session.signedIn() ? (helpPages[this.topic() ?? 'projects'] ?? helpPages['projects']) : helpPages['signin'])
}
