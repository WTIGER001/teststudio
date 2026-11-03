import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { NotificationChannelsService } from './notification-channels.service'
import { NotificationEditorService } from './notification-editor.service'
import { NotificationEditorComponent } from './notification-editor.component'
import { NotificationDeliveriesComponent } from './notification-deliveries.component'

@Component({
  standalone: true, imports: [FormsModule, NotificationEditorComponent, NotificationDeliveriesComponent], providers: [NotificationChannelsService, NotificationEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Notifications</h1><p class="muted">Send completion summaries and inspect delivery history.</p></div>@if (service.workspace.can('notifications.manage')) { <button type="button" class="button primary" [disabled]="editor.busy()" (click)="editor.open()">+ New channel</button> }</div>
    @if (editor.opened()) { <ts-notification-editor /> }
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    <section class="panel" aria-labelledby="notification-channels-heading">
      <div class="section-top"><h2 id="notification-channels-heading" tabindex="-1">Notification channels</h2><button type="button" class="button secondary" [disabled]="service.loading() || service.busy()" (click)="service.load()">Refresh channels</button></div>
      <label class="checkbox-field"><input type="checkbox" [ngModel]="service.archived()" (ngModelChange)="service.archived.set($event); service.load()"> Include archived channels</label>
      <div class="table-scroll"><table><caption class="sr-only">Project notification channels</caption><thead><tr><th scope="col">Channel / destination</th><th scope="col">State</th><th scope="col">Filters</th><th scope="col">Actions</th></tr></thead><tbody>
        @for (item of service.items(); track item.id) { <tr><td><strong>{{ item.data.name }}</strong><code class="table-subtitle small wrap-anywhere">{{ item.id }}</code><span class="table-subtitle">{{ item.data.kind }} · {{ item.data.secretReference }}</span><span class="table-subtitle muted">Owner: {{ item.data.owner }}</span></td><td><span class="badge" [class.success]="item.data.enabled" [class.warning]="item.data.archived">{{ item.data.archived ? 'Archived' : item.data.enabled ? 'Enabled' : 'Paused' }}</span><span class="table-subtitle muted">{{ item.data.maxAttempts }} automatic attempts</span></td><td><details><summary>Channel scope</summary><dl><dt>Suites</dt><dd>{{ item.data.suiteIds?.join(', ') || 'All suites' }}</dd><dt>Environments</dt><dd>{{ item.data.environmentIds?.join(', ') || 'All environments' }}</dd><dt>Stages</dt><dd>{{ item.data.stages?.join(', ') || 'All stages' }}</dd><dt>Triggers</dt><dd>{{ item.data.triggers?.join(', ') || 'All triggers' }}</dd><dt>Outcomes</dt><dd>{{ item.data.outcomes?.join(', ') || 'Evidence gaps only' }}</dd><dt>Evidence gaps</dt><dd>{{ item.data.includeEvidenceGaps ? 'Included' : 'Not selected' }}</dd></dl></details></td><td>@if (service.workspace.can('notifications.manage')) { <div class="row-actions">@if (item.data.archived) { <button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.action(item, 'restore')">Restore</button> } @else { <button type="button" class="button small subtle" [disabled]="editor.busy()" (click)="editor.open(item)">Edit</button><button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.action(item, item.data.enabled ? 'pause' : 'enable')">{{ item.data.enabled ? 'Pause' : 'Enable' }}</button><button type="button" class="button small subtle" [disabled]="service.busy()" (click)="service.action(item, 'archive')">Archive</button> }</div> }</td></tr> }
        @empty { @if (!service.loading() && !service.error()) { <tr><td colspan="4">No readable channels in this page. Create a channel or include archived channels.</td></tr> } }
      </tbody></table></div>
      @if (service.loading()) { <p class="small" role="status">Loading channels…</p> }
      @if (service.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.loading()" (click)="service.load(true)">Older channels</button> }
    </section>
    <ts-notification-deliveries />
  `,
  styles: `dd { margin: 2px 0 8px; overflow-wrap: anywhere; } dt { font-weight: 600; }`
})
export class NotificationsPage { readonly service = inject(NotificationChannelsService); readonly editor = inject(NotificationEditorService) }
