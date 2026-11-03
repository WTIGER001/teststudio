import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { PermissionExplanationService } from './permission-explanation.service'
import { PermissionSourceComponent } from './permission-source.component'
import { explanationReason } from './permission-explanation.utils'

@Component({
  standalone: true, imports: [DatePipe, ReactiveFormsModule, PermissionSourceComponent], providers: [PermissionExplanationService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow wrap-anywhere">{{ service.projectId() ? 'Project · ' + service.projectId() : 'Installation scope' }}</span><h1>Explain access</h1><p class="muted">Check one permission and see the grants and role paths behind the decision.</p></div></div>
    <section class="panel" aria-labelledby="access-context-heading"><h2 id="access-context-heading">Inspection context</h2><p class="small muted">An empty object checks the displayed scope. To check one resource, enter its exact object, such as suite/health. Object checks do not include descendants.</p>
      <form [formGroup]="service.form" (ngSubmit)="service.check()">
        <div class="form-grid"><div class="field"><label for="explain-user">Inspect user identifier</label><input id="explain-user" formControlName="userId" maxlength="128" [readOnly]="!service.canInspectOthers()" aria-describedby="explain-user-help"><p id="explain-user-help" class="small muted">{{ service.canInspectOthers() ? 'You can inspect registered users within your current administrative authority.' : 'Inspect your own access. Inspecting another user requires membership management and permission read authority.' }}</p></div><div class="field"><label for="explain-object">Exact object (optional)</label><input id="explain-object" formControlName="object" placeholder="suite/health" maxlength="400"></div><div class="field"><label for="explain-permission">Permission to check</label><select id="explain-permission" formControlName="permission">@for (permission of service.permissions(); track permission.id) { <option [value]="permission.id">{{ permission.area }} · {{ permission.action }} — {{ permission.description }}</option> }</select></div></div>
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="service.busy() || service.initializing() || !service.permissions().length">Check access</button><button type="button" class="button secondary" [disabled]="service.initializing()" (click)="service.inspectSelf()">Use my account</button>@if (!service.permissions().length && !service.initializing()) { <button type="button" class="button secondary" (click)="service.initialize()">Reload permission choices</button> }</div>
      </form>
    </section>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
    @if (service.busy() || service.initializing()) { <p role="status">Checking access and scanning sources…</p> }
    @if (service.report(); as report) {
      <section class="panel" aria-labelledby="access-decision-heading"><div class="section-top"><h2 id="access-decision-heading">Current decision</h2><span class="badge" [class.success]="report.allowed" [class.warning]="!report.allowed">{{ report.allowed ? 'Allowed' : 'Denied' }}</span></div><p class="decision-reason" role="status">{{ reason(report.reason) }}</p>
        <dl class="detail-grid"><dt>User</dt><dd class="wrap-anywhere">{{ report.userName }} · {{ report.userId }}</dd><dt>Permission</dt><dd><code class="wrap-anywhere">{{ report.permission }}</code></dd><dt>Object</dt><dd><code class="wrap-anywhere">{{ report.object || 'Whole scope' }}</code></dd><dt>Checked at</dt><dd>{{ report.checkedAt | date:'medium':'UTC' }} UTC</dd></dl><p class="small muted guidance">This is the complete decision at the check time. Source scanning does not change it. Later access changes require a new check.</p>
      </section>
      <section class="panel" aria-labelledby="access-sources-heading"><h2 id="access-sources-heading">Permission sources</h2><p class="small muted">{{ report.items.length }} source paths loaded. Active denies override allows. Expired sources explain earlier rules and have no effect. Field restrictions follow the permission evaluator.</p>
        <div class="source-list">@for (source of report.items; track source.id) { <ts-permission-source [source]="source" /> }</div>
        @if (report.partial) { <p class="notice warning">Source scan is incomplete. More sources can appear even after an empty page.</p><button type="button" class="button secondary" [disabled]="service.busy()" (click)="service.check(true)">Scan more sources</button> }
        @else { <p class="small muted scan-complete">Source scan complete. {{ report.items.length ? 'All matching paths and expired matching rules are listed.' : 'No matching or expired source paths were found.' }}</p> }
      </section>
    }
  `,
  styles: `form { margin-top: 20px; } .form-grid { margin-bottom: 20px; } .field { min-width: 0; } .field p { margin-top: 8px; } .source-list { display: grid; grid-template-columns: minmax(0, 1fr); gap: 14px; margin-top: 20px; } .decision-reason { margin: 16px 0; } .guidance, .scan-complete { margin-top: 20px; } .row-actions { gap: 10px; } select { width: 100%; } @media (max-width: 700px) { .form-grid, .detail-grid { grid-template-columns: minmax(0, 1fr); } .detail-grid { gap: 6px; } .detail-grid dd { margin-bottom: 10px; } }`
})
export class PermissionExplanationPage { readonly service = inject(PermissionExplanationService); readonly reason = explanationReason }
