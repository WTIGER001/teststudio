import { ChangeDetectionStrategy, Component, inject, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'
import { APIDraftWizardService } from './api-draft-wizard.service'

@Component({
  selector: 'ts-api-draft-wizard', standalone: true, imports: [ReactiveFormsModule, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], providers: [APIDraftWizardService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section aria-labelledby="api-setup-title"><h3 id="api-setup-title">Set up an API test</h3>
    <ts-wizard-stepper [steps]="['Name', 'Request', 'Review']" [current]="wizard.flow.step()" [reached]="wizard.flow.reached()" [disabled]="wizard.editor.busy()" (selected)="wizard.flow.go($event); feedback.reveal()" />
    <form [formGroup]="wizard.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()">
      <ts-wizard-heading [title]="titles[wizard.flow.step()]" [step]="wizard.flow.step()">Create a source first. Save, prepare, save, and run separately.</ts-wizard-heading>
      @if (wizard.error()) { <p class="notice error" role="alert">{{ wizard.error() }}</p> }
      <fieldset [disabled]="wizard.editor.busy()">
      @if (wizard.flow.step() === 0) {
        <div class="field"><label for="api-test-title">API test title</label><input id="api-test-title" formControlName="title" required maxlength="120" autocomplete="off"></div>
        <details><summary>Advanced details</summary><div class="field"><label for="api-test-id">Permanent API test identifier</label><input id="api-test-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" maxlength="128" autocomplete="off"></div></details>
      }
      @if (wizard.flow.step() === 1) {
        <p class="small muted">Choose the environment when you start a run. This request uses its registered endpoint, not an arbitrary external URL.</p>
        <div class="form-grid"><div class="field"><label for="api-method">Request method</label><select id="api-method" formControlName="method">@for (method of methods; track method) { <option [value]="method">{{ method }}</option> }</select></div><div class="field"><label for="api-status">Expected response status</label><input id="api-status" type="number" min="100" max="599" required formControlName="status"></div></div>
        <div class="field"><label for="api-path">Request path</label><input id="api-path" formControlName="path" required placeholder="/health"><p class="small muted">Use a relative path. A query string is allowed.</p></div>
		<div class="field"><label for="api-authentication">Authentication</label><select id="api-authentication" formControlName="authScheme"><option value="none">No credential</option><option value="bearer">Bearer token</option><option value="basic">Basic authentication</option><option value="header">API key header</option></select></div>
		@if (wizard.form.controls.authScheme.value !== 'none') {
		  <div class="field"><label for="api-credential-binding">Environment credential binding</label><input id="api-credential-binding" formControlName="credentialBinding"><p class="small muted">Use the binding name configured on the environment, such as api-login. The gateway supplies the secret. Do not enter a token or password.</p></div>
		  @if (wizard.form.controls.authScheme.value === 'header') { <div class="field"><label for="api-credential-header">Credential header name</label><input id="api-credential-header" formControlName="credentialHeader"></div> }
		}
        @if (wizard.bodyMethod(wizard.form.controls.method.value) || wizard.form.controls.body.value) { <div class="field"><label for="api-body">JSON request data (optional)</label><textarea id="api-body" rows="5" formControlName="body" spellcheck="false"></textarea><p class="small muted">Use test data here. The generated source keeps this data with the suite.</p></div> }
        @if (wizard.readOnlyMethod(wizard.form.controls.method.value)) { <label class="checkbox-field"><input type="checkbox" formControlName="readOnly">Requests are reviewed as read-only</label> }
        <p class="small muted">Read-only request policy and test safety are separate reviews. Writes cannot run against protected environments.</p>
        <details><summary>Advanced request policy</summary>
          <div class="field"><label for="api-endpoint">Registered endpoint name</label><input id="api-endpoint" formControlName="endpoint" required></div>
          <div class="field"><label for="api-prefix">Allowed path prefix</label><input id="api-prefix" formControlName="prefix" required><p class="small muted">Allows this path and its descendants. Later coded steps must stay within the reviewed rules.</p></div>
          <div class="form-grid"><div class="field"><label for="api-request-count">Maximum requests per attempt</label><input id="api-request-count" type="number" formControlName="maxRequests" required min="1" max="1000"></div><div class="field"><label for="api-response-size">Maximum response bytes</label><input id="api-response-size" type="number" formControlName="maxResponseBytes" required min="1" max="4194304"></div></div>
          @if (wizard.bodyMethod(wizard.form.controls.method.value)) { <div class="field"><label for="api-body-size">Maximum request body bytes</label><input id="api-body-size" type="number" formControlName="maxBodyBytes" required min="0" max="1048576"></div> }
        </details>
      }
      @if (wizard.flow.step() === 2) {
        <dl><dt>Test</dt><dd>{{ wizard.form.controls.title.value }} <span class="small muted">{{ wizard.form.controls.id.value }}</span></dd><dt>Request</dt><dd class="wrap-anywhere">{{ wizard.form.controls.method.value }} {{ wizard.form.controls.path.value }}</dd><dt>Expected status</dt><dd>{{ wizard.form.controls.status.value }}</dd><dt>Endpoint</dt><dd>{{ wizard.form.controls.endpoint.value }} in the environment selected at run time</dd><dt>Allowed path prefix</dt><dd class="wrap-anywhere">{{ wizard.form.controls.prefix.value }}</dd><dt>Target policy</dt><dd>{{ wizard.form.controls.readOnly.value ? 'Reviewed read-only requests. Protected execution still requires safe test labels and permission.' : 'Unprotected targets only.' }}</dd><dt>Safety label</dt><dd>Needs review. Inspect the generated code and catalog before save.</dd><dt>Request budget</dt><dd>{{ wizard.form.controls.maxRequests.value }} per attempt</dd><dt>Redirects</dt><dd>Disabled</dd></dl>
        @if (wizard.form.controls.body.value.trim()) { <h4>JSON request data</h4><pre>{{ wizard.form.controls.body.value }}</pre> }
		<p>Authentication: {{ wizard.form.controls.authScheme.value === 'none' ? 'No credential' : wizard.form.controls.authScheme.value + ' with environment binding ' + wizard.form.controls.credentialBinding.value }}.</p>
		@if (wizard.form.controls.authScheme.value !== 'none') { <p class="notice">The selected environment must map this binding to an approved credential reference. Its exact origin must be approved by the operator. HTTPS is required outside local environments.</p> }
        <p class="small muted">Request body limit: {{ wizard.bodyMethod(wizard.form.controls.method.value) ? wizard.form.controls.maxBodyBytes.value : 0 }} bytes. Response limit: {{ wizard.form.controls.maxResponseBytes.value }} bytes.</p>
        <p class="notice">Create source opens the editor. It does not save, save, or run the test.</p>
      }
      </fieldset>
      <div class="row-actions wrap-actions">@if (wizard.flow.step() > 0) { <button type="button" class="button secondary" [disabled]="wizard.editor.busy()" (click)="wizard.flow.back()">Back</button> }<button class="button primary" type="submit" [disabled]="wizard.editor.busy()">{{ wizard.editor.busy() ? 'Creating…' : wizard.flow.step() === 2 ? 'Create source' : 'Continue' }}</button><button class="button secondary" type="button" [disabled]="wizard.editor.busy()" (click)="cancelled.emit()">Cancel API setup</button></div>
    </form></section>
  `,
  styles: `fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } details { margin: 1rem 0; } summary { cursor: pointer; padding: .75rem 0; } dt { font-weight: 650; } dd { margin: .3rem 0 1rem; overflow-wrap: anywhere; } textarea { width: 100%; box-sizing: border-box; } pre { white-space: pre-wrap; overflow-wrap: anywhere; max-width: 100%; }`
})
export class APIDraftWizardComponent {
  readonly wizard = inject(APIDraftWizardService)
  readonly created = output<void>()
  readonly cancelled = output<void>()
  readonly titles = ['Name the API test', 'Configure its first request', 'Review the API source']
  readonly methods = ['GET', 'HEAD', 'OPTIONS', 'POST', 'PUT', 'PATCH', 'DELETE']
  async advance(): Promise<void> { if (this.wizard.flow.step() < 2) this.wizard.flow.next(); else if (await this.wizard.submit()) this.created.emit() }
}
