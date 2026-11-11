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
      <p class="small muted">Connect your repository, find its tests, and choose where to import them. You can use existing Playwright tests without adding metadata files.</p>
      <ts-wizard-stepper [steps]="['Repository', 'Suites', 'Review']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.busy()" (selected)="service.flow.go($event); feedback.reveal()" />
      <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
        <ts-wizard-heading [step]="service.flow.step()" [title]="titles[service.flow.step()]" />
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        <fieldset [disabled]="service.busy()"><legend class="sr-only">Repository source configuration</legend>
          @if (service.flow.step() === 0) {
            @if (!service.options().providers.length) { <p class="notice warning">No Git providers are available. Ask your operator to enable GitHub or approve your GitLab server before connecting a repository.</p> }
            <div class="form-grid">
              <div class="field"><label for="repository-name">Source name</label><input id="repository-name" formControlName="name" required maxlength="200"></div>
              <div class="field"><label for="repository-provider">Git provider</label><select id="repository-provider" formControlName="provider">@for (provider of service.options().providers; track provider) { <option [value]="provider">{{ provider }}</option> }</select><small>Self-managed GitLab servers are approved by your operator.</small></div><div class="field"><label for="repository-mode">Metadata ownership</label><select id="repository-mode" formControlName="mode"><option value="read-only">Read-only — store metadata in TestStudio</option><option value="read-write">Read/write — commit metadata to Git</option></select><small>Read/write changes use a review branch and pull or merge request. Environment secrets never enter Git.</small></div><div class="field"><label for="repository-path">Repository path</label><input id="repository-path" formControlName="repository" required placeholder="owner/repository"><small>Enter owner/repository or group/subgroup/repository without credentials.</small></div>
              <div class="field"><label for="repository-credential">Repository access</label><select id="repository-credential" formControlName="credential"><option value="">Public repository</option>@for (credential of service.options().credentials; track credential.id) { <option [value]="credential.id">{{ credential.name }}</option> }</select><small>Choose an approved GitHub API token, GitHub App, or GitLab token reference. Read/write mode requires approved write access.</small></div>
              <div class="field"><label for="repository-reference">Reference name or commit</label><input id="repository-reference" formControlName="reference" required maxlength="256"><small>The default reference type is Branch. Choose Tag or Commit under Advanced repository options.</small></div>
            </div>
            <details><summary>Advanced repository options</summary><div class="form-grid">
              <div class="field"><label for="repository-id">Source identifier</label><input id="repository-id" formControlName="id" required><small>Generated from the name. This identifier stays permanent after saving.</small></div>
              <div class="field"><label for="repository-reference-kind">Reference type</label><select id="repository-reference-kind" formControlName="referenceKind"><option value="branch">Branch</option><option value="tag">Tag</option><option value="commit">Commit</option></select></div>
            </div></details>
          }
          @if (service.flow.step() === 1) {
            <p class="small muted">Choose the application-owned suite that will contain these tests. Use its Default suite when you do not need a separate collection. Imported tests start disabled and need review.</p>
            @if (service.inspection(); as inspected) {
              <p class="notice success" role="status">Repository access checked. Found {{ inspected.candidates.length }} Playwright configuration(s) and {{ inspected.manifests.length }} existing suite declaration(s). Nothing has been saved or run.</p>
              @if (!inspected.candidates.length && !inspected.manifests.length) { <p class="notice warning">No supported test configuration was found. Enter its location under Advanced test configuration, or go Back to check the branch.</p> }
            } @else { <p class="notice">Repository access has not been checked. <button type="button" class="button secondary" (click)="service.findConfigurations()">Find test configurations</button></p> }
            @if (!service.sources.suites().length) { <p class="notice warning">Create an application and its Default suite in Project settings before importing tests.</p> }
            <div formArrayName="bindings">@for (binding of service.form.controls.bindings.controls; track binding; let index = $index) {
              <div class="binding-row" [formGroupName]="index">
                <div class="field"><label [for]="'repository-suite-' + index">Suite {{ index + 1 }}</label><select [id]="'repository-suite-' + index" formControlName="suite" required (change)="service.chooseSuite(index)"><option value="">Choose a suite</option>@for (suite of service.sources.suites(); track suite.id) { <option [value]="suite.id">{{ service.suiteLabel(suite.id) }}</option> }</select></div>
                @if (binding.controls.runner.value === 'playwright') { <label class="checkbox-field"><input type="checkbox" formControlName="discover">Discover tests without metadata</label> }
 @if (binding.controls.discover.value && service.inspection()?.candidates?.length) {
 <div class="field"><label [for]="'repository-detected-' + index">Detected configuration {{ index + 1 }}</label><select [id]="'repository-detected-' + index" (change)="service.useConfiguration(index, +$any($event.target).value)"><option value="-1" [selected]="service.configurationIndex(index) === -1">Choose a configuration</option>@for (candidate of service.inspection()!.candidates; track candidate.moduleRoot + candidate.config; let candidateIndex = $index) { <option [value]="candidateIndex" [selected]="service.configurationIndex(index) === candidateIndex">{{ candidate.moduleRoot === '.' ? candidate.config : candidate.moduleRoot + '/' + candidate.config }}</option> }</select></div>
 }
 @if (!binding.controls.discover.value && service.inspection()?.manifests?.length) { <div class="field"><label [for]="'repository-existing-' + index">Existing manifest {{ index + 1 }}</label><select [id]="'repository-existing-' + index" (change)="service.useManifest(index, $any($event.target).value)"><option value="">Choose a declaration</option>@for (manifest of service.inspection()!.manifests; track manifest) { <option [value]="manifest" [selected]="binding.controls.manifest.value === manifest">{{ manifest }}</option> }</select></div> }
 <details class="binding-advanced"><summary>Advanced test configuration {{ index + 1 }}</summary><div class="form-grid">
                <div class="field"><label [for]="'repository-runner-' + index">Runner {{ index + 1 }}</label><select [id]="'repository-runner-' + index" formControlName="runner"><option value="go">Go module</option><option value="playwright">Playwright runtime</option></select></div>
                <div class="field"><label [for]="'repository-manifest-' + index">Manifest path {{ index + 1 }}</label><input [id]="'repository-manifest-' + index" formControlName="manifest" required></div>
 @if (binding.controls.runner.value === 'playwright' && binding.controls.discover.value) {
 <div class="field"><label [for]="'repository-root-' + index">Source folder {{ index + 1 }}</label><input [id]="'repository-root-' + index" formControlName="moduleRoot"><small>Use . for the repository root.</small></div>
 <div class="field"><label [for]="'repository-config-' + index">Playwright configuration {{ index + 1 }}</label><input [id]="'repository-config-' + index" formControlName="config"><small>Relative to the source folder. Use a detected configuration above when available.</small></div>
 <div class="field"><label [for]="'repository-project-' + index">Playwright project {{ index + 1 }}</label><input [id]="'repository-project-' + index" formControlName="project"><small>Leave blank to discover all projects.</small></div>
 }
 </div></details>
 <div class="field"><label [for]="'repository-endpoint-' + index">Environment endpoint {{ index + 1 }}</label><input [id]="'repository-endpoint-' + index" formControlName="endpoint"><small>Use an endpoint name, such as website or api. Its URL comes from each environment. Leave blank for targetless tests.</small></div>
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
              <dt>Metadata ownership</dt><dd>{{ service.form.controls.mode.value }} · {{ service.form.controls.provider.value }}</dd><dt>Repository access</dt><dd>{{ service.form.controls.credential.value || 'Public repository' }}</dd>
              <dt>Suite bindings</dt><dd><ul>@for (binding of service.form.controls.bindings.controls; track binding) { <li>{{ service.suiteLabel(binding.controls.suite.value) }}<span class="table-subtitle">{{ binding.controls.discover.value ? binding.controls.moduleRoot.value + '/' + binding.controls.config.value : binding.controls.manifest.value }} · {{ binding.controls.endpoint.value ? 'Environment endpoint: ' + binding.controls.endpoint.value : 'No environment endpoint' }}</span></li> }</ul></dd>
              <dt>Test review</dt><dd>Imported tests stay disabled until their metadata and safety are reviewed. Saving never starts a test run.</dd>
              <dt>Connection check</dt><dd>{{ service.inspection() ? 'Access checked and configurations detected' : 'Configured manually; access will be checked during preparation' }}</dd>
              <dt>Automatic sync</dt><dd>{{ service.form.controls.push.value ? 'Signed push enabled' : 'Signed push disabled' }} · {{ service.form.controls.intervalSeconds.value ? 'Periodic sync every ' + service.form.controls.intervalSeconds.value + ' seconds' : 'Periodic sync disabled' }}</dd>
            </dl>
            @if (service.selected() && (service.selected()?.data?.configuration?.accessMode || 'read-only') !== service.form.controls.mode.value) {
 <p class="notice warning">Changing ownership makes {{ service.form.controls.mode.value === 'read-write' ? 'committed Git metadata' : 'the database metadata copy' }} authoritative for future synchronization. Earlier versions stay retained. Review the current metadata before switching.</p><label class="checkbox-field"><input type="checkbox" formControlName="confirmModeChange">I reviewed the metadata and confirm this ownership change</label>
 }
 <details><summary>Advanced synchronization options</summary>
              <div class="form-grid source-policies">
                <div class="field"><label for="repository-webhook-credential">Push webhook access</label><select id="repository-webhook-credential" formControlName="webhookCredential"><option value="">Choose an approved webhook secret</option>@for (credential of service.options().webhookCredentials; track credential.id) { <option [value]="credential.id">{{ credential.name }}</option> }</select><small>Webhook signing uses a separate approved secret reference.</small></div>
                <div class="field"><label for="repository-sync-interval">Periodic sync interval (seconds)</label><input id="repository-sync-interval" type="number" formControlName="intervalSeconds" min="0" max="86400" step="1"><small>Use 0 to disable periodic sync, or 60 through 86400 seconds.</small></div>
              </div>
              <label class="checkbox-field"><input type="checkbox" formControlName="push">Enable signed push synchronization</label>
              <p class="small muted">Push sync watches a branch or tag. Complete one manual sync to verify repository identity before enabling its webhook.</p>
            </details>
            @if (service.form.controls.enabled.value) { <label class="checkbox-field"><input type="checkbox" [formControl]="service.prepareAfterSave">Prepare tests after saving</label><p class="small muted">Download pinned source and discover its test list in an isolated worker. This does not run tests. Leave unchecked to save the connection only.</p> }
            <div class="source-policies">
              <label class="checkbox-field"><input type="checkbox" formControlName="enabled">Enable repository synchronization</label>
              <p class="small muted">Successful setup makes the saved version available for future runs. Current permissions, configuration, and safety rules still apply.</p>
              <p class="small muted">Saving the connection only stores settings. Prepare tests after saving queues source setup. Enabled push and periodic policies can also queue future preparations. Test runs start separately.</p>
            </div>
          }
          <div class="row-actions wrap-actions">
            @if (service.flow.step() > 0) { <button type="button" class="button secondary" (click)="service.flow.back()">Back</button> }
            <button type="submit" class="button primary">{{ service.busy() ? 'Working…' : service.flow.step() === 2 ? service.saveLabel : service.flow.step() === 0 && !service.selected() ? 'Check repository and continue' : 'Continue' }}</button>
            @if (service.flow.step() === 0 && !service.selected()) { <button type="button" class="button secondary" (click)="service.continueManually(); feedback.reveal()">Configure manually</button> }
            <button type="button" class="button secondary" (click)="service.closeEditor()">Close editor</button>
          </div>
        </fieldset>
      </form>
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } .binding-row { display: grid; grid-template-columns: 1fr 1fr; align-items: end; gap: 1rem; margin: 1rem 0; } .binding-advanced { grid-column: 1 / -1; } .source-policies { margin: 1.5rem 0; } details { margin: 1rem 0; } summary { padding: .75rem 0; cursor: pointer; } dt { font-weight: 600; } dd { margin: .25rem 0 1rem; overflow-wrap: anywhere; } ul { padding-left: 1.25rem; } @media(max-width: 640px) { .binding-row { grid-template-columns: 1fr; } }`]
})
export class RepositoryEditorComponent {
  readonly service = inject(RepositoriesService)
  readonly titles = ['Choose a repository', 'Choose tests and destination suites', 'Review synchronization and save']
  async advance(): Promise<void> { await this.service.advance() }
  constructor() {
    const host: ElementRef<HTMLElement> = inject(ElementRef)
    afterNextRender(() => host.nativeElement.querySelector<HTMLElement>('ts-wizard-heading h3')?.focus())
  }
}
