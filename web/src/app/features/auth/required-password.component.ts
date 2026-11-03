import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { SessionService } from '../../core/session.service'
import { PasswordChangeComponent } from './password-change.component'

@Component({
  selector: 'ts-required-password', standalone: true, imports: [PasswordChangeComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `<section class="signin-card" aria-labelledby="required-password-heading"><span class="eyebrow">{{ session.user()?.name }}</span><h1 id="required-password-heading" tabindex="-1">Change your password</h1><p class="hint">Your administrator requires a new password before you can open the workspace.</p><ts-password-change /></section>`,
  styles: `.signin-card h1 { font-size: 30px; letter-spacing: -.6px; } .hint { color: var(--muted); font-size: 13px; line-height: 1.7; }`
})
export class RequiredPasswordComponent { readonly session = inject(SessionService) }
