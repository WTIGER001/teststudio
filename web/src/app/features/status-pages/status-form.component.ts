import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { StatusEditorService } from './status-editor.service'
import { statusWindows } from './status.utils'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'

@Component({
 selector: 'ts-status-form', standalone: true, imports: [ReactiveFormsModule, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective], changeDetection: ChangeDetectionStrategy.OnPush,
 template: `    <section class="panel" aria-label="Status page setup"><ts-wizard-stepper [singleRow]="true" [steps]="['Appearance', 'Checks', 'Review']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.busy()" (selected)="service.flow.go($event)" />
      <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="advance(); feedback.reveal()"><ts-wizard-heading [title]="['Page appearance', 'Choose public checks', 'Review private draft'][service.flow.step()]" [step]="service.flow.step()" />
      <fieldset [disabled]="service.busy() || service.locked() || !service.canManage()">
        @switch (service.flow.step()) {
          @case (0) {
            <div class="field"><label for="status-title">Page title</label><input id="status-title" formControlName="title" maxlength="256" required></div>
            <div class="field"><label for="status-slug">Vanity URL (optional)</label><div class="vanity-input"><span>{{ service.origin }}/status/</span><input id="status-slug" formControlName="slug" maxlength="63" autocomplete="off" spellcheck="false" placeholder="customer-health" aria-describedby="slug-guidance slug-feedback"></div><p id="slug-guidance" class="small muted">Unique across TestStudio. Letters, numbers, and hyphens; names ignore case. A draft reserves its name. Publish to activate it.</p><p id="slug-feedback" class="small" role="status" [class.error]="service.slugState() === 'taken'">{{ service.slugMessage() }}</p></div>
            <div class="form-grid"><div class="field"><label for="status-background">Background color</label><input id="status-background" type="color" formControlName="background"></div><div class="field"><label for="status-default-window">Default availability period</label><select id="status-default-window" formControlName="defaultWindow">@for (window of windows; track window.id) { <option [value]="window.id">{{ window.label }}</option> }</select></div></div>
            <details><summary>Advanced monitoring settings</summary><div class="field"><label for="status-freshness">Mark observations stale after (seconds)</label><input id="status-freshness" type="number" min="60" max="86400" formControlName="maxAgeSeconds"><p class="small muted">Default: 900 seconds (15 minutes). After this, time without another observation is unknown. Match this to your monitoring cadence.</p></div></details>
          }
          @case (1) {
            <p>Only the public text below appears on the status page. Probe targets and private diagnostics remain private.</p>
            <label for="status-probe-picker">Add HTTP probe</label><select id="status-probe-picker" #picker (change)="add(picker)"><option value="">Choose a configured probe</option>@for (probe of service.probes(); track probe.id) { <option [value]="probe.id" [disabled]="service.selected(probe.id)">{{ probe.name }} · {{ probe.stage }} · {{ probe.applicationId }}</option> }</select>
            <details (toggle)="service.loadChecks($any($event.target).open)"><summary>Other test types</summary><p class="small muted">Choose any registered test. Outcome history is the default. Monitoring uses this page's freshness policy and does not create a schedule.</p>
              @if (service.checksLoading()) { <p role="status">Loading checks…</p> }
              @if (service.checksError()) { <p role="alert">{{ service.checksError() }}</p><button type="button" class="button secondary" (click)="service.loadChecks(true)">Retry checks</button> }
              <div class="field"><label for="status-check">Test to include</label><select id="status-check" #check><option value="">Choose a test</option>@for (row of service.checks(); track row.suiteId + '/' + row.id) { <option [value]="row.suiteId + '/' + row.id">{{ row.data.title }} · {{ row.data.type }} · {{ row.suiteName }}</option> }</select></div>
              <div class="field"><label for="status-check-environment">Test environment</label><select id="status-check-environment" #environment><option value="">No target</option>@for (row of service.environments(); track row.id) { <option [value]="row.id">{{ row.name }}</option> }</select></div>
              <div class="field"><label for="status-check-location">Test execution location</label><input id="status-check-location" #location placeholder="Installation default" maxlength="128"></div>
              <label><input type="checkbox" #monitor>Report this test as a monitor</label>
              <button type="button" class="button secondary" (click)="service.chooseCheck(check.value, environment.value, location.value, monitor.checked)">Add selected test</button>
              @if (service.checksCursor()) { <button type="button" class="button secondary" (click)="service.loadChecks(true, true)">More checks</button> }
            </details>
            @if (!service.probes().length && !service.busy()) { <p class="notice">No HTTP probes are configured. Use Other test types to include a registered test, or add a probe from HTTPS probes.</p> }
            @for (entry of service.entries(); track service.entryKey(entry); let index = $index) {
              <section class="entry" [attr.aria-label]="'Public check ' + (index + 1)"><h3>Check {{ index + 1 }}</h3><div class="field"><label [for]="'public-title-' + index">Public title {{ index + 1 }}</label><input [id]="'public-title-' + index" [value]="entry.title" (input)="service.edit(index, 'title', $any($event.target).value)" maxlength="256" required></div><div class="field"><label [for]="'public-description-' + index">Description {{ index + 1 }} (optional)</label><textarea [id]="'public-description-' + index" [value]="entry.description" (input)="service.edit(index, 'description', $any($event.target).value)" maxlength="2048"></textarea></div>
              <div class="row-actions"><button type="button" class="button secondary small" [disabled]="index === 0" (click)="service.move(index, -1)" [attr.aria-label]="'Move check ' + (index + 1) + ' up'">Move up</button><button type="button" class="button secondary small" [disabled]="index === service.entries().length - 1" (click)="service.move(index, 1)" [attr.aria-label]="'Move check ' + (index + 1) + ' down'">Move down</button><button type="button" class="button subtle small" (click)="service.remove(index)" [attr.aria-label]="'Remove check ' + (index + 1)">Remove</button></div></section>
            }
          }
          @case (2) {
            <h3>{{ service.form.controls.title.value }}</h3><p>Background: {{ service.form.controls.background.value }} · Default period: {{ service.form.controls.defaultWindow.value }} · Freshness: {{ service.form.controls.maxAgeSeconds.value }} seconds</p><ol>@for (entry of service.entries(); track service.entryKey(entry)) { <li><strong>{{ entry.title }}</strong><p>{{ entry.probeId || entry.monitor ? 'Time-based availability' : 'Execution outcomes' }}</p>@if (entry.description) { <p>{{ entry.description }}</p> }</li> }</ol>
            <p><strong>Public address after publication:</strong> {{ service.origin }}/status/{{ service.normalizedSlug() || service.record()?.data?.publicToken || 'generated-on-save' }}</p><p class="notice">Save draft stores this configuration privately. Publish is a separate reviewed action. The public page will show these titles, descriptions, statuses, observation times, and availability aggregates to anyone with its URL.</p>
          }
        }
        <div class="row-actions wrap-actions">@if (service.flow.step() > 0) { <button class="button secondary" type="button" (click)="service.flow.back()">Back</button> }<button class="button primary" type="submit">{{ service.flow.step() === 2 ? 'Save draft' : 'Continue' }}</button></div>
      </fieldset></form>
    </section>`,
 styles: `fieldset {border:0;padding:0;min-width:0;margin-top:20px;} .entry {padding:20px 0;border-bottom:1px solid var(--border);} .row-actions {margin-top:20px;flex-wrap:wrap;} details {margin:20px 0;} input[type=color] {height:44px;max-width:100px;} select,textarea {max-width:100%;} li p {margin-top:6px;} .panel {overflow-wrap:anywhere;} .vanity-input {display:flex;align-items:center;gap:8px;flex-wrap:wrap;} .vanity-input span {font-size:12px;overflow-wrap:anywhere;} .vanity-input input {flex:1;min-width:12rem;} .error {color:var(--error);}`
})
export class StatusFormComponent {
 readonly service = inject(StatusEditorService); readonly windows = statusWindows
 advance(): void { if (this.service.flow.step() < 2) this.service.flow.next(); else void this.service.save() }
 add(picker: HTMLSelectElement): void { const probe = this.service.probes().find(row => row.id === picker.value); if (probe) this.service.choose(probe); picker.value = '' }
}
