import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { AccessActionComponent } from './access-action.component'
import { UsersService } from './users.service'
import { UserEditorService } from './user-editor.service'
import { UserEditorComponent } from './user-editor.component'
import { UserIdentitiesComponent } from './user-identities.component'
import { UserMembershipsComponent } from './user-memberships.component'
import { UserSessionsComponent } from './user-sessions.component'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, AccessActionComponent, UserEditorComponent, UserIdentitiesComponent, UserMembershipsComponent, UserSessionsComponent], providers: [UsersService, UserEditorService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a class="small" routerLink="/settings/users">← User directory</a>
    <div class="page-heading access-heading"><div><span class="eyebrow">Installation administration</span><h1 id="user-details-heading" tabindex="-1">User details</h1><code class="small muted wrap-anywhere">{{ service.userId() }}</code></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.refreshDetails()">Refresh user</button>@if (service.profile(); as item) { @if (service.manage()) { <button type="button" class="button primary" [disabled]="editor.busy() || service.action.busy()" (click)="editor.open(item)">Edit profile</button><button type="button" class="button secondary" [disabled]="editor.busy() || service.action.busy()" (click)="service.state(!item.data.disabled)">{{ item.data.disabled ? 'Restore user' : 'Disable user' }}</button> } }</div></div>
    @if (editor.opened()) { <ts-user-editor /> }
    <ts-access-action />
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.loading()) { <p class="small" role="status">Loading user details…</p> }
    @if (service.profile(); as item) {
      <section class="panel" aria-labelledby="user-profile-heading"><h2 id="user-profile-heading">{{ item.data.name }}</h2><dl class="detail-grid"><dt>Account</dt><dd><span class="badge" [class.warning]="item.data.disabled">{{ item.data.disabled ? 'Disabled' : 'Enabled' }}</span></dd><dt>Email</dt><dd>{{ item.data.email || 'No email' }} · {{ item.data.emailVerified ? 'Verified' : 'Not verified' }}</dd><dt>Created</dt><dd><time [attr.datetime]="item.data.createdAt">{{ item.data.createdAt | date:'medium':'UTC' }} UTC</time></dd><dt>Last sign-in</dt><dd>{{ item.data.lastSignIn ? (item.data.lastSignIn | date:'medium':'UTC') + ' UTC' : 'No recorded sign-in' }}</dd></dl><p class="small muted guidance">An enabled profile still needs a configured sign-in identity. Registration does not grant project access.</p></section>
      <ts-user-identities /><ts-user-memberships /><ts-user-sessions />
    }
  `,
  styleUrl: './access-detail.css'
})
export class UserDetailPage { readonly service = inject(UsersService); readonly editor = inject(UserEditorService) }
