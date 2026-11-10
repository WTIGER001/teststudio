import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { ApplicationConfigurationService } from './application-configuration.service'

@Component({
  standalone: true, imports: [ReactiveFormsModule, RouterLink], providers: [ApplicationConfigurationService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p><a [routerLink]="['/projects', service.workspace.projectId(), 'applications', service.workspace.applicationId()]">← Application</a></p>
    <div class="page-heading"><div><span class="eyebrow">{{ service.data()?.application?.data?.name || 'Application' }}</span><h1>Environment configuration</h1><p class="muted">Define what tests need, then configure each deployment of this application.</p></div><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.reload()">Reload configuration</button></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.busy()) { <p role="status">Loading or saving configuration…</p> }
    @if (service.data(); as data) {
      @if (!data.application.data.defaultSuiteId && service.workspace.can('applications.manage') && service.workspace.can('suites.manage')) { <section class="panel"><h2>Default suite</h2><p>This application predates automatic default suites. Create an empty Default suite while preserving its existing suites and history.</p><button type="button" class="button secondary" [disabled]="service.busy() || service.dirty()" (click)="service.createDefaultSuite()">Create default suite</button></section> }
      <section class="panel" aria-labelledby="readiness-title"><div class="section-top"><h2 id="readiness-title">Configuration readiness</h2>@if (service.workspace.can('applications.manage')) { <button type="button" class="button secondary" [disabled]="service.busy() || service.dirty()" (click)="service.editContract()">Edit requirements</button> }</div>
        <p>Required for every run applies to each selected environment. Other requirements apply when selected tests or their dependencies need them.</p>
        <p class="small muted">Configured secrets have an approved reference. Availability is checked before execution. This does not establish that the application accepts the credential.</p>
        <div class="table-scroll" tabindex="0" role="region" aria-label="Environment readiness matrix"><table><caption>Required configuration by environment</caption><thead><tr><th scope="col">Configuration</th>@for (row of data.environments; track row.id) { <th scope="col">{{ row.data.name }}<span class="table-subtitle">{{ row.readiness.ready ? 'Configured' : 'Needs configuration' }}</span>@if (service.workspace.can('environments.manage')) { <button type="button" class="button small secondary" [disabled]="service.busy() || service.dirty()" (click)="service.editEnvironment(row)" [attr.aria-label]="'Configure ' + row.data.name">Configure</button> }</th> }</tr></thead><tbody>
          @for (requirement of data.requirements; track requirement.kind + '/' + requirement.name) { <tr><th scope="row">{{ requirement.name }}<span class="table-subtitle">{{ requirement.kind }} · {{ requirement.required ? 'Every run' : 'When used by tests' }}</span></th>@for (row of data.environments; track row.id) { <td [title]="service.description(row, requirement)"><span class="badge" [class.warning]="!['ready', 'configured'].includes(service.status(row, requirement))">{{ service.status(row, requirement) }}</span></td> }</tr> }
        </tbody></table></div>
        @if (!data.requirements.length) { <p>No requirements yet. Add application requirements or save tests that declare the configuration they use.</p> }
        @if (!data.environments.length) { <p>No readable environments. Add an environment from the application page when you have permission.</p> }
      </section>
      @if (service.mode() === 'contract') {
        <section class="panel" aria-labelledby="contract-title"><h2 id="contract-title">{{ service.review() ? 'Review requirements' : 'Application requirements' }}</h2>
          <form [formGroup]="service.contractForm" (ngSubmit)="service.beginReview()"><fieldset [disabled]="service.busy() || service.conflict() || service.review()"><div formArrayName="requirements">
            @for (row of service.requirements.controls; track row; let index = $index) { <fieldset [formGroupName]="index" class="requirement"><legend>Requirement {{ index + 1 }}</legend><div class="form-grid">
              <div class="field"><label [for]="'requirement-name-' + index">Configuration name</label><input [id]="'requirement-name-' + index" formControlName="name" required maxlength="128"></div>
              <div class="field"><label [for]="'requirement-kind-' + index">Kind</label><select [id]="'requirement-kind-' + index" formControlName="kind"><option value="parameter">Parameter</option><option value="secret">Secret reference</option><option value="endpoint">Base URL</option></select></div>
              @if (row.controls.kind.value === 'parameter') { <div class="field"><label [for]="'requirement-type-' + index">Value type</label><select [id]="'requirement-type-' + index" formControlName="valueType"><option value="text">Text</option><option value="url">HTTPS URL</option><option value="integer">Whole number</option><option value="boolean">True or false</option></select></div><div class="field"><label [for]="'requirement-default-' + index">Default value (optional)</label><input [id]="'requirement-default-' + index" formControlName="defaultValue" maxlength="4096"><small>Used when an environment has no override. Keep secrets in secret references.</small></div> }
            </div><div class="field"><label [for]="'requirement-description-' + index">Description</label><input [id]="'requirement-description-' + index" formControlName="description" maxlength="4096"></div><label class="checkbox-field"><input type="checkbox" formControlName="required">Required for every run against an environment</label><button type="button" class="button subtle" (click)="service.removeRequirement(index)">Remove requirement {{ index + 1 }}</button></fieldset> }
          </div><button type="button" class="button secondary" [disabled]="service.requirements.length >= 64" (click)="service.addRequirement()">Add requirement</button></fieldset>@if (!service.review()) { <button class="button primary" type="submit" [disabled]="service.busy() || service.conflict()">Review requirements</button> }</form>
          @if (service.review()) { <p>Save {{ service.requirements.length }} requirements for this application. New requirements can make existing environments incomplete. Tests and history stay saved.</p> }
        </section>
      }
      @if (service.mode() === 'environment' && service.selected(); as selected) {
        <section class="panel" aria-labelledby="environment-config-title"><h2 id="environment-config-title">{{ service.review() ? 'Review configuration: ' : 'Configure: ' }}{{ selected.data.name }}</h2><p>Base URLs choose where tests run. Parameters hold nonsecret values. Secrets select approved references.</p>
          <form [formGroup]="service.environmentForm" (ngSubmit)="service.beginReview()"><fieldset [disabled]="service.busy() || service.conflict() || service.review()"><div formArrayName="values">
            @for (row of service.values.controls; track row; let index = $index) { <div [formGroupName]="index" class="field"><label [for]="'environment-value-' + index">{{ row.controls.name.value }} · {{ row.controls.kind.value === 'endpoint' ? 'Base URL' : row.controls.kind.value === 'secret' ? 'Secret reference' : 'Parameter' }}</label>
              @if (row.controls.kind.value === 'secret') { <select [id]="'environment-value-' + index" formControlName="value"><option value="">Not configured</option>@for (reference of service.references(); track reference.id) { <option [value]="reference.id">{{ reference.name }}</option> }</select> } @else { <input [id]="'environment-value-' + index" [type]="row.controls.kind.value === 'endpoint' ? 'url' : 'text'" formControlName="value" maxlength="4096"> }
            </div> }
          </div></fieldset>@if (!service.references().length) { <p class="small muted">An operator must register approved credential references before you can bind secrets here.</p> }@if (!service.review()) { <button type="submit" class="button primary" [disabled]="service.busy() || service.conflict()">Review environment configuration</button> }</form>
          @if (service.review()) { <p>Save these values for {{ selected.data.name }}. Blank entries remove the environment value; parameter defaults can still apply. Incomplete configuration can be saved. Execution checks only the required configuration.</p> }
        </section>
      }
      @if (service.mode()) { <div class="row-actions wrap-actions">@if (service.review()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.review.set(false)">Back to editing</button><button type="button" class="button primary" [disabled]="service.busy() || service.conflict()" (click)="service.save()">Save configuration</button> }<button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.cancel()">Cancel configuration changes</button></div> }
    }
  `,
  styles: [`:host { display: block; } table { min-width: 28rem; } th, td { overflow-wrap: anywhere; max-width: 20rem; } fieldset { border: 0; padding: 0; margin: 1rem 0; min-width: 0; } fieldset.requirement { border: 1px solid var(--border); padding: 1rem; border-radius: .5rem; } .section-top { flex-wrap: wrap; } legend { font-weight: 600; } form > button { margin: .75rem 0; }`]
})
export class ApplicationConfigurationPage {
  readonly service = inject(ApplicationConfigurationService)
  canLeave(): boolean { return this.service.canLeave() }
}
