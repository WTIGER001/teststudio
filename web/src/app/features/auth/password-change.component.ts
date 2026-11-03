import { ChangeDetectionStrategy, Component, inject, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { PasswordChangeService } from './password-change.service'

@Component({
  selector: 'ts-password-change', standalone: true, imports: [ReactiveFormsModule], providers: [PasswordChangeService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-labelledby="password-change-heading"><h2 id="password-change-heading">Change password</h2><p class="hint">Use 15 through 256 characters, at most 1024 bytes. Spaces are significant. The server rejects common passwords.</p>
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      <form [formGroup]="service.form" (ngSubmit)="submit()"><fieldset [disabled]="service.busy() || service.session.authenticating() || service.needsRefresh()"><legend class="sr-only">Your password change</legend>
        <div class="field"><label for="current-password">Current password</label><input id="current-password" type="password" formControlName="currentPassword" autocomplete="current-password" required maxlength="1024"></div>
        <div class="field"><label for="new-password">New password</label><input id="new-password" type="password" formControlName="newPassword" autocomplete="new-password" required maxlength="1024" aria-describedby="password-change-impact"></div>
        <div class="field"><label for="confirm-password">Confirm new password</label><input id="confirm-password" type="password" formControlName="confirmation" autocomplete="new-password" required maxlength="1024"></div>
        <p id="password-change-impact" class="hint">A successful change signs out earlier sessions and gives this browser a new session.</p>
        <button type="submit" class="button primary">Change password</button>
      </fieldset></form>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.needsRefresh()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="refresh()">Refresh session</button> }
    </section>
  `,
  styles: `fieldset { border: 0; padding: 0; margin: 18px 0; min-width: 0; } .hint { color: var(--muted); font-size: 12px; line-height: 1.65; margin: 12px 0; }`
})
export class PasswordChangeComponent {
  readonly service = inject(PasswordChangeService)
  readonly changed = output<void>()
  async submit(): Promise<void> { if (await this.service.submit()) this.changed.emit() }
  async refresh(): Promise<void> { if (await this.service.refresh()) this.changed.emit() }
}
