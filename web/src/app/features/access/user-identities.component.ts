import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { UsersService } from './users.service'

@Component({
  selector: 'ts-user-identities', standalone: true, changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="user-identities-heading"><div class="section-top"><h2 id="user-identities-heading">Linked identities</h2><button type="button" class="button secondary" [disabled]="service.identities.loading()" (click)="service.loadIdentities()">Refresh identities</button></div>
      <p class="small muted">Issuer and subject identify the account. Matching email addresses do not link identities.</p>
      @if (service.identities.error()) { <p class="notice error" role="alert">{{ service.identities.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">User external identities</caption><thead><tr><th scope="col">Issuer</th><th scope="col">Subject</th></tr></thead><tbody>
        @for (item of service.identities.items(); track item.id) { <tr><td class="wrap-anywhere">{{ item.data.issuer }}</td><td><code class="wrap-anywhere">{{ item.data.subject }}</code></td></tr> }
        @empty { @if (!service.identities.loading() && !service.identities.error()) { <tr><td colspan="2">No readable linked identities in this page. Local sign-in does not require an external identity.</td></tr> } }
      </tbody></table></div>
      @if (service.identities.loading()) { <p class="small" role="status">Loading linked identities…</p> }
      @if (service.identities.partial()) { <p class="small muted">This page covers a bounded scan. Continue for more readable identities.</p> }
      @if (service.identities.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.identities.loading()" (click)="service.loadIdentities(true)">More identities</button> }
    </section>
  `
})
export class UserIdentitiesComponent { readonly service = inject(UsersService) }
