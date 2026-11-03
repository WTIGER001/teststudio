import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { NotificationEditorService } from './notification-editor.service'

@Component({
  selector: 'ts-notification-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="notification-editor-heading">
      <div class="section-top"><h2 id="notification-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit notification channel' : 'Create notification channel' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close editor</button></div>
      <p class="small muted">New settings apply to future runs. Saving assigns delivery ownership to you. Test results remain independent from delivery.</p>
      <form [formGroup]="editor.form" (ngSubmit)="editor.submit()">
        <fieldset [disabled]="editor.busy()"><legend class="sr-only">Notification channel configuration</legend>
          <div class="form-grid">
            <div class="field"><label for="channel-name">Channel name</label><input id="channel-name" formControlName="name" required maxlength="256"></div>
            <div class="field"><label for="channel-id">Channel identifier</label><input id="channel-id" formControlName="id" required [readOnly]="!!editor.editing()"><small>Permanent identifier for configuration and history.</small></div>
            <div class="field"><label for="channel-kind">Transport</label><select id="channel-kind" formControlName="kind"><option value="webhook">Generic webhook</option><option value="discord">Discord webhook</option></select></div>
            <div class="field"><label for="channel-reference">Secret reference</label><select id="channel-reference" formControlName="secretReference" required><option value="">Choose a registered reference</option>@if (editor.unavailableReference()) { <option [value]="editor.unavailableReference()">{{ editor.unavailableReference() }} · Unavailable</option> }@for (reference of editor.references(); track reference.id) { <option [value]="reference.id">{{ reference.name }} ({{ reference.id }})</option> }</select><small>Only registered names appear. The server resolves the destination.</small></div>
            <div class="field"><label for="channel-suites">Suite identifiers (optional)</label><input id="channel-suites" formControlName="suites" placeholder="All suites"><small>Separate identifiers with commas or spaces.</small></div>
            <div class="field"><label for="channel-environments">Environment identifiers (optional)</label><input id="channel-environments" formControlName="environments" placeholder="All environments"><small>Empty includes targeted and targetless runs.</small></div>
            <div class="field"><label for="channel-stages">Stages (optional)</label><input id="channel-stages" formControlName="stages" placeholder="All stages"><small>Separate exact stage names with commas. Stage does not imply protection.</small></div>
            <div class="field"><label for="channel-attempts">Automatic delivery attempts</label><input id="channel-attempts" type="number" formControlName="maxAttempts" min="1" max="10" step="1" required><small>Includes the first attempt. Explicit retry adds this budget.</small></div>
          </div>
          <fieldset class="choices"><legend>Run triggers</legend><label class="checkbox-field"><input type="checkbox" formControlName="manual"> Manual runs</label><label class="checkbox-field"><input type="checkbox" formControlName="schedule"> Scheduled runs</label><label class="checkbox-field"><input type="checkbox" formControlName="rerun"> Failure reruns</label></fieldset>
          <fieldset class="choices"><legend>Run outcomes</legend><label class="checkbox-field"><input type="checkbox" formControlName="passed"> Passed</label><label class="checkbox-field"><input type="checkbox" formControlName="failed"> Failed</label><label class="checkbox-field"><input type="checkbox" formControlName="timedOut"> Timed out</label><label class="checkbox-field"><input type="checkbox" formControlName="flaky"> Flaky</label><label class="checkbox-field"><input type="checkbox" formControlName="unknown"> Unknown</label></fieldset>
          <label class="checkbox-field"><input type="checkbox" formControlName="includeEvidenceGaps"> Also notify when evidence capture is incomplete</label>
          <label class="checkbox-field"><input type="checkbox" formControlName="enabled"> Enable this channel</label>
          <p class="small muted">Pausing blocks future attempts. Reenabling does not revive earlier messages without an explicit retry. Sent messages cannot be recalled.</p>
          <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="!editor.referencesReady() || !editor.references().length">Save notification channel</button>@if (editor.editing()) { <button type="button" class="button subtle" (click)="editor.reload()">Reload stored configuration</button> }</div>
        </fieldset>
      </form>
      @if (editor.busy()) { <p role="status">Loading channel configuration…</p> }
      @if (editor.referencesReady() && !editor.references().length) { <p class="notice">No notification secret references are registered for this project. Ask the installation operator to add a notification-webhook binding and restart the service.</p> }
      @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
      @if (editor.conflict()) { <p class="small">Another configuration change exists. Your edits remain here. Reload stored configuration before retrying.</p> }
    </section>
  `,
  styles: `fieldset { border: 0; margin: 0; padding: 0; min-width: 0; } .choices { margin: 20px 0; display: flex; flex-wrap: wrap; gap: 8px 24px; } .choices legend { margin-bottom: 8px; font-weight: 600; }`
})
export class NotificationEditorComponent { readonly editor = inject(NotificationEditorService) }
