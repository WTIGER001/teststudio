import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ProbeURLFieldComponent } from '../../shared/probes/probe-url-field.component'
import { ProbeTargetService } from './probe-target.service'

@Component({
  selector: 'ts-probe-target', standalone: true, imports: [ReactiveFormsModule, ProbeURLFieldComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <fieldset><legend>Request target</legend><div class="row-actions wrap-actions"><label class="checkbox-field"><input type="radio" name="probe-target-mode" value="url" [formControl]="targets.mode">Enter a website URL</label><label class="checkbox-field"><input type="radio" name="probe-target-mode" value="registered" [formControl]="targets.mode">Use a registered environment</label></div></fieldset>
    @if (targets.direct()) {
      <ts-probe-url-field [control]="targets.address" />
      @if (targets.matches().length > 1) { <div class="field"><label for="probe-url-choice">Matching registered target</label><select id="probe-url-choice" [formControl]="targets.choice"><option value="">Choose the target to use</option>@for (match of targets.matches(); track match.key) { <option [value]="match.key">{{ match.environment.data.name }} · {{ match.endpoint }} · {{ match.environment.data.stage }}{{ match.environment.data.protected ? ' · Protected' : '' }}{{ match.environment.data.disabled ? ' · Disabled' : '' }}</option> }</select></div> }
      @if (targets.target(); as target) { <p class="notice">Uses registered target {{ target.data.name }} · {{ target.data.stage }} · {{ target.data.protected ? 'Protected' : 'Not protected' }}{{ target.data.disabled ? ' · Disabled' : '' }}. Its settings stay unchanged.</p> }
      @if (targets.needsTarget() && targets.loaded()) {
        <p class="small muted">This address will be saved as a target under {{ targets.applicationName() || 'the application you choose' }}. Review will show the configuration before saving.</p>
        <details><summary>Advanced target options</summary>
          @if (!targets.needsApplication() && targets.applications().length > 1) { <div class="field"><label for="probe-application">Application for this target</label><select id="probe-application" [formControl]="targets.application"><option value="">Choose an application</option>@for (app of targets.applications(); track app.id) { <option [value]="app.id">{{ app.name }}</option> }</select></div> }
          <div class="field"><label for="probe-stage">Target stage</label><select id="probe-stage" [formControl]="targets.stage"><option value="development">Development</option><option value="beta">Beta</option><option value="production">Production</option><option value="local">Local</option></select><small>Choose Local explicitly for a plain HTTP or private-network target.</small></div>
          <label class="checkbox-field"><input type="checkbox" [formControl]="targets.protected">Require production-safe tests for this target</label>
        </details>
      }
    } @else {
      <div class="field"><label for="probe-target">Target environment</label><select id="probe-target" [formControl]="targets.environment"><option value="">Choose later — save a draft</option>@for (environment of targets.environments(); track environment.id) { <option [value]="environment.id">{{ environment.data.name }} · {{ environment.data.stage }}{{ environment.data.protected ? ' · Protected' : '' }}</option> }</select><small>The test keeps its endpoint alias for use in other environments.</small></div>
      @if (targets.loaded() && !targets.environments().length) { <p class="notice">No readable enabled target environments. Enter a website URL above, or save a draft with an endpoint name.</p> }
    }
    @if (targets.loading()) { <p role="status">Loading target options…</p> }
    @if (targets.error()) { <p class="notice error" role="alert">{{ targets.error() }}</p><button type="button" class="button secondary" [disabled]="targets.loading()" (click)="targets.load()">Retry target options</button> }
  `,
  styles: [`fieldset { padding: 0; border: 0; margin: 0 0 1rem; } legend { font-weight: 600; margin-bottom: .5rem; } summary { cursor: pointer; padding: .75rem 0; } details { margin: 1rem 0; }`]
})
export class ProbeTargetComponent { readonly targets = inject(ProbeTargetService) }
