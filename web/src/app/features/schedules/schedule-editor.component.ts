import { ScheduleReviewComponent } from '../../shared/probes/schedule-review.component'
import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ScheduleEditorService } from './schedule-editor.service'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'

@Component({
  selector: 'ts-schedule-editor', standalone: true, imports: [ScheduleReviewComponent, ReactiveFormsModule, DatePipe, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel schedule-editor" aria-labelledby="schedule-editor-heading">
      <div class="section-top"><h2 id="schedule-editor-heading" tabindex="-1">{{ editor.editing() ? 'Edit schedule' : 'Create schedule' }}</h2><button type="button" class="button subtle" [disabled]="editor.busy()" (click)="editor.close()">Close editor</button></div>
      <p class="small muted">Triggered firings create normal runs. Saving assigns execution to you. Missed times produce one firing after downtime.</p>
      <ts-wizard-stepper [steps]="['Tests', 'Timing', 'Review']" [current]="editor.flow.step()" [reached]="editor.flow.reached()" [disabled]="editor.busy()" (selected)="editor.flow.go($event); feedback.reveal()" />
      <form [formGroup]="editor.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
        <ts-wizard-heading [step]="editor.flow.step()" [title]="titles[editor.flow.step()]" />
        @if (editor.error()) { <p class="notice error" role="alert">{{ editor.error() }}</p> }
        @if (editor.probe(); as probe) { <p class="notice">Probe selection: {{ probe.name }} · Test {{ probe.testId }} · Target {{ probe.environmentId }} · {{ probe.executionLocation }}. Save and run does not activate a release. Use a reviewed active release or pin its published release below.</p> }
        <fieldset [disabled]="editor.busy()"><legend class="sr-only">Schedule configuration</legend>
          @if (editor.flow.step() === 0) { <div class="form-grid">
            <div class="field"><label for="schedule-name">Schedule name</label><input id="schedule-name" formControlName="name" required maxlength="256"></div>

            <div class="field"><label for="schedule-suite">Suite</label><select id="schedule-suite" formControlName="suite" required><option value="">Choose a suite</option>@for (suite of editor.lookups.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}{{ suite.data.archived ? ' · Archived' : suite.data.activeReleaseId ? '' : ' · Unpublished' }}</option> }</select></div>
            <div class="field"><label for="schedule-environment">Target environment</label><select id="schedule-environment" formControlName="environment"><option value="">No target</option>@for (environment of editor.lookups.environments(); track environment.id) { <option [value]="environment.id">{{ environment.data.name }}{{ environment.data.disabled ? ' · Disabled' : '' }}{{ environment.data.protected ? ' · Protected' : '' }}</option> }</select><small>Required when the suite uses target endpoints.</small></div>
          </div><details><summary>Advanced selection options</summary><div class="form-grid">            <div class="field"><label for="schedule-id">Schedule identifier</label><input id="schedule-id" formControlName="id" required [readOnly]="!!editor.editing()"><small>The permanent identifier connects the schedule to its history.</small></div>            <div class="field"><label for="schedule-location">Execution location</label><input id="schedule-location" formControlName="location" required></div>
            <div class="field"><label for="schedule-release">Release pin (optional)</label><input id="schedule-release" formControlName="release" placeholder="Follow active release"><small>Leave empty to use the active release at each firing.</small></div>
            <div class="field full-span"><label for="schedule-tests">Test identifiers (optional)</label><input id="schedule-tests" formControlName="ids" placeholder="All enabled tests"><small>Separate identifiers with commas or spaces. Dependencies follow normal selection rules.</small></div>
</div></details> }
          @if (editor.flow.step() === 1) { <div class="form-grid">
            <div class="field"><label for="schedule-kind">Cadence</label><select id="schedule-kind" formControlName="kind"><option value="interval">Fixed interval</option><option value="cron">Cron expression</option></select></div>
            @if (editor.form.controls.kind.value === 'interval') {
              <div class="field"><label for="schedule-interval">Interval (seconds)</label><input id="schedule-interval" type="number" formControlName="interval" min="5" max="2592000" required><small>5 seconds through 30 days. Intervals use UTC.</small></div>
            } @else {
              <div class="field"><label for="schedule-cron">Cron expression</label><input id="schedule-cron" formControlName="cron" required placeholder="0 2 * * *"><small>Minute · hour · day of month · month · day of week.</small></div>
              <div class="field"><label for="schedule-timezone">Timezone</label><input id="schedule-timezone" formControlName="timezone" required placeholder="UTC"><small>Use an IANA name, such as America/New_York.</small></div>
            }
          </div><details><summary>Advanced run options</summary><div class="form-grid">            <div class="field"><label for="schedule-overlap">When the previous run is active</label><select id="schedule-overlap" formControlName="overlap"><option value="skip">Skip this firing</option><option value="allow">Allow another run</option></select></div>
            <div class="field"><label for="schedule-timeout">Run time limit (seconds)</label><input id="schedule-timeout" type="number" formControlName="timeout" min="1" max="3600" required></div>
            <div class="field"><label for="schedule-retries">Retries per test</label><input id="schedule-retries" type="number" formControlName="retries" min="0" max="2" required></div>
          </div></details> }
          @if (editor.flow.step() === 2) {
          <ts-schedule-review [name]="editor.form.controls.name.value" [suite]="editor.suiteName()" [target]="editor.targetName()" [cadence]="editor.form.controls.kind.value === 'interval' ? 'Every '+editor.form.controls.interval.value+' seconds · UTC' : editor.form.controls.cron.value+' · '+editor.form.controls.timezone.value" [tests]="editor.form.controls.ids.value" [release]="editor.form.controls.release.value" [options]="editor.form.controls.location.value+' · '+editor.form.controls.timeout.value+' seconds · '+editor.form.controls.retries.value+' retries · '+editor.form.controls.overlap.value" />
          <p class="notice">Enabling this schedule repeatedly starts runs against the selected target. Protected targets require safe tests and protected execution permission.</p>
          <label class="checkbox-field"><input type="checkbox" formControlName="enabled"> Enable this schedule</label>
          @if (editor.conflict()) { <p class="small">Another configuration change exists. Reload stored configuration before retrying.</p> }
          }
          <div class="row-actions wrap-actions">@if (editor.flow.step() > 0) { <button type="button" class="button secondary" (click)="editor.flow.back()">Back</button> }@if (editor.flow.step() === 2) { <button type="button" class="button secondary" (click)="editor.submit(true); feedback.reveal()">Preview schedule</button> }<button type="submit" class="button primary">{{ editor.flow.step() === 2 ? 'Save schedule' : 'Continue' }}</button>@if (editor.editing()) { <button type="button" class="button subtle" (click)="editor.reload()">Reload stored configuration</button> }</div>
        </fieldset>
      </form>
      @if (editor.busy()) { <p role="status">Checking schedule configuration…</p> }
      @if (editor.preview(); as preview) {
        <section class="schedule-preview" aria-labelledby="schedule-preview-heading"><h3 id="schedule-preview-heading">Schedule preview</h3><p class="small muted">Next five occurrences in UTC. Preview does not create a run.</p>
          <ol>@for (at of preview.next; track at) { <li><time [attr.datetime]="at">{{ at | date:'medium':'UTC' }} UTC</time></li> }</ol>
          <p class="small">{{ preview.timing.totalTests }} selected {{ preview.timing.totalTests === 1 ? 'test' : 'tests' }} · {{ preview.timing.readyTests }} timing baselines ready · {{ preview.timing.testTimeoutBudgetMs / 1000 }} seconds of attempt timeout budget.</p>
          @if (preview.timing.provisional) { <p class="small muted">Duration estimates remain provisional while comparable history learns.</p> }
        </section>
      }
    </section>
  `,
  styles: `summary { padding: .75rem 0; cursor: pointer; } details, dl { margin-bottom: 1rem; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; } fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } .full-span { grid-column: 1 / -1; } .schedule-preview { margin-top: 24px; border-top: 1px solid var(--border); padding-top: 18px; } li { margin: 8px 0; }`
})
export class ScheduleEditorComponent {
  readonly editor = inject(ScheduleEditorService)
  readonly titles = ['Choose what to run', 'Choose when it runs', 'Review and save the schedule']
  advance(): void { if (this.editor.flow.step() < 2) this.editor.flow.next(); else void this.editor.submit() }
}
