import { ChangeDetectionStrategy, Component, inject } from '@angular/core'
import { RouterLink } from '@angular/router'
import { ConfigurationService } from '../configuration/configuration.service'
import { ResourceRegistrationFormComponent } from '../configuration/resource-registration-form.component'
import { ApplicationsService } from './applications.service'

@Component({
  standalone: true, imports: [RouterLink, ResourceRegistrationFormComponent], providers: [ConfigurationService, ApplicationsService], changeDetection: ChangeDetectionStrategy.OnPush,
  template: `
    @if (service.applicationId()) { <p class="back-link"><a [routerLink]="['/projects', config.projectId(), 'applications']">← Applications</a></p> }
    <div class="page-heading"><div><span class="eyebrow">{{ config.workspace.project()?.name || 'Project' }}</span><h1>{{ service.applicationId() ? service.current()?.name || 'Application' : 'Applications' }}</h1><p class="muted">{{ service.applicationId() ? 'Test suites and deployments owned by this application.' : 'Explore the systems you test and their environments and suites.' }}</p></div>
      @if (!service.applicationId() && config.workspace.can('projects.manage')) { <button type="button" class="button primary" (click)="service.openForm('applications')">+ Add application</button> }
    </div>
    @if (config.error()) { <p class="notice error" role="alert">{{ config.error() }}</p> }
    @if (config.success()) { <p class="notice success" role="status">{{ config.success() }}</p> }
    @if (config.loading()) { <p role="status">Loading applications…</p> }
    @else if (!service.applicationId()) {
      <div class="application-grid">@for (app of service.applications(); track app.id) {
        <article class="panel application-card"><div><h2><a [routerLink]="['/projects', config.projectId(), 'applications', app.id]">{{ app.name }}</a></h2><code>{{ app.id }}</code></div>@if (app.description) { <p class="muted">{{ app.description }}</p> }
          <div class="row-actions wrap-actions">@if (config.workspace.can('environments.read')) { <span class="badge">{{ app.environments }} environments</span> }@if (config.workspace.can('suites.read')) { <span class="badge">{{ app.suites }} suites</span> }</div>
          <a class="button secondary" [routerLink]="['/projects', config.projectId(), 'applications', app.id]" [attr.aria-label]="'Open application ' + app.name">Open application</a>
        </article>
      } @empty { <section class="panel empty-state"><h2>No applications yet.</h2><p>Register an application to organize its environments and organize test suites.</p></section> }</div>
    } @else if (service.current(); as app) {
      <p class="small muted application-description"><code>{{ app.id }}</code>@if (app.description) { <span> · {{ app.description }}</span> }</p>
      <div class="row-actions wrap-actions"><a class="button primary" [routerLink]="config.workspace.path('catalog')">Test catalog</a><a class="button secondary" [routerLink]="['/projects', config.projectId(), 'applications', app.id, 'configuration']">Environment configuration</a></div>
      <p class="muted">Choose this application to select what you test. Choose an environment to select where you test it.</p>
      <div class="two-column">
        <section class="panel" aria-labelledby="application-environments-title"><div class="section-top"><h2 id="application-environments-title">Environments</h2>@if (config.workspace.can('environments.manage')) { <button type="button" class="button secondary" (click)="service.openForm('environments')">Add environment</button> }</div>
          @if (!config.workspace.can('environments.read')) { <p class="muted">Environment read permission is required to view this application's targets.</p> }
          @else { @for (row of service.related().environments; track row.id) { <article class="resource-row"><div><h3>{{ row.data.name }}</h3><code>{{ row.id }}</code><p class="small wrap-anywhere">Base URL: {{ row.data.endpoints['api'] || 'See environment configuration' }}</p></div><div class="row-actions wrap-actions"><span class="badge">{{ row.data.stage }}</span><span class="badge" [class.warning]="row.data.protected">{{ row.data.protected ? 'Protected' : 'Standard' }}</span></div></article> } @empty { <p class="muted">No environments registered for this application.</p> } }
        </section>
        <section class="panel" aria-labelledby="application-suites-title"><div class="section-top"><h2 id="application-suites-title">Suites</h2>@if (config.workspace.can('suites.manage')) { <button type="button" class="button secondary" (click)="service.openForm('suites')">Add suite</button> }</div>
          <p class="small muted">Each suite belongs to this application. Choose an environment when running its tests.</p>
          @if (!config.workspace.can('suites.read')) { <p class="muted">Suite read permission is required to view suites.</p> }
          @else { @for (row of service.related().suites; track row.id) { <article class="suite-row"><div><h3>{{ row.data.name }}</h3><code>{{ row.id }}</code><span class="badge" [class.warning]="row.data.archived">{{ row.data.archived ? 'Archived' : 'Active' }}</span></div><div class="row-actions wrap-actions">@if (config.workspace.can('catalog.read')) { <a class="button small secondary" [routerLink]="config.workspace.path('catalog')" [queryParams]="{ suite: row.id }" [attr.aria-label]="'View tests in ' + row.data.name">View tests</a> }@if (!row.data.archived && config.workspace.can('catalog.create')) { <a class="button small secondary" [routerLink]="config.workspace.path('tests', 'new')" [queryParams]="{ suite: row.id }">Create test</a> }</div></article> } @empty { <p class="muted">No suites belong to this application.</p> } }
        </section>
      </div>
    } @else if (!config.error()) { <section class="panel empty-state"><h2>Application not found.</h2><p>Return to Applications to view systems in this project.</p></section> }
    @if (service.formOpen() && config.canCreate()) {
      <section class="panel" aria-labelledby="application-registration-title"><div class="section-top"><h2 id="application-registration-title">{{ config.kind() === 'applications' ? 'Add an application' : config.kind() === 'environments' ? 'Add an environment' : 'Add a suite' }}</h2><button type="button" class="button secondary" [disabled]="config.saving()" (click)="service.closeForm()">Cancel registration</button></div><ts-resource-registration-form (submitted)="service.create()" /></section>
    }
  `,
  styles: [`:host { display: block; } .back-link { margin-bottom: 1rem; } .application-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 20rem), 1fr)); gap: 1.25rem; } .application-card { margin: 0; display: flex; flex-direction: column; gap: 1rem; align-items: flex-start; } .application-card h2 { font-size: 18px; margin-bottom: .25rem; } .application-card .button { margin-top: auto; } .application-description { margin: 0 0 1.25rem; overflow-wrap: anywhere; } .suite-row { display: flex; flex-wrap: wrap; gap: .75rem; justify-content: space-between; align-items: center; padding: 1rem 0; border-bottom: 1px solid var(--border); } .suite-row code { display: block; margin: .25rem 0; } .resource-row { flex-wrap: wrap; gap: .75rem; } h2, h3, code { overflow-wrap: anywhere; }`]
})
export class ApplicationsPage {
  readonly service = inject(ApplicationsService)
  readonly config = this.service.registration
  canLeave(): boolean { return this.service.canLeave() }
}
