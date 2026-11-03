import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject, input, OnChanges, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { LocalCredentialService } from './local-credential.service'

@Component({
  selector: 'ts-local-credential', standalone: true, imports: [DatePipe, ReactiveFormsModule], providers: [LocalCredentialService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="local-credential-heading"><div class="section-top"><h2 id="local-credential-heading" tabindex="-1">Local password credential</h2><button type="button" class="button secondary" [disabled]="service.loading() || service.busy()" (click)="service.load()">Refresh local credential</button></div>
      @if (service.loading()) { <p role="status">Loading local credential…</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      @if (service.record(); as item) {
        @if (!item.data.localAccountsEnabled) { <p class="small muted">Managed local accounts are disabled for this installation.</p> }
        @else {
          <dl class="detail-grid"><dt>Credential</dt><dd>{{ !item.data.configured ? 'Not configured' : item.data.disabled ? 'Disabled' : 'Enabled' }}</dd><dt>Password change</dt><dd>{{ item.data.mustChangePassword ? 'Required at password sign-in' : item.data.configured ? 'Not required' : 'No password configured' }}</dd>@if (item.data.updatedAt) { <dt>Updated</dt><dd>{{ item.data.updatedAt | date:'medium':'UTC' }} UTC</dd> }</dl>
          <p class="small muted">User identifier: <code>{{ item.id }}</code>. Provisioning grants no project roles and links no external identity.</p>
          @if (item.data.canManage) { <div class="row-actions wrap-actions"><button type="button" class="button primary" [disabled]="service.busy() || service.action.busy()" (click)="service.open()">{{ item.data.configured ? 'Reset local password' : 'Set local password' }}</button>@if (item.data.configured) { <button type="button" class="button secondary" [disabled]="service.busy() || service.action.busy()" (click)="service.state(!item.data.disabled)">{{ item.data.disabled ? 'Enable local sign-in' : 'Disable local sign-in' }}</button> }</div> }
        }
      }
      @if (service.opened()) {
        <section class="credential-editor" aria-labelledby="local-password-editor-heading"><div class="section-top"><h3 id="local-password-editor-heading" tabindex="-1">Provision or reset local password</h3><button type="button" class="button subtle" [disabled]="service.busy()" (click)="service.close()">Close password editor</button></div>
          <p class="small muted">Use 15 through 256 characters and at most 1024 bytes. The account must change this password before ordinary password access. A reset preserves disabled state.</p>
          <form [formGroup]="service.form" (ngSubmit)="submit()"><fieldset [disabled]="service.busy() || !service.record()?.data?.canManage"><legend class="sr-only">Administrator local password reset</legend>
            <div class="field"><label for="admin-local-password">Temporary local password</label><input id="admin-local-password" type="password" formControlName="password" autocomplete="new-password" required maxlength="1024"></div>
            <div class="field"><label for="admin-confirm-password">Confirm temporary password</label><input id="admin-confirm-password" type="password" formControlName="confirmation" autocomplete="new-password" required maxlength="1024"></div>
            <div class="field"><label for="admin-password-reason">Credential change reason</label><textarea id="admin-password-reason" formControlName="reason" required maxlength="1024"></textarea></div>
            @if (service.conflict()) { <p class="small">The credential changed or the request result is unknown. Password fields were cleared. Close this editor, refresh credential state, and review before reopening.</p> }
            <button type="submit" class="button primary" [disabled]="service.conflict()">Save local password</button>
          </fieldset></form>
        </section>
      }
    </section>
  `,
  styles: `fieldset { border: 0; margin: 0; padding: 0; min-width: 0; } .credential-editor { margin-top: 24px; padding-top: 24px; border-top: 1px solid var(--border); }`
})
export class LocalCredentialComponent implements OnChanges {
  readonly userId = input.required<string>()
  readonly displayName = input.required<string>()
  readonly changed = output<void>()
  readonly service = inject(LocalCredentialService)
  constructor() { this.service.onStateChanged = () => this.changed.emit() }
  ngOnChanges(): void { this.service.select(this.userId(), this.displayName()) }
  async submit(): Promise<void> { if (await this.service.submit()) this.changed.emit() }
}
