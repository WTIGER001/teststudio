import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { PendingAccessService } from './pending-access.service'
import { pendingBlockReason } from './pending-access.utils'

@Component({
  selector: 'ts-pending-access-list', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.records.error()) { <p class="notice error" role="alert">{{ service.records.error() }}</p> }
    <div class="table-scroll"><table><caption class="sr-only">Pending project role approvals</caption><thead><tr><th scope="col">Verified email and provider</th><th scope="col">Project role</th><th scope="col">State and history</th><th scope="col">Actions</th></tr></thead><tbody>
      @for (row of service.records.items(); track row.id) { <tr><td><strong class="wrap-anywhere">{{ row.data.email }}</strong><span class="cell-detail">{{ row.data.providerName }}</span><code class="cell-detail wrap-anywhere">{{ row.data.issuer }}</code></td><td>{{ row.data.roleName }}<code class="cell-detail wrap-anywhere">{{ row.data.roleId }}</code></td><td><span class="badge" [class.warning]="row.data.state === 'blocked'">{{ row.data.state }}</span><span class="cell-detail">Approved {{ row.data.createdAt | date:'medium':'UTC' }} UTC</span><details><summary>Approval details</summary><dl><dt>Approved by</dt><dd class="wrap-anywhere">{{ row.data.approvedBy }}</dd><dt>Reason</dt><dd class="wrap-anywhere">{{ row.data.reason }}</dd><dt>Provider revision</dt><dd>{{ row.data.providerRevision }}</dd>@if (row.data.activatedUserId) { <dt>Activated user</dt><dd class="wrap-anywhere">{{ row.data.activatedUserId }}</dd><dt>Activated</dt><dd>{{ row.data.activatedAt | date:'medium':'UTC' }} UTC</dd> }</dl></details>@if (row.data.blockReason) { <p class="small muted">{{ blockReason(row.data.blockReason) }}</p> }</td><td>@if (row.data.canCancel) { <button type="button" class="button subtle" [disabled]="service.busy() || service.access.action.busy()" (click)="service.cancel(row)">Cancel pending access for {{ row.data.email }}</button> } @else if (row.data.state === 'activated') { <span class="small muted">Manage the resulting role in Role assignments.</span> } @else { <span class="small muted">Retained history</span> }</td></tr> }
      @empty { <tr><td colspan="4">No readable approvals in this page.</td></tr> }
    </tbody></table></div>
    @if (service.records.partial()) { <p class="small muted">This bounded scan can omit unreadable records. Continue to inspect more history.</p> }
    @if (service.records.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.records.loading()" (click)="service.load(true)">More pending access</button> }
  `,
  styles: `.cell-detail { display: block; margin-top: 6px; font-size: 12px; } details { margin-top: 10px; } summary { cursor: pointer; } dl { font-size: 12px; } dd { margin: 5px 0 12px; }`
})
export class PendingAccessListComponent { readonly service = inject(PendingAccessService); readonly blockReason = pendingBlockReason }
