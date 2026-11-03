import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { RepositoriesComponent } from './repositories.component'
import { PreparationComponent } from './preparation.component'
import { SourcesService } from './sources.service'
import { SourceUploadComponent } from './source-upload.component'
import { SourcePreviewComponent } from './source-preview.component'

@Component({
  standalone: true, imports: [DatePipe, ReactiveFormsModule, RouterLink, SourceUploadComponent, SourcePreviewComponent, PreparationComponent, RepositoriesComponent], providers: [SourcesService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.workspace.project()?.name || 'Project' }}</span><h1>Sources</h1><p class="muted">Review definitions and prepared suite images. Retain the source behind each release.</p></div>
      @if (service.workspace.can('sources.manage') && service.workspace.can('catalog.import')) { <button type="button" class="button primary" [disabled]="!service.form.controls.suite.value || service.saving()" (click)="service.openEditor()">+ Upload tests</button> }
    </div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    <ts-repository-sources />
    <section class="panel" aria-labelledby="source-suite-title"><h2 id="source-suite-title">Destination suite</h2><fieldset [disabled]="service.saving()"><div class="field" [formGroup]="service.form"><label for="source-suite">Suite for sources</label><select id="source-suite" formControlName="suite" [attr.aria-busy]="service.loading()"><option value="">Choose a suite</option>@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div></fieldset>
      <a class="small" [routerLink]="['/projects', service.project(), 'configuration']">Create or manage suites</a>
    </section>
    @if (service.editor()) { <ts-source-upload /> }
    <ts-source-preview />
    @if (service.form.controls.suite.value) { <ts-source-preparation /> }
    @if (service.form.controls.suite.value) {
      <section class="panel" aria-labelledby="source-history-title"><div class="section-top"><h2 id="source-history-title">Upload history</h2><button type="button" class="button secondary" [disabled]="service.loading() || service.saving()" (click)="service.list()">Refresh uploads</button></div>
        @if (service.loading()) { <p role="status">Loading uploads…</p> }
        @if (service.items().length) {
          <div class="table-scroll"><table><caption class="sr-only">Retained uploads for the selected suite</caption><thead><tr><th>Source</th><th>State</th><th>Uploaded</th><th>Release</th><th>Action</th></tr></thead><tbody>
            @for (item of service.items(); track item.id) { <tr><td><strong>{{ item.data.name }}</strong><div class="small muted">{{ item.data.format }} · {{ item.data.bytes }} bytes</div></td><td>{{ item.data.state }}</td><td><time [attr.datetime]="item.data.createdAt">{{ item.data.createdAt | date:'medium':'UTC' }} UTC</time></td><td class="release-id">{{ item.data.releaseId || 'Not published' }}</td><td><button type="button" class="button small secondary" [disabled]="service.saving()" (click)="service.inspect(item)" [attr.aria-label]="'Inspect ' + item.data.name">Inspect</button></td></tr> }
          </tbody></table></div>
        } @else if (!service.loading()) { <div class="empty-state"><h3>No uploads in this suite.</h3><p>Upload native HTTP checks to preview a versioned release.</p></div> }
        @if (service.cursor()) { <button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.list(true)">More uploads</button> }
      </section>
    }
  `,
  styles: [`fieldset { border: 0; padding: 0; min-width: 0; margin: 0; } .table-scroll table { min-width: 46rem; } .release-id { min-width: 12rem; max-width: 16rem; overflow-wrap: anywhere; font-family: ui-monospace, monospace; font-size: 11px; }`]
})
export class SourcesPage { readonly service = inject(SourcesService) }
