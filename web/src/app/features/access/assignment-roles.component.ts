import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { roleRoute } from './role-policy.utils'
import { AssignmentsService } from './assignments.service'

@Component({
  selector: 'ts-assignment-roles', standalone: true, imports: [RouterLink], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <section class="panel" aria-labelledby="assignment-roles-heading"><div class="section-top"><h2 id="assignment-roles-heading">Role choices</h2><button type="button" class="button secondary" [disabled]="service.rolesLoading() || service.loading() || !service.rights().readRoles" (click)="service.loadRoles()">Refresh roles</button></div>
      @if (service.rolesError()) { <p class="notice error" role="alert">{{ service.rolesError() }}</p> }
      @if (service.rolesLoading()) { <p class="small" role="status">Loading roles…</p> }
      <ul class="role-choices">@for (role of service.roleChoices(); track role.id) { <li><div class="section-top"><h3>{{ role.name }}</h3><span class="badge" [class.warning]="role.disabled">{{ role.disabled ? 'Disabled' : role.delegatable ? 'Can assign' : 'Read only' }}</span></div><code class="small muted wrap-anywhere">{{ role.id }}</code>@if (!role.builtIn) { <p class="small guidance">Custom policy with exact rules and inheritance. <a [routerLink]="route(service.projectId(), role.id)">Open {{ role.name }} policy</a></p> } @else { <details><summary>View permissions for {{ role.name }}</summary><ul class="permission-list">@for (permission of role.permissions; track permission) { <li><code class="small wrap-anywhere">{{ permission }}</code></li> }</ul></details> }</li> }</ul>
      @if (service.customRoles.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.customRoles.loading()" (click)="service.loadCustomRoles(true)">More custom role choices</button> }
    </section>
  `,
  styles: `.role-choices { list-style: none; margin: 16px 0 0; padding: 0; display: grid; gap: 14px; } .role-choices > li { border: 1px solid var(--border); border-radius: 7px; padding: 16px; min-width: 0; } h3 { margin: 0; } details { margin-top: 12px; } summary { cursor: pointer; } .permission-list { margin: 12px 0 0; padding-left: 20px; }`
})
export class AssignmentRolesComponent { readonly service = inject(AssignmentsService); readonly route = roleRoute }
