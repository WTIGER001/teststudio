import { afterRenderEffect, ChangeDetectionStrategy, Component, ElementRef, inject, viewChild } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { SuiteManagerService } from './suite-manager.service'

@Component({
  selector: 'ts-suite-manager', standalone: true, imports: [ReactiveFormsModule, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (manager.opened()) {
      <section class="panel" aria-labelledby="suite-manager-title"><div class="section-top"><h2 #heading tabindex="-1" id="suite-manager-title">Suite settings</h2><div class="row-actions"><button type="button" class="button secondary" [disabled]="manager.busy()" (click)="manager.reload()">Reload suite settings</button><button type="button" class="button secondary" [disabled]="manager.busy()" (click)="manager.close()">Close suite settings</button></div></div>
        @if (manager.error()) { <p class="notice error" role="alert">{{ manager.error() }}</p> }@if (manager.message()) { <p class="notice success" role="status">{{ manager.message() }}</p> }
        @if (manager.state(); as state) {
          <dl class="detail-grid"><dt>Permanent identifier</dt><dd><code>{{ state.suite.id }}</code></dd><dt>Lifecycle</dt><dd>{{ state.suite.archived ? 'Archived' : 'Active' }}</dd><dt>Current version</dt><dd><code>{{ state.suite.activeReleaseId || 'No tests ready yet' }}</code></dd></dl>
          @if (!state.canManage) { <p class="small muted">You can inspect this suite. Current access does not allow changes.</p> }
          @if (state.suite.archived) { <p class="notice">This suite is archived. Restore it before editing settings or dispatching new tests.</p> }
          <form [formGroup]="manager.form" (ngSubmit)="manager.save()"><fieldset [disabled]="manager.busy() || manager.conflict() || !state.canManage || state.suite.archived"><div class="form-grid"><div class="field"><label for="suite-settings-name">Suite name</label><input id="suite-settings-name" formControlName="name" maxlength="200" required></div><div class="field"><label for="suite-settings-applications">Suite applications</label><select id="suite-settings-applications" multiple formControlName="applications">@for (application of manager.configuration.applications(); track application.id) { <option [value]="application.id">{{ application.data.name }}</option> }</select><small>Select each application this suite can target. Leave empty for a targetless suite.</small></div></div><div class="field"><label for="suite-settings-description">Suite description</label><textarea id="suite-settings-description" formControlName="description" maxlength="4096" rows="3"></textarea></div><label class="checkbox-field"><input type="checkbox" formControlName="requiresTarget">Suite requires a target environment</label><p class="small muted">Use a target environment for deployed services. Targetless suites use their isolated worker context.</p>@if (state.canManage && !state.suite.archived) { <button type="submit" class="button primary" [disabled]="manager.form.invalid || !manager.form.dirty">Save suite settings</button> }</fieldset></form>
          <h3>Affected schedules</h3><p class="small muted">Only schedules you can read appear here. Archiving blocks new suite runs and new attempts. Existing runs, versions, evidence, and schedule pins remain retained.</p>
          @if (state.schedules.length) { <div class="table-scroll"><table><caption class="sr-only">Schedules that reference this suite</caption><thead><tr><th>Schedule</th><th>State</th><th>Version policy</th></tr></thead><tbody>@for (schedule of state.schedules; track schedule.id) { <tr><td>{{ schedule.name }}<code class="table-subtitle">{{ schedule.id }}</code></td><td>{{ schedule.enabled ? 'Enabled' : 'Paused' }}</td><td>{{ schedule.releaseId ? 'Pinned version' : 'Follows current version' }}@if (schedule.releaseId) { <code class="table-subtitle">{{ schedule.releaseId }}</code> }</td></tr> }</tbody></table></div> } @else { <p class="small muted">No readable schedules reference this suite.</p> }
          <div class="row-actions lifecycle-actions"><a class="button secondary" [routerLink]="['/projects', manager.configuration.projectId(), 'releases']" [queryParams]="{ suiteId: state.suite.id }">Suite version history</a><a class="button secondary" [routerLink]="['/projects', manager.configuration.projectId(), 'schedules']">Manage schedules</a>@if (state.canManage) { @if (state.suite.archived) { <button type="button" class="button primary" [disabled]="manager.busy() || manager.conflict() || manager.form.dirty" (click)="manager.changeState('restore')">Restore suite</button> } @else { <button type="button" class="button secondary" [disabled]="manager.busy() || manager.conflict() || manager.form.dirty" (click)="manager.changeState('archive')">Archive suite</button> } }</div>@if (manager.form.dirty) { <p class="small muted">Save or reload settings before changing the suite lifecycle.</p> }
        } @else if (manager.busy()) { <p role="status">Loading suite settings…</p> }
      </section>
    }
  `,
  styles: [`.section-top, .row-actions { flex-wrap: wrap; } fieldset { border: 0; padding: 0; margin: 1.5rem 0; min-width: 0; } dd, code { overflow-wrap: anywhere; } code { font-size: 11px; } table { min-width: 36rem; } td:last-child { max-width: 24rem; } .lifecycle-actions { margin: 1.5rem 0; } select[multiple] { min-height: 6rem; }`]
})
export class SuiteManagerComponent {
  readonly manager = inject(SuiteManagerService)
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  constructor() { afterRenderEffect(() => { this.heading()?.nativeElement.focus() }) }
}
