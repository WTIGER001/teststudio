import { afterNextRender, ChangeDetectionStrategy, Component, ElementRef, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'
import { RepositoriesService } from './repositories.service'

@Component({
  selector: 'ts-repository-editor', standalone: true, imports: [ReactiveFormsModule, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="repository-editor-title">
      <h3 id="repository-editor-title">{{ service.selected() ? 'Edit repository source' : 'Register repository source' }}</h3>
      <p class="small muted">Connect a repository, choose its suites, then review synchronization and publication before saving.</p>
      <ts-wizard-stepper [steps]="['Repository', 'Suites', 'Review']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.busy()" (selected)="service.flow.go($event); feedback.reveal()" />
      <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="service.advance(); feedback.reveal()">
        <ts-wizard-heading [step]="service.flow.step()" [title]="titles[service.flow.step()]" />
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        <fieldset [disabled]="service.busy()"><legend class="sr-only">Repository source configuration</legend>
          @if (service.flow.step() === 0) {
            <div class="form-grid">
              <div class="field"><label for="repository-name">Source name</label><input id="repository-name" formControlName="name" required maxlength="200"></div>
              <div class="field"><label for="repository-path">GitHub repository</label><input id="repository-path" formControlName="repository" required placeholder="owner/repository"><small>Enter owner/repository without a URL or credentials.</small></div>
              <div class="field"><label for="repository-credential">Repository access</label><select id="repository-credential" formControlName="credential"><option value="">Public repository</option>@for (credential of service.options().credentials; track credential.id) { <option [value]="credential.id">{{ credential.name }}</option> }</select><small>Private repositories use an approved GitHub App credential reference.</small></div>
              <div class="field"><label for="repository-reference">Reference name or commit</label><input id="repository-reference" formControlName="reference" required maxlength="256"><small>The default reference type is Branch. Choose Tag or Commit under Advanced repository options.</small></div>
            </div>
            <details><summary>Advanced repository options</summary><div class="form-grid">
              <div class="field"><label for="repository-id">Source identifier</label><input id="repository-id" formControlName="id" required><small>Generated from the name. This identifier stays permanent after saving.</small></div>
              <div class="field"><label for="repository-reference-kind">Reference type</label><select id="repository-reference-kind" formControlName="referenceKind"><option value="branch">Branch</option><option value="tag">Tag</option><option value="commit">Commit</option></select></div>
            </div></details>
          }
          @if (service.flow.step() === 1) {
            <p class="small muted">Manifest paths refer to the repository root. Each suite uses one offline Go or Playwright runtime profile.</p>
            <div formArrayName="bindings">@for (binding of service.form.controls.bindings.controls; track binding; let index = $index) {
              <div class="binding-row" [formGroupName]="index">
                <div class="field"><label [for]="'repository-suite-' + index">Suite {{ index + 1 }}</label><select [id]="'repository-suite-' + index" formControlName="suite" required><option value="">Choose a suite</option>@for (suite of service.sources.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div>
                <div class="field"><label [for]="'repository-manifest-' + index">Manifest path {{ index + 1 }}</label><input [id]="'repository-manifest-' + index" formControlName="manifest" required></div>
                <div class="field"><label [for]="'repository-runner-' + index">Runner {{ index + 1 }}</label><select [id]="'repository-runner-' + index" formControlName="runner"><option value="go">Go module</option><option value="playwright">Playwright runtime</option></select></div>
                <button type="button" class="button secondary" [disabled]="service.form.controls.bindings.length < 2" (click)="service.removeBinding(index)" [attr.aria-label]="'Remove suite binding ' + (index + 1)">Remove</button>
              </div>
            }</div>
            <button type="button" class="button secondary" [disabled]="service.form.controls.bindings.length >= 20" (click)="service.addBinding()">Add suite binding</button>
          }
          @if (service.flow.step() === 2) {
            <dl>
              <dt>Repository</dt><dd>{{ service.form.controls.name.value }} · {{ service.form.controls.repository.value }}</dd>
              <dt>Permanent source identifier</dt><dd>{{ service.form.controls.id.value }}</dd>
              <dt>Reference</dt><dd>{{ service.form.controls.referenceKind.value }} · {{ service.form.controls.reference.value }}</dd>
              <dt>Repository access</dt><dd>{{ service.form.controls.credential.value || 'Public repository' }}</dd>
              <dt>Suite bindings</dt><dd><ul>@for (binding of service.form.controls.bindings.controls; track binding) { <li>{{ binding.controls.suite.value }} · {{ binding.controls.manifest.value }} · {{ binding.controls.runner.value }}</li> }</ul></dd>
              <dt>Automatic sync</dt><dd>{{ service.form.controls.push.value ? 'Signed push enabled' : 'Signed push disabled' }} · {{ service.form.controls.intervalSeconds.value ? 'Periodic sync every ' + service.form.controls.intervalSeconds.value + ' seconds' : 'Periodic sync disabled' }}</dd>
            </dl>
            <details><summary>Advanced synchronization options</summary>
              <div class="form-grid source-policies">
                <div class="field"><label for="repository-webhook-credential">Push webhook access</label><select id="repository-webhook-credential" formControlName="webhookCredential"><option value="">Choose an approved webhook secret</option>@for (credential of service.options().webhookCredentials; track credential.id) { <option [value]="credential.id">{{ credential.name }}</option> }</select><small>Webhook signing uses a separate approved secret reference.</small></div>
                <div class="field"><label for="repository-sync-interval">Periodic sync interval (seconds)</label><input id="repository-sync-interval" type="number" formControlName="intervalSeconds" min="0" max="86400" step="1"><small>Use 0 to disable periodic sync, or 60 through 86400 seconds.</small></div>
              </div>
              <label class="checkbox-field"><input type="checkbox" formControlName="push">Enable signed push synchronization</label>
              <p class="small muted">Push sync watches a branch or tag. Complete one manual sync to verify repository identity before enabling its webhook.</p>
            </details>
            <div class="source-policies">
              <label class="checkbox-field"><input type="checkbox" formControlName="enabled">Enable repository synchronization</label>
              <label class="checkbox-field"><input type="checkbox" formControlName="autoActivate">Publish and activate successful preparations automatically</label>
              <p class="small muted">Automatic publication changes the active release for future runs and requires current publishing authority. With it disabled, review and publish each preparation yourself.</p>
              <p class="small muted">Saving does not start a manual sync, build, or test. Enabled push and periodic policies can queue future preparations. Synchronization never starts tests.</p>
            </div>
          }
          <div class="row-actions wrap-actions">
            @if (service.flow.step() > 0) { <button type="button" class="button secondary" (click)="service.flow.back()">Back</button> }
            <button type="submit" class="button primary">{{ service.flow.step() === 2 ? 'Save repository source' : 'Continue' }}</button>
            <button type="button" class="button secondary" (click)="service.editor.set(false)">Close editor</button>
          </div>
        </fieldset>
      </form>
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } .binding-row { display: grid; grid-template-columns: 1fr 1fr 1fr auto; align-items: end; gap: 1rem; margin: 1rem 0; } .source-policies { margin: 1.5rem 0; } details { margin: 1rem 0; } summary { padding: .75rem 0; cursor: pointer; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; overflow-wrap: anywhere; } ul { padding-left: 1.25rem; } @media(max-width: 640px) { .binding-row { grid-template-columns: 1fr; } }`]
})
export class RepositoryEditorComponent {
  readonly service = inject(RepositoriesService)
  readonly titles = ['Choose a repository', 'Choose suites and manifests', 'Review synchronization and publication']
  constructor() {
    const host: ElementRef<HTMLElement> = inject(ElementRef)
    afterNextRender(() => host.nativeElement.querySelector<HTMLElement>('ts-wizard-heading h3')?.focus())
  }
}
