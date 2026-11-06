import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { JsonPipe } from '@angular/common'
import { ReactiveFormsModule } from '@angular/forms'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'
import { TimingPolicyEditorComponent } from '../timing/timing-policy-editor.component'
import { ProbeReviewComponent } from '../../shared/probes/probe-review.component'
import { HelpAnchorDirective } from '../help/help-anchor.directive'
import { HTTPCheckEditorComponent } from './http-check-editor.component'
import { HTTPSWizardService } from './https-wizard.service'
import { ProbeTargetService } from './probe-target.service'
import { ProbeTargetComponent } from './probe-target.component'

@Component({
  selector: 'ts-https-wizard', standalone: true, imports: [ProbeReviewComponent, HelpAnchorDirective, ReactiveFormsModule, JsonPipe, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective, HTTPCheckEditorComponent, TimingPolicyEditorComponent, ProbeTargetComponent], providers: [ProbeTargetService, HTTPSWizardService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ts-wizard-stepper [steps]="titles" [current]="wizard.flow.step()" [reached]="wizard.flow.reached()" [disabled]="wizard.busy() || wizard.locked()" (selected)="wizard.flow.go($event); feedback.reveal()" />
    <form [formGroup]="wizard.editor.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
      <ts-wizard-heading [title]="headings[wizard.flow.step()]" [step]="wizard.flow.step()" />
      @if (wizard.error()) { <p class="notice error" role="alert">{{ wizard.error() }}</p> }
      @if (wizard.progress()) { <p role="status">{{ wizard.progress() }}</p> }
      <fieldset [disabled]="wizard.busy() || wizard.locked()">
      @if (wizard.flow.step() === 0) {
        <div class="field"><label for="test-title">Title</label><input id="test-title" formControlName="title" required placeholder="Homepage is available" autocomplete="off"></div>
        <div class="field"><label for="test-suite">Suite</label><select id="test-suite" formControlName="suite" required><option value="">Choose a suite</option>@for (suite of wizard.editor.suites(); track suite.id) { @if (!suite.data.archived) { <option [value]="suite.id">{{ suite.data.name }}</option> } }</select></div>
        <details><summary>Advanced details</summary><div class="field"><label for="test-id">Permanent identifier</label><input id="test-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}"><small>Generated from the title. You can replace it before saving.</small></div><div class="field"><label for="test-category">Category</label><input id="test-category" formControlName="category" required></div></details>
      }
      @if (wizard.flow.step() === 1) {
        <ts-probe-target />
        <p class="request-url wrap-anywhere">{{ wizard.requestURL() }}</p>
        <ts-http-check-editor [compact]="true" [addressOnly]="wizard.targets.direct()" [endpoints]="wizard.endpoints()" />
        <details><summary>Advanced timeout and slowdown policy</summary><div class="field"><label for="test-timeout">Catalog timeout ceiling (seconds)</label><input id="test-timeout" type="number" formControlName="timeout" min="1" max="3600" step="1" required></div><ts-timing-policy-editor /></details>
      }
      @if (wizard.flow.step() === 2) {
        <section tabindex="-1" tsHelpAnchor="probe-review"><ts-probe-review [title]="wizard.editor.form.controls.title.value" [suite]="wizard.suiteName()" [url]="wizard.requestURL()" [method]="wizard.editor.checkEditor.form.controls.method.value" [statuses]="wizard.editor.checkEditor.form.controls.statusCodes.value" [timeout]="wizard.editor.form.controls.timeout.value" /></section>
        @if (wizard.target(); as target) { <p class="notice">Target: {{ target.data.name }} · {{ target.data.stage }} · {{ target.data.protected ? 'Protected — safe tests and protected execution permission required.' : 'Not protected.' }}</p> }
        @if (wizard.targets.needsTarget()) {
          <section class="notice" aria-label="Target configuration to save"><h4>Target configuration to save</h4><p>Saving this probe also {{ wizard.targets.targetSaved() ? 'keeps the saved' : 'registers a' }} target for {{ wizard.targets.parsed()?.origin }} under {{ wizard.targets.applicationName() }}.</p><p>{{ wizard.targets.stage.value }} · {{ wizard.targets.protected.value ? 'Protected — safe tests and protected execution permission required.' : 'Not protected.' }}</p>@if (wizard.targets.needsApplication()) { <p>{{ wizard.targets.applicationSaved() ? 'Application saved' : 'A new application will also be created' }}: {{ wizard.targets.applicationName() }}.</p> }<p class="small">Save draft stores this configuration without contacting the website. Save and run starts the selected check on this installation.</p></section>
        }
        <div class="field"><label for="test-safety">Production-safety label</label><select id="test-safety" formControlName="safety"><option value="needs-review">Needs review</option><option value="safe">Safe</option><option value="unsafe">Unsafe</option></select><small>GET and HEAD requests can have side effects. Confirm safety based on the target application.</small></div>
        <div class="field"><label for="test-rationale">Safety rationale</label><textarea id="test-rationale" formControlName="rationale" required rows="2"></textarea></div>
        <details><summary>Review advanced settings</summary>
          <dl><dt>Identifier</dt><dd class="wrap-anywhere">{{ wizard.editor.form.controls.id.value }}</dd><dt>Category</dt><dd>{{ wizard.editor.form.controls.category.value }}</dd><dt>Required text</dt><dd>{{ wizard.editor.checkEditor.form.controls.requiredText.value || 'None' }}</dd><dt>Forbidden text</dt><dd>{{ wizard.editor.checkEditor.form.controls.forbiddenText.value || 'None' }}</dd><dt>Expected response headers</dt><dd><pre>{{ wizard.editor.checkEditor.form.controls.headers.value || 'None' }}</pre></dd><dt>Expected JSON fields</dt><dd><pre>{{ wizard.editor.checkEditor.form.controls.json.value || 'None' }}</pre></dd><dt>Response time limit</dt><dd>{{ wizard.editor.checkEditor.form.controls.maxResponseMs.value }} ms (0 disables)</dd><dt>Minimum certificate lifetime</dt><dd>{{ wizard.editor.checkEditor.form.controls.minCertificateDays.value }} days (0 disables)</dd><dt>Slowdown and adaptive timeout</dt><dd>{{ wizard.editor.timingEditor.override.value ? 'Custom policy' : 'Inherit suite policy' }}</dd></dl>
          @if (wizard.editor.timingEditor.override.value) { <pre>{{ wizard.editor.timingEditor.policy() | json }}</pre> }
        </details>
      }
      </fieldset>
      @if (wizard.flow.step() === 2) { <p class="small muted">Save draft adds this test to the catalog. Save and run also validates and publishes the suite snapshot without changing its active release, then runs this test and its dependencies on this installation.</p> }
      <div class="row-actions">@if (wizard.flow.step() > 0 && !wizard.locked()) { <button class="button secondary" type="button" [disabled]="wizard.busy()" (click)="wizard.flow.back()">Back</button> }
      <button class="button primary" type="submit" [disabled]="wizard.busy()">{{ wizard.flow.step() === 2 ? wizard.saved() ? 'Open saved draft' : wizard.targets.locked() ? 'Retry save' : 'Save draft' : 'Continue' }}</button>
      @if (wizard.flow.step() === 2 && wizard.canRun()) { <button class="button primary" type="button" [disabled]="wizard.busy()" (click)="wizard.save(true); feedback.reveal()">{{ wizard.saved() ? 'Retry run' : 'Save and run' }}</button> }
      </div>
    </form>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } details, dl { margin-bottom: 1.25rem; } summary { cursor: pointer; padding: .75rem 0; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; } pre { white-space: pre-wrap; max-width: 100%; font-size: 12px; } .request-url { font-family: monospace; }`]
})
export class HTTPSWizardComponent {
  readonly wizard = inject(HTTPSWizardService)
  readonly titles = ['Name', 'Request', 'Review']
  readonly headings = ['Name your HTTPS probe', 'What should the request check?', 'Review and save your test']
  advance(): void { if (this.wizard.flow.step() < 2) this.wizard.flow.next(); else void this.wizard.save(false) }
}
