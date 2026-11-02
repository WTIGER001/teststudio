import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { NewRunService } from './new-run.service'

@Component({
  selector: 'ts-new-run', standalone: true, imports: [ReactiveFormsModule], providers: [NewRunService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="new-run-title">
      <h2 id="new-run-title">Start a run</h2><p class="small muted">Use the active published release and a registered environment. Definitions and settings are captured when the run starts.</p>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      <form [formGroup]="service.form" (ngSubmit)="service.create()">
        <div class="form-grid">
          <div class="field"><label for="run-suite">Suite</label><select id="run-suite" formControlName="suite" required><option value="">Choose a published suite</option>@for (suite of service.lookups.suites(); track suite.id) { @if (suite.data.activeReleaseId && !suite.data.archived) { <option [value]="suite.id">{{ suite.data.name }}</option> } }</select></div>
          <div class="field"><label for="run-environment">Target environment</label><select id="run-environment" formControlName="environment"><option value="">No target</option>@for (environment of service.lookups.environments(); track environment.id) { @if (!environment.data.disabled) { <option [value]="environment.id">{{ environment.data.name }} · {{ environment.data.stage }}{{ environment.data.protected ? ' · Protected' : '' }}</option> } }</select><small>Protected targets require permission and safe test definitions.</small></div>
          <div class="field"><label for="run-tests">Test identifiers (optional)</label><input id="run-tests" formControlName="tests" placeholder="HEALTH-001, HEALTH-002"><small>Leave empty to run all eligible tests. Dependencies are included.</small></div>
          <div class="field"><label for="run-timeout">Run time limit (seconds)</label><input id="run-timeout" type="number" formControlName="timeout" min="1" max="3600" step="1" required><small>The local worker accepts limits up to one hour.</small></div>
          <div class="field"><label for="run-retries">Retries per test</label><select id="run-retries" formControlName="retries"><option [ngValue]="0">No retries</option><option [ngValue]="1">1 retry</option><option [ngValue]="2">2 retries</option></select></div>
        </div>
        <p class="small muted">Execution location: local worker</p>
        <button class="button primary" type="submit" [disabled]="service.form.invalid || service.saving()">{{ service.saving() ? 'Starting…' : 'Start run' }}</button>
      </form>
    </section>
  `
})
export class NewRunComponent { readonly service = inject(NewRunService) }
