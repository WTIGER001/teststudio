import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { SchedulesService } from './schedules.service'

@Component({
  selector: 'ts-schedule-history', standalone: true, imports: [DatePipe, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.history(); as selected) {
      <section class="panel" aria-labelledby="schedule-history-heading">
        <div class="section-top"><h2 id="schedule-history-heading" tabindex="-1">Firings: {{ selected.data.name }}</h2><button type="button" class="button subtle" (click)="service.closeHistory(); service.focus('schedules-heading')">Close history</button></div>
        <p class="small muted">Triggered firings link to normal runs. Skipped and blocked firings preserve their reason. Times use UTC.</p>
        @if (service.historyError()) { <p class="notice error" role="alert">{{ service.historyError() }}</p> }
        @if (service.firings().length) {
          <div class="table-scroll"><table><caption class="sr-only">Schedule firing history</caption><thead><tr><th scope="col">Due / observed</th><th scope="col">State</th><th scope="col">Run / reason</th><th scope="col">Execution owner</th></tr></thead><tbody>
            @for (item of service.firings(); track item.id) { <tr><td><time [attr.datetime]="item.data.dueAt">{{ item.data.dueAt | date:'medium':'UTC' }}</time><time class="table-subtitle muted" [attr.datetime]="item.data.at">{{ item.data.at | date:'medium':'UTC' }}</time></td><td><span class="badge" [class.warning]="item.data.state === 'blocked'">{{ item.data.state }}</span>@if (item.data.coalesced) { <span class="table-subtitle muted">Missed occurrences coalesced</span> }</td><td>@if (item.data.runId) { <a class="wrap-anywhere" [routerLink]="['/projects', service.projectId(), 'runs', item.data.runId]">{{ item.data.runId }}</a> }@if (item.data.error) { <span class="table-subtitle">{{ item.data.error }}</span> }</td><td>{{ item.data.owner }}<span class="table-subtitle muted">Configuration {{ item.data.generation }}</span></td></tr> }
          </tbody></table></div>
        } @else if (!service.historyLoading()) { <p class="muted">No firings recorded.</p> }
        @if (service.historyLoading()) { <p class="small" role="status">Loading firing history…</p> }
        @if (service.historyCursor()) { <button type="button" class="button secondary more-button" [disabled]="service.historyLoading()" (click)="service.loadHistory(true)">Load older firings</button> }
      </section>
    }
  `
})
export class ScheduleHistoryComponent { readonly service = inject(SchedulesService) }
