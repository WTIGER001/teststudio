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
    map(() => this.router.routerState.snapshot.root.firstChild?.data['helpTopic'] as string | undefined)
  ), { initialValue: this.router.routerState.snapshot.root.firstChild?.data['helpTopic'] as string | undefined })
  readonly page = computed(() => this.session.signedIn() ? (helpPages[this.topic() ?? 'projects'] ?? helpPages['projects']) : helpPages['signin'])
}
