import { ChangeDetectionStrategy, Component, inject, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ConfigurationService } from './configuration.service'

@Component({
  selector: 'ts-resource-registration-form', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
          <form [formGroup]="service.form" (ngSubmit)="submitted.emit()">
            <div class="field"><label for="resource-name">Name</label><input id="resource-name" formControlName="name" required autocomplete="off"></div>
            <div class="field"><label for="resource-id">Permanent identifier</label><input id="resource-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" autocomplete="off"></div>
            @if (service.kind() !== 'applications') {
              <div class="field"><label for="resource-application">Application {{ service.kind() === 'suites' ? '(optional)' : '' }}</label><select id="resource-application" formControlName="application" [required]="service.kind() === 'environments'"><option value="">Select an application</option>@for (app of service.applications(); track app.id) { <option [value]="app.id">{{ app.data.name }}</option> }</select></div>
            }
            @if (service.kind() === 'suites') { <label class="checkbox-field"><input type="checkbox" formControlName="requiresTarget">Requires a target environment</label><p class="small muted">Turn this off for pure unit test suites.</p> }
            @if (service.kind() === 'environments') {
              <div class="field"><label for="environment-stage">Stage</label><select id="environment-stage" formControlName="stage"><option value="development">Development</option><option value="beta">Beta</option><option value="production">Production</option><option value="local">Local</option></select></div>
              <div class="field"><label for="endpoint-name">Endpoint name</label><input id="endpoint-name" formControlName="endpoint" required></div>
              <div class="field"><label for="endpoint-url">Endpoint URL</label><input id="endpoint-url" type="url" formControlName="url" required placeholder="https://example.com"></div>
              <label class="checkbox-field"><input type="checkbox" formControlName="protected">Require production-safe tests</label><p class="small muted">Protection is separate from the named stage.</p>
            }
            <button class="button primary" type="submit" [disabled]="service.form.invalid || service.saving()">{{ service.saving() ? 'Creating…' : 'Create' }}</button>
          </form>
  `
})
export class ResourceRegistrationFormComponent {
  readonly service = inject(ConfigurationService)
  readonly submitted = output<void>()
}
