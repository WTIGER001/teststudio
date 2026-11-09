import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { RunLookupsService } from '../runs/run-lookups.service'
import { SchedulesService } from './schedules.service'
import { ScheduleEditorService } from './schedule-editor.service'
import { ScheduleEditorComponent } from './schedule-editor.component'
import { ScheduleHistoryComponent } from './schedule-history.component'

@Component({
  standalone: true, imports: [DatePipe, FormsModule, RouterLink, ScheduleEditorComponent, ScheduleHistoryComponent],
  providers: [SchedulesService, ScheduleEditorService, RunLookupsService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Schedules</h1><p class="muted">Repeat saved tests and keep a record of every firing.</p></div>
      @if (service.workspace.can('schedules.create')) { <button type="button" class="button primary" [disabled]="editor.busy()" (click)="editor.open()">+ New schedule</button> }
    </div>
    @if (editor.opened()) { <ts-schedule-editor /> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    <section class="panel" aria-labelledby="schedules-heading">
      <div class="section-top"><h2 id="schedules-heading" tabindex="-1">Project schedules</h2><button type="button" class="button secondary" [disabled]="service.loading() || service.busy()" (click)="service.load()">Refresh schedules</button></div>
      <label class="checkbox-field"><input type="checkbox" [ngModel]="service.archived()" (ngModelChange)="service.archived.set($event); service.load()"> Include archived schedules</label>
      @if (service.rows().length) {
        <div class="table-scroll"><table><caption class="sr-only">Project schedules and next execution</caption><thead><tr><th scope="col">Schedule / selection</th><th scope="col">State / cadence</th><th scope="col">Next occurrence</th><th scope="col">Last firing</th><th scope="col">Actions</th></tr></thead><tbody>
          @for (item of service.rows(); track item.id) {
            <tr><td><strong>{{ item.data.name }}</strong><span class="table-subtitle muted">{{ item.id }} · {{ item.data.request.suiteId }}</span><span class="table-subtitle muted">{{ item.data.request.environmentId || 'No target' }} · {{ item.data.request.executionLocation }}</span><span class="table-subtitle muted">{{ item.data.request.releaseId ? 'Pinned version' : 'Follows active version' }} · {{ item.data.owner }}</span></td>
              <td><span class="badge" [class.success]="item.data.enabled" [class.warning]="item.data.archived">{{ item.status }}</span><span class="table-subtitle">{{ item.cadence }}</span><span class="table-subtitle muted">{{ item.data.overlap === 'skip' ? 'Skips overlapping firings' : 'Allows overlapping runs' }}</span></td>
              <td>@if (item.data.nextAt; as next) { <time [attr.datetime]="next">{{ next | date:'medium':'UTC' }} UTC</time> } @else { <span class="muted">Not scheduled</span> }</td>
              <td>@if (item.data.lastAt; as last) { <span class="badge" [class.warning]="item.data.lastState === 'blocked'">{{ item.data.lastState }}</span><time class="table-subtitle muted" [attr.datetime]="last">{{ last | date:'medium':'UTC' }} UTC</time> } @else { <span class="muted">No firing yet</span> }@if (item.data.lastRunId) { <a class="table-subtitle wrap-anywhere" [routerLink]="['/projects', service.projectId(), 'runs', item.data.lastRunId]">Latest run</a> }@if (item.data.lastError) { <span class="table-subtitle">{{ item.data.lastError }}</span> }</td>
              <td><div class="row-actions"><button type="button" class="button small subtle" (click)="service.openHistory(item)">History</button>
                @if (service.workspace.can('schedules.update')) { @if (item.data.archived) { <button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.reviewAction(item, 'restore')">Restore</button> } @else { <button type="button" class="button small subtle" [disabled]="editor.busy()" (click)="editor.open(item)">Edit</button><button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.reviewAction(item, item.data.enabled ? 'pause' : 'enable')">{{ item.data.enabled ? 'Pause' : 'Enable' }}</button> } }
                @if (service.workspace.can('schedules.delete') && !item.data.archived) { <button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.reviewAction(item, 'archive')">Archive</button> }
              </div></td>
            </tr>
          }
        </tbody></table></div>
      } @else if (!service.loading()) { <div class="empty-state"><h3>No schedules found</h3><p>Create a schedule for a saved suite, or include archived schedules.</p></div> }
      @if (service.loading()) { <p class="small" role="status">Loading schedules…</p> }
      @if (service.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.load(true)">Load more schedules</button> }
    </section>
    @if(service.pendingAction();as pending){<section class="panel" aria-label="Review shared schedule action"><h2>Review {{ pending.action }} shared schedule</h2><p>{{ pending.item.data.name }} · Suite {{ pending.item.data.request.suiteId }} · Target {{ pending.item.data.request.environmentId }}</p><p>This affects {{ pending.item.data.request.testIds?.join(', ') || 'all eligible tests and dependencies' }}. Pausing stops future firings, while already queued runs may finish.</p><button class="button primary" type="button" [disabled]="service.busy()" (click)="service.confirmAction()">Confirm shared schedule action</button><button class="button secondary" type="button" [disabled]="service.busy()" (click)="service.pendingAction.set(null)">Cancel</button></section>}
    <ts-schedule-history />
  `
})
export class SchedulesPage { readonly service = inject(SchedulesService); readonly editor = inject(ScheduleEditorService) }
