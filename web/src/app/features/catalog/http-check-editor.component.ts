import { ChangeDetectionStrategy, Component, inject, input } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { HTTPCheckEditorService } from './http-check-editor.service'

@Component({
  selector: 'ts-http-check-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div [formGroup]="service.form">
      <fieldset class="http-check-fieldset"><legend>HTTP request and status</legend><div class="form-grid">
        <div class="field"><label for="test-endpoint">Registered endpoint</label>@if (endpoints().length) { <select id="test-endpoint" formControlName="endpoint">@for (endpoint of endpoints(); track endpoint) { <option [value]="endpoint">{{ endpoint }}</option> }</select> } @else { <input id="test-endpoint" formControlName="endpoint" required [attr.aria-invalid]="!!service.errors().endpoint" aria-describedby="http-endpoint-help http-endpoint-error"> }<small id="http-endpoint-help">Use the endpoint name from the target environment.</small>@if (service.errors().endpoint) { <p class="field-error" id="http-endpoint-error">{{ service.errors().endpoint }}</p> }</div>
        <div class="field"><label for="test-path">Relative path</label><input id="test-path" formControlName="path" required placeholder="/healthz" [attr.aria-invalid]="!!service.errors().path" aria-describedby="http-path-error">@if (service.errors().path) { <p class="field-error" id="http-path-error">{{ service.errors().path }}</p> }</div>
        @if (!compact()) { <div class="field"><label for="http-method">Request method</label><select id="http-method" formControlName="method" [attr.aria-invalid]="!!service.errors().method" aria-describedby="http-method-error"><option value="GET">GET</option><option value="HEAD">HEAD</option></select>@if (service.errors().method) { <p class="field-error" id="http-method-error">{{ service.errors().method }}</p> }</div> }
        <div class="field"><label for="http-status-codes">Accepted status codes</label><input id="http-status-codes" formControlName="statusCodes" required placeholder="200, 204" [attr.aria-invalid]="!!service.errors().statusCodes" aria-describedby="http-status-error">@if (service.errors().statusCodes) { <p class="field-error" id="http-status-error">{{ service.errors().statusCodes }}</p> }</div>
      </div><p class="small muted">Redirects remain disabled. GET and HEAD do not by themselves establish production safety.</p></fieldset>
      <details [open]="!compact()"><summary>Advanced response assertions</summary>
      <fieldset class="http-check-fieldset"><legend>Response assertions</legend>
        <p class="small muted">Assertions are optional. HEAD supports status and headers, with no response body assertions.</p>
        <div class="form-grid">
          <div class="field"><label for="http-required-text">Required response text</label><textarea id="http-required-text" rows="2" formControlName="requiredText" [attr.aria-invalid]="!!service.errors().requiredText" aria-describedby="http-required-error"></textarea>@if (service.errors().requiredText) { <p class="field-error" id="http-required-error">{{ service.errors().requiredText }}</p> }</div>
          <div class="field"><label for="http-forbidden-text">Forbidden response text</label><textarea id="http-forbidden-text" rows="2" formControlName="forbiddenText" [attr.aria-invalid]="!!service.errors().forbiddenText" aria-describedby="http-forbidden-error"></textarea>@if (service.errors().forbiddenText) { <p class="field-error" id="http-forbidden-error">{{ service.errors().forbiddenText }}</p> }</div>
          <div class="field"><label for="http-headers">Expected response headers (JSON)</label><textarea class="code-input" id="http-headers" rows="4" formControlName="headers" placeholder='{"Content-Type": "application/json"}' [attr.aria-invalid]="!!service.errors().headers" aria-describedby="http-headers-help http-headers-error"></textarea><small id="http-headers-help">Map header names to exact string values. These are response assertions, not request credentials.</small>@if (service.errors().headers) { <p class="field-error" id="http-headers-error">{{ service.errors().headers }}</p> }</div>
          <div class="field"><label for="http-json">Expected JSON fields</label><textarea class="code-input" id="http-json" rows="4" formControlName="json" placeholder='{"status": "up", "items.0.id": 1}' [attr.aria-invalid]="!!service.errors().json" aria-describedby="http-json-help http-json-error"></textarea><small id="http-json-help">Map dot-separated paths to JSON values. Use numeric path segments for array indexes.</small>@if (service.errors().json) { <p class="field-error" id="http-json-error">{{ service.errors().json }}</p> }</div>
        </div>
      </fieldset>
      </details><details [open]="!compact()"><summary>Advanced request, timing and TLS options</summary>
      @if (compact()) { <div class="field"><label for="http-method-advanced">Request method</label><select id="http-method-advanced" formControlName="method"><option value="GET">GET</option><option value="HEAD">HEAD</option></select></div> }
      <fieldset class="http-check-fieldset"><legend>Response time and TLS</legend><div class="form-grid">
        <div class="field"><label for="http-response-limit">Maximum response time (ms)</label><input id="http-response-limit" type="number" formControlName="maxResponseMs" min="0" max="3600000" step="1" required [attr.aria-invalid]="!!service.errors().maxResponseMs" aria-describedby="http-response-help http-response-error"><small id="http-response-help">0 disables this assertion. This limit is separate from adaptive timeouts and slowdown rules.</small>@if (service.errors().maxResponseMs) { <p class="field-error" id="http-response-error">{{ service.errors().maxResponseMs }}</p> }</div>
        <div class="field"><label for="http-certificate-days">Minimum certificate lifetime (days)</label><input id="http-certificate-days" type="number" formControlName="minCertificateDays" min="0" max="36500" step="1" required [attr.aria-invalid]="!!service.errors().minCertificateDays" aria-describedby="http-certificate-help http-certificate-error"><small id="http-certificate-help">0 disables the extra lifetime check. TLS trust, hostname, and certificate dates still require validation.</small>@if (service.errors().minCertificateDays) { <p class="field-error" id="http-certificate-error">{{ service.errors().minCertificateDays }}</p> }</div>
      </div></fieldset>
      </details>
    </div>
  `,
  styles: `
    .http-check-fieldset { margin: 1.25rem 0; padding: 1rem; border: 1px solid var(--border); border-radius: 7px; }
    legend { padding: 0 .4rem; font-weight: 600; }
    fieldset > p { margin-bottom: 16px; }
    .field-error { color: var(--error); font-size: 12px; }
  `
})
export class HTTPCheckEditorComponent { readonly service = inject(HTTPCheckEditorService); readonly compact = input(false); readonly endpoints = input<readonly string[]>([]) }
