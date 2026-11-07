import { DatePipe } from '@angular/common'
import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { ProviderEditorComponent } from './provider-editor.component'
import { ProviderEditorService } from './provider-editor.service'
import { ProvidersService } from './providers.service'
import { providerRoute } from './provider.utils'

@Component({
  standalone: true, imports: [DatePipe, RouterLink, ProviderEditorComponent], providers: [ProvidersService, ProviderEditorService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">System administration</span><h1 id="providers-heading" tabindex="-1">Identity providers</h1><p class="muted">Configure OpenID Connect sign-in and exact external group mappings.</p></div><div class="row-actions wrap-actions"><button type="button" class="button secondary" [disabled]="service.loading() || editor.busy()" (click)="service.refresh()">Refresh providers</button>@if (service.canCreate()) { <button type="button" class="button primary" [disabled]="editor.busy()" (click)="editor.open()">+ Register provider</button> }</div></div>
    @if (editor.opened()) { <ts-provider-editor /> }@if (service.error()) { <p class="notice error" role="alert">{{ service.error() }}</p> }@if (service.message()) { <p class="notice success" role="status">{{ service.message() }}</p> }
    <section class="panel" aria-labelledby="provider-directory-heading"><h2 id="provider-directory-heading">Provider configuration</h2><p class="small muted">Disabled providers cannot start sign-in. Every configuration save requires fresh provider authentication. Local accounts remain separate.</p>
      @if (service.providers.error()) { <p class="notice error" role="alert">{{ service.providers.error() }}</p> }
      <div class="table-scroll" tabindex="0" role="region" aria-label="Identity provider directory"><table><caption class="sr-only">Identity provider configuration</caption><thead><tr><th scope="col">Provider</th><th scope="col">State</th><th scope="col">Issuer</th><th scope="col">Provisioning</th><th scope="col">Updated</th></tr></thead><tbody>@for (row of service.providers.items(); track row.id) { <tr><th scope="row"><a [routerLink]="route(row.id)">{{ row.data.name }}</a><code class="table-subtitle wrap-anywhere">{{ row.id }}</code></th><td><span class="badge" [class.warning]="row.data.disabled">{{ row.data.disabled ? 'Disabled' : 'Enabled' }}</span></td><td class="wrap-anywhere">{{ row.data.issuer }}</td><td>{{ row.data.autoProvision ? 'Create pending profiles' : 'Existing accounts only' }}</td><td>{{ row.updatedAt | date:'medium':'UTC' }} UTC</td></tr> }@empty { @if (!service.loading() && !service.providers.loading() && !service.providers.error() && !service.error()) { <tr><td colspan="5">No readable providers in this page.</td></tr> } }</tbody></table></div>
      @if (service.loading() || service.providers.loading()) { <p class="small" role="status">Loading providers…</p> }@if (service.providers.partial()) { <p class="small muted">This page covers a bounded scan. More readable providers can appear after an empty page.</p> }@if (service.providers.cursor()) { <button type="button" class="button secondary more-button" [disabled]="service.providers.loading()" (click)="service.load(true)">More providers</button> }
    </section>
  `,
  styles: `table { min-width: 820px; } th[scope="row"] { min-width: 180px; } td:nth-child(3) { min-width: 240px; max-width: 340px; }`
})
export class ProvidersPage { readonly service = inject(ProvidersService); readonly editor = inject(ProviderEditorService); readonly route = providerRoute }
