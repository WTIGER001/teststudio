import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { TimingEstimateComponent } from '../timing/timing-estimate.component'
import { NewRunService } from './new-run.service'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'

@Component({
  selector: 'ts-new-run', standalone: true, imports: [ReactiveFormsModule, TimingEstimateComponent, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], providers: [NewRunService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="new-run-title">
      <h2 id="new-run-title">Start a run</h2><p class="small muted">Use the active saved version and a registered environment. Definitions and settings are captured when the run starts.</p>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      <ts-wizard-stepper [steps]="['Selection', 'Review and run']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.saving()" (selected)="service.flow.go($event)" />
      <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
      <ts-wizard-heading [step]="service.flow.step()" [title]="service.flow.step() === 0 ? 'What would you like to run?' : 'Review this run'" />
      <fieldset [disabled]="service.saving()">
      @if (service.flow.step() === 0) {
        <div class="form-grid">
          <div class="field"><label for="run-suite">Suite</label><select id="run-suite" formControlName="suite" required><option value="">Choose a saved suite</option>@for (suite of service.lookups.suites(); track suite.id) { @if (suite.data.activeReleaseId && !suite.data.archived) { <option [value]="suite.id">{{ suite.data.name }}</option> } }</select></div>
          <div class="field"><label for="run-environment">Target environment</label><select id="run-environment" formControlName="environment"><option value="">No target</option>@for (environment of service.lookups.environments(); track environment.id) { @if (!environment.data.disabled) { <option [value]="environment.id">{{ environment.data.name }} · {{ environment.data.stage }}{{ environment.data.protected ? ' · Protected' : '' }}</option> } }</select><small>Protected targets require permission and safe test definitions.</small></div>
        </div><details><summary>Advanced selection and run options</summary><div class="form-grid">
          <div class="field"><label for="run-saved-selection">Saved selection</label><select id="run-saved-selection" formControlName="selection"><option value="">Use manual identifiers or the full suite</option>@for (selection of service.selections(); track selection.id) { <option [value]="selection.id">{{ selection.data.name }}</option> }</select>@if (service.selectionLoading()) { <small role="status">Loading saved selections…</small> }@if (service.selectionCursor()) { <button class="button secondary small" type="button" [disabled]="service.selectionLoading()" (click)="service.loadSelections(true)">Load more saved selections</button> }</div>
          <div class="field"><label for="run-tests">Test identifiers (optional)</label><input id="run-tests" formControlName="tests" placeholder="HEALTH-001, HEALTH-002"><small>Leave empty to run all eligible tests. Dependencies are included.</small></div>
          <div class="field"><label for="run-timeout">Run time limit (seconds)</label><input id="run-timeout" type="number" formControlName="timeout" min="1" max="3600" step="1" required><small>The local worker accepts limits up to one hour.</small></div>
          <div class="field"><label for="run-retries">Retries per test</label><select id="run-retries" formControlName="retries"><option [ngValue]="0">No retries</option><option [ngValue]="1">1 retry</option><option [ngValue]="2">2 retries</option></select></div>
        </div>
        @if (service.selectionId()) { <section aria-labelledby="run-selection-preview-title"><h3 id="run-selection-preview-title">Review saved selection</h3><p class="small muted">Preview freezes the selected version and test identifiers. Current permissions, lifecycle, dependencies, and target safety still apply when the run starts.</p><button type="button" class="button secondary" [disabled]="service.selectionResolving() || service.saving()" (click)="service.previewSelection()">{{ service.selectionResolving() ? 'Resolving…' : 'Preview saved selection' }}</button>@if (service.selectionPreview(); as preview) { <p role="status">{{ preview.selection.definition.name }} · revision {{ preview.selection.revision }} · {{ preview.resolvedTestIds.length }} tests including dependencies</p><p class="small wrap-anywhere">Version <code>{{ preview.releaseId }}</code></p><ul>@for (id of preview.resolvedTestIds; track id) { <li><code>{{ id }}</code></li> }</ul> }</section> }
        </details>
      } @else {
        @if (service.endpointReview().length) { <div class="notice"><strong>Registered target endpoints</strong><ul>@for (endpoint of service.endpointReview(); track endpoint.name) { <li class="wrap-anywhere"><code>{{ endpoint.name }}</code>: {{ endpoint.address }}</li> }</ul><p class="small">Recorded website tests use their declared endpoint. The default is api. Other origins are blocked by the local website profile.</p></div> }
        <dl><dt>Suite</dt><dd>{{ service.suiteName() }}</dd><dt>Target environment</dt><dd>{{ service.environmentName() }}</dd><dt>Tests</dt><dd>{{ service.selectionPreview()?.selection?.definition?.name || service.form.controls.tests.value || 'All eligible tests, including dependencies' }}</dd><dt>Limits</dt><dd>{{ service.form.controls.timeout.value }} seconds · {{ service.form.controls.retries.value }} retries per test</dd></dl>
        <p class="notice">{{ service.targetWarning() }}</p>
      }
      </fieldset>
        <p class="small muted">Execution location: local worker</p>
        <div class="row-actions"><button class="button secondary" type="button" [disabled]="service.form.invalid || service.estimating() || service.saving() || service.needsSelectionPreview()" (click)="service.estimateTiming()">{{ service.estimating() ? 'Estimating…' : 'Estimate timing' }}</button>@if (service.flow.step() > 0) { <button class="button secondary" type="button" [disabled]="service.saving()" (click)="service.flow.back()">Back</button> }<button class="button primary" type="submit" [disabled]="service.saving()">{{ service.saving() ? 'Starting…' : service.flow.step() === 0 ? 'Review run' : 'Start run' }}</button><button type="button" class="button secondary" [disabled]="service.saving()" (click)="service.refreshReview()">Refresh run review</button></div>
        <ts-timing-estimate [estimate]="service.estimate()" />
      </form>
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } details, dl { margin-bottom: 1.25rem; } summary { padding: .75rem 0; cursor: pointer; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; }`]
})
export class NewRunComponent {
  readonly service = inject(NewRunService)
  advance(): void { if (this.service.flow.step() === 0) this.service.flow.next(); else void this.service.create() }
}
