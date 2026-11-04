import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { BundleUploadComponent } from './bundle-upload.component'
import { PlaywrightEditorComponent } from './playwright-editor.component'
import { PlaywrightEditorService } from './playwright-editor.service'
import { BrowserRecorderComponent } from './browser-recorder.component'
import { PreparationService } from './preparation.service'

@Component({
  selector: 'ts-source-preparation', standalone: true, imports: [DatePipe, ReactiveFormsModule, BundleUploadComponent, PlaywrightEditorComponent, BrowserRecorderComponent], providers: [PreparationService, PlaywrightEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (mode() === 'all' || mode() === 'playwright') { <ts-browser-recorder /><ts-playwright-editor /> }
    @if (mode() === 'api') { <ts-playwright-editor [api]="true" /> }
    @if (mode() !== 'playwright' && mode() !== 'api' && service.bundleRunners().length && service.sources.workspace.can('sources.manage') && service.sources.workspace.can('catalog.import')) { <ts-source-bundle /> }
    <section class="panel" aria-labelledby="preparation-title">
      <div class="section-top"><h2 id="preparation-title">Suite preparation</h2><button class="button secondary" type="button" [disabled]="service.busy()" (click)="service.load()">Refresh preparations</button></div>
      <p class="muted">Follow a source bundle build or inspect an immutable Go or Playwright image. Review its discovered catalog before publication. Preparation does not start a test run.</p>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (mode() !== 'playwright' && mode() !== 'api' && service.runners().length && service.sources.workspace.can('sources.manage') && service.sources.workspace.can('catalog.import')) {
        <form [formGroup]="service.form" (ngSubmit)="service.prepare()"><fieldset [disabled]="service.busy()"><div class="form-grid">
          <div class="field"><label for="prebuilt-runner">Prebuilt runner</label><select id="prebuilt-runner" formControlName="runner">@for (runner of service.runners(); track runner) { <option [value]="runner">{{ runner }}</option> }</select></div>
          <div class="field"><label for="prebuilt-image">Immutable image</label><input id="prebuilt-image" formControlName="image" required placeholder="registry/repository@sha256:…"><small>The image must already be available to this worker and belong to this project. Choose a suite without a target environment.</small></div>
        </div><div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="service.form.invalid || service.busy()">Prepare image</button><button type="button" class="button secondary" (click)="service.newRequest()">New preparation request</button></div></fieldset></form>
      } @else if (mode() !== 'playwright' && mode() !== 'api' && !service.runners().length) { <p class="notice">Prebuilt image preparation is unavailable on this server.</p> }
      @if (service.selected(); as row) {
        <article class="preparation-detail" aria-labelledby="preparation-detail-title"><h3 id="preparation-detail-title">Preparation details</h3>
          <dl class="detail-grid"><dt>Identifier</dt><dd>{{ row.id }}</dd><dt>Image</dt><dd>{{ row.data.image || 'Build pending' }}</dd>@if (row.data.sourceName) { <dt>Source ZIP</dt><dd>{{ row.data.sourceName }}</dd><dt>Manifest path</dt><dd>{{ row.data.manifestPath }}</dd><dt>Source digest</dt><dd>{{ row.data.sourceDigest }}</dd> }@if (row.data.build; as build) { <dt>Assembly started</dt><dd>{{ build.startedAt | date:'medium':'UTC' }} UTC</dd><dt>Assembly finished</dt><dd>{{ build.finishedAt | date:'medium':'UTC' }} UTC</dd> }@if (row.data.runtime) { <dt>Runtime</dt><dd>{{ row.data.runtime }}</dd> }@if (row.data.sourceDraftId) { <dt>Editor draft</dt><dd>{{ row.data.sourceDraftId }} · revision {{ row.data.sourceDraftRevision }}</dd> }<dt>State</dt><dd>{{ row.data.state }} · {{ row.data.phase }}</dd><dt>Created</dt><dd>{{ row.data.createdAt | date:'medium':'UTC' }} UTC</dd>@if (row.data.releaseId) { <dt>Published release</dt><dd>{{ row.data.releaseId }}</dd> }</dl>
          @if (row.data.error) { <p class="notice error">{{ row.data.error }}</p> }
          @if (row.data.sourceExpired) { <p class="notice">The original source ZIP has expired. Preparation diagnostics remain available.</p> }
          @if (row.data.cleanupPending) { <p class="notice warning">Resource cleanup is pending. The server retains the reference for recovery.</p> }
          <div class="row-actions wrap-actions">
            <button type="button" class="button secondary" (click)="service.inspect(row.id)">Refresh preparation</button>
            @if (row.data.uploadId) { <button type="button" class="button primary" (click)="service.review()">Review prepared source</button> }
            @if ((row.data.state === 'received' || row.data.state === 'preparing' || row.data.state === 'discovering' || row.data.state === 'ready') && service.sources.workspace.can('sources.manage')) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.cancel()">{{ row.data.state === 'ready' ? 'Withdraw review' : 'Cancel preparation' }}</button> }
            @if (row.data.sourceBundleId && !row.data.sourceExpired && row.data.state !== 'expired') { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.downloadBundle()">Get original source ZIP</button> }
            @if (row.data.sourceBundleId && row.data.runner === 'playwright' && !row.data.sourceExpired && row.data.state !== 'expired' && service.sources.workspace.can('sources.manage')) { <button type="button" class="button secondary" [disabled]="editor.busy() || service.sources.draftDirty()" (click)="editor.importPreparation(row.id)">Edit a copy in Playwright</button> }
            @if (service.downloadURL()) { <a class="button secondary" [href]="service.downloadURL()" [download]="row.data.sourceName || 'source.zip'">Save original source ZIP</a> }
          </div>
          @if (row.data.state === 'received' && row.data.sourceBundleId) { <p role="status" class="muted">Waiting for a preparation worker at {{ row.data.executionLocation }}.</p> }
          @if (row.data.log) { <details><summary>Preparation log</summary><pre>{{ row.data.log }}</pre>@if (row.data.logTruncated) { <p class="notice warning">The retained preparation log is truncated.</p> }</details> }
        </article>
      }
      @if (service.jobs().length) { <div class="table-scroll"><table><caption class="sr-only">Source preparation history</caption><thead><tr><th>Runner and image</th><th>State</th><th>Created</th><th>Action</th></tr></thead><tbody>@for (row of service.jobs(); track row.id) { <tr><td>{{ row.data.runner }}<span class="table-subtitle image-reference">{{ row.data.sourceName || row.data.image }}</span></td><td>{{ row.data.state }}</td><td>{{ row.data.createdAt | date:'medium':'UTC' }} UTC</td><td><button type="button" class="button small secondary" [disabled]="service.busy()" (click)="service.inspect(row.id)" [attr.aria-label]="'Inspect preparation ' + row.id">Inspect</button></td></tr> }</tbody></table></div> }
      @if (service.cursor()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load(true)">More preparations</button> }
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } .preparation-detail { margin: 1.5rem 0; } .row-actions { margin: 1rem 0; } table { min-width: 42rem; } .image-reference, dd { overflow-wrap: anywhere; } pre { max-height: 22rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 12px; }`]
})
export class PreparationComponent {
  readonly mode = input<'all' | 'playwright' | 'go' | 'api'>('all')
  readonly service = inject(PreparationService)
  readonly editor = inject(PlaywrightEditorService)
}
