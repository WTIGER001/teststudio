import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { IdentityLinkService } from './identity-link.service'

@Component({
  selector: 'ts-identity-link', standalone: true, imports: [DatePipe, ReactiveFormsModule], providers: [IdentityLinkService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="identity-link-heading"><h2 id="identity-link-heading">Link another sign-in</h2>
      <p class="small muted">Verify this account and the new identity, then confirm the exact issuer and subject. A linked identity keeps this account’s existing permissions.</p>
      @if (service.busy()) { <p role="status">Checking identity link…</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.needsReview()) { <p class="small muted">Review the stored state before another change. If this session expired, sign in again. Unconfirmed requests expire after five minutes.</p><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load()">Review current link state</button>@if (!service.operation()) { <button type="button" class="button secondary cancel" [disabled]="service.busy()" (click)="service.reset()">Clear unavailable link view</button> } }
      @if (service.operation(); as operation) {
        <dl class="detail-grid"><dt>Target provider</dt><dd>{{ operation.targetProvider.name }}</dd><dt>Reason</dt><dd class="wrap-anywhere">{{ operation.reason }}</dd><dt>Expires</dt><dd>{{ operation.expiresAt | date:'medium':'UTC' }} UTC</dd></dl>
        @switch (operation.state) {
          @case ('authenticate-source') { <p>Verify your current account with {{ operation.sourceProvider?.name }}. Choose the identity already linked to this account.</p><a class="button primary" [attr.aria-disabled]="service.busy() || service.needsReview()" [href]="service.authorizePath()" (click)="service.guardNavigation($event)">Verify current account</a> }
          @case ('prove-target') { <p>Current account verified. Sign in with the identity you want to add using {{ operation.targetProvider.name }}.</p><a class="button primary" [attr.aria-disabled]="service.busy() || service.needsReview()" [href]="service.authorizePath()" (click)="service.guardNavigation($event)">Verify new identity</a> }
          @case ('review') {
            <h3>Review the identity to link</h3>
            <dl class="detail-grid"><dt>Exact issuer</dt><dd><code class="wrap-anywhere">{{ operation.identity?.issuer }}</code></dd><dt>Exact subject</dt><dd><code class="wrap-anywhere">{{ operation.identity?.subject }}</code></dd></dl>
            <p class="small muted">Confirmation lets this identity sign in to your account. Earlier sessions are signed out and this browser receives a new session. An identity already owned by another account cannot be transferred.</p>
            <button type="button" class="button primary" [disabled]="service.busy() || service.needsReview()" (click)="service.mutate('confirm')">Confirm identity link</button>
          }
          @case ('complete') { <p class="notice success" role="status">Identity linked. Earlier sessions are invalid.</p><button type="button" class="button secondary" [disabled]="service.busy() || service.needsReview()" (click)="service.reset()">Link another identity</button> }
          @case ('cancelled') { <p role="status">Identity linking cancelled. No identity was added.</p><button type="button" class="button secondary" [disabled]="service.busy() || service.needsReview()" (click)="service.reset()">Start a new identity link</button> }
          @case ('failed') { <p role="status">Identity proof was rejected or cancelled. Use the correct account and a fresh sign-in. Already linked identities cannot be merged.</p><button type="button" class="button secondary" [disabled]="service.busy() || service.needsReview()" (click)="service.reset()">Start a new identity link</button> }
          @case ('expired') { <p role="status">This request expired or its sign-in configuration changed. Start again with fresh authentication.</p><button type="button" class="button secondary" [disabled]="service.busy() || service.needsReview()" (click)="service.reset()">Start a new identity link</button> }
        }
        @if (operation.state === 'authenticate-source' || operation.state === 'prove-target' || operation.state === 'review') { <button type="button" class="button secondary cancel" [disabled]="service.busy() || service.needsReview()" (click)="service.mutate('cancel')">Cancel identity link</button> }
      } @else if (service.options(); as options) {
        @if (!options.canStart) { <p>Sign in with an enabled local password or OIDC provider to link an identity. Local operator confirmation is available only on loopback.</p> }
        @else if (!options.providers.length) { <p>No enabled identity provider is available. An installation administrator can configure one.</p> }
        @else {
          <form [formGroup]="service.form" (ngSubmit)="service.start()"><fieldset [disabled]="service.busy() || service.needsReview()"><legend class="sr-only">Verify an additional sign-in identity</legend>
            <div class="field"><label for="link-provider">New identity provider</label><select id="link-provider" formControlName="providerId" required><option value="">Choose a provider</option>@for (provider of options.providers; track provider.id) { <option [value]="provider.id">{{ provider.name }}</option> }</select></div>
            <div class="field"><label for="link-reason">Identity link reason</label><textarea id="link-reason" formControlName="reason" maxlength="1024" required rows="2"></textarea></div>
            @if (options.sourceMethod === 'password') { <div class="field"><label for="link-password">Current account password</label><input id="link-password" type="password" formControlName="password" autocomplete="current-password" maxlength="1024" required></div> }
            @if (options.sourceMethod === 'local') { <label class="local-confirm"><input type="checkbox" formControlName="confirmLocal" required> I control this local operator account</label> }
            @if (options.sourceMethod === 'oidc') { <p class="small muted">Your current provider will request fresh authentication before you can prove the new identity.</p> }
            <button type="submit" class="button primary">Start identity link</button>
          </fieldset></form>
        }
      }
    </section>
  `,
  styles: `fieldset { border: 0; padding: 0; margin: 18px 0 0; min-width: 0; } p { margin: 16px 0; } .cancel { margin: 12px 0 0 10px; } .local-confirm { display: flex; gap: 10px; align-items: center; margin: 18px 0; } .local-confirm input { width: auto; }`
})
export class IdentityLinkComponent { readonly service = inject(IdentityLinkService) }
