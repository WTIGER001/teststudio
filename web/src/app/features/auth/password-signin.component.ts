import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { PasswordSigninService } from './password-signin.service'

@Component({
  selector: 'ts-password-signin', standalone: true, imports: [ReactiveFormsModule], providers: [PasswordSigninService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-labelledby="password-signin-heading"><h2 id="password-signin-heading">Sign in with a password</h2><p class="hint">Use your internal user identifier. Email addresses do not link accounts.</p>
      <form [formGroup]="service.form" (ngSubmit)="service.submit()"><fieldset [disabled]="service.busy() || service.session.authenticating()"><legend class="sr-only">Local account sign-in</legend>
        <div class="field"><label for="local-user-id">User identifier</label><input id="local-user-id" formControlName="userId" autocomplete="username" autocapitalize="none" spellcheck="false" required maxlength="128"></div>
        <div class="field"><label for="local-password">Password</label><input id="local-password" type="password" formControlName="password" autocomplete="current-password" required maxlength="1024"></div>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        <button type="submit" class="button primary full">Sign in with password</button>
        @if (service.error()) { <button type="button" class="button secondary full" (click)="service.refresh()">Refresh sign-in</button> }
      </fieldset></form>
    </section>
  `,
  styles: `:host { display: block; margin: 24px 0; } fieldset { border: 0; margin: 0; padding: 0; min-width: 0; } .hint { color: var(--muted); font-size: 12px; line-height: 1.65; margin: 10px 0 18px; }`
})
export class PasswordSigninComponent { readonly service = inject(PasswordSigninService) }
