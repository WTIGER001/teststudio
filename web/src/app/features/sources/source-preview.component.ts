import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { FormsModule } from '@angular/forms'
import { RouterLink } from '@angular/router'
import { SourcesService } from './sources.service'
import { sourceDefinition } from './source.utils'

@Component({
  selector: 'ts-source-preview', standalone: true, imports: [DatePipe, FormsModule, RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.preview(); as preview) {
      <section class="panel" aria-labelledby="source-review-title">
        <h2 id="source-review-title">Source review</h2>
        <dl class="detail-grid"><dt>File</dt><dd>{{ preview.upload.data.name }}</dd><dt>Suite</dt><dd>{{ preview.upload.data.suiteId }}</dd><dt>State</dt><dd>{{ preview.upload.data.state }}</dd><dt>Source digest</dt><dd class="code-input">{{ preview.upload.data.digest }}</dd><dt>Staging expires</dt><dd><time [attr.datetime]="preview.upload.data.expiresAt">{{ preview.upload.data.expiresAt | date:'medium':'UTC' }} UTC</time></dd>
          @if (preview.upload.data.artifact; as artifact) { <dt>Prepared image</dt><dd>{{ artifact.image }}</dd>@if (artifact.provenance; as origin) { <dt>Image-reported source</dt><dd>{{ origin.repository || origin.snapshotDigest }} @if (origin.reference) { <span>· {{ origin.reference }}</span> } @if (origin.commit) { <div>{{ origin.commit }}</div> }</dd> } }
          @if (preview.upload.data.releaseId) { <dt>Immutable release</dt><dd class="code-input">{{ preview.upload.data.releaseId }}</dd><dt>Activated on publication</dt><dd>{{ preview.upload.data.activated ? 'Yes' : 'No' }}</dd> }
        </dl>
        @for (issue of preview.upload.data.issues; track $index) { <p class="notice warning">{{ issue.testId }} · {{ issue.field }}: {{ issue.message }}</p> }
        @if (preview.plan; as plan) {
          <div class="table-scroll"><table><caption class="sr-only">Discovered test changes</caption><thead><tr><th>Change</th><th>Identifier and title</th><th>Category</th><th>Type</th><th>Safety</th><th>Definition</th></tr></thead><tbody>
            @for (change of plan.changes; track change.test.id) {
              <tr><td>{{ change.action }}</td><td><strong>{{ change.test.id }}</strong><div class="small muted">{{ change.test.title }}</div></td><td>{{ change.test.category }}</td><td>{{ change.test.type }}</td><td>{{ change.test.safety }}</td><td><details><summary>Inspect changes</summary>@if (change.before) { <h3>Before</h3><pre>{{ definition(change.before) }}</pre> }<h3>After</h3><pre>{{ definition(change.test) }}</pre></details></td></tr>
            }
          </tbody></table></div>
        }
        <div class="row-actions wrap-actions source-actions">
          @if (preview.upload.data.state !== 'expired') { <button type="button" class="button secondary" [disabled]="service.saving()" (click)="service.showContent()">View retained source</button> }
          @if (preview.upload.data.releaseId) { <a class="button secondary" [routerLink]="['/projects', service.project(), 'catalog']">Open test catalog</a> }
        </div>
        @if (service.content()) { <div class="field"><label for="retained-source">Retained source document</label><textarea id="retained-source" class="code-input" [value]="service.content()" rows="10" readonly spellcheck="false"></textarea></div> }
        @if (preview.upload.data.state === 'staged' && preview.upload.data.actor === service.session.user()?.id && service.workspace.can('sources.manage') && service.workspace.can('catalog.publish')) {
          <p class="notice">Publication freezes an executable catalog. It does not mean the checks passed. Tests start only through a separate run action.</p>
          <label class="checkbox-field"><input type="checkbox" [ngModel]="service.activate()" (ngModelChange)="service.activate.set($event)" [disabled]="service.saving()">Activate this release for future runs</label>
          <button class="button primary" type="button" (click)="service.publish()" [disabled]="service.saving()">Publish source release</button>
        }
      </section>
    }
  `,
  styles: [`table { min-width: 44rem; } td { vertical-align: top; } pre { max-width: 34rem; max-height: 20rem; overflow: auto; white-space: pre-wrap; overflow-wrap: anywhere; font-size: 11px; } .source-actions { margin: 1.25rem 0; }`]
})
export class SourcePreviewComponent { readonly service = inject(SourcesService); readonly definition = sourceDefinition }
