import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { NotificationEditorService } from './notification-editor.service'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'

@Component({
  selector: 'ts-notification-editor', standalone: true, imports: [ReactiveFormsModule, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="notification-editor-heading">
      <div class="section-top"><h2 id="notification-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit notification channel' : 'Create notification channel' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close editor</button></div>
      <p class="small muted">New settings apply to future runs. Saving assigns delivery ownership to you. Test results remain independent from delivery.</p>
      <ts-wizard-stepper [steps]="['Destination', 'When to notify', 'Review']" [current]="editor.flow.step()" [reached]="editor.flow.reached()" [disabled]="editor.busy()" (selected)="editor.flow.go($event); feedback.reveal()" />
      <form [formGroup]="editor.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
        <ts-wizard-heading [step]="editor.flow.step()" [title]="titles[editor.flow.step()]" />
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
          <fieldset [disabled]="editor.busy()"><legend class="sr-only">Notification channel configuration</legend>
          @if (editor.flow.step() === 0) { <div class="form-grid">
            <div class="field"><label for="channel-name">Channel name</label><input id="channel-name" formControlName="name" required maxlength="256"></div>

            <div class="field"><label for="channel-kind">Transport</label><select id="channel-kind" formControlName="kind"><option value="webhook">Generic webhook</option><option value="discord">Discord webhook</option></select></div>
            <div class="field"><label for="channel-reference">Secret reference</label><select id="channel-reference" formControlName="secretReference" required><option value="">Choose a registered reference</option>@if (editor.unavailableReference()) { <option [value]="editor.unavailableReference()">{{ editor.unavailableReference() }} · Unavailable</option> }@for (reference of editor.references(); track reference.id) { <option [value]="reference.id">{{ reference.name }} ({{ reference.id }})</option> }</select><small>Only registered names appear. The server resolves the destination.</small></div>
          </div><details><summary>Advanced identity options</summary>            <div class="field"><label for="channel-id">Channel identifier</label><input id="channel-id" formControlName="id" required [readOnly]="!!editor.editing()"><small>Permanent identifier for configuration and history.</small></div></details> }
          @if (editor.flow.step() === 1) {
          <fieldset class="choices"><legend>Run triggers</legend><label class="checkbox-field"><input type="checkbox" formControlName="manual"> Manual runs</label><label class="checkbox-field"><input type="checkbox" formControlName="schedule"> Scheduled runs</label><label class="checkbox-field"><input type="checkbox" formControlName="rerun"> Failure reruns</label></fieldset>
          <fieldset class="choices"><legend>Run outcomes</legend><label class="checkbox-field"><input type="checkbox" formControlName="passed"> Passed</label><label class="checkbox-field"><input type="checkbox" formControlName="failed"> Failed</label><label class="checkbox-field"><input type="checkbox" formControlName="timedOut"> Timed out</label><label class="checkbox-field"><input type="checkbox" formControlName="flaky"> Flaky</label><label class="checkbox-field"><input type="checkbox" formControlName="unknown"> Unknown</label></fieldset>
          <label class="checkbox-field"><input type="checkbox" formControlName="includeEvidenceGaps"> Also notify when evidence capture is incomplete</label>
          <details><summary>Advanced filters and delivery options</summary><div class="form-grid">            <div class="field"><label for="channel-suites">Suite identifiers (optional)</label><input id="channel-suites" formControlName="suites" placeholder="All suites"><small>Separate identifiers with commas or spaces.</small></div>
            <div class="field"><label for="channel-environments">Environment identifiers (optional)</label><input id="channel-environments" formControlName="environments" placeholder="All environments"><small>Empty includes targeted and targetless runs.</small></div>
            <div class="field"><label for="channel-stages">Stages (optional)</label><input id="channel-stages" formControlName="stages" placeholder="All stages"><small>Separate exact stage names with commas. Stage does not imply protection.</small></div>
            <div class="field"><label for="channel-attempts">Automatic delivery attempts</label><input id="channel-attempts" type="number" formControlName="maxAttempts" min="1" max="10" step="1" required><small>Includes the first attempt. Explicit retry adds this budget.</small></div>
          </div>
</details> }
          @if (editor.flow.step() === 2) {
          <dl><dt>Channel</dt><dd>{{ editor.form.controls.name.value }}</dd><dt>Destination</dt><dd>{{ editor.destinationName() }} · {{ editor.form.controls.kind.value }}</dd><dt>Run triggers</dt><dd>{{ editor.summary().triggers?.join(', ') || 'Manual, scheduled, and failure reruns' }}</dd><dt>Outcomes</dt><dd>{{ editor.summary().outcomes?.join(', ') || 'Evidence gaps only' }}</dd><dt>Evidence gaps</dt><dd>{{ editor.form.controls.includeEvidenceGaps.value ? 'Included' : 'Not selected' }}</dd><dt>Suites</dt><dd class="wrap-anywhere">{{ editor.form.controls.suites.value || 'All suites' }}</dd><dt>Environments</dt><dd class="wrap-anywhere">{{ editor.form.controls.environments.value || 'All environments' }}</dd><dt>Stages</dt><dd>{{ editor.form.controls.stages.value || 'All stages' }}</dd><dt>Delivery attempts</dt><dd>{{ editor.form.controls.maxAttempts.value }}</dd></dl>
          <p class="notice">An enabled channel sends future matching run summaries to this registered destination. Review its scope before enabling it.</p>
          <label class="checkbox-field"><input type="checkbox" formControlName="enabled"> Enable this channel</label>
          <p class="small muted">Pausing blocks future attempts. Reenabling does not revive earlier messages without an explicit retry. Sent messages cannot be recalled.</p>
          }
          <div class="row-actions wrap-actions">@if (editor.flow.step() > 0) { <button type="button" class="button secondary" (click)="editor.flow.back()">Back</button> }<button type="submit" class="button primary" [disabled]="!editor.referencesReady() || !editor.references().length">{{ editor.flow.step() === 2 ? 'Save notification channel' : 'Continue' }}</button>@if (editor.editing()) { <button type="button" class="button subtle" (click)="editor.reload()">Reload stored configuration</button> }</div>
        </fieldset>
      </form>
      @if (editor.busy()) { <p role="status">Loading channel configuration…</p> }
      @if (editor.referencesReady() && !editor.references().length) { <p class="notice">No notification secret references are registered for this project. Ask the installation operator to add a notification-webhook binding and restart the service.</p> }
      @if (editor.conflict()) { <p class="small">Another configuration change exists. Your edits remain here. Reload stored configuration before retrying.</p> }
    </section>
  `,
  styles: `summary { cursor: pointer; padding: .75rem 0; } details, dl { margin-bottom: 1rem; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; } fieldset { border: 0; margin: 0; padding: 0; min-width: 0; } .choices { margin: 20px 0; display: flex; flex-wrap: wrap; gap: 8px 24px; } .choices legend { margin-bottom: 8px; font-weight: 600; }`
})
export class NotificationEditorComponent {
  readonly editor = inject(NotificationEditorService)
  readonly titles = ['Choose a notification destination', 'Choose when to notify', 'Review and save the channel']
  advance(): void { if (this.editor.flow.step() < 2) this.editor.flow.next(); else void this.editor.submit() }
}
