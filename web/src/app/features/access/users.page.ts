import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { AccessActionService } from './access-action.service'
import { UsersService } from './users.service'
import { UserEditorService } from './user-editor.service'
import { UserEditorComponent } from './user-editor.component'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, UserEditorComponent], providers: [UsersService, UserEditorService, AccessActionService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">System administration</span><h1 id="users-heading" tabindex="-1">Users</h1><p class="muted">Manage profiles and account state across this installation.</p></div>@if (service.manage()) { <button type="button" class="button primary" (click)="editor.open()">+ Register user</button> }</div>
    @if (editor.opened()) { <ts-user-editor /> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    <section class="panel" aria-labelledby="user-directory-heading"><div class="section-top"><h2 id="user-directory-heading">User directory</h2><button type="button" class="button secondary" [disabled]="service.users.loading() || service.loading()" (click)="service.loadUsers()">Refresh users</button></div>
      @if (service.users.error()) { <p class="notice error" role="alert">{{ service.users.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">Installation users</caption><thead><tr><th scope="col">User</th><th scope="col">Account</th><th scope="col">Email</th><th scope="col">Last sign-in</th></tr></thead><tbody>
        @for (item of service.users.items(); track item.id) { <tr><td><a [routerLink]="['/settings', 'users', item.id]">{{ item.data.name }}</a><code class="table-subtitle wrap-anywhere">{{ item.id }}</code></td><td><span class="badge" [class.warning]="item.data.disabled">{{ item.data.disabled ? 'Disabled' : 'Enabled' }}</span></td><td>{{ item.data.email || 'No email' }}<span class="table-subtitle muted">{{ item.data.emailVerified ? 'Verified' : 'Not verified' }}</span></td><td>@if (item.data.lastSignIn) { <time [attr.datetime]="item.data.lastSignIn">{{ item.data.lastSignIn | date:'medium':'UTC' }} UTC</time> } @else { <span class="muted">No recorded sign-in</span> }</td></tr> }
        @empty { @if (!service.users.loading() && !service.loading() && !service.users.error() && !service.error()) { <tr><td colspan="4">No readable users in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.loading() || service.users.loading()) { <p class="small" role="status">Loading users…</p> }
      @if (service.users.cursor()) { <p class="small muted">This page covers a bounded scan. More readable users may appear on later pages.</p><button type="button" class="button secondary more-button" [disabled]="service.users.loading()" (click)="service.loadUsers(true)">More users</button> }
    </section>
  `
})
export class UsersPage { readonly service = inject(UsersService); readonly editor = inject(UserEditorService) }
