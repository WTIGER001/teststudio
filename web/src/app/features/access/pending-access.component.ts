import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { PendingAccessService } from './pending-access.service'
import { PendingAccessEditorComponent } from './pending-access-editor.component'
import { PendingAccessListComponent } from './pending-access-list.component'

@Component({
  selector: 'ts-pending-access', standalone: true, imports: [PendingAccessEditorComponent, PendingAccessListComponent], providers: [PendingAccessService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="pending-access-heading"><div class="section-top pending-heading"><h2 id="pending-access-heading" tabindex="-1">Pending project access</h2><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || service.busy()" (click)="service.refresh()">Refresh pending access</button>@if (service.canAdd()) { <button type="button" class="button primary" [disabled]="service.busy() || service.access.action.busy()" (click)="service.open()">+ Prepare project access</button> }</div></div>
      <p class="small muted">Add access before a user’s first sign-in. The allowed provider must confirm the verified email. No invitation is sent and no accounts are merged.</p>
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      @if (service.error() && !service.opened()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.loading()) { <p role="status">Loading pending access…</p> }
      @if (service.access.rights().manage && !service.providers().length && !service.loading() && !service.error()) { <p class="small muted">An installation administrator must enable an identity provider before you can prepare access.</p> }
      @if (service.opened()) { <ts-pending-access-editor /> }
      <ts-pending-access-list />
    </section>
  `,
  styles: `.pending-heading .row-actions { gap: 8px; } @media (max-width: 640px) { .pending-heading { flex-direction: column; align-items: stretch; gap: 12px; } .pending-heading .row-actions { justify-content: flex-start; } }`
})
export class PendingAccessComponent { readonly service = inject(PendingAccessService) }
