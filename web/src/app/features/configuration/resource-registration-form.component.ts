import { ChangeDetectionStrategy, Component, effect, inject, output, untracked } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ConfigurationService } from './configuration.service'
import { RegistrationWizardService } from './registration-wizard.service'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'
import { CredentialBindingsComponent } from './credential-bindings.component'

@Component({
  selector: 'ts-resource-registration-form', standalone: true, imports: [ReactiveFormsModule, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective, CredentialBindingsComponent], providers: [RegistrationWizardService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
          <ts-wizard-stepper [steps]="['Name', 'Options', 'Review']" [current]="wizard.flow.step()" [reached]="wizard.flow.reached()" [disabled]="service.saving()" (selected)="wizard.flow.go($event); feedback.reveal()" />
          <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
          <ts-wizard-heading [title]="titles[wizard.flow.step()]" [step]="wizard.flow.step()">{{ wizard.flow.step() === 2 ? 'Nothing is created until you confirm.' : 'Your entries stay in place when you go back.' }}</ts-wizard-heading>
          @if (wizard.error()) { <p class="notice error" role="alert">{{ wizard.error() }}</p> }
          <fieldset [disabled]="service.saving()">
          @if (wizard.flow.step() === 0) {
            <div class="field"><label for="resource-name">Name</label><input id="resource-name" formControlName="name" required autocomplete="off"></div>
            <details><summary>Advanced details</summary>
            <div class="field"><label for="resource-id">Permanent identifier</label><input id="resource-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" autocomplete="off"></div>
            </details>
          }
          @if (wizard.flow.step() === 1) {
            @if (service.kind() !== 'applications') {
              <div class="field"><label for="resource-application">Application {{ service.kind() === 'suites' ? '(optional)' : '' }}</label><select id="resource-application" formControlName="application" [required]="service.kind() === 'environments'"><option value="">Select an application</option>@for (app of service.applications(); track app.id) { <option [value]="app.id">{{ app.data.name }}</option> }</select></div>
            }
            @if (service.kind() === 'suites') { <label class="checkbox-field"><input type="checkbox" formControlName="requiresTarget">Requires a target environment</label><p class="small muted">Turn this off for pure unit test suites.</p> }
            @if (service.kind() === 'environments') {
              <div class="field"><label for="environment-stage">Stage</label><select id="environment-stage" formControlName="stage"><option value="development">Development</option><option value="beta">Beta</option><option value="production">Production</option><option value="local">Local</option></select></div>
              <details><summary>Advanced endpoint options</summary><div class="field"><label for="endpoint-name">Endpoint name</label><input id="endpoint-name" formControlName="endpoint" required></div></details>
              <div class="field"><label for="endpoint-url">Endpoint URL</label><input id="endpoint-url" type="url" formControlName="url" required placeholder="https://example.com"></div>
              <label class="checkbox-field"><input type="checkbox" formControlName="protected">Require production-safe tests</label><p class="small muted">Protection is separate from the named stage.</p>
              <ts-credential-bindings />
            }
            @if (service.kind() === 'applications') { <p>Applications group their target environments and related suites. You can add those after creating this application.</p> }
          }
          @if (wizard.flow.step() === 2) {
            <dl><dt>Name</dt><dd>{{ service.form.controls.name.value }}</dd><dt>Identifier</dt><dd class="wrap-anywhere">{{ service.form.controls.id.value }}</dd>
            @if (service.kind() !== 'applications') { <dt>Application</dt><dd>{{ wizard.applicationName() }}</dd> }
            @if (service.kind() === 'suites') { <dt>Requires target</dt><dd>{{ service.form.controls.requiresTarget.value ? 'Yes' : 'No' }}</dd> }
            @if (service.kind() === 'environments') { <dt>Target</dt><dd class="wrap-anywhere">{{ service.form.controls.endpoint.value }} → {{ service.form.controls.url.value }}</dd><dt>Stage</dt><dd>{{ service.form.controls.stage.value }}</dd><dt>Protection</dt><dd>{{ service.form.controls.protected.value ? 'Only production-safe tests, with protected execution permission' : 'Not protected' }}</dd> }
            </dl>
            @if (service.kind() === 'environments' && service.form.controls.credentials.length) { <h3>Credential bindings</h3><ul>@for (row of service.form.controls.credentials.controls; track row) { <li class="wrap-anywhere">{{ row.controls.binding.value }} → {{ row.controls.reference.value }}. The secret stays outside test code.</li> }</ul> }
          }
          </fieldset>
          <div class="row-actions">@if (wizard.flow.step() > 0) { <button type="button" class="button secondary" [disabled]="service.saving()" (click)="wizard.flow.back()">Back</button> }<button class="button primary" type="submit" [disabled]="service.saving()">{{ service.saving() ? 'Creating…' : wizard.flow.step() === 2 ? 'Create' : 'Continue' }}</button></div>
          </form>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } details, dl { margin-bottom: 1.25rem; } summary { cursor: pointer; padding: .75rem 0; } dd { margin: .25rem 0 1rem; } dt { font-weight: 600; }`]
})
export class ResourceRegistrationFormComponent {
  readonly service = inject(ConfigurationService)
  readonly wizard = inject(RegistrationWizardService)
  readonly titles = ['Give it a name', 'Configure options', 'Review before creating']
  readonly submitted = output<void>()
  constructor() { effect(() => { this.service.kind(); this.service.created(); untracked(() => this.wizard.flow.reset()) }) }
  advance(): void { if (this.wizard.flow.step() < 2) this.wizard.flow.next(); else if (this.wizard.submit()) this.submitted.emit() }
}
