import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { IdentityLinkComponent } from './identity-link.component'
import { AccountService } from './account.service'
import { PasswordChangeComponent } from '../auth/password-change.component'

@Component({
  standalone: true, imports: [DatePipe, PasswordChangeComponent, IdentityLinkComponent], providers: [AccountService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">Personal settings</span><h1>Your account</h1><p class="muted">Review your identity and manage your local password.</p></div><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.refresh()">Refresh account</button></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.loading()) { <p role="status">Loading your account…</p> }
    @if (service.user(); as user) {
      <section class="panel" aria-labelledby="own-profile-heading"><h2 id="own-profile-heading">{{ user.name }}</h2><dl class="detail-grid"><dt>User identifier</dt><dd><code class="wrap-anywhere">{{ user.id }}</code></dd><dt>Email</dt><dd>{{ user.email || 'No email' }} · {{ user.emailVerified ? 'Verified' : 'Not verified' }}</dd><dt>Created</dt><dd>{{ user.createdAt | date:'medium':'UTC' }} UTC</dd></dl><p class="small muted guidance">Sign-in alone grants no project access. Matching email addresses never link accounts.</p></section>
      @if (service.credential(); as item) {
        <section class="panel" aria-labelledby="own-credential-heading"><h2 id="own-credential-heading">Your local sign-in</h2>
          @if (!item.data.localAccountsEnabled) { <p>Managed local accounts are disabled for this installation.</p> }
          @else if (!item.data.configured) { <p>No local password is configured. An installation administrator can provision one for your account.</p> }
          @else if (item.data.disabled) { <p>Your local credential is disabled. An installation administrator can review and enable it.</p> }
          @else { <p class="small muted">{{ item.data.mustChangePassword ? 'A password change is required before password sign-in opens the workspace.' : 'Local password sign-in is enabled.' }}</p><ts-password-change (changed)="service.refresh()" /> }
        </section>
      }
      <ts-identity-link />
      <section class="panel" aria-labelledby="own-identities-heading"><h2 id="own-identities-heading">Your linked identities</h2><p class="small muted">These exact issuer and subject pairs identify your external sign-in accounts.</p>
        @if (service.identities.error()) { <p class="notice error" role="alert">{{ service.identities.error() }}</p> }
        <div class="table-scroll"><table><caption class="sr-only">Your external identities</caption><thead><tr><th scope="col">Issuer</th><th scope="col">Subject</th></tr></thead><tbody>@for (item of service.identities.items(); track item.id) { <tr><td class="wrap-anywhere">{{ item.data.issuer }}</td><td><code class="wrap-anywhere">{{ item.data.subject }}</code></td></tr> } @empty { <tr><td colspan="2">No external identities in this page.</td></tr> }</tbody></table></div>
        @if (service.identities.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.identities.loading()" (click)="service.moreIdentities()">More linked identities</button> }
      </section>
    }
  `,
  styles: `.guidance { margin-top: 20px; }`
})
export class AccountPage { readonly service = inject(AccountService) }
