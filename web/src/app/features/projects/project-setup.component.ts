import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { ProjectsService } from './projects.service'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'

@Component({
  selector: 'ts-project-setup', standalone: true, imports: [ReactiveFormsModule, RouterLink, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <ts-wizard-stepper [steps]="['Project', 'Application', 'Target', 'Review']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.saving() || service.locked()" (selected)="service.flow.go($event); feedback.reveal()" />
    <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
      <ts-wizard-heading [title]="titles[service.flow.step()]" [step]="service.flow.step()" />
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.completed().length) { <p role="status">Created: {{ service.completed().join(', ') }}</p><p><a [routerLink]="['/projects', service.form.controls.id.value, 'applications']">Open the created project</a></p> }
      <fieldset [disabled]="service.saving() || service.locked()">
      @switch (service.flow.step()) {
        @case (0) {
          <div class="field"><label for="project-name">Project name</label><input id="project-name" formControlName="name" required placeholder="Customer portal" autocomplete="off"></div>
          <details><summary>Advanced details</summary><div class="field"><label for="project-id">Permanent identifier</label><input id="project-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}"><small>Generated from the project name. You can replace it before creation.</small></div><div class="field"><label for="project-description">Description (optional)</label><textarea id="project-description" formControlName="description" rows="2"></textarea></div></details>
        }
        @case (1) {
          <p class="small muted">An application groups target environments and related test suites.</p>
          <label class="checkbox-field"><input type="checkbox" formControlName="addApplication">Set up my first application</label>
          @if (service.form.controls.addApplication.value) { <div class="field"><label for="setup-application">Application name</label><input id="setup-application" formControlName="applicationName" required></div> }
          <label class="checkbox-field"><input type="checkbox" formControlName="addSuite">Create an empty Availability suite for my first tests</label><p class="small muted">You can add applications and suites later. This creates no tests.</p>
        }
        @case (2) {
          @if (service.form.controls.addApplication.value) {
            <label class="checkbox-field"><input type="checkbox" formControlName="addTarget">Add a target environment now</label>
            @if (service.form.controls.addTarget.value) {
              <div class="field"><label for="setup-target-name">Environment name</label><input id="setup-target-name" formControlName="targetName" required></div>
              <div class="field"><label for="setup-url">Base URL</label><input id="setup-url" formControlName="url" type="url" required placeholder="https://app.example.com"></div>
              <div class="field"><label for="setup-stage">Stage</label><select id="setup-stage" formControlName="stage"><option value="development">Development</option><option value="beta">Beta</option><option value="production">Production</option><option value="local">Local</option></select><small>Choose Local for a local HTTP or private-network target.</small></div>
              <label class="checkbox-field"><input type="checkbox" formControlName="protected">Require production-safe tests</label><p class="small muted">Protection is independent of stage. Protected execution also requires permission.</p>
            }
          } @else { <p>Add an application first to register a target environment. You can do this later from Applications.</p> }
          <p class="small muted">Target registration stores the address. It does not contact the application or run a test.</p>
        }
        @case (3) {
          <p class="small muted">Confirm the resources to create. Nothing runs during setup.</p>
          <dl><dt>Project</dt><dd>{{ service.form.controls.name.value }}<br><code>{{ service.form.controls.id.value }}</code></dd>
          <dt>Application</dt><dd>{{ service.form.controls.addApplication.value ? service.form.controls.applicationName.value : 'Add later' }}</dd><dt>Suite</dt><dd>{{ service.form.controls.addSuite.value ? 'Availability — initially empty' : 'Add later' }}</dd><dt>Target environment</dt><dd class="wrap-anywhere">{{ service.form.controls.addTarget.value ? service.form.controls.targetName.value + ' · ' + service.form.controls.stage.value + ' · ' + service.form.controls.url.value : 'Add later' }}</dd>
          @if (service.form.controls.addTarget.value) { <dt>Protection</dt><dd>{{ service.form.controls.protected.value ? 'Only production-safe tests, with protected execution permission' : 'Not protected' }}</dd> }
          </dl>
          <p class="notice">Creating a project gives an ordinary creator project administration, test editing, publishing, and run access in this project. Existing installation administrators keep their current authority.</p>
          <p class="small muted">Setup uses your current permissions. If a step is denied, successful resources remain available and you can retry the remaining steps.</p>
        }
      }
      </fieldset>
      <div class="row-actions">@if (service.flow.step() > 0 && !service.locked()) { <button type="button" class="button secondary" [disabled]="service.saving()" (click)="service.flow.back()">Back</button> }<button class="button primary" type="submit" [disabled]="service.saving()">{{ service.saving() ? 'Creating…' : service.flow.step() === 3 ? service.locked() ? 'Retry remaining setup' : 'Create project' : 'Continue' }}</button></div>
    </form>
  `,
  styles: [`fieldset { padding: 0; border: 0; margin: 0; min-width: 0; } summary { padding: .75rem 0; cursor: pointer; } details, dl { margin-bottom: 1.25rem; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; } code { overflow-wrap: anywhere; }`]
})
export class ProjectSetupComponent {
  readonly service = inject(ProjectsService)
  readonly titles = ['Name your project', 'Set up your first application', 'Choose where tests will run', 'Review your project setup']
  advance(): void { if (this.service.flow.step() < 3) this.service.flow.next(); else void this.service.create() }
}
