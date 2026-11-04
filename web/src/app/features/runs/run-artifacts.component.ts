import { DatePipe, DecimalPipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, ElementRef, afterNextRender, inject, Injector, viewChild } from '@angular/core'
import type { Artifact } from './run.models'
import { ArtifactViewService } from './artifact-view.service'
import { ArtifactActionsComponent } from './artifact-actions.component'
import { ArtifactDeletionComponent } from './artifact-deletion.component'

@Component({
  selector: 'ts-run-artifacts', standalone: true, imports: [ArtifactActionsComponent, DatePipe, DecimalPipe, ArtifactDeletionComponent], providers: [ArtifactViewService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="artifacts-title">
      <div class="section-top"><h2 #heading tabindex="-1" id="artifacts-title">Run artifacts</h2><span class="small muted">Private evidence with retention and checksums.</span></div>
      @if (service.evidence.focusedTest()) { <p class="small muted">Test {{ service.evidence.focusedTest() }} · {{ service.evidence.focusedAttempt() ? 'Attempt ' + service.evidence.focusedAttempt() : 'All attempts' }}. Change the scope under Investigate a test.</p> }
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      <ts-artifact-deletion (deleted)="deleted()" />
      @if (service.loading()) { <p role="status" class="small muted">Preparing artifact access…</p> }
      @if (service.evidence.artifacts().length) {
        @if (service.evidence.focusedTest()) {
          @for (item of service.evidence.artifacts(); track item.id) {
            <article class="artifact-card" [attr.aria-label]="item.data.filename">
              <div class="section-top"><h3 class="wrap-anywhere">{{ item.data.filename }}</h3><span class="badge">{{ service.retention.artifactState(item.data) }}</span></div>
              <p class="small muted">{{ item.data.kind }} · {{ item.data.size | number }} bytes · {{ item.data.testId }} · Attempt {{ item.data.attempt }}</p>
              <p class="small muted">Access expiry: {{ item.data.expiresAt | date:'medium' }}</p>
              @if (item.data.error) { <p class="notice warning wrap-anywhere">{{ item.data.error }}</p> }
              <ts-artifact-actions [record]="item" (open)="open($event.artifact, $event.action, $event.event)" />
            </article>
          }
        } @else {
        <div class="table-scroll"><table><caption class="sr-only">Logs, screenshots, traces, videos, and other retained evidence</caption><thead><tr><th scope="col">Artifact</th><th scope="col">Test / attempt</th><th scope="col">Size</th><th scope="col">State / expiry</th><th scope="col">Actions</th></tr></thead><tbody>
          @for (item of service.evidence.artifacts(); track item.id) { <tr><td><strong class="wrap-anywhere">{{ item.data.filename }}</strong><span class="table-subtitle muted">{{ item.data.kind }}</span></td><td>{{ item.data.testId || 'Run' }}<span class="table-subtitle muted">{{ item.data.attempt ? 'Attempt ' + item.data.attempt : '' }}</span></td><td>{{ item.data.size | number }} bytes</td><td>{{ service.retention.artifactState(item.data) }}<time class="table-subtitle muted" [attr.datetime]="item.data.expiresAt">{{ item.data.expiresAt | date:'mediumDate' }}</time>@if (item.data.error) { <span class="table-subtitle">{{ item.data.error }}</span> }</td><td><ts-artifact-actions [record]="item" (open)="open($event.artifact, $event.action, $event.event)" /></td></tr> }
        </tbody></table></div>
        }
      } @else { <p class="small muted">{{ service.evidence.loading() ? 'Loading artifacts…' : service.evidence.error() ? 'Artifact data is unavailable. Read the evidence warning and retry.' : service.evidence.cursors()['artifacts'] ? 'No matching artifacts found in this scan. Load more artifacts to continue.' : 'No artifacts are recorded for this selection.' }}</p> }
      @if (service.evidence.cursors()['artifacts']) { <button type="button" class="button secondary more-button" [disabled]="service.evidence.loading()" (click)="service.evidence.more('artifacts')">Load more artifacts</button> }
      @if (service.preview(); as preview) {
        <section class="artifact-preview" aria-labelledby="preview-title">
          <div class="section-top"><h3 #previewHeading tabindex="-1" id="preview-title" class="wrap-anywhere">{{ preview.artifact.filename }}</h3><button type="button" class="button subtle" (click)="closePreview()">Close preview</button></div>
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
  `,
  styles: [`.artifact-card { min-width: 0; border: 1px solid var(--border); border-radius: 8px; padding: 14px; margin: 12px 0; } h3 { margin: 0; font-size: 14px; } .artifact-card .section-top { flex-wrap: wrap; gap: 10px; }`]
})
export class RunArtifactsComponent {
  readonly service = inject(ArtifactViewService)
  private readonly heading = viewChild<ElementRef<HTMLHeadingElement>>('heading')
  private readonly injector = inject(Injector)
  private readonly previewHeading = viewChild<ElementRef<HTMLHeadingElement>>('previewHeading')
  private returnFocus: HTMLElement | null = null
  async open(artifact: Artifact, action: 'view' | 'download', event: Event): Promise<void> {
    const trigger = event.currentTarget as HTMLElement
    await this.service.open(artifact, action)
    if (!this.service.preview()) return
    this.returnFocus = trigger
    afterNextRender(() => this.previewHeading()?.nativeElement.focus(), { injector: this.injector })
  }
  closePreview(): void { this.service.close(); if (this.returnFocus?.isConnected) this.returnFocus.focus() }
  deleted(): void { this.service.close(); this.heading()?.nativeElement.focus() }
}
