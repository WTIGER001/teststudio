import { ChangeDetectionStrategy, Component, inject, output } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { TimingPolicyEditorComponent } from '../timing/timing-policy-editor.component'
import { HTTPCheckEditorComponent } from './http-check-editor.component'
import { CatalogTestEditorService } from './catalog-test-editor.service'

@Component({
  selector: 'ts-catalog-test-form', standalone: true, imports: [ReactiveFormsModule, TimingPolicyEditorComponent, HTTPCheckEditorComponent], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        <form [formGroup]="service.form" (ngSubmit)="save()">
          <div class="form-grid">
            <div class="field"><label for="test-suite">Suite</label><select id="test-suite" formControlName="suite" required>@for (suite of service.suites(); track suite.id) { <option [value]="suite.id">{{ suite.data.name }}</option> }</select></div>
            <div class="field"><label for="test-id">Permanent identifier</label><input id="test-id" formControlName="id" required pattern="[A-Za-z0-9](?:[A-Za-z0-9._]|-){0,127}" placeholder="HEALTH-001"></div>
            <div class="field"><label for="test-title">Title</label><input id="test-title" formControlName="title" required></div>
            <div class="field"><label for="test-category">Category</label><input id="test-category" formControlName="category" required></div>
            <div class="field"><label for="test-timeout">Catalog timeout ceiling (seconds)</label><input id="test-timeout" type="number" formControlName="timeout" min="1" max="3600" step="1" required><small>Adaptive timeouts cannot exceed this ceiling.</small></div>
            <div class="field"><label for="test-safety">Production-safety label</label><select id="test-safety" formControlName="safety"><option value="needs-review">Needs review</option><option value="safe">Safe</option><option value="unsafe">Unsafe</option></select></div>
            <div class="field"><label for="test-rationale">Safety rationale</label><textarea id="test-rationale" formControlName="rationale" rows="2" required placeholder="Describe requests, side effects, and required conditions."></textarea></div>
          </div>
          @if (service.editingHTTP()) { <ts-http-check-editor /> }
          <ts-timing-policy-editor />
          <button class="button primary" type="submit" [disabled]="service.saving() || service.form.invalid || !service.timingEditor.valid() || (service.editingHTTP() && !service.checkEditor.valid())">{{ service.saving() ? 'Saving…' : 'Save changes' }}</button>
        </form>
  `
})
export class CatalogTestFormComponent {
  readonly service = inject(CatalogTestEditorService)
  readonly saved = output<void>()
  async save(): Promise<void> { if (await this.service.save()) this.saved.emit() }
}
