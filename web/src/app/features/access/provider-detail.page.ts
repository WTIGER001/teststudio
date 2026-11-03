import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { ProviderEditorComponent } from './provider-editor.component'
import { ProviderEditorService } from './provider-editor.service'
import { ProviderGroupsComponent } from './provider-groups.component'
import { ProvidersService } from './providers.service'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, ProviderEditorComponent, ProviderGroupsComponent], providers: [ProvidersService, ProviderEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <a class="small" routerLink="/settings/providers">← Identity providers</a>
    <div class="page-heading"><div><span class="eyebrow">Installation administration</span><h1 id="provider-details-heading" tabindex="-1">{{ service.detail()?.data?.name || 'Identity provider' }}</h1><p class="muted">Inspect stored sign-in configuration and provider-owned group mappings.</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy()" (click)="service.refresh()">Refresh provider</button>@if (service.detail()?.data?.canManage) { <button type="button" class="button primary" [disabled]="service.loading() || editor.busy()" (click)="editor.open(service.detail()!)">Edit provider</button> }</div></div>
    @if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }@if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }@if (editor.opened()) { <ts-provider-editor /> }
    @if (service.loading()) { <p role="status">Loading provider…</p> }
    @if (service.detail(); as item) {
      <section class="panel" aria-labelledby="stored-provider-heading"><div class="section-top"><h2 id="stored-provider-heading">Stored provider</h2><span class="badge" [class.warning]="item.data.disabled">{{ item.data.disabled ? 'Disabled' : 'Enabled' }}</span></div><dl class="detail-grid"><dt>Identifier</dt><dd><code>{{ item.id }}</code></dd><dt>Revision</dt><dd>{{ item.revision }}</dd><dt>Exact issuer</dt><dd class="wrap-anywhere">{{ item.data.issuer }}</dd><dt>Client identifier</dt><dd class="wrap-anywhere">{{ item.data.clientId }}</dd><dt>Callback URL</dt><dd class="wrap-anywhere">{{ item.data.callbackUrl }}</dd><dt>Provider logout endpoint</dt><dd class="wrap-anywhere">{{ item.data.logoutEndpoint || "TestStudio-only sign-out" }}</dd><dt>Post-logout callback URL</dt><dd class="wrap-anywhere">{{ item.data.logoutCallbackUrl }}</dd><dt>Credential reference</dt><dd class="wrap-anywhere">{{ item.data.secretRef || 'No client secret' }}</dd><dt>Requested scopes</dt><dd class="wrap-anywhere">{{ item.data.scopes.join(' ') || 'Default · openid profile email' }}</dd><dt>Provisioning</dt><dd>{{ item.data.autoProvision ? 'Create pending user profiles' : 'Existing accounts only' }}</dd><dt>Membership lifetime</dt><dd>{{ item.data.membershipTtlSeconds || 1800 }} seconds, bounded by token expiry</dd><dt>Updated</dt><dd>{{ item.updatedAt | date:'medium':'UTC' }} UTC</dd></dl><p class="small muted guidance">{{ item.data.canManage ? 'Changes require a reason and the current stored revision.' : 'Read only. Provider management and mapped-group membership authority are required for changes.' }}</p></section>
      <section class="panel" aria-labelledby="stored-provider-mappings-heading"><h2 id="stored-provider-mappings-heading">Stored group mappings</h2><p class="small muted wrap-anywhere">Claim: {{ item.data.groupClaim || 'No claim configured' }}. Exact values only. Destination groups must belong to oidc:{{ item.id }}.</p><div class="table-scroll" tabindex="0" role="region" aria-label="Provider group mappings"><table><caption class="sr-only">Stored external group mappings</caption><thead><tr><th scope="col">External claim value</th><th scope="col">Destination group identifier</th></tr></thead><tbody>@for (row of service.mappings(); track row.external) { <tr><th scope="row" class="wrap-anywhere">{{ row.external }}</th><td class="wrap-anywhere">{{ row.groupId }}</td></tr> }@empty { <tr><td colspan="2">No external group mappings configured.</td></tr> }</tbody></table></div></section>
      @if (item.data.canManage) { <ts-provider-groups /> }
    }
  `,
  styles: `.guidance { margin-top: 20px; } table { min-width: 560px; } @media (max-width: 700px) { .detail-grid { grid-template-columns: minmax(0, 1fr); gap: 6px; } dd { margin-bottom: 10px; } }`
})
export class ProviderDetailPage { readonly service = inject(ProvidersService); readonly editor = inject(ProviderEditorService) }
