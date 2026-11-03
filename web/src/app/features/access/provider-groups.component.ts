import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { ReactiveFormsModule } from '@angular/forms'
import { ProviderGroupsService } from './provider-groups.service'

@Component({
  selector: 'ts-provider-groups', standalone: true, imports: [ReactiveFormsModule], providers: [ProviderGroupsService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="provider-group-heading"><h2 id="provider-group-heading">Create a provider group</h2><p class="small muted">Create an empty group with permanent provider ownership. Membership comes from validated provider claims. Assign roles through the normal access pages.</p>
      <form [formGroup]="service.form" (ngSubmit)="service.submit()"><fieldset [disabled]="service.busy()"><legend class="sr-only">New provider group</legend><div class="form-grid"><div class="field"><label for="provider-group-id">New group identifier</label><input id="provider-group-id" formControlName="id" required maxlength="128"></div><div class="field"><label for="provider-group-name">New group name</label><input id="provider-group-name" formControlName="name" required maxlength="256"></div><div class="field"><label for="provider-group-project">Group project identifier (optional)</label><input id="provider-group-project" formControlName="projectId" maxlength="128"><p class="small muted">Leave empty for installation ownership. Ownership remains permanent.</p></div></div>
        @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }@if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }@if (service.blocked()) { <p class="notice warning">Review the group directory before another creation attempt. Starting another group clears these fields.</p> }
        <div class="row-actions wrap-actions"><button type="submit" class="button secondary" [disabled]="service.blocked()">Create provider group</button><button type="button" class="button subtle" (click)="service.reset()">Start another group</button></div>
      </fieldset></form>
    </section>
  `,
  styles: `fieldset { border: 0; padding: 0; margin: 18px 0 0; min-width: 0; } .field { min-width: 0; align-content: start; } @media (max-width: 700px) { .form-grid { grid-template-columns: minmax(0, 1fr); } }`
})
export class ProviderGroupsComponent { readonly service = inject(ProviderGroupsService) }
