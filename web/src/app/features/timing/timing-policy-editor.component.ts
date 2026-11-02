import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { TimingPolicyEditorService } from './timing-policy-editor.service'

@Component({
  selector: 'ts-timing-policy-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <fieldset class="timing-fieldset"><legend>Duration and slowdown rules</legend>
      <label class="checkbox-field"><input type="checkbox" [formControl]="service.override"> Override the published suite timing defaults for this test</label>
      <p class="small muted">Defaults use five clean observations, a 2× slowdown threshold with a 100 ms noise guard, and a bounded adaptive timeout. Suite or project settings can change these defaults.</p>
      @if (service.enabled()) {
        <div [formGroup]="service.form" class="form-grid">
          <div class="field"><label for="timing-mode">Slowdown action</label><select id="timing-mode" formControlName="mode"><option value="fail">Fail test</option><option value="warn">Warn only</option><option value="off">Off</option></select></div>
          <div class="field"><label for="timing-factor">Slowdown factor</label><input id="timing-factor" type="number" formControlName="slowdownFactor" min="1" max="100" step="0.1" required><small>2 means twice the historical median.</small></div>
          <div class="field"><label for="timing-increase">Minimum increase (ms)</label><input id="timing-increase" type="number" formControlName="minimumIncreaseMs" min="0" max="3600000" required></div>
          <div class="field"><label for="timing-samples">Minimum clean observations</label><input id="timing-samples" type="number" formControlName="minimumSamples" min="3" max="1000" step="1" required></div>
          <div class="field"><label for="timing-window">History window (observations)</label><input id="timing-window" type="number" formControlName="window" min="3" max="1000" step="1" required></div>
          <div class="field"><label for="timing-age">Maximum history age (days)</label><input id="timing-age" type="number" formControlName="maxAgeDays" min="1" max="3650" step="1" required></div>
          <div class="field"><label for="timing-timeout-mode">Attempt timeout</label><select id="timing-timeout-mode" formControlName="timeoutMode"><option value="adaptive">Adaptive within the catalog ceiling</option><option value="fixed">Fixed catalog ceiling</option></select></div>
          <div class="field"><label for="timing-timeout-factor">Timeout factor</label><input id="timing-timeout-factor" type="number" formControlName="timeoutFactor" min="1" max="100" step="0.1" required><small>Multiplies the historical 95th percentile.</small></div>
          <div class="field"><label for="timing-timeout-margin">Timeout margin (ms)</label><input id="timing-timeout-margin" type="number" formControlName="timeoutMarginMs" min="0" max="3600000" required></div>
          <div class="field"><label for="timing-timeout-minimum">Minimum adaptive timeout (ms)</label><input id="timing-timeout-minimum" type="number" formControlName="minimumTimeoutMs" min="1" max="3600000" step="1" required></div>
        </div>
        @if (service.value() && service.issue()) { <p class="notice error" role="alert">{{ service.issue() }}</p> }
        @if (service.form.invalid) { <p class="small notice warning">Complete all timing controls within their permitted ranges.</p> }
        <p class="small muted">The historical 95th percentile also guards the slowdown threshold. The catalog ceiling always caps the timeout.</p>
      }
    </fieldset>
  `
})
export class TimingPolicyEditorComponent { readonly service = inject(TimingPolicyEditorService) }
