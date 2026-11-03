import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { UsersService } from './users.service'
import { groupRoute } from './access.utils'

@Component({
  selector: 'ts-user-memberships', standalone: true, imports: [DatePipe, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="user-memberships-heading"><div class="section-top"><h2 id="user-memberships-heading">Group memberships</h2><button type="button" class="button secondary" [disabled]="service.memberships.loading()" (click)="service.loadMemberships()">Refresh memberships</button></div>
      <p class="small muted">Membership does not grant access without a role assignment or grant. Open a group to manage permitted local membership.</p>
      @if (service.memberships.error()) { <p class="notice error" role="alert">{{ service.memberships.error() }}</p> }
      <div class="table-scroll"><table><caption class="sr-only">User group memberships</caption><thead><tr><th scope="col">Group / ownership</th><th scope="col">Source</th><th scope="col">Activity / expiry</th></tr></thead><tbody>
        @for (item of service.memberships.items(); track item.id) { <tr><td><a [routerLink]="route({ id: item.data.groupId, projectId: item.data.projectId })">{{ item.data.groupName || item.data.groupId }}</a><code class="table-subtitle wrap-anywhere">{{ item.data.groupId }}</code><span class="table-subtitle muted">{{ item.data.projectId || 'Installation' }}</span></td><td class="wrap-anywhere">{{ item.data.source }}<span class="table-subtitle muted">{{ item.data.readOnly ? 'Externally managed · Read only' : 'Locally managed' }}</span></td><td><span class="badge" [class.success]="item.data.active">{{ item.data.active ? 'Active' : 'Inactive' }}</span>@if (item.data.expiresAt) { <time class="table-subtitle muted" [attr.datetime]="item.data.expiresAt">{{ item.data.expiresAt | date:'medium':'UTC' }} UTC</time> } @else { <span class="table-subtitle muted">No expiry</span> }</td></tr> }
        @empty { @if (!service.memberships.loading() && !service.memberships.error()) { <tr><td colspan="3">No readable memberships in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.memberships.loading()) { <p class="small" role="status">Loading group memberships…</p> }
      @if (service.memberships.partial()) { <p class="small muted">This page covers a bounded scan. Continue for more readable memberships.</p> }
      @if (service.memberships.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.memberships.loading()" (click)="service.loadMemberships(true)">More memberships</button> }
    </section>
  `
})
export class UserMembershipsComponent { readonly service = inject(UsersService); readonly route = groupRoute }
