import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { UsersService } from './users.service'

@Component({
  selector: 'ts-user-sessions', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="user-sessions-heading"><div class="section-top"><h2 id="user-sessions-heading" tabindex="-1">Active sessions</h2><button type="button" class="button secondary" [disabled]="service.sessions.loading()" (click)="service.loadSessions()">Refresh sessions</button></div>
      <p class="small muted">Sessions can expire before their absolute limit through inactivity or authentication expiry. Refresh for current status.</p>
      @if (service.sessions.error()) { <p class="notice error" role="alert">{{ service.sessions.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">User active sessions</caption><thead><tr><th scope="col">Provider / session</th><th scope="col">Created / last seen</th><th scope="col">Absolute / authentication expiry</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (item of service.sessions.items(); track item.id) { <tr><td>{{ item.data.provider }}@if (item.data.client) { <span class="table-subtitle muted">{{ item.data.clientName || item.data.client }}</span> }@if (item.data.current) { <span class="badge success table-subtitle">Current browser</span> }<details><summary>Session identifier</summary><code class="wrap-anywhere">{{ item.id }}</code></details></td><td><time [attr.datetime]="item.data.createdAt">{{ item.data.createdAt | date:'medium':'UTC' }} UTC</time><time class="table-subtitle muted" [attr.datetime]="item.data.lastSeen">{{ item.data.lastSeen | date:'medium':'UTC' }} UTC</time></td><td><time [attr.datetime]="item.data.expiresAt">{{ item.data.expiresAt | date:'medium':'UTC' }} UTC</time><time class="table-subtitle muted" [attr.datetime]="item.data.authenticationUntil">{{ item.data.authenticationUntil | date:'medium':'UTC' }} UTC</time></td><td>@if (service.manage()) { <button type="button" class="button small subtle" [disabled]="service.action.busy()" (click)="service.revoke(item)">Revoke session</button> }</td></tr> }
        @empty { @if (!service.sessions.loading() && !service.sessions.error()) { <tr><td colspan="4">No readable active sessions in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.sessions.loading()) { <p class="small" role="status">Loading sessions…</p> }
      @if (service.sessions.partial()) { <p class="small muted">This page covers a bounded scan. Continue for more active sessions.</p> }
      @if (service.sessions.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.sessions.loading()" (click)="service.loadSessions(true)">More sessions</button> }
    </section>
  `
})
export class UserSessionsComponent { readonly service = inject(UsersService) }
