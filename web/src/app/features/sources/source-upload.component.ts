import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { SourcesService } from './sources.service'

@Component({
  selector: 'ts-source-upload', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="upload-title">
      <div class="section-top"><h2 id="upload-title">Upload native checks</h2><button class="button subtle" type="button" [disabled]="service.saving()" (click)="service.editor.set(false)">Close upload</button></div>
      <p class="muted">Upload JSON or YAML in the TestStudio catalog format. Review changes before publication. Omitted tests remain in the suite.</p>
      <p class="small muted">Each upload accepts up to 30 native HTTP checks and 1 MiB of UTF-8 source. Staging expires after 30 minutes. For Playwright code, use the editor below. ZIP bundles and GitHub sources support complete coded suites.</p>
      <form [formGroup]="service.form" (ngSubmit)="service.stage()">
        <fieldset [disabled]="service.saving()">
          <div class="field"><label for="source-file">Source file</label><input id="source-file" type="file" accept=".json,.yaml,.yml,application/json,application/yaml,text/yaml" (change)="service.file($any($event.target).files?.[0])"></div>
          <div class="form-grid">
            <div class="field"><label for="source-name">Source file name</label><input id="source-name" formControlName="name" maxlength="200" required placeholder="health.yaml"></div>
            <div class="field"><label for="source-format">Source format</label><select id="source-format" formControlName="format"><option value="json">JSON</option><option value="yaml">YAML</option></select></div>
          </div>
          <div class="field"><label for="source-document">Source document</label><textarea id="source-document" class="code-input" formControlName="content" rows="10" spellcheck="false" required placeholder='{"schemaVersion":"teststudio.catalog/v1","tests":[]}'></textarea></div>
          <button class="button primary" type="submit" [disabled]="service.form.invalid || service.saving()">Validate source and preview</button>
        </fieldset>
      </form>
    </section>
  `,
  styles: [`fieldset { border: 0; padding: 0; min-width: 0; margin: 0; }`]
})
export class SourceUploadComponent { readonly service = inject(SourcesService) }
