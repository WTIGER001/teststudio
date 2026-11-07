import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { PendingAccessService } from './pending-access.service'

@Component({
  selector: 'ts-pending-access-editor', standalone: true, imports: [ReactiveFormsModule], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="pending-editor" aria-labelledby="pending-editor-heading"><div class="section-top"><h3 id="pending-editor-heading" tabindex="-1">Prepare project access</h3><button type="button" class="button subtle" [disabled]="service.busy()" (click)="service.close()">Close pending access editor</button></div>
      <form [formGroup]="service.form" (ngSubmit)="service.submit()"><fieldset [disabled]="service.busy() || !service.access.rights().manage"><legend class="sr-only">Pending project role approval</legend>
        <div class="field"><label for="pending-provider">Allowed sign-in provider</label><select id="pending-provider" formControlName="providerId" required><option value="">Choose a provider</option>@for (provider of service.reviewedProviders(); track provider.id) { <option [value]="provider.id">{{ provider.name }}</option> }</select></div>
        @if (service.selectedProvider(); as provider) { <p class="small muted wrap-anywhere">Exact issuer: {{ provider.issuer }} · Configuration revision {{ provider.revision }}.</p>@if (!provider.autoProvision) { <p class="small muted">Automatic first-sign-in profiles are disabled. The user needs an existing linked identity before this approval can activate.</p> } }
        <div class="field"><label for="pending-email">Required verified email</label><input id="pending-email" type="email" formControlName="email" required maxlength="254" autocomplete="off" aria-describedby="pending-email-help"></div>
        <p id="pending-email-help" class="small muted">The mailbox name before @ must match exactly. Domain letters ignore case. Provider verification is required; profile edits cannot activate access.</p>
        <div class="field"><label for="pending-role">Pending project role</label><select id="pending-role" formControlName="roleId" required><option value="">Choose a project role</option>@for (role of service.access.roleChoices(); track role.id) { <option [value]="role.id" [disabled]="!role.delegatable || role.disabled">{{ role.name }}{{ role.disabled ? ' · Disabled' : !role.delegatable ? ' · Read only' : '' }}</option> }</select></div>
        <div class="field"><label for="pending-reason">Pending access reason</label><textarea id="pending-reason" formControlName="reason" required maxlength="1024" rows="2"></textarea></div>
        <p class="small muted">Activation rechecks this project, the approved provider and role, and your current authority. Changed approvals require a new review. System administration is unavailable here.</p>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }
        @if (service.needsReview()) { <p class="small muted">Review the current records before another submission. Cancel an existing pending approval before replacing it. Your entries keep their opened provider revision until you reload choices.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button primary" [disabled]="service.needsReview() || !service.canAdd()">Prepare access</button>@if (service.needsReview()) { <button type="button" class="button secondary" [disabled]="service.loading()" (click)="service.review()">Review pending records and reload choices</button> }</div>
      </fieldset></form>
    </section>
  `,
  styles: `.pending-editor { margin: 20px 0; padding: 20px; border: 1px solid var(--border); border-radius: 7px; } fieldset { border: 0; margin: 18px 0 0; padding: 0; min-width: 0; } h3 { margin: 0; } p { margin: 14px 0; } @media (max-width: 640px) { .section-top { flex-direction: column; align-items: stretch; gap: 12px; } }`
})
export class PendingAccessEditorComponent { readonly service = inject(PendingAccessService) }
