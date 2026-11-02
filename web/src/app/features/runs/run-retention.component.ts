import { DatePipe, KeyValuePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RunRetentionService } from './run-retention.service'
import { bytesLabel } from './run-retention.utils'

@Component({
  selector: 'ts-run-retention', standalone: true, imports: [DatePipe, KeyValuePipe, ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="retention-title"><div class="section-top"><h2 id="retention-title">Evidence retention</h2><button class="button secondary" type="button" (click)="service.load()" [disabled]="service.busy()">Refresh retention</button></div>
      @if (service.error() && !service.deleting()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.success()) { <p class="notice success" role="status">{{ service.success() }}</p> }
      @if (service.view(); as view) {
        <dl class="detail-grid"><dt>Evidence pin</dt><dd>{{ service.pinned() ? 'Pinned' : view.pin.pinned ? 'Pin expired' : 'Not pinned' }}@if (service.pinned() && view.pin.until) { · until {{ view.pin.until | date:'medium' }} }@if (service.pinned() && !view.pin.until) { · until removed }</dd>
          <dt>Retention processing</dt><dd>{{ view.status.state }}@if (view.status.error) { · {{ view.status.error }} }</dd>
          <dt>Run storage</dt><dd>{{ bytes(view.runQuota.reservedBytes) }} / {{ bytes(view.runQuota.limitBytes) }}</dd><dt>Project storage</dt><dd>{{ bytes(view.projectQuota.reservedBytes) }} / {{ bytes(view.projectQuota.limitBytes) }}</dd>
          @if (view.pin.reason) { <dt>Pin reason</dt><dd>{{ view.pin.reason }}</dd> }
        </dl>
        @if (view.pin.deletionsInProgress) { <p class="notice warning">Storage deletion is in progress. A new pin must wait until it finishes.</p> }
        @if (service.workspace.can('evidence.retain')) {
          <details class="retention-editor"><summary>Manage evidence pin</summary><p class="small muted">Preserve retained files for investigation. A pin cannot restore files that storage has already deleted.</p>
            <form [formGroup]="service.pinForm" (ngSubmit)="service.pin(true)"><div class="form-grid">
              <div class="field"><label for="pin-reason">Pin change reason</label><textarea id="pin-reason" formControlName="reason" rows="2" required maxlength="500"></textarea></div>
              <div class="field"><label for="pin-until">Pin until (optional)</label><input id="pin-until" formControlName="until" type="datetime-local"><small>Leave empty to retain evidence until an authorized user removes the pin.</small></div>
            </div><div class="row-actions wrap-actions"><button class="button primary" type="submit" [disabled]="service.pinForm.invalid || service.busy() || view.pin.deletionsInProgress > 0">{{ service.pinned() ? 'Update evidence pin' : 'Pin evidence' }}</button>
              @if (view.pin.pinned) { <button class="button secondary" type="button" (click)="service.pin(false)" [disabled]="service.pinForm.invalid || service.busy()">Remove evidence pin</button> }
              <button class="button subtle" type="button" (click)="service.reloadPin()" [disabled]="service.busy()">Reload pin settings</button>
            </div></form>
          </details>
        }
        <details class="retention-editor"><summary>Capture policy at execution</summary><dl class="detail-grid"><dt>Logs</dt><dd>{{ view.capture.logs ? 'Enabled' : 'Off' }}</dd><dt>Screenshots</dt><dd>{{ view.capture.screenshots }}</dd><dt>Traces</dt><dd>{{ view.capture.trace }}</dd><dt>Video</dt><dd>{{ view.capture.video }}</dd><dt>Default retention</dt><dd>{{ view.capture.retentionDays }} days</dd><dt>Failure retention</dt><dd>{{ view.capture.failureRetentionDays || view.capture.retentionDays }} days</dd>@for (rule of view.capture.kindRetentionDays | keyvalue; track rule.key) { <dt>{{ rule.key }} retention</dt><dd>{{ rule.value }} days</dd> }</dl></details>
      } @else { <p class="small muted">Loading retention and storage usage…</p> }
    </section>
  `
})
export class RunRetentionComponent { readonly service = inject(RunRetentionService); readonly bytes = bytesLabel }
