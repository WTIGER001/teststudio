import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { BrowserRecorderService } from './browser-recorder.service'

@Component({
  selector: 'ts-browser-recorder', standalone: true, imports: [ReactiveFormsModule, DatePipe], providers: [BrowserRecorderService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="browser-recorder-title">
      <h2 id="browser-recorder-title">Record a browser test</h2>
      <p class="muted">Open your website in a dedicated browser. Click through the flow and add assertions with the Playwright recorder toolbar. Stop recording, then review and save the generated test below.</p>
      @if (!service.enabled()) { <p class="notice">This installation has no local desktop recorder. Enable the trusted Playwright recorder on the computer running TestStudio. Hosted recording needs a local companion or streamed browser.</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.enabled() && service.sources.workspace.can('sources.manage')) {
        <form [formGroup]="service.form" (ngSubmit)="service.start()"><fieldset [disabled]="service.busy() || service.active() || service.sources.draftDirty()">
          <div class="form-grid"><div class="field"><label for="recording-address">Recording start address</label><input id="recording-address" type="url" formControlName="url" placeholder="http://localhost:3000" required><small>The recording browser opens on this TestStudio server's desktop.</small></div><div class="field"><label for="recording-test-id">Recorded test ID</label><input id="recording-test-id" formControlName="testId" required></div><div class="field"><label for="recording-title">Recorded test title</label><input id="recording-title" formControlName="title" maxlength="120" required></div></div>
          <button type="submit" class="button primary" [disabled]="service.form.invalid">Start browser recording</button>
        </fieldset></form>
      }
      @if (service.selected(); as row) {
        <article aria-labelledby="recording-session-title"><h3 id="recording-session-title">{{ row.data.title }}</h3>
          <p role="status">Recording: {{ row.data.state }} · {{ row.data.testId }}</p><p class="small muted">Time limit: {{ row.data.deadlineAt | date:'short':'UTC' }} UTC. Sessions and generated code remain private to the recording owner until saved as a source draft.</p>
          @if (row.data.error) { <p class="notice warning">{{ row.data.error }}</p> }
          <div class="row-actions wrap-actions">
            @if (service.active()) { <button type="button" class="button primary" [disabled]="service.busy()" (click)="service.stop()">Stop and keep recording</button><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.stop(true)">Cancel recording</button> }
            @if (row.data.state === 'ready') { <button type="button" class="button primary" [disabled]="service.editor.busy() || service.sources.draftDirty()" (click)="service.editor.importRecording(row.id)">Review recorded test</button> }
            <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.read(row.id)">Refresh recording</button>
          </div>
          @if (row.data.code) { <details><summary>Generated Playwright code</summary><pre>{{ row.data.code }}</pre></details> }
        </article>
      }
      @if (service.items().length) { <details><summary>Your recording history</summary><ul>@for (row of service.items(); track row.id) { <li><button type="button" class="button small secondary" [disabled]="service.busy() || service.active()" (click)="service.read(row.id)">{{ row.data.title }} · {{ row.data.state }}</button></li> }</ul></details> }
      @if (service.cursor()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load(true)">More recordings</button> }
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 1rem 0; min-width: 0; } article { margin: 1rem 0; } .row-actions { margin: .75rem 0; } pre { max-height: 22rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 12px; } li { margin: .5rem 0; }`]
})
export class BrowserRecorderComponent { readonly service = inject(BrowserRecorderService) }
