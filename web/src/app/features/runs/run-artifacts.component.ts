import { DatePipe, DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ArtifactViewService } from './artifact-view.service'

@Component({
  selector: 'ts-run-artifacts', standalone: true, imports: [DatePipe, DecimalPipe], providers: [ArtifactViewService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="artifacts-title">
      <div class="section-top"><h2 id="artifacts-title">Run artifacts</h2><span class="small muted">Private evidence with retention and checksums.</span></div>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.loading()) { <p role="status" class="small muted">Preparing artifact access…</p> }
      @if (service.evidence.artifacts().length) {
        <div class="table-scroll"><table><caption class="sr-only">Logs, screenshots, traces, videos, and other retained evidence</caption><thead><tr><th scope="col">Artifact</th><th scope="col">Test / attempt</th><th scope="col">Size</th><th scope="col">State / expiry</th><th scope="col">Actions</th></tr></thead><tbody>
          @for (item of service.evidence.artifacts(); track item.id) { <tr><td><strong class="wrap-anywhere">{{ item.data.filename }}</strong><span class="table-subtitle muted">{{ item.data.kind }}</span></td><td>{{ item.data.testId || 'Run' }}<span class="table-subtitle muted">{{ item.data.attempt ? 'Attempt ' + item.data.attempt : '' }}</span></td><td>{{ item.data.size | number }} bytes</td><td>{{ item.data.state }}<time class="table-subtitle muted" [attr.datetime]="item.data.expiresAt">{{ item.data.expiresAt | date:'mediumDate' }}</time>@if (item.data.error) { <span class="table-subtitle">{{ item.data.error }}</span> }</td><td><div class="row-actions">@if (item.data.state === 'available') { @if (service.workspace.can('evidence.read')) { <button class="button small subtle" type="button" [disabled]="service.loading()" (click)="service.open(item.data, 'view')">View</button> }@if (service.workspace.can('evidence.download')) { <button class="button small subtle" type="button" [disabled]="service.loading()" (click)="service.open(item.data, 'download')">Download</button> } }</div></td></tr> }
        </tbody></table></div>
      } @else { <p class="small muted">No artifacts are recorded yet.</p> }
      @if (service.evidence.cursors()['artifacts']) { <button type="button" class="button secondary more-button" [disabled]="service.evidence.loading()" (click)="service.evidence.more('artifacts')">Load more artifacts</button> }
      @if (service.preview(); as preview) {
        <section class="artifact-preview" aria-labelledby="preview-title">
          <div class="section-top"><h3 id="preview-title" class="wrap-anywhere">{{ preview.artifact.filename }}</h3><button type="button" class="button subtle" (click)="service.close()">Close preview</button></div>
          @switch (preview.kind) {
            @case ('text') { <pre class="log-view" tabindex="0">{{ preview.text }}</pre>@if (preview.truncated) { <p class="notice">The preview shows the first 1 MiB. Use Download for the complete artifact.</p> } }
            @case ('image') { <img class="evidence-image" [src]="preview.url" [alt]="preview.artifact.filename" (error)="service.error.set('The image link expired or content is unavailable. Select View to refresh it.')"> }
            @case ('video') { <video class="evidence-video" [src]="preview.url" controls preload="metadata" aria-label="Recorded test video" (error)="service.error.set('The video link expired or content is unavailable. Select View to refresh it.')"></video> }
            @case ('download') { <p class="small muted">Your download link is ready.</p><a class="button primary" [href]="preview.url" target="_blank" rel="noopener noreferrer">Download {{ preview.artifact.filename }}</a> }
          }
          <p class="small muted">Access expires {{ preview.expiresAt | date:'mediumTime' }}. Select the artifact again to refresh access.</p>
          <details><summary>Artifact identity</summary><dl class="detail-grid"><dt>SHA-256</dt><dd class="wrap-anywhere"><code>{{ preview.artifact.checksum }}</code></dd><dt>Content type</dt><dd>{{ preview.artifact.contentType }}</dd></dl></details>
        </section>
      }
    </section>
  `
})
export class RunArtifactsComponent { readonly service = inject(ArtifactViewService) }
