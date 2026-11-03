import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { NotificationAttemptsComponent } from './notification-attempts.component'
import { NotificationDeliveriesService } from './notification-deliveries.service'

@Component({
  selector: 'ts-notification-deliveries', standalone: true, imports: [DatePipe, RouterLink, NotificationAttemptsComponent], providers: [NotificationDeliveriesService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="notification-deliveries-heading">
      <div class="section-top"><div><h2 id="notification-deliveries-heading" tabindex="-1">Notification deliveries</h2><p class="small muted">Delivery retries do not rerun tests or change their outcomes.</p></div><button type="button" class="button secondary" [disabled]="service.loading() || service.busy()" (click)="service.load()">Refresh deliveries</button></div>
      @if (service.runId()) { <p class="small wrap-anywhere">Run: <code>{{ service.runId() }}</code> · <a [routerLink]="['/projects', service.projectId(), 'notifications']">All project deliveries</a></p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      @if (service.partial() || service.cursor()) { <p class="notice small">This page covers a bounded portion of readable history. Continue to older deliveries for more records. Automatic refresh pauses after loading older records.</p> }
      <div class="table-scroll"><table><caption class="sr-only">Notification delivery history</caption><thead><tr><th scope="col">Channel / run</th><th scope="col">Delivery</th><th scope="col">Run result</th><th scope="col">Next attempt / confirmed</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (item of service.rows(); track item.id) {
          <tr><td><strong>{{ item.data.channelName }}</strong><a class="table-subtitle wrap-anywhere" [routerLink]="['/projects', service.projectId(), 'runs', item.data.runId]">{{ item.data.runId }}</a><code class="table-subtitle small wrap-anywhere">{{ item.id }}</code></td>
            <td><span class="badge" [class.success]="item.data.state === 'delivered'" [class.warning]="item.data.state === 'failed' || item.data.state === 'blocked' || item.data.acceptanceUnknown">{{ item.status }}</span><span class="table-subtitle">{{ item.data.attempts }} / {{ item.data.attemptLimit }} attempts{{ item.data.httpStatus ? ' · HTTP ' + item.data.httpStatus : '' }}</span>@if (item.data.error) { <span class="table-subtitle">{{ item.data.error }}</span> }@if (item.data.acceptanceUnknown) { <span class="table-subtitle">Earlier remote acceptance unknown</span> }</td>
            <td><span class="badge" [class.timeout]="item.data.message.outcome === 'timed-out'">{{ item.data.message.outcome }}</span><span class="table-subtitle muted">{{ item.data.message.trigger }} · {{ item.data.message.evidenceState }} evidence</span><details><summary>Frozen run summary</summary><dl class="summary-grid"><dt>Suite</dt><dd>{{ item.data.message.suiteId }}</dd><dt>Environment</dt><dd>{{ item.data.message.environmentId || 'No target' }}</dd><dt>Stage</dt><dd>{{ item.data.message.stage || 'None' }}</dd><dt>Counts</dt><dd>{{ item.data.message.counts.passed }} passed · {{ item.data.message.counts.failed }} failed · {{ item.data.message.counts.timedOut }} timed out · {{ item.data.message.counts.flaky }} flaky · {{ item.data.message.counts.skipped }} skipped · {{ item.data.message.counts.completed }} / {{ item.data.message.counts.total }} completed</dd><dt>Destination</dt><dd>{{ item.data.destination.channel.kind }} · {{ item.data.destination.channel.secretReference }}</dd></dl></details></td>
            <td>@if (item.data.deliveredAt) { <time [attr.datetime]="item.data.deliveredAt">{{ item.data.deliveredAt | date:'medium':'UTC' }} UTC</time> } @else if (item.data.state === 'pending' || item.data.state === 'retrying') { <time [attr.datetime]="item.data.nextAt">{{ item.data.nextAt | date:'medium':'UTC' }} UTC</time> } @else { <span class="muted">{{ item.data.state === 'sending' ? 'Awaiting checkpoint' : 'Stopped' }}</span> }</td>
            <td><div class="row-actions"><button type="button" class="button small subtle" (click)="service.openHistory(item)">Attempts</button>@if (service.workspace.can('notifications.manage')) { @if (item.retry) { <button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.action(item, 'retry')">Retry delivery</button> }@if (item.cancel) { <button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.action(item, 'cancel')">Stop retries</button> } }</div></td>
          </tr>
        } @empty { @if (!service.loading() && !service.error()) { <tr><td colspan="5">No readable deliveries in this history page. Channels must be enabled before a matching run is queued.</td></tr> } }
      </tbody></table></div>
      @if (service.loading()) { <p class="small" role="status">Loading deliveries…</p> }
      @if (service.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.load(true)">Older deliveries</button> }
    </section>
    <ts-notification-attempts />
  `,
  styles: `.summary-grid { margin: 12px 0; min-width: 200px; } dt { font-weight: 600; margin-top: 8px; } dd { margin: 2px 0; overflow-wrap: anywhere; }`
})
export class NotificationDeliveriesComponent { readonly service = inject(NotificationDeliveriesService) }
