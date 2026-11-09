import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { ConfigurationService } from './configuration.service'
import { ResourceRegistrationFormComponent } from './resource-registration-form.component'
import { SuiteManagerComponent } from './suite-manager.component'
import { SuiteManagerService } from './suite-manager.service'
import { SuitesPageService } from './suites-page.service'

@Component({
  standalone: true, imports: [RouterLink, ResourceRegistrationFormComponent, SuiteManagerComponent], providers: [ConfigurationService, SuiteManagerService, SuitesPageService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    <div class="page-heading"><div><span class="eyebrow">{{ service.configuration.workspace.project()?.name || 'Project' }}</span><h1>Suites</h1><p class="muted">Group related tests. Open a suite to see its tests or add a new one.</p></div>@if (service.configuration.workspace.can('suites.manage')) { <button type="button" class="button primary" [disabled]="service.configuration.saving() || service.manager.busy()" (click)="service.openCreation()">+ New suite</button> }</div>
    @if (service.configuration.error()) { <p class="notice error" role="alert">{{ service.configuration.error() }}</p> }
    @if (service.configuration.success()) { <p role="status" class="notice success">{{ service.configuration.success() }}</p> }
    @if (service.creating()) { <section class="panel" aria-labelledby="new-suite-heading"><div class="section-top"><h2 id="new-suite-heading">Create suite</h2><button type="button" class="button subtle" [disabled]="service.configuration.saving()" (click)="service.closeCreation()">Cancel suite creation</button></div><ts-resource-registration-form (submitted)="service.create()" /></section> }
    @if (service.configuration.loading()) { <p role="status">Loading suites…</p> } @else if (!service.configuration.workspace.can('suites.read')) { <p class="notice">Suite read permission is required to view suites in this project.</p> }
    <div class="suite-grid">@for (suite of service.suites(); track suite.id) { <article class="panel" [attr.aria-label]="suite.data.name"><div class="section-top"><h2>{{ suite.data.name }}</h2><span class="badge" [class.warning]="suite.data.archived">{{ suite.data.archived ? 'Archived' : 'Active' }}</span></div><p class="muted">{{ suite.data.description || 'Keep related checks and their results together.' }}</p><p class="small">@if (suite.data.applications.length > 1) { <span class="badge">Shared suite · {{ suite.data.applications.length }} applications</span> } @else if (!suite.data.applications.length) { <span class="badge">No application linked</span> }</p><p class="small">{{ suite.data.requiresTarget ? 'Uses a target environment' : 'Can run without a target environment' }}</p><div class="row-actions wrap-actions"><a class="button secondary" [routerLink]="service.configuration.workspace.path('catalog')" [queryParams]="{ suite: suite.id }" [attr.aria-label]="'View tests in ' + suite.data.name">View tests</a>@if (!suite.data.archived && service.configuration.workspace.can('catalog.create')) { <a class="button primary" [routerLink]="service.configuration.workspace.path('tests', 'new')" [queryParams]="{ suite: suite.id }" [attr.aria-label]="'Create test in ' + suite.data.name">Create test</a> }<button type="button" class="button subtle" [disabled]="service.manager.busy() || service.configuration.saving()" (click)="service.inspect(suite.id)" [attr.aria-label]="'Suite settings for ' + suite.data.name">Suite settings</button></div></article> } @empty { @if (!service.configuration.loading() && !service.configuration.error() && service.configuration.workspace.can('suites.read')) { <section class="panel empty-state"><h2>Create your first suite</h2><p>A suite holds related tests and their history. Use New suite to follow the setup steps.</p></section> } }</div>
    <ts-suite-manager />
  `,
  styles: [`.suite-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 22rem), 1fr)); gap: 1.25rem; } .suite-grid .panel { margin: 0; } .suite-grid p { margin: 1rem 0; } .section-top { flex-wrap: wrap; }`]
})
export class SuitesPage { readonly service = inject(SuitesPageService); canLeave(): boolean { return this.service.canLeave() } }
