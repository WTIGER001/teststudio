import { ChangeDetectionStrategy, Component, ElementRef, inject, ViewChild } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ProjectCoversService } from './project-covers.service'

@Component({
  selector: 'ts-project-cover-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <dialog #dialog aria-labelledby="cover-title" aria-describedby="cover-description" (cancel)="$event.preventDefault(); close()">
      <div class="section-top"><h2 id="cover-title">Project image</h2><button type="button" class="button secondary small" [disabled]="service.busy() === 'saving'" (click)="close()" aria-label="Close project image">Close ×</button></div>
      <p id="cover-description" class="muted">Choose a background for {{ service.name() }}. Review it before saving.</p>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      @if (service.conflict()) { <button type="button" class="button secondary" (click)="service.open(service.selected(), service.name())">Reload current image</button> }
      @if (service.busy()) { <p role="status">{{ service.busy() === 'capturing' ? 'Choose the website tab in the browser sharing prompt. Capturing one frame…' : service.busy() === 'saving' ? 'Saving image…' : service.busy() === 'loading' ? 'Loading image…' : 'Preparing image…' }}</p> }
      <div class="cover-preview" [class.with-image]="service.preview()">
        @if (service.preview()) { <img [src]="service.preview()" alt="Project image preview"> }
        <div><h3>{{ service.name() }}</h3><span>Open project →</span></div>
      </div>
      <fieldset [disabled]="!service.ready() || !!service.busy() || service.conflict()">
        <div class="field"><label for="project-image-upload">Upload a screenshot or image</label><input id="project-image-upload" type="file" accept="image/png,image/jpeg,image/webp" (change)="upload($event)"><small>PNG, JPEG, or WebP, up to 10 MB. The saved thumbnail is resized. Images fill the square tile and crop at the center.</small></div>
        <details><summary>Capture from a website</summary>
          <p class="small muted">Open your website, then select its tab in the browser sharing prompt. Capture takes one frame and stops sharing. Verify the selected page in the preview.</p>
          <div class="field"><label for="project-image-url">Website address</label><input id="project-image-url" type="url" [formControl]="service.address" placeholder="https://app.example.com"></div>
          <div class="row-actions">@if (service.website()) { <a class="button secondary" [href]="service.website()" target="_blank" rel="noopener noreferrer">Open website ↗</a> }<button class="button secondary" type="button" [disabled]="!service.captureAvailable" (click)="service.captureTab()">Capture website tab</button></div>
          @if (!service.captureAvailable) { <p class="notice">This browser does not support tab capture. Open TestStudio in a browser with screen sharing, or upload a screenshot.</p> }
        </details>
        @if (service.preview()) { <button class="button secondary small" type="button" (click)="service.remove()">Remove image</button> }
      </fieldset>
      <div class="row-actions"><button class="button primary" type="button" [disabled]="!service.dirty() || !!service.busy() || service.conflict()" (click)="save()">Save image</button><button class="button secondary" type="button" [disabled]="service.busy() === 'saving'" (click)="close()">Cancel</button></div>
    </dialog>
  `,
  styles: [`dialog { width: min(36rem, calc(100vw - 2rem)); max-height: calc(100dvh - 2rem); overflow-y: auto; padding: 1.5rem; background: var(--surface); color: var(--text); border: 1px solid var(--border); border-radius: 12px; } dialog::backdrop { background: rgb(0 0 0 / 65%); } .section-top { margin-bottom: .75rem; } h2 { font-size: 22px; } fieldset { border: 0; margin: 1rem 0; padding: 0; min-width: 0; } details { margin-bottom: 1rem; } summary { cursor: pointer; padding: .75rem 0; } .cover-preview { width: min(100%, 18rem); aspect-ratio: 1; position: relative; overflow: hidden; border-radius: 10px; background: radial-gradient(at top left, var(--accent-soft), var(--surface-muted)); border: 1px solid var(--border); } .cover-preview img { position: absolute; width: 100%; height: 100%; object-fit: cover; } .cover-preview > div { position: absolute; inset: 0; padding: 1.25rem; display: flex; flex-direction: column; justify-content: end; gap: .75rem; overflow-wrap: anywhere; } .with-image > div { color: white; background: linear-gradient(transparent, rgb(0 0 0 / 92%)); }`]
})
export class ProjectCoverEditorComponent {
  readonly service = inject(ProjectCoversService)
  @ViewChild('dialog', { static: true }) private dialog!: ElementRef<HTMLDialogElement>
  open(id: string, name: string): void { this.dialog.nativeElement.showModal(); void this.service.open(id, name) }
  close(): void { if (this.service.close()) this.dialog.nativeElement.close() }
  async save(): Promise<void> { if (await this.service.save()) { this.service.close(); this.dialog.nativeElement.close() } }
  upload(event: Event): void { const input = event.target as HTMLInputElement, file = input.files?.[0]; if (file) void this.service.upload(file); input.value = '' }
}
