import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { WizardStepperComponent } from '../../shared/wizard/wizard-stepper.component'
import { WizardHeadingComponent } from '../../shared/wizard/wizard-heading.component'
import { WizardFormDirective } from '../../shared/wizard/wizard-form.directive'
import { BrowserRecorderService } from './browser-recorder.service'
import { RecorderConnectionComponent } from './recorder-connection.component'
import { EmbeddedRecorderService } from './embedded-recorder.service'
import { EmbeddedRecorderComponent } from './embedded-recorder.component'

@Component({
  selector: 'ts-browser-recorder', standalone: true, imports: [ReactiveFormsModule, DatePipe, WizardStepperComponent, WizardHeadingComponent, WizardFormDirective, RecorderConnectionComponent, EmbeddedRecorderComponent], providers: [BrowserRecorderService, EmbeddedRecorderService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="browser-recorder-title">
      <h2 id="browser-recorder-title">Record a browser test</h2>
      <p class="muted">Configure the test, record actions and assertions, then review the generated source before saving a draft.</p>
      @if (!service.enabled() && !service.busy()) { <p class="notice">Browser recording is unavailable on this installation. Ask your operator to enable a recording worker or desktop recorder. You can still use the source editor below.</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.enabled()) {
        <ts-wizard-stepper [steps]="service.flow().steps" [current]="service.flow().current" [interactive]="false" label="Browser recording progress" />
        <ts-wizard-heading [title]="service.flow().title" [step]="service.flow().current">Recording stays private to your account until you save a source draft. Recording does not run a test.</ts-wizard-heading>
      }
      @if (service.enabled() && !service.selected() && service.sources.workspace.can('sources.manage')) {
        <form [formGroup]="service.form" tsWizardForm #feedback="tsWizardForm" (ngSubmit)="service.start(); feedback.reveal()"><fieldset [disabled]="service.busy() || service.sources.draftDirty()">
          <div class="form-grid"><div class="field"><label for="recording-title">Recorded test title</label><input id="recording-title" formControlName="title" maxlength="120" required></div><div class="field"><label for="recording-address">Recording start address</label><input id="recording-address" type="url" formControlName="url" placeholder="https://example.com" required><small>{{ service.mode() === 'embedded' ? 'Use the recording browser right here. You do not need to install anything.' : service.mode() === 'companion' ? 'The browser opens on the computer running your desktop companion.' : 'The browser opens on this TestStudio server’s desktop.' }} HTTPS uses port 443 automatically.</small></div></div>
          <details><summary>Advanced recording details</summary><div class="field"><label for="recording-test-id">Recorded test ID</label><input id="recording-test-id" formControlName="testId" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" maxlength="128"><small>The generated permanent identifier links this test to its future run history.</small></div></details>
          <p class="small muted">{{ service.mode() === 'embedded' ? 'Start recording opens a private browser in this page. Use the website, add checks, and review the captured steps.' : service.mode() === 'companion' ? 'Continue creates a private session and connection code. The browser opens after you connect the desktop recorder.' : 'Start browser recording opens a dedicated browser and Playwright Inspector.' }} Review generated source and safety before saving, publication, or execution.</p>
          <div class="row-actions wrap-actions"><button type="submit" class="button primary">{{ service.busy() ? 'Starting…' : service.mode() === 'embedded' ? 'Start recording' : service.mode() === 'companion' ? 'Continue to desktop connection' : 'Start browser recording' }}</button><button type="button" class="button secondary" (click)="service.resetConfiguration()">Reset configuration</button></div>
        </fieldset></form>
      }
      @if (service.selected(); as row) {
        <article aria-labelledby="recording-session-title"><h3 id="recording-session-title">{{ row.data.title }}</h3>
          <p role="status">{{ service.stateLabel(row.data.state) }} · {{ row.data.testId }}</p><p class="small muted wrap-anywhere">Start address: {{ row.data.url }} · Session ends {{ row.data.deadlineAt | date:'short':'UTC' }} UTC.</p>
          @if (row.data.error) { <p class="notice warning">{{ row.data.error }}</p> }
          @if (row.data.state === 'pairing') { <ts-recorder-connection /> }
          @if (row.data.mode === 'embedded' && row.data.state === 'recording') { <ts-embedded-recorder [workspace]="workspace()" /> }
          @else if (row.data.state === 'starting' || row.data.state === 'recording') { <p>Use the dedicated browser and Playwright Inspector to record clicks and assertions. Return here when the flow is complete.</p> }
          @if (row.data.state === 'stopping') { <p class="notice">Waiting for the recorder to close and send its final captured steps. Keep this session open to see the result.</p> }
          @if (row.data.state === 'ready' && row.data.mode === 'embedded') { <h4>Review your test</h4><ol aria-label="Recorded test steps">@for (step of row.data.steps; track $index) { <li>{{ step.description }}</li> }</ol><p>Save this draft to keep the recorded steps. Review its target and safety before publishing or running.</p>@if (service.savedRecording()) { <p role="status">Test draft saved. No tests were started.</p><section aria-label="Next step for saved recording"><h4>Next: prepare your test</h4><p>Preparation checks this saved revision and discovers its test. You will review publication and choose when to run it.</p>@if (service.editor.preparation.bundleRunners().includes('playwright')) { <button type="button" class="button primary" [disabled]="service.editor.busy() || service.editor.preparation.busy() || service.sources.draftDirty() || !service.sources.workspace.can('catalog.import')" (click)="service.prepareRecording()">Prepare saved test</button>@if (service.editor.message()) { <p role="status">{{ service.editor.message() }}</p> } } @else { <p class="notice">Your recording is saved. This installation needs a Playwright preparation worker before it can prepare or run browser tests. Ask your operator to finish that setup.</p> }</section> }@if (service.editor.error()) { <p role="alert" class="notice error">{{ service.editor.error() }}</p> } }
          @else if (row.data.state === 'ready') { <p>Review the generated code, assertions, environment target, and safety label. Save a source draft in the editor below. Preparing, publishing, and running remain separate actions.</p> }
          @if (row.data.state === 'cancelled') { <p>This recording was cancelled. Its private history remains available.</p> }
          <div class="row-actions wrap-actions">
            @if (row.data.state === 'starting' || row.data.state === 'recording') { <button type="button" class="button primary" [disabled]="service.busy() || service.sources.recordingInputPending() || embedded.external() || embedded.uncertain()" (click)="service.stop()">{{ row.data.mode === 'embedded' ? 'Finish recording' : 'Stop and keep recording' }}</button> }
            @if (service.active()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.stop(true)">Cancel recording</button> }
            @if (row.data.state === 'ready' && row.data.mode === 'embedded') { <button type="button" class="button primary" [disabled]="service.editor.busy() || service.savedRecording()" (click)="service.saveRecording()">Save test draft</button> }
            @if (row.data.state === 'ready') { <button type="button" class="button secondary" [disabled]="service.editor.busy() || service.sources.draftDirty()" (click)="service.reviewSource(row.id)">{{ row.data.mode === 'embedded' ? 'Advanced: review source' : 'Review recorded test' }}</button> }
            @if (!service.active()) { <button type="button" class="button secondary" [disabled]="service.busy() || service.sources.draftDirty()" (click)="service.newRecording()">New recording</button> }
            <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.read(row.id)">Refresh recording</button>
          </div>
          @if (row.data.code) { <details><summary>Generated Playwright code</summary><pre>{{ row.data.code }}</pre></details> }
        </article>
      }
      @if (service.items().length) { <details><summary>Your recording history</summary><ul>@for (row of service.items(); track row.id) { <li><button type="button" class="button small secondary" [disabled]="service.busy() || service.active() || service.sources.creationDirty()" (click)="service.read(row.id)">{{ row.data.title }} · {{ service.stateLabel(row.data.state) }}</button></li> }</ul></details> }
      @if (service.cursor()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load(true)">More recordings</button> }
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 1rem 0; min-width: 0; } article { margin: 1rem 0; } .row-actions { margin: .75rem 0; } details { margin: 1rem 0; } pre { max-height: 22rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 12px; } li { margin: .5rem 0; }`]
})
export class BrowserRecorderComponent { readonly workspace = input(false); readonly service = inject(BrowserRecorderService); readonly embedded = inject(EmbeddedRecorderService) }
