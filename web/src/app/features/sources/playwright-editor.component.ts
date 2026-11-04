import { afterNextRender, ChangeDetectionStrategy, Component, ElementRef, inject, Injector, input, signal } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { CodeEditorComponent } from './code-editor.component'
import { PlaywrightEditorService } from './playwright-editor.service'
import { protectedEditorFile } from './playwright-editor.utils'
import { APIDraftWizardComponent } from './api-draft-wizard.component'

@Component({
  selector: 'ts-playwright-editor', standalone: true, imports: [FormsModule, CodeEditorComponent, APIDraftWizardComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="playwright-editor-title">
      <div class="section-top"><div><h2 id="playwright-editor-title" tabindex="-1">{{ api() ? 'API test editor' : 'Playwright editor' }}</h2><p class="small muted">Write tests here. Each draft contains a complete suite workspace. TestStudio handles the source bundle.</p></div>
        @if (service.sources.workspace.can('sources.manage')) { <button type="button" class="button primary" [disabled]="service.busy() || service.sources.draftDirty() || creating()" (click)="api() ? creating.set(true) : service.create()">{{ api() ? 'New API draft' : 'New Playwright draft' }}</button> }
      </div>
      <ol class="editor-steps"><li>Write and save</li><li>Validate and prepare</li><li>Review and publish</li><li>Start a test run</li></ol>
      @if (api()) { <p class="notice">API tests use a guarded request client and a registered environment. Start with one request, then add steps and assertions in code. Review request rules and safety before publishing.</p> }
      @else { <p class="notice">The starter runs an offline HTML fixture. Recorded website drafts use the selected environment’s api endpoint. They can reach only that origin. Review steps and safety labels before publishing.</p> }
      @if (!service.preparation.bundleRunners().includes('playwright')) { <p class="notice">Browser source preparation is unavailable on this installation. You can still save and edit drafts.</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
      @if (api() && creating()) { <ts-api-draft-wizard (created)="closeWizard(true)" (cancelled)="closeWizard(false)" /> }
      @if (!creating() && service.document(); as document) {
        @if (document.targetUrl) { <p class="notice warning">Recorded start address: {{ document.targetUrl }}. The review source binds recorded start-origin navigation to TESTSTUDIO_BASE_URL. Execution uses the selected registered environment, not your personal recording session. Register a worker-reachable api endpoint; container localhost cannot reach your desktop server.</p> }
        @if (document.manifestPath) { <p class="notice">This workspace retains its source paths and package declarations. Its manifest is {{ document.manifestPath }}. Preparation uses only the trusted runtime dependencies.</p> }
        <fieldset [disabled]="service.busy() || !service.sources.workspace.can('sources.manage')">
          <div class="section-top"><div class="field draft-name"><label for="playwright-draft-name">Draft name</label><input id="playwright-draft-name" maxlength="120" [ngModel]="document.name" (ngModelChange)="service.editName($event)"></div><span class="small muted" role="status">{{ service.sources.draftDirty() ? 'Unsaved changes' : 'Saved revision ' + service.saved()?.revision }}</span></div>
          <div class="file-actions"><label for="playwright-editor-file">File</label><select id="playwright-editor-file" [ngModel]="service.selectedPath()" (ngModelChange)="service.selectedPath.set($event)">@for (file of document.files; track file.path) { <option [value]="file.path">{{ file.path }}</option> }</select>
            @if (!protectedFile(document, service.selectedPath())) { <button type="button" class="button small secondary" (click)="service.removeFile(service.selectedPath())">Remove selected file</button> }
          </div>
          <p id="playwright-code-hint" class="small muted editor-hint">Edit a test file to change test steps and assertions. Keep each test tag in sync with its catalog selector. Edit the catalog for test IDs, titles, timeouts, and capture settings. Ctrl/Cmd+F searches; Tab moves focus.</p>
          @for (file of document.files; track file.path) {
            @if (file.path === service.selectedPath()) { <ts-code-editor [path]="file.path" [content]="file.content" [disabled]="service.busy() || !service.sources.workspace.can('sources.manage')" (changed)="service.editFile(file.path, $event)" /> }
          }
          <details class="add-file"><summary>Add a source or helper file</summary><div class="file-actions"><label for="playwright-new-file">New file path</label><input id="playwright-new-file" placeholder="checks/another.spec.ts" [ngModel]="newPath()" (ngModelChange)="newPath.set($event)"><button type="button" class="button secondary" (click)="addFile()">Add file</button></div><p class="small muted">Up to 50 files and 1 MiB of text. New starter drafts use generated package files. Imported workspaces retain their own declarations.</p></details>
          <div class="row-actions wrap-actions">
            <button type="button" class="button primary" [disabled]="!service.sources.draftDirty() || !document.name.trim()" (click)="service.save()">{{ api() ? 'Save API draft' : 'Save Playwright draft' }}</button>
            <button type="button" class="button secondary" [disabled]="!service.saved() || service.sources.draftDirty() || service.preparation.busy() || !service.preparation.bundleRunners().includes('playwright') || !service.sources.workspace.can('catalog.import')" (click)="service.prepare()">Validate and prepare draft</button>
            @if (service.sources.draftDirty()) { <button type="button" class="button secondary" (click)="service.discard()">Discard local changes</button> }
          </div>
        </fieldset>
        @if (service.sources.draftDirty()) { <p class="small muted">Save or discard these changes before choosing another suite or draft. Saving does not execute or publish code.</p> }
      }
      @if (service.drafts().length) {
        <div class="table-scroll"><table><caption class="sr-only">Saved Playwright drafts</caption><thead><tr><th>Draft</th><th>Revision</th><th>Action</th></tr></thead><tbody>@for (row of service.drafts(); track row.id) { <tr><td>{{ row.data.name }}</td><td>{{ row.revision }}</td><td><button type="button" class="button small secondary" [disabled]="service.busy() || service.sources.draftDirty()" (click)="service.open(row.id)" [attr.aria-label]="'Open draft ' + row.data.name">Open draft</button></td></tr> }</tbody></table></div>
      }
      @if (service.cursor()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.load(true)">More drafts</button> }
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } .editor-steps { display: flex; flex-wrap: wrap; gap: .75rem 2rem; padding-left: 1.25rem; color: var(--muted); font-size: 12px; margin: 0 0 1rem; } .draft-name { flex: 1; max-width: 32rem; } .file-actions { display: flex; gap: .75rem; align-items: center; flex-wrap: wrap; } .file-actions input { width: min(100%, 25rem); } .file-actions select { max-width: 100%; } .editor-hint { margin: .75rem 0; } .row-actions { margin: 1rem 0 .5rem; } .add-file { margin-top: 1rem; } .add-file .file-actions { margin: .75rem 0; } table { width: 100%; margin-top: 1rem; }`]
})
export class PlaywrightEditorComponent {
  private readonly host = inject<ElementRef<HTMLElement>>(ElementRef)
  private readonly injector = inject(Injector)
  readonly api = input(false)
  readonly creating = signal(false)
  readonly service = inject(PlaywrightEditorService)
  readonly newPath = signal('')
  readonly protectedFile = protectedEditorFile
  closeWizard(created: boolean): void {
    this.creating.set(false)
    afterNextRender(() => this.host.nativeElement.querySelector<HTMLElement>(created ? 'h2' : 'button')?.focus(), { injector: this.injector })
  }
  addFile(): void { if (this.service.addFile(this.newPath())) this.newPath.set('') }
}
