import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { BundleUploadService } from './bundle-upload.service'

@Component({
  selector: 'ts-source-bundle', standalone: true, imports: [ReactiveFormsModule], providers: [BundleUploadService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="bundle-upload-title">
      <h2 id="bundle-upload-title">Upload a source bundle</h2>
      <p class="muted">Upload a complete Go or Playwright suite as a ZIP. Include its manifest, catalog, source, and dependency declarations. Offline browser bundles use the trusted runtime dependencies. Builds do not start tests.</p>
      @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
      <form [formGroup]="service.form" (ngSubmit)="service.upload()"><fieldset [disabled]="service.preparation.busy()"><div class="form-grid">
        <div class="field"><label for="source-bundle-file">Source ZIP</label><input id="source-bundle-file" type="file" accept=".zip,application/zip" required (change)="choose($event)"><small>Maximum 32 MiB compressed and 128 MiB expanded. Links and paths outside the source context are rejected.</small></div>
        <div class="field"><label for="source-bundle-manifest">Suite manifest path</label><input id="source-bundle-manifest" formControlName="manifest" required placeholder="teststudio-suite.json"><small>Paths in this manifest refer to the ZIP root. Select a different manifest to prepare another suite from the same repository.</small></div>
      </div><button type="submit" class="button primary" [disabled]="service.form.invalid || !service.file() || service.preparation.busy()">Upload and prepare bundle</button></fieldset></form>
      <p class="small muted">A separate preparation worker processes queued builds. Review the discovered catalog below before publishing a release.</p>
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; margin: 0; min-width: 0; } button { margin-top: 1rem; }`]
})
export class BundleUploadComponent {
  readonly service = inject(BundleUploadService)
  choose(event: Event): void { this.service.select((event.target as HTMLInputElement).files?.[0] ?? null) }
}
