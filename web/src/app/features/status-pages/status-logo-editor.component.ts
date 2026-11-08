import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { StatusEditorService } from './status-editor.service'
import { StatusLogoComponent } from './status-logo.component'

@Component({
 selector: 'ts-status-logo-editor', standalone: true, imports: [ReactiveFormsModule, StatusLogoComponent], changeDetection: ChangeDetectionStrategy.OnPush,
 template: `<section [formGroup]="service.form" aria-labelledby="logo-heading"><h3 id="logo-heading">Logo (optional)</h3><p class="small muted">Centered above the page title. Uploads keep their transparency and proportions.</p>
  <div class="field"><label for="status-logo-source">Logo source</label><select id="status-logo-source" [value]="service.logoMode()" (change)="service.logoSource($any($event.target).value)" [disabled]="service.logoUploading()"><option value="none">No logo</option><option value="upload">Upload image</option><option value="url">Image URL</option></select></div>
  @if (service.logoMode() === 'upload') { <div class="field"><label for="status-logo-upload">Upload logo</label><input id="status-logo-upload" type="file" accept="image/png,image/jpeg,image/webp" [disabled]="service.logoUploading()" (change)="upload($event)"><small>PNG, JPEG, or WebP, up to 10 MB. Resized automatically for the page.</small></div> }
  @if (service.logoMode() === 'url') { <div class="field"><label for="status-logo-url">Logo image URL</label><input id="status-logo-url" type="url" formControlName="logo" maxlength="2048" placeholder="https://example.com/logo.png"><small>Use a public HTTPS image URL. Visitors load it from that host.</small></div> }
  @if (service.logoMode() !== 'none') { <div class="field"><label for="status-logo-alt">Logo alternative text (optional)</label><input id="status-logo-alt" formControlName="logoAlt" maxlength="256"><small>Describe the organization or brand. Defaults to the page title plus “logo”.</small></div> }
  @if (service.logoUploading()) { <p role="status">Preparing logo…</p> } @if (service.logoError()) { <p class="notice error" role="alert">{{ service.logoError() }}</p> }
  @if (service.logoPreview()) { <div class="logo-preview"><ts-status-logo [source]="service.logoPreview()" [description]="service.form.controls.logoAlt.value || service.form.controls.title.value + ' logo'" /><button type="button" class="button subtle small" [disabled]="service.logoUploading()" (click)="service.logoSource('none')">Remove logo</button></div> }
 </section>`,
 styles: `section {margin:24px 0;border-block:1px solid var(--border);padding:20px 0;} h3 {margin-top:0;} .logo-preview {display:grid;gap:12px;padding:20px;background:var(--surface-alt);border-radius:8px;} .logo-preview button {justify-self:center;}`
})
export class StatusLogoEditorComponent {
 readonly service = inject(StatusEditorService)
 upload(event: Event): void { const input = event.target as HTMLInputElement, file = input.files?.[0]; if (file) void this.service.uploadLogo(file); input.value = '' }
}
