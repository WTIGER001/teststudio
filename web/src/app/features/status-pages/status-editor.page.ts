import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { CopyButtonComponent } from '../../shared/clipboard/copy-button.component'
import { StatusEditorService } from './status-editor.service'
import { StatusViewComponent } from './status-view.component'
import { StatusFormComponent } from './status-form.component'

@Component({
 standalone: true, imports: [RouterLink, StatusViewComponent, StatusFormComponent, CopyButtonComponent], providers: [StatusEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
 template: `
  <div [class.creation-page]="service.editing()">
   <div class="page-heading"><div><span class="eyebrow">Project status</span><h1>{{ service.mode() === 'create' ? 'Create status page' : service.mode() === 'edit' ? 'Edit status page' : service.record()?.data?.title || 'Status page' }}</h1><p class="muted">{{ service.editing() ? 'Choose appearance and checks, review, then save.' : 'Enable this page and share its public URL or embed.' }}</p></div><a class="button secondary" [routerLink]="service.editing() && service.record() ? ['/projects', service.project(), 'status-pages', service.id()] : ['/projects', service.project(), 'status-pages']">{{ service.editing() && service.record() ? 'Back to status page' : 'All status pages' }}</a></div>
   @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }@if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
   @if (service.locked()) { <p class="notice warning">The saved state changed or could not be confirmed. Your entries remain. Inspect the saved page before saving again.</p> }
   @if (service.busy() && !service.record()) { <p role="status">Loading status page…</p> }
   @if (service.editing()) {
    <ts-status-form />
    <div class="edit-footer"><a [routerLink]="service.record() ? ['/projects', service.project(), 'status-pages', service.id()] : ['/projects', service.project(), 'status-pages']">Cancel {{ service.mode() === 'create' ? 'setup' : 'editing' }}</a>@if (service.record() || service.locked()) { <button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.reload()">Reload saved page</button> }</div>
   } @else if (service.record(); as page) {
    <section class="panel summary"><div><span class="badge" [class.success]="!!page.data.published">{{ page.data.published ? 'Enabled' : 'Disabled' }}</span><h2>Configuration</h2><p>{{ page.data.probes.length }} checks · {{ page.data.defaultWindow }} default period</p><p class="small muted">Saving changes updates an enabled public page immediately.</p>@if (service.canManage() && service.canPublish()) { <label><input type="checkbox" [checked]="!!page.data.published" [disabled]="service.busy() || service.locked()" (change)="service.setEnabled($any($event.target).checked)">Enabled</label><p class="small muted">Anyone with the URL can view an enabled page.</p> }</div><div class="row-actions">@if (service.canManage()) { <a class="button primary" [routerLink]="['/projects', service.project(), 'status-pages', service.id(), 'edit']">Edit status page</a> }</div></section>
    @if (page.data.published) { <section class="panel"><h2>Share this page</h2><div class="field"><div class="share-heading"><label for="status-url">Public URL</label><ts-copy-button name="Public URL" [value]="service.publicURL()" /></div><input id="status-url" readonly [value]="service.publicURL()"><a [href]="service.publicURL()" target="_blank" rel="noopener noreferrer">Open public status page</a></div><div class="field"><div class="share-heading"><label for="status-embed">Iframe embed code</label><ts-copy-button name="Iframe embed code" [value]="service.embed()" /></div><textarea id="status-embed" readonly [value]="service.embed()"></textarea></div></section> }
    @if (service.preview(); as preview) { <section class="preview"><h2>Saved page preview</h2><ts-status-view [report]="preview" (windowChanged)="service.showPreview($event)" /></section> }
    <details class="panel"><summary>Page actions</summary><div class="row-actions"><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.reload()">Reload saved page</button>@if (service.canDelete()) { <button type="button" class="button danger" [disabled]="service.busy() || service.locked()" (click)="service.delete()">Delete status page</button> }</div><p class="small muted">Disable the page to withdraw public access. Deletion removes its configuration; retired vanity names remain reserved.</p></details>
   }
  </div>
 `,
 styles: `:host {display:block;min-width:0;} .share-heading,.summary,.edit-footer,.row-actions {display:flex;align-items:center;justify-content:space-between;gap:16px;flex-wrap:wrap;} .creation-page {max-width:56rem;margin-inline:auto;} .edit-footer,.preview,details {margin-block:24px;} .panel {overflow-wrap:anywhere;} textarea,input {max-width:100%;}`
})
export class StatusEditorPage {
 readonly service = inject(StatusEditorService)
 canLeave(): boolean { return !this.service.dirty() || window.confirm('Leave this status page without saving your changes?') }
}
