import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { NotificationDeliveriesService } from './notification-deliveries.service'

@Component({
  selector: 'ts-notification-attempts', standalone: true, imports: [DatePipe], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.attemptsError() && !service.selected()) { <p class="notice error" role="alert">{{ service.attemptsError() }}</p> }
    @if (service.selected(); as item) {
      <section class="panel" aria-labelledby="notification-attempts-heading">
        <div class="section-top"><div><h2 id="notification-attempts-heading" tabindex="-1">Delivery attempts</h2><code class="small wrap-anywhere">{{ item.id }}</code></div><button type="button" class="button subtle" (click)="service.closeHistory(); service.focus('notification-deliveries-heading')">Close attempts</button></div>
        <p class="small">{{ item.data.channelName }} · {{ item.data.state }} · {{ item.data.attempts }} / {{ item.data.attemptLimit }} attempts</p>
        @if (item.data.acceptanceUnknown) { <p class="notice warning">An earlier attempt may have reached the receiver before its checkpoint. Later confirmation does not erase that uncertainty. Receivers can deduplicate using this delivery identifier.</p> }
        @if (service.attemptsError()) { <p class="notice error" role="alert">{{ service.attemptsError() }}</p> }
        <div class="table-scroll"><table><caption class="sr-only">Retained delivery attempt history</caption><thead><tr><th scope="col">Attempt</th><th scope="col">State / response</th><th scope="col">Started / finished</th><th scope="col">Diagnostic</th></tr></thead><tbody>
          @for (attempt of service.attempts(); track attempt.id) { <tr><td>{{ attempt.data.number }}</td><td><span class="badge" [class.success]="attempt.data.state === 'delivered'" [class.warning]="attempt.data.state !== 'delivered'">{{ attempt.data.state }}</span><span class="table-subtitle">{{ attempt.data.httpStatus ? 'HTTP ' + attempt.data.httpStatus : 'No retained HTTP status' }}</span></td><td><time [attr.datetime]="attempt.data.startedAt">{{ attempt.data.startedAt | date:'medium':'UTC' }} UTC</time>@if (attempt.data.finishedAt) { <time class="table-subtitle muted" [attr.datetime]="attempt.data.finishedAt">{{ attempt.data.finishedAt | date:'medium':'UTC' }} UTC</time> } @else { <span class="table-subtitle muted">No finish checkpoint</span> }</td><td>{{ attempt.data.error || '—' }}@if (attempt.data.acceptanceUnknown) { <span class="table-subtitle">Remote acceptance unknown</span> }</td></tr> }
          @empty { @if (!service.attemptsLoading() && !service.attemptsError()) { <tr><td colspan="4">No delivery attempts are recorded yet.</td></tr> } }
        </tbody></table></div>
        @if (service.attemptsLoading()) { <p class="small" role="status">Loading delivery attempts…</p> }
        <div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.attemptsLoading()" (click)="service.loadHistory()">Refresh attempts</button>@if (service.attemptsCursor()) { <button type="button" class="button secondary" [disabled]="service.attemptsLoading()" (click)="service.loadHistory(true)">Older attempts</button> }</div>
      </section>
    }
  `
})
export class NotificationAttemptsComponent { readonly service = inject(NotificationDeliveriesService) }
