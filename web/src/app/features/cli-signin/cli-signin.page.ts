import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { CLISignInService } from './cli-signin.service'

@Component({
  standalone: true, imports: [DatePipe, ReactiveFormsModule, RouterLink], providers: [CLISignInService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-header"><div><span class="eyebrow">Your account</span><h1>Confirm CLI sign-in</h1><p class="muted">Connect a terminal session to TestStudio using your current account.</p></div></div>
    <section class="panel confirmation" aria-labelledby="cli-confirmation-title"><h2 id="cli-confirmation-title">Review this sign-in</h2>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      @if (service.view(); as view) {
        <dl class="detail-grid"><dt>Client</dt><dd>{{ view.clientName }}</dd><dt>Account</dt><dd>{{ service.account.user()?.name }} <span class="small muted">({{ service.account.user()?.id }})</span></dd><dt>Request expires</dt><dd>{{ view.expiresAt | date:'medium':'UTC' }} UTC</dd><dt>State</dt><dd>{{ view.state }}</dd></dl>
        @if (view.state === 'pending') {
          <p>Approve only a sign-in you start in your terminal. Enter the terminal code to confirm this client.</p><p class="small muted">The CLI uses your current roles and project access. It receives a separate managed session. Your browser stays signed in.</p>
          <form [formGroup]="service.form" (ngSubmit)="service.confirm(true)"><fieldset [disabled]="service.busy()"><div class="field"><label for="cli-terminal-code">Terminal code</label><input id="cli-terminal-code" formControlName="code" required autocomplete="off" autocapitalize="characters" spellcheck="false" placeholder="1234-ABCD" maxlength="9"></div><div class="row-actions"><button type="submit" class="button primary" [disabled]="service.code.invalid">Approve CLI sign-in</button><button type="button" class="button secondary" (click)="service.confirm(false)">Decline CLI sign-in</button></div></fieldset></form>
        } @else { <p class="muted">This request no longer accepts another approval. Use the terminal to start a new request when needed.</p> }
      }
      <div class="row-actions"><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.refresh()">Refresh sign-in request</button><a class="button secondary" routerLink="/account">Your account</a></div>
    </section>
  `,
  styles: [`.confirmation { max-width: 48rem; } fieldset { border: 0; padding: 0; margin: 1.25rem 0; min-width: 0; } form .field { max-width: 18rem; margin-bottom: 1rem; } input { font-family: ui-monospace, monospace; text-transform: uppercase; } .row-actions { flex-wrap: wrap; } dd { overflow-wrap: anywhere; }`]
})
export class CLISignInPage { readonly service = inject(CLISignInService) }
