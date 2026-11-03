import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { AuthNavigationService } from './auth-navigation.service'

@Component({
  selector: 'ts-auth-feedback', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (navigation.feedback().message) { <div class="notice" [class.error]="navigation.feedback().error" [attr.role]="navigation.feedback().error ? 'alert' : 'status'">{{ navigation.feedback().message }}<button type="button" class="button subtle dismiss" (click)="navigation.dismiss()">Dismiss sign-in message</button></div> }
    @if (!navigation.session.signedIn() && navigation.session.signOutCompleted()) {
      <section class="notice" aria-labelledby="provider-signout-heading"><h2 id="provider-signout-heading">Signed out of TestStudio</h2>
        @if (navigation.session.providerLogout(); as offer) { <p>Your identity provider session may still be active. You can also sign out of {{ offer.providerName }}.</p><a class="button secondary" [href]="offer.url">Sign out of {{ offer.providerName }}</a><p class="small muted">Available until {{ offer.expiresAt | date:'shortTime':'UTC' }} UTC. The provider may ask you to confirm.</p> }
        @else if (navigation.session.providerLogoutUnavailable()) { <p>Provider sign-out is unavailable. Your TestStudio session is already closed.</p> }
        <button type="button" class="button subtle dismiss" (click)="navigation.dismiss()">Dismiss sign-out message</button>
      </section>
    }
  `,
  styles: `h2 { font-size: 16px; margin: 0 0 12px; } p { margin: 10px 0; overflow-wrap: anywhere; } .button { white-space: normal; overflow-wrap: anywhere; max-width: 100%; } .dismiss { display: block; margin-top: 12px; }`
})
export class AuthFeedbackComponent { readonly navigation = inject(AuthNavigationService) }
