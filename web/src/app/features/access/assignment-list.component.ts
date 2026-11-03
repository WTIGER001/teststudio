import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { AssignmentsService } from './assignments.service'

@Component({
  selector: 'ts-assignment-list', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="assignment-list-heading"><div class="section-top"><h2 id="assignment-list-heading">Role assignments</h2><button type="button" class="button secondary" [disabled]="service.assignments.loading() || service.loading() || !service.rights().read" (click)="service.loadAssignments()">Refresh assignments</button></div>
      @if (service.assignments.error()) { <p class="notice error" role="alert">{{ service.assignments.error() }}</p> }
      <div class="table-scroll" tabindex="0" role="region" aria-label="Scrollable role assignments"><table><caption class="sr-only">Role assignments in the selected scope</caption><thead><tr><th scope="col">Principal</th><th scope="col">Role</th><th scope="col">Scope</th><th scope="col">Last managed change</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (item of service.assignments.items(); track item.id) { <tr><td class="wrap-anywhere">{{ item.data.principalName }}<code class="table-subtitle">{{ item.data.principalId }}</code><span class="table-subtitle muted">{{ item.data.principalKind === 'user' ? 'User' : 'Group' }} · {{ item.data.principalActive ? 'Active' : 'Inactive' }}</span></td><td class="wrap-anywhere">{{ item.data.roleName }}@if (item.data.roleDisabled) { <span class="table-subtitle badge warning">Role disabled</span> }</td><td class="wrap-anywhere">{{ item.data.bindingValues.team || 'Installation' }}</td><td>@if (item.updatedAt) { <time [attr.datetime]="item.updatedAt">{{ item.updatedAt | date:'medium':'UTC' }} UTC</time> } @else { <span class="muted">Bootstrap or operator-managed</span> }</td><td>@if (service.rights().manage && item.data.canRevoke) { <button type="button" class="button subtle" [disabled]="service.action.busy()" (click)="service.remove(item)">Remove role</button> } @else { <span class="muted">Read only</span> }</td></tr> }
        @empty { @if (!service.assignments.loading() && !service.loading() && !service.error() && !service.assignments.error()) { <tr><td colspan="5">No readable role assignments in this page.</td></tr> } }
      </tbody></table></div>
      @if (service.loading() || service.assignments.loading()) { <p class="small" role="status">Loading role assignments…</p> }
      @if (service.assignments.partial()) { <p class="small muted">This page covers a bounded scan. More readable assignments may appear on later pages.</p> }
      @if (service.assignments.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.assignments.loading()" (click)="service.loadAssignments(true)">More assignments</button> }
      <p class="small muted guidance">Removing one assignment preserves other roles and group memberships. Inactive principals and disabled roles remain visible for review.</p>
    </section>
  `,
  styles: `table { min-width: 820px; } td:first-child { min-width: 180px; } td:nth-child(2) { min-width: 140px; } td:nth-child(3) { min-width: 150px; }`
})
export class AssignmentListComponent { readonly service = inject(AssignmentsService) }
