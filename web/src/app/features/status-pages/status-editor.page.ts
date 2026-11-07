import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { StatusEditorService } from './status-editor.service'
import { StatusViewComponent, statusWindows } from './status-view.component'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'

@Component({
  standalone: true, imports: [ReactiveFormsModule, RouterLink, StatusViewComponent, WizardStepperComponent, WizardHeadingComponent], providers: [StatusEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <p><a [routerLink]="['/projects', service.project(), 'status-pages']">← Status pages</a></p>
    <div class="page-heading"><div><span class="eyebrow">Project status</span><h1>{{ service.record() ? 'Manage status page' : 'Create status page' }}</h1><p class="muted">Choose what visitors can see. A saved draft stays private until published.</p></div><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.reload()">Reload saved page</button></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }@if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    @if (service.locked()) { <p class="notice warning">The saved state changed or could not be confirmed. Your entries remain. Inspect Status pages or reload the saved page before another mutation.</p> }
    <section class="panel"><ts-wizard-stepper [singleRow]="true" [steps]="['Appearance', 'Probes', 'Review']" [current]="service.flow.step()" [reached]="service.flow.reached()" [disabled]="service.busy()" (selected)="service.flow.go($event)" />
      <form [formGroup]="service.form" (ngSubmit)="advance()"><ts-wizard-heading [title]="['Page appearance', 'Choose public probes', 'Review private draft'][service.flow.step()]" [step]="service.flow.step()" />
      <fieldset [disabled]="service.busy() || service.locked() || !service.canManage()">
        @switch (service.flow.step()) {
          @case (0) {
            <div class="field"><label for="status-title">Page title</label><input id="status-title" formControlName="title" maxlength="256" required></div>
            <div class="form-grid"><div class="field"><label for="status-background">Background color</label><input id="status-background" type="color" formControlName="background"></div><div class="field"><label for="status-default-window">Default availability period</label><select id="status-default-window" formControlName="defaultWindow">@for (window of windows; track window.id) { <option [value]="window.id">{{ window.label }}</option> }</select></div></div>
            <details><summary>Advanced monitoring settings</summary><div class="field"><label for="status-freshness">Mark observations stale after (seconds)</label><input id="status-freshness" type="number" min="60" max="86400" formControlName="maxAgeSeconds"><p class="small muted">Default: 900 seconds (15 minutes). After this, time without another observation is unknown. Match this to your monitoring cadence.</p></div></details>
          }
          @case (1) {
            <p>Only the public text below appears on the status page. Probe targets and private diagnostics remain private.</p>
            <label for="status-probe-picker">Add HTTP probe</label><select id="status-probe-picker" #picker (change)="add(picker)"><option value="">Choose a configured probe</option>@for (probe of service.probes(); track probe.id) { <option [value]="probe.id" [disabled]="service.selected(probe.id)">{{ probe.name }} · {{ probe.stage }} · {{ probe.applicationId }}</option> }</select>
            @if (!service.probes().length && !service.busy()) { <p class="notice">Add an HTTP probe from HTTPS probes before configuring this page.</p> }
            @for (entry of service.entries(); track entry.probeId; let index = $index) {
              <section class="entry" [attr.aria-label]="'Public probe ' + (index + 1)"><h3>Probe {{ index + 1 }}</h3><div class="field"><label [for]="'public-title-' + index">Public title {{ index + 1 }}</label><input [id]="'public-title-' + index" [value]="entry.title" (input)="service.edit(index, 'title', $any($event.target).value)" maxlength="256" required></div><div class="field"><label [for]="'public-description-' + index">Description {{ index + 1 }} (optional)</label><textarea [id]="'public-description-' + index" [value]="entry.description" (input)="service.edit(index, 'description', $any($event.target).value)" maxlength="2048"></textarea></div>
              <div class="row-actions"><button type="button" class="button secondary small" [disabled]="index === 0" (click)="service.move(index, -1)" [attr.aria-label]="'Move probe ' + (index + 1) + ' up'">Move up</button><button type="button" class="button secondary small" [disabled]="index === service.entries().length - 1" (click)="service.move(index, 1)" [attr.aria-label]="'Move probe ' + (index + 1) + ' down'">Move down</button><button type="button" class="button subtle small" (click)="service.remove(index)" [attr.aria-label]="'Remove probe ' + (index + 1)">Remove</button></div></section>
            }
          }
          @case (2) {
            <h3>{{ service.form.controls.title.value }}</h3><p>Background: {{ service.form.controls.background.value }} · Default period: {{ service.form.controls.defaultWindow.value }} · Freshness: {{ service.form.controls.maxAgeSeconds.value }} seconds</p><ol>@for (entry of service.entries(); track entry.probeId) { <li><strong>{{ entry.title }}</strong>@if (entry.description) { <p>{{ entry.description }}</p> }</li> }</ol>
            <p class="notice">Save draft stores this configuration privately. Publish is a separate reviewed action. The public page will show these titles, descriptions, statuses, observation times, and availability aggregates to anyone with its URL.</p>
          }
        }
        <div class="row-actions wrap-actions">@if (service.flow.step() > 0) { <button class="button secondary" type="button" (click)="service.flow.back()">Back</button> }<button class="button primary" type="submit">{{ service.flow.step() === 2 ? 'Save draft' : 'Continue' }}</button></div>
      </fieldset></form>
    </section>
    @if (service.record()) {
      <section class="panel"><h2>Publication</h2><p>{{ service.record()!.data.published ? 'Published snapshot is live. Draft edits do not affect it.' : 'Private draft. Visitors cannot access it.' }}</p>
        @if (service.dirty()) { <p class="notice">Save your changes before previewing or publishing.</p> }
        <div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.busy() || service.dirty()" (click)="service.showPreview()">Preview saved draft</button>@if (service.canPublish()) { <button type="button" class="button primary" [disabled]="service.busy() || service.dirty() || service.locked()" (click)="service.publish()">{{ service.reviewPublish() ? 'Confirm publish' : 'Review publication' }}</button>@if (service.record()!.data.published) { <button type="button" class="button secondary" [disabled]="service.busy() || service.locked()" (click)="service.unpublish()">Unpublish</button> } }</div>
        @if (service.reviewPublish()) { <p class="notice warning">Review the preview below. Confirm publish makes its selected text and status history public without sign-in. Private targets, logs, and evidence stay private.</p> }
        @if (service.record()!.data.published) { <div class="field"><label for="status-url">Public URL</label><input id="status-url" readonly [value]="service.publicURL()"><a [href]="service.publicURL()" target="_blank" rel="noopener noreferrer">Open public status page</a></div><div class="field"><label for="status-embed">Iframe embed code</label><textarea id="status-embed" readonly [value]="service.embed()" rows="3"></textarea></div> }
        @if (service.canDelete()) { <details><summary>Advanced page actions</summary><p>Deletion withdraws the URL and removes the page configuration. Probe tests and history remain.</p><button type="button" class="button danger" [disabled]="service.busy() || service.locked()" (click)="service.delete()">Delete status page</button></details> }
      </section>
    }
    @if (service.preview(); as preview) { <section class="panel"><h2>Public preview</h2><ts-status-view [report]="preview" (windowChanged)="service.showPreview($event)" /></section> }
  `,
  styles: `fieldset {border:0;padding:0;min-width:0;margin-top:20px;} .entry {padding:20px 0;border-bottom:1px solid var(--border);} .row-actions {margin-top:20px;flex-wrap:wrap;} details {margin:20px 0;} input[type=color] {height:44px;max-width:100px;} select,textarea {max-width:100%;} li p {margin-top:6px;} .panel {overflow-wrap:anywhere;}`
})
export class StatusEditorPage {
  readonly service = inject(StatusEditorService); readonly windows = statusWindows
  advance(): void { if (this.service.flow.step() < 2) this.service.flow.next(); else void this.service.save() }
  add(picker: HTMLSelectElement): void { const probe = this.service.probes().find(row => row.id === picker.value); if (probe) this.service.choose(probe); picker.value = '' }
  canLeave(): boolean { return !this.service.dirty() || window.confirm('Leave this status page without saving your changes?') }
}
