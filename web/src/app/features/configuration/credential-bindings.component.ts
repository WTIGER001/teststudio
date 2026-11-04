import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ConfigurationService } from './configuration.service'

@Component({
  selector: 'ts-credential-bindings', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <details><summary>Advanced credentials</summary><div [formGroup]="service.form">
      <p class="small muted">Map a test's credential binding name to an operator-approved reference. TestStudio supplies the secret through the gateway. Do not enter tokens or passwords.</p>
      @if (!service.credentialReferences().length) { <p class="notice">No API credential references are available. An installation operator must register a project credential and its exact allowed target origins.</p> }
      <div formArrayName="credentials">@for (row of service.form.controls.credentials.controls; track row; let index = $index) {
        <fieldset [formGroupName]="index"><legend>Credential binding {{ index + 1 }}</legend>
          <div class="field"><label [for]="'credential-binding-' + index">Binding name {{ index + 1 }}</label><input [id]="'credential-binding-' + index" formControlName="binding" required></div>
          <div class="field"><label [for]="'credential-reference-' + index">Approved credential reference {{ index + 1 }}</label><select [id]="'credential-reference-' + index" formControlName="reference" required><option value="">Choose a reference</option>@for (ref of service.credentialReferences(); track ref.id) { <option [value]="ref.id">{{ ref.name }}</option> }</select></div>
          <button class="button small secondary" type="button" [disabled]="service.saving()" (click)="service.removeCredential(index)">Remove binding {{ index + 1 }}</button>
        </fieldset>
      }</div>
      <button class="button secondary" type="button" [disabled]="!service.credentialReferences().length || service.form.controls.credentials.length >= 16 || service.saving()" (click)="service.addCredential()">Add credential binding</button>
    </div></details>
  `,
  styles: `details { margin: 1rem 0; } summary { padding: .75rem 0; cursor: pointer; } fieldset { border: 1px solid var(--border); padding: 1rem; margin: 1rem 0; min-width: 0; }`
})
export class CredentialBindingsComponent { readonly service = inject(ConfigurationService) }
